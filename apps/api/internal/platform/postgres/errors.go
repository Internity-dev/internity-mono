package postgres

import (
	"errors"

	"internity/internal/httpx"

	"github.com/jackc/pgx/v5/pgconn"
)

// Postgres error codes: https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	codeUniqueViolation     = "23505"
	codeForeignKeyViolation = "23503"
	codeCheckViolation      = "23514"
	// codeRestrictViolation — an `ON DELETE/UPDATE RESTRICT` FK violation.
	// Postgres reports this separately from a plain FK violation (23503) —
	// confirmed live against this schema (appliances.vacancy_id ON DELETE
	// RESTRICT), not assumed from the docs alone.
	codeRestrictViolation = "23001"
	// codeExclusionViolation — a GiST/GIN EXCLUDE constraint (e.g.
	// intern_dates' no-overlapping-placements rule).
	codeExclusionViolation = "23P01"
)

// TranslateError maps a raw driver error to the API's error taxonomy so a
// unique/FK/check-constraint violation becomes a clean 409 CONFLICT instead
// of a leaked "pq: duplicate key value violates constraint..." message.
// Returns nil if err isn't a Postgres constraint violation — the caller
// should fall through to httpx.FailFromErr (which logs + generic-500s it).
func TranslateError(err error) *httpx.APIError {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return nil
	}

	switch pgErr.Code {
	case codeUniqueViolation:
		return httpx.NewError(httpx.ErrConflict, "This record already exists.")
	case codeForeignKeyViolation, codeRestrictViolation:
		return httpx.NewError(httpx.ErrConflict, "This record is still referenced by other data and can't be removed.")
	case codeCheckViolation:
		return httpx.NewError(httpx.ErrValidation, "Invalid input.")
	case codeExclusionViolation:
		return httpx.NewError(httpx.ErrConflict, "This conflicts with another existing record.")
	default:
		return nil
	}
}
