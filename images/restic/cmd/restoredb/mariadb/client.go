package mariadb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
	"syscall"

	"github.com/USA-RedDragon/dockers/images/restic/internal/xtrace"
)

// conn runs SQL against the server.
type conn interface {
	// query runs one statement and returns its rows.
	query(ctx context.Context, sql string) ([][]string, error)
	// execute runs statements separated by semicolons, stopping at the first
	// that fails.
	execute(ctx context.Context, sql string) error
	// load feeds a dump to the client with database as its default database,
	// in binary mode, so that client commands such as \u and connect, which
	// could switch databases, are sent to the server as text instead of run.
	// write is called with the client's stdin and must return once it has
	// written everything it will. If it fails, the client is killed before it
	// sees the end of its input, which it would take as the end of the last
	// statement and run.
	load(ctx context.Context, database string, write func(io.Writer) error) error
}

// client is a conn that runs the mariadb command-line client, the same one the
// restore has always replayed dumps with.
type client struct {
	host, port, user, password string
	extra                      []string
	stdout, stderr             io.Writer
}

func (c *client) args(more ...string) []string {
	return slices.Concat([]string{
		"--host=" + c.host,
		"--port=" + c.port,
		"--user=" + c.user,
		// The charset decides how the server reads the bytes of the dump, and
		// the restore's lexer follows only charsets in which no character ends
		// in a backslash. utf8mb4 is one, whatever the server's own default is.
		"--default-character-set=utf8mb4",
	}, more, c.extra)
}

func (c *client) command(ctx context.Context, args []string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "mariadb", args...)
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+c.password)
	return cmd
}

// run sends sql to the client on stdin, never on its command line, which Linux
// limits to 128 KiB per argument: a swap of many tables is longer. Binary mode
// keeps the client from reading anything in sql as a client command.
func (c *client) run(ctx context.Context, sql string, trace bool) (string, error) {
	cmd := c.command(ctx, c.args("--batch", "--skip-column-names", "--binary-mode"))
	if trace {
		_, _ = fmt.Fprintf(c.stderr, "%s <<'SQL'\n%s\nSQL\n",
			strings.TrimSuffix(xtrace.XTraceString(cmd.Args), "\n"), sql)
	}
	cmd.Stdin = strings.NewReader(sql + ";\n")
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		if msg := clientError(errOut.String()); msg != "" {
			return "", fmt.Errorf("%s: %w", msg, err)
		}
		return "", err
	}
	return out.String(), nil
}

func (c *client) query(ctx context.Context, sql string) ([][]string, error) {
	out, err := c.run(ctx, sql, false)
	if err != nil {
		return nil, err
	}
	return parseBatch(out), nil
}

func (c *client) execute(ctx context.Context, sql string) error {
	_, err := c.run(ctx, sql, true)
	return err
}

func (c *client) load(ctx context.Context, database string, write func(io.Writer) error) error {
	cmd := c.command(ctx, c.args("--binary-mode", "--database="+database))
	xtrace.XTrace(c.stderr, cmd.Args)
	cmd.Stdout, cmd.Stderr = c.stdout, c.stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	writeErr := write(stdin)
	if writeErr != nil && !errors.Is(writeErr, syscall.EPIPE) {
		_ = cmd.Process.Kill()
		_ = stdin.Close()
		_ = cmd.Wait()
		return writeErr
	}
	_ = stdin.Close()
	waitErr := cmd.Wait()
	if waitErr != nil && (writeErr == nil || errors.Is(writeErr, syscall.EPIPE)) {
		return waitErr
	}
	return errors.Join(writeErr, waitErr)
}

// clientError picks the client's ERROR lines out of its stderr, leaving out
// warnings, or returns all of it when it has none.
func clientError(stderr string) string {
	var errs []string
	for line := range strings.SplitSeq(stderr, "\n") {
		if strings.HasPrefix(line, "ERROR") {
			errs = append(errs, line)
		}
	}
	if len(errs) == 0 {
		return strings.TrimSpace(stderr)
	}
	return strings.Join(errs, "; ")
}

// parseBatch splits the client's --batch output into rows of fields, undoing
// the escapes it writes for backslash, tab, newline and NUL.
func parseBatch(out string) [][]string {
	var rows [][]string
	for line := range strings.SplitSeq(strings.TrimSuffix(out, "\n"), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		for i, f := range fields {
			fields[i] = unescapeBatch(f)
		}
		rows = append(rows, fields)
	}
	return rows
}

var batchEscapes = strings.NewReplacer(`\\`, `\`, `\t`, "\t", `\n`, "\n", `\0`, "\x00")

func unescapeBatch(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	return batchEscapes.Replace(s)
}
