package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/lexfrei/go-unifi/api/network"
)

func main() {
	// Get configuration from environment variables
	controllerURL := os.Getenv("UNIFI_TEST_CONTROLLER_URL")
	apiKey := os.Getenv("UNIFI_TEST_API_KEY")

	if controllerURL == "" || apiKey == "" {
		log.Fatal("UNIFI_TEST_CONTROLLER_URL and UNIFI_TEST_API_KEY environment variables are required")
	}

	// Create client
	client, err := network.New(controllerURL, apiKey)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// First, get the site internal reference (NOT UUID!)
	fmt.Println("Fetching sites...")
	sites, err := client.ListSites(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to list sites: %v", err)
	}

	if len(sites.Data) == 0 {
		log.Fatal("No sites found")
	}

	site := sites.Data[0].InternalReference
	siteName := sites.Data[0].Name
	fmt.Printf("Using site: %s (internal reference: %s)\n", siteName, site)
	fmt.Printf("Note: Analytics API v2 uses internalReference, not UUID!\n\n")

	// Get aggregated dashboard statistics (last 24 hours by default)
	historySeconds := 86400 // 24 hours
	params := &network.GetAggregatedDashboardParams{
		HistorySeconds: &historySeconds,
	}

	fmt.Printf("Fetching dashboard statistics (last %d hours)...\n\n", historySeconds/3600)
	dashboard, err := client.GetAggregatedDashboard(ctx, network.Site(site), params)
	if err != nil {
		log.Fatalf("Failed to get dashboard: %v", err)
	}

	// Print dashboard metadata
	if dashboard.DashboardMeta != nil {
		fmt.Println("=== Dashboard Metadata ===")
		if dashboard.DashboardMeta.Layout != nil {
			fmt.Printf("Layout: %s\n", *dashboard.DashboardMeta.Layout)
		}
		if dashboard.DashboardMeta.Widgets != nil {
			fmt.Printf("Enabled widgets: %d\n", len(*dashboard.DashboardMeta.Widgets))
		}
		fmt.Println()
	}

	// Print internet health
	if dashboard.Internet != nil && dashboard.Internet.HealthHistory != nil && len(*dashboard.Internet.HealthHistory) > 0 {
		fmt.Println("=== Internet Health (recent) ===")
		history := *dashboard.Internet.HealthHistory
		recentEvents := 5
		if len(history) < recentEvents {
			recentEvents = len(history)
		}

		for i := len(history) - recentEvents; i < len(history); i++ {
			event := history[i]
			if event.WanDowntime != nil && *event.WanDowntime {
				fmt.Println("  ⚠️  WAN downtime detected")
			}
			if event.HighLatency != nil && *event.HighLatency {
				fmt.Println("  ⚠️  High latency detected")
			}
			if event.PacketLoss != nil && *event.PacketLoss {
				fmt.Println("  ⚠️  Packet loss detected")
			}
		}
		fmt.Println()
	}

	// Print most active clients
	if dashboard.MostActiveClients != nil {
		fmt.Println("=== Most Active Clients ===")
		if dashboard.MostActiveClients.TotalBytes != nil {
			fmt.Printf("Total Bytes: %.2f GB\n", float64(*dashboard.MostActiveClients.TotalBytes)/(1024*1024*1024))
		}
		if dashboard.MostActiveClients.UsageByClient != nil && len(*dashboard.MostActiveClients.UsageByClient) > 0 {
			fmt.Printf("Number of clients tracked: %d\n", len(*dashboard.MostActiveClients.UsageByClient))
		}
		fmt.Println()
	}

	// Print most active APs
	if dashboard.MostActiveAps != nil {
		fmt.Println("=== Most Active Access Points ===")
		if dashboard.MostActiveAps.TotalBytes != nil {
			fmt.Printf("Total Bytes: %.2f GB\n", float64(*dashboard.MostActiveAps.TotalBytes)/(1024*1024*1024))
		}
		if dashboard.MostActiveAps.UsageByAp != nil && len(*dashboard.MostActiveAps.UsageByAp) > 0 {
			fmt.Printf("Number of APs tracked: %d\n", len(*dashboard.MostActiveAps.UsageByAp))
		}
		fmt.Println()
	}

	// Print full JSON if verbose flag is set
	if len(os.Args) > 1 && os.Args[1] == "-v" {
		fmt.Println("\n=== Full JSON Response ===")
		jsonData, err := json.MarshalIndent(dashboard, "", "  ")
		if err != nil {
			log.Printf("Failed to marshal JSON: %v", err)
		} else {
			fmt.Println(string(jsonData))
		}
	}
}
