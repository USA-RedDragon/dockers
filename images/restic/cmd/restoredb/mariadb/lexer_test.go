package mariadb

import (
	"slices"
	"strings"
	"testing"
)

// statements splits sql the way clientLexer does and returns the text of each
// statement as the client sends it, with a space where a comment ended.
func statements(t *testing.T, sql string) ([]string, *clientLexer) {
	t.Helper()
	l := &clientLexer{}
	var got []string
	var cur strings.Builder
	for line := range strings.SplitAfterSeq(sql, "\n") {
		_, err := l.scan([]byte(line), true, func(p []byte, kind byteKind) error {
			switch kind {
			case byteText:
				cur.Write(p)
			case byteGap:
				if l.inStatement {
					cur.WriteByte(' ')
				}
			case byteEnd:
				got = append(got, cur.String())
				cur.Reset()
			}
			return nil
		})
		if err != nil {
			t.Fatalf("%q: %v", sql, err)
		}
	}
	return got, l
}

func TestClientLexerSplitsLikeTheClient(t *testing.T) {
	t.Parallel()
	const selectOne = "SELECT 1"
	for _, tt := range []struct {
		sql  string
		want []string
	}{
		{"SELECT ';' ; SELECT \"a;\";SELECT `x;y`;", []string{"SELECT ';' ", "SELECT \"a;\"", "SELECT `x;y`"}},
		{"SELECT 'it\\'s;', 'a''b;';", []string{"SELECT 'it\\'s;', 'a''b;'"}},
		{"SELECT `a\\`;\n", []string{"SELECT `a\\`"}},
		{"SELECT 1 -- c;\n;", []string{"SELECT 1  "}},
		{"SELECT 1--1;", []string{"SELECT 1--1"}},
		{"--x;\nSELECT 1;", []string{selectOne}},
		{"# c;\nSELECT 1 # d;\n;", []string{"SELECT 1  "}},
		{"/* ; */ SELECT 1;", []string{selectOne}},
		{"SELECT/* ; */1;", []string{selectOne}},
		{"/*!40101 SET a=1 */;", []string{"/*!40101 SET a=1 */"}},
		{"/*!40101 SET a=1 /* ; */ */;", []string{"/*!40101 SET a=1  "}},
		{"/*M!999999\\- enable the sandbox mode */ \nSELECT 1;",
			[]string{"/*M!999999 enable the sandbox mode */ \n" + selectOne}},
		{"SELECT 'a\nb;c';", []string{"SELECT 'a\nb;c'"}},
		{"SELECT \\N;", []string{"SELECT \\N"}},
		{"\\N--c\n;", []string{"\\N--c\n"}},
		{"/*!40000\n/* */ SELECT 2; /* */\nSELECT 1;", []string{"/*!40000\n  SELECT 2", "SELECT 1"}},
	} {
		got, l := statements(t, tt.sql)
		if !slices.Equal(got, tt.want) {
			t.Errorf("%q split into %q, want %q", tt.sql, got, tt.want)
		}
		if !l.idle() {
			t.Errorf("%q: not idle at the end", tt.sql)
		}
	}
	for _, open := range []string{"SELECT 'a", selectOne, "/* a", "SELECT `a"} {
		if _, l := statements(t, open); l.idle() {
			t.Errorf("%q: idle", open)
		}
	}
}
