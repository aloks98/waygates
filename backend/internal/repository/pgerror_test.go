package repository

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestIsUniqueViolation_MatchesCodeAndConstraint(t *testing.T) {
	err := &pgconn.PgError{Code: "23505", ConstraintName: "uq_proxy_groups_name"}
	assert.True(t, IsUniqueViolation(err, "uq_proxy_groups_name"))
}

func TestIsUniqueViolation_FindsErrorThroughWrapping(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23505", ConstraintName: "proxies_hostname_key"}
	wrapped := fmt.Errorf("re-homing proxy 4 to %q: %w", "abc.example.com", pgErr)

	assert.True(t, IsUniqueViolation(wrapped, "proxies_hostname_key"))
}

func TestIsUniqueViolation_DifferentConstraintDoesNotMatch(t *testing.T) {
	err := &pgconn.PgError{Code: "23505", ConstraintName: "proxies_hostname_key"}
	assert.False(t, IsUniqueViolation(err, "uq_proxy_groups_name"),
		"a violation on a different constraint must not be misclassified")
}

func TestIsUniqueViolation_DifferentSQLSTATEDoesNotMatch(t *testing.T) {
	// 23503 is foreign_key_violation, not unique_violation.
	err := &pgconn.PgError{Code: "23503", ConstraintName: "uq_proxy_groups_name"}
	assert.False(t, IsUniqueViolation(err, "uq_proxy_groups_name"))
}

func TestIsUniqueViolation_NonPgErrorDoesNotMatch(t *testing.T) {
	assert.False(t, IsUniqueViolation(errors.New("connection reset"), "uq_proxy_groups_name"))
}
