# go-unifi

Pure Go client library for UniFi Site Manager API v1.

## Features

- ✅ **Type-safe client** generated from OpenAPI specification
- ✅ **Rate limiting** (10,000 req/min for v1, 100 req/min for EA)
- ✅ **Automatic retries** with exponential backoff for 5xx and 429 errors
- ✅ **Error handling** using `github.com/cockroachdb/errors`
- ✅ **Context support** for all operations
- ✅ **Detailed type definitions** for Hosts, Sites, Devices, ISP Metrics, and SD-WAN

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

    "github.com/lexfrei/go-unifi"
)

func main() {
    // Create client
    client, err := unifi.NewUnifiClient(unifi.ClientConfig{
        APIKey: "your-api-key-here",
    })
    if err != nil {
        log.Fatal(err)
    }

    // List all hosts
    hosts, err := client.ListHosts(context.Background(), &unifi.ListHostsParams{})
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

```go
client, err := unifi.NewUnifiClient(unifi.ClientConfig{
    // Required: Your API key from unifi.ui.com
    APIKey: "your-api-key",

    // Optional: Custom base URL (defaults to https://api.ui.com)
    BaseURL: "https://api.ui.com",

    // Optional: Rate limit per minute (defaults to 10000 for v1)
    RateLimitPerMinute: 10000,

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

### List Hosts with Pagination

```go
params := &unifi.ListHostsParams{
    PageSize:  unifi.PtrString("10"),
    NextToken: unifi.PtrString("token-from-previous-response"),
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

The client automatically handles rate limiting:

- **Client-side rate limiter** prevents exceeding API limits
- **Automatic retries** for 429 (Too Many Requests) responses
- **Respects Retry-After header** from server

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

### Generate Client Code

```bash
make generate
```

### Run Linters

```bash
make lint
```

### Run Tests

```bash
make test
```

### Build Example

```bash
go build -o list-hosts examples/list_hosts/main.go
UNIFI_API_KEY=your-key ./list-hosts
```

## API Documentation

- [Official UniFi Site Manager API Documentation](https://developer.ui.com/site-manager-api/gettingstarted)
- [OpenAPI Specification](./openapi.yaml)

## Authentication

1. Go to [unifi.ui.com](https://unifi.ui.com)
2. Navigate to **API** section
3. Click **Create API Key**
4. Store the key securely (it displays only once)
5. Pass the key to the client configuration

## Rate Limits

- **v1 endpoints**: 10,000 requests per minute
- **Early Access endpoints**: 100 requests per minute

Exceeding limits returns `429 Too Many Requests` with a `Retry-After` header.

## License

MIT

## Maintainer

**Aleksei Sviridkin**
- Email: f@lex.la
- GPG: F57F 85FC 7975 F22B BC3F 2504 9C17 3EB1 B531 AA1F
