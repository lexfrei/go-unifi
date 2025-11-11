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

	// Create client with default configuration
	client, err := network.New(controllerURL, apiKey)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// List all sites
	fmt.Println("Fetching sites...")
	sites, err := client.ListSites(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to list sites: %v", err)
	}

	// Print response metadata
	fmt.Printf("Offset: %d\n", sites.Offset)
	fmt.Printf("Limit: %d\n", sites.Limit)
	fmt.Printf("Count: %d\n", sites.Count)
	fmt.Printf("Total Count: %d\n", sites.TotalCount)
	fmt.Println()

	// Print each site
	for i, site := range sites.Data {
		fmt.Printf("Site #%d:\n", i+1)
		fmt.Printf("  ID (UUID): %s\n", site.Id)
		fmt.Printf("  Internal Reference: %s\n", site.InternalReference)
		fmt.Printf("  Name: %s\n", site.Name)
		fmt.Println()
		fmt.Printf("  Note: Use 'id' for Integration API v1 endpoints\n")
		fmt.Printf("        Use 'internalReference' for DNS/Firewall v2 endpoints\n")
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
}
