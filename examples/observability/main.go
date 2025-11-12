package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lexfrei/go-unifi/api/sitemanager"
	"github.com/lexfrei/go-unifi/internal/observability"
)

// customLogger implements observability.Logger using standard library log package.
type customLogger struct {
	*log.Logger
}

func newCustomLogger() *customLogger {
	return &customLogger{
		Logger: log.New(os.Stdout, "[UniFi] ", log.LstdFlags|log.Lmsgprefix),
	}
}

func (l *customLogger) Debug(msg string, fields ...observability.Field) {
	l.logWithFields("DEBUG", msg, fields)
}

func (l *customLogger) Info(msg string, fields ...observability.Field) {
	l.logWithFields("INFO", msg, fields)
}

func (l *customLogger) Warn(msg string, fields ...observability.Field) {
	l.logWithFields("WARN", msg, fields)
}

func (l *customLogger) Error(msg string, fields ...observability.Field) {
	l.logWithFields("ERROR", msg, fields)
}

func (l *customLogger) With(fields ...observability.Field) observability.Logger {
	// For simplicity, return same logger. In real implementation, could store fields.
	return l
}

func (l *customLogger) logWithFields(level, msg string, fields []observability.Field) {
	fieldStr := ""
	for _, f := range fields {
		fieldStr += fmt.Sprintf(" %s=%v", f.Key, f.Value)
	}
	l.Printf("[%s] %s%s", level, msg, fieldStr)
}

// customMetricsRecorder implements observability.MetricsRecorder.
type customMetricsRecorder struct {
	requestCount int
	retryCount   int
	errorCount   int
}

func newCustomMetricsRecorder() *customMetricsRecorder {
	return &customMetricsRecorder{}
}

func (m *customMetricsRecorder) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration) {
	m.requestCount++
	fmt.Printf("[METRICS] HTTP Request: %s %s -> %d (took %v)\n", method, path, statusCode, duration)
}

func (m *customMetricsRecorder) RecordRetry(attemptNumber int, endpoint string) {
	m.retryCount++
	fmt.Printf("[METRICS] Retry #%d for endpoint: %s\n", attemptNumber, endpoint)
}

func (m *customMetricsRecorder) RecordRateLimit(endpoint string, waitDuration time.Duration) {
	fmt.Printf("[METRICS] Rate limited on %s, waited %v\n", endpoint, waitDuration)
}

func (m *customMetricsRecorder) RecordError(operation, errorType string) {
	m.errorCount++
	fmt.Printf("[METRICS] Error in %s: %s\n", operation, errorType)
}

func (m *customMetricsRecorder) PrintSummary() {
	fmt.Println("\n=== Metrics Summary ===")
	fmt.Printf("Total HTTP Requests: %d\n", m.requestCount)
	fmt.Printf("Total Retries: %d\n", m.retryCount)
	fmt.Printf("Total Errors: %d\n", m.errorCount)
}

func main() {
	// Get API key from environment variable
	apiKey := os.Getenv("UNIFI_API_KEY")
	if apiKey == "" {
		log.Fatal("UNIFI_API_KEY environment variable is required")
	}

	// Create custom logger and metrics recorder
	logger := newCustomLogger()
	metrics := newCustomMetricsRecorder()

	fmt.Println("Creating Site Manager client with custom observability...")
	fmt.Println("This example demonstrates how to integrate your own logging and metrics.\n")

	// Create client with custom observability
	client, err := sitemanager.NewWithConfig(&sitemanager.ClientConfig{
		APIKey:  apiKey,
		Logger:  logger,
		Metrics: metrics,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Make some API calls to generate observability events
	fmt.Println("Fetching hosts...")
	hosts, err := client.ListHosts(ctx, &sitemanager.ListHostsParams{})
	if err != nil {
		log.Fatalf("Failed to list hosts: %v", err)
	}

	fmt.Printf("\nReceived %d hosts (HTTP %d)\n\n", len(hosts.Data), hosts.HttpStatusCode)

	// Fetch sites
	fmt.Println("Fetching sites...")
	sites, err := client.ListSites(ctx)
	if err != nil {
		log.Fatalf("Failed to list sites: %v", err)
	}

	fmt.Printf("\nReceived %d sites (HTTP %d)\n\n", len(sites.Data), sites.HttpStatusCode)

	// Print metrics summary
	metrics.PrintSummary()

	fmt.Println("\n=== Observability Example Complete ===")
	fmt.Println("Check the logs above to see how custom logger and metrics recorder work.")
	fmt.Println("\nKey points:")
	fmt.Println("1. Custom logger receives all HTTP requests, responses, and errors")
	fmt.Println("2. Metrics recorder tracks request counts, retries, and errors")
	fmt.Println("3. You can integrate with any logging/metrics system (Prometheus, Datadog, etc.)")
	fmt.Println("4. Both logger and metrics are optional - defaults to no-op implementations")
}
