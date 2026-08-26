// Package httpx defines the API's response envelope and error taxonomy so
// every handler returns the same success/error shape (spec requirement:
// consistent { success, data, message } responses with no leaked internals).
package httpx

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type ErrorCode string

const (
	// ErrBadRequest: the request itself is malformed (unparseable JSON body,
	// wrong-typed path/query param) — the server can't even make sense of what
	// was sent. Distinct from ErrValidation, where the request parses fine but
	// a value fails a business rule (required/format/range/etc).
	ErrBadRequest      ErrorCode = "BAD_REQUEST"
	ErrValidation      ErrorCode = "VALIDATION_ERROR"
	ErrUnauthenticated ErrorCode = "UNAUTHENTICATED"
	ErrForbidden       ErrorCode = "FORBIDDEN"
	ErrNotFound        ErrorCode = "NOT_FOUND"
	ErrConflict        ErrorCode = "CONFLICT"
	ErrRateLimited     ErrorCode = "RATE_LIMITED"
	ErrInternal        ErrorCode = "INTERNAL_ERROR"
)

var statusByCode = map[ErrorCode]int{
	ErrBadRequest:      400,
	ErrValidation:      422,
	ErrUnauthenticated: 401,
	ErrForbidden:       403,
	ErrNotFound:        404,
	ErrConflict:        409,
	ErrRateLimited:     429,
	ErrInternal:        500,
}

type ErrorDetail struct {
	Field string `json:"field,omitempty"`
	Issue string `json:"issue"`
}

// APIError is the typed error every service/handler returns; it never carries
// a raw driver/DB error message meant for the client (that's logged separately).
type APIError struct {
	Code    ErrorCode     `json:"code"`
	Details []ErrorDetail `json:"details,omitempty"`
	Message string        `json:"-"`
}

func (e *APIError) Error() string { return e.Message }

func NewError(code ErrorCode, message string, details ...ErrorDetail) *APIError {
	return &APIError{Code: code, Message: message, Details: details}
}

type meta struct {
	RequestID  string      `json:"request_id,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

type Pagination struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

// OK writes a 200/201/204-style success envelope. Pass httpStatus explicitly
// since the same helper covers 200 (read), 201 (create), and 204-with-body-suppressed callers.
func OK(c *gin.Context, httpStatus int, data any, message string, pagination *Pagination) {
	c.JSON(httpStatus, gin.H{
		"success": true,
		"data":    data,
		"message": message,
		"meta":    meta{RequestID: requestID(c), Pagination: pagination},
	})
}

// Fail writes the error envelope and aborts the chain. It never receives a
// raw error string from a DB/driver — callers must translate first.
func Fail(c *gin.Context, err *APIError) {
	status, ok := statusByCode[err.Code]
	if !ok {
		status = 500
	}
	c.AbortWithStatusJSON(status, gin.H{
		"success": false,
		"data":    nil,
		"message": err.Message,
		"error": gin.H{
			"code":    err.Code,
			"details": err.Details,
		},
		"meta": meta{RequestID: requestID(c)},
	})
}

// FailFromErr translates any error into the envelope: a known *APIError
// passes through as-is (its message is safe to show); anything else — a raw
// GORM/driver/network error — is logged with full detail server-side and
// answered with a generic message only, so internals never reach the client.
func FailFromErr(c *gin.Context, err error) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		Fail(c, apiErr)
		return
	}
	log.Error().Err(err).Str("request_id", requestID(c)).Str("path", c.Request.URL.Path).Msg("unhandled error")
	Fail(c, NewError(ErrInternal, "Something went wrong. Please try again."))
}

func requestID(c *gin.Context) string {
	if v, ok := c.Get("request_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
