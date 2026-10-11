// Package testutil holds helpers shared by the dumpdb and restoredb golden
// tests. It is only ever imported from _test.go files.
package testutil

import (
	"bytes"
	"io"
	"testing"

	"github.com/spf13/cobra"
)

// ClearEnv blanks every variable the flag source chains consult, so a golden
// test never picks up the developer's shell.
func ClearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"DB_HOST", "DB_PORT", "DB_DATABASE", "DB_USERNAME", "DB_PASSWORD",
		"DB_DRY_RUN", "AUTHENTICATION_DB",
	} {
		t.Setenv(k, "")
	}
}

// RunGolden executes cmd and returns the xtrace line RunCmd writes to stderr.
func RunGolden(t *testing.T, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()
	var stderr bytes.Buffer
	cmd.SetArgs(args)
	cmd.SetOut(io.Discard)
	cmd.SetErr(&stderr)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.Execute()
	return stderr.String(), err
}
