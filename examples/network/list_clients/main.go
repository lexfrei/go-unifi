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

	// List clients for the site
	fmt.Println("Fetching clients...")
	clients, err := client.ListSiteClients(ctx, siteID, nil)
	if err != nil {
		log.Fatalf("Failed to list clients: %v", err)
	}

	// Print response metadata
	fmt.Printf("Offset: %d\n", clients.Offset)
	fmt.Printf("Limit: %d\n", clients.Limit)
	fmt.Printf("Count: %d\n", clients.Count)
	fmt.Printf("Total Count: %d\n", clients.TotalCount)
	fmt.Println()

	// Print each client
	for i, client := range clients.Data {
		fmt.Printf("Client #%d:\n", i+1)
		fmt.Printf("  ID: %s\n", client.Id)
		fmt.Printf("  Name: %s\n", client.Name)
		fmt.Printf("  MAC: %s\n", client.MacAddress)
		fmt.Printf("  IP: %s\n", client.IpAddress)
		fmt.Printf("  Connection Type: %s\n", client.Type)
		fmt.Printf("  Connected At: %s\n", client.ConnectedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Access Type: %s\n", client.Access.Type)

		fmt.Println()
	}

	// Print full JSON if verbose flag is set
	if len(os.Args) > 1 && os.Args[1] == "-v" {
		fmt.Println("\n=== Full JSON Response ===")
		jsonData, err := json.MarshalIndent(clients, "", "  ")
		if err != nil {
			log.Printf("Failed to marshal JSON: %v", err)
		} else {
			fmt.Println(string(jsonData))
		}
	}
}
