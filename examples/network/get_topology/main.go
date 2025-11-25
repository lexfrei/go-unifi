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

	// Get network topology
	topology, err := client.GetTopology(ctx, "default")
	if err != nil {
		log.Fatalf("Failed to get topology: %v", err)
	}

	fmt.Println("Network Topology:")
	fmt.Printf("  Vertices: %d\n", len(*topology.Vertices))
	fmt.Printf("  Edges: %d\n", len(*topology.Edges))

	if topology.Vertices != nil {
		fmt.Println("\nVertices (first 5):")
		for i, v := range *topology.Vertices {
			if i >= 5 {
				fmt.Println("  ...")
				break
			}
			vertexType := "<nil>"
			if v.Type != nil {
				vertexType = string(*v.Type)
			}
			fmt.Printf("  - MAC: %s, Type: %s, Name: %s\n",
				ptrStr(v.Mac), vertexType, ptrStr(v.Name))
		}
	}

	if topology.Edges != nil {
		fmt.Println("\nEdges (first 5):")
		for i, e := range *topology.Edges {
			if i >= 5 {
				fmt.Println("  ...")
				break
			}
			edgeType := "<nil>"
			if e.Type != nil {
				edgeType = string(*e.Type)
			}
			fmt.Printf("  - Uplink: %s -> Downlink: %s, Type: %s\n",
				ptrStr(e.UplinkMac), ptrStr(e.DownlinkMac), edgeType)
		}
	}
}

func ptrStr(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
