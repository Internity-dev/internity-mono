package middleware

import (
	"internity/internal/httpx"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Recovery replaces gin's default recovery so a panic always becomes the
// standard error envelope (never a bare 500 with a stack trace in the body).
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error().
					Interface("panic", r).
					Str("request_id", requestIDFrom(c)).
					Str("trace_id", traceID(c)).
					Str("path", c.Request.URL.Path).
					Msg("panic recovered")
				httpx.Fail(c, httpx.NewError(httpx.ErrInternal, "Something went wrong. Please try again."))
			}
		}()
		c.Next()
	}
}

func requestIDFrom(c *gin.Context) string {
	if v, ok := c.Get("request_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func traceID(c *gin.Context) string {
	sc := oteltrace.SpanContextFromContext(c.Request.Context())
	if !sc.HasTraceID() {
		return ""
	}
	return sc.TraceID().String()
}
