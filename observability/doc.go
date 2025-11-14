// Package observability provides interfaces for logging and metrics collection
// in the go-unifi library.
//
// This package defines standard interfaces that allow users to integrate their
// own logging and metrics implementations with UniFi API clients.
//
// # Logger Interface
//
// The Logger interface supports structured logging with key-value pairs:
//
//	logger := myCustomLogger{} // implements observability.Logger
//	client, err := sitemanager.NewWithConfig(&sitemanager.ClientConfig{
//		APIKey:  apiKey,
//		Logger:  logger,
//	})
//
// Supported log levels:
//   - Debug: Detailed diagnostic information
//   - Info: General informational messages
//   - Warn: Warning messages for potentially problematic situations
//   - Error: Error messages for failures
//
// # MetricsRecorder Interface
//
// The MetricsRecorder interface tracks API client metrics:
//
//	metrics := myMetricsRecorder{} // implements observability.MetricsRecorder
//	client, err := sitemanager.NewWithConfig(&sitemanager.ClientConfig{
//		APIKey:  apiKey,
//		Metrics: metrics,
//	})
//
// Tracked metrics include:
//   - HTTP request count, status codes, and duration
//   - Retry attempts for failed requests
//   - Rate limiting events and wait times
//   - Error occurrences by type
//
// # Default Behavior
//
// If no logger or metrics recorder is provided, the client uses no-op
// implementations that discard all events. This ensures zero overhead
// when observability is not needed.
//
// # Example
//
// See examples/observability/main.go for a complete working example showing
// how to integrate custom logging (using slog) and metrics collection.
package observability
