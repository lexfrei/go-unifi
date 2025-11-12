# UniFi Network API Client

[![Go Reference](https://pkg.go.dev/badge/github.com/lexfrei/go-unifi/api/network.svg)](https://pkg.go.dev/github.com/lexfrei/go-unifi/api/network)
[![Go Report Card](https://goreportcard.com/badge/github.com/lexfrei/go-unifi)](https://goreportcard.com/report/github.com/lexfrei/go-unifi)
[![License](https://img.shields.io/github/license/lexfrei/go-unifi)](https://github.com/lexfrei/go-unifi/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/lexfrei/go-unifi)](https://github.com/lexfrei/go-unifi/blob/main/go.mod)

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

> **Note:** All methods have been tested against real UniFi Dream Router (UDR7) hardware. The API specification is derived from:
> - UniFi Network controller endpoint documentation (accessible from the controller UI)
> - Official UniFi Network API documentation
> - Reverse engineering of controller behavior
>
> When a method is present in the endpoint documentation, we trust its behavior. Types and tests represent actual API behavior observed during testing.

### Sites

| Method | Version | Description |
|--------|---------|-------------|
| `ListSites` | v1 | List all configured sites with metadata |

### Devices

| Method | Version | Description |
|--------|---------|-------------|
| `ListSiteDevices` | v1 | List all devices for a specific site |
| `GetDeviceByID` | v1 | Get detailed device information by ID |

### Clients

| Method | Version | Description |
|--------|---------|-------------|
| `ListSiteClients` | v1 | List all connected clients for a site |
| `GetClientByID` | v1 | Get detailed client information by ID |

### DNS Records

| Method | Version | Description |
|--------|---------|-------------|
| `ListDNSRecords` | v2 | List all static DNS records |
| `CreateDNSRecord` | v2 | Create a new DNS record |
| `UpdateDNSRecord` | v2 | Update existing DNS record |
| `DeleteDNSRecord` | v2 | Delete DNS record |

### Firewall Policies

| Method | Version | Description |
|--------|---------|-------------|
| `ListFirewallPolicies` | v2 | List all firewall policies |
| `CreateFirewallPolicy` | v2 | Create a new firewall policy |
| `UpdateFirewallPolicy` | v2 | Update existing firewall policy |
| `DeleteFirewallPolicy` | v2 | Delete firewall policy |

### Traffic Rules

| Method | Version | Description |
|--------|---------|-------------|
| `ListTrafficRules` | v2 | List all traffic routing rules |
| `CreateTrafficRule` | v2 | Create a new traffic rule |
| `UpdateTrafficRule` | v2 | Update existing traffic rule |
| `DeleteTrafficRule` | v2 | Delete traffic rule |

### Hotspot Vouchers

| Method | Version | Description |
|--------|---------|-------------|
| `ListHotspotVouchers` | v1 | List all guest portal vouchers |
| `CreateHotspotVouchers` | v1 | Create vouchers with custom limits |
| `GetHotspotVoucher` | v1 | Get voucher details by ID |
| `DeleteHotspotVoucher` | v1 | Delete voucher |

### Analytics

| Method | Version | Description |
|--------|---------|-------------|
| `GetAggregatedDashboard` | v2 | Get aggregated dashboard statistics |

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

## Testing

This library has been tested and validated against:

### Tested Targets

- **UniFi Dream Router (UDR7)** running:
  - UniFi OS **4.3.9** with Network Application **9.4.19**
- **VMs** running:
  - UniFi OS **4.3.6** with Network Application **9.4.19**
  - UniFi OS **4.3.6** with Network Application **9.5.21**

**Note:** API responses may vary depending on hardware models, firmware versions, and Network Application releases. The types and tests reflect the actual behavior observed on tested configurations.

## Related

- [UniFi Site Manager API Client](../sitemanager/) - Cloud-based multi-site management
- [Main Project](../../) - Full go-unifi library with all APIs

## API Documentation

- [Official UniFi Network API](https://developer.ui.com/network-api/unifi-network-api)
- [OpenAPI Specification](./openapi.yaml)
