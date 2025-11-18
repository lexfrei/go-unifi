package middleware_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lexfrei/go-unifi/internal/middleware"
)

// BenchmarkRetryBodyBuffering benchmarks the retry middleware with different payload sizes
// to measure the effect of sync.Pool optimization.
func BenchmarkRetryBodyBuffering(b *testing.B) {
	testCases := []struct {
		name        string
		payloadSize int
	}{
		{"SmallPayload_60B", 60},
		{"MediumPayload_1KB", 1024},
		{"LargePayload_10KB", 10 * 1024},
		{"XLargePayload_100KB", 100 * 1024},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Create test payload
			payload := bytes.Repeat([]byte("x"), tc.payloadSize)

			// Server that succeeds on first attempt (no retries)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status":"ok"}`))
			}))
			defer server.Close()

			// Create retry transport
			transport := middleware.Retry(middleware.RetryConfig{
				MaxRetries:  3,
				InitialWait: 10 * time.Millisecond,
			})(http.DefaultTransport)

			client := &http.Client{Transport: transport}

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				// Create request with payload
				req, _ := http.NewRequestWithContext(
					context.Background(),
					http.MethodPost,
					server.URL,
					io.NopCloser(bytes.NewReader(payload)),
				)

				resp, err := client.Do(req)
				if err != nil {
					b.Fatal(err)
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		})
	}
}

// BenchmarkRetryWithRetries benchmarks retry middleware when retries actually happen.
func BenchmarkRetryWithRetries(b *testing.B) {
	payload := bytes.Repeat([]byte("x"), 1024)
	attempts := 0

	// Server that fails first 2 attempts, succeeds on 3rd
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts%3 != 0 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	transport := middleware.Retry(middleware.RetryConfig{
		MaxRetries:  3,
		InitialWait: time.Millisecond,
	})(http.DefaultTransport)

	client := &http.Client{Transport: transport}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		req, _ := http.NewRequestWithContext(
			context.Background(),
			http.MethodPost,
			server.URL,
			io.NopCloser(bytes.NewReader(payload)),
		)

		resp, err := client.Do(req)
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
