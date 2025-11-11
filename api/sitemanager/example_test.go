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

func ExampleUnifiClient_ListSites() {
	// Create client
	client, _ := sitemanager.New("your-api-key")

	_ = client
	// sites, err := client.ListSites(context.Background())
	// Output:
}

func ExampleUnifiClient_ListSDWANConfigs() {
	// Early Access endpoint - client automatically uses EA rate limiter
	client, _ := sitemanager.New("your-api-key")

	_ = client
	// configs, err := client.ListSDWANConfigs(context.Background())
	// Output:
}
