package middleware_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lexfrei/go-unifi/internal/middleware"
	"github.com/lexfrei/go-unifi/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for auth header
		apiKey := r.Header.Get("X-Api-Key")
		assert.Equal(t, "test-key-123", apiKey)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := middleware.Auth("X-Api-Key", "test-key-123")(http.DefaultTransport)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAuthDoesNotModifyOriginalRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := middleware.Auth("X-Api-Key", "test-key")(http.DefaultTransport)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
	originalHeaders := len(req.Header)

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Original request should not be modified
	assert.Len(t, req.Header, originalHeaders, "Original request should not be modified")
}

func TestTLSConfig(t *testing.T) {
	t.Parallel()

	config := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	transport := middleware.TLSConfig(config)(http.DefaultTransport)

	// Verify it's an HTTP transport with TLS config
	httpTransport, ok := transport.(*http.Transport)
	require.True(t, ok, "Transport should be *http.Transport")

	require.NotNil(t, httpTransport.TLSClientConfig, "TLSClientConfig should not be nil")

	assert.Equal(t, uint16(tls.VersionTLS12), httpTransport.TLSClientConfig.MinVersion)
}

func TestInsecureSkipVerify(t *testing.T) {
	t.Parallel()

	config := middleware.InsecureSkipVerify()

	require.NotNil(t, config, "InsecureSkipVerify() should not return nil")

	assert.True(t, config.InsecureSkipVerify, "InsecureSkipVerify should be true")
}

func TestObservability(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := observability.NoopLogger()
	metrics := observability.NoopMetricsRecorder()

	transport := middleware.Observability(logger, metrics)(http.DefaultTransport)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestObservabilityWithNilParams(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Should use no-op implementations
	transport := middleware.Observability(nil, nil)(http.DefaultTransport)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	resp.Body.Close()
}
