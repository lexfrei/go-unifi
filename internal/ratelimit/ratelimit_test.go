package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
)

func TestNewRateLimiter(t *testing.T) {
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
			limiter := NewRateLimiter(tt.requestsPerMinute)

			if limiter == nil {
				t.Fatal("NewRateLimiter returned nil")
			}

			// Check rate
			gotRate := float64(limiter.Limit())
			if gotRate != tt.wantRate {
				t.Errorf("Rate = %v, want %v", gotRate, tt.wantRate)
			}

			// Check burst
			gotBurst := limiter.Burst()
			if gotBurst != tt.wantBurst {
				t.Errorf("Burst = %v, want %v", gotBurst, tt.wantBurst)
			}
		})
	}
}

func TestRateLimiterAllowsBurst(t *testing.T) {
	// Create limiter with 60 req/min (1 req/sec, burst of 60)
	limiter := NewRateLimiter(60)

	ctx := context.Background()

	// Should allow burst of 60 requests immediately
	for i := 0; i < 60; i++ {
		if err := limiter.Wait(ctx); err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
	}
}

func TestRateLimiterThrottles(t *testing.T) {
	// Create limiter with 60 req/min (1 req/sec)
	limiter := NewRateLimiter(60)

	ctx := context.Background()

	// Exhaust burst
	for i := 0; i < 60; i++ {
		if err := limiter.Wait(ctx); err != nil {
			t.Fatalf("Burst request %d failed: %v", i, err)
		}
	}

	// Next request should be throttled
	start := time.Now()
	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("Throttled request failed: %v", err)
	}
	elapsed := time.Since(start)

	// Should wait approximately 1 second (with some tolerance)
	minWait := 900 * time.Millisecond
	maxWait := 1100 * time.Millisecond

	if elapsed < minWait || elapsed > maxWait {
		t.Errorf("Wait time = %v, want between %v and %v", elapsed, minWait, maxWait)
	}
}

func TestRateLimiterContextCancellation(t *testing.T) {
	// Create limiter with very low rate
	limiter := NewRateLimiter(1)

	ctx, cancel := context.WithCancel(context.Background())

	// Exhaust burst
	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("First request failed: %v", err)
	}

	// Cancel context
	cancel()

	// Next request should fail with context cancellation
	if err := limiter.Wait(ctx); err == nil {
		t.Error("Expected error from cancelled context, got nil")
	} else if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

func TestRateLimiterHighThroughput(t *testing.T) {
	// Create limiter with 1000 req/min
	limiter := NewRateLimiter(1000)

	ctx := context.Background()
	start := time.Now()

	// Make 100 requests
	requestCount := 100
	for i := 0; i < requestCount; i++ {
		if err := limiter.Wait(ctx); err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
	}

	elapsed := time.Since(start)

	// At 1000 req/min (16.67 req/sec), 100 requests should take ~6 seconds
	// But with burst capacity of 1000, all should be near-instant
	maxExpected := 1 * time.Second

	if elapsed > maxExpected {
		t.Errorf("100 requests took %v, expected less than %v with burst", elapsed, maxExpected)
	}
}

func BenchmarkNewRateLimiter(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewRateLimiter(1000)
	}
}

func BenchmarkRateLimiterAllow(b *testing.B) {
	limiter := NewRateLimiter(10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow()
	}
}
