package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/lexfrei/go-unifi/api/sitemanager"
)

func main() {
	// Get API key from environment variable
	apiKey := os.Getenv("UNIFI_API_KEY")
	if apiKey == "" {
		log.Fatal("UNIFI_API_KEY environment variable is required")
	}

	// Create client with default configuration
	client, err := sitemanager.New(apiKey)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// List all sites
	fmt.Println("Fetching sites...")
	sites, err := client.ListSites(ctx)
	if err != nil {
		log.Fatalf("Failed to list sites: %v", err)
	}

	// Print response
	fmt.Printf("HTTP Status Code: %d\n", sites.HttpStatusCode)
	fmt.Printf("Trace ID: %s\n", sites.TraceId)
	fmt.Printf("Number of sites: %d\n", len(sites.Data))
	fmt.Println()

	// Print each site
	for i, site := range sites.Data {
		fmt.Printf("Site #%d:\n", i+1)

		if site.SiteId != nil {
			fmt.Printf("  Site ID: %s\n", *site.SiteId)
		}

		if site.HostId != nil {
			fmt.Printf("  Host ID: %s\n", *site.HostId)
		}

		if site.Permission != nil {
			fmt.Printf("  Permission: %s\n", *site.Permission)
		}

		if site.IsOwner != nil {
			fmt.Printf("  Is Owner: %t\n", *site.IsOwner)
		}

		if site.Meta != nil {
			fmt.Printf("  Meta:\n")
			if site.Meta.Name != nil {
				fmt.Printf("    Name: %s\n", *site.Meta.Name)
			}
			if site.Meta.Desc != nil {
				fmt.Printf("    Description: %s\n", *site.Meta.Desc)
			}
			if site.Meta.Timezone != nil {
				fmt.Printf("    Timezone: %s\n", *site.Meta.Timezone)
			}
			if site.Meta.GatewayMac != nil {
				fmt.Printf("    Gateway MAC: %s\n", *site.Meta.GatewayMac)
			}
		}

		if site.Statistics != nil {
			fmt.Printf("  Statistics:\n")
			if site.Statistics.Counts != nil {
				if site.Statistics.Counts.TotalDevice != nil {
					fmt.Printf("    Total Devices: %d\n", *site.Statistics.Counts.TotalDevice)
				}
				if site.Statistics.Counts.WifiClient != nil {
					fmt.Printf("    WiFi Clients: %d\n", *site.Statistics.Counts.WifiClient)
				}
				if site.Statistics.Counts.WiredClient != nil {
					fmt.Printf("    Wired Clients: %d\n", *site.Statistics.Counts.WiredClient)
				}
			}
			if site.Statistics.Gateway != nil && site.Statistics.Gateway.Shortname != nil {
				fmt.Printf("    Gateway Model: %s\n", *site.Statistics.Gateway.Shortname)
			}
		}

		fmt.Println()
	}

	// Print full JSON if verbose flag is set
	if len(os.Args) > 1 && os.Args[1] == "-v" {
		fmt.Println("\n=== Full JSON Response ===")
		jsonData, err := json.MarshalIndent(sites, "", "  ")
		if err != nil {
			log.Printf("Failed to marshal JSON: %v", err)
		} else {
			fmt.Println(string(jsonData))
		}
	}

	// Test pagination if nextToken is present
	if sites.NextToken != nil && *sites.NextToken != "" {
		fmt.Printf("\nNext token available: %s\n", *sites.NextToken)
		fmt.Println("To fetch next page, use PageSize and NextToken parameters")
	}
}
