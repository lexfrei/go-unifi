package middleware_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lexfrei/go-unifi/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRetryResourceCleanupOnCancellation verifies that all resources are properly
// cleaned up when context is canceled during retry wait.
func TestRetryResourceCleanupOnCancellation(t *testing.T) {
	t.Parallel()

	var requestCount atomic.Int32
	var bodiesClosed atomic.Int32

	// Server that always returns 500 to trigger retries
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	// Custom HTTP client that tracks body closures
	trackingTransport := &bodyTrackingTransport{
		base:         http.DefaultTransport,
		bodiesClosed: &bodiesClosed,
	}

	// Apply retry middleware
	transport := middleware.Retry(middleware.RetryConfig{
		MaxRetries:  5,
		InitialWait: 100 * time.Millisecond,
	})(trackingTransport)

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Make request that will be canceled during retry
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, http.NoBody)
	require.NoError(t, err)

	client := &http.Client{Transport: transport}
	resp, err := client.Do(req)

	// Should fail due to context cancellation
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context")

	// Response should be nil on context cancellation
	if resp != nil {
		resp.Body.Close()
	}
	assert.Nil(t, resp)

	// Wait a bit for cleanup
	time.Sleep(100 * time.Millisecond)

	// Verify: at least one request was made
	assert.Positive(t, requestCount.Load(), "should have made at least one request")

	// Verify: all response bodies were closed
	closedCount := bodiesClosed.Load()
	assert.Equal(t, requestCount.Load(), closedCount,
		"all response bodies should be closed (requests: %d, closed: %d)",
		requestCount.Load(), closedCount)
}

// TestRetryNoGoroutineLeaks verifies that retry middleware doesn't leak goroutines.
func TestRetryNoGoroutineLeaks(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	transport := middleware.Retry(middleware.RetryConfig{
		MaxRetries:  3,
		InitialWait: 10 * time.Millisecond,
	})(http.DefaultTransport)

	// Force GC and get baseline goroutine count
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baselineGoroutines := runtime.NumGoroutine()

	// Make 100 requests with cancellation
	for range 100 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, http.NoBody)

		client := &http.Client{Transport: transport}
		resp, err := client.Do(req)
		if resp != nil {
			resp.Body.Close()
		}
		_ = err

		cancel()
	}

	// Force GC and wait for cleanup
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()

	// Allow small variance (±5 goroutines) due to test framework overhead
	assert.InDelta(t, baselineGoroutines, finalGoroutines, 5,
		"goroutine leak detected (baseline: %d, final: %d)",
		baselineGoroutines, finalGoroutines)
}

// TestRetryStressTestConcurrentCancellations stress tests the retry middleware
// with many concurrent requests that get canceled, verifying no resource leaks.
func TestRetryStressTestConcurrentCancellations(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`error`))
	}))
	defer server.Close()

	transport := middleware.Retry(middleware.RetryConfig{
		MaxRetries:  5,
		InitialWait: 10 * time.Millisecond,
	})(http.DefaultTransport)

	// Get baseline metrics before stress test
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	baselineGoroutines := runtime.NumGoroutine()

	// Run 500 concurrent requests with random cancellation
	const numRequests = 500
	var wg sync.WaitGroup
	wg.Add(numRequests)

	for i := range numRequests {
		go func(index int) {
			defer wg.Done()

			// Random short timeout to trigger cancellation at different stages
			timeout := time.Duration(5+index%50) * time.Millisecond
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, http.NoBody)
			client := &http.Client{Transport: transport}

			resp, err := client.Do(req)
			if resp != nil {
				// Always close the response body
				resp.Body.Close()
			}
			_ = err
		}(i)
	}

	wg.Wait()

	// Force cleanup and check for resource leaks
	runtime.GC()
	time.Sleep(200 * time.Millisecond)
	finalGoroutines := runtime.NumGoroutine()

	t.Logf("Stress test complete: baseline goroutines=%d, final=%d",
		baselineGoroutines, finalGoroutines)

	// Verify no significant goroutine leaks (allow some variance for test framework)
	assert.InDelta(t, baselineGoroutines, finalGoroutines, 10,
		"no goroutine leaks detected after %d concurrent cancellations", numRequests)
}

// BenchmarkRetryCancellation benchmarks the retry middleware with context cancellation.
func BenchmarkRetryCancellation(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	transport := middleware.Retry(middleware.RetryConfig{
		MaxRetries:  3,
		InitialWait: 1 * time.Millisecond,
	})(http.DefaultTransport)

	b.ResetTimer()
	for b.Loop() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Millisecond)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, http.NoBody)

		client := &http.Client{Transport: transport}
		resp, err := client.Do(req)
		if resp != nil {
			resp.Body.Close()
		}
		_ = err

		cancel()
	}
}

// bodyTrackingTransport wraps http.RoundTripper to track response body closures.
type bodyTrackingTransport struct {
	base             http.RoundTripper
	bodiesClosed     *atomic.Int32
	responsesCreated *atomic.Int32
}

func (t *bodyTrackingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		//nolint:wrapcheck // Test helper passes through transport errors unchanged
		return nil, err
	}

	// Track that we created a response
	if t.responsesCreated != nil {
		t.responsesCreated.Add(1)
	}

	// Wrap response body to track closure
	resp.Body = &trackingReadCloser{
		ReadCloser:   resp.Body,
		bodiesClosed: t.bodiesClosed,
	}

	return resp, nil
}

// trackingReadCloser wraps io.ReadCloser to count closures.
type trackingReadCloser struct {
	io.ReadCloser

	bodiesClosed *atomic.Int32
	closed       atomic.Bool
}

func (r *trackingReadCloser) Close() error {
	if r.closed.CompareAndSwap(false, true) {
		r.bodiesClosed.Add(1)
	}

	//nolint:wrapcheck // Test helper delegates close to wrapped ReadCloser
	return r.ReadCloser.Close()
}
