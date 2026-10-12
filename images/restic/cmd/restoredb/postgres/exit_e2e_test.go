//go:build postgres

package postgres_test

import (
	"cmp"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/USA-RedDragon/dockers/images/restic/internal/untouched"
)

// restoredb exits 75 only when it stopped before psql or pg_restore was given
// any of the dump, and a failure inside the restore exits with another status.
// Either way app is left as it was.
func TestPostgresExitStatusTellsAFailureBeforeTheRestore(t *testing.T) {
	custom, _ := seed(t, "", "-Fc")
	plain, _ := seed(t, "", "--clean", "--if-exists")
	cutPlain, _, ok := strings.Cut(plain, "ALTER TABLE ONLY public.t")
	if !ok {
		t.Fatal("the dump adds no constraints to t")
	}
	tests := []struct {
		name  string
		user  string
		input string
		args  []string
		want  int
		says  string
	}{
		{name: "a role that cannot log in", user: "nobody", input: custom, want: untouched.ExitCode,
			says: `role "nobody" does not exist`},
		{name: "a server that is not there", input: custom, args: []string{"-P", "1"}, want: untouched.ExitCode,
			says: "query the database's extensions"},
		{name: "an empty dump", input: "", want: untouched.ExitCode, says: "no dump on stdin"},
		{name: "a custom dump cut short", input: custom[:len(custom)/2], want: 1},
		{name: "a plain dump cut short", input: cutPlain, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := fingerprint(t)
			out, err := restoreAs(t, cmp.Or(tt.user, "postgres"), pgPassword, []byte(tt.input), tt.args...)
			exitErr, ok := errors.AsType[*exec.ExitError](err)
			if !ok || exitErr.ExitCode() != tt.want {
				t.Errorf("err = %v, want exit %d", err, tt.want)
			}
			if !strings.Contains(out, tt.says) {
				t.Errorf("the output does not say %q", tt.says)
			}
			if after := fingerprint(t); after != before {
				t.Errorf("the database changed: %s", firstDiff(before, after))
			}
		})
	}
}
