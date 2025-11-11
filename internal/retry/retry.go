package retry

import (
	"strconv"
	"time"
)

// ShouldRetry returns true if the HTTP status code indicates a retryable error.
// Retryable errors include:
//   - 429 (Too Many Requests) - rate limit exceeded
//   - 5xx (Server Errors) - temporary server-side issues
func ShouldRetry(statusCode int) bool {
	return statusCode >= 500 || statusCode == 429
}

// ParseRetryAfter parses the Retry-After HTTP header and returns the duration to wait.
// The Retry-After header can contain either:
//   - Number of seconds (e.g., "120")
//   - HTTP-date (not currently supported, returns 0)
//
// Returns 0 if the header is empty or cannot be parsed.
func ParseRetryAfter(retryAfterHeader string) time.Duration {
	if retryAfterHeader == "" {
		return 0
	}

	seconds, err := strconv.Atoi(retryAfterHeader)
	if err == nil {
		return time.Duration(seconds) * time.Second
	}

	return 0
}
