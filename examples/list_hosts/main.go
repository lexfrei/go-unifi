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

	// Create client with default configuration
	client, err := unifi.NewUnifiClient(unifi.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// List all hosts
	fmt.Println("Fetching hosts...")
	hosts, err := client.ListHosts(ctx, &unifi.ListHostsParams{})
	if err != nil {
		log.Fatalf("Failed to list hosts: %v", err)
	}

	// Print response
	fmt.Printf("HTTP Status Code: %d\n", hosts.HttpStatusCode)
	fmt.Printf("Trace ID: %s\n", hosts.TraceId)
	fmt.Printf("Number of hosts: %d\n", len(hosts.Data))
	fmt.Println()

	// Print each host
	for i, host := range hosts.Data {
		fmt.Printf("Host #%d:\n", i+1)
		fmt.Printf("  ID: %s\n", host.Id)
		fmt.Printf("  Hardware ID: %s\n", host.HardwareId)
		fmt.Printf("  Type: %s\n", host.Type)

		if host.IpAddress != nil {
			fmt.Printf("  IP Address: %s\n", *host.IpAddress)
		}

		if host.Owner != nil {
			fmt.Printf("  Owner: %t\n", *host.Owner)
		}

		if host.RegistrationTime != nil {
			fmt.Printf("  Registration Time: %s\n", host.RegistrationTime.Format("2006-01-02 15:04:05"))
		}

		if host.IsBlocked != nil {
			fmt.Printf("  Blocked: %t\n", *host.IsBlocked)
		}

		// UserData and ReportedState are complex nested structures
		// Access them directly from the JSON for detailed info
		if host.UserData != nil {
			fmt.Printf("  User Data: present\n")
		}

		if host.ReportedState != nil {
			fmt.Printf("  Reported State: present\n")
		}

		fmt.Println()
	}

	// Print full JSON if verbose flag is set
	if len(os.Args) > 1 && os.Args[1] == "-v" {
		fmt.Println("\n=== Full JSON Response ===")
		jsonData, err := json.MarshalIndent(hosts, "", "  ")
		if err != nil {
			log.Printf("Failed to marshal JSON: %v", err)
		} else {
			fmt.Println(string(jsonData))
		}
	}

	// Test pagination if nextToken is present
	if hosts.NextToken != nil && *hosts.NextToken != "" {
		fmt.Printf("\nNext token available: %s\n", *hosts.NextToken)
		fmt.Println("To fetch next page, use PageSize and NextToken parameters")
	}
}
