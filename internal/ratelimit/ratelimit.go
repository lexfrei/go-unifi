package ratelimit

import "golang.org/x/time/rate"

// NewRateLimiter creates a new rate limiter with specified requests per minute.
// It uses a token bucket algorithm where tokens are replenished continuously
// at the rate of requestsPerMinute/60 per second, with a burst capacity equal
// to requestsPerMinute.
func NewRateLimiter(requestsPerMinute int) *rate.Limiter {
	return rate.NewLimiter(rate.Limit(float64(requestsPerMinute)/60.0), requestsPerMinute)
}
