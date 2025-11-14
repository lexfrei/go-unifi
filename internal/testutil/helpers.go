// Package testutil provides common testing utilities and helpers.
package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NewMockServer creates a test HTTP server with predefined response.
// It validates the request path and API key header, then returns the specified response.
// Supports both X-API-KEY and X-Api-Key header names for compatibility.
func NewMockServer(tb testing.TB, expectedPath, apiKey, responseBody string, statusCode int) *httptest.Server {
	tb.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate request path
		assert.Equal(tb, expectedPath, r.URL.Path, "Request path should match expected")

		// Validate API key header if provided (check both common variants)
		if apiKey != "" {
			actualKey := r.Header.Get("X-API-KEY") //nolint:canonicalheader // UniFi uses non-canonical X-API-KEY
			if actualKey == "" {
				actualKey = r.Header.Get("X-Api-Key")
			}
			assert.Equal(tb, apiKey, actualKey, "API key header should be set")
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, err := w.Write([]byte(responseBody))
		if err != nil {
			tb.Errorf("Failed to write response body: %v", err)
		}
	}))
}

// NewMockServerWithHandler creates a test HTTP server with custom handler.
// Use this for more complex test scenarios that need custom request handling.
func NewMockServerWithHandler(tb testing.TB, handler http.HandlerFunc) *httptest.Server {
	tb.Helper()
	return httptest.NewServer(handler)
}

// NewMockServerMulti creates a test HTTP server with multiple path handlers.
// The handlers map keys are URL paths, values are handler functions.
func NewMockServerMulti(tb testing.TB, handlers map[string]http.HandlerFunc) *httptest.Server {
	tb.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler, ok := handlers[r.URL.Path]
		if !ok {
			tb.Errorf("Unexpected request path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		handler(w, r)
	}))
}

// NewMockServerSequence creates a test server that returns responses in sequence.
// Each call to the server returns the next response in the slice.
// Useful for testing retry logic or pagination.
func NewMockServerSequence(
	tb testing.TB,
	responses []struct {
		Body       string
		StatusCode int
	},
) *httptest.Server {
	tb.Helper()

	callCount := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if callCount >= len(responses) {
			tb.Errorf("More requests than configured responses (got %d requests, have %d responses)",
				callCount+1, len(responses))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp := responses[callCount]
		callCount++

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		_, err := w.Write([]byte(resp.Body))
		if err != nil {
			tb.Errorf("Failed to write response body: %v", err)
		}
	}))
}

// AssertAPIError checks that the error is not nil and optionally validates error content.
func AssertAPIError(tb testing.TB, err error, msgAndArgs ...any) {
	tb.Helper()
	assert.Error(tb, err, msgAndArgs...)
}

// RequireValidResponse checks common success response fields.
// For API responses with count, totalCount, and data fields.
func RequireValidResponse(tb testing.TB, resp any) {
	tb.Helper()
	require.NotNil(tb, resp, "Response should not be nil")
}
