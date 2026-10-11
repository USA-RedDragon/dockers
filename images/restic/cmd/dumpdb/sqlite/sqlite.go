package sqlite

import (
	"os"

	"github.com/gabe565/docker-restic/internal/cobrax"
	"github.com/gabe565/docker-restic/internal/dumpdb"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	var dryRun bool

	fs := &cobrax.Flags{}
	cmd := &cobra.Command{
		Use:   "sqlite file",
		Short: "Dump a SQLite database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := fs.Resolve(); err != nil {
				return err
			}

			path := args[0]
			args = args[1:]

			if _, err := os.Stat(path); err != nil {
				return err
			}

			args = append([]string{
				"-bail",
				path,
				".dump",
			}, args...)

			return dumpdb.RunCmd(cmd, "sqlite3", args, &dumpdb.RunOpts{
				DryRun: dryRun,
			})
		},
	}

	fs.FlagSet = cmd.Flags()
	fs.Bool(&dryRun, dumpdb.FlagDryRun, "", false, "Dry run",
		cobrax.Env("DB_DRY_RUN"))

	return cmd
}
