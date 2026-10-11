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
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restoredb",
		Short: "Restore a database dump read from stdin",
		Long: `Restore a database dump read from stdin.

Each subcommand is the inverse of the matching dumpdb subcommand and resolves
connection details the same way, so a dump can be piped straight back in:

  restic dump latest "$RESTIC_HOST.dmp" | restoredb cnpg`,
	}

	cmd.AddCommand(
		mariadb.New(),
		mongodb.New(),
		postgres.New(),
		sqlite.New(),
	)

	return cmd
}

func main() {
	if err := New().Execute(); err != nil {
		slog.Error(err.Error())

		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}
