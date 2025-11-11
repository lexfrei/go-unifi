package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lexfrei/go-unifi"
)

func main() {
	// Get API key from environment variable
	apiKey := os.Getenv("UNIFI_API_KEY")
	if apiKey == "" {
		log.Fatal("UNIFI_API_KEY environment variable is required")
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

	// First, get list of sites to query
	fmt.Println("Fetching sites to build query...")
	sites, err := client.ListSites(ctx)
	if err != nil {
		log.Fatalf("Failed to list sites: %v", err)
	}

	if len(sites.Data) == 0 {
		log.Fatal("No sites found")
	}

	// Build query for the first site
	// Query last 2 hours of 5-minute metrics
	endTime := time.Now()
	beginTime := endTime.Add(-2 * time.Hour)

	querySites := []unifi.ISPMetricsQuerySiteItem{
		{
			HostId:         *sites.Data[0].HostId,
			SiteId:         *sites.Data[0].SiteId,
			BeginTimestamp: &beginTime,
			EndTimestamp:   &endTime,
		},
	}

	query := unifi.ISPMetricsQuery{
		Sites: &querySites,
	}

	// Query ISP metrics
	fmt.Printf("Querying ISP metrics for site %s (last 2 hours)...\n", *sites.Data[0].SiteId)
	metrics, err := client.QueryISPMetrics(ctx, "5m", query)
	if err != nil {
		log.Fatalf("Failed to query ISP metrics: %v", err)
	}

	// Print response summary
	fmt.Printf("HTTP Status Code: %d\n", metrics.HttpStatusCode)
	fmt.Printf("Trace ID: %s\n", metrics.TraceId)

	// Check for partial success
	if metrics.Data.Status != nil {
		fmt.Printf("Query Status: %s\n", *metrics.Data.Status)
	}
	if metrics.Data.Message != nil {
		fmt.Printf("Message: %s\n", *metrics.Data.Message)
	}

	if metrics.Data.Metrics != nil {
		fmt.Printf("Number of metric items: %d\n", len(*metrics.Data.Metrics))
		fmt.Println()

		// Print each metric item
		for i, item := range *metrics.Data.Metrics {
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
						if wan.PacketLoss != nil {
							fmt.Printf("    Packet Loss: %d%%\n", *wan.PacketLoss)
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
	} else {
		fmt.Println("No metrics returned")
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
