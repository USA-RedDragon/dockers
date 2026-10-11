package main

import (
	"log/slog"
	"os"

	"github.com/USA-RedDragon/dockers/images/restic/cmd/dumpdb/mariadb"
	"github.com/USA-RedDragon/dockers/images/restic/cmd/dumpdb/mongodb"
	"github.com/USA-RedDragon/dockers/images/restic/cmd/dumpdb/postgres"
	"github.com/USA-RedDragon/dockers/images/restic/cmd/dumpdb/sqlite"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dumpdb",
		Short: "Database utilities",
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
		os.Exit(1)
	}
}
