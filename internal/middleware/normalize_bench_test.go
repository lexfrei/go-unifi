package middleware

import (
	"fmt"
	"testing"
)

// BenchmarkNormalizePathCached benchmarks normalizePath with cache hits.
func BenchmarkNormalizePathCached(b *testing.B) {
	testPaths := []string{
		"/api/site/default/dns/record/507f1f77bcf86cd799439011",
		"/api/site/my-site/device/12345678",
		"/proxy/network/v2/api/site/default/setting/wan",
		"/api/v1/host/a1b2c3d4-e5f6-7890-abcd-ef1234567890/stats",
		"/api/site/production/client/100000",
	}

	// Pre-warm the cache
	for _, path := range testPaths {
		_ = normalizePath(path)
	}

	b.ResetTimer()
	b.ReportAllocs()

	i := 0
	for b.Loop() {
		// Cycle through paths to simulate production traffic patterns
		path := testPaths[i%len(testPaths)]
		_ = normalizePath(path)
		i++
	}
}

// BenchmarkNormalizePathUncached benchmarks normalizePath with all cache misses.
func BenchmarkNormalizePathUncached(b *testing.B) {
	b.ReportAllocs()

	i := 0
	for b.Loop() {
		// Create unique path each time to force cache miss
		path := fmt.Sprintf("/api/site/site-%d/device/%d", i, i)
		_ = normalizePath(path)
		i++
	}
}

// BenchmarkNormalizePathMixed benchmarks realistic mixed workload.
// Simulates 80% cache hits (common endpoints) + 20% cache misses (unique IDs).
func BenchmarkNormalizePathMixed(b *testing.B) {
	commonPaths := []string{
		"/api/site/default/dns/record/507f1f77bcf86cd799439011",
		"/api/site/my-site/device/12345678",
		"/proxy/network/v2/api/site/default/setting/wan",
		"/api/v1/host/a1b2c3d4-e5f6-7890-abcd-ef1234567890/stats",
	}

	// Pre-warm cache with common paths
	for _, path := range commonPaths {
		_ = normalizePath(path)
	}

	b.ResetTimer()
	b.ReportAllocs()

	i := 0
	for b.Loop() {
		var path string
		// 80% cache hits (common paths), 20% cache misses (unique paths)
		if i%5 == 0 {
			path = fmt.Sprintf("/api/site/unique-%d/device/%d", i, i)
		} else {
			path = commonPaths[i%len(commonPaths)]
		}
		_ = normalizePath(path)
		i++
	}
}

// BenchmarkNormalizePathConcurrent benchmarks normalizePath under concurrent load.
// Tests the performance of sync.Map in concurrent scenarios.
func BenchmarkNormalizePathConcurrent(b *testing.B) {
	paths := []string{
		"/api/site/default/dns/record/507f1f77bcf86cd799439011",
		"/api/site/my-site/device/12345678",
		"/proxy/network/v2/api/site/default/setting/wan",
		"/api/v1/host/a1b2c3d4-e5f6-7890-abcd-ef1234567890/stats",
		"/api/site/production/client/100000",
	}

	// Pre-warm cache
	for _, path := range paths {
		_ = normalizePath(path)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			path := paths[i%len(paths)]
			_ = normalizePath(path)
			i++
		}
	})
}
