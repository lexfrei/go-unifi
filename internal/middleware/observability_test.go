package middleware

import (
	"testing"
)

// Shared test path constants for normalizePath tests and benchmarks.
const (
	pathDNSRecordObjectID    = "/api/site/default/dns/record/507f1f77bcf86cd799439011"
	pathDeviceNumericID      = "/api/site/my-site/device/12345678"
	pathSettingWAN           = "/proxy/network/v2/api/site/default/setting/wan"
	pathHostStats            = "/api/v1/host/a1b2c3d4-e5f6-7890-abcd-ef1234567890/stats"
	pathClientNumericID      = "/api/site/production/client/100000"
	pathProxyDNSRecord       = "/proxy/network/v2/api/site/default/dns/record/507f1f77bcf86cd799439011"
	pathNormalizedDevicePort = "/api/site/:site/device/:id/port/:id"
	pathDeviceUUID           = "/api/site/default/device/550e8400-e29b-41d4-a716-446655440000"
	pathNormalizedDevice     = "/api/site/:site/device/:id"
	pathSiteCustomDNSRecord  = "/api/site/my-custom-site/dns/record"
	pathSystemInfo           = "/api/system/info"
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
			input:    pathProxyDNSRecord,
			expected: "/proxy/network/v2/api/site/:site/dns/record/:id",
		},
		{
			name:     "Multiple ObjectIDs",
			input:    "/api/site/default/device/507f1f77bcf86cd799439011/port/507f1f77bcf86cd799439012",
			expected: pathNormalizedDevicePort,
		},
		{
			name:     "UUID format",
			input:    pathDeviceUUID,
			expected: pathNormalizedDevice,
		},
		{
			name:     "Numeric ID (long)",
			input:    "/api/site/default/device/12345678",
			expected: pathNormalizedDevice,
		},
		{
			name:     "Short numeric ID preserved (version numbers)",
			input:    "/proxy/network/v2/api/site/default",
			expected: "/proxy/network/v2/api/site/:site",
		},
		{
			name:     "Site name normalization",
			input:    pathSiteCustomDNSRecord,
			expected: "/api/site/:site/dns/record",
		},
		{
			name:     "Multiple site references",
			input:    "/api/site/site1/device/abc/site/site2/config",
			expected: "/api/site/:site/device/abc/site/:site/config",
		},
		{
			name:     "Path without IDs",
			input:    pathSystemInfo,
			expected: pathSystemInfo,
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
			expected: pathNormalizedDevicePort,
		},
		{
			name:     "Path ending with ID",
			input:    pathDNSRecordObjectID,
			expected: "/api/site/:site/dns/record/:id",
		},
		{
			name:     "Numeric ID at end of path",
			input:    "/api/site/default/device/123456789",
			expected: pathNormalizedDevice,
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
		pathProxyDNSRecord,
		pathDeviceUUID,
		pathSiteCustomDNSRecord,
		pathSystemInfo,
	}

	b.ResetTimer()
	for b.Loop() {
		for _, path := range paths {
			_ = normalizePath(path)
		}
	}
}
