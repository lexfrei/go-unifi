# go-unifi

Pure Go client library for UniFi Site Manager API v1.

## Features

- ✅ **Type-safe client** generated from OpenAPI specification
- ✅ **Dual rate limiting** (automatic: 10,000 req/min for v1, 100 req/min for EA)
- ✅ **Automatic retries** with exponential backoff for 5xx and 429 errors
- ✅ **Error handling** using `github.com/cockroachdb/errors`
- ✅ **Context support** for all operations
- ✅ **Detailed type definitions** for Hosts, Sites, Devices, ISP Metrics, and SD-WAN

## Tested Hardware

This library has been tested against:
- **UniFi Dream Router (UDR7)** running:
  - UniFi OS **4.3.87**
  - Network Application **9.4.19**

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

### Stable (v1)

- ✅ **Hosts**
  - `ListHosts(ctx, params)` - List all hosts with pagination
  - `GetHostByID(ctx, id)` - Get host details by ID

- ✅ **Sites**
  - `ListSites(ctx)` - List all sites

- ✅ **Devices**
  - `ListDevices(ctx)` - List all UniFi devices

### Early Access (EA)

- ⚠️ **ISP Metrics**
  - `GetISPMetrics(ctx, type)` - Get ISP metrics
  - `QueryISPMetrics(ctx, type, query)` - Query ISP metrics with filters

- ⚠️ **SD-WAN**
  - `ListSDWANConfigs(ctx)` - List SD-WAN configurations
  - `GetSDWANConfigByID(ctx, id)` - Get SD-WAN config by ID
  - `GetSDWANConfigStatus(ctx, id)` - Get SD-WAN config status

## Examples

See the [examples/](./examples/) directory for complete working examples:

- **[list_hosts](./examples/list_hosts/)** - List all hosts with pagination
- **[get_host](./examples/get_host/)** - Get detailed host information
- **[list_sites](./examples/list_sites/)** - List all sites with metadata and statistics
- **[list_devices](./examples/list_devices/)** - List all devices with typed access
- **[get_isp_metrics](./examples/get_isp_metrics/)** - Get ISP metrics with WAN performance data

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
go run ./cmd/test-reality -api-key your-key

# Verbose mode with full JSON samples
go run ./cmd/test-reality -api-key your-key -verbose
```

See [cmd/test-reality/README.md](./cmd/test-reality/README.md) for details.

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

### Coming Soon

- UniFi Protect API
- UniFi Access API
- UniFi Talk API

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
