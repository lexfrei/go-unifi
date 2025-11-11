package sitemanager_test

import (
	"github.com/lexfrei/go-unifi/api/sitemanager"
)

func ExampleNew() {
	client, _ := sitemanager.New("your-api-key")

	_ = client // use client for API calls
	// Output:
}

func ExampleNewWithConfig() {
	// For custom configuration (e.g., custom rate limits)
	client, _ := sitemanager.NewWithConfig(&sitemanager.ClientConfig{
		APIKey:               "your-api-key",
		V1RateLimitPerMinute: 5000, // Custom v1 limit
		EARateLimitPerMinute: 50,   // Custom EA limit
	})

	_ = client // use client with custom config
	// Output:
}

func ExampleUnifiClient_ListHosts() {
	// Create client
	client, _ := sitemanager.New("your-api-key")

	// List hosts with pagination
	pageSize := "10"
	params := &sitemanager.ListHostsParams{
		PageSize: &pageSize,
	}

	_ = client
	_ = params
	// hosts, err := client.ListHosts(context.Background(), params)
	// Output:
}

func ExampleUnifiClient_GetHostByID() {
	// Create client
	client, _ := sitemanager.New("your-api-key")

	hostID := "host-id-here"

	_ = client
	_ = hostID
	// host, err := client.GetHostByID(context.Background(), hostID)
	// Output:
}

func ExampleUnifiClient_ListSites() {
	// Create client
	client, _ := sitemanager.New("your-api-key")

	_ = client
	// sites, err := client.ListSites(context.Background())
	// Output:
}

func ExampleUnifiClient_ListDevices() {
	// Create client
	client, _ := sitemanager.New("your-api-key")

	_ = client
	// devices, err := client.ListDevices(context.Background(), nil)
	// Output:
}

func ExampleUnifiClient_GetISPMetrics() {
	// Early Access endpoint - client automatically uses EA rate limiter
	client, _ := sitemanager.New("your-api-key")

	duration := sitemanager.GetISPMetricsParamsDuration("24h")
	params := &sitemanager.GetISPMetricsParams{
		Duration: &duration,
	}

	_ = client
	_ = params
	// metrics, err := client.GetISPMetrics(context.Background(), "5m", params)
	// Output:
}

func ExampleUnifiClient_ListSDWANConfigs() {
	// Early Access endpoint - client automatically uses EA rate limiter
	client, _ := sitemanager.New("your-api-key")

	_ = client
	// configs, err := client.ListSDWANConfigs(context.Background())
	// Output:
}
