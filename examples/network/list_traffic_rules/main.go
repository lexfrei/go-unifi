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
	fmt.Printf("Note: Traffic Rules API v2 uses internalReference, not UUID!\n\n")

	// List traffic rules
	fmt.Println("Fetching traffic rules...")
	rules, err := client.ListTrafficRules(ctx, network.Site(site))
	if err != nil {
		log.Fatalf("Failed to list traffic rules: %v", err)
	}

	fmt.Printf("Found %d traffic rule(s)\n\n", len(rules))

	if len(rules) == 0 {
		fmt.Println("No traffic rules found.")
		return
	}

	// Print each rule
	for i, rule := range rules {
		fmt.Printf("Rule #%d:\n", i+1)
		fmt.Printf("  ID: %s\n", rule.UnderscoreId)
		fmt.Printf("  Enabled: %t\n", rule.Enabled)
		fmt.Printf("  Matching Target: %s\n", rule.MatchingTarget)

		if rule.Description != nil {
			fmt.Printf("  Description: %s\n", *rule.Description)
		}

		if rule.Action != nil {
			fmt.Printf("  Action: %s\n", *rule.Action)
		}

		if rule.BandwidthLimit != nil {
			fmt.Printf("  Bandwidth Limit: configured\n")
		}

		fmt.Println()
	}

	// Print full JSON if verbose flag is set
	if len(os.Args) > 1 && os.Args[1] == "-v" {
		fmt.Println("\n=== Full JSON Response ===")
		jsonData, err := json.MarshalIndent(rules, "", "  ")
		if err != nil {
			log.Printf("Failed to marshal JSON: %v", err)
		} else {
			fmt.Println(string(jsonData))
		}
	}
}
