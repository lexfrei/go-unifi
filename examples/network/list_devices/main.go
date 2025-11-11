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

	// First, get the site ID
	fmt.Println("Fetching sites...")
	sites, err := client.ListSites(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to list sites: %v", err)
	}

	if len(sites.Data) == 0 {
		log.Fatal("No sites found")
	}

	siteID := sites.Data[0].Id
	siteName := sites.Data[0].Name
	fmt.Printf("Using site: %s (ID: %s)\n\n", siteName, siteID)

	// List devices for the site
	fmt.Println("Fetching devices...")
	devices, err := client.ListSiteDevices(ctx, siteID, nil)
	if err != nil {
		log.Fatalf("Failed to list devices: %v", err)
	}

	// Print response metadata
	fmt.Printf("Offset: %d\n", devices.Offset)
	fmt.Printf("Limit: %d\n", devices.Limit)
	fmt.Printf("Count: %d\n", devices.Count)
	fmt.Printf("Total Count: %d\n", devices.TotalCount)
	fmt.Println()

	// Print each device
	for i, device := range devices.Data {
		fmt.Printf("Device #%d:\n", i+1)
		fmt.Printf("  ID: %s\n", device.Id)
		fmt.Printf("  Name: %s\n", device.Name)
		fmt.Printf("  Model: %s\n", device.Model)
		fmt.Printf("  MAC: %s\n", device.MacAddress)
		fmt.Printf("  IP: %s\n", device.IpAddress)
		fmt.Printf("  State: %s\n", device.State)

		fmt.Println()
	}

	// Print full JSON if verbose flag is set
	if len(os.Args) > 1 && os.Args[1] == "-v" {
		fmt.Println("\n=== Full JSON Response ===")
		jsonData, err := json.MarshalIndent(devices, "", "  ")
		if err != nil {
			log.Printf("Failed to marshal JSON: %v", err)
		} else {
			fmt.Println(string(jsonData))
		}
	}
}
