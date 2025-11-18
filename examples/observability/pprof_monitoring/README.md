# pprof Monitoring Example

This example demonstrates production-ready performance monitoring for go-unifi using Go's built-in pprof profiler.

## What This Example Shows

- HTTP endpoint for live profiling (`/debug/pprof/`)
- Continuous API polling to generate realistic load
- Real-time memory and goroutine statistics
- Automatic profile generation on shutdown (CPU, heap, goroutine)
- Integration with `go tool pprof` for analysis

## Prerequisites

- UniFi Site Manager API key
- Network access to UniFi cloud API

## Running the Example

```bash
export UNIFI_API_KEY="your-api-key-here"
go run main.go
```

The example will:
1. Start pprof HTTP server on `http://localhost:6060`
2. Poll UniFi API every 5 seconds
3. Display runtime statistics (memory, goroutines, GC)
4. Generate profile files on Ctrl+C shutdown

## Using pprof Endpoints

While the example is running, you can access various profiling endpoints:

### Interactive CPU Profile

```bash
# Collect 30 seconds of CPU profile and open interactive analysis
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

### Heap Memory Profile

```bash
# Analyze current heap allocations
go tool pprof http://localhost:6060/debug/pprof/heap
```

### Goroutine Profile

```bash
# Check for goroutine leaks
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

### Save Profiles for Later Analysis

```bash
# Save profiles to files
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
curl http://localhost:6060/debug/pprof/heap > heap.prof
curl http://localhost:6060/debug/pprof/goroutine > goroutine.prof

# Analyze saved profiles
go tool pprof cpu.prof
go tool pprof heap.prof
go tool pprof goroutine.prof
```

## pprof Interactive Commands

Once in the pprof shell, you can use these commands:

```bash
# Show top 10 functions by resource usage
top

# Show top 20
top20

# List source code for specific function
list normalizePath

# Generate web visualization (requires graphviz)
web

# Generate PNG image
png > profile.png

# Show cumulative usage
top -cum

# Filter by function name
top normalizePath

# Exit
quit
```

## Automated Profile Generation

Press Ctrl+C to trigger graceful shutdown. The example will automatically generate:

- `cpu_profile.prof` - 5-second CPU profile
- `heap_profile.prof` - Current heap allocations
- `goroutine_profile.prof` - Active goroutines

## Analyzing Performance Issues

### Checking for Memory Leaks

```bash
# Take heap snapshot
go tool pprof http://localhost:6060/debug/pprof/heap

# In pprof shell, find allocations:
top          # Show top memory allocators
list <func>  # View source of suspicious function
```

### Checking for Goroutine Leaks

```bash
# Take goroutine snapshot
go tool pprof http://localhost:6060/debug/pprof/goroutine

# In pprof shell:
top          # Should show stable count
traces       # Show goroutine stack traces
```

### Finding CPU Hotspots

```bash
# Collect CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# In pprof shell:
top          # Show functions using most CPU
list <func>  # View source of expensive function
web          # Visualize call graph (requires graphviz)
```

## Production Deployment

For production use:

1. **Restrict pprof endpoint access:**
   ```go
   // Only bind to localhost
   http.ListenAndServe("localhost:6060", nil)

   // Or use authentication middleware
   pprofMux := http.NewServeMux()
   pprofMux.HandleFunc("/debug/pprof/", pprof.Index)
   // Add auth wrapper
   ```

2. **Enable pprof conditionally:**
   ```go
   if os.Getenv("ENABLE_PPROF") == "true" {
       go func() {
           log.Println(http.ListenAndServe("localhost:6060", nil))
       }()
   }
   ```

3. **Use firewall rules** to restrict access to pprof port

4. **Monitor key metrics:**
   - `runtime.NumGoroutine()` - detect goroutine leaks
   - `runtime.MemStats.HeapAlloc` - monitor memory growth
   - `runtime.MemStats.NumGC` - track GC frequency

## What to Look For

### Healthy Application

- Stable goroutine count (no continuous growth)
- Stable heap allocation (GC keeps up with allocations)
- Low GC frequency (not thrashing)
- No blocking operations in hot paths

### Warning Signs

- Goroutine count growing continuously → goroutine leak
- Heap allocation growing continuously → memory leak
- High GC frequency → too many allocations
- Long GC pauses → heap too large

## Further Reading

- [Go pprof documentation](https://pkg.go.dev/net/http/pprof)
- [Profiling Go Programs](https://go.dev/blog/pprof)
- [Diagnostics Guide](https://go.dev/doc/diagnostics)
