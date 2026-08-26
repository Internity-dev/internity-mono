package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/go-playground/validator/v10"
)

// BindingError classifies a c.ShouldBindJSON error: malformed JSON (syntax
// error, wrong-typed field, empty body) is a 400 — the request couldn't even
// be parsed; a struct-tag validation failure (required/min/email/etc, request
// parsed fine) is a 422 — see the ErrBadRequest vs ErrValidation doc comment.
func BindingError(err error) *APIError {
	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		details := make([]ErrorDetail, 0, len(validationErrs))
		for _, fe := range validationErrs {
			details = append(details, ErrorDetail{
				Field: toSnakeCase(fe.Field()),
				Issue: fe.Tag(),
			})
		}
		return NewError(ErrValidation, "Validation failed", details...)
	}

	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &syntaxErr) || errors.As(err, &typeErr) || errors.Is(err, io.ErrUnexpectedEOF) {
		return NewError(ErrBadRequest, "Request body is not valid JSON")
	}

	// Anything else (missing body, io errors) is still the caller's fault, not ours.
	return NewError(ErrBadRequest, "Invalid request", ErrorDetail{Issue: err.Error()})
}

// BadPathParam is for a path/query param that fails to parse as the expected
// type (e.g. /vacancies/abc when :id must be numeric) — malformed request, 400.
func BadPathParam(field, issue string) *APIError {
	return NewError(ErrBadRequest, "Invalid request", ErrorDetail{Field: field, Issue: issue})
}

func toSnakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r + ('a' - 'A'))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
