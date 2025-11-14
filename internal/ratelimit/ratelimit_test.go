package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRateLimiter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		requestsPerMinute int
		wantRate          float64
		wantBurst         int
	}{
		{
			name:              "1000 requests per minute",
			requestsPerMinute: 1000,
			wantRate:          1000.0 / 60.0,
			wantBurst:         1000,
		},
		{
			name:              "10000 requests per minute",
			requestsPerMinute: 10000,
			wantRate:          10000.0 / 60.0,
			wantBurst:         10000,
		},
		{
			name:              "100 requests per minute",
			requestsPerMinute: 100,
			wantRate:          100.0 / 60.0,
			wantBurst:         100,
		},
		{
			name:              "60 requests per minute (1 per second)",
			requestsPerMinute: 60,
			wantRate:          1.0,
			wantBurst:         60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			limiter := NewRateLimiter(tt.requestsPerMinute)

			require.NotNil(t, limiter)

			// Check rate
			gotRate := float64(limiter.Limit())
			assert.InDelta(t, tt.wantRate, gotRate, 0.001)

			// Check burst
			gotBurst := limiter.Burst()
			assert.Equal(t, tt.wantBurst, gotBurst)
		})
	}
}

func TestRateLimiterAllowsBurst(t *testing.T) {
	t.Parallel()
	// Create limiter with 60 req/min (1 req/sec, burst of 60)
	limiter := NewRateLimiter(60)

	ctx := context.Background()

	// Should allow burst of 60 requests immediately
	for i := range 60 {
		err := limiter.Wait(ctx)
		require.NoError(t, err, "Request %d failed", i)
	}
}

func TestRateLimiterThrottles(t *testing.T) {
	t.Parallel()
	// Create limiter with 60 req/min (1 req/sec)
	limiter := NewRateLimiter(60)

	ctx := context.Background()

	// Exhaust burst
	for i := range 60 {
		err := limiter.Wait(ctx)
		require.NoError(t, err, "Burst request %d failed", i)
	}

	// Next request should be throttled
	start := time.Now()
	err := limiter.Wait(ctx)
	require.NoError(t, err)
	elapsed := time.Since(start)

	// Should wait approximately 1 second (with some tolerance)
	minWait := 900 * time.Millisecond
	maxWait := 1100 * time.Millisecond

	assert.GreaterOrEqual(t, elapsed, minWait)
	assert.LessOrEqual(t, elapsed, maxWait)
}

func TestRateLimiterContextCancellation(t *testing.T) {
	t.Parallel()
	// Create limiter with very low rate
	limiter := NewRateLimiter(1)

	ctx, cancel := context.WithCancel(context.Background())

	// Exhaust burst
	err := limiter.Wait(ctx)
	require.NoError(t, err)

	// Cancel context
	cancel()

	// Next request should fail with context cancellation
	err = limiter.Wait(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRateLimiterHighThroughput(t *testing.T) {
	t.Parallel()
	// Create limiter with 1000 req/min
	limiter := NewRateLimiter(1000)

	ctx := context.Background()
	start := time.Now()

	// Make 100 requests
	requestCount := 100
	for i := range requestCount {
		err := limiter.Wait(ctx)
		require.NoError(t, err, "Request %d failed", i)
	}

	elapsed := time.Since(start)

	// At 1000 req/min (16.67 req/sec), 100 requests should take ~6 seconds
	// But with burst capacity of 1000, all should be near-instant
	maxExpected := 1 * time.Second

	assert.LessOrEqual(t, elapsed, maxExpected)
}

func BenchmarkNewRateLimiter(b *testing.B) {
	for b.Loop() {
		NewRateLimiter(1000)
	}
}

func BenchmarkRateLimiterAllow(b *testing.B) {
	limiter := NewRateLimiter(10000)

	for b.Loop() {
		limiter.Allow()
	}
}

func TestConcurrentAccess(t *testing.T) {
	t.Parallel()

	limiter := NewRateLimiter(1000) // High rate for fast test

	const goroutines = 10
	const requestsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			for range requestsPerGoroutine {
				_ = limiter.Allow()
			}
		}()
	}

	wg.Wait()
	// If no race detector warnings, test passes
}

func TestConcurrentWait(t *testing.T) {
	t.Parallel()

	limiter := NewRateLimiter(6000) // High rate: 100 req/sec for fast test

	const goroutines = 10
	const requestsPerGoroutine = 10

	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			for range requestsPerGoroutine {
				err := limiter.Wait(ctx)
				assert.NoError(t, err)
			}
		}()
	}

	wg.Wait()
	// If no race detector warnings, test passes
}
