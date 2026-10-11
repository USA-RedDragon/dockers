package dumpdb_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/USA-RedDragon/dockers/images/restic/internal/dumpdb"
	"github.com/spf13/cobra"
)

func testCmd(out *bytes.Buffer) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetOut(out)
	cmd.SetErr(out)
	return cmd
}

func TestFailedCommandKeepsItsOwnOutputAndStatus(t *testing.T) {
	var out bytes.Buffer
	cmd := testCmd(&out)

	err := dumpdb.RunCmd(cmd, "sh", []string{"-c", "echo the real reason >&2; exit 42"}, nil)

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected an exit status, got %v", err)
	}
	if got := exitErr.ExitCode(); got != 42 {
		t.Errorf("exit code = %d, want 42", got)
	}
	if !strings.Contains(out.String(), "the real reason") {
		t.Errorf("the command's own message was lost:\n%s", out.String())
	}
	if !cmd.SilenceUsage {
		t.Error("usage would be printed over the command's own message")
	}
}

// Failing to start the command is our error, not the command's.
func TestUnstartableCommandIsAnOrdinaryError(t *testing.T) {
	var out bytes.Buffer

	err := dumpdb.RunCmd(testCmd(&out), "definitely-not-a-real-binary", nil, nil)
	if err == nil {
		t.Fatal("expected an error when the command cannot be started")
	}
	if _, ok := errors.AsType[*exec.ExitError](err); ok {
		t.Errorf("a missing binary should not look like an exit status: %v", err)
	}
}

func TestDryRunDoesNotExecute(t *testing.T) {
	var out bytes.Buffer
	// `false` would exit 1 if it ran.
	if err := dumpdb.RunCmd(testCmd(&out), "false", nil, &dumpdb.RunOpts{DryRun: true}); err != nil {
		t.Errorf("dry run should not execute: %v", err)
	}
}

func TestStdinOptionReplacesCommandStdin(t *testing.T) {
	tests := []struct {
		name  string
		stdin io.Reader
		want  string
	}{
		{name: "override", stdin: strings.NewReader("from opts"), want: "from opts"},
		{name: "default", stdin: nil, want: "from cobra"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())
			cmd.SetIn(strings.NewReader("from cobra"))
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			if err := dumpdb.RunCmd(cmd, "cat", nil, &dumpdb.RunOpts{Stdin: tt.stdin}); err != nil {
				t.Fatalf("run: %v", err)
			}
			if stdout.String() != tt.want {
				t.Errorf("stdout = %q, want %q", stdout.String(), tt.want)
			}
		})
	}
}

// RunCmd silences usage, so a bad flag -- rejected before it runs -- still gets it.
func TestBadFlagStillPrintsUsage(t *testing.T) {
	var out bytes.Buffer
	root := &cobra.Command{Use: "root"}
	sub := &cobra.Command{
		Use: "sub",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return dumpdb.RunCmd(cmd, "true", nil, nil)
		},
	}
	root.AddCommand(sub)
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"sub", "--bogus"})

	if err := root.Execute(); err == nil {
		t.Fatal("expected the flag to be rejected")
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Errorf("a bad flag should still print usage:\n%s", out.String())
	}
}

// A Filter that fails stops the command before it sees the end of its input,
// even though Complete would accept what reached it.
func TestFailingFilterStopsTheCommand(t *testing.T) {
	status := t.TempDir() + "/status"
	var out bytes.Buffer
	refused := errors.New("refused")
	err := dumpdb.RunCmd(testCmd(&out), "sh", []string{"-c", "cat >/dev/null; echo done >" + status}, &dumpdb.RunOpts{
		Stdin:    strings.NewReader("first\nsecond\n"),
		Complete: func([]byte) bool { return true },
		Filter: func(dst io.Writer, _ io.Reader) error {
			_, _ = io.WriteString(dst, "first\n")
			return refused
		},
	})
	if !errors.Is(err, refused) || !errors.Is(err, dumpdb.ErrStopped) {
		t.Errorf("err = %v", err)
	}
	if _, statErr := os.Stat(status); statErr == nil {
		t.Error("the command reached the end of its input")
	}
}
