package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// pgUniqueViolation is the Postgres SQLSTATE for a unique-constraint violation.
const pgUniqueViolation = "23505"

// pgForeignKeyViolation is the Postgres SQLSTATE for a foreign-key violation.
const pgForeignKeyViolation = "23503"

// IsUniqueViolation reports whether err is a Postgres unique-violation
// (SQLSTATE 23505) raised against the named constraint or index.
//
// internal/database is configured without gorm.Config.TranslateError (see
// db.go), so GORM never produces gorm.ErrDuplicatedKey — the driver's raw
// *pgconn.PgError is the only signal available. Matching err.Error() text
// would be brittle and can't distinguish which constraint fired; matching the
// SQLSTATE + constraint name is the portable way to classify a duplicate-key
// error without string-sniffing. Enabling TranslateError globally was
// considered and rejected: it would change error behavior for every
// repository in the codebase, not just this one.
func IsUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == pgUniqueViolation && pgErr.ConstraintName == constraint
}

// IsForeignKeyViolation reports whether err is a Postgres foreign-key
// violation (SQLSTATE 23503) raised against the named constraint. Mirrors
// IsUniqueViolation for the same reason: with TranslateError disabled (see
// IsUniqueViolation's doc comment), the driver's raw *pgconn.PgError is the
// only signal available, and matching SQLSTATE + constraint name is the
// portable way to classify it without string-sniffing.
func IsForeignKeyViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == pgForeignKeyViolation && pgErr.ConstraintName == constraint
}
