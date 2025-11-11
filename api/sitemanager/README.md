# UniFi Site Manager API Client

[![Go Reference](https://pkg.go.dev/badge/github.com/lexfrei/go-unifi/api/sitemanager.svg)](https://pkg.go.dev/github.com/lexfrei/go-unifi/api/sitemanager)
[![Go Report Card](https://goreportcard.com/badge/github.com/lexfrei/go-unifi)](https://goreportcard.com/report/github.com/lexfrei/go-unifi)
[![License](https://img.shields.io/github/license/lexfrei/go-unifi)](https://github.com/lexfrei/go-unifi/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/lexfrei/go-unifi)](https://github.com/lexfrei/go-unifi/blob/main/go.mod)

Pure Go client library for UniFi Site Manager API v1.

## Features

- ✅ **Type-safe client** generated from OpenAPI specification
- ✅ **Dual rate limiting** (automatic: 10,000 req/min for v1, 100 req/min for EA)
- ✅ **Automatic retries** with exponential backoff for 5xx and 429 errors
- ✅ **Error handling** using `github.com/cockroachdb/errors`
- ✅ **Context support** for all operations
- ✅ **Detailed type definitions** for Hosts, Sites, Devices, ISP Metrics, and SD-WAN

## Tested Hardware

This library has been tested and validated against:
- **UniFi Dream Router (UDR7)** running:
  - UniFi OS **4.3.87**
  - Network Application **9.4.19**
- **Official UniFi Site Manager API v1 Documentation**

**Note:** API responses may vary depending on hardware models, firmware versions, and UniFi OS releases. The types and tests reflect the actual behavior observed on tested configurations.

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

    "github.com/lexfrei/go-unifi/api/sitemanager"
)

func main() {
    // Create client with defaults
    client, err := sitemanager.New("your-api-key")
    if err != nil {
        log.Fatal(err)
    }

    // List all hosts
    hosts, err := client.ListHosts(context.Background(), nil)
    if err != nil {
        log.Fatal(err)
    }

    // Print hosts
    for _, host := range hosts.Data {
        fmt.Printf("Host: %s (%s)\n", host.Id, host.Type)
    }
}
```

## Configuration

### Simple (Recommended)

```go
// Most common use case - uses sensible defaults
client, err := sitemanager.New("your-api-key")
```

### Custom Configuration

```go
// For advanced use cases (custom timeouts, rate limits, etc.)
client, err := sitemanager.NewWithConfig(&sitemanager.ClientConfig{
    // Required: Your API key from sitemanager.ui.com
    APIKey: "your-api-key",

    // Optional: Custom base URL (defaults to https://api.ui.com)
    BaseURL: "https://api.ui.com",

    // Optional: Rate limits (defaults: v1=10000, EA=100 requests/minute)
    // The client automatically selects the appropriate limiter based on endpoint
    V1RateLimitPerMinute: 5000,  // Custom v1 rate limit
    EARateLimitPerMinute: 50,    // Custom EA rate limit

    // Optional: Maximum number of retries (defaults to 3)
    MaxRetries: 3,

    // Optional: Wait time between retries (defaults to 1s)
    RetryWaitTime: time.Second,

    // Optional: HTTP client timeout (defaults to 30s)
    Timeout: 30 * time.Second,

    // Optional: Custom HTTP client
    HTTPClient: &http.Client{},
})
```

## API Coverage

> **Note:** All methods have been tested against real UniFi Dream Router (UDR7) hardware and validated against official UniFi Site Manager API documentation. Types and tests represent actual API behavior.

### Hosts

| Method | Version | Description |
|--------|---------|-------------|
| `ListHosts` | v1 | List all hosts with pagination support |
| `GetHostByID` | v1 | Get detailed host information by ID |

### Sites

| Method | Version | Description |
|--------|---------|-------------|
| `ListSites` | v1 | List all sites with metadata |

### Devices

| Method | Version | Description |
|--------|---------|-------------|
| `ListDevices` | v1 | List all UniFi devices across sites |

### ISP Metrics (Early Access)

| Method | Version | Description |
|--------|---------|-------------|
| `GetISPMetrics` | EA | Get ISP metrics for specified metric type |
| `QueryISPMetrics` | EA | Query ISP metrics with filters and time ranges |

### SD-WAN (Early Access)

| Method | Version | Description |
|--------|---------|-------------|
| `ListSDWANConfigs` | EA | List all SD-WAN configurations |
| `GetSDWANConfigByID` | EA | Get SD-WAN configuration details by ID |
| `GetSDWANConfigStatus` | EA | Get SD-WAN configuration status and health |

## Examples

See the [examples/](../../examples/sitemanager/) directory for complete working examples:

- **[list_hosts](../../examples/sitemanager/list_hosts/)** - List all hosts with pagination
- **[get_host](../../examples/sitemanager/get_host/)** - Get detailed host information
- **[list_sites](../../examples/sitemanager/list_sites/)** - List all sites with metadata and statistics
- **[list_devices](../../examples/sitemanager/list_devices/)** - List all devices with typed access
- **[get_isp_metrics](../../examples/sitemanager/get_isp_metrics/)** - Get ISP metrics with WAN performance data
- **[query_isp_metrics](../../examples/sitemanager/query_isp_metrics/)** - Query ISP metrics with time ranges and filters
- **[list_sdwan_configs](../../examples/sitemanager/list_sdwan_configs/)** - List all SD-WAN configurations
- **[get_sdwan_config](../../examples/sitemanager/get_sdwan_config/)** - Get SD-WAN configuration details
- **[get_sdwan_status](../../examples/sitemanager/get_sdwan_status/)** - Get SD-WAN configuration status

### List Hosts with Pagination

```go
params := &sitemanager.ListHostsParams{
    PageSize:  sitemanager.PtrString("10"),
    NextToken: sitemanager.PtrString("token-from-previous-response"),
}

hosts, err := client.ListHosts(ctx, params)
if err != nil {
    log.Fatal(err)
}

// Check if more pages available
if hosts.NextToken != nil && *hosts.NextToken != "" {
    fmt.Printf("Next page token: %s\n", *hosts.NextToken)
}
```

### Get Host Details

```go
host, err := client.GetHostByID(ctx, "host-id-here")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Host Type: %s\n", host.Data.Type)
fmt.Printf("IP Address: %s\n", *host.Data.IpAddress)
```

### Error Handling

The library uses `github.com/cockroachdb/errors` for enhanced error handling:

```go
hosts, err := client.ListHosts(ctx, params)
if err != nil {
    // Errors include full stack traces
    fmt.Printf("Error: %+v\n", err)
    return
}
```

## Rate Limiting

The client automatically manages separate rate limiters for different endpoint types:

- **v1 endpoints**: 10,000 requests/minute (automatic)
- **Early Access endpoints** (/api/ea/*): 100 requests/minute (automatic)
- **Automatic rate limiter selection** based on request URL
- **Client-side rate limiting** prevents exceeding API limits
- **Automatic retries** for 429 (Too Many Requests) responses
- **Respects Retry-After header** from server

No manual configuration needed - the client handles rate limiting transparently.

## Retry Logic

Automatic retries for:

- **Network errors** (connection failures, timeouts)
- **5xx server errors**
- **429 rate limit errors**

Retry strategy:

- Exponential backoff
- Configurable max retries (default: 3)
- Configurable wait time (default: 1s)

## Development

### Generate Code from OpenAPI

```bash
cd api/sitemanager && oapi-codegen -config .oapi-codegen.yaml openapi.yaml
```

Or use go generate:

```bash
cd api/sitemanager && go generate
```

### Test Against Reality

Validate types against real API responses to find `any` usage and type mismatches:

```bash
# Run directly
go run github.com/lexfrei/go-unifi/cmd/test-reality@latest -api-key your-key

# Or install globally
go install github.com/lexfrei/go-unifi/cmd/test-reality@latest
test-reality -api-key your-key

# Verbose mode with full JSON samples
test-reality -api-key your-key -verbose
```

See [cmd/test-reality/README.md](../../cmd/test-reality/README.md) for details.

### Run Linters

```bash
golangci-lint run ./...
```

### Run Tests

```bash
go test ./...
```

### Build Examples

```bash
cd examples/list_hosts && go build
UNIFI_API_KEY=your-key ./list_hosts
```

## Project Structure

```
go-unifi/
├── api/
│   └── sitemanager/        # UniFi Site Manager API v1 client
├── internal/               # Shared infrastructure (rate limiting, retry logic)
├── examples/               # Runnable examples
```

This structure follows [golang-standards/project-layout](https://github.com/golang-standards/project-layout).

## Supported APIs

### UniFi Site Manager API v1

`import "github.com/lexfrei/go-unifi/api/sitemanager"`

- ✅ Hosts, Sites, Devices management
- ✅ ISP Metrics (GET and POST query)
- ✅ SD-WAN configuration and status

## API Documentation

- [Official UniFi Site Manager API Documentation](https://developer.ui.com/site-manager-api/gettingstarted)
- [OpenAPI Specification](./api/sitemanager/openapi.yaml)

## Authentication

1. Go to [sitemanager.ui.com](https://sitemanager.ui.com)
2. Navigate to **API** section
3. Click **Create API Key**
4. Store the key securely (it displays only once)
5. Pass the key to the client configuration

## Rate Limits

- **v1 endpoints**: 10,000 requests per minute
- **Early Access endpoints**: 100 requests per minute

Exceeding limits returns `429 Too Many Requests` with a `Retry-After` header.

## License

BSD-3-Clause - see [LICENSE](./LICENSE) file for details

## Maintainer

### Aleksei Sviridkin

- Email: <f@lex.la>
- GPG: F57F 85FC 7975 F22B BC3F 2504 9C17 3EB1 B531 AA1F
