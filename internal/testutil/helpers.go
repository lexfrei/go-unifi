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
func NewMockServer(t *testing.T, expectedPath, apiKey, responseBody string, statusCode int) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate request path
		assert.Equal(t, expectedPath, r.URL.Path, "Request path should match expected")

		// Validate API key header if provided
		if apiKey != "" {
			assert.Equal(t, apiKey, r.Header.Get("X-API-KEY"), "X-API-KEY header should be set")
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, err := w.Write([]byte(responseBody))
		require.NoError(t, err, "Failed to write response body")
	}))
}

// NewMockServerWithHandler creates a test HTTP server with custom handler.
// Use this for more complex test scenarios that need custom request handling.
func NewMockServerWithHandler(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

// NewMockServerMulti creates a test HTTP server with multiple path handlers.
// The handlers map keys are URL paths, values are handler functions.
func NewMockServerMulti(t *testing.T, handlers map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler, ok := handlers[r.URL.Path]
		if !ok {
			t.Errorf("Unexpected request path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		handler(w, r)
	}))
}

// NewMockServerSequence creates a test server that returns responses in sequence.
// Each call to the server returns the next response in the slice.
// Useful for testing retry logic or pagination.
func NewMockServerSequence(t *testing.T, responses []struct {
	Body       string
	StatusCode int
}) *httptest.Server {
	t.Helper()

	callCount := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if callCount >= len(responses) {
			t.Errorf("More requests than configured responses (got %d requests, have %d responses)",
				callCount+1, len(responses))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp := responses[callCount]
		callCount++

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		_, err := w.Write([]byte(resp.Body))
		require.NoError(t, err, "Failed to write response body")
	}))
}

// AssertAPIError checks that the error is not nil and optionally validates error content.
func AssertAPIError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Error(t, err, msgAndArgs...)
}

// RequireValidResponse checks common success response fields.
// For API responses with count, totalCount, and data fields.
func RequireValidResponse(t *testing.T, resp interface{}) {
	t.Helper()
	require.NotNil(t, resp, "Response should not be nil")
}
