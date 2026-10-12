package sqlite

import (
	"errors"
	"fmt"
	"os"

	"github.com/USA-RedDragon/dockers/images/restic/internal/cobrax"
	"github.com/USA-RedDragon/dockers/images/restic/internal/dumpdb"
	"github.com/USA-RedDragon/dockers/images/restic/internal/untouched"
	"github.com/spf13/cobra"
)

var ErrFileExists = errors.New("database file already exists")

func New() *cobra.Command {
	var dryRun, force bool

	fs := &cobrax.Flags{}
	cmd := &cobra.Command{
		Use:   "sqlite file",
		Short: "Restore a SQLite database",
		Args: func(cmd *cobra.Command, args []string) error {
			return untouched.Wrap(cobra.ExactArgs(1)(cmd, args))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := fs.Resolve(); err != nil {
				return untouched.Wrap(err)
			}

			path := args[0]
			args = args[1:]

			// `dumpdb sqlite` emits CREATE statements, which fail against an
			// existing schema, so restoring over a populated file needs --force.
			if stat, err := os.Stat(path); err == nil && stat.Size() != 0 {
				if !force {
					return untouched.Wrap(fmt.Errorf("%w: %s (use --force to replace it)", ErrFileExists, path))
				}
				if !dryRun {
					if err := os.Remove(path); err != nil {
						return err
					}
				}
			} else if err != nil && !errors.Is(err, os.ErrNotExist) {
				return untouched.Wrap(err)
			}

			args = append([]string{
				"-bail",
				path,
			}, args...)

			return dumpdb.RunCmd(cmd, "sqlite3", args, &dumpdb.RunOpts{
				DryRun: dryRun,
			})
		},
	}

	fs.FlagSet = cmd.Flags()
	fs.Bool(&force, "force", "f", false, "Replace the database file if it already exists")
	fs.Bool(&dryRun, dumpdb.FlagDryRun, "", false, "Dry run",
		cobrax.Env("DB_DRY_RUN"))

	return cmd
}
