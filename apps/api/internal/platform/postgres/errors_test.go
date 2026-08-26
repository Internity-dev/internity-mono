package postgres

import (
	"errors"
	"testing"

	"internity/internal/httpx"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestTranslateError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode httpx.ErrorCode
		wantNil  bool
	}{
		{name: "nil error", err: nil, wantNil: true},
		{name: "non-postgres error", err: errors.New("boom"), wantNil: true},
		{name: "unique violation", err: &pgconn.PgError{Code: codeUniqueViolation}, wantCode: httpx.ErrConflict},
		{name: "foreign key violation", err: &pgconn.PgError{Code: codeForeignKeyViolation}, wantCode: httpx.ErrConflict},
		// ON DELETE/UPDATE RESTRICT reports a distinct SQLSTATE from a plain
		// FK violation — confirmed live against appliances.vacancy_id
		// (ON DELETE RESTRICT), not assumed from the Postgres docs alone.
		{name: "restrict violation", err: &pgconn.PgError{Code: codeRestrictViolation}, wantCode: httpx.ErrConflict},
		{name: "check violation", err: &pgconn.PgError{Code: codeCheckViolation}, wantCode: httpx.ErrValidation},
		{name: "exclusion violation", err: &pgconn.PgError{Code: codeExclusionViolation}, wantCode: httpx.ErrConflict},
		{name: "unmapped postgres code", err: &pgconn.PgError{Code: "40001"}, wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TranslateError(tt.err)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("TranslateError(%v) = %v, want nil", tt.err, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("TranslateError(%v) = nil, want code %v", tt.err, tt.wantCode)
			}
			if got.Code != tt.wantCode {
				t.Fatalf("TranslateError(%v).Code = %v, want %v", tt.err, got.Code, tt.wantCode)
			}
			if got.Message == "" {
				t.Fatalf("TranslateError(%v).Message is empty", tt.err)
			}
		})
	}
}
