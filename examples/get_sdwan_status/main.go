package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lexfrei/go-unifi/api/sitemanager"
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
	client, err := sitemanager.New(apiKey)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Get SD-WAN configuration status
	fmt.Printf("Fetching SD-WAN status for config %s...\n", configID)
	status, err := client.GetSDWANConfigStatus(ctx, configID)
	if err != nil {
		log.Fatalf("Failed to get SD-WAN status: %v", err)
	}

	// Print response
	fmt.Printf("HTTP Status Code: %d\n", status.HttpStatusCode)
	fmt.Printf("Trace ID: %s\n", status.TraceId)
	fmt.Println()

	// Print status details
	fmt.Println("SD-WAN Status:")

	if status.Data.Id != nil {
		fmt.Printf("  Config ID: %s\n", *status.Data.Id)
	}

	if status.Data.Fingerprint != nil {
		fmt.Printf("  Fingerprint: %s\n", *status.Data.Fingerprint)
	}

	if status.Data.UpdatedAt != nil {
		updatedTime := time.Unix(*status.Data.UpdatedAt/1000, 0)
		fmt.Printf("  Updated At: %s\n", updatedTime.Format("2006-01-02 15:04:05"))
	}

	if status.Data.LastGeneratedAt != nil {
		generatedTime := time.Unix(*status.Data.LastGeneratedAt/1000, 0)
		fmt.Printf("  Last Generated: %s\n", generatedTime.Format("2006-01-02 15:04:05"))
	}

	if status.Data.GenerateStatus != nil {
		fmt.Printf("  Generate Status: %s\n", *status.Data.GenerateStatus)
	}

	if status.Data.Errors != nil && len(*status.Data.Errors) > 0 {
		fmt.Printf("  Errors: %d\n", len(*status.Data.Errors))
		for _, errMsg := range *status.Data.Errors {
			fmt.Printf("    - %s\n", errMsg)
		}
	}

	if status.Data.Warnings != nil && len(*status.Data.Warnings) > 0 {
		fmt.Printf("  Warnings: %d\n", len(*status.Data.Warnings))
		for _, warnMsg := range *status.Data.Warnings {
			fmt.Printf("    - %s\n", warnMsg)
		}
	}

	// Print hubs status
	if status.Data.Hubs != nil {
		fmt.Printf("\n  Hubs: %d\n", len(*status.Data.Hubs))
		for i, hub := range *status.Data.Hubs {
			fmt.Printf("\n    Hub #%d:\n", i+1)

			if hub.Name != nil {
				fmt.Printf("      Name: %s\n", *hub.Name)
			}

			if hub.SiteId != nil {
				fmt.Printf("      Site ID: %s\n", *hub.SiteId)
			}

			if hub.ApplyStatus != nil {
				fmt.Printf("      Apply Status: %s\n", *hub.ApplyStatus)
			}

			// Print WAN status
			if hub.PrimaryWanStatus != nil {
				fmt.Println("      Primary WAN:")
				if hub.PrimaryWanStatus.WanId != nil {
					fmt.Printf("        WAN ID: %s\n", *hub.PrimaryWanStatus.WanId)
				}
				if hub.PrimaryWanStatus.Ip != nil {
					fmt.Printf("        IP: %s\n", *hub.PrimaryWanStatus.Ip)
				}
				if hub.PrimaryWanStatus.Latency != nil {
					fmt.Printf("        Latency: %d ms\n", *hub.PrimaryWanStatus.Latency)
				}
				if hub.PrimaryWanStatus.InternetIssues != nil && len(*hub.PrimaryWanStatus.InternetIssues) > 0 {
					fmt.Printf("        Internet Issues: %v\n", *hub.PrimaryWanStatus.InternetIssues)
				}
			}

			if hub.Networks != nil && len(*hub.Networks) > 0 {
				fmt.Printf("      Networks: %d\n", len(*hub.Networks))
				for _, net := range *hub.Networks {
					if net.Name != nil {
						fmt.Printf("        - %s", *net.Name)
						if net.NetworkId != nil {
							fmt.Printf(" (%s)", *net.NetworkId)
						}
						fmt.Println()
					}
				}
			}

			if hub.Routes != nil && len(*hub.Routes) > 0 {
				fmt.Printf("      Routes: %d\n", len(*hub.Routes))
			}

			if hub.Errors != nil && len(*hub.Errors) > 0 {
				fmt.Printf("      Errors: %v\n", *hub.Errors)
			}

			if hub.Warnings != nil && len(*hub.Warnings) > 0 {
				fmt.Printf("      Warnings: %v\n", *hub.Warnings)
			}
		}
	}

	// Print spokes status
	if status.Data.Spokes != nil {
		fmt.Printf("\n  Spokes: %d\n", len(*status.Data.Spokes))
		for i, spoke := range *status.Data.Spokes {
			fmt.Printf("\n    Spoke #%d:\n", i+1)

			if spoke.Name != nil {
				fmt.Printf("      Name: %s\n", *spoke.Name)
			}

			if spoke.SiteId != nil {
				fmt.Printf("      Site ID: %s\n", *spoke.SiteId)
			}

			if spoke.ApplyStatus != nil {
				fmt.Printf("      Apply Status: %s\n", *spoke.ApplyStatus)
			}

			// Print WAN status
			if spoke.PrimaryWanStatus != nil {
				fmt.Println("      Primary WAN:")
				if spoke.PrimaryWanStatus.WanId != nil {
					fmt.Printf("        WAN ID: %s\n", *spoke.PrimaryWanStatus.WanId)
				}
				if spoke.PrimaryWanStatus.Ip != nil {
					fmt.Printf("        IP: %s\n", *spoke.PrimaryWanStatus.Ip)
				}
				if spoke.PrimaryWanStatus.Latency != nil {
					fmt.Printf("        Latency: %d ms\n", *spoke.PrimaryWanStatus.Latency)
				}
			}

			if spoke.Routes != nil && len(*spoke.Routes) > 0 {
				fmt.Printf("      Routes: %d\n", len(*spoke.Routes))
				for _, route := range *spoke.Routes {
					if route.RouteValue != nil {
						fmt.Printf("        - %s\n", *route.RouteValue)
					}
				}
			}

			// Print connections to hubs
			if spoke.Connections != nil && len(*spoke.Connections) > 0 {
				fmt.Printf("      Connections: %d\n", len(*spoke.Connections))
				for _, conn := range *spoke.Connections {
					if conn.HubId != nil {
						fmt.Printf("        Hub: %s\n", *conn.HubId)
					}
					if conn.Tunnels != nil && len(*conn.Tunnels) > 0 {
						fmt.Printf("        Tunnels: %d\n", len(*conn.Tunnels))
						for _, tunnel := range *conn.Tunnels {
							if tunnel.Status != nil {
								fmt.Printf("          Status: %s", *tunnel.Status)
								if tunnel.SpokeWanId != nil && tunnel.HubWanId != nil {
									fmt.Printf(" (%s -> %s)", *tunnel.SpokeWanId, *tunnel.HubWanId)
								}
								fmt.Println()
							}
						}
					}
				}
			}

			if spoke.Errors != nil && len(*spoke.Errors) > 0 {
				fmt.Printf("      Errors: %v\n", *spoke.Errors)
			}

			if spoke.Warnings != nil && len(*spoke.Warnings) > 0 {
				fmt.Printf("      Warnings: %v\n", *spoke.Warnings)
			}
		}
	}

	// Print full JSON if verbose flag is set
	if len(os.Args) > 1 && os.Args[1] == "-v" {
		fmt.Println("\n=== Full JSON Response ===")
		jsonData, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			log.Printf("Failed to marshal JSON: %v", err)
		} else {
			fmt.Println(string(jsonData))
		}
	}
}
