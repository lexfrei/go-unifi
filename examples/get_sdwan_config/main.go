package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/lexfrei/go-unifi"
)

func main() {
	// Get API key and config ID from environment variables
	apiKey := os.Getenv("UNIFI_API_KEY")
	if apiKey == "" {
		log.Fatal("UNIFI_API_KEY environment variable is required")
	}

	configID := os.Getenv("SDWAN_CONFIG_ID")
	if configID == "" {
		log.Fatal("SDWAN_CONFIG_ID environment variable is required (e.g., b344034f-2636-478c-8c7a-e3350f8ed37a)")
	}

	// Create client with EA rate limit (100 req/min for Early Access endpoints)
	client, err := unifi.NewUnifiClient(unifi.ClientConfig{
		APIKey:             apiKey,
		RateLimitPerMinute: unifi.EARateLimit,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Get SD-WAN configuration by ID
	fmt.Printf("Fetching SD-WAN config %s...\n", configID)
	config, err := client.GetSDWANConfigByID(ctx, configID)
	if err != nil {
		log.Fatalf("Failed to get SD-WAN config: %v", err)
	}

	// Print response
	fmt.Printf("HTTP Status Code: %d\n", config.HttpStatusCode)
	fmt.Printf("Trace ID: %s\n", config.TraceId)
	fmt.Println()

	// Print config details
	fmt.Println("SD-WAN Configuration:")

	if config.Data.Id != nil {
		fmt.Printf("  ID: %s\n", *config.Data.Id)
	}

	if config.Data.Name != nil {
		fmt.Printf("  Name: %s\n", *config.Data.Name)
	}

	if config.Data.Type != nil {
		fmt.Printf("  Type: %s\n", *config.Data.Type)
	}

	if config.Data.Variant != nil {
		fmt.Printf("  Variant: %s\n", *config.Data.Variant)
	}

	// Print settings
	if config.Data.Settings != nil {
		fmt.Println("\n  Settings:")

		if config.Data.Settings.HubsInterconnect != nil {
			fmt.Printf("    Hubs Interconnect: %v\n", *config.Data.Settings.HubsInterconnect)
		}

		if config.Data.Settings.SpokeToHubTunnelsMode != nil {
			fmt.Printf("    Tunnels Mode: %s\n", *config.Data.Settings.SpokeToHubTunnelsMode)
		}

		if config.Data.Settings.SpokesAutoScaleAndNatEnabled != nil {
			fmt.Printf("    Auto Scale & NAT: %t\n", *config.Data.Settings.SpokesAutoScaleAndNatEnabled)
		}

		if config.Data.Settings.SpokesAutoScaleAndNatRange != nil {
			fmt.Printf("    Auto Scale Range: %s\n", *config.Data.Settings.SpokesAutoScaleAndNatRange)
		}

		if config.Data.Settings.SpokesIsolate != nil {
			fmt.Printf("    Spokes Isolate: %t\n", *config.Data.Settings.SpokesIsolate)
		}

		if config.Data.Settings.SpokeStandardSettingsEnabled != nil {
			fmt.Printf("    Standard Settings: %t\n", *config.Data.Settings.SpokeStandardSettingsEnabled)
		}

		if config.Data.Settings.SpokeToHubRouting != nil {
			fmt.Printf("    Spoke to Hub Routing: %s\n", *config.Data.Settings.SpokeToHubRouting)
		}
	}

	// Print hubs
	if config.Data.Hubs != nil {
		fmt.Printf("\n  Hubs: %d\n", len(*config.Data.Hubs))
		for i, hub := range *config.Data.Hubs {
			fmt.Printf("\n    Hub #%d:\n", i+1)

			if hub.Id != nil {
				fmt.Printf("      ID: %s\n", *hub.Id)
			}

			if hub.HostId != nil {
				fmt.Printf("      Host ID: %s\n", *hub.HostId)
			}

			if hub.SiteId != nil {
				fmt.Printf("      Site ID: %s\n", *hub.SiteId)
			}

			if hub.NetworkIds != nil && len(*hub.NetworkIds) > 0 {
				fmt.Printf("      Networks: %d\n", len(*hub.NetworkIds))
				for _, netID := range *hub.NetworkIds {
					fmt.Printf("        - %s\n", netID)
				}
			}

			if hub.Routes != nil && len(*hub.Routes) > 0 {
				fmt.Printf("      Routes: %d\n", len(*hub.Routes))
				for _, route := range *hub.Routes {
					fmt.Printf("        - %s\n", route)
				}
			}

			if hub.PrimaryWan != nil {
				fmt.Printf("      Primary WAN: %s\n", *hub.PrimaryWan)
			}

			if hub.WanFailover != nil {
				fmt.Printf("      WAN Failover: %t\n", *hub.WanFailover)
			}
		}
	}

	// Print spokes
	if config.Data.Spokes != nil {
		fmt.Printf("\n  Spokes: %d\n", len(*config.Data.Spokes))
		for i, spoke := range *config.Data.Spokes {
			fmt.Printf("\n    Spoke #%d:\n", i+1)

			if spoke.Id != nil {
				fmt.Printf("      ID: %s\n", *spoke.Id)
			}

			if spoke.HostId != nil {
				fmt.Printf("      Host ID: %s\n", *spoke.HostId)
			}

			if spoke.SiteId != nil {
				fmt.Printf("      Site ID: %s\n", *spoke.SiteId)
			}

			if spoke.NetworkIds != nil && len(*spoke.NetworkIds) > 0 {
				fmt.Printf("      Networks: %d\n", len(*spoke.NetworkIds))
				for _, netID := range *spoke.NetworkIds {
					fmt.Printf("        - %s\n", netID)
				}
			}

			if spoke.Routes != nil && len(*spoke.Routes) > 0 {
				fmt.Printf("      Routes: %d\n", len(*spoke.Routes))
				for _, route := range *spoke.Routes {
					fmt.Printf("        - %s\n", route)
				}
			}

			if spoke.PrimaryWan != nil {
				fmt.Printf("      Primary WAN: %s\n", *spoke.PrimaryWan)
			}

			if spoke.WanFailover != nil {
				fmt.Printf("      WAN Failover: %t\n", *spoke.WanFailover)
			}

			if spoke.HubsPriority != nil {
				fmt.Printf("      Hubs Priority: %v\n", *spoke.HubsPriority)
			}
		}
	}

	// Print full JSON if verbose flag is set
	if len(os.Args) > 1 && os.Args[1] == "-v" {
		fmt.Println("\n=== Full JSON Response ===")
		jsonData, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			log.Printf("Failed to marshal JSON: %v", err)
		} else {
			fmt.Println(string(jsonData))
		}
	}
}
