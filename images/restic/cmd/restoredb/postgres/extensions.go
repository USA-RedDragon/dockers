package postgres

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
)

// ErrMissingExtension is returned when the dump needs an extension that the
// target database does not have and the restoring user cannot create.
var ErrMissingExtension = errors.New("the dump needs extensions the restoring user cannot create")

// Extension is what the target database says about one extension.
type Extension struct {
	Owner     string
	Installed bool
	// Owned is set when the restoring user may drop, create and comment on
	// the installed extension: it owns it or has the privileges of its owner.
	Owned bool
	// Creatable is set when the restoring user may create the extension.
	Creatable bool
	// Dependents are the installed extensions that require this one.
	Dependents []string
}

type extAction int

const (
	extKeep extAction = iota
	extSkip
	extMissing
)

// ExtensionPolicy decides which extension statements of a dump a restore
// that is not run by a superuser keeps. The extensions of the dump that the
// target has and the user cannot manage keep their place in the target, and
// the dump's DROP, CREATE and COMMENT for them are skipped. An extension the
// target lacks and the user cannot create stops the restore.
type ExtensionPolicy struct {
	User, Database string
	Extensions     map[string]Extension
	skipped        map[string]bool
}

// NewExtensionPolicy returns the policy for exts. inDump, when known, lists
// the extensions the dump drops and creates. An extension the user owns is
// skipped as well when an installed extension that requires it stays, since
// it could not be dropped.
func NewExtensionPolicy(user, database string, exts map[string]Extension, inDump []string) *ExtensionPolicy {
	p := &ExtensionPolicy{User: user, Database: database, Extensions: exts, skipped: map[string]bool{}}
	for name, e := range exts {
		if e.Installed && !e.Owned {
			p.skipped[name] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for name, e := range exts {
			if !e.Installed || p.skipped[name] {
				continue
			}
			for _, d := range e.Dependents {
				if p.skipped[d] || inDump != nil && !slices.Contains(inDump, d) {
					p.skipped[name] = true
					changed = true
					break
				}
			}
		}
	}
	return p
}

func (p *ExtensionPolicy) action(name string) extAction {
	e := p.Extensions[name]
	switch {
	case p.skipped[name]:
		return extSkip
	case e.Installed || e.Creatable:
		return extKeep
	}
	return extMissing
}

// reason says why the entries of a skipped extension are skipped.
func (p *ExtensionPolicy) reason(name string) string {
	if e := p.Extensions[name]; !e.Owned {
		return fmt.Sprintf("the database has it, owned by %s, and %s cannot drop or replace it", e.Owner, p.User)
	}
	return "the database has it, and an extension that requires it is kept"
}

func (p *ExtensionPolicy) missingError(names []string) error {
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = quoteIdent(n)
	}
	return fmt.Errorf("%w: database %s does not have %s, and %s cannot create %s; "+
		"a superuser must run CREATE EXTENSION %s in it first",
		ErrMissingExtension, p.Database, strings.Join(quoted, ", "), p.User, plural(len(names), "it", "them"),
		strings.Join(quoted, ", CREATE EXTENSION "))
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

var plainIdent = regexp.MustCompile(`^[a-z_][a-z0-9_$]*$`)

func quoteIdent(name string) string {
	if plainIdent.MatchString(name) {
		return name
	}
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func unquoteIdent(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return strings.ReplaceAll(s[1:len(s)-1], `""`, `"`)
	}
	return s
}

const superuserQuery = `SELECT rolsuper, current_user, current_database() FROM pg_roles WHERE rolname = current_user`

const extensionQuery = `SELECT coalesce(e.extname, a.name),
	e.oid IS NOT NULL,
	coalesce(pg_get_userbyid(e.extowner), ''),
	coalesce(pg_has_role(e.extowner, 'USAGE'), false),
	coalesce(v.trusted AND has_database_privilege(current_database(), 'CREATE') OR NOT v.superuser, false),
	coalesce((SELECT string_agg(d.extname, ',') FROM pg_depend p JOIN pg_extension d ON d.oid = p.objid
		WHERE p.classid = 'pg_extension'::regclass AND p.refclassid = 'pg_extension'::regclass
		AND p.refobjid = e.oid), '')
FROM pg_extension e
FULL JOIN pg_available_extensions a ON a.name = e.extname
LEFT JOIN pg_available_extension_versions v ON v.name = a.name AND v.version = a.default_version`

// QueryExtensions asks the target database, through psql with conn and env,
// whether the restoring user is a superuser and, when it is not, what it may
// do with each extension.
func QueryExtensions(ctx context.Context, conn, env []string) (super bool, user, database string,
	exts map[string]Extension, err error,
) {
	run := func(query string) ([][]string, error) {
		c := exec.CommandContext(ctx, "psql", slices.Concat([]string{"--no-psqlrc", "-At", "-F", "\t",
			"-v", "ON_ERROR_STOP=1", "-c", query}, conn)...)
		c.Env = append(os.Environ(), env...)
		var stderr bytes.Buffer
		c.Stderr = &stderr
		out, err := c.Output()
		if err != nil {
			return nil, fmt.Errorf("query the database's extensions: %w: %s", err, strings.TrimSpace(stderr.String()))
		}
		var rows [][]string
		for line := range strings.SplitSeq(strings.TrimRight(string(out), "\n"), "\n") {
			if line != "" {
				rows = append(rows, strings.Split(line, "\t"))
			}
		}
		return rows, nil
	}
	rows, err := run(superuserQuery)
	if err != nil {
		return false, "", "", nil, err
	}
	if len(rows) != 1 || len(rows[0]) != 3 {
		return false, "", "", nil, fmt.Errorf("query the database's extensions: unexpected answer %q", rows)
	}
	super, user, database = rows[0][0] == "t", rows[0][1], rows[0][2]
	if super {
		return super, user, database, nil, nil
	}
	if rows, err = run(extensionQuery); err != nil {
		return false, "", "", nil, err
	}
	exts = map[string]Extension{}
	for _, r := range rows {
		if len(r) != 6 {
			return false, "", "", nil, fmt.Errorf("query the database's extensions: unexpected row %q", r)
		}
		e := Extension{Installed: r[1] == "t", Owner: r[2], Owned: r[3] == "t", Creatable: r[4] == "t"}
		if r[5] != "" {
			e.Dependents = strings.Split(r[5], ",")
		}
		exts[r[0]] = e
	}
	return super, user, database, exts, nil
}

var (
	tocExtension = regexp.MustCompile(`^\d+; \d+ \d+ EXTENSION - (\S+)`)
	tocComment   = regexp.MustCompile(`^\d+; \d+ \d+ COMMENT - EXTENSION ("(?:[^"]|"")+"|\S+)`)
)

// tocEntry returns the extension a pg_restore -l line creates or comments on.
func tocEntry(line string) (string, bool) {
	if m := tocExtension.FindStringSubmatch(line); m != nil {
		return m[1], true
	}
	if m := tocComment.FindStringSubmatch(line); m != nil {
		return unquoteIdent(m[1]), true
	}
	return "", false
}

// TOCExtensions lists the extensions a pg_restore -l listing creates.
func TOCExtensions(toc []byte) []string {
	var names []string
	for line := range strings.SplitSeq(string(toc), "\n") {
		if m := tocExtension.FindStringSubmatch(line); m != nil {
			names = append(names, m[1])
		}
	}
	return names
}

// FilterTOC turns the entries of a pg_restore -l listing for the extensions
// p skips into comments, so pg_restore -L neither drops, creates nor comments
// on them. It returns the new listing and the entries it skipped.
func (p *ExtensionPolicy) FilterTOC(toc []byte) ([]byte, []string, error) {
	var missing []string
	for _, name := range TOCExtensions(toc) {
		if p.action(name) == extMissing {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return nil, nil, p.missingError(missing)
	}
	var out bytes.Buffer
	var skipped []string
	for line := range strings.SplitAfterSeq(string(toc), "\n") {
		if name, ok := tocEntry(line); ok && p.action(name) == extSkip {
			skipped = append(skipped, strings.TrimSpace(line))
			out.WriteString(";")
		}
		out.WriteString(line)
	}
	return out.Bytes(), skipped, nil
}

// PrepareCustom reads the table of contents at the head of the custom-format
// dump in with pg_restore -l and filters it with FilterTOC. It returns a
// reader that gives back the whole dump, the listing for pg_restore -L, or nil
// when nothing is skipped, and the entries it skips. Only what pg_restore -l
// reads is held in memory.
func (p *ExtensionPolicy) PrepareCustom(ctx context.Context, in io.Reader, stderr io.Writer) (io.Reader, []byte,
	[]string, error,
) {
	var head bytes.Buffer
	c := exec.CommandContext(ctx, "pg_restore", "--list")
	c.Stdin = io.TeeReader(in, &head)
	c.Stderr = stderr
	toc, err := c.Output()
	dump := io.MultiReader(&head, in)
	if err != nil {
		return dump, nil, nil, fmt.Errorf("list the dump: %w", err)
	}
	list, skipped, err := p.FilterTOC(toc)
	if err != nil || len(skipped) == 0 {
		return dump, nil, nil, err
	}
	return dump, list, skipped, nil
}

var plainExtension = regexp.MustCompile(
	`^(?:DROP EXTENSION IF EXISTS|DROP EXTENSION|CREATE EXTENSION IF NOT EXISTS|CREATE EXTENSION|COMMENT ON EXTENSION) ` +
		`("(?:[^"]|"")+"|[A-Za-z_][A-Za-z0-9_$]*)(?:[ ;]|$)`)

// plainEntry returns the extension a statement of a plain dump that starts on
// line drops, creates or comments on.
func plainEntry(line []byte) (string, bool) {
	m := plainExtension.FindSubmatch(line)
	if m == nil {
		return "", false
	}
	return unquoteIdent(string(m[1])), true
}
