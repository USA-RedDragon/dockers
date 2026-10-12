package main

import (
	"errors"
	"log/slog"
	"os"
	"os/exec"

	"github.com/USA-RedDragon/dockers/images/restic/cmd/restoredb/mariadb"
	"github.com/USA-RedDragon/dockers/images/restic/cmd/restoredb/mongodb"
	"github.com/USA-RedDragon/dockers/images/restic/cmd/restoredb/postgres"
	"github.com/USA-RedDragon/dockers/images/restic/cmd/restoredb/sqlite"
	"github.com/USA-RedDragon/dockers/images/restic/internal/untouched"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restoredb",
		Short: "Restore a database dump read from stdin",
		Long: `Restore a database dump read from stdin.

Each subcommand is the inverse of the matching dumpdb subcommand and resolves
connection details the same way, so a dump can be piped straight back in:

  restic dump latest "$RESTIC_HOST.dmp" | restoredb cnpg

restoredb exits 75 when one of its checks stops it before the restore starts,
so nothing was written to the target: bad flags, an empty or unreadable dump,
a database it cannot connect to or query, a custom-format Postgres dump that
needs an extension the user cannot create, a MariaDB target it refuses, an
existing SQLite file without --force, or a restore tool that cannot start. Any
other failure exits with another status, and the target may have been changed.`,
	}
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return untouched.Wrap(err) })

	cmd.AddCommand(
		mariadb.New(),
		mongodb.New(),
		postgres.New(),
		sqlite.New(),
	)

	return cmd
}

// exitCode is the status restoredb exits with after err.
func exitCode(err error) int {
	if untouched.Is(err) {
		return untouched.ExitCode
	}
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		return exitErr.ExitCode()
	}
	return 1
}

func main() {
	if err := New().Execute(); err != nil {
		slog.Error(err.Error())
		if untouched.Is(err) {
			slog.Info("Nothing was written to the target")
		}
		os.Exit(exitCode(err))
	}
}
