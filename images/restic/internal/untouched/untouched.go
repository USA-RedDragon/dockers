// Package untouched marks the restoredb failures that happen before anything
// is written to the target, so a caller can tell them from a failed restore.
package untouched

import "errors"

// ExitCode is the status restoredb exits with when it failed before it wrote
// anything to the target. It is 75, EX_TEMPFAIL in sysexits.h. No program
// restoredb runs or passes the status of uses it: psql exits 1 to 3, the
// mariadb client 1 or 2, pg_restore, mongorestore and sqlite3 1, and restic 1,
// 3, 10 to 12 or 130. A shell uses 126 and up.
const ExitCode = 75

// Error is a failure that happened before anything was written to the target.
type Error struct{ Err error }

func (e *Error) Error() string { return e.Err.Error() }

func (e *Error) Unwrap() error { return e.Err }

// Wrap marks err, when it is not nil, as a failure before any write.
func Wrap(err error) error {
	if err == nil {
		return nil
	}
	return &Error{Err: err}
}

// Is reports whether err is a failure before any write.
func Is(err error) bool {
	_, ok := errors.AsType[*Error](err)
	return ok
}
