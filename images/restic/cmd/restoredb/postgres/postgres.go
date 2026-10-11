package postgres

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/USA-RedDragon/dockers/images/restic/internal/cobrax"
	"github.com/USA-RedDragon/dockers/images/restic/internal/dumpdb"
	"github.com/spf13/cobra"
)

// ErrEmptyDump is returned when stdin holds no dump at all.
var ErrEmptyDump = errors.New("no dump on stdin")

var customMagic = []byte("PGDMP")

// PlainDumpComplete reports whether a plain SQL dump ends the way pg_dump ends
// one: the "dump complete" comment block, then at most blank lines and the
// \unrestrict line pg_dump 17.6 and later add. A dump cut short by a failed
// download does not, even if the same text appears earlier in its data. key is
// the dump's \restrict key, when its head has one; the \unrestrict line must
// then carry the same key, which no data row can.
func PlainDumpComplete(key string, tail []byte) bool {
	lines := strings.Split(string(tail), "\n")
	last := func() string {
		for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
			lines = lines[:len(lines)-1]
		}
		if len(lines) == 0 {
			return ""
		}
		line := strings.TrimRight(lines[len(lines)-1], "\r")
		lines = lines[:len(lines)-1]
		return line
	}
	line := last()
	switch {
	case key != "":
		if line != `\unrestrict `+key {
			return false
		}
		line = last()
	case strings.HasPrefix(line, `\unrestrict `):
		line = last()
	}
	return line == "--" && last() == "-- PostgreSQL database dump complete"
}

// RestrictKey returns the key of the \restrict line at the head of a plain dump,
// or "" when it has none. pg_dump writes that line after its opening comment
// block, before any statement, so the search stops at the first line that is
// not a comment or blank: a COPY row can read as a \restrict line too.
func RestrictKey(head []byte) string {
	for line := range strings.SplitSeq(string(head), "\n") {
		line = strings.TrimRight(line, "\r")
		if key, ok := strings.CutPrefix(line, `\restrict `); ok {
			return strings.TrimSpace(key)
		}
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "--") {
			return ""
		}
	}
	return ""
}

// IsCustomFormat reports whether head starts with the pg_dump custom-format
// magic bytes.
func IsCustomFormat(head []byte) bool {
	return bytes.HasPrefix(head, customMagic)
}

func New() *cobra.Command {
	var mount, host, port, database, username, password string
	var dryRun bool

	fs := &cobrax.Flags{}
	cmd := &cobra.Command{
		Use:     "postgres",
		Aliases: []string{"cnpg"},
		Short:   "Restore a Postgres database",
		Long: `Restore a Postgres database from a dump read from stdin.

A pg_dump custom-format dump, as written by dumpdb postgres, is restored with
pg_restore. Anything else is taken as a plain SQL dump from pg_dump and run with
psql. Both run in a single transaction, so a failed restore leaves the database
untouched. pg_restore reads the whole archive before it commits; psql commits
when its input ends, so a plain dump that ends without pg_dump's "dump complete"
trailer, outside any COPY data and statement, stops psql first, and nothing is
committed.

A plain dump with large objects has its own BEGIN; and COMMIT; lines around
them. The COMMIT would end the restore's transaction partway, so those lines
reach psql as comments, and any other statement that ends, rolls back or
prepares a transaction is refused before psql runs it. So is any psql
meta-command but \restrict and \unrestrict, since one such as \connect would
carry on outside the transaction.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := fs.Resolve(); err != nil {
				return err
			}

			in := bufio.NewReader(cmd.InOrStdin())
			head, err := in.Peek(len(customMagic))
			if err != nil && !errors.Is(err, io.EOF) {
				cmd.SilenceUsage = true
				return fmt.Errorf("read dump: %w", err)
			}
			if len(head) == 0 {
				cmd.SilenceUsage = true
				return ErrEmptyDump
			}

			name := "psql"
			opts := []string{"-v", "ON_ERROR_STOP=1", "--single-transaction", "--no-psqlrc", "--quiet"}
			key := ""
			if more, _ := in.Peek(in.Size()); len(more) > 0 {
				key = RestrictKey(more)
			}
			plain := NewPlainFilter()
			complete := func(tail []byte) bool { return plain.Ended() && PlainDumpComplete(key, tail) }
			filter := plain.Copy
			if IsCustomFormat(head) {
				name = "pg_restore"
				opts = []string{"--clean", "--if-exists", "--no-owner", "--single-transaction"}
				complete, filter = nil, nil
			}
			conn := []string{
				"--host=" + host,
				"--port=" + port,
				"--username=" + username,
				"--dbname=" + database,
			}

			return dumpdb.RunCmd(cmd, name, slices.Concat(opts, conn, args), &dumpdb.RunOpts{
				Envs:     []string{"PGPASSWORD=" + password},
				DryRun:   dryRun,
				Stdin:    in,
				Complete: complete,
				Filter:   filter,
			})
		},
	}

	fs.FlagSet = cmd.Flags()
	fs.String(&mount, "secret-mount", "", "/postgresql-app", "Directory where secrets are mounted")
	fs.String(&host, dumpdb.FlagHost, "H", "", "Database host",
		cobrax.Env("DB_HOST"), cobrax.SecretFile(&mount, "host"))
	fs.String(&port, dumpdb.FlagPort, "P", "5432", "Database port",
		cobrax.Env("DB_PORT"), cobrax.SecretFile(&mount, "port"))
	fs.String(&database, dumpdb.FlagDatabase, "d", "", "Database name",
		cobrax.Env("DB_DATABASE"), cobrax.SecretFile(&mount, "dbname"))
	fs.String(&username, dumpdb.FlagUsername, "u", "", "Database user",
		cobrax.Env("DB_USERNAME"), cobrax.SecretFile(&mount, "username"))
	fs.String(&password, dumpdb.FlagPassword, "p", "", "Database password",
		cobrax.Env("DB_PASSWORD"), cobrax.SecretFile(&mount, "password"))
	fs.Bool(&dryRun, dumpdb.FlagDryRun, "", false,
		"Print the command and read the dump to the end without restoring it, checking a plain dump as a restore would",
		cobrax.Env("DB_DRY_RUN"))

	return cmd
}
