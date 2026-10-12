package main

import (
	"cmp"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/USA-RedDragon/dockers/images/restic/cmd/restoredb/postgres"
	"github.com/USA-RedDragon/dockers/images/restic/cmd/restoredb/sqlite"
	"github.com/USA-RedDragon/dockers/images/restic/internal/dumpdb"
	"github.com/USA-RedDragon/dockers/images/restic/internal/testutil"
	"github.com/USA-RedDragon/dockers/images/restic/internal/untouched"
	"github.com/spf13/cobra"
)

const (
	argDryRun    = "--dry-run"
	testDB       = "app"
	engineCNPG   = "cnpg"
	customDump   = "PGDMP\x01\x10\x00\x04\x08\x01\x01\x01"
	plainDump    = "--\n-- PostgreSQL database dump\n--\n\\restrict abc\nSET statement_timeout = 0;\n"
	plainTrailer = "--\n-- PostgreSQL database dump complete\n--\n\n\\unrestrict abc\n\n"
	keylessDump  = "--\n-- PostgreSQL database dump\n--\n\nSET statement_timeout = 0;\n"
)

// superuserAnswer makes a fake psql answer restoredb's question about the
// restoring user as a superuser would.
const superuserAnswer = "case \" $* \" in *\" -At \"*) printf 't\\tapp\\tapp\\n'; exit 0;; esac\n"

func withStdin(s string) *cobra.Command {
	cmd := New()
	cmd.SetIn(strings.NewReader(s))
	return cmd
}

func TestPostgresPicksRestoreToolFromDumpFormat(t *testing.T) {
	const conn = " --host=db --port=5432 --username=app --dbname=app"
	tests := []struct {
		name    string
		stdin   string
		extra   []string
		want    string
		wantErr error
	}{
		{
			name:  "custom format uses pg_restore",
			stdin: customDump,
			want:  "+ pg_restore --clean --if-exists --no-owner --single-transaction" + conn + "\n",
		},
		{
			name:  "plain sql uses psql",
			stdin: plainDump + plainTrailer,
			want:  "+ psql -v ON_ERROR_STOP=1 --single-transaction --no-psqlrc --quiet --output=/dev/null" + conn + "\n",
		},
		{
			name:    "input shorter than the magic is plain sql",
			stdin:   "PGD",
			want:    "+ psql -v ON_ERROR_STOP=1 --single-transaction --no-psqlrc --quiet --output=/dev/null" + conn + "\n",
			wantErr: dumpdb.ErrIncomplete,
		},
		{
			name:    "lowercase magic is plain sql",
			stdin:   "pgdmp",
			want:    "+ psql -v ON_ERROR_STOP=1 --single-transaction --no-psqlrc --quiet --output=/dev/null" + conn + "\n",
			wantErr: dumpdb.ErrIncomplete,
		},
		{
			name:  "psql passthrough after separator",
			stdin: plainDump + plainTrailer,
			extra: []string{"--", "--echo-errors"},
			want:  "+ psql -v ON_ERROR_STOP=1 --single-transaction --no-psqlrc --quiet --output=/dev/null" + conn + " --echo-errors\n",
		},
		{
			name:    "empty input is refused",
			stdin:   "",
			wantErr: postgres.ErrEmptyDump,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.ClearEnv(t)
			args := append([]string{engineCNPG, argDryRun, "-H", "db", "-d", testDB, "-u", testDB}, tt.extra...)
			got, err := testutil.RunGolden(t, withStdin(tt.stdin), args...)
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if tt.wantErr == nil && err != nil {
				t.Fatalf("execute: %v", err)
			}
			if got != tt.want {
				t.Errorf("argv mismatch\n got: %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestPostgresStreamsWholeDumpToRestoreTool(t *testing.T) {
	bin := t.TempDir()
	for _, name := range []string{"psql", "pg_restore"} {
		script := "#!/bin/sh\n" + superuserAnswer + "printf '%s:' " + name + "\nexec cat\n"
		if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	tests := []struct {
		name  string
		stdin string
		tool  string
	}{
		{name: "custom", stdin: customDump + strings.Repeat("x", 1<<16), tool: "pg_restore"},
		{name: "plain", stdin: plainDump + strings.Repeat("SELECT 1;\n", 1<<13) + plainTrailer, tool: "psql"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.ClearEnv(t)
			var stdout strings.Builder
			cmd := withStdin(tt.stdin)
			cmd.SetArgs([]string{engineCNPG, "-H", "db"})
			cmd.SetOut(&stdout)
			cmd.SetErr(io.Discard)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("execute: %v", err)
			}
			if want := tt.tool + ":" + tt.stdin; stdout.String() != want {
				t.Errorf("%s got %d bytes, want %d starting %q", tt.tool, stdout.Len(), len(want), want[:16])
			}
		})
	}
}

func TestGoldenArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "postgres defaults",
			args: []string{"postgres", argDryRun},
			want: "+ pg_restore --clean --if-exists --no-owner --single-transaction" +
				" --host= --port=5432 --username= --dbname=\n",
		},
		{
			name: "cnpg alias matches postgres",
			args: []string{engineCNPG, argDryRun, "-H", "db", "-d", testDB, "-u", "postgres"},
			want: "+ pg_restore --clean --if-exists --no-owner --single-transaction" +
				" --host=db --port=5432 --username=postgres --dbname=app\n",
		},
		{
			name: "mongodb redacts the password",
			args: []string{"mongodb", argDryRun, "-H", "h", "-d", testDB, "-u", "u", "-p", "hunter2"},
			want: "+ mongorestore --archive --drop --authenticationDatabase=" +
				" --host=h --port=27017 --username=u --password=*** --db=app\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.ClearEnv(t)
			got, err := testutil.RunGolden(t, withStdin(customDump), tt.args...)
			if err != nil {
				t.Fatalf("execute: %v", err)
			}
			if got != tt.want {
				t.Errorf("argv mismatch\n got: %q\nwant: %q", got, tt.want)
			}
		})
	}
}

// The operator mounts the CNPG Secret at dumpdb's fixed default path and passes
// no DB_* env, so restore has to resolve every connection detail from files.
func TestPostgresResolvesFromSecretMount(t *testing.T) {
	testutil.ClearEnv(t)

	dir := t.TempDir()
	for name, content := range map[string]string{
		"host":     "postgresql-rw",
		"port":     "5432",
		"dbname":   testDB,
		"username": testDB,
		"password": "hunter2\n", // trailing newline must be trimmed
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	got, err := testutil.RunGolden(t, withStdin(customDump), engineCNPG, argDryRun, "--secret-mount", dir)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := "+ pg_restore --clean --if-exists --no-owner --single-transaction" +
		" --host=postgresql-rw --port=5432 --username=app --dbname=app\n"
	if got != want {
		t.Errorf("argv mismatch\n got: %q\nwant: %q", got, want)
	}
}

func TestExplicitFlagBeatsSecretMount(t *testing.T) {
	testutil.ClearEnv(t)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "host"), []byte("from-file"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := testutil.RunGolden(t, withStdin(customDump),
		engineCNPG, argDryRun, "--secret-mount", dir, "-H", "from-flag")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if want := "--host=from-flag"; !strings.Contains(got, want) {
		t.Errorf("expected %q in %q", want, got)
	}
}

func TestSqliteRefusesToOverwriteWithoutForce(t *testing.T) {
	testutil.ClearEnv(t)

	path := filepath.Join(t.TempDir(), "app.db")
	if err := os.WriteFile(path, []byte("existing data"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := testutil.RunGolden(t, New(), "sqlite", path, argDryRun); !errors.Is(err, sqlite.ErrFileExists) {
		t.Fatalf("expected ErrFileExists, got %v", err)
	}

	// The guarded file must survive the refusal.
	if data, err := os.ReadFile(path); err != nil || string(data) != "existing data" {
		t.Errorf("file was modified: %q, %v", data, err)
	}
}

func TestSqliteRestoresToMissingFile(t *testing.T) {
	testutil.ClearEnv(t)

	path := filepath.Join(t.TempDir(), "app.db")
	got, err := testutil.RunGolden(t, New(), "sqlite", path, argDryRun)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if want := "+ sqlite3 -bail " + path + "\n"; got != want {
		t.Errorf("argv mismatch\n got: %q\nwant: %q", got, want)
	}
}

func TestPostgresNeverLetsPsqlCommitATruncatedPlainDump(t *testing.T) {
	bin := t.TempDir()
	status := filepath.Join(bin, "status")
	script := "#!/bin/sh\n" + superuserAnswer + "cat >/dev/null\necho committed >" + status + "\n"
	if err := os.WriteFile(filepath.Join(bin, "psql"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	tests := []struct {
		name      string
		stdin     string
		committed bool
	}{
		{"cut short", plainDump + strings.Repeat("INSERT INTO t VALUES (1);\n", 1<<12), false},
		{"keyless, cut inside COPY at rows that read as the trailer", keylessDump + "COPY t (v) FROM stdin;\n" +
			strings.Repeat("x\n", 1<<12) + "--\n-- PostgreSQL database dump complete\n--\n\n", false},
		{"keyless, a row that reads as a \\restrict line", keylessDump + "COPY t (v) FROM stdin;\n\\restrict x\n" +
			"\\.\n\n--\n-- PostgreSQL database dump complete\n--\n\n", true},
		{"keyless, complete", keylessDump + "COPY t (v) FROM stdin;\n--\n-- PostgreSQL database dump complete\n--\n" +
			"\\.\n\n--\n-- PostgreSQL database dump complete\n--\n\n", true},
		{"complete", plainDump + strings.Repeat("INSERT INTO t VALUES (1);\n", 1<<12) + plainTrailer, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.ClearEnv(t)
			_ = os.Remove(status)
			cmd := withStdin(tt.stdin)
			cmd.SetArgs([]string{engineCNPG, "-H", "db"})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			err := cmd.Execute()
			_, statErr := os.Stat(status)
			if committed := statErr == nil; committed != tt.committed {
				t.Errorf("psql reached the end of its input = %t, want %t (err %v)", committed, tt.committed, err)
			}
			if tt.committed != (err == nil) {
				t.Errorf("err = %v", err)
			}
		})
	}
}

func TestPlainDumpCompleteOnlyAtTheEnd(t *testing.T) {
	t.Parallel()
	body := plainDump + "COPY t (v) FROM stdin;\nx\n-- PostgreSQL database dump complete\ny\n"
	tests := []struct {
		name string
		dump string
		want bool
	}{
		{"pg_dump 18 with \\unrestrict", body + "\\.\n\n" + plainTrailer, true},
		{"before \\restrict was added", strings.Replace(body, "\\restrict abc\n", "", 1) +
			"\\.\n\n--\n-- PostgreSQL database dump complete\n--\n\n", true},
		{"the trailer text inside data, then cut", body + "z\n", false},
		{"cut inside the trailer", body + "\\.\n\n--\n-- PostgreSQL database dump compl", false},
		{"cut before the closing --", body + "\\.\n\n--\n-- PostgreSQL database dump complete\n", false},
		{"empty", "", false},
		{"trailer-like rows at a cut inside COPY", body + "-- PostgreSQL database dump complete\n--\n", false},
		{"another key", body + "\\.\n\n--\n-- PostgreSQL database dump complete\n--\n\n\\unrestrict xyz\n", false},
	}
	for _, tt := range tests {
		if got := postgres.PlainDumpComplete(postgres.RestrictKey([]byte(tt.dump)), []byte(tt.dump)); got != tt.want {
			t.Errorf("%s: PlainDumpComplete = %t, want %t", tt.name, got, tt.want)
		}
	}
}

func TestPsqlFailingOnItsOwnKeepsItsExitStatus(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\nhead -c 10 >/dev/null\nexit 3\n"
	if err := os.WriteFile(filepath.Join(bin, "psql"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	testutil.ClearEnv(t)
	cmd := withStdin(plainDump + strings.Repeat("SELECT 1;\n", 1<<16) + plainTrailer)
	cmd.SetArgs([]string{engineCNPG, "-H", "db"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()
	exitErr, ok := errors.AsType[*exec.ExitError](err)
	if !ok || exitErr.ExitCode() != 3 {
		t.Errorf("err = %v, want psql's own exit status 3", err)
	}
}

// eofReader records whether it was read to the end.
type eofReader struct {
	r    io.Reader
	done bool
}

func (e *eofReader) Read(p []byte) (int, error) {
	n, err := e.r.Read(p)
	if errors.Is(err, io.EOF) {
		e.done = true
	}
	return n, err
}

// A dry run reads all of its input, as restic dump writes it into a pipeline
// under set -o pipefail, and reports a plain dump a real run would stop.
func TestPostgresDryRunReadsItsInput(t *testing.T) {
	big := strings.Repeat("SELECT 1;\n", 1<<14)
	for _, tt := range []struct {
		name, stdin string
		want        error
	}{
		{"custom", customDump + strings.Repeat("x", 1<<18), nil},
		{"plain", plainDump + big + plainTrailer, nil},
		{"plain, cut", plainDump + big, dumpdb.ErrIncomplete},
		{"plain, meta-command", plainDump + "\\connect other\n" + big + plainTrailer, postgres.ErrMetaCommand},
	} {
		t.Run(tt.name, func(t *testing.T) {
			testutil.ClearEnv(t)
			in := &eofReader{r: strings.NewReader(tt.stdin)}
			cmd := New()
			cmd.SetIn(in)
			_, err := testutil.RunGolden(t, cmd, engineCNPG, argDryRun, "-H", "db")
			if !errors.Is(err, tt.want) || (tt.want == nil) != (err == nil) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
			if !in.done {
				t.Error("the dry run did not read its input to the end")
			}
		})
	}
}

// A fake psql answers as app, which owns pg_trgm but not cube, in a database
// that lacks earthdistance. A fake pg_restore prints its list, its argv and
// its input.
func TestPostgresSkipsExtensionsTheUserCannotManage(t *testing.T) {
	bin := t.TempDir()
	psql := "#!/bin/sh\ncase \"$*\" in\n" +
		"*rolsuper*) printf 'f\\tapp\\tappdb\\n' ;;\n" +
		"*pg_extension*) printf 'cube\\tt\\tpostgres\\tf\\tt\\t\\npg_trgm\\tt\\tapp\\tt\\tt\\t\\n" +
		"earthdistance\\tf\\t\\tf\\tf\\t\\nhstore\\tf\\t\\tf\\tt\\t\\n' ;;\n" +
		"*) printf 'psql:'; exec cat ;;\nesac\n"
	pgRestore := "#!/bin/sh\nfor a; do case \"$a\" in\n" +
		"--list) head -c 16 >/dev/null; cat \"$TOC\"; exit 0 ;;\n" +
		"--use-list=*) printf 'list:'; cat \"${a#--use-list=}\" ;;\nesac; done\n" +
		"printf 'pg_restore:'; exec cat\n"
	for name, script := range map[string]string{"psql": psql, "pg_restore": pgRestore} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	const toc = "2; 3079 16384 EXTENSION - cube \n3617; 0 0 COMMENT - EXTENSION cube \n" +
		"4; 3079 16489 EXTENSION - pg_trgm \n3619; 0 0 COMMENT - EXTENSION pg_trgm \n" +
		"224; 1259 16588 TABLE public t app\n"
	custom := customDump + strings.Repeat("x", 1<<17)
	plain := plainDump + "DROP EXTENSION IF EXISTS cube;\nDROP EXTENSION IF EXISTS pg_trgm;\n" +
		"CREATE EXTENSION IF NOT EXISTS cube WITH SCHEMA public;\nCOMMENT ON EXTENSION cube IS 'cubes';\n" +
		"CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;\nCOMMENT ON EXTENSION pg_trgm IS 'trigrams';\n" +
		plainTrailer
	tests := []struct {
		name, toc, stdin, want string
		wantErr                error
	}{
		{
			name:  "custom",
			toc:   toc,
			stdin: custom,
			want: "list:;2; 3079 16384 EXTENSION - cube \n;3617; 0 0 COMMENT - EXTENSION cube \n" +
				"4; 3079 16489 EXTENSION - pg_trgm \n3619; 0 0 COMMENT - EXTENSION pg_trgm \n" +
				"224; 1259 16588 TABLE public t app\npg_restore:" + custom,
		},
		{
			name:  "custom, nothing to skip",
			toc:   "4; 3079 16489 EXTENSION - pg_trgm \n8; 3079 1 EXTENSION - hstore \n",
			stdin: custom,
			want:  "pg_restore:" + custom,
		},
		{
			name:    "custom, a missing extension",
			toc:     toc + "3; 3079 16473 EXTENSION - earthdistance \n",
			stdin:   custom,
			wantErr: postgres.ErrMissingExtension,
		},
		{
			name:  "plain",
			stdin: plain,
			want: "psql:" + plainDump + "DROP EXTENSION IF EXISTS pg_trgm;\n" +
				"CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;\nCOMMENT ON EXTENSION pg_trgm IS 'trigrams';\n" +
				plainTrailer,
		},
		{
			name:    "plain, a missing extension",
			stdin:   plainDump + "CREATE EXTENSION IF NOT EXISTS earthdistance WITH SCHEMA public;\n" + plainTrailer,
			wantErr: postgres.ErrMissingExtension,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.ClearEnv(t)
			tocFile := filepath.Join(t.TempDir(), "toc")
			if err := os.WriteFile(tocFile, []byte(tt.toc), 0o600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("TOC", tocFile)
			var stdout, stderr strings.Builder
			cmd := withStdin(tt.stdin)
			cmd.SetArgs([]string{engineCNPG, "-H", "db"})
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			err := cmd.Execute()
			t.Logf("stderr:\n%s", stderr.String())
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				if !strings.Contains(err.Error(), "CREATE EXTENSION earthdistance") {
					t.Errorf("the error does not name earthdistance: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("execute: %v", err)
			}
			if stdout.String() != tt.want {
				t.Errorf("got %q\nwant %q", stdout.String()[:min(stdout.Len(), 400)], tt.want[:min(len(tt.want), 400)])
			}
			if strings.Contains(tt.want, ";2;") && !strings.Contains(stderr.String(), "skipping") {
				t.Errorf("stderr does not say what was skipped")
			}
		})
	}
}

// Fake tools for TestExitCodeTellsAFailureBeforeAnyWrite. psql answers as a
// user that is not a superuser, in a database without earthdistance, and fails
// to connect when PSQL_DOWN is set; run as the restore it records what it was
// given. pg_restore lists $TOC, or fails to when it is empty, and records what
// it was given.
const (
	exitPsql = "#!/bin/sh\n[ -n \"$PSQL_DOWN\" ] && { echo 'connection refused' >&2; exit 2; }\n" +
		"case \"$*\" in\n" +
		"*rolsuper*) printf 'f\\tapp\\tappdb\\n' ;;\n" +
		"*pg_extension*) printf 'earthdistance\\tf\\t\\tf\\tf\\t\\npg_trgm\\tt\\tapp\\tt\\tt\\t\\n' ;;\n" +
		"*) cat >\"$WROTE\"; exit \"${PSQL_EXIT:-0}\" ;;\nesac\n"
	exitPgRestore = "#!/bin/sh\nfor a; do [ \"$a\" = --list ] && { [ -s \"$TOC\" ] || exit 1; cat >/dev/null; " +
		"cat \"$TOC\"; exit 0; }; done\ncat >\"$WROTE\"; exit \"${PG_RESTORE_EXIT:-0}\"\n"
)

func TestExitCodeTellsAFailureBeforeAnyWrite(t *testing.T) {
	bin := t.TempDir()
	for name, script := range map[string]string{"psql": exitPsql, "pg_restore": exitPgRestore} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	noRestore := t.TempDir()
	if err := os.WriteFile(filepath.Join(noRestore, "psql"), []byte(exitPsql), 0o700); err != nil {
		t.Fatal(err)
	}
	path := os.Getenv("PATH")
	existing := filepath.Join(t.TempDir(), "app.db")
	if err := os.WriteFile(existing, []byte("existing data"), 0o600); err != nil {
		t.Fatal(err)
	}
	const pgTrgm = "4; 3079 16489 EXTENSION - pg_trgm \n"
	missing := pgTrgm + "3; 3079 16473 EXTENSION - earthdistance \n"
	custom := customDump + strings.Repeat("x", 1<<12)
	plain := plainDump + "CREATE TABLE t (v int);\n" + plainTrailer
	pg := []string{engineCNPG, "-H", "db"}

	tests := []struct {
		name  string
		args  []string
		stdin string
		env   map[string]string
		path  string
		want  int
		why   string
	}{
		{name: "an unknown flag", args: []string{engineCNPG, "-H", "db", "--no-such-flag"}, stdin: custom, want: untouched.ExitCode,
			why: "unknown flag"},
		{name: "sqlite without a file", args: []string{"sqlite"}, want: untouched.ExitCode, why: "accepts 1 arg"},
		{name: "sqlite over a file without --force", args: []string{"sqlite", existing}, stdin: "x",
			want: untouched.ExitCode, why: "already exists"},
		{name: "an empty dump", args: pg, want: untouched.ExitCode, why: "no dump"},
		{name: "a database it cannot connect to", args: pg, stdin: custom,
			env: map[string]string{"PSQL_DOWN": "1"}, want: untouched.ExitCode, why: "connection refused"},
		{name: "a custom dump pg_restore cannot list", args: pg, stdin: custom,
			want: untouched.ExitCode, why: "list the dump"},
		{name: "a custom dump that needs a missing extension", args: pg, stdin: custom,
			env: map[string]string{"TOC": missing}, want: untouched.ExitCode, why: "CREATE EXTENSION earthdistance"},
		{name: "a restore tool that cannot start", args: pg, stdin: custom,
			env: map[string]string{"TOC": pgTrgm}, path: noRestore, want: untouched.ExitCode,
			why: "executable file not found"},
		{name: "a plain dump that needs a missing extension, found as psql reads it", args: pg,
			stdin: plainDump + "CREATE EXTENSION IF NOT EXISTS earthdistance WITH SCHEMA public;\n" + plainTrailer,
			want:  1},
		{name: "a plain dump cut short", args: pg, stdin: plainDump + "CREATE TABLE t (v int);\n",
			want: 1},
		{name: "psql failing in the restore", args: pg, stdin: plain,
			env: map[string]string{"PSQL_EXIT": "3"}, want: 3},
		{name: "pg_restore failing in the restore", args: pg, stdin: custom,
			env: map[string]string{"TOC": pgTrgm, "PG_RESTORE_EXIT": "1"}, want: 1},
		{name: "a custom dump restored", args: pg, stdin: custom,
			env: map[string]string{"TOC": pgTrgm}, want: 0},
		{name: "a plain dump restored", args: pg, stdin: plain, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.ClearEnv(t)
			dir := t.TempDir()
			wrote := filepath.Join(dir, "wrote")
			toc := filepath.Join(dir, "toc")
			if err := os.WriteFile(toc, []byte(tt.env["TOC"]), 0o600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("TOC", toc)
			t.Setenv("WROTE", wrote)
			for _, k := range []string{"PSQL_DOWN", "PSQL_EXIT", "PG_RESTORE_EXIT"} {
				t.Setenv(k, tt.env[k])
			}
			t.Setenv("PATH", cmp.Or(tt.path, bin+string(os.PathListSeparator)+path))
			cmd := withStdin(tt.stdin)
			cmd.SetArgs(tt.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SilenceErrors = true
			err := cmd.Execute()
			got := 0
			if err != nil {
				got = exitCode(err)
			}
			if got != tt.want {
				t.Errorf("exit %d, want %d (err %v)", got, tt.want, err)
			}
			if tt.why != "" && (err == nil || !strings.Contains(err.Error(), tt.why)) {
				t.Errorf("err = %v, want one that says %q", err, tt.why)
			}
			_, statErr := os.Stat(wrote)
			if gave := statErr == nil; got == untouched.ExitCode && gave {
				t.Errorf("exit %d, but the restore tool was given the dump", got)
			}
		})
	}
}
