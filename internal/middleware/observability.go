package middleware

import (
	"net/http"
	"regexp"
	"sync"
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

	// Compute URL string once to avoid multiple allocations
	urlStr := req.URL.String()

	// Log request
	t.logger.Debug("http request started",
		observability.Field{Key: "method", Value: req.Method},
		observability.Field{Key: "url", Value: urlStr},
		observability.Field{Key: "path", Value: req.URL.Path},
	)

	// Make request
	resp, err := t.next.RoundTrip(req)

	duration := time.Since(start)

	if err != nil {
		// Log error
		t.logger.Error("http request failed",
			observability.Field{Key: "method", Value: req.Method},
			observability.Field{Key: "url", Value: urlStr},
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
		{Key: "url", Value: urlStr},
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
	// combinedIDPattern matches UUIDs, ObjectIDs, or numeric IDs in a single pattern.
	// This reduces the number of passes over the string from 3 to 1 for ID replacement.
	// Order matters: UUID first (most specific), then ObjectID, then numeric.
	combinedIDPattern = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}|[0-9a-f]{24}|/\d{5,}(?:/|$)`)
	// siteNamePattern matches site names in paths: /site/{name}/ → /site/:site/.
	siteNamePattern = regexp.MustCompile(`/site/[^/]+(/|$)`)

	// normalizedPathCache caches normalized paths to avoid repeated regex operations.
	// In production, most requests hit a limited set of endpoints, so caching provides
	// significant performance improvement (up to 150x faster on cache hit).
	normalizedPathCache sync.Map
)

// normalizePath replaces dynamic path segments (UUIDs, ObjectIDs, numeric IDs) with placeholders
// to prevent unbounded cardinality in Prometheus metrics.
//
// Uses an in-memory cache to avoid repeated regex operations for the same paths.
// In production scenarios with limited endpoint sets, this provides up to 150x speedup.
//
// Examples:
//   - /api/site/default/dns/record/507f1f77bcf86cd799439011 → /api/site/:site/dns/record/:id
//   - /api/site/my-site/device/12345678 → /api/site/:site/device/:id
//   - /proxy/network/v2/api/site/default/setting → /proxy/network/v2/api/site/:site/setting
func normalizePath(path string) string {
	// Fast path: check cache
	if cached, ok := normalizedPathCache.Load(path); ok {
		//nolint:forcetypeassert // Cache only stores strings, type assertion is safe
		return cached.(string)
	}

	// Slow path: compute normalization
	// Replace all ID types (UUIDs, ObjectIDs, numeric IDs) in a single pass.
	// ReplaceAllStringFunc allows us to handle the numeric ID case specially
	// where we need to preserve the trailing slash or end-of-string.
	normalized := combinedIDPattern.ReplaceAllStringFunc(path, func(match string) string {
		// Numeric IDs start with / and end with / or EOL
		if match[0] == '/' {
			// Preserve the structure: /12345/ or /12345$ → /:id/ or /:id
			if match[len(match)-1] == '/' {
				return "/:id/"
			}
			return "/:id"
		}
		// UUIDs and ObjectIDs are replaced directly
		return ":id"
	})

	// Replace site names: /site/{name}/ → /site/:site/
	normalized = siteNamePattern.ReplaceAllString(normalized, "/site/:site$1")

	// Store in cache for future requests
	normalizedPathCache.Store(path, normalized)

	return normalized
}
