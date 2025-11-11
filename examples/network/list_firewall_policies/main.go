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
	fmt.Printf("Note: Firewall API v2 uses internalReference, not UUID!\n\n")

	// List firewall policies
	fmt.Println("Fetching firewall policies...")
	policies, err := client.ListFirewallPolicies(ctx, network.Site(site))
	if err != nil {
		log.Fatalf("Failed to list firewall policies: %v", err)
	}

	fmt.Printf("Found %d firewall policy/policies\n\n", len(policies))

	if len(policies) == 0 {
		fmt.Println("No firewall policies found.")
		return
	}

	// Count policies by action
	actionCounts := make(map[string]int)
	for _, policy := range policies {
		actionCounts[string(policy.Action)]++
	}

	fmt.Println("Policies by action:")
	for action, count := range actionCounts {
		fmt.Printf("  %s: %d\n", action, count)
	}
	fmt.Println()

	// Print first 10 policies as examples
	displayLimit := 10
	if len(policies) < displayLimit {
		displayLimit = len(policies)
	}

	fmt.Printf("Showing first %d policies:\n\n", displayLimit)

	for i := 0; i < displayLimit; i++ {
		policy := policies[i]
		fmt.Printf("Policy #%d:\n", i+1)
		fmt.Printf("  ID: %s\n", policy.UnderscoreId)
		fmt.Printf("  Enabled: %t\n", policy.Enabled)
		fmt.Printf("  Action: %s\n", policy.Action)
		fmt.Printf("  Name: %s\n", policy.Name)

		if policy.Index != nil {
			fmt.Printf("  Index: %d\n", *policy.Index)
		}

		fmt.Println()
	}

	if len(policies) > displayLimit {
		fmt.Printf("... and %d more policies (use -v flag to see all)\n", len(policies)-displayLimit)
	}

	// Print full JSON if verbose flag is set
	if len(os.Args) > 1 && os.Args[1] == "-v" {
		fmt.Println("\n=== Full JSON Response ===")
		jsonData, err := json.MarshalIndent(policies, "", "  ")
		if err != nil {
			log.Printf("Failed to marshal JSON: %v", err)
		} else {
			fmt.Println(string(jsonData))
		}
	}
}
