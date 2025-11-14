package observability_test

import (
	"testing"

	"github.com/lexfrei/go-unifi/observability"
)

func TestNoopLogger(t *testing.T) {
	t.Parallel()

	logger := observability.NoopLogger()

	// All methods should execute without panicking
	logger.Debug("test debug")
	logger.Info("test info")
	logger.Warn("test warn")
	logger.Error("test error")

	// With should return a logger
	newLogger := logger.With(observability.Field{Key: "key", Value: "value"})
	if newLogger == nil {
		t.Error("With() returned nil")
	}

	// With'd logger should also work
	newLogger.Info("test with logger")
}

func TestField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		field observability.Field
		key   string
		value any
	}{
		{
			name:  "string value",
			field: observability.Field{Key: "name", Value: "test"},
			key:   "name",
			value: "test",
		},
		{
			name:  "int value",
			field: observability.Field{Key: "count", Value: 42},
			key:   "count",
			value: 42,
		},
		{
			name:  "nil value",
			field: observability.Field{Key: "null", Value: nil},
			key:   "null",
			value: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.field.Key != tt.key {
				t.Errorf("Field.Key = %v, want %v", tt.field.Key, tt.key)
			}

			if tt.field.Value != tt.value {
				t.Errorf("Field.Value = %v, want %v", tt.field.Value, tt.value)
			}
		})
	}
}

// BenchmarkNoopLogger measures the overhead of noop logger calls.
func BenchmarkNoopLogger(b *testing.B) {
	logger := observability.NoopLogger()

	b.Run("Info", func(b *testing.B) {
		for range b.N {
			logger.Info("test message")
		}
	})

	b.Run("InfoWithFields", func(b *testing.B) {
		fields := []observability.Field{
			{Key: "key1", Value: "value1"},
			{Key: "key2", Value: 42},
		}

		for range b.N {
			logger.Info("test message", fields...)
		}
	})

	b.Run("With", func(b *testing.B) {
		for range b.N {
			logger.With(observability.Field{Key: "key", Value: "value"})
		}
	})
}
