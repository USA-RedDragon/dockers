package mariadb

import (
	"bufio"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
)

const (
	probeTable  = "__b2bk_probe"
	swapTimeout = "SET SESSION lock_wait_timeout = 60"
)

// restore puts a dump in place of a database's tables all at once. It restores
// the dump into a staging database, then moves the live tables into a rollback
// database and the staged ones into their place with one RENAME TABLE.
type restore struct {
	db                   conn
	live, staging, old   string
	dryRun, dropOld      bool
	out                  io.Writer
	charset, collation   string
	liveTables, incoming []string
}

func newRestore(db conn, live string, dryRun, dropOld bool, out io.Writer) (*restore, error) {
	staging, old, err := Names(live)
	if err != nil {
		return nil, err
	}
	return &restore{db: db, live: live, staging: staging, old: old, dryRun: dryRun, dropOld: dropOld, out: out}, nil
}

func (r *restore) printf(format string, args ...any) {
	_, _ = fmt.Fprintf(r.out, format, args...)
}

func (r *restore) run(ctx context.Context, dump io.Reader) error {
	in := bufio.NewReaderSize(dump, 64<<10)
	if _, err := in.Peek(1); err != nil {
		if errors.Is(err, io.EOF) {
			return ErrEmptyDump
		}
		return fmt.Errorf("read dump: %w", err)
	}
	if err := r.check(ctx); err != nil {
		return fmt.Errorf("%w; %s was not changed", err, QuoteIdent(r.live))
	}
	if err := r.prepare(ctx); err != nil {
		return r.abort(ctx, err)
	}
	if err := r.load(ctx, in); err != nil {
		return r.abort(ctx, err)
	}
	if err := r.plan(ctx); err != nil {
		return r.abort(ctx, err)
	}
	stmt := RenameStatement(r.live, r.staging, r.old, r.liveTables, r.incoming)
	if r.dryRun {
		if stmt != "" {
			if err := r.db.execute(ctx, PrepareStatement(stmt)); err != nil {
				return r.abort(ctx, fmt.Errorf("the server did not accept the swap: %w", err))
			}
		}
		r.printf("dry run: the dump restored cleanly into %s; %s was not changed\n",
			QuoteIdent(r.staging), QuoteIdent(r.live))
		if stmt != "" {
			r.printf("the swap would run: %s\n", stmt)
		}
		return r.cleanup(ctx)
	}
	return r.swap(ctx, stmt)
}

// check refuses, before anything is written, a target the swap cannot handle
// and a rollback database an earlier restore left behind.
func (r *restore) check(ctx context.Context) error {
	rows, err := r.db.query(ctx, "SELECT DEFAULT_CHARACTER_SET_NAME, DEFAULT_COLLATION_NAME "+
		"FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = "+literal(r.live))
	if err != nil {
		return err
	}
	if len(rows) != 1 || len(rows[0]) != 2 {
		return fmt.Errorf("%w: %s", ErrNoDatabase, QuoteIdent(r.live))
	}
	r.charset, r.collation = rows[0][0], rows[0][1]

	if err := r.refuseObjects(ctx, r.live, false); err != nil {
		return err
	}
	if kept, err := r.objects(ctx, r.live, true); err != nil {
		return err
	} else if len(kept) > 0 {
		r.printf("%s keeps its %s; they are not in a mariadb-dump without --routines and --events, "+
			"and the swap leaves them as they are\n", QuoteIdent(r.live), strings.Join(kept, ", "))
	}

	if err := r.refuseForeign(ctx); err != nil {
		return err
	}
	held, err := r.db.query(ctx, "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = "+
		literal(r.old)+" UNION ALL SELECT ROUTINE_NAME FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA = "+
		literal(r.old)+" UNION ALL SELECT EVENT_NAME FROM information_schema.EVENTS WHERE EVENT_SCHEMA = "+
		literal(r.old))
	if err != nil {
		return err
	}
	if len(held) > 0 {
		return fmt.Errorf("%w: %s holds the tables an earlier restore replaced. Once you no longer need to roll "+
			"back to them, drop it with DROP DATABASE %s; and run the restore again",
			ErrOldExists, QuoteIdent(r.old), QuoteIdent(r.old))
	}
	return nil
}

// existing returns the staging and rollback schemas that exist, and whether
// each carries SchemaComment.
func (r *restore) existing(ctx context.Context) (map[string]bool, error) {
	rows, err := r.db.query(ctx, "SELECT SCHEMA_NAME, SCHEMA_COMMENT FROM information_schema.SCHEMATA "+
		"WHERE SCHEMA_NAME IN ("+literal(r.staging)+", "+literal(r.old)+")")
	if err != nil {
		return nil, err
	}
	marked := map[string]bool{}
	for _, row := range rows {
		if len(row) == 2 {
			marked[row[0]] = row[1] == SchemaComment
		}
	}
	return marked, nil
}

// refuseForeign fails when a schema by the staging or rollback name exists
// that restoredb did not create, so that it is never dropped or filled.
func (r *restore) refuseForeign(ctx context.Context) error {
	marked, err := r.existing(ctx)
	if err != nil {
		return err
	}
	for _, schema := range []string{r.staging, r.old} {
		if mine, ok := marked[schema]; ok && !mine {
			return fmt.Errorf("%w: %s has no %q comment. If an earlier restore left it, drop it with "+
				"DROP DATABASE %s; once nothing in it is needed, and run the restore again; otherwise restore "+
				"into another database", ErrForeignSchema, QuoteIdent(schema), SchemaComment, QuoteIdent(schema))
		}
	}
	return nil
}

// objects lists the views and triggers in schema, or with routines its stored
// routines and events, as "kind name".
func (r *restore) objects(ctx context.Context, schema string, routines bool) ([]string, error) {
	s := literal(schema)
	sql := "SELECT 'view', TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = " + s +
		" AND TABLE_TYPE = 'VIEW' UNION ALL SELECT 'trigger', TRIGGER_NAME FROM information_schema.TRIGGERS " +
		"WHERE TRIGGER_SCHEMA = " + s
	if routines {
		sql = "SELECT LOWER(ROUTINE_TYPE), ROUTINE_NAME FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA = " + s +
			" UNION ALL SELECT 'event', EVENT_NAME FROM information_schema.EVENTS WHERE EVENT_SCHEMA = " + s
	}
	rows, err := r.db.query(ctx, sql)
	if err != nil {
		return nil, err
	}
	found := make([]string, 0, len(rows))
	for _, row := range rows {
		if len(row) == 2 {
			found = append(found, row[0]+" "+QuoteIdent(row[1]))
		}
	}
	return found, nil
}

// refuseObjects fails when schema holds an object RENAME TABLE cannot move to
// another database: MariaDB refuses to move a view (error 1450) or a table that
// has triggers (error 1435). With all, stored routines and events count too,
// which a staged dump must not have, since nothing would move them.
func (r *restore) refuseObjects(ctx context.Context, schema string, all bool) error {
	found, err := r.objects(ctx, schema, false)
	if err != nil {
		return err
	}
	if all {
		more, err := r.objects(ctx, schema, true)
		if err != nil {
			return err
		}
		found = append(found, more...)
	}
	if len(found) == 0 {
		return nil
	}
	where := QuoteIdent(schema)
	if schema == r.staging {
		where = "the dump"
	}
	return fmt.Errorf("%w: %s has %s, which RENAME TABLE cannot move to another database. "+
		"To restore anyway, pipe the dump into the mariadb client by hand, which is not atomic",
		ErrUnsupportedObjects, where, strings.Join(found, ", "))
}

// prepare creates empty staging and rollback databases and checks that the
// user can create, rename and drop tables in both, so a missing grant fails
// before the dump is read.
func (r *restore) prepare(ctx context.Context) error {
	opts := " COMMENT '" + SchemaComment + "'"
	if r.charset != "" && r.collation != "" {
		opts = " CHARACTER SET " + r.charset + " COLLATE " + r.collation + opts
	}
	stagingProbe, oldProbe := qualified(r.staging, probeTable), qualified(r.old, probeTable)
	sql := strings.Join([]string{
		"DROP DATABASE IF EXISTS " + QuoteIdent(r.staging),
		"DROP DATABASE IF EXISTS " + QuoteIdent(r.old),
		"CREATE DATABASE " + QuoteIdent(r.staging) + opts,
		"CREATE DATABASE " + QuoteIdent(r.old) + opts,
		"CREATE TABLE " + stagingProbe + " (i INT)",
		"RENAME TABLE " + stagingProbe + " TO " + oldProbe,
		"RENAME TABLE " + oldProbe + " TO " + stagingProbe,
		"DROP TABLE " + stagingProbe,
	}, "; ")
	if err := r.db.execute(ctx, sql); err != nil {
		return fmt.Errorf("%w %s and %s (grant ALL on both): %w",
			ErrPrivileges, QuoteIdent(r.staging), QuoteIdent(r.old), err)
	}
	return nil
}

func (r *restore) load(ctx context.Context, in *bufio.Reader) error {
	f := &dumpFilter{}
	err := r.db.load(ctx, r.staging, func(w io.Writer) error { return f.copy(w, in) })
	if f.database != "" && f.database != r.live {
		r.printf("the dump's USE and CREATE DATABASE lines name %s; they were left out, so it restores into %s\n",
			QuoteIdent(f.database), QuoteIdent(r.live))
	}
	switch {
	case errors.Is(err, ErrUnsupportedObjects), errors.Is(err, ErrUnsupportedStatement),
		errors.Is(err, ErrMultipleDatabases), errors.Is(err, ErrServerStatement), errors.Is(err, ErrOtherDatabase),
		errors.Is(err, ErrClientCommand):
		return err
	case !f.complete():
		if err != nil {
			return fmt.Errorf("%w: %w", ErrIncomplete, err)
		}
		return ErrIncomplete
	case err != nil:
		return fmt.Errorf("restore the dump into %s: %w", QuoteIdent(r.staging), err)
	}
	return nil
}

// plan lists what the swap moves, once the staged dump is known to hold only tables.
func (r *restore) plan(ctx context.Context) error {
	if err := r.refuseObjects(ctx, r.staging, true); err != nil {
		return err
	}
	var err error
	if r.incoming, err = r.tables(ctx, r.staging); err != nil {
		return err
	}
	r.liveTables, err = r.tables(ctx, r.live)
	return err
}

func (r *restore) tables(ctx context.Context, schema string) ([]string, error) {
	rows, err := r.db.query(ctx, "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = "+
		literal(schema)+" AND TABLE_TYPE <> 'VIEW' ORDER BY TABLE_NAME")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		if len(row) > 0 {
			names = append(names, row[0])
		}
	}
	return names, nil
}

func (r *restore) swap(ctx context.Context, stmt string) error {
	if err := r.refuseObjects(ctx, r.live, false); err != nil {
		return r.abort(ctx, err)
	}
	if stmt != "" {
		if err := r.db.execute(ctx, swapTimeout+"; "+stmt); err != nil {
			if err := r.afterFailedSwap(context.WithoutCancel(ctx), err); err != nil {
				return err
			}
		}
	}
	r.printf("restored %s: %d tables swapped in, %d moved to %s\n",
		QuoteIdent(r.live), len(r.incoming), len(r.liveTables), QuoteIdent(r.old))
	if err := r.db.execute(ctx, "DROP DATABASE "+QuoteIdent(r.staging)); err != nil {
		r.printf("could not drop the empty staging database %s: %v\n", QuoteIdent(r.staging), err)
	}
	if r.dropOld {
		if err := r.db.execute(ctx, "DROP DATABASE "+QuoteIdent(r.old)); err != nil {
			return fmt.Errorf("the restore succeeded, but dropping %s failed: %w", QuoteIdent(r.old), err)
		}
		r.printf("dropped %s\n", QuoteIdent(r.old))
		return nil
	}
	r.printf("the replaced tables are kept in %s. Drop them once the restore checks out:\n  DROP DATABASE %s;\n",
		QuoteIdent(r.old), QuoteIdent(r.old))
	if back := RenameStatement(r.live, r.old, r.staging, r.incoming, r.liveTables); back != "" {
		r.printf("To roll back instead:\n  CREATE DATABASE %s COMMENT '%s';\n  %s;\n  DROP DATABASE %s;\n",
			QuoteIdent(r.staging), SchemaComment, back, QuoteIdent(r.staging))
	}
	return nil
}

// afterFailedSwap looks at where the tables are after the swap reported err,
// since a connection lost after the server ran the RENAME TABLE reports an
// error for a swap that happened. It returns nil when the swap was applied,
// and otherwise an error that says what is known.
func (r *restore) afterFailedSwap(ctx context.Context, err error) error {
	live, lerr := r.tables(ctx, r.live)
	staging, serr := r.tables(ctx, r.staging)
	old, oerr := r.tables(ctx, r.old)
	if lerr := cmp.Or(lerr, serr, oerr); lerr != nil {
		return fmt.Errorf("swap: %w; whether it was applied is unknown, since the tables could not be listed "+
			"afterwards (%w). If %s holds the restored tables, %s holds the replaced ones; otherwise the swap did "+
			"not happen and %s still holds the restored tables. Nothing was dropped", err, lerr,
			QuoteIdent(r.live), QuoteIdent(r.old), QuoteIdent(r.staging))
	}
	switch {
	case slices.Equal(live, r.incoming) && slices.Equal(old, r.liveTables) && len(staging) == 0:
		r.printf("the swap was applied although the client reported an error: %v\n", err)
		return nil
	case slices.Equal(live, r.liveTables) && slices.Equal(staging, r.incoming) && len(old) == 0:
		return r.abort(ctx, fmt.Errorf("swap: %w", err))
	}
	return fmt.Errorf("swap: %w; afterwards %s holds %d tables, %s %d and %s %d, which is neither the state "+
		"before the swap nor after it. Nothing was dropped", err, QuoteIdent(r.live), len(live),
		QuoteIdent(r.staging), len(staging), QuoteIdent(r.old), len(old))
}

// abort drops the staging database and the rollback database, which is still
// empty, and returns err with a note that the live database was not changed.
func (r *restore) abort(ctx context.Context, err error) error {
	err = fmt.Errorf("%w; %s was not changed", err, QuoteIdent(r.live))
	if cerr := r.cleanup(context.WithoutCancel(ctx)); cerr != nil {
		return errors.Join(err, cerr)
	}
	return err
}

func (r *restore) cleanup(ctx context.Context) error {
	marked, err := r.existing(ctx)
	if err != nil {
		return err
	}
	if marked[r.staging] {
		if err := r.db.execute(ctx, "DROP DATABASE IF EXISTS "+QuoteIdent(r.staging)); err != nil {
			return fmt.Errorf("drop the staging database %s by hand: %w", QuoteIdent(r.staging), err)
		}
	}
	if !marked[r.old] {
		return nil
	}
	held, err := r.tables(ctx, r.old)
	if err != nil || len(held) > 0 {
		return err
	}
	if err := r.db.execute(ctx, "DROP DATABASE IF EXISTS "+QuoteIdent(r.old)); err != nil {
		return fmt.Errorf("drop the empty rollback database %s by hand: %w", QuoteIdent(r.old), err)
	}
	return nil
}
