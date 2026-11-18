package store

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// Errors the http package translates into specific responses. Anything else a
// store method returns is a bug or an outage and becomes a 500.
var (
	ErrNotFound   = errors.New("store: not found")
	ErrEmailTaken = errors.New("store: email already registered")
	// ErrSceneNumberTaken: the project already has a scene with that number.
	ErrSceneNumberTaken = errors.New("store: scene number already in project")
)

// Postgres error codes we act on. Full list:
// https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	pgUniqueViolation  = "23505"
	pgInvalidTextValue = "22P02" // e.g. "not-a-uuid" compared with a uuid column
)

// isUniqueViolation reports whether err is a unique-constraint failure on the
// named index or constraint. Matching on the name, not just the code, keeps one
// constraint's failure from being reported as another's.
func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == pgUniqueViolation &&
		pgErr.ConstraintName == constraint
}

// isInvalidText reports a value Postgres could not parse for its column type.
// For lookups by id that just means "no such row".
func isInvalidText(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgInvalidTextValue
}
