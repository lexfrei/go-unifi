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
	// Get API key from environment variable
	apiKey := os.Getenv("UNIFI_API_KEY")
	if apiKey == "" {
		log.Fatal("UNIFI_API_KEY environment variable is required")
	}

	// Get host ID from command line or use default
	hostID := "942A6FCE26520000000008A62C8000000000091C92E70000000067801E31:392959371"
	if len(os.Args) > 1 {
		hostID = os.Args[1]
	}

	// Create client with default configuration
	client, err := unifi.NewUnifiClient(unifi.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Get host by ID
	fmt.Printf("Fetching host with ID: %s\n\n", hostID)
	host, err := client.GetHostByID(ctx, hostID)
	if err != nil {
		log.Fatalf("Failed to get host: %v", err)
	}

	// Print response metadata
	fmt.Printf("HTTP Status Code: %d\n", host.HttpStatusCode)
	fmt.Printf("Trace ID: %s\n\n", host.TraceId)

	// Print host details
	fmt.Println("=== Host Details ===")
	fmt.Printf("ID: %s\n", host.Data.Id)
	fmt.Printf("Hardware ID: %s\n", host.Data.HardwareId)
	fmt.Printf("Type: %s\n", host.Data.Type)

	if host.Data.IpAddress != nil {
		fmt.Printf("IP Address: %s\n", *host.Data.IpAddress)
	}

	if host.Data.Owner != nil {
		fmt.Printf("Owner: %t\n", *host.Data.Owner)
	}

	if host.Data.RegistrationTime != nil {
		fmt.Printf("Registration Time: %s\n", host.Data.RegistrationTime.Format("2006-01-02 15:04:05"))
	}

	if host.Data.IsBlocked != nil {
		fmt.Printf("Blocked: %t\n", *host.Data.IsBlocked)
	}

	// UserData and ReportedState are complex nested structures
	if host.Data.UserData != nil {
		fmt.Printf("User Data: present\n")
	}

	if host.Data.ReportedState != nil {
		fmt.Printf("Reported State: present\n")
	}

	// Print full JSON if verbose flag is set
	if len(os.Args) > 1 && (os.Args[1] == "-v" || (len(os.Args) > 2 && os.Args[2] == "-v")) {
		fmt.Println("\n=== Full JSON Response ===")
		jsonData, err := json.MarshalIndent(host, "", "  ")
		if err != nil {
			log.Printf("Failed to marshal JSON: %v", err)
		} else {
			fmt.Println(string(jsonData))
		}
	}
}
