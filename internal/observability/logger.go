// Package observability provides interfaces for logging and metrics collection.
// These interfaces allow users to plug in their own logging and metrics implementations.
package observability

// Field represents a structured logging field (key-value pair).
type Field struct {
	Key   string
	Value any
}

// Logger is an interface for structured logging.
// Implementations can use any logging library (slog, zap, logrus, etc.).
type Logger interface {
	// Debug logs a debug-level message with optional structured fields.
	Debug(msg string, fields ...Field)

	// Info logs an info-level message with optional structured fields.
	Info(msg string, fields ...Field)

	// Warn logs a warning-level message with optional structured fields.
	Warn(msg string, fields ...Field)

	// Error logs an error-level message with optional structured fields.
	Error(msg string, fields ...Field)

	// With returns a new logger with the given fields pre-populated.
	// Subsequent log calls on the returned logger will include these fields.
	With(fields ...Field) Logger
}

// noopLogger is a no-operation logger that discards all log messages.
type noopLogger struct{}

// NoopLogger returns a logger that does nothing.
// This is the default logger used when none is provided.
//
//nolint:ireturn // Factory function must return interface for dependency injection pattern
func NoopLogger() Logger {
	return &noopLogger{}
}

func (l *noopLogger) Debug(string, ...Field) {}
func (l *noopLogger) Info(string, ...Field)  {}
func (l *noopLogger) Warn(string, ...Field)  {}
func (l *noopLogger) Error(string, ...Field) {}

//nolint:ireturn // Method must return interface to satisfy Logger interface
func (l *noopLogger) With(...Field) Logger { return l }
