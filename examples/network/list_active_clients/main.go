package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lexfrei/go-unifi/api/network"
)

func main() {
	baseURL := os.Getenv("UNIFI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://172.16.0.1"
	}

	apiKey := os.Getenv("UNIFI_API_KEY")
	if apiKey == "" {
		log.Fatal("UNIFI_API_KEY environment variable is required")
	}

	client, err := network.New(baseURL, apiKey)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// List active clients
	clients, err := client.ListActiveClients(ctx, "default")
	if err != nil {
		log.Fatalf("Failed to list active clients: %v", err)
	}

	fmt.Printf("Active Clients: %d\n\n", len(clients))

	for i, c := range clients {
		if i >= 10 {
			fmt.Printf("... and %d more clients\n", len(clients)-10)
			break
		}
		fmt.Printf("Client %d:\n", i+1)
		fmt.Printf("  MAC: %s\n", ptrStr(c.Mac))
		fmt.Printf("  IP: %s\n", ptrStr(c.Ip))
		fmt.Printf("  Hostname: %s\n", ptrStr(c.Hostname))
		fmt.Printf("  Display Name: %s\n", ptrStr(c.DisplayName))
		if c.Type != nil {
			fmt.Printf("  Type: %s\n", string(*c.Type))
		}
		fmt.Printf("  Network: %s\n", ptrStr(c.NetworkName))
		if c.Essid != nil {
			fmt.Printf("  SSID: %s\n", *c.Essid)
		}
		if c.Signal != nil {
			fmt.Printf("  Signal: %d dBm\n", *c.Signal)
		}
		fmt.Println()
	}
}

func ptrStr(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
