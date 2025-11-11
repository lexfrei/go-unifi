// Package network provides a Go client for the UniFi Network Integration API.
//
// The Network Integration API allows programmatic access to UniFi network infrastructure
// running on local controllers. It provides endpoints for managing sites, devices, and clients.
//
// # API Access
//
// This API is accessed locally through your UniFi controller at the path:
//
//	https://<controller-ip>/proxy/network/integration/v1/
//
// # Authentication
//
// All requests require an API key generated from your UniFi controller:
//
//  1. Navigate to Settings > Control Plane > Integrations
//  2. Create a new API key
//  3. Use the key in the X-API-KEY header
//
// # Features
//
//   - Site management and listing
//   - Device inventory and monitoring (routers, switches, access points)
//   - Client tracking and access control (wired/wireless)
//   - Real-time status information
//   - Port and radio interface details
//
// # Basic Usage
//
//	package main
//
//	import (
//	    "context"
//	    "fmt"
//	    "log"
//
//	    "github.com/lexfrei/go-unifi/api/network"
//	)
//
//	func main() {
//	    // Create client
//	    client, err := network.New("https://192.168.1.1", "your-api-key")
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // List all sites
//	    sites, err := client.ListSites(context.Background(), nil)
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    for _, site := range sites.Data {
//	        fmt.Printf("Site: %s (ID: %s)\n", site.Name, site.Id)
//
//	        // List devices for this site
//	        devices, err := client.ListSiteDevices(context.Background(), site.Id, nil)
//	        if err != nil {
//	            log.Fatal(err)
//	        }
//
//	        for _, device := range devices.Data {
//	            fmt.Printf("  Device: %s (%s) - %s\n", device.Name, device.Model, device.State)
//	        }
//	    }
//	}
//
// # Advanced Configuration
//
// For custom rate limits, timeouts, or TLS settings:
//
//	client, err := network.NewWithConfig(&network.ClientConfig{
//	    ControllerURL:      "https://192.168.1.1",
//	    APIKey:             "your-api-key",
//	    InsecureSkipVerify: true,              // For self-signed certificates
//	    RateLimitPerMinute: 500,                // Custom rate limit
//	    MaxRetries:         5,                  // Custom retry count
//	    RetryWaitTime:      2 * time.Second,    // Custom retry wait
//	})
//
// # Error Handling
//
// The client uses github.com/cockroachdb/errors for enhanced error handling:
//
//	devices, err := client.ListSiteDevices(ctx, siteID, nil)
//	if err != nil {
//	    // Error already includes context and stack trace
//	    log.Printf("Failed to list devices: %+v", err)
//	    return
//	}
//
// # Rate Limiting
//
// The client automatically handles rate limiting with a default limit of 1000 requests/minute.
// Requests are throttled locally to prevent hitting API rate limits, and retried automatically
// if the API returns 429 (Too Many Requests).
//
// # Retry Logic
//
// Failed requests are automatically retried up to 3 times (configurable) with exponential backoff:
//
//   - Network errors (connection failures, timeouts)
//   - 5xx server errors
//   - 429 rate limit errors (respects Retry-After header)
//
// Client errors (4xx) are not retried.
//
// # TLS/SSL Certificates
//
// By default, TLS certificate verification is disabled to support self-signed certificates
// common in local UniFi deployments. For production use with valid certificates:
//
//	client, err := network.NewWithConfig(&network.ClientConfig{
//	    ControllerURL:      "https://unifi.example.com",
//	    APIKey:             "your-api-key",
//	    InsecureSkipVerify: false,  // Enable certificate verification
//	})
//
// # API Coverage
//
// Currently supported endpoints:
//
//   - GET /sites - List all sites
//   - GET /sites/{siteId}/devices - List devices for a site
//   - GET /sites/{siteId}/devices/{deviceId} - Get device details
//   - GET /sites/{siteId}/clients - List clients for a site
//   - GET /sites/{siteId}/clients/{clientId} - Get client details
//
// # Hardware Support
//
// Tested on:
//
//   - UniFi Dream Router (UDR7) running UniFi OS 4.3.87
//   - UniFi Network 9.0.0+
//
// # Related Packages
//
//   - github.com/lexfrei/go-unifi/api/sitemanager - Site Manager API (cloud-based)
//   - github.com/lexfrei/go-unifi/internal/ratelimit - Rate limiting utilities
//   - github.com/lexfrei/go-unifi/internal/retry - Retry logic utilities
package network
