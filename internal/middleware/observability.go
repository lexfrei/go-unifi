package middleware

import (
	"net/http"
	"time"

	"github.com/lexfrei/go-unifi/internal/observability"
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

	// Record metrics
	t.metrics.RecordHTTPRequest(req.Method, req.URL.Path, resp.StatusCode, duration)

	return resp, nil
}
