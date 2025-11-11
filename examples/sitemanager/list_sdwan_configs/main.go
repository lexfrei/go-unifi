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

	// Create client with EA rate limit (100 req/min for Early Access endpoints)
	client, err := sitemanager.New(apiKey)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// List all SD-WAN configurations
	fmt.Println("Fetching SD-WAN configurations...")
	configs, err := client.ListSDWANConfigs(ctx)
	if err != nil {
		log.Fatalf("Failed to list SD-WAN configs: %v", err)
	}

	// Print response
	fmt.Printf("HTTP Status Code: %d\n", configs.HttpStatusCode)
	fmt.Printf("Trace ID: %s\n", configs.TraceId)
	fmt.Printf("Number of SD-WAN configs: %d\n", len(configs.Data))
	fmt.Println()

	// Print each config
	for i, cfg := range configs.Data {
		fmt.Printf("SD-WAN Config #%d:\n", i+1)

		if cfg.Id != nil {
			fmt.Printf("  ID: %s\n", *cfg.Id)
		}

		if cfg.Name != nil {
			fmt.Printf("  Name: %s\n", *cfg.Name)
		}

		if cfg.Type != nil {
			fmt.Printf("  Type: %s\n", *cfg.Type)
		}

		if cfg.Variant != nil {
			fmt.Printf("  Variant: %s\n", *cfg.Variant)
		}

		if cfg.Settings != nil {
			fmt.Printf("  Settings: present\n")
			if cfg.Settings.SpokeToHubTunnelsMode != nil {
				fmt.Printf("    Tunnels Mode: %s\n", *cfg.Settings.SpokeToHubTunnelsMode)
			}
			if cfg.Settings.SpokesIsolate != nil {
				fmt.Printf("    Spokes Isolate: %t\n", *cfg.Settings.SpokesIsolate)
			}
		}

		if cfg.Hubs != nil {
			fmt.Printf("  Hubs: %d\n", len(*cfg.Hubs))
		}

		if cfg.Spokes != nil {
			fmt.Printf("  Spokes: %d\n", len(*cfg.Spokes))
		}

		fmt.Println()
	}

	// Print full JSON if verbose flag is set
	if len(os.Args) > 1 && os.Args[1] == "-v" {
		fmt.Println("\n=== Full JSON Response ===")
		jsonData, err := json.MarshalIndent(configs, "", "  ")
		if err != nil {
			log.Printf("Failed to marshal JSON: %v", err)
		} else {
			fmt.Println(string(jsonData))
		}
	}
}
