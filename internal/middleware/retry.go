// Package middleware provides reusable HTTP middleware components.
package middleware

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/lexfrei/go-unifi/internal/observability"
	"github.com/lexfrei/go-unifi/internal/retry"
)

// RetryConfig configures the retry middleware.
type RetryConfig struct {
	MaxRetries  int
	InitialWait time.Duration
	Logger      observability.Logger
	Metrics     observability.MetricsRecorder
}

// Retry returns a middleware that retries failed requests with exponential backoff.
// It retries on:
// - Network errors (connection failures, timeouts).
// - 5xx server errors.
// - 429 rate limit errors (respects Retry-After header).
//
// It does NOT retry on:
// - 4xx client errors (except 429).
// - Successful responses (2xx, 3xx).
func Retry(cfg RetryConfig) func(http.RoundTripper) http.RoundTripper {
	if cfg.Logger == nil {
		cfg.Logger = observability.NoopLogger()
	}
	if cfg.Metrics == nil {
		cfg.Metrics = observability.NoopMetricsRecorder()
	}

	return func(next http.RoundTripper) http.RoundTripper {
		return &retryTransport{
			next:        next,
			maxRetries:  cfg.MaxRetries,
			initialWait: cfg.InitialWait,
			logger:      cfg.Logger,
			metrics:     cfg.Metrics,
		}
	}
}

type retryTransport struct {
	next        http.RoundTripper
	maxRetries  int
	initialWait time.Duration
	logger      observability.Logger
	metrics     observability.MetricsRecorder
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	// Read and buffer request body for retries
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, errors.Wrap(err, "failed to read request body")
		}
	}

	var lastErr error
	var lastResp *http.Response

	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		// Restore request body for retry
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		// Make request
		resp, err := t.next.RoundTrip(req)

		// Success case
		if err == nil && !retry.ShouldRetry(resp.StatusCode) {
			return resp, nil
		}

		// Store last error/response
		lastErr = err
		lastResp = resp

		// No more retries
		if attempt == t.maxRetries {
			break
		}

		// Log retry
		t.logger.Warn("retrying request",
			observability.Field{Key: "attempt", Value: attempt + 1},
			observability.Field{Key: "max_retries", Value: t.maxRetries},
			observability.Field{Key: "url", Value: req.URL.String()},
			observability.Field{Key: "method", Value: req.Method},
		)

		t.metrics.RecordRetry(attempt+1, req.URL.Path)

		// Calculate wait time
		waitTime := t.calculateWait(attempt, resp)

		// Wait before retry (respect context cancellation)
		select {
		case <-time.After(waitTime):
		case <-ctx.Done():
			return nil, errors.Wrap(ctx.Err(), "context canceled during retry wait")
		}

		// Close previous response body if exists
		if resp != nil {
			resp.Body.Close()
		}
	}

	// All retries exhausted
	if lastResp != nil {
		return lastResp, nil
	}

	return nil, errors.Wrapf(lastErr, "request failed after %d retries", t.maxRetries)
}

// calculateWait determines how long to wait before next retry.
// Uses exponential backoff: initialWait * 2^attempt
// Respects Retry-After header for 429 responses.
func (t *retryTransport) calculateWait(attempt int, resp *http.Response) time.Duration {
	// Check Retry-After header for 429 responses
	if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			if wait := retry.ParseRetryAfter(retryAfter); wait > 0 {
				t.logger.Debug("using Retry-After header",
					observability.Field{Key: "retry_after", Value: retryAfter},
					observability.Field{Key: "wait", Value: wait},
				)
				return wait
			}
		}
	}

	// Exponential backoff: initialWait * 2^attempt
	wait := t.initialWait * time.Duration(1<<attempt)

	t.logger.Debug("calculated exponential backoff",
		observability.Field{Key: "attempt", Value: attempt},
		observability.Field{Key: "wait", Value: wait},
	)

	return wait
}
