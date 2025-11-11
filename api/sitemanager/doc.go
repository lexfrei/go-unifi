// Package sitemanager provides a Go client for UniFi Site Manager API v1.
//
// UniFi Site Manager API provides programmatic access to UniFi network infrastructure,
// allowing you to manage hosts, sites, devices, ISP metrics, and SD-WAN configurations.
//
// # Rate Limiting
//
// The client implements token bucket rate limiting:
//   - v1 endpoints: 10,000 requests per minute (V1RateLimit constant)
//   - Early Access endpoints: 100 requests per minute (EARateLimit constant)
//
// # Retry Logic
//
// Automatic exponential backoff retry for:
//   - 5xx server errors
//   - 429 rate limit errors (respects Retry-After header)
//
// # Example Usage
//
//	client, err := sitemanager.NewUnifiClient(sitemanager.ClientConfig{
//	    APIKey:             "your-api-key",
//	    RateLimitPerMinute: sitemanager.V1RateLimit,
//	})
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
//	    fmt.Printf("Host: %s\n", *host.Name)
//	}
package sitemanager
