package sitemanager_test

import (
	"context"
	"log"
	"os"

	"github.com/lexfrei/go-unifi/api/sitemanager"
)

func ExampleNew() {
	client, err := sitemanager.New(os.Getenv("UNIFI_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	_ = client // use client for API calls
	// Output:
}

func ExampleNewWithConfig() {
	// For custom configuration (e.g., custom rate limits)
	client, err := sitemanager.NewWithConfig(&sitemanager.ClientConfig{
		APIKey:               os.Getenv("UNIFI_API_KEY"),
		V1RateLimitPerMinute: 5000, // Custom v1 limit
		EARateLimitPerMinute: 50,   // Custom EA limit
	})
	if err != nil {
		log.Fatal(err)
	}

	_ = client // use client with custom config
	// Output:
}

func ExampleUnifiClient_ListHosts() {
	client, _ := sitemanager.New(os.Getenv("UNIFI_API_KEY"))

	ctx := context.Background()
	_, err := client.ListHosts(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
}

func ExampleUnifiClient_GetHostByID() {
	client, _ := sitemanager.New(os.Getenv("UNIFI_API_KEY"))

	ctx := context.Background()
	hostID := "host-id-here"

	_, err := client.GetHostByID(ctx, hostID)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
}

func ExampleUnifiClient_ListSites() {
	client, _ := sitemanager.New(os.Getenv("UNIFI_API_KEY"))

	ctx := context.Background()
	_, err := client.ListSites(ctx)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
}

func ExampleUnifiClient_ListDevices() {
	client, _ := sitemanager.New(os.Getenv("UNIFI_API_KEY"))

	ctx := context.Background()
	_, err := client.ListDevices(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
}

func ExampleUnifiClient_GetISPMetrics() {
	// Early Access endpoint - client automatically uses EA rate limiter
	client, _ := sitemanager.New(os.Getenv("UNIFI_API_KEY"))

	ctx := context.Background()
	duration := sitemanager.GetISPMetricsParamsDuration("24h")

	_, err := client.GetISPMetrics(ctx, "5m", &sitemanager.GetISPMetricsParams{
		Duration: &duration,
	})
	if err != nil {
		log.Fatal(err)
	}
	// Output:
}

func ExampleUnifiClient_ListSDWANConfigs() {
	// Early Access endpoint - client automatically uses EA rate limiter
	client, _ := sitemanager.New(os.Getenv("UNIFI_API_KEY"))

	ctx := context.Background()
	_, err := client.ListSDWANConfigs(ctx)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
}
