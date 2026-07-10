package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// pgUniqueViolation is the Postgres SQLSTATE for a unique-constraint violation.
const pgUniqueViolation = "23505"

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
