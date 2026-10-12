package mariadb

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/USA-RedDragon/dockers/images/restic/internal/untouched"
)

const (
	testLive    = "app"
	testStaging = "app__b2bk_restore"
	testOld     = "app__b2bk_old"
	goodDump    = "CREATE TABLE `t` (i int);\nINSERT INTO `t` VALUES (1);\n-- Dump completed\n"
)

// fakeConn answers information_schema queries from its fields and records
// every statement it is asked to run.
type fakeConn struct {
	tables  map[string][]string
	objects map[string][]string
	oldHeld bool
	foreign string
	failOn  string
	// dropAfterSwap makes the swap report an error after it was applied, as a
	// connection lost before the reply would, and every query fail after it
	// when queriesFail is set too.
	dropAfterSwap bool
	queriesFail   bool
	swapped       bool
	loaded        bytes.Buffer
	execs         []string
}

func (f *fakeConn) query(_ context.Context, sql string) ([][]string, error) {
	if f.swapped && f.queriesFail {
		return nil, errors.New("ERROR 2013: Lost connection to server during query")
	}
	schema := func(name string) bool { return strings.Contains(sql, literal(name)) }
	var rows [][]string
	switch {
	case strings.Contains(sql, "SCHEMA_NAME IN"):
		for _, name := range []string{testStaging, testOld} {
			comment := SchemaComment
			if name == f.foreign {
				comment = ""
			}
			rows = append(rows, []string{name, comment})
		}
	case strings.Contains(sql, "information_schema.SCHEMATA"):
		rows = [][]string{{"utf8mb4", "utf8mb4_uca1400_ai_ci"}}
	case strings.Contains(sql, "UNION ALL SELECT ROUTINE_NAME"):
		if f.oldHeld {
			rows = [][]string{{"t"}}
		}
	case strings.Contains(sql, "TABLE_TYPE = 'VIEW'"), strings.Contains(sql, "information_schema.ROUTINES"):
		for name, objs := range f.objects {
			if schema(name) && strings.Contains(sql, "TABLE_TYPE = 'VIEW'") {
				for _, o := range objs {
					rows = append(rows, []string{"view", o})
				}
			}
		}
	case strings.Contains(sql, "TABLE_TYPE <> 'VIEW'"):
		for name, tables := range f.tables {
			if schema(name) {
				for _, t := range tables {
					rows = append(rows, []string{t})
				}
			}
		}
	}
	return rows, nil
}

func (f *fakeConn) execute(_ context.Context, sql string) error {
	f.execs = append(f.execs, sql)
	if f.failOn != "" && strings.Contains(sql, f.failOn) {
		return errors.New("ERROR 1142: denied")
	}
	if strings.Contains(sql, "RENAME TABLE `app`.") {
		f.tables[testLive], f.tables[testOld] = f.tables[testStaging], f.tables[testLive]
		f.tables[testStaging] = nil
		f.swapped = true
		if f.dropAfterSwap {
			return errors.New("ERROR 2013: Lost connection to server during query")
		}
	}
	return nil
}

func (f *fakeConn) load(_ context.Context, database string, write func(io.Writer) error) error {
	if database != testStaging {
		return errors.New("loaded into " + database)
	}
	err := write(&f.loaded)
	f.tables[testStaging] = []string{"t"}
	return err
}

func (f *fakeConn) ran(stmt string) bool {
	for _, e := range f.execs {
		if strings.Contains(e, stmt) {
			return true
		}
	}
	return false
}

func newFake() *fakeConn {
	return &fakeConn{tables: map[string][]string{testLive: {"t", "extra"}}, objects: map[string][]string{}}
}

func runRestore(t *testing.T, f *fakeConn, dump string, dryRun, dropOld bool) (string, error) {
	t.Helper()
	var out strings.Builder
	r, err := newRestore(f, testLive, dryRun, dropOld, &out)
	if err != nil {
		t.Fatal(err)
	}
	err = r.run(context.Background(), strings.NewReader(dump))
	return out.String(), err
}

func TestRestoreSwapsOnlyAfterACompleteDump(t *testing.T) {
	t.Parallel()
	f := newFake()
	out, err := runRestore(t, f, goodDump, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if f.loaded.String() != goodDump {
		t.Errorf("loaded %q", f.loaded.String())
	}
	want := "RENAME TABLE `app`.`extra` TO `app__b2bk_old`.`extra`, `app`.`t` TO `app__b2bk_old`.`t`, " +
		"`app__b2bk_restore`.`t` TO `app`.`t`"
	if !f.ran(want) {
		t.Errorf("no swap among %q", f.execs)
	}
	for _, schema := range []string{testStaging, testOld} {
		if !f.ran("CREATE DATABASE `" + schema + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_uca1400_ai_ci COMMENT '" +
			SchemaComment + "'") {
			t.Errorf("%s was not created with the mark: %q", schema, f.execs)
		}
	}
	if !f.ran("DROP DATABASE `app__b2bk_restore`") || f.ran("DROP DATABASE `app__b2bk_old`") {
		t.Errorf("staging should be dropped and the rollback copy kept: %q", f.execs)
	}
	if !strings.Contains(out, "DROP DATABASE `app__b2bk_old`;") {
		t.Errorf("output does not say how to drop the rollback copy:\n%s", out)
	}
	back := "RENAME TABLE `app`.`t` TO `app__b2bk_restore`.`t`, `app__b2bk_old`.`extra` TO `app`.`extra`, " +
		"`app__b2bk_old`.`t` TO `app`.`t`;"
	if !strings.Contains(out, back) {
		t.Errorf("output does not give the rollback:\n%s", out)
	}
}

func TestRestoreDropOld(t *testing.T) {
	t.Parallel()
	f := newFake()
	if _, err := runRestore(t, f, goodDump, false, true); err != nil {
		t.Fatal(err)
	}
	if !f.ran("DROP DATABASE `app__b2bk_old`") {
		t.Errorf("the rollback copy was kept: %q", f.execs)
	}
}

func TestRestoreNeverSwapsAfterAFailure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		dump    string
		setup   func(*fakeConn)
		want    error
		prepare bool
	}{
		{name: "empty", dump: "", want: ErrEmptyDump},
		{name: "cut short", dump: strings.TrimSuffix(goodDump, "-- Dump completed\n"), want: ErrIncomplete,
			prepare: true},
		{name: "two databases", dump: "USE `a`;\nUSE `b`;\n" + goodDump, want: ErrMultipleDatabases, prepare: true},
		{name: "rollback copy left over", dump: goodDump, setup: func(f *fakeConn) { f.oldHeld = true },
			want: ErrOldExists},
		{name: "live view", dump: goodDump, setup: func(f *fakeConn) { f.objects[testLive] = []string{"v"} },
			want: ErrUnsupportedObjects},
		{name: "view in the staged dump", dump: goodDump,
			setup: func(f *fakeConn) { f.objects[testStaging] = []string{"v"} }, want: ErrUnsupportedObjects,
			prepare: true},
		{name: "missing grants", dump: goodDump, setup: func(f *fakeConn) { f.failOn = "DROP DATABASE IF EXISTS" },
			want: ErrPrivileges, prepare: true},
		{name: "a staging database restoredb did not create", dump: goodDump,
			setup: func(f *fakeConn) { f.foreign = testStaging }, want: ErrForeignSchema},
		{name: "a rollback database restoredb did not create", dump: goodDump,
			setup: func(f *fakeConn) { f.foreign = testOld }, want: ErrForeignSchema},
		{name: "a rollback database restoredb did not create, with tables", dump: goodDump,
			setup: func(f *fakeConn) {
				f.foreign, f.oldHeld, f.tables[testOld] = testOld, true, []string{"t"}
			}, want: ErrForeignSchema},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFake()
			if tt.setup != nil {
				tt.setup(f)
			}
			_, err := runRestore(t, f, tt.dump, false, false)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
			if tt.want != ErrEmptyDump && !strings.Contains(err.Error(), "`app` was not changed") {
				t.Errorf("err does not say the live database is untouched: %v", err)
			}
			if f.ran("RENAME TABLE `app`") {
				t.Errorf("swapped after a failure: %q", f.execs)
			}
			if got, want := untouched.Is(err), len(f.execs) == 0; got != want {
				t.Errorf("marked as failing before any write = %t, want %t: %q", got, want, f.execs)
			}
			if tt.prepare != f.ran("CREATE DATABASE `app__b2bk_restore`") {
				t.Errorf("staging created = %t, want %t: %q", !tt.prepare, tt.prepare, f.execs)
			}
			if tt.prepare && !f.ran("DROP DATABASE IF EXISTS `app__b2bk_old`") {
				t.Errorf("the empty rollback database was not cleaned up: %q", f.execs)
			}
			if !tt.prepare && f.ran("DROP DATABASE") {
				t.Errorf("a database was dropped before the restore began: %q", f.execs)
			}
		})
	}
}

func TestRestoreReportsAFailedSwap(t *testing.T) {
	t.Parallel()
	f := newFake()
	f.failOn = "RENAME TABLE `app`"
	_, err := runRestore(t, f, goodDump, false, false)
	if err == nil || !strings.Contains(err.Error(), "swap") || !strings.Contains(err.Error(), "was not changed") {
		t.Errorf("err = %v", err)
	}
	if last := f.execs[len(f.execs)-1]; last != "DROP DATABASE IF EXISTS `app__b2bk_old`" {
		t.Errorf("cleanup did not end by dropping the empty rollback database: %q", f.execs)
	}
}

func TestRestoreDryRunNeverSwaps(t *testing.T) {
	t.Parallel()
	f := newFake()
	out, err := runRestore(t, f, goodDump, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if f.ran("RENAME TABLE `app`") {
		t.Errorf("the dry run swapped: %q", f.execs)
	}
	if !f.ran("DROP DATABASE IF EXISTS `app__b2bk_restore`") {
		t.Errorf("staging was not dropped: %q", f.execs)
	}
	if !strings.Contains(out, "the swap would run: RENAME TABLE") {
		t.Errorf("output:\n%s", out)
	}
	want := PrepareStatement("RENAME TABLE `app`.`extra` TO `app__b2bk_old`.`extra`, `app`.`t` TO `app__b2bk_old`.`t`, " +
		"`app__b2bk_restore`.`t` TO `app`.`t`")
	if !f.ran(want) {
		t.Errorf("the server was not asked to prepare the swap: %q", f.execs)
	}
}

// A statement longer than the reader's buffer is read whole before it is
// classified.
func TestFilterLeavesOutADatabaseStatementLongerThanItsBuffer(t *testing.T) {
	t.Parallel()
	f := &dumpFilter{}
	var out strings.Builder
	name := strings.Repeat("x", 40)
	err := f.copy(&out, bufio.NewReaderSize(strings.NewReader("USE `"+name+"`;\n"+goodDump), 16))
	if err != nil || f.database != name || out.String() != "\n"+goodDump {
		t.Errorf("database %q, passed %q, err %v", f.database, out.String(), err)
	}
}

// A swap that reports an error may still have been applied, as when the
// connection drops after the server ran it. The tables are looked at again
// before saying what happened.
func TestRestoreRechecksAFailedSwap(t *testing.T) {
	t.Parallel()

	t.Run("applied", func(t *testing.T) {
		t.Parallel()
		f := newFake()
		f.dropAfterSwap = true
		out, err := runRestore(t, f, goodDump, false, false)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if !strings.Contains(out, "the swap was applied") || !strings.Contains(out, "Lost connection") {
			t.Errorf("the output does not say the swap went through despite the error:\n%s", out)
		}
		if slices.Contains(f.execs, "DROP DATABASE IF EXISTS `app__b2bk_old`") {
			t.Errorf("the rollback copy was dropped: %q", f.execs)
		}
	})

	t.Run("unknown", func(t *testing.T) {
		t.Parallel()
		f := newFake()
		f.dropAfterSwap, f.queriesFail = true, true
		_, err := runRestore(t, f, goodDump, false, false)
		if err == nil || strings.Contains(err.Error(), "was not changed") || !strings.Contains(err.Error(), "unknown") {
			t.Errorf("err = %v", err)
		}
		if slices.Contains(f.execs, "DROP DATABASE IF EXISTS `app__b2bk_old`") {
			t.Errorf("the rollback database was dropped: %q", f.execs)
		}
	})
}
