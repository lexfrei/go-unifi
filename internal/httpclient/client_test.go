package httpclient_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lexfrei/go-unifi/internal/httpclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	client := httpclient.New()
	require.NotNil(t, client, "New() returned nil")

	httpClient := client.HTTPClient()
	require.NotNil(t, httpClient, "HTTPClient() returned nil")

	assert.Equal(t, 30*time.Second, httpClient.Timeout, "Default timeout mismatch")
}

func TestWithTimeout(t *testing.T) {
	t.Parallel()

	timeout := 10 * time.Second
	client := httpclient.New(httpclient.WithTimeout(timeout))

	assert.Equal(t, timeout, client.HTTPClient().Timeout, "Timeout mismatch")
}

func TestWithHTTPClient(t *testing.T) {
	t.Parallel()

	customClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	client := httpclient.New(httpclient.WithHTTPClient(customClient))

	assert.Same(t, customClient, client.HTTPClient(), "HTTPClient() did not return the custom client")
}

func TestWithTransport(t *testing.T) {
	t.Parallel()

	customTransport := &http.Transport{}
	client := httpclient.New(httpclient.WithTransport(customTransport))

	assert.Same(t, customTransport, client.HTTPClient().Transport, "Transport was not set correctly")
}

func TestMiddlewareChaining(t *testing.T) {
	t.Parallel()

	var order []string

	// Create middleware that records execution order
	middleware1 := func(next http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			order = append(order, "middleware1-before")
			resp, err := next.RoundTrip(req)
			order = append(order, "middleware1-after")
			//nolint:wrapcheck // Test middleware - passes through errors unchanged
			return resp, err
		})
	}

	middleware2 := func(next http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			order = append(order, "middleware2-before")
			resp, err := next.RoundTrip(req)
			order = append(order, "middleware2-after")
			//nolint:wrapcheck // Test middleware - passes through errors unchanged
			return resp, err
		})
	}

	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		order = append(order, "server")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with middleware
	client := httpclient.New(
		httpclient.WithMiddleware(middleware1, middleware2),
	)

	// Make request
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
	resp, err := client.Do(req)
	require.NoError(t, err, "Request failed")
	defer resp.Body.Close()

	// Verify order: middleware1 (outer) wraps middleware2 (inner)
	expectedOrder := []string{
		"middleware1-before",
		"middleware2-before",
		"server",
		"middleware2-after",
		"middleware1-after",
	}

	assert.Equal(t, expectedOrder, order, "Middleware execution order mismatch")
}

func TestDo(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test response"))
	}))
	defer server.Close()

	client := httpclient.New()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)

	resp, err := client.Do(req)
	require.NoError(t, err, "Do() failed")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "StatusCode mismatch")

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "test response", string(body), "Body mismatch")
}

// roundTripperFunc is an adapter to use functions as http.RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// BenchmarkClient measures the overhead of the client with and without middleware.
func BenchmarkClient(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	b.Run("NoMiddleware", func(b *testing.B) {
		client := httpclient.New()
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)

		b.ResetTimer()
		for range b.N {
			resp, err := client.Do(req)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})

	b.Run("WithMiddleware", func(b *testing.B) {
		noop := func(next http.RoundTripper) http.RoundTripper {
			return next
		}

		client := httpclient.New(httpclient.WithMiddleware(noop))
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)

		b.ResetTimer()
		for range b.N {
			resp, err := client.Do(req)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}
