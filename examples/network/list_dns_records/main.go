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
	fmt.Printf("Note: DNS API v2 uses internalReference, not UUID!\n\n")

	// List DNS records
	fmt.Println("Fetching static DNS records...")
	records, err := client.ListDNSRecords(ctx, network.Site(site))
	if err != nil {
		log.Fatalf("Failed to list DNS records: %v", err)
	}

	fmt.Printf("Found %d DNS record(s)\n\n", len(records))

	if len(records) == 0 {
		fmt.Println("No DNS records found.")
		return
	}

	// Print each record
	for i, record := range records {
		fmt.Printf("Record #%d:\n", i+1)
		fmt.Printf("  ID: %s\n", record.UnderscoreId)
		fmt.Printf("  Enabled: %t\n", record.Enabled)
		fmt.Printf("  Hostname: %s\n", record.Key)
		fmt.Printf("  Type: %s\n", record.RecordType)
		fmt.Printf("  Value: %s\n", record.Value)

		if record.Ttl != nil {
			fmt.Printf("  TTL: %d seconds\n", *record.Ttl)
		}

		if record.Port != nil {
			fmt.Printf("  Port: %d\n", *record.Port)
		}

		if record.Priority != nil {
			fmt.Printf("  Priority: %d\n", *record.Priority)
		}

		if record.Weight != nil {
			fmt.Printf("  Weight: %d\n", *record.Weight)
		}

		fmt.Println()
	}

	// Print full JSON if verbose flag is set
	if len(os.Args) > 1 && os.Args[1] == "-v" {
		fmt.Println("\n=== Full JSON Response ===")
		jsonData, err := json.MarshalIndent(records, "", "  ")
		if err != nil {
			log.Printf("Failed to marshal JSON: %v", err)
		} else {
			fmt.Println(string(jsonData))
		}
	}
}
