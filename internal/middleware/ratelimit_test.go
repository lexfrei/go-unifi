package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lexfrei/go-unifi/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestRateLimit(t *testing.T) {
	t.Parallel()

	t.Run("single limiter", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Allow 2 requests per second
		limiter := rate.NewLimiter(2, 2)

		transport := middleware.RateLimit(middleware.RateLimitConfig{
			Limiter: limiter,
		})(http.DefaultTransport)

		// First 2 requests should be fast
		for i := range 2 {
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
			start := time.Now()
			resp, err := transport.RoundTrip(req)
			duration := time.Since(start)

			require.NoError(t, err)
			resp.Body.Close()

			assert.Less(t, duration, 100*time.Millisecond, "request %d should complete quickly", i+1)
		}

		// Third request should be rate limited
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
		start := time.Now()
		resp, err := transport.RoundTrip(req)
		duration := time.Since(start)

		require.NoError(t, err)
		resp.Body.Close()

		assert.GreaterOrEqual(t, duration, 100*time.Millisecond, "third request should be rate limited")
	})

	t.Run("selector mode", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Two limiters: fast and slow
		fastLimiter := rate.NewLimiter(100, 100)
		slowLimiter := rate.NewLimiter(1, 1)

		selector := func(req *http.Request) (*rate.Limiter, string) {
			if strings.Contains(req.URL.Path, "/fast") {
				return fastLimiter, "fast"
			}
			return slowLimiter, "slow"
		}

		transport := middleware.RateLimit(middleware.RateLimitConfig{
			Selector: selector,
		})(http.DefaultTransport)

		// Fast endpoint should not be rate limited
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/fast", http.NoBody)
		start := time.Now()
		resp, err := transport.RoundTrip(req)
		duration := time.Since(start)

		require.NoError(t, err)
		resp.Body.Close()

		assert.Less(t, duration, 50*time.Millisecond, "fast endpoint should not be rate limited")

		// Slow endpoint - use up the token
		req, _ = http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/slow", http.NoBody)
		resp, _ = transport.RoundTrip(req)
		resp.Body.Close()

		// Second slow request should be rate limited
		req, _ = http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/slow", http.NoBody)
		start = time.Now()
		resp, err = transport.RoundTrip(req)
		duration = time.Since(start)

		require.NoError(t, err)
		resp.Body.Close()

		assert.GreaterOrEqual(t, duration, 500*time.Millisecond, "slow endpoint should be rate limited")
	})

	t.Run("nil limiter - no rate limiting", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		transport := middleware.RateLimit(middleware.RateLimitConfig{
			Limiter: nil, // No limiter
		})(http.DefaultTransport)

		// Should complete quickly without rate limiting
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
		start := time.Now()
		resp, err := transport.RoundTrip(req)
		duration := time.Since(start)

		require.NoError(t, err)
		resp.Body.Close()

		assert.Less(t, duration, 50*time.Millisecond, "request should complete quickly without rate limiting")
	})

	t.Run("context cancellation", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Very restrictive limiter
		limiter := rate.NewLimiter(0.1, 1)
		limiter.Allow() // Use up the token

		transport := middleware.RateLimit(middleware.RateLimitConfig{
			Limiter: limiter,
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
