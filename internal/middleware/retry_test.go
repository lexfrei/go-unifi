package middleware_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lexfrei/go-unifi/internal/middleware"
	"github.com/lexfrei/go-unifi/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetry(t *testing.T) {
	t.Parallel()

	t.Run("success on first attempt", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		transport := middleware.Retry(middleware.RetryConfig{
			MaxRetries:  3,
			InitialWait: time.Millisecond,
		})(http.DefaultTransport)

		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
		resp, err := transport.RoundTrip(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("retry on 500 error", func(t *testing.T) {
		t.Parallel()

		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			attempts++
			if attempts < 3 {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()

		transport := middleware.Retry(middleware.RetryConfig{
			MaxRetries:  3,
			InitialWait: time.Millisecond,
		})(http.DefaultTransport)

		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
		resp, err := transport.RoundTrip(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 3, attempts)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("no retry on 404 error", func(t *testing.T) {
		t.Parallel()

		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			attempts++
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		transport := middleware.Retry(middleware.RetryConfig{
			MaxRetries:  3,
			InitialWait: time.Millisecond,
		})(http.DefaultTransport)

		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
		resp, err := transport.RoundTrip(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 1, attempts, "no retry on 4xx")
	})

	t.Run("retry with body", func(t *testing.T) {
		t.Parallel()

		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++

			// Read body
			body, _ := io.ReadAll(r.Body)
			assert.Equal(t, "test body", string(body))

			if attempts < 2 {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()

		transport := middleware.Retry(middleware.RetryConfig{
			MaxRetries:  3,
			InitialWait: time.Millisecond,
		})(http.DefaultTransport)

		req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, strings.NewReader("test body"))
		resp, err := transport.RoundTrip(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 2, attempts)
	})

	t.Run("respect Retry-After header", func(t *testing.T) {
		t.Parallel()

		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			attempts++
			if attempts == 1 {
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()

		transport := middleware.Retry(middleware.RetryConfig{
			MaxRetries:  3,
			InitialWait: time.Hour, // Would normally wait 1 hour on first retry
		})(http.DefaultTransport)

		start := time.Now()
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
		resp, err := transport.RoundTrip(req)
		duration := time.Since(start)

		require.NoError(t, err)
		defer resp.Body.Close()

		// Should respect Retry-After (1 second) instead of initialWait (1 hour)
		assert.Less(t, duration, 2*time.Second, "should use Retry-After instead of initialWait")
	})

	t.Run("context cancellation during retry", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		transport := middleware.Retry(middleware.RetryConfig{
			MaxRetries:  10,
			InitialWait: time.Second,
		})(http.DefaultTransport)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, http.NoBody)
		resp, err := transport.RoundTrip(req)
		if resp != nil {
			resp.Body.Close()
		}

		require.Error(t, err, "expected error on context cancellation")

		assert.Contains(t, err.Error(), "context", "error should be context-related")
	})
}

func TestRetryWithObservability(t *testing.T) {
	t.Parallel()

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	logger := observability.NoopLogger()
	metrics := observability.NoopMetricsRecorder()

	transport := middleware.Retry(middleware.RetryConfig{
		MaxRetries:  3,
		InitialWait: time.Millisecond,
		Logger:      logger,
		Metrics:     metrics,
	})(http.DefaultTransport)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 3, attempts)
}
