package postgres

import (
	"bufio"
	"errors"
	"strings"
	"testing"
)

const (
	loBegin  = "BEGIN;\n"
	loCommit = "COMMIT;\n"
)

func filterPlain(t *testing.T, dump string) (string, *PlainFilter, error) {
	t.Helper()
	var out strings.Builder
	f := NewPlainFilter()
	err := f.Copy(&out, bufio.NewReaderSize(strings.NewReader(dump), 16))
	return out.String(), f, err
}

// pg_dump's own BEGIN; and COMMIT; lines between statements become comments,
// and the same text anywhere psql reads as data stays as it is.
func TestPlainFilterNeutralizesOnlyTopLevelTransactionLines(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, dump string
		left       []string
	}{
		{
			name: "large objects",
			dump: "SELECT pg_catalog.lo_create('16391');\n\n" + loBegin + "\n" +
				"SELECT pg_catalog.lo_open('16391', 131072);\nSELECT pg_catalog.lo_close(0);\n\n" + loCommit + "\n",
			left: []string{"-- BEGIN;\n", "-- COMMIT;\n"},
		},
		{
			name: "CRLF and no final newline",
			dump: "BEGIN;\r\nSELECT 1;\r\nCOMMIT;",
			left: []string{"-- BEGIN;\r\n", "-- COMMIT;"},
		},
		{
			name: "COPY data rows",
			dump: "COPY public.t (v) FROM stdin;\n" + loBegin + loCommit + "\\.\n",
		},
		{
			name: "a function body",
			dump: "CREATE FUNCTION public.f() RETURNS void\n    LANGUAGE plpgsql\n    AS $_$\n" + loBegin +
				"  PERFORM 1;\n" + loCommit + "END;\n$_$;\n",
		},
		{
			name: "a multi-line string",
			dump: "COMMENT ON TABLE public.t IS 'first\n" + loCommit + "last';\n",
		},
		{
			name: "an escaped quote in an E string",
			dump: "SELECT E'it\\'s\n" + loCommit + "';\n",
		},
		{
			name: "a doubled quote",
			dump: "SELECT 'it''s\n" + loCommit + "';\n",
		},
		{
			name: "a continued string",
			dump: "SELECT E'a'\n'\\'\n" + loCommit + "';\n",
		},
		{
			name: "a quoted identifier",
			dump: "CREATE TABLE \"x\n" + loCommit + "\" (i int);\n",
		},
		{
			name: "a nested block comment",
			dump: "/* a /* b */\n" + loCommit + "*/\nSELECT 1;\n",
		},
		{
			name: "a statement that is not finished",
			dump: "SELECT 1\n" + loBegin + ";\n",
		},
		{
			name: "after SET standard_conforming_strings = off",
			dump: "SET standard_conforming_strings = off;\nSELECT 'a\\';\n" + loCommit + "';\n",
		},
		{
			name: "after SET standard_conforming_strings to a quoted off",
			dump: "SET standard_conforming_strings = \"off\";\nSELECT 'a\\';\n" + loCommit + "';\n",
		},
		{
			name: "the semicolons of a SQL-standard function body",
			dump: "CREATE FUNCTION public.g() RETURNS integer\n    LANGUAGE sql\n    BEGIN ATOMIC\n SELECT 1;\n" +
				"END;\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, f, err := filterPlain(t, tt.dump)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			want := tt.dump
			for _, line := range tt.left {
				want = strings.Replace(want, strings.TrimPrefix(line, "-- "), "\x00", 1)
			}
			for _, line := range tt.left {
				want = strings.Replace(want, "\x00", line, 1)
			}
			if got != want {
				t.Errorf("got\n%q\nwant\n%q", got, want)
			}
			if !f.Ended() {
				t.Errorf("the filter does not end between statements: %+v", f.lex)
			}
		})
	}
}

func TestPlainFilterRefusesOtherTransactionControl(t *testing.T) {
	t.Parallel()
	for _, stmt := range []string{
		"ROLLBACK;", "END;", "commit;", "  COMMIT;", "COMMIT AND CHAIN;", "ABORT;", "START TRANSACTION;",
		"PREPARE TRANSACTION 'x';", "COMMIT PREPARED 'x';", "/* c */ COMMIT;", "BEGIN\n;",
		"BEGIN; SELECT 1;",
	} {
		dump := "CREATE TABLE t (i int);\n" + stmt + "\nDROP TABLE t;\n"
		got, _, err := filterPlain(t, dump)
		if !errors.Is(err, ErrTransactionControl) {
			t.Errorf("%q: err = %v", stmt, err)
		}
		if strings.Contains(got, "DROP") {
			t.Errorf("%q: the filter went on after it: %q", stmt, got)
		}
	}
	for _, stmt := range []string{
		"SELECT pg_catalog.set_config('standard_conforming_strings', 'off', false);",
		"SELECT 1, set_config('Standard_Conforming_Strings', 'off', false);",
		"SELECT set_config($$standard_conforming_strings$$, 'off', false);",
		"SET standard_conforming_strings TO DEFAULT, x;",
	} {
		if _, _, err := filterPlain(t, stmt+"\n"); !errors.Is(err, ErrUnsupportedDump) {
			t.Errorf("%q: err = %v", stmt, err)
		}
	}
	for _, stmt := range []string{
		"PREPARE p AS SELECT 1;", "SELECT 'COMMIT;';", "SAVEPOINT s;",
		"INSERT INTO docs VALUES ('set standard_conforming_strings to off');",
		"SET standard_conforming_strings = on;", "RESET standard_conforming_strings;",
	} {
		if _, _, err := filterPlain(t, stmt+"\n"); err != nil {
			t.Errorf("%q: err = %v", stmt, err)
		}
	}
}

// COPY data ends only at a line that is \. alone.
func TestPlainFilterFollowsCopyData(t *testing.T) {
	t.Parallel()
	dump := "COPY public.t (v) FROM stdin;\nx\\.\n\\. \n \\.\n\\.\n" + loCommit
	got, f, err := filterPlain(t, dump)
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.Replace(dump, "\\.\n"+loCommit, "\\.\n-- "+loCommit, 1); got != want {
		t.Errorf("got %q", got)
	}
	if f.inCopy {
		t.Error("still in COPY")
	}
	for _, copyIn := range []string{
		"COPY t FROM STDIN;", "copy public.t (a, b) from stdin with (format csv);", "COPY BINARY t FROM stdin;",
		"COPY \"T\" FROM stdin;",
	} {
		_, f, err := filterPlain(t, copyIn+"\nrow\n")
		if err != nil || !f.inCopy {
			t.Errorf("%q: in COPY = %t, err = %v", copyIn, f.inCopy, err)
		}
	}
	for _, other := range []string{"COPY t TO stdout;", "COPY (SELECT 1 FROM stdin) TO stdout;", "SELECT 1;"} {
		_, f, err := filterPlain(t, other+"\n")
		if err != nil || f.inCopy {
			t.Errorf("%q: in COPY = %t, err = %v", other, f.inCopy, err)
		}
	}
	if _, _, err := filterPlain(t, "COPY t FROM stdin; SELECT 1;\n"); !errors.Is(err, ErrUnsupportedDump) {
		t.Errorf("text after COPY on its line: err = %v", err)
	}
}

func TestLexerDollarQuotes(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		sql  string
		idle bool
	}{
		{"SELECT $$a;b$$;", true},
		{"SELECT $tag$a$$b$ta$tag$;", true},
		{"SELECT $tag$a$$b$ta$TAG$;", false},
		{"SELECT a$b$ ;", true},
		{"SELECT $1;", true},
		{"SELECT 1$a$;$a$;", true},
		{"SELECT $a b;", true},
		{"SELECT $$;", false},
	} {
		l := newLexer()
		if err := l.feed([]byte(tt.sql)); err != nil {
			t.Fatalf("%q: %v", tt.sql, err)
		}
		if l.idle() != tt.idle {
			t.Errorf("%q: idle = %t, want %t (state %d)", tt.sql, l.idle(), tt.idle, l.state)
		}
	}
}

func TestPlainFilterEnded(t *testing.T) {
	t.Parallel()
	trailer := "--\n-- PostgreSQL database dump complete\n--\n\n"
	for _, tt := range []struct {
		name, dump string
		ended      bool
	}{
		{"after the trailer", "SELECT 1;\n" + trailer, true},
		{"inside COPY data", "COPY t (v) FROM stdin;\n" + trailer, false},
		{"inside a string", "SELECT '\n" + trailer, false},
		{"inside a dollar quote", "SELECT $$\n" + trailer, false},
		{"inside a comment", "/*\n" + trailer, false},
		{"inside a statement", "SELECT 1\n" + trailer, false},
	} {
		_, f, err := filterPlain(t, tt.dump)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if f.Ended() != tt.ended {
			t.Errorf("%s: Ended = %t, want %t", tt.name, f.Ended(), tt.ended)
		}
	}
}

func TestPlainFilterRefusesMetaCommands(t *testing.T) {
	t.Parallel()
	for _, line := range []string{
		"\\connect other", "\\c other", "\\! touch /tmp/x", "SELECT 1 \\g", "\\set ON_ERROR_STOP 0", "\\.",
		"SELECT 1 \\; COMMIT;", "\\i other.sql", "  \\connect other", "\\restrict abc extra",
		"SELECT 1\n\\restrict abc", "\\unrestrict",
	} {
		dump := "SELECT 1;\n" + line + "\nDROP TABLE t;\n"
		got, _, err := filterPlain(t, dump)
		if !errors.Is(err, ErrMetaCommand) {
			t.Errorf("%q: err = %v", line, err)
		}
		if strings.Contains(got, "DROP") {
			t.Errorf("%q: the filter went on after it: %q", line, got)
		}
	}
	dump := "\\restrict abc\n\nSELECT 'a \\b', E'\\\\', $$\\x$$, \"\\\";\n-- \\connect\n/* \\connect */\n" +
		"COPY t FROM stdin;\n\\N\t\\\\\n\\.\n\\unrestrict abc\r\n"
	if got, f, err := filterPlain(t, dump); err != nil || got != dump || !f.Ended() {
		t.Errorf("got %q, ended %t, err %v", got, f.Ended(), err)
	}
}
