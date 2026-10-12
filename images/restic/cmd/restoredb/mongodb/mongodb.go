package mongodb

import (
	"github.com/USA-RedDragon/dockers/images/restic/internal/cobrax"
	"github.com/USA-RedDragon/dockers/images/restic/internal/dumpdb"
	"github.com/USA-RedDragon/dockers/images/restic/internal/untouched"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	var mount, host, port, database, username, password, authDB string
	var dryRun bool

	fs := &cobrax.Flags{}
	cmd := &cobra.Command{
		Use:   "mongodb",
		Short: "Restore a MongoDB database",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := fs.Resolve(); err != nil {
				return untouched.Wrap(err)
			}

			// --archive with no value reads the dump from stdin, mirroring
			// `dumpdb mongodb`. --drop clears each collection before it is restored.
			args = append([]string{
				"--archive",
				"--drop",
				"--authenticationDatabase=" + authDB,
				"--host=" + host,
				"--port=" + port,
				"--username=" + username,
				"--password=" + password,
				"--db=" + database,
			}, args...)

			return dumpdb.RunCmd(cmd, "mongorestore", args, &dumpdb.RunOpts{
				Redact: []string{password},
				DryRun: dryRun,
			})
		},
	}

	fs.FlagSet = cmd.Flags()
	fs.String(&mount, "secret-mount", "", "/mongodb", "Directory where secrets are mounted")
	fs.String(&host, dumpdb.FlagHost, "H", "", "Database host",
		cobrax.Env("DB_HOST"))
	fs.String(&port, dumpdb.FlagPort, "P", "27017", "Database port",
		cobrax.Env("DB_PORT"))
	fs.String(&database, dumpdb.FlagDatabase, "d", "", "Database name",
		cobrax.Env("DB_DATABASE"))
	fs.String(&username, dumpdb.FlagUsername, "u", "", "Database user",
		cobrax.Env("DB_USERNAME"))
	fs.String(&password, dumpdb.FlagPassword, "p", "", "Database password",
		cobrax.Env("DB_PASSWORD"), cobrax.SecretFile(&mount, "mongodb-passwords"))
	fs.String(&authDB, "authentication-db", "", "", "Authentication database",
		cobrax.Env("AUTHENTICATION_DB"))
	fs.Bool(&dryRun, dumpdb.FlagDryRun, "", false, "Dry run",
		cobrax.Env("DB_DRY_RUN"))

	return cmd
}
