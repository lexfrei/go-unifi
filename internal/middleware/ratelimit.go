package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/lexfrei/go-unifi/internal/observability"
	"golang.org/x/time/rate"
)

// RateLimiterSelector chooses which rate limiter to use for a given request.
// Returns the rate limiter and a descriptive name for logging/metrics.
type RateLimiterSelector func(*http.Request) (*rate.Limiter, string)

// RateLimitConfig configures the rate limit middleware.
type RateLimitConfig struct {
	Limiter  *rate.Limiter         // Single limiter (used if Selector is nil)
	Selector RateLimiterSelector   // Optional: select limiter based on request
	Logger   observability.Logger
	Metrics  observability.MetricsRecorder
}

// RateLimit returns a middleware that applies rate limiting to requests.
//
// Two modes of operation:
// 1. Single limiter: Set cfg.Limiter for uniform rate limiting
// 2. Selector mode: Set cfg.Selector to choose limiter per request (e.g., v1 vs EA endpoints)
func RateLimit(cfg RateLimitConfig) func(http.RoundTripper) http.RoundTripper {
	if cfg.Logger == nil {
		cfg.Logger = observability.NoopLogger()
	}
	if cfg.Metrics == nil {
		cfg.Metrics = observability.NoopMetricsRecorder()
	}

	return func(next http.RoundTripper) http.RoundTripper {
		return &rateLimitTransport{
			next:     next,
			limiter:  cfg.Limiter,
			selector: cfg.Selector,
			logger:   cfg.Logger,
			metrics:  cfg.Metrics,
		}
	}
}

type rateLimitTransport struct {
	next     http.RoundTripper
	limiter  *rate.Limiter
	selector RateLimiterSelector
	logger   observability.Logger
	metrics  observability.MetricsRecorder
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	// Select rate limiter
	limiter := t.limiter
	endpoint := "default"

	if t.selector != nil {
		limiter, endpoint = t.selector(req)
	}

	if limiter == nil {
		// No rate limiting
		return t.next.RoundTrip(req)
	}

	// Wait for rate limiter
	if err := t.waitWithObservability(ctx, limiter, endpoint, req.URL.Path); err != nil {
		return nil, err
	}

	return t.next.RoundTrip(req)
}

func (t *rateLimitTransport) waitWithObservability(
	ctx context.Context,
	limiter *rate.Limiter,
	endpoint string,
	path string,
) error {
	// Check if we need to wait
	reservation := limiter.Reserve()
	if !reservation.OK() {
		return errors.New("rate limit reservation failed")
	}

	delay := reservation.Delay()
	if delay > 0 {
		t.logger.Debug("rate limit delay",
			observability.Field{Key: "endpoint", Value: endpoint},
			observability.Field{Key: "delay", Value: delay},
			observability.Field{Key: "path", Value: path},
		)

		t.metrics.RecordRateLimit(path, delay)

		// Wait with context cancellation support
		timer := time.NewTimer(delay)
		defer timer.Stop()

		select {
		case <-timer.C:
			// Rate limit satisfied
		case <-ctx.Done():
			reservation.Cancel()
			return errors.Wrap(ctx.Err(), "context canceled during rate limit wait")
		}
	}

	return nil
}
