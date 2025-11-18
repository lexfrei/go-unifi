package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/lexfrei/go-unifi/api/sitemanager"
)

func main() {
	// Get API key from environment variable
	apiKey := os.Getenv("UNIFI_API_KEY")
	if apiKey == "" {
		log.Fatal("UNIFI_API_KEY environment variable is required")
	}

	fmt.Println("=== go-unifi pprof Monitoring Example ===")
	fmt.Println()
	fmt.Println("This example demonstrates production-ready performance monitoring using pprof.")
	fmt.Println()

	// Start pprof HTTP server in background
	pprofAddr := ":6060"
	go func() {
		fmt.Printf("Starting pprof HTTP server on http://localhost%s/debug/pprof/\n", pprofAddr)
		fmt.Println()
		fmt.Println("Available endpoints:")
		fmt.Printf("  - CPU profile:    http://localhost%s/debug/pprof/profile?seconds=30\n", pprofAddr)
		fmt.Printf("  - Heap profile:   http://localhost%s/debug/pprof/heap\n", pprofAddr)
		fmt.Printf("  - Goroutines:     http://localhost%s/debug/pprof/goroutine\n", pprofAddr)
		fmt.Printf("  - Allocations:    http://localhost%s/debug/pprof/allocs\n", pprofAddr)
		fmt.Printf("  - Mutex contention: http://localhost%s/debug/pprof/mutex\n", pprofAddr)
		fmt.Printf("  - Full index:     http://localhost%s/debug/pprof/\n", pprofAddr)
		fmt.Println()

		if err := http.ListenAndServe(pprofAddr, nil); err != nil {
			log.Fatalf("pprof server failed: %v", err)
		}
	}()

	// Wait a bit for pprof server to start
	time.Sleep(100 * time.Millisecond)

	// Create client
	client, err := sitemanager.New(apiKey)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start continuous API polling to generate load
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	fmt.Println("Starting continuous API polling (Ctrl+C to stop)...")
	fmt.Println("Monitor performance using pprof endpoints above.")
	fmt.Println()
	fmt.Println("Example pprof commands:")
	fmt.Println("  # Interactive CPU profile (run for 30 seconds)")
	fmt.Printf("  go tool pprof http://localhost%s/debug/pprof/profile?seconds=30\n", pprofAddr)
	fmt.Println()
	fmt.Println("  # Heap profile analysis")
	fmt.Printf("  go tool pprof http://localhost%s/debug/pprof/heap\n", pprofAddr)
	fmt.Println()
	fmt.Println("  # Save profiles for later analysis")
	fmt.Printf("  curl http://localhost%s/debug/pprof/profile?seconds=30 > cpu.prof\n", pprofAddr)
	fmt.Printf("  curl http://localhost%s/debug/pprof/heap > heap.prof\n", pprofAddr)
	fmt.Println()
	fmt.Println("  # Analyze saved profiles")
	fmt.Println("  go tool pprof cpu.prof")
	fmt.Println("  go tool pprof heap.prof")
	fmt.Println()

	requestCount := 0
	startTime := time.Now()

	for {
		select {
		case <-ticker.C:
			requestCount++

			// Make API request
			hosts, err := client.ListHosts(ctx, &sitemanager.ListHostsParams{})
			if err != nil {
				log.Printf("Request #%d failed: %v", requestCount, err)
				continue
			}

			// Print runtime stats
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			fmt.Printf("[%s] Request #%d: %d hosts | Goroutines: %d | HeapAlloc: %.2f MB | NumGC: %d\n",
				time.Since(startTime).Round(time.Second),
				requestCount,
				len(hosts.Data),
				runtime.NumGoroutine(),
				float64(m.HeapAlloc)/1024/1024,
				m.NumGC,
			)

		case <-sigChan:
			fmt.Println("\n\nReceived shutdown signal, generating final profiles...")

			// Generate CPU profile
			cpuFile, err := os.Create("cpu_profile.prof")
			if err != nil {
				log.Printf("Could not create CPU profile: %v", err)
			} else {
				if err := pprof.StartCPUProfile(cpuFile); err != nil {
					log.Printf("Could not start CPU profile: %v", err)
				} else {
					// Run for 5 seconds
					time.Sleep(5 * time.Second)
					pprof.StopCPUProfile()
					cpuFile.Close()
					fmt.Println("✓ CPU profile saved to cpu_profile.prof")
				}
			}

			// Generate heap profile
			heapFile, err := os.Create("heap_profile.prof")
			if err != nil {
				log.Printf("Could not create heap profile: %v", err)
			} else {
				runtime.GC() // Get up-to-date statistics
				if err := pprof.WriteHeapProfile(heapFile); err != nil {
					log.Printf("Could not write heap profile: %v", err)
				}
				heapFile.Close()
				fmt.Println("✓ Heap profile saved to heap_profile.prof")
			}

			// Generate goroutine profile
			goroutineFile, err := os.Create("goroutine_profile.prof")
			if err != nil {
				log.Printf("Could not create goroutine profile: %v", err)
			} else {
				if err := pprof.Lookup("goroutine").WriteTo(goroutineFile, 0); err != nil {
					log.Printf("Could not write goroutine profile: %v", err)
				}
				goroutineFile.Close()
				fmt.Println("✓ Goroutine profile saved to goroutine_profile.prof")
			}

			fmt.Println()
			fmt.Println("=== Final Statistics ===")
			fmt.Printf("Total requests: %d\n", requestCount)
			fmt.Printf("Total runtime: %v\n", time.Since(startTime).Round(time.Second))
			fmt.Printf("Requests/second: %.2f\n", float64(requestCount)/time.Since(startTime).Seconds())

			var finalMem runtime.MemStats
			runtime.ReadMemStats(&finalMem)
			fmt.Printf("\nFinal memory stats:\n")
			fmt.Printf("  Heap Allocated: %.2f MB\n", float64(finalMem.HeapAlloc)/1024/1024)
			fmt.Printf("  Total Allocated: %.2f MB\n", float64(finalMem.TotalAlloc)/1024/1024)
			fmt.Printf("  System Memory: %.2f MB\n", float64(finalMem.Sys)/1024/1024)
			fmt.Printf("  GC Runs: %d\n", finalMem.NumGC)
			fmt.Printf("  Goroutines: %d\n", runtime.NumGoroutine())

			fmt.Println()
			fmt.Println("Analyze profiles with:")
			fmt.Println("  go tool pprof cpu_profile.prof")
			fmt.Println("  go tool pprof heap_profile.prof")
			fmt.Println("  go tool pprof goroutine_profile.prof")
			fmt.Println()
			fmt.Println("Use 'top', 'list <function>', 'web' commands in pprof interactive mode.")
			fmt.Println()

			return
		}
	}
}
