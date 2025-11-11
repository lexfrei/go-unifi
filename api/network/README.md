# UniFi Network API Client

Pure Go client for the UniFi Network API (Local Application API),
providing access to UniFi network devices, clients, and configurations
on local controllers.

## Features

- ✅ **Type-safe client** generated from OpenAPI specification
- ✅ **Rate limiting** with configurable limits (default: 1000 req/min)
- ✅ **Automatic retries** with exponential backoff
- ✅ **Self-signed certificates** support for local deployments
- ✅ **Context support** for all operations
- ✅ **Comprehensive API coverage** - Sites, Devices, Clients,
  Hotspot Vouchers, DNS, Firewall, Traffic Rules, Analytics
- ✅ **Detailed type definitions** with full schema validation

## Installation

```bash
go get github.com/lexfrei/go-unifi
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lexfrei/go-unifi/api/network"
)

func main() {
    // Create client (supports unifi.local mDNS hostname or IP)
    client, err := network.New("https://unifi.local", "your-api-key")
    if err != nil {
        log.Fatal(err)
    }

    // List all sites
    sites, err := client.ListSites(context.Background(), nil)
    if err != nil {
        log.Fatal(err)
    }

    // List devices for each site
    for _, site := range sites.Data {
        fmt.Printf("Site: %s\n", site.Name)

        devices, _ := client.ListSiteDevices(context.Background(), site.Id, nil)
        for _, device := range devices.Data {
            fmt.Printf("  - %s (%s): %s\n", device.Name, device.Model, device.State)
        }
    }
}
```

## API Coverage

- ✅ **Sites**
  - `ListSites(ctx, params)` - List all configured sites

- ✅ **Devices**
  - `ListSiteDevices(ctx, siteID, params)` - List devices for a site
  - `GetDeviceByID(ctx, siteID, deviceID)` - Get detailed device information

- ✅ **Clients**
  - `ListSiteClients(ctx, siteID, params)` - List clients for a site
  - `GetClientByID(ctx, siteID, clientID)` - Get client details

- ✅ **Hotspot Vouchers**
  - `ListHotspotVouchers(ctx, siteID, params)` - List guest vouchers
  - `CreateHotspotVouchers(ctx, siteID, request)` - Create vouchers with custom limits
  - `GetHotspotVoucher(ctx, siteID, voucherID)` - Get voucher details
  - `DeleteHotspotVoucher(ctx, siteID, voucherID)` - Delete voucher

- ✅ **DNS (v2 API)**
  - `ListDNSRecords(ctx, site)` - List static DNS records
  - `CreateDNSRecord(ctx, site, record)` - Create DNS record
  - `GetDNSRecord(ctx, site, recordID)` - Get DNS record details
  - `UpdateDNSRecord(ctx, site, recordID, record)` - Update DNS record
  - `DeleteDNSRecord(ctx, site, recordID)` - Delete DNS record

- ✅ **Firewall (v2 API)**
  - `ListFirewallPolicies(ctx, site)` - List firewall policies
  - `UpdateFirewallPolicy(ctx, site, policyID, policy)` - Update policy

- ✅ **Traffic Rules (v2 API)**
  - `ListTrafficRules(ctx, site)` - List traffic routing rules
  - `UpdateTrafficRule(ctx, site, ruleID, rule)` - Update traffic rule

- ✅ **Analytics (v2 API)**
  - `GetAggregatedDashboard(ctx, site, params)` - Get dashboard statistics

## Controller Access

UniFi controllers are accessible via:

- **mDNS hostname**: `https://unifi.local` (recommended)
- **Browser shortcut**: `unifi/` (UniFi Gateways only)
- **Custom hostname**: `https://<hostname>.local` (if mDNS enabled)
- **IP address**: `https://192.168.1.1`

## Configuration

### Simple (Recommended)

```go
client, err := network.New("https://unifi.local", "your-api-key")
```

### Custom Configuration

```go
client, err := network.NewWithConfig(&network.ClientConfig{
    ControllerURL:      "https://unifi.local",
    APIKey:             "your-api-key",
    InsecureSkipVerify: true,              // For self-signed certificates
    RateLimitPerMinute: 500,                // Custom rate limit
    MaxRetries:         5,                  // Custom retry count
    RetryWaitTime:      2 * time.Second,    // Custom retry wait
})
```

## Authentication

1. Open your UniFi Network controller
2. Navigate to **Settings > Control Plane > Integrations**
3. Click **Create API Key**
4. Give it a name (e.g., "go-unifi-client")
5. Copy the key and use it in your code

**Security Note:** API keys have Site Admin permissions. Keep them secure.

## Examples

See [examples/network/](../../examples/network/) for complete working examples.

## Development

### Generate Code from OpenAPI

```bash
cd api/network && oapi-codegen -config .oapi-codegen.yaml openapi.yaml
```

Or use go generate:

```bash
cd api/network && go generate
```

## Full Documentation

See [doc.go](./doc.go) for comprehensive package documentation including:

- Detailed usage examples
- Error handling best practices
- Rate limiting behavior
- Retry logic explanation
- TLS/SSL certificate handling

## Tested Hardware

- **UniFi Dream Router (UDR7)**
  - UniFi OS 4.3.87
  - Network Application 9.4.19

## Related

- [UniFi Site Manager API Client](../sitemanager/) - Cloud-based multi-site management
- [Main Project](../../) - Full go-unifi library with all APIs

## API Documentation

- [Official UniFi Network API](https://developer.ui.com/network-api/unifi-network-api)
- [OpenAPI Specification](./openapi.yaml)
