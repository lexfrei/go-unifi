package middleware

import (
	"net/http"
	"regexp"
	"time"

	"github.com/lexfrei/go-unifi/observability"
)

// Observability returns a middleware that logs and records metrics for HTTP requests.
func Observability(logger observability.Logger, metrics observability.MetricsRecorder) func(http.RoundTripper) http.RoundTripper {
	if logger == nil {
		logger = observability.NoopLogger()
	}
	if metrics == nil {
		metrics = observability.NoopMetricsRecorder()
	}

	return func(next http.RoundTripper) http.RoundTripper {
		return &observabilityTransport{
			next:    next,
			logger:  logger,
			metrics: metrics,
		}
	}
}

type observabilityTransport struct {
	next    http.RoundTripper
	logger  observability.Logger
	metrics observability.MetricsRecorder
}

func (t *observabilityTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	// Log request
	t.logger.Debug("http request started",
		observability.Field{Key: "method", Value: req.Method},
		observability.Field{Key: "url", Value: req.URL.String()},
		observability.Field{Key: "path", Value: req.URL.Path},
	)

	// Make request
	resp, err := t.next.RoundTrip(req)

	duration := time.Since(start)

	if err != nil {
		// Log error
		t.logger.Error("http request failed",
			observability.Field{Key: "method", Value: req.Method},
			observability.Field{Key: "url", Value: req.URL.String()},
			observability.Field{Key: "duration", Value: duration},
			observability.Field{Key: "error", Value: err.Error()},
		)

		t.metrics.RecordError("http_request", "NetworkError")

		//nolint:wrapcheck // Observability middleware logs error but passes it through unchanged
		return nil, err
	}

	// Log response
	fields := []observability.Field{
		{Key: "method", Value: req.Method},
		{Key: "url", Value: req.URL.String()},
		{Key: "status", Value: resp.StatusCode},
		{Key: "duration", Value: duration},
	}

	if resp.StatusCode >= http.StatusBadRequest {
		t.logger.Warn("http request completed with error", fields...)
	} else {
		t.logger.Debug("http request completed", fields...)
	}

	// Record metrics with normalized path to avoid unbounded cardinality
	normalizedPath := normalizePath(req.URL.Path)
	t.metrics.RecordHTTPRequest(req.Method, normalizedPath, resp.StatusCode, duration)

	return resp, nil
}

var (
	// uuidPattern matches UUID format (8-4-4-4-12).
	uuidPattern = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	// objectIDPattern matches MongoDB ObjectID (24 hex characters).
	objectIDPattern = regexp.MustCompile(`[0-9a-f]{24}`)
	// numericIDPattern matches numeric IDs (5+ digits to avoid replacing version numbers).
	numericIDPattern = regexp.MustCompile(`/\d{5,}(/|$)`)
)

// normalizePath replaces dynamic path segments (UUIDs, ObjectIDs, numeric IDs) with placeholders
// to prevent unbounded cardinality in Prometheus metrics.
//
// Examples:
//   - /api/site/default/dns/record/507f1f77bcf86cd799439011 → /api/site/:site/dns/record/:id
//   - /api/site/my-site/device/12345678 → /api/site/:site/device/:id
//   - /proxy/network/v2/api/site/default/setting → /proxy/network/v2/api/site/:site/setting
func normalizePath(path string) string {
	// Replace UUIDs with :id
	normalized := uuidPattern.ReplaceAllString(path, ":id")

	// Replace ObjectIDs with :id
	normalized = objectIDPattern.ReplaceAllString(normalized, ":id")

	// Replace numeric IDs with :id
	normalized = numericIDPattern.ReplaceAllString(normalized, "/:id$1")

	// Replace site names: /site/{name}/ → /site/:site/
	// This pattern looks for /site/ followed by non-slash characters followed by / or end of string
	normalized = regexp.MustCompile(`/site/[^/]+(/|$)`).ReplaceAllString(normalized, "/site/:site$1")

	return normalized
}
