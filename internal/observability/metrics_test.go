package observability_test

import (
	"testing"
	"time"

	"github.com/lexfrei/go-unifi/internal/observability"
)

func TestNoopMetricsRecorder(t *testing.T) {
	t.Parallel()

	recorder := observability.NoopMetricsRecorder()

	// All methods should execute without panicking
	recorder.RecordHTTPRequest("GET", "/test", 200, time.Second)
	recorder.RecordRetry(1, "/endpoint")
	recorder.RecordRateLimit("/endpoint", time.Millisecond*100)
	recorder.RecordError("operation", "NetworkError")
}

// BenchmarkNoopMetricsRecorder measures the overhead of noop metrics recorder calls.
func BenchmarkNoopMetricsRecorder(b *testing.B) {
	recorder := observability.NoopMetricsRecorder()

	b.Run("RecordHTTPRequest", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			recorder.RecordHTTPRequest("GET", "/test", 200, time.Second)
		}
	})

	b.Run("RecordRetry", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			recorder.RecordRetry(1, "/endpoint")
		}
	})

	b.Run("RecordRateLimit", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			recorder.RecordRateLimit("/endpoint", time.Millisecond*100)
		}
	})

	b.Run("RecordError", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			recorder.RecordError("operation", "NetworkError")
		}
	})
}
