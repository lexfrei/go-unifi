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
	client, err := sitemanager.NewUnifiClient(sitemanager.ClientConfig{
		APIKey:             apiKey,
		RateLimitPerMinute: sitemanager.EARateLimit,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Get ISP metrics with 24h duration (5-minute intervals)
	fmt.Println("Fetching ISP metrics (5m intervals, last 24h)...")
	duration := sitemanager.GetISPMetricsParamsDuration("24h")
	metrics, err := client.GetISPMetrics(ctx, "5m", &sitemanager.GetISPMetricsParams{
		Duration: &duration,
	})
	if err != nil {
		log.Fatalf("Failed to get ISP metrics: %v", err)
	}

	// Print response summary
	fmt.Printf("HTTP Status Code: %d\n", metrics.HttpStatusCode)
	fmt.Printf("Trace ID: %s\n", metrics.TraceId)
	fmt.Printf("Number of metric items: %d\n", len(metrics.Data))
	fmt.Println()

	// Print each metric item
	for i, item := range metrics.Data {
		fmt.Printf("Metric Item #%d:\n", i+1)

		if item.MetricType != nil {
			fmt.Printf("  Metric Type: %s\n", *item.MetricType)
		}

		if item.HostId != nil {
			fmt.Printf("  Host ID: %s\n", *item.HostId)
		}

		if item.SiteId != nil {
			fmt.Printf("  Site ID: %s\n", *item.SiteId)
		}

		if item.Periods != nil {
			fmt.Printf("  Periods: %d entries\n", len(*item.Periods))

			// Show first and last period if available
			if len(*item.Periods) > 0 {
				firstPeriod := (*item.Periods)[0]
				fmt.Println("  First Period:")

				if firstPeriod.MetricTime != nil {
					fmt.Printf("    Time: %s\n", firstPeriod.MetricTime.Format("2006-01-02 15:04:05"))
				}

				if firstPeriod.Data != nil && firstPeriod.Data.Wan != nil {
					wan := firstPeriod.Data.Wan
					if wan.IspName != nil {
						fmt.Printf("    ISP: %s", *wan.IspName)
						if wan.IspAsn != nil {
							fmt.Printf(" (ASN: %s)", *wan.IspAsn)
						}
						fmt.Println()
					}
					if wan.DownloadKbps != nil {
						fmt.Printf("    Download: %d kbps\n", *wan.DownloadKbps)
					}
					if wan.UploadKbps != nil {
						fmt.Printf("    Upload: %d kbps\n", *wan.UploadKbps)
					}
					if wan.AvgLatency != nil {
						fmt.Printf("    Avg Latency: %d ms\n", *wan.AvgLatency)
					}
					if wan.MaxLatency != nil {
						fmt.Printf("    Max Latency: %d ms\n", *wan.MaxLatency)
					}
					if wan.PacketLoss != nil {
						fmt.Printf("    Packet Loss: %d%%\n", *wan.PacketLoss)
					}
					if wan.Uptime != nil {
						fmt.Printf("    Uptime: %d seconds\n", *wan.Uptime)
					}
					if wan.Downtime != nil {
						fmt.Printf("    Downtime: %d seconds\n", *wan.Downtime)
					}
				}

				if len(*item.Periods) > 1 {
					lastPeriod := (*item.Periods)[len(*item.Periods)-1]
					fmt.Println("  Last Period:")
					if lastPeriod.MetricTime != nil {
						fmt.Printf("    Time: %s\n", lastPeriod.MetricTime.Format("2006-01-02 15:04:05"))
					}
				}
			}
		}

		fmt.Println()
	}

	// Print full JSON if verbose flag is set
	if len(os.Args) > 1 && os.Args[1] == "-v" {
		fmt.Println("\n=== Full JSON Response ===")
		jsonData, err := json.MarshalIndent(metrics, "", "  ")
		if err != nil {
			log.Printf("Failed to marshal JSON: %v", err)
		} else {
			fmt.Println(string(jsonData))
		}
	}
}
