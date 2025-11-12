package middleware_test

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lexfrei/go-unifi/internal/middleware"
	"github.com/lexfrei/go-unifi/internal/observability"
)

func TestAuth(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for auth header
		apiKey := r.Header.Get("X-Api-Key")
		if apiKey != "test-key-123" {
			t.Errorf("X-Api-Key = %s, want %s", apiKey, "test-key-123")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := middleware.Auth("X-Api-Key", "test-key-123")(http.DefaultTransport)

	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestAuthDoesNotModifyOriginalRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := middleware.Auth("X-Api-Key", "test-key")(http.DefaultTransport)

	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	originalHeaders := len(req.Header)

	_, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}

	// Original request should not be modified
	if len(req.Header) != originalHeaders {
		t.Errorf("Original request was modified: headers = %d, want %d", len(req.Header), originalHeaders)
	}
}

func TestTLSConfig(t *testing.T) {
	t.Parallel()

	config := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	transport := middleware.TLSConfig(config)(http.DefaultTransport)

	// Verify it's an HTTP transport with TLS config
	httpTransport, ok := transport.(*http.Transport)
	if !ok {
		t.Fatal("Transport is not *http.Transport")
	}

	if httpTransport.TLSClientConfig == nil {
		t.Fatal("TLSClientConfig is nil")
	}

	if httpTransport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion = %d, want %d", httpTransport.TLSClientConfig.MinVersion, tls.VersionTLS12)
	}
}

func TestInsecureSkipVerify(t *testing.T) {
	t.Parallel()

	config := middleware.InsecureSkipVerify()

	if config == nil {
		t.Fatal("InsecureSkipVerify() returned nil")
	}

	if !config.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should be true")
	}
}

func TestObservability(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := observability.NoopLogger()
	metrics := observability.NoopMetricsRecorder()

	transport := middleware.Observability(logger, metrics)(http.DefaultTransport)

	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestObservabilityWithNilParams(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Should use no-op implementations
	transport := middleware.Observability(nil, nil)(http.DefaultTransport)

	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	defer resp.Body.Close()
}
