package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests, labeled by method, route, and status code.",
	}, []string{"method", "path", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds, labeled by method and route.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})
)

// Metrics must be registered before Recovery in the middleware chain: a
// panic unwinds straight to Recovery's deferred recover, skipping any
// post-c.Next() code in a middleware nested inside it. Registered outside
// Recovery instead, Metrics' own post-c.Next() code still runs normally
// after Recovery has already handled the panic and written the response.
//
// Labeled by c.FullPath() (the route template, e.g. "/api/v1/users/:id")
// rather than the raw request path, so a path parameter never blows up
// cardinality.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		status := strconv.Itoa(c.Writer.Status())
		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(time.Since(start).Seconds())
	}
}
