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

	// List all devices
	fmt.Println("Fetching devices...")
	devices, err := client.ListDevices(ctx, &unifi.ListDevicesParams{})
	if err != nil {
		log.Fatalf("Failed to list devices: %v", err)
	}

	// Print response
	fmt.Printf("HTTP Status Code: %d\n", devices.HttpStatusCode)
	fmt.Printf("Trace ID: %s\n", devices.TraceId)
	fmt.Printf("Number of device hosts: %d\n", len(devices.Data))
	fmt.Println()

	// Print each device host
	for i, device := range devices.Data {
		fmt.Printf("Device Host #%d:\n", i+1)

		if device.HostId != nil {
			fmt.Printf("  Host ID: %s\n", *device.HostId)
		}

		if device.HostName != nil {
			fmt.Printf("  Host Name: %s\n", *device.HostName)
		}

		if device.UpdatedAt != nil {
			fmt.Printf("  Updated At: %s\n", device.UpdatedAt.Format("2006-01-02 15:04:05"))
		}

		if device.Devices != nil {
			fmt.Printf("  Devices: %d items\n", len(*device.Devices))

			// Print device details
			for j, dev := range *device.Devices {
				fmt.Printf("\n    Device #%d:\n", j+1)

				if dev.Name != nil {
					fmt.Printf("      Name: %s\n", *dev.Name)
				}
				if dev.Model != nil {
					fmt.Printf("      Model: %s\n", *dev.Model)
				}
				if dev.Mac != nil {
					fmt.Printf("      MAC: %s\n", *dev.Mac)
				}
				if dev.Ip != nil {
					fmt.Printf("      IP: %s\n", *dev.Ip)
				}
				if dev.Status != nil {
					fmt.Printf("      Status: %s\n", *dev.Status)
				}
				if dev.Version != nil {
					fmt.Printf("      Firmware: %s\n", *dev.Version)
				}
				if dev.FirmwareStatus != nil {
					fmt.Printf("      Firmware Status: %s\n", *dev.FirmwareStatus)
				}
				if dev.IsConsole != nil {
					fmt.Printf("      Is Console: %t\n", *dev.IsConsole)
				}
				if dev.ProductLine != nil {
					fmt.Printf("      Product Line: %s\n", *dev.ProductLine)
				}
				if dev.AdoptionTime != nil {
					fmt.Printf("      Adoption Time: %s\n", dev.AdoptionTime.Format("2006-01-02 15:04:05"))
				}
				if dev.StartupTime != nil {
					fmt.Printf("      Startup Time: %s\n", dev.StartupTime.Format("2006-01-02 15:04:05"))
				}
			}
		}

		if device.Uidb != nil {
			fmt.Printf("  UIDB Info: present\n")
			if device.Uidb.Images != nil {
				fmt.Printf("    Images: %d items\n", len(*device.Uidb.Images))
			}
		}

		fmt.Println()
	}

	// Print full JSON if verbose flag is set
	if len(os.Args) > 1 && os.Args[1] == "-vv" {
		fmt.Println("\n=== Full JSON Response ===")
		jsonData, err := json.MarshalIndent(devices, "", "  ")
		if err != nil {
			log.Printf("Failed to marshal JSON: %v", err)
		} else {
			fmt.Println(string(jsonData))
		}
	}

	// Test pagination if nextToken is present
	if devices.NextToken != nil && *devices.NextToken != "" {
		fmt.Printf("\nNext token available: %s\n", *devices.NextToken)
		fmt.Println("To fetch next page, use PageSize and NextToken parameters")
	}
}
