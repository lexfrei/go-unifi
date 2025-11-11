// Package sitemanager provides a Go client for UniFi Site Manager API v1.
//
// UniFi Site Manager API provides programmatic access to UniFi network infrastructure,
// allowing you to manage hosts, sites, devices, ISP metrics, and SD-WAN configurations.
//
// # Rate Limiting
//
// The client automatically manages separate rate limiters for different endpoint types:
//   - v1 endpoints: 10,000 requests per minute
//   - Early Access endpoints (paths starting with /api/ea/): 100 requests per minute
//
// Rate limiter selection is automatic based on request URL - no manual configuration needed.
//
// # Retry Logic
//
// Automatic exponential backoff retry for:
//   - 5xx server errors
//   - 429 rate limit errors (respects Retry-After header)
//
// # Example Usage
//
//	// Simple: create client with defaults
//	client, err := sitemanager.New("your-api-key")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	hosts, err := client.ListHosts(context.Background(), nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, host := range hosts.Data {
//	    fmt.Printf("Host: %s (%s)\n", host.Id, host.Type)
//	}
//
// # Custom Configuration
//
// For custom rate limits or other settings:
//
//	client, err := sitemanager.NewWithConfig(&sitemanager.ClientConfig{
//	    APIKey:               "your-api-key",
//	    V1RateLimitPerMinute: 5000,  // Custom v1 rate limit
//	    EARateLimitPerMinute: 50,    // Custom EA rate limit
//	})
package sitemanager
