package dumpdb

import (
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/gabe565/docker-restic/internal/xtrace"
	"github.com/spf13/cobra"
)

const (
	FlagHost     = "host"
	FlagPort     = "port"
	FlagDatabase = "database"
	FlagUsername = "username"
	FlagPassword = "password"
	FlagDryRun   = "dry-run"
)

type RunOpts struct {
	Envs   []string
	Redact []string
	DryRun bool
}

func RunCmd(cmd *cobra.Command, name string, args []string, opts *RunOpts) error {
	if opts == nil {
		opts = &RunOpts{}
	}

	e := exec.CommandContext(cmd.Context(), name, args...)
	e.Env = append(os.Environ(), opts.Envs...)
	e.Stdin = cmd.InOrStdin()
	e.Stdout = cmd.OutOrStdout()
	e.Stderr = cmd.ErrOrStderr()

	xtrace := xtrace.XTraceString(e.Args)
	for _, v := range opts.Redact {
		xtrace = strings.ReplaceAll(xtrace, v, "***")
	}
	_, _ = io.WriteString(e.Stderr, xtrace)

	if opts.DryRun {
		return nil
	}

	return e.Run()
}
