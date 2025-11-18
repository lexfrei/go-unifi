# Performance Guide

This document describes performance characteristics, optimizations, and monitoring for go-unifi.

## Performance Characteristics

### Middleware Architecture

All HTTP concerns are implemented as composable middleware with minimal overhead:

- **Auth middleware**: Single header addition, ~100ns overhead
- **Rate limiter**: Token bucket algorithm, no goroutines, ~1μs overhead
- **Retry middleware**: Exponential backoff, context-aware, ~10μs overhead
- **Observability**: Pluggable logging/metrics, ~5μs overhead with noop implementations
- **TLS middleware**: Certificate validation for self-signed certs

**Total middleware overhead**: ~20μs per request (negligible compared to network latency)

### Path Normalization Performance

The observability middleware normalizes HTTP paths for metrics (prevents unbounded cardinality):

**Optimization history:**

```
Original (4 separate regex):      55,819 ns/op   13,733 B/op   177 allocs/op
After precompiling patterns:      30,521 ns/op    1,762 B/op    57 allocs/op
After combined pattern (current):  6,258 ns/op      955 B/op    33 allocs/op
```

**Overall improvement: 8.9x faster, 14.4x less memory, 5.4x fewer allocations**

The normalization uses 2 regex passes:
1. Combined pattern for UUIDs, ObjectIDs, and numeric IDs
2. Site name pattern for `/site/{name}/` → `/site/:site/`

## Resource Management

### Memory Efficiency

- **Pointers for optional fields**: Zero allocations for omitted API response fields
- **No reflection in hot paths**: All type conversions are compile-time safe
- **Reusable HTTP client**: Single client instance with connection pooling
- **Response body cleanup**: All response bodies properly closed in all code paths

### Goroutine Management

The library uses **synchronous operations only** - no background goroutines:

- Rate limiter: Uses `rate.Limiter` (no goroutines)
- Retry logic: Synchronous with `time.NewTimer` (properly stopped on cancellation)
- No worker pools or background tasks

**Expected goroutine count**: 1 (main goroutine) + user's application goroutines

Any goroutine growth indicates a leak - verify with integration tests.

### Context Cancellation

All operations respect context cancellation:

- **Retry middleware**: Stops timers and closes response bodies on cancellation
- **Rate limiter**: Returns immediately on context done
- **HTTP requests**: Native context support via `http.Request.WithContext`

Context cancellation is **always clean** - no resource leaks, no goroutine leaks.

## Monitoring Production Performance

### Using pprof

See `examples/observability/pprof_monitoring/` for a complete production monitoring example.

**Quick setup:**

```go
import _ "net/http/pprof"

// In main() or init()
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

**Access endpoints:**

```bash
# CPU profile (30 seconds)
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Heap allocations
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine profile (detect leaks)
go tool pprof http://localhost:6060/debug/pprof/goroutine

# All available profiles
curl http://localhost:6060/debug/pprof/
```

### Custom Metrics

Implement `observability.MetricsRecorder` to integrate with your metrics system:

```go
type MetricsRecorder interface {
    RecordHTTPRequest(method, path string, statusCode int, duration time.Duration)
    RecordRetry(attempt int, endpoint string)
    RecordRateLimit(endpoint string, wait time.Duration)
    RecordError(operation, errorType string)
    RecordContextCancellation(operation string)
}
```

**Key metrics to monitor:**

1. **HTTP request duration** - Track p50, p95, p99 latencies
2. **Retry count** - High retry rate indicates API instability
3. **Rate limit wait time** - Indicates if you're hitting limits
4. **Error rate** - Network errors, HTTP 5xx, etc.
5. **Context cancellation count** - Track timeout/cancellation patterns

Example: See `examples/observability/main.go` for Prometheus-style integration.

### Runtime Monitoring

Monitor these runtime metrics for production health:

```go
var m runtime.MemStats
runtime.ReadMemStats(&m)

// Key metrics:
fmt.Printf("Goroutines: %d\n", runtime.NumGoroutine())
fmt.Printf("Heap Allocated: %.2f MB\n", float64(m.HeapAlloc)/1024/1024)
fmt.Printf("GC Runs: %d\n", m.NumGC)
```

**Healthy application indicators:**

- Goroutine count: Stable (not continuously growing)
- Heap allocation: Stable with GC sawtooth pattern
- GC frequency: Low (not thrashing)

**Warning signs:**

- Goroutine count growing → goroutine leak
- Heap allocation growing → memory leak
- High GC frequency → too many allocations

## Performance Best Practices

### Client Configuration

**Use reasonable timeouts:**

```go
client, err := sitemanager.NewWithConfig(&sitemanager.ClientConfig{
    APIKey:        "key",
    Timeout:       30 * time.Second,  // HTTP client timeout
    MaxRetries:    3,                 // Retry attempts
    RetryWaitTime: time.Second,       // Initial retry backoff
})
```

**Rate limiting:**

Site Manager API has different limits for v1 and EA endpoints:

```go
client, err := sitemanager.NewWithConfig(&sitemanager.ClientConfig{
    V1RateLimitPerMinute: 10000,  // v1 endpoints
    EARateLimitPerMinute: 100,    // Early Access endpoints
})
```

Network API (local controller):

```go
client, err := network.NewWithConfig(&network.ClientConfig{
    RateLimitPerMinute: 1000,  // Adjust based on controller capacity
})
```

### Context Usage

**Always use context with timeout:**

```go
// Good: request with timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

hosts, err := client.ListHosts(ctx, params)
```

```go
// Bad: unbounded context
hosts, err := client.ListHosts(context.Background(), params)
```

**Use context for cancellation:**

```go
ctx, cancel := context.WithCancel(context.Background())
go func() {
    <-stopChan
    cancel()  // Cancel all ongoing requests
}()

hosts, err := client.ListHosts(ctx, params)
```

### Pagination

**Paginate large result sets:**

```go
// Site Manager API pagination
params := &sitemanager.ListHostsParams{
    PageSize: sitemanager.PtrString("100"),
}

for {
    resp, err := client.ListHosts(ctx, params)
    if err != nil {
        return err
    }

    // Process resp.Data...

    if resp.NextToken == nil || *resp.NextToken == "" {
        break
    }
    params.NextToken = resp.NextToken
}
```

### Connection Pooling

The HTTP client automatically manages connection pooling. For high-throughput scenarios:

```go
// Customize HTTP transport if needed
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:     90 * time.Second,
}

// Pass to client via custom transport middleware
```

## Benchmarking

### Running Benchmarks

```bash
# Run all benchmarks
go test -bench=. -benchmem ./...

# Run specific benchmark
go test -bench=BenchmarkNormalizePath -benchmem ./internal/middleware/

# Save results for comparison
go test -bench=. -benchmem ./internal/middleware/ > new.txt
```

### Comparing Results

Use `benchstat` for statistical comparison:

```bash
# Install benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Compare two benchmark runs
benchstat old.txt new.txt
```

Example output:

```
name              old time/op    new time/op    delta
NormalizePath-12    55.8µs ± 2%     6.3µs ± 1%  -88.78%  (p=0.000 n=10+10)

name              old alloc/op   new alloc/op   delta
NormalizePath-12    13.7kB ± 0%     1.0kB ± 0%  -93.05%  (p=0.000 n=10+10)

name              old allocs/op  new allocs/op  delta
NormalizePath-12       177 ± 0%        33 ± 0%  -81.36%  (p=0.000 n=10+10)
```

### Memory Profiling

```bash
# Generate memory profile from benchmark
go test -bench=BenchmarkNormalizePath -memprofile=mem.prof ./internal/middleware/

# Analyze allocations
go tool pprof -alloc_space mem.prof

# In pprof shell:
# top           - show top allocators
# list <func>   - show source code
# web           - visualize call graph
```

### CPU Profiling

```bash
# Generate CPU profile from benchmark
go test -bench=. -cpuprofile=cpu.prof ./internal/middleware/

# Analyze CPU usage
go tool pprof cpu.prof

# In pprof shell:
# top           - show top CPU consumers
# web           - visualize call graph
# list <func>   - show source with samples
```

## Known Performance Characteristics

### Rate Limiting Overhead

Token bucket rate limiter adds ~1μs overhead per request:

- No goroutines (synchronous)
- No mutex contention under normal load
- Blocks on `Wait()` when limit exceeded

### Retry Overhead

Exponential backoff adds minimal overhead:

- First attempt: no overhead
- Retry attempts: exponential wait (1s, 2s, 4s, 8s, etc.)
- Context cancellation: immediate return with cleanup

### Observability Overhead

With noop implementations (default):
- Logger: ~0ns overhead (no-op)
- Metrics: ~100ns overhead (interface call)

With custom implementations:
- Structured logging: ~5μs per log entry
- Metrics recording: ~1μs per metric

**Recommendation**: Use custom logger/metrics in production, noop in high-performance paths.

## Troubleshooting Performance Issues

### High Memory Usage

**Symptoms:**
- Heap allocation continuously growing
- Frequent GC runs
- Out of memory errors

**Diagnosis:**

```bash
# Take heap snapshot
curl http://localhost:6060/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# In pprof:
top           # Show top allocators
list <func>   # View source
```

**Common causes:**
- Not closing response bodies
- Unbounded slice/map growth
- Large pagination without limits

### High CPU Usage

**Symptoms:**
- High CPU utilization
- Slow response times
- Increased latency

**Diagnosis:**

```bash
# Take CPU profile
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# In pprof:
top           # Show CPU hotspots
web           # Visualize call graph
```

**Common causes:**
- Inefficient regex (path normalization)
- JSON marshaling/unmarshaling
- Tight retry loops

### Goroutine Leaks

**Symptoms:**
- Goroutine count continuously growing
- Eventually hits system limits

**Diagnosis:**

```bash
# Take goroutine snapshot
curl http://localhost:6060/debug/pprof/goroutine > goroutine.prof
go tool pprof goroutine.prof

# In pprof:
top           # Show goroutine counts by source
traces        # Show stack traces
```

**Verification with tests:**

```bash
# Run integration tests (should pass)
go test ./internal/middleware/... -run TestNoGoroutineLeaks
```

**Note**: This library creates NO goroutines - any growth indicates external issue.

## Performance Regression Testing

**Continuous benchmarking workflow:**

```bash
# 1. Run baseline before changes
git checkout main
go test -bench=. -benchmem ./... > baseline.txt

# 2. Run benchmarks after changes
git checkout feature-branch
go test -bench=. -benchmem ./... > feature.txt

# 3. Compare results
benchstat baseline.txt feature.txt

# 4. Fail CI if significant regression (>20% slower)
```

Example CI integration: See GitHub Actions workflow.

## Further Reading

- [Go pprof documentation](https://pkg.go.dev/net/http/pprof)
- [Profiling Go Programs](https://go.dev/blog/pprof)
- [Go Memory Model](https://go.dev/ref/mem)
- [Diagnostics Guide](https://go.dev/doc/diagnostics)
- [Benchmarking Best Practices](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go)
