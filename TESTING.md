# Testing Guide

## Philosophy

**This project does NOT aim for 100% test coverage or follow strict TDD.**

Tests reflect real-world API behavior observed on actual UniFi hardware. We write tests only when we are confident in the expected behavior, not for theoretical scenarios or arbitrary coverage goals.

## Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Run Go linters
golangci-lint run ./...

# Run Markdown linters
markdownlint *.md **/*.md
```

## Testing Approach

- **Framework**: Standard Go `testing` package with `httptest` for mocks
- **Style**: Table-driven tests with `t.Parallel()`
- **Mock Responses**: Must match actual API responses exactly
- **Validation**: Test against real controllers first, then write mocks

## When to Write Tests

Write tests when:

- Adding new API endpoint support
- Fixing bugs discovered on real hardware
- Adding complex logic (rate limiting, retry logic, etc.)

Do NOT write tests for:

- Generated code (already validated by oapi-codegen)
- Simple wrappers with no logic
- Theoretical scenarios never observed on real hardware
- Arbitrary coverage percentage goals

Coverage is NOT a goal. It increases when we add features, decreases when we remove dead code. Both are acceptable.

## Integration Tests

Integration tests verify resource management and cleanup behavior, especially under stress conditions:

### Context Cancellation Tests

Located in `internal/middleware/retry_integration_test.go`, these tests verify:

- Response bodies are closed when context is canceled
- No goroutine leaks occur during request cancellation
- Timers are properly stopped on cancellation
- Resource cleanup works under concurrent stress

**Running integration tests:**

```bash
# Run all integration tests (may take longer)
go test ./internal/middleware/...

# Skip integration tests in short mode
go test -short ./...
```

**Stress tests:**

- `TestRetryStressTestConcurrentCancellations` - 500 concurrent requests with random cancellation
- `TestRetryNoGoroutineLeaks` - 100 requests checking for goroutine leaks

These tests verify the fixes for memory/resource leaks on context cancellation.

## Performance Testing

### Benchmarks

Run benchmarks to measure performance:

```bash
# Run all benchmarks
go test -bench=. -benchmem ./...

# Run specific benchmark
go test -bench=BenchmarkNormalizePath -benchmem ./internal/middleware/

# Compare before/after optimization
go test -bench=. -benchmem ./internal/middleware/ > new.txt
# Compare with: benchstat old.txt new.txt
```

### Race Detection

Always run tests with race detector before committing:

```bash
# Detect race conditions
go test -race ./...

# Race detection with coverage
go test -race -coverprofile=coverage.out ./...
```

### Memory Profiling

Use pprof for memory leak detection:

```bash
# Generate memory profile
go test -memprofile=mem.prof -bench=. ./internal/middleware/

# Analyze profile
go tool pprof mem.prof

# In pprof shell:
# top      - show top allocators
# list     - show source code
# web      - visualize (requires graphviz)
```

### CPU Profiling

Profile CPU usage:

```bash
# Generate CPU profile
go test -cpuprofile=cpu.prof -bench=. ./internal/middleware/

# Analyze profile
go tool pprof cpu.prof
```

See `examples/observability/pprof_monitoring/` for production profiling guide.
