//go:build postgres

package postgres_test

import (
	"slices"
	"strings"
	"testing"
)

const (
	appRole     = "appuser"
	appPassword = "app-pw"
)

// asApp runs stmts in app as appRole.
func asApp(t *testing.T, stmts string) string {
	t.Helper()
	return docker(t, stmts, "env", "PGPASSWORD="+appPassword, "psql", "-h127.0.0.1", "-U"+appRole, "-d", appDB,
		"-v", "ON_ERROR_STOP=1", "--no-psqlrc", "-qAt")
}

// availableExtensions returns which of names the server can install, in the
// order given, so a server with pgvector and VectorChord tests them too.
func availableExtensions(t *testing.T, names ...string) []string {
	t.Helper()
	have := strings.Fields(sql(t, "postgres", "SELECT name FROM pg_available_extensions"))
	var out []string
	for _, n := range names {
		if slices.Contains(have, n) {
			out = append(out, n)
		}
	}
	return out
}

func createExtensions(t *testing.T, names []string) {
	t.Helper()
	var b strings.Builder
	for _, n := range names {
		b.WriteString(`CREATE EXTENSION IF NOT EXISTS "` + n + `" WITH SCHEMA public CASCADE;`)
	}
	sql(t, appDB, b.String())
}

// recreateAppDB makes app empty and owned by appRole, as CNPG makes an
// application database, with names created in it by the superuser.
func recreateAppDB(t *testing.T, names []string) {
	t.Helper()
	sql(t, "postgres", "DROP DATABASE IF EXISTS "+appDB+" WITH (FORCE);")
	sql(t, "postgres", "DO $$BEGIN IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '"+appRole+"') THEN "+
		"CREATE ROLE "+appRole+" LOGIN PASSWORD '"+appPassword+"'; END IF; END$$;")
	sql(t, "postgres", "CREATE DATABASE "+appDB+" OWNER "+appRole)
	createExtensions(t, names)
}

func extensionOwners(t *testing.T) string {
	t.Helper()
	return sql(t, appDB, "SELECT extname || ' ' || pg_get_userbyid(extowner) FROM pg_extension ORDER BY 1")
}

// The extensions of an application database are created by the superuser, so
// the application user that owns the database and restores it does not own
// them. Immich's database is like this.
func TestPostgresRestoresAsAUserThatDoesNotOwnTheExtensions(t *testing.T) {
	exts := availableExtensions(t, "cube", "earthdistance", "pg_trgm", "unaccent", "uuid-ossp", "vector", "vchord")
	t.Logf("extensions: %v", exts)
	vector := slices.Contains(exts, "vector")
	vchord := slices.Contains(exts, "vchord")

	untrusted := strings.Fields(sql(t, "postgres", "SELECT name FROM pg_available_extensions a "+
		"JOIN pg_available_extension_versions v USING (name) WHERE v.version = a.default_version "+
		"AND v.superuser AND NOT v.trusted AND name IN ('"+strings.Join(exts, "', '")+"')"))
	t.Logf("extensions only a superuser can create: %v", untrusted)
	fingerprint := func(t *testing.T) string {
		t.Helper()
		return dump(t, "--restrict-key=fingerprint", "--no-owner")
	}

	seedApp := func(t *testing.T) {
		t.Helper()
		recreateAppDB(t, exts)
		schema := `
CREATE TABLE place (id int PRIMARY KEY, name text NOT NULL, pos cube NOT NULL,
	uid uuid NOT NULL DEFAULT uuid_generate_v4());
CREATE INDEX place_name_trgm ON place USING gin (name gin_trgm_ops);
INSERT INTO place (id, name, pos) SELECT g, unaccent('Café ' || g), ll_to_earth(g % 90, g % 180)
	FROM generate_series(1, 2000) g;
`
		if vector {
			schema += `CREATE TABLE embedding (id int PRIMARY KEY, v vector(3) NOT NULL);
INSERT INTO embedding SELECT g, ARRAY[g, g + 1, g + 2]::real[]::vector FROM generate_series(1, 2000) g;
`
		}
		if vchord {
			schema += "CREATE INDEX embedding_v ON embedding USING vchordrq (v vector_l2_ops);\n"
		}
		asApp(t, schema)
	}
	seedApp(t)
	owners := extensionOwners(t)
	if !strings.Contains(owners, "earthdistance postgres") {
		t.Fatalf("earthdistance is not owned by postgres:\n%s", owners)
	}
	want := fingerprint(t)
	appDump := func(args ...string) string {
		return docker(t, "", slices.Concat([]string{"env", "PGPASSWORD=" + appPassword, "pg_dump", "-h127.0.0.1",
			"-U" + appRole, "--clean", "--if-exists", "--no-owner"}, args, []string{appDB})...)
	}
	dumps := map[string]string{
		"custom": appDump("--format=custom", "--compress=0"),
		"plain":  appDump("--restrict-key=" + strings.Repeat("k", 16)),
	}
	change := func(t *testing.T) {
		t.Helper()
		asApp(t, "UPDATE place SET name = 'changed'; DELETE FROM place WHERE id > 1000;")
	}

	for _, format := range []string{"custom", "plain"} {
		for _, tt := range []struct {
			name string
			// prepare leaves app as the restore finds it.
			prepare func(t *testing.T)
			user    string
			ok      bool
			stderr  []string
		}{
			{
				name:    "into the live database",
				prepare: func(t *testing.T) { seedApp(t); change(t) },
				user:    appRole,
				ok:      true,
				stderr:  []string{"skipping", "earthdistance", "owned by postgres"},
			},
			{
				name:    "into an empty database with the untrusted extensions",
				prepare: func(t *testing.T) { recreateAppDB(t, untrusted) },
				user:    appRole,
				ok:      true,
				stderr:  []string{"skipping", "earthdistance"},
			},
			{
				name:    "into an empty database",
				prepare: func(t *testing.T) { recreateAppDB(t, nil) },
				user:    appRole,
				stderr:  []string{"does not have", "a superuser must run CREATE EXTENSION"},
			},
			{
				name:    "into an empty database, as the superuser",
				prepare: func(t *testing.T) { recreateAppDB(t, nil) },
				user:    "postgres",
				ok:      true,
			},
		} {
			t.Run(format+", "+tt.name, func(t *testing.T) {
				tt.prepare(t)
				before, beforeOwners := fingerprint(t), extensionOwners(t)
				password := appPassword
				if tt.user == "postgres" {
					password = pgPassword
				}
				out, err := restoreAs(t, tt.user, password, []byte(dumps[format]))
				for _, s := range tt.stderr {
					if !strings.Contains(out, s) {
						t.Errorf("the output does not have %q", s)
					}
				}
				if !tt.ok {
					if err == nil {
						t.Fatal("the restore succeeded")
					}
					named := 0
					for _, n := range untrusted {
						if strings.Contains(out, "CREATE EXTENSION "+n) {
							named++
						}
					}
					// A plain dump is read as it streams, so it stops at the first.
					if want := map[string]int{"custom": len(untrusted), "plain": 1}[format]; named != want {
						t.Errorf("the output names %d of %v, want %d", named, untrusted, want)
					}
					if after := fingerprint(t); after != before {
						t.Errorf("the database changed: %s", firstDiff(before, after))
					}
					if after := extensionOwners(t); after != beforeOwners {
						t.Errorf("the extensions changed:\n%s\nwant:\n%s", after, beforeOwners)
					}
					return
				}
				if err != nil {
					t.Fatalf("restore: %v", err)
				}
				if got := fingerprint(t); got != want {
					t.Errorf("app does not match the backup: %s", firstDiff(want, got))
				}
				if tt.user == appRole && beforeOwners != "" {
					got := extensionOwners(t)
					for line := range strings.SplitSeq(strings.TrimSpace(beforeOwners), "\n") {
						if !strings.Contains(got, line) {
							t.Errorf("extension %q is gone or has a new owner:\n%s", line, got)
						}
					}
				}
			})
		}
	}
}
