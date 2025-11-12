package httpclient_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lexfrei/go-unifi/internal/httpclient"
)

func TestNew(t *testing.T) {
	t.Parallel()

	client := httpclient.New()
	if client == nil {
		t.Fatal("New() returned nil")
	}

	httpClient := client.HTTPClient()
	if httpClient == nil {
		t.Fatal("HTTPClient() returned nil")
	}

	if httpClient.Timeout != 30*time.Second {
		t.Errorf("Default timeout = %v, want %v", httpClient.Timeout, 30*time.Second)
	}
}

func TestWithTimeout(t *testing.T) {
	t.Parallel()

	timeout := 10 * time.Second
	client := httpclient.New(httpclient.WithTimeout(timeout))

	if client.HTTPClient().Timeout != timeout {
		t.Errorf("Timeout = %v, want %v", client.HTTPClient().Timeout, timeout)
	}
}

func TestWithHTTPClient(t *testing.T) {
	t.Parallel()

	customClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	client := httpclient.New(httpclient.WithHTTPClient(customClient))

	if client.HTTPClient() != customClient {
		t.Error("HTTPClient() did not return the custom client")
	}
}

func TestWithTransport(t *testing.T) {
	t.Parallel()

	customTransport := &http.Transport{}
	client := httpclient.New(httpclient.WithTransport(customTransport))

	if client.HTTPClient().Transport != customTransport {
		t.Error("Transport was not set correctly")
	}
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
			return resp, err
		})
	}

	middleware2 := func(next http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			order = append(order, "middleware2-before")
			resp, err := next.RoundTrip(req)
			order = append(order, "middleware2-after")
			return resp, err
		})
	}

	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "server")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with middleware
	client := httpclient.New(
		httpclient.WithMiddleware(middleware1, middleware2),
	)

	// Make request
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Verify order: middleware1 (outer) wraps middleware2 (inner)
	expectedOrder := []string{
		"middleware1-before",
		"middleware2-before",
		"server",
		"middleware2-after",
		"middleware1-after",
	}

	if len(order) != len(expectedOrder) {
		t.Fatalf("Order length = %d, want %d", len(order), len(expectedOrder))
	}

	for i, want := range expectedOrder {
		if order[i] != want {
			t.Errorf("order[%d] = %s, want %s", i, order[i], want)
		}
	}
}

func TestDo(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test response"))
	}))
	defer server.Close()

	client := httpclient.New()
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "test response" {
		t.Errorf("Body = %s, want %s", string(body), "test response")
	}
}

// roundTripperFunc is an adapter to use functions as http.RoundTripper
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// BenchmarkClient measures the overhead of the client with and without middleware.
func BenchmarkClient(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	b.Run("NoMiddleware", func(b *testing.B) {
		client := httpclient.New()
		req, _ := http.NewRequest(http.MethodGet, server.URL, nil)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
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
		req, _ := http.NewRequest(http.MethodGet, server.URL, nil)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := client.Do(req)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}
