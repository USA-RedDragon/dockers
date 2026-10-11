package postgres

import (
	"github.com/gabe565/docker-restic/internal/cobrax"
	"github.com/gabe565/docker-restic/internal/dumpdb"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	var mount, host, port, database, username, password string
	var dryRun bool

	fs := &cobrax.Flags{}
	cmd := &cobra.Command{
		Use:     "postgres",
		Aliases: []string{"cnpg"},
		Short:   "Dump a Postgres database",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := fs.Resolve(); err != nil {
				return err
			}

			args = append([]string{
				"--format=custom",
				"--compress=0",
				"--clean",
				"--if-exists",
				"--no-owner",
				"--host=" + host,
				"--port=" + port,
				"--username=" + username,
				"--dbname=" + database,
			}, args...)

			return dumpdb.RunCmd(cmd, "pg_dump", args, &dumpdb.RunOpts{
				Envs:   []string{"PGPASSWORD=" + password},
				DryRun: dryRun,
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
	fs.Bool(&dryRun, dumpdb.FlagDryRun, "", false, "Dry run",
		cobrax.Env("DB_DRY_RUN"))

	return cmd
}
