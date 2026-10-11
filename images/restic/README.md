# Restic

[restic](https://restic.net) with database clients and dump tools, forked from
[gabe565/docker-restic](https://github.com/gabe565/docker-restic). It is the default backup image of
[b2-backup-operator](https://github.com/USA-RedDragon/b2-backup-operator). The image tag is the restic version.

It is gabe565's image with `restoredb` added. The base image, packages, binary paths and environment are the same.

## Binaries

| Path | What it is |
| --- | --- |
| `/usr/local/bin/restic` | Wrapper that adds flags, then runs `/usr/bin/restic` |
| `/usr/bin/restic` | restic itself |
| `/usr/bin/dumpdb` | Dumps a CNPG, MariaDB, MongoDB or SQLite database to stdout |
| `/usr/bin/restoredb` | Restores a dump read from stdin. MariaDB restores are staged and swapped in atomically |
| `/usr/bin/runitor` | [runitor](https://github.com/bdd/runitor) |
| `/usr/bin/ts` | `ts` from moreutils |
| `/usr/local/bin/backup-*.sh` | Dump scripts kept from upstream |

The image also has `bash`, `pg_dump`/`psql`, the MariaDB client, `mongodump`/`mongorestore` and `sqlite3`.

## Wrapper behavior

The wrapper only changes `backup`, `snapshots` and `forget`. It passes other commands through unchanged.

- `RESTIC_GROUP_BY` (default `tags`): the wrapper adds `--group-by=$RESTIC_GROUP_BY` when `--group-by` is not given.
- `RESTIC_HOST`: with `--stdin-from-command` and no `--stdin-filename`, the wrapper adds
  `--stdin-filename=$RESTIC_HOST` plus `.dmp` for `dumpdb cnpg` and `dumpdb mongodb`, or `.sql` for other `dumpdb`
  commands. For `dumpdb sqlite <path>`, the name is `<path>` without a `.db`, `.sqlite` or `.sqlite3` suffix, plus `.sql`.
- `forget` gets `--compact` when stdout is not a terminal.
- `RESTIC_WRAPPER_DRY_RUN=true` prints the final command line and does not run restic.

## Tests

```sh
make test           # go vet and go test
make test-mariadb   # restoredb mariadb against a throwaway MariaDB container
make test-postgres  # restoredb postgres against a throwaway Postgres container
```
