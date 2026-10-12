package postgres

import (
	"bufio"
	"errors"
	"slices"
	"strings"
	"testing"
)

func testExtensions() map[string]Extension {
	return map[string]Extension{
		"cube":          {Installed: true, Owner: "postgres", Dependents: []string{"earthdistance"}},
		"earthdistance": {Installed: true, Owner: "postgres"},
		"pg_trgm":       {Installed: true, Owner: "app", Owned: true, Creatable: true},
		"uuid-ossp":     {Installed: true, Owner: "postgres", Creatable: true},
		"citext":        {Installed: true, Owner: "app", Owned: true, Dependents: []string{"vchord"}},
		"vchord":        {Installed: true, Owner: "postgres"},
		"hstore":        {Creatable: true},
		"postgis":       {},
	}
}

func TestExtensionPolicyActions(t *testing.T) {
	t.Parallel()
	p := NewExtensionPolicy("app", "appdb", testExtensions(), nil)
	for _, tt := range []struct {
		name string
		want extAction
	}{
		{"cube", extSkip},
		{"earthdistance", extSkip},
		{"uuid-ossp", extSkip},
		{"pg_trgm", extKeep},
		{"citext", extSkip},
		{"vchord", extSkip},
		{"hstore", extKeep},
		{"postgis", extMissing},
		{"unknown", extMissing},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := p.action(tt.name); got != tt.want {
				t.Errorf("action(%q) = %d, want %d", tt.name, got, tt.want)
			}
		})
	}
}

func TestExtensionPolicySkipsAnOwnedExtensionWhoseDependentIsNotInTheDump(t *testing.T) {
	t.Parallel()
	exts := map[string]Extension{
		"cube":          {Installed: true, Owned: true, Dependents: []string{"earthdistance"}},
		"earthdistance": {Installed: true, Owned: true},
	}
	for _, tt := range []struct {
		name   string
		inDump []string
		want   extAction
	}{
		{"dependent in the dump", []string{"cube", "earthdistance"}, extKeep},
		{"dependent not in the dump", []string{"cube"}, extSkip},
		{"dump unknown", nil, extKeep},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := NewExtensionPolicy("app", "appdb", exts, tt.inDump).action("cube"); got != tt.want {
				t.Errorf("action(cube) = %d, want %d", got, tt.want)
			}
		})
	}
}

const testTOC = `;
; Selected TOC Entries:
;
2; 3079 16384 EXTENSION - cube
3617; 0 0 COMMENT - EXTENSION cube
4; 3079 16489 EXTENSION - pg_trgm
3619; 0 0 COMMENT - EXTENSION pg_trgm
5; 3079 16570 EXTENSION - uuid-ossp
3621; 0 0 COMMENT - EXTENSION "uuid-ossp"
224; 1259 16588 TABLE public cube app
3609; 0 16588 TABLE DATA public big app
`

func TestFilterTOC(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name        string
		toc         string
		wantSkipped []string
		wantErr     string
	}{
		{
			name: "skips the entries of extensions the user cannot manage",
			toc:  testTOC,
			wantSkipped: []string{
				"2; 3079 16384 EXTENSION - cube", "3617; 0 0 COMMENT - EXTENSION cube",
				"5; 3079 16570 EXTENSION - uuid-ossp", `3621; 0 0 COMMENT - EXTENSION "uuid-ossp"`,
			},
		},
		{
			name: "stops at a missing extension the user cannot create",
			toc:  testTOC + "7; 3079 1 EXTENSION - postgis \n8; 3079 2 EXTENSION - hstore \n",
			wantErr: `database appdb does not have postgis, and app cannot create it; ` +
				`a superuser must run CREATE EXTENSION postgis in it first`,
		},
		{
			name:    "names every missing extension",
			toc:     "7; 3079 1 EXTENSION - postgis \n8; 3079 2 EXTENSION - My-Ext \n",
			wantErr: `CREATE EXTENSION postgis, CREATE EXTENSION "My-Ext" in it first`,
		},
		{
			name: "keeps a missing extension the user can create",
			toc:  "8; 3079 2 EXTENSION - hstore \n9; 0 0 COMMENT - EXTENSION hstore \n",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := NewExtensionPolicy("app", "appdb", testExtensions(), TOCExtensions([]byte(tt.toc)))
			out, skipped, err := p.FilterTOC([]byte(tt.toc))
			if tt.wantErr != "" {
				if !errors.Is(err, ErrMissingExtension) || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(skipped, tt.wantSkipped) {
				t.Errorf("skipped = %q, want %q", skipped, tt.wantSkipped)
			}
			for line := range strings.SplitSeq(string(out), "\n") {
				name, ok := tocEntry(line)
				commented := strings.HasPrefix(line, ";")
				if ok && commented {
					t.Errorf("entry %q of %s still reads as an entry", line, name)
				}
				if ok && p.action(name) == extSkip {
					t.Errorf("entry %q is still in the list", line)
				}
			}
			if got := strings.Count(string(out), "\n"); got != strings.Count(tt.toc, "\n") {
				t.Errorf("the list has %d lines, want %d", got, strings.Count(tt.toc, "\n"))
			}
		})
	}
}

func TestPlainFilterSkipsExtensionStatements(t *testing.T) {
	t.Parallel()
	head := "SET row_security = off;\n\n"
	for _, tt := range []struct {
		name, dump, want string
		skipped          []string
		wantErr          string
	}{
		{
			name: "a clean dump",
			dump: head + "DROP TABLE IF EXISTS public.t;\nDROP EXTENSION IF EXISTS \"uuid-ossp\";\n" +
				"DROP EXTENSION IF EXISTS pg_trgm;\nDROP EXTENSION IF EXISTS cube;\n" +
				"--\n-- Name: cube; Type: EXTENSION; Schema: -; Owner: -\n--\n\n" +
				"CREATE EXTENSION IF NOT EXISTS cube WITH SCHEMA public;\n" +
				"COMMENT ON EXTENSION cube IS 'data type for multidimensional cubes';\n" +
				"CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;\n" +
				"COMMENT ON EXTENSION pg_trgm IS 'trigrams';\n" +
				"CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\" WITH SCHEMA public;\n" +
				"COMMENT ON EXTENSION \"uuid-ossp\" IS 'generate universally unique identifiers (UUIDs)';\n" +
				"CREATE TABLE public.t (v text);\n",
			want: head + "DROP TABLE IF EXISTS public.t;\nDROP EXTENSION IF EXISTS pg_trgm;\n" +
				"--\n-- Name: cube; Type: EXTENSION; Schema: -; Owner: -\n--\n\n" +
				"CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;\n" +
				"COMMENT ON EXTENSION pg_trgm IS 'trigrams';\n" +
				"CREATE TABLE public.t (v text);\n",
			skipped: []string{
				`DROP EXTENSION IF EXISTS "uuid-ossp";`, "DROP EXTENSION IF EXISTS cube;",
				"CREATE EXTENSION IF NOT EXISTS cube WITH SCHEMA public;",
				"COMMENT ON EXTENSION cube IS 'data type for multidimensional cubes';",
				`CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;`,
				`COMMENT ON EXTENSION "uuid-ossp" IS 'generate universally unique identifiers (UUIDs)';`,
			},
		},
		{
			name: "a comment over several lines with a quote",
			dump: "COMMENT ON EXTENSION cube IS 'it''s\nCOMMIT;\nDROP EXTENSION IF EXISTS pg_trgm;\n';\nSELECT 1;\n",
			want: "SELECT 1;\n",
			skipped: []string{
				"COMMENT ON EXTENSION cube IS 'it''s",
			},
		},
		{
			name: "COPY rows and strings that read as extension statements",
			dump: "COPY public.t (v) FROM stdin;\nDROP EXTENSION IF EXISTS cube;\n\\.\n" +
				"SELECT '\nDROP EXTENSION IF EXISTS cube;\n';\n",
			want: "COPY public.t (v) FROM stdin;\nDROP EXTENSION IF EXISTS cube;\n\\.\n" +
				"SELECT '\nDROP EXTENSION IF EXISTS cube;\n';\n",
		},
		{
			name:    "a missing extension the user cannot create",
			dump:    head + "DROP EXTENSION IF EXISTS postgis;\n",
			want:    head,
			wantErr: "a superuser must run CREATE EXTENSION postgis in it first",
		},
		{
			name: "a missing extension the user can create",
			dump: "CREATE EXTENSION IF NOT EXISTS hstore WITH SCHEMA public;\n",
			want: "CREATE EXTENSION IF NOT EXISTS hstore WITH SCHEMA public;\n",
		},
		{
			name:    "a transaction statement after a skipped one",
			dump:    "DROP EXTENSION IF EXISTS cube;\nCOMMIT; SELECT 1;\n",
			want:    "",
			skipped: []string{"DROP EXTENSION IF EXISTS cube;"},
			wantErr: ErrTransactionControl.Error(),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var out strings.Builder
			var skipped []string
			f := NewPlainFilter()
			f.Extensions = NewExtensionPolicy("app", "appdb", testExtensions(), nil)
			f.Skipped = func(statement, _ string) { skipped = append(skipped, statement) }
			err := f.Copy(&out, bufio.NewReaderSize(strings.NewReader(tt.dump), 16))
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatal(err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Fatalf("err = %v, want %q", err, tt.wantErr)
			}
			if out.String() != tt.want {
				t.Errorf("output:\n%s\nwant:\n%s", out.String(), tt.want)
			}
			if !slices.Equal(skipped, tt.skipped) {
				t.Errorf("skipped = %q, want %q", skipped, tt.skipped)
			}
		})
	}
}
