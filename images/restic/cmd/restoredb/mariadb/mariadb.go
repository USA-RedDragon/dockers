package mariadb

import (
	"github.com/USA-RedDragon/dockers/images/restic/internal/cobrax"
	"github.com/USA-RedDragon/dockers/images/restic/internal/dumpdb"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	var mount, host, port, database, username, password string
	var dryRun, dropOld bool

	fs := &cobrax.Flags{}
	cmd := &cobra.Command{
		Use:   "mariadb",
		Short: "Restore a MariaDB database",
		Long: `Restore a MariaDB database from a mariadb-dump or mysqldump read from stdin.

The restore is all or nothing. The dump is restored into a staging database,
<database>` + StagingSuffix + `, which is dropped first if an earlier run left it
behind. restoredb creates the staging and rollback databases with the comment
"` + SchemaComment + `" and refuses to start while a database by either name
exists without it, so it never drops a database it did not create; this needs
MariaDB 10.5 or later. Only when the whole dump has gone in cleanly and ends with the dump
tool's "-- Dump completed" trailer does one RENAME TABLE statement, which
MariaDB applies atomically, move every live table into <database>` + OldSuffix + `
and every staged table into the live database. Any failure before that drops
the staging database and leaves the live database as it was. If the swap itself
reports an error, the tables are listed again to tell whether it went through.

The replaced tables stay in <database>` + OldSuffix + ` for a rollback until it is
dropped, by hand or with --drop-old. A restore refuses to start while that
database still holds tables.

The dump is read a statement at a time, split the way the mariadb client splits
it, and each statement is checked before the client sees it. USE and CREATE
DATABASE statements from a --databases dump are left out, so the dump always
lands in the database given; a dump of more than one database is refused. So is
any name qualified with a database, anywhere in a statement, such as INSERT
INTO ` + "`other`.`t`" + ` or a multi-table UPDATE, which could write past the staging
database or run code there (CREATE TABLE, where mariadb-dump writes such names
for a foreign key to another database, is the exception unless it has a
SELECT); a table whose engine or options reach other tables, such as MERGE's
UNION or FEDERATED's CONNECTION, or a SET of the default engine to one of
those; a client charset, from SET NAMES or character_set_client, in which the
server would read a quoted string differently, such as gbk; one that changes
the
server, replication or accounts, such as SET GLOBAL, CHANGE MASTER or GRANT, or
builds SQL at run time, such as PREPARE or EXECUTE, or writes a file on the
server; and a client command other than the \- a mariadb-dump starts with.
RENAME TABLE cannot move views or tables with triggers to another database, so
a database or dump that has views, triggers, or (in the dump) stored routines
or events is refused before the live database is touched.

The user needs ALL on <database>` + StagingSuffix + ` and <database>` + OldSuffix + `,
and ALTER, DROP, CREATE and INSERT on the database itself.

--dry-run restores into the staging database, checks it, has the server parse
the RENAME TABLE as a prepared statement without running it, and drops the
staging database again without swapping.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := fs.Resolve(); err != nil {
				return err
			}
			cmd.SilenceUsage = true

			db := &client{
				host:     host,
				port:     port,
				user:     username,
				password: password,
				extra:    args,
				stdout:   cmd.OutOrStdout(),
				stderr:   cmd.ErrOrStderr(),
			}
			r, err := newRestore(db, database, dryRun, dropOld, cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			return r.run(cmd.Context(), cmd.InOrStdin())
		},
	}

	fs.FlagSet = cmd.Flags()
	fs.String(&mount, "secret-mount", "", "/mariadb", "Directory where secrets are mounted")
	fs.String(&host, dumpdb.FlagHost, "H", "", "Database host",
		cobrax.Env("DB_HOST"))
	fs.String(&port, dumpdb.FlagPort, "P", "3306", "Database port",
		cobrax.Env("DB_PORT"))
	fs.String(&database, dumpdb.FlagDatabase, "d", "", "Database name",
		cobrax.Env("DB_DATABASE"))
	fs.String(&username, dumpdb.FlagUsername, "u", "", "Database user",
		cobrax.Env("DB_USERNAME"))
	fs.String(&password, dumpdb.FlagPassword, "p", "", "Database password",
		cobrax.Env("DB_PASSWORD"), cobrax.SecretFile(&mount, "mariadb-password"))
	fs.Bool(&dryRun, dumpdb.FlagDryRun, "", false,
		"Restore into the staging database and drop it again, without swapping",
		cobrax.Env("DB_DRY_RUN"))
	fs.Bool(&dropOld, "drop-old", "", false,
		"Drop the replaced tables after a successful swap instead of keeping them for a rollback")

	return cmd
}
