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
	"github.com/lexfrei/go-unifi/internal/observability"
)

func TestRetry(t *testing.T) {
	t.Parallel()

	t.Run("success on first attempt", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		transport := middleware.Retry(middleware.RetryConfig{
			MaxRetries:  3,
			InitialWait: time.Millisecond,
		})(http.DefaultTransport)

		req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
		resp, err := transport.RoundTrip(req)
		if err != nil {
			t.Fatalf("RoundTrip() error = %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
		}
	})

	t.Run("retry on 500 error", func(t *testing.T) {
		t.Parallel()

		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
		resp, err := transport.RoundTrip(req)
		if err != nil {
			t.Fatalf("RoundTrip() error = %v", err)
		}
		defer resp.Body.Close()

		if attempts != 3 {
			t.Errorf("attempts = %d, want %d", attempts, 3)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
		}
	})

	t.Run("no retry on 404 error", func(t *testing.T) {
		t.Parallel()

		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		transport := middleware.Retry(middleware.RetryConfig{
			MaxRetries:  3,
			InitialWait: time.Millisecond,
		})(http.DefaultTransport)

		req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
		resp, err := transport.RoundTrip(req)
		if err != nil {
			t.Fatalf("RoundTrip() error = %v", err)
		}
		defer resp.Body.Close()

		if attempts != 1 {
			t.Errorf("attempts = %d, want %d (no retry on 4xx)", attempts, 1)
		}
	})

	t.Run("retry with body", func(t *testing.T) {
		t.Parallel()

		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++

			// Read body
			body, _ := io.ReadAll(r.Body)
			if string(body) != "test body" {
				t.Errorf("body = %s, want %s", string(body), "test body")
			}

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

		req, _ := http.NewRequest(http.MethodPost, server.URL, strings.NewReader("test body"))
		resp, err := transport.RoundTrip(req)
		if err != nil {
			t.Fatalf("RoundTrip() error = %v", err)
		}
		defer resp.Body.Close()

		if attempts != 2 {
			t.Errorf("attempts = %d, want %d", attempts, 2)
		}
	})

	t.Run("respect Retry-After header", func(t *testing.T) {
		t.Parallel()

		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
		resp, err := transport.RoundTrip(req)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("RoundTrip() error = %v", err)
		}
		defer resp.Body.Close()

		// Should respect Retry-After (1 second) instead of initialWait (1 hour)
		if duration > 2*time.Second {
			t.Errorf("duration = %v, want < 2s (should use Retry-After)", duration)
		}
	})

	t.Run("context cancellation during retry", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		transport := middleware.Retry(middleware.RetryConfig{
			MaxRetries:  10,
			InitialWait: time.Second,
		})(http.DefaultTransport)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
		_, err := transport.RoundTrip(req)

		if err == nil {
			t.Error("expected error on context cancellation")
		}

		if !strings.Contains(err.Error(), "context") {
			t.Errorf("error = %v, want context-related error", err)
		}
	})
}

func TestRetryWithObservability(t *testing.T) {
	t.Parallel()

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	defer resp.Body.Close()

	if attempts != 3 {
		t.Errorf("attempts = %d, want %d", attempts, 3)
	}
}
