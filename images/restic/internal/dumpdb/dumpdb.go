package dumpdb

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/USA-RedDragon/dockers/images/restic/internal/xtrace"
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
	// DryRun only traces the command. When Stdin is set it is still read to
	// the end, through Filter and Complete, so whatever writes it is not cut
	// off and a dry run reports what a real run would refuse.
	DryRun bool
	Stdin  io.Reader

	// Complete, when set, is shown the last tailSize bytes of what the command
	// was given once stdin is read to the end. Unless it returns true the
	// command is killed before it sees the end of its input, so a tool that
	// commits at end of input, such as psql --single-transaction, never commits
	// a truncated stream. That holds only if nothing in the input commits
	// earlier; Filter is where such input is changed or refused.
	Complete func(tail []byte) bool

	// Filter, when set with Complete, copies stdin to the command in place of
	// io.Copy. If it fails, the command is killed before it sees the end of its
	// input.
	Filter func(dst io.Writer, src io.Reader) error

	// Extra are passed to the command as file descriptors 3 and up.
	Extra []*os.File
}

var (
	// ErrIncomplete is returned when Complete rejected the input.
	ErrIncomplete = errors.New("the input ended before it was complete")
	// ErrStopped is returned when stdin could not be read or Filter failed.
	ErrStopped = errors.New("the command was stopped before it could commit")
)

const tailSize = 64 << 10

func RunCmd(cmd *cobra.Command, name string, args []string, opts *RunOpts) error {
	cmd.SilenceUsage = true

	if opts == nil {
		opts = &RunOpts{}
	}

	e := exec.CommandContext(cmd.Context(), name, args...)
	e.Env = append(os.Environ(), opts.Envs...)
	e.Stdin = cmd.InOrStdin()
	if opts.Stdin != nil {
		e.Stdin = opts.Stdin
	}
	e.Stdout = cmd.OutOrStdout()
	e.Stderr = cmd.ErrOrStderr()
	e.ExtraFiles = opts.Extra

	trace := xtrace.XTraceString(e.Args)
	for _, v := range opts.Redact {
		trace = strings.ReplaceAll(trace, v, "***")
	}
	_, _ = io.WriteString(e.Stderr, trace)

	filter := opts.Filter
	if filter == nil {
		filter = func(dst io.Writer, src io.Reader) error {
			_, err := io.Copy(dst, src)
			return err
		}
	}
	if opts.DryRun {
		if opts.Stdin == nil {
			return nil
		}
		return drain(opts.Stdin, filter, opts.Complete)
	}
	if opts.Complete != nil {
		return runGuarded(e, opts.Complete, filter)
	}

	return e.Run()
}

// drain reads a dry run's input to the end, through filter and complete as a
// real run would, so a pipeline feeding it is not cut off and a dump that a
// real run would stop is reported.
func drain(in io.Reader, filter func(io.Writer, io.Reader) error, complete func([]byte) bool) error {
	tail := &tailWriter{}
	if err := filter(tail, in); err != nil {
		_, _ = io.Copy(io.Discard, in)
		return err
	}
	if complete != nil && !complete(tail.buf) {
		return ErrIncomplete
	}
	return nil
}

func runGuarded(e *exec.Cmd, complete func(tail []byte) bool, filter func(io.Writer, io.Reader) error) error {
	in := e.Stdin
	e.Stdin = nil
	pipe, err := e.StdinPipe()
	if err != nil {
		return err
	}
	if err := e.Start(); err != nil {
		return err
	}

	tail := &tailWriter{}
	copyErr := filter(io.MultiWriter(pipe, tail), in)
	if copyErr != nil || !complete(tail.buf) {
		_ = e.Process.Kill()
		state, _ := e.Process.Wait()
		_ = pipe.Close()
		_ = e.Wait()
		if state != nil && state.Exited() && !state.Success() {
			return &exec.ExitError{ProcessState: state}
		}
		if copyErr != nil {
			return fmt.Errorf("%w: %w", ErrStopped, copyErr)
		}
		return fmt.Errorf("%w, so the command was stopped before it could commit", ErrIncomplete)
	}
	if err := pipe.Close(); err != nil {
		_ = e.Process.Kill()
		_ = e.Wait()
		return err
	}
	return e.Wait()
}

type tailWriter struct{ buf []byte }

func (t *tailWriter) Write(p []byte) (int, error) {
	t.buf = append(t.buf, p...)
	if over := len(t.buf) - tailSize; over > 0 {
		t.buf = append(t.buf[:0], t.buf[over:]...)
	}
	return len(p), nil
}
