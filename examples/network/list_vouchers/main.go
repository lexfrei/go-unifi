package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

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

	// List hotspot vouchers
	fmt.Println("Fetching hotspot vouchers...")
	vouchers, err := client.ListHotspotVouchers(ctx, siteID, nil)
	if err != nil {
		log.Fatalf("Failed to list vouchers: %v", err)
	}

	// Print response metadata
	fmt.Printf("Offset: %d\n", vouchers.Offset)
	fmt.Printf("Limit: %d\n", vouchers.Limit)
	fmt.Printf("Count: %d\n", vouchers.Count)
	fmt.Printf("Total Count: %d\n", vouchers.TotalCount)
	fmt.Println()

	if len(vouchers.Data) == 0 {
		fmt.Println("No vouchers found. Create some with the create_vouchers example.")
		return
	}

	// Print each voucher
	for i, voucher := range vouchers.Data {
		fmt.Printf("Voucher #%d:\n", i+1)
		fmt.Printf("  ID: %s\n", voucher.UnderscoreId)
		fmt.Printf("  Code: %s\n", voucher.Code)

		if voucher.Duration != nil {
			duration := time.Duration(*voucher.Duration) * time.Minute
			fmt.Printf("  Duration: %v (%d minutes)\n", duration, *voucher.Duration)
		}

		if voucher.Quota != nil {
			fmt.Printf("  Quota (max uses): %d\n", *voucher.Quota)
		}

		if voucher.Used != nil {
			fmt.Printf("  Used: %d times\n", *voucher.Used)
		}

		if voucher.Status != nil {
			fmt.Printf("  Status: %s\n", *voucher.Status)
		}

		if voucher.Note != nil {
			fmt.Printf("  Note: %s\n", *voucher.Note)
		}

		createTime := time.Unix(int64(voucher.CreateTime), 0)
		fmt.Printf("  Created: %s\n", createTime.Format("2006-01-02 15:04:05"))

		if voucher.QosOverwrite != nil && *voucher.QosOverwrite {
			fmt.Printf("  QoS Limits:\n")
			if voucher.QosRateMaxDown != nil {
				fmt.Printf("    Download: %d Kbps\n", *voucher.QosRateMaxDown)
			}
			if voucher.QosRateMaxUp != nil {
				fmt.Printf("    Upload: %d Kbps\n", *voucher.QosRateMaxUp)
			}
		}

		fmt.Println()
	}

	// Print full JSON if verbose flag is set
	if len(os.Args) > 1 && os.Args[1] == "-v" {
		fmt.Println("\n=== Full JSON Response ===")
		jsonData, err := json.MarshalIndent(vouchers, "", "  ")
		if err != nil {
			log.Printf("Failed to marshal JSON: %v", err)
		} else {
			fmt.Println(string(jsonData))
		}
	}
}
