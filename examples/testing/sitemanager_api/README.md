# Testing Site Manager API with Mocks

This example demonstrates how to test code that uses the Site Manager API client using mock implementations.

## Overview

The `sitemanager.SiteManagerAPIClient` interface allows you to create mock implementations for testing without making actual API calls to the UniFi cloud API.

## Examples Included

### testify/mock Example

See `example_testify_test.go` for a complete example using `github.com/stretchr/testify/mock`.

## Running the Examples

```bash
# Install dependencies
go get github.com/stretchr/testify/mock

# Run tests
go test -v
```

## Using in Your Project

```go
import (
    "context"
    "testing"
    "github.com/stretchr/testify/mock"
    "github.com/lexfrei/go-unifi/api/sitemanager"
)

type MockSiteManagerClient struct {
    mock.Mock
}

func (m *MockSiteManagerClient) ListHosts(ctx context.Context, params *sitemanager.ListHostsParams) (*sitemanager.HostsResponse, error) {
    args := m.Called(ctx, params)
    return args.Get(0).(*sitemanager.HostsResponse), args.Error(1)
}

// Implement other methods as needed...
```

## Best Practices

1. **Accept interfaces, return structs** - Your functions should accept `sitemanager.SiteManagerAPIClient` interface
2. **Mock only what you need** - You don't need to implement all 9 methods, only the ones your code uses
3. **Test both success and error cases** - Ensure your code handles API errors gracefully

## Example Function to Test

```go
package myapp

import (
    "context"
    "fmt"
    "github.com/lexfrei/go-unifi/api/sitemanager"
)

type HostMonitor struct {
    client sitemanager.SiteManagerAPIClient
}

func NewHostMonitor(client sitemanager.SiteManagerAPIClient) *HostMonitor {
    return &HostMonitor{client: client}
}

func (hm *HostMonitor) GetTotalHosts(ctx context.Context) (int, error) {
    resp, err := hm.client.ListHosts(ctx, nil)
    if err != nil {
        return 0, fmt.Errorf("failed to list hosts: %w", err)
    }
    return len(resp.Data), nil
}
```

Now you can easily test `HostMonitor` by injecting a mock client!
