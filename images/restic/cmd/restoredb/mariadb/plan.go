package mariadb

import (
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	// StagingSuffix names the schema a dump is restored into before the swap.
	StagingSuffix = "__b2bk_restore"
	// OldSuffix names the schema the replaced tables are moved into.
	OldSuffix = "__b2bk_old"
	// SchemaComment marks the staging and rollback schemas restoredb creates.
	// It drops only schemas that carry it.
	SchemaComment = "created by b2bk restoredb for a restore"

	maxIdentifier = 64
)

var (
	// ErrEmptyDump is returned when stdin holds no dump at all.
	ErrEmptyDump = errors.New("no dump on stdin")
	// ErrIncomplete is returned when the dump does not end with the dump tool's trailer.
	ErrIncomplete = errors.New(`the dump does not end with mariadb-dump's "-- Dump completed" trailer, ` +
		"so it was cut short")
	// ErrMultipleDatabases is returned for a dump that holds more than one database.
	ErrMultipleDatabases = errors.New("the dump holds more than one database; restoredb restores exactly one")
	// ErrUnsupportedObjects is returned when a schema or the dump holds objects the
	// swap cannot move atomically.
	ErrUnsupportedObjects = errors.New("views, triggers, stored routines and events cannot be swapped atomically")
	// ErrUnsupportedStatement is returned for a statement the restore cannot
	// keep out of the live database, or cannot check.
	ErrUnsupportedStatement = errors.New("the dump has a statement restoredb cannot restore safely")
	// ErrServerStatement is returned for a statement that changes the server,
	// replication or accounts rather than the dump's own tables.
	ErrServerStatement = errors.New("the dump changes the server, replication or accounts, not only its own tables")
	// ErrOtherDatabase is returned for a statement that names its table with a
	// database, which would write past the staging database.
	ErrOtherDatabase = errors.New("the dump writes to a table it names with a database, outside the staging database")
	// ErrOldExists is returned when the rollback schema of an earlier restore is still there.
	ErrOldExists = errors.New("the rollback database of an earlier restore still holds tables")
	// ErrForeignSchema is returned when a database by the staging or rollback
	// name exists that restoredb did not create.
	ErrForeignSchema = errors.New("a database by the name restoredb stages or keeps a rollback in exists, " +
		"and restoredb did not create it")
	// ErrNoDatabase is returned when the target database does not exist or is not visible.
	ErrNoDatabase = errors.New("the database does not exist, or the user cannot see it")
	// ErrPrivileges is returned when the user cannot create and drop the staging and rollback schemas.
	ErrPrivileges = errors.New("the user cannot create, fill and drop the staging and rollback databases")
	// ErrNameTooLong is returned when a derived schema name exceeds MariaDB's identifier limit.
	ErrNameTooLong = errors.New("the database name is too long for the staging and rollback names")
)

// Names returns the staging and rollback schema names for database.
func Names(database string) (staging, old string, err error) {
	if database == "" {
		return "", "", fmt.Errorf("%w: no database given", ErrNoDatabase)
	}
	staging, old = database+StagingSuffix, database+OldSuffix
	if utf8.RuneCountInString(staging) > maxIdentifier {
		return "", "", fmt.Errorf("%w: %s is longer than %d characters", ErrNameTooLong, staging, maxIdentifier)
	}
	return staging, old, nil
}

// QuoteIdent quotes an identifier with backticks.
func QuoteIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func qualified(schema, table string) string {
	return QuoteIdent(schema) + "." + QuoteIdent(table)
}

// literal renders s as a hex string literal. It compares byte for byte with
// information_schema's case-insensitive columns and means the same under any
// sql_mode.
func literal(s string) string {
	return "X'" + hex.EncodeToString([]byte(s)) + "'"
}

// RenameStatement returns one RENAME TABLE statement that moves every table in
// current out of live into out, and then every table in incoming from in into
// live. MariaDB applies a multi-table RENAME TABLE atomically: if any part
// fails, none of it happens. It also applies the parts in order, and a foreign
// key name is unique per database, so every table leaves live before any
// arrives: a constraint a schema change moved to another table would otherwise
// be in live twice partway through. It returns "" when there is nothing to move.
func RenameStatement(live, in, out string, current, incoming []string) string {
	current, incoming = slices.Sorted(slices.Values(current)), slices.Sorted(slices.Values(incoming))
	moves := make([]string, 0, len(current)+len(incoming))
	for _, name := range current {
		moves = append(moves, qualified(live, name)+" TO "+qualified(out, name))
	}
	for _, name := range incoming {
		moves = append(moves, qualified(in, name)+" TO "+qualified(live, name))
	}
	if len(moves) == 0 {
		return ""
	}
	return "RENAME TABLE " + strings.Join(moves, ", ")
}

// PrepareStatement returns statements that have the server parse stmt as a
// prepared statement, without running it, over the same client the swap uses.
// A dry run checks the swap with them.
func PrepareStatement(stmt string) string {
	return swapTimeout + "; PREPARE b2bk_swap FROM CONVERT(" + literal(stmt) + " USING utf8mb4); " +
		"DEALLOCATE PREPARE b2bk_swap"
}

const trailer = "-- Dump completed"

// DumpComplete reports whether tail, the end of a dump, ends with the trailer
// mariadb-dump and mysqldump write only after a successful dump: "-- Dump
// completed", followed by " on <date>" unless --skip-dump-date was given. A dump
// cut short does not end with it, since every data row is a line of its own
// that starts with INSERT.
func DumpComplete(tail []byte) bool {
	lines := strings.Split(string(tail), "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return false
	}
	last := strings.TrimRight(lines[len(lines)-1], "\r")
	return last == trailer || strings.HasPrefix(last, trailer+" on ")
}
