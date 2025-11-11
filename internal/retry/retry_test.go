package retry

import (
	"testing"
	"time"
)

func TestShouldRetry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{
			name:       "429 Too Many Requests",
			statusCode: 429,
			want:       true,
		},
		{
			name:       "500 Internal Server Error",
			statusCode: 500,
			want:       true,
		},
		{
			name:       "502 Bad Gateway",
			statusCode: 502,
			want:       true,
		},
		{
			name:       "503 Service Unavailable",
			statusCode: 503,
			want:       true,
		},
		{
			name:       "504 Gateway Timeout",
			statusCode: 504,
			want:       true,
		},
		{
			name:       "200 OK",
			statusCode: 200,
			want:       false,
		},
		{
			name:       "400 Bad Request",
			statusCode: 400,
			want:       false,
		},
		{
			name:       "401 Unauthorized",
			statusCode: 401,
			want:       false,
		},
		{
			name:       "403 Forbidden",
			statusCode: 403,
			want:       false,
		},
		{
			name:       "404 Not Found",
			statusCode: 404,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ShouldRetry(tt.statusCode); got != tt.want {
				t.Errorf("ShouldRetry(%d) = %v, want %v", tt.statusCode, got, tt.want)
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		header string
		want   time.Duration
	}{
		{
			name:   "empty header",
			header: "",
			want:   0,
		},
		{
			name:   "valid seconds - 60",
			header: "60",
			want:   60 * time.Second,
		},
		{
			name:   "valid seconds - 120",
			header: "120",
			want:   120 * time.Second,
		},
		{
			name:   "valid seconds - 1",
			header: "1",
			want:   1 * time.Second,
		},
		{
			name:   "valid seconds - 0",
			header: "0",
			want:   0,
		},
		{
			name:   "invalid format - text",
			header: "invalid",
			want:   0,
		},
		{
			name:   "invalid format - HTTP date (not supported)",
			header: "Wed, 21 Oct 2015 07:28:00 GMT",
			want:   0,
		},
		{
			name:   "invalid format - float",
			header: "60.5",
			want:   0,
		},
		{
			name:   "invalid format - negative",
			header: "-1",
			want:   -1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ParseRetryAfter(tt.header); got != tt.want {
				t.Errorf("ParseRetryAfter(%q) = %v, want %v", tt.header, got, tt.want)
			}
		})
	}
}

func BenchmarkShouldRetry(b *testing.B) {
	statusCodes := []int{200, 400, 429, 500, 502, 503, 504}

	for i := 0; i < b.N; i++ {
		for _, code := range statusCodes {
			ShouldRetry(code)
		}
	}
}

func BenchmarkParseRetryAfter(b *testing.B) {
	headers := []string{"", "60", "120", "invalid"}

	for i := 0; i < b.N; i++ {
		for _, header := range headers {
			ParseRetryAfter(header)
		}
	}
}
