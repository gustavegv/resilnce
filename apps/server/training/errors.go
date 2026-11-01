package training

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Database details stay behind the application operations, including constraint
// names. The adapter only translates application error categories to HTTP.
func storageError(err error) error {
	var constraint *pgconn.PgError
	if errors.As(err, &constraint) && constraint.Code == "23505" && (constraint.ConstraintName == "exercise_owner_name" || constraint.ConstraintName == "variant_exercise_name") {
		return conflict("name-conflict", "An exercise or variant already has this name. Select the existing entry or choose another name")
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return &Error{404, "not-found", "Resource is unavailable to this account", nil}
	}
	return err
}
