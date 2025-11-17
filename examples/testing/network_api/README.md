# Testing Network API with Mocks

This example demonstrates how to test code that uses the Network API client using mock implementations.

## Overview

The `network.NetworkAPIClient` interface allows you to create mock implementations for testing without making actual API calls to your UniFi controller.

## Examples Included

### 1. testify/mock Example

See `example_testify_test.go` for a complete example using `github.com/stretchr/testify/mock`.

**Pros:**
- Simple to use
- No code generation required
- Great for small to medium projects

**Cons:**
- Requires manual implementation of all interface methods
- More boilerplate code

### 2. gomock Example

For gomock examples, see the `example_gomock_test.go` file.

**Pros:**
- Automatic mock generation
- Type-safe
- Less boilerplate

**Cons:**
- Requires code generation step
- Additional tooling

## Running the Examples

```bash
# Install dependencies
go get github.com/stretchr/testify/mock
go get go.uber.org/mock/gomock

# Run tests
go test -v
```

## Using in Your Project

### Option 1: testify/mock

```go
import (
    "context"
    "testing"
    "github.com/stretchr/testify/mock"
    "github.com/lexfrei/go-unifi/api/network"
)

type MockNetworkClient struct {
    mock.Mock
}

func (m *MockNetworkClient) ListDNSRecords(ctx context.Context, site network.Site) ([]network.DNSRecord, error) {
    args := m.Called(ctx, site)
    return args.Get(0).([]network.DNSRecord), args.Error(1)
}

// Implement other methods as needed...
```

### Option 2: gomock

```go
//go:generate mockgen -destination=mocks/network_client.go -package=mocks github.com/lexfrei/go-unifi/api/network NetworkAPIClient

import (
    "context"
    "testing"
    "go.uber.org/mock/gomock"
    "your-project/mocks"
)

func TestYourFunction(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockClient := mocks.NewMockNetworkAPIClient(ctrl)
    // Set expectations...
}
```

## Best Practices

1. **Accept interfaces, return structs** - Your functions should accept `network.NetworkAPIClient` interface, not `*network.APIClient` struct
2. **Mock only what you need** - You don't need to implement all 22 methods, only the ones your code uses
3. **Use table-driven tests** - Combine mocks with table-driven tests for comprehensive coverage
4. **Test error cases** - Don't forget to test how your code handles API errors

## Example Function to Test

```go
package myapp

import (
    "context"
    "fmt"
    "github.com/lexfrei/go-unifi/api/network"
)

type DNSManager struct {
    client network.NetworkAPIClient
}

func NewDNSManager(client network.NetworkAPIClient) *DNSManager {
    return &DNSManager{client: client}
}

func (dm *DNSManager) GetRecordCount(ctx context.Context, site string) (int, error) {
    records, err := dm.client.ListDNSRecords(ctx, network.Site(site))
    if err != nil {
        return 0, fmt.Errorf("failed to list DNS records: %w", err)
    }
    return len(records), nil
}
```

Now you can easily test `DNSManager` by injecting a mock client!
