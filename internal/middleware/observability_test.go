package middleware

import (
	"testing"
)

func TestNormalizePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ObjectID in DNS record path",
			input:    "/proxy/network/v2/api/site/default/dns/record/507f1f77bcf86cd799439011",
			expected: "/proxy/network/v2/api/site/:site/dns/record/:id",
		},
		{
			name:     "Multiple ObjectIDs",
			input:    "/api/site/default/device/507f1f77bcf86cd799439011/port/507f1f77bcf86cd799439012",
			expected: "/api/site/:site/device/:id/port/:id",
		},
		{
			name:     "UUID format",
			input:    "/api/site/default/device/550e8400-e29b-41d4-a716-446655440000",
			expected: "/api/site/:site/device/:id",
		},
		{
			name:     "Numeric ID (long)",
			input:    "/api/site/default/device/12345678",
			expected: "/api/site/:site/device/:id",
		},
		{
			name:     "Short numeric ID preserved (version numbers)",
			input:    "/proxy/network/v2/api/site/default",
			expected: "/proxy/network/v2/api/site/:site",
		},
		{
			name:     "Site name normalization",
			input:    "/api/site/my-custom-site/dns/record",
			expected: "/api/site/:site/dns/record",
		},
		{
			name:     "Multiple site references",
			input:    "/api/site/site1/device/abc/site/site2/config",
			expected: "/api/site/:site/device/abc/site/:site/config",
		},
		{
			name:     "Path without IDs",
			input:    "/api/system/info",
			expected: "/api/system/info",
		},
		{
			name:     "Empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "Root path",
			input:    "/",
			expected: "/",
		},
		{
			name:     "Mixed UUID and ObjectID",
			input:    "/api/site/default/device/550e8400-e29b-41d4-a716-446655440000/port/507f1f77bcf86cd799439011",
			expected: "/api/site/:site/device/:id/port/:id",
		},
		{
			name:     "Path ending with ID",
			input:    "/api/site/default/dns/record/507f1f77bcf86cd799439011",
			expected: "/api/site/:site/dns/record/:id",
		},
		{
			name:     "Numeric ID at end of path",
			input:    "/api/site/default/device/123456789",
			expected: "/api/site/:site/device/:id",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := normalizePath(testCase.input)
			if result != testCase.expected {
				t.Errorf("normalizePath(%q) = %q, want %q", testCase.input, result, testCase.expected)
			}
		})
	}
}

func BenchmarkNormalizePath(b *testing.B) {
	paths := []string{
		"/proxy/network/v2/api/site/default/dns/record/507f1f77bcf86cd799439011",
		"/api/site/default/device/550e8400-e29b-41d4-a716-446655440000",
		"/api/site/my-custom-site/dns/record",
		"/api/system/info",
	}

	b.ResetTimer()
	for b.Loop() {
		for _, path := range paths {
			_ = normalizePath(path)
		}
	}
}
