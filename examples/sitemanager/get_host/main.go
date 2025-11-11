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
	apiKey := os.Getenv("UNIFI_API_KEY")
	if apiKey == "" {
		log.Fatal("UNIFI_API_KEY environment variable is required")
	}

	hostID := "942A6FCE26520000000008A62C8000000000091C92E70000000067801E31:392959371"
	if len(os.Args) > 1 {
		hostID = os.Args[1]
	}

	client, err := sitemanager.New(apiKey)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	fmt.Printf("Fetching host with ID: %s\n\n", hostID)
	host, err := client.GetHostByID(ctx, hostID)
	if err != nil {
		log.Fatalf("Failed to get host: %v", err)
	}

	fmt.Printf("HTTP Status Code: %d\n", host.HttpStatusCode)
	fmt.Printf("Trace ID: %s\n\n", host.TraceId)

	fmt.Println("=== Host Details ===")
	fmt.Printf("ID: %s\n", host.Data.Id)
	fmt.Printf("Hardware ID: %s\n", host.Data.HardwareId)
	fmt.Printf("Type: %s\n", host.Data.Type)

	if host.Data.IpAddress != nil {
		fmt.Printf("IP Address: %s\n", *host.Data.IpAddress)
	}

	if host.Data.ReportedState != nil && host.Data.ReportedState.Hostname != nil {
		fmt.Printf("Hostname: %s\n", *host.Data.ReportedState.Hostname)
	}

	if len(os.Args) > 1 && (os.Args[1] == "-v" || (len(os.Args) > 2 && os.Args[2] == "-v")) {
		fmt.Println("\n=== Full JSON Response ===")
		jsonData, _ := json.MarshalIndent(host, "", "  ")
		fmt.Println(string(jsonData))
	}
}
