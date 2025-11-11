package sitemanager_test

import (
	"context"
	"log"
	"os"

	"github.com/lexfrei/go-unifi/api/sitemanager"
)

func ExampleNewUnifiClient() {
	client, err := sitemanager.NewUnifiClient(sitemanager.ClientConfig{
		APIKey:             os.Getenv("UNIFI_API_KEY"),
		RateLimitPerMinute: sitemanager.V1RateLimit,
	})
	if err != nil {
		log.Fatal(err)
	}

	_ = client // use client for API calls
	// Output:
}

func ExampleNewUnifiClient_earlyAccess() {
	// For Early Access endpoints, use EARateLimit (100 req/min)
	client, err := sitemanager.NewUnifiClient(sitemanager.ClientConfig{
		APIKey:             os.Getenv("UNIFI_API_KEY"),
		RateLimitPerMinute: sitemanager.EARateLimit,
	})
	if err != nil {
		log.Fatal(err)
	}

	_ = client // use client for EA endpoints
	// Output:
}

func ExampleUnifiClient_ListHosts() {
	client, _ := sitemanager.NewUnifiClient(sitemanager.ClientConfig{
		APIKey: os.Getenv("UNIFI_API_KEY"),
	})

	ctx := context.Background()
	_, err := client.ListHosts(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
}

func ExampleUnifiClient_GetHostByID() {
	client, _ := sitemanager.NewUnifiClient(sitemanager.ClientConfig{
		APIKey: os.Getenv("UNIFI_API_KEY"),
	})

	ctx := context.Background()
	hostID := "host-id-here"

	_, err := client.GetHostByID(ctx, hostID)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
}

func ExampleUnifiClient_ListSites() {
	client, _ := sitemanager.NewUnifiClient(sitemanager.ClientConfig{
		APIKey: os.Getenv("UNIFI_API_KEY"),
	})

	ctx := context.Background()
	_, err := client.ListSites(ctx)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
}

func ExampleUnifiClient_ListDevices() {
	client, _ := sitemanager.NewUnifiClient(sitemanager.ClientConfig{
		APIKey: os.Getenv("UNIFI_API_KEY"),
	})

	ctx := context.Background()
	_, err := client.ListDevices(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
}

func ExampleUnifiClient_GetISPMetrics() {
	client, _ := sitemanager.NewUnifiClient(sitemanager.ClientConfig{
		APIKey:             os.Getenv("UNIFI_API_KEY"),
		RateLimitPerMinute: sitemanager.EARateLimit, // EA endpoint
	})

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
	client, _ := sitemanager.NewUnifiClient(sitemanager.ClientConfig{
		APIKey:             os.Getenv("UNIFI_API_KEY"),
		RateLimitPerMinute: sitemanager.EARateLimit, // EA endpoint
	})

	ctx := context.Background()
	_, err := client.ListSDWANConfigs(ctx)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
}
