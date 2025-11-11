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

	// Create vouchers
	count := 3
	duration := 480      // 8 hours
	quota := 1           // Single use
	note := "Test vouchers created by go-unifi example"
	qosOverwrite := true
	qosDown := 10000 // 10 Mbps
	qosUp := 5000    // 5 Mbps

	fmt.Printf("Creating %d hotspot vouchers...\n", count)
	fmt.Printf("  Duration: %d minutes (%v)\n", duration, time.Duration(duration)*time.Minute)
	fmt.Printf("  Quota: %d use(s)\n", quota)
	fmt.Printf("  Download limit: %d Kbps\n", qosDown)
	fmt.Printf("  Upload limit: %d Kbps\n", qosUp)
	fmt.Println()

	request := &network.CreateVouchersRequest{
		Count:           count,
		Duration:        &duration,
		Quota:           &quota,
		Note:            &note,
		QosOverwrite:    &qosOverwrite,
		QosRateMaxDown:  &qosDown,
		QosRateMaxUp:    &qosUp,
	}

	vouchers, err := client.CreateHotspotVouchers(ctx, siteID, request)
	if err != nil {
		log.Fatalf("Failed to create vouchers: %v", err)
	}

	// Print created vouchers
	fmt.Printf("Successfully created %d voucher(s):\n\n", len(vouchers.Data))

	for i, voucher := range vouchers.Data {
		fmt.Printf("Voucher #%d:\n", i+1)
		fmt.Printf("  ID: %s\n", voucher.UnderscoreId)
		fmt.Printf("  Code: %s\n", voucher.Code)

		if voucher.Duration != nil {
			duration := time.Duration(*voucher.Duration) * time.Minute
			fmt.Printf("  Duration: %v\n", duration)
		}

		if voucher.Quota != nil {
			fmt.Printf("  Quota: %d use(s)\n", *voucher.Quota)
		}

		if voucher.Status != nil {
			fmt.Printf("  Status: %s\n", *voucher.Status)
		}

		createTime := time.Unix(int64(voucher.CreateTime), 0)
		fmt.Printf("  Created: %s\n", createTime.Format("2006-01-02 15:04:05"))

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
