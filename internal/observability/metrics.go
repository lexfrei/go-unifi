package observability

import "time"

// MetricsRecorder is an interface for recording metrics.
// Implementations can use any metrics library (Prometheus, StatsD, etc.).
type MetricsRecorder interface {
	// RecordHTTPRequest records an HTTP request with method, path, status code, and duration.
	RecordHTTPRequest(method, path string, statusCode int, duration time.Duration)

	// RecordRetry records a retry attempt for an endpoint.
	RecordRetry(attempt int, endpoint string)

	// RecordRateLimit records a rate limit wait event.
	RecordRateLimit(endpoint string, wait time.Duration)

	// RecordError records an error occurrence.
	RecordError(operation, errorType string)
}

// noopMetricsRecorder is a no-operation metrics recorder that does nothing.
type noopMetricsRecorder struct{}

// NoopMetricsRecorder returns a metrics recorder that does nothing.
// This is the default recorder used when none is provided.
func NoopMetricsRecorder() MetricsRecorder {
	return &noopMetricsRecorder{}
}

func (m *noopMetricsRecorder) RecordHTTPRequest(string, string, int, time.Duration) {}
func (m *noopMetricsRecorder) RecordRetry(int, string)                              {}
func (m *noopMetricsRecorder) RecordRateLimit(string, time.Duration)                {}
func (m *noopMetricsRecorder) RecordError(string, string)                           {}
