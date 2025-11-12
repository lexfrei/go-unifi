# Testing Strategy

This document describes the testing philosophy and practices for the go-unifi project.

## Philosophy

**go-unifi does NOT aim for 100% test coverage or follow strict TDD (Test-Driven Development).**

Our tests serve a specific purpose: **they reflect the real-world behavior of UniFi APIs as we have observed them**. Tests are written based on actual API responses from real UniFi hardware, not based on assumptions or theoretical behavior.

### Key Principles

1. **Reality-Based Testing**: Tests represent actual API behavior observed on real UniFi controllers (UDR7, etc.)
2. **Pragmatic Coverage**: We test what matters - API interactions, error handling, rate limiting, retry logic
3. **No Theoretical Tests**: If we haven't seen it happen on real hardware, we don't test for it
4. **Living Documentation**: Tests serve as examples of how the API actually behaves

## Testing Approach

### Technology Stack

- **Framework**: Standard Go `testing` package (`go test`)
- **Mocking**: `net/http/httptest` for HTTP mocks
- **Table-Driven Tests**: Standard Go pattern for test organization
- **Parallel Execution**: Tests run with `t.Parallel()` where applicable

### What We Test

#### ✅ API Client Behavior
- Request construction (URLs, headers, parameters)
- Response parsing and type mapping
- Error handling for various HTTP status codes
- Rate limiting behavior
- Retry logic with exponential backoff

#### ✅ Real API Responses
- Type correctness based on actual API responses
- Field presence and nullability
- Enum values observed in practice
- Status code behavior (e.g., DELETE returns 200, not 204)

#### ❌ What We Don't Test
- Internal implementation details
- Third-party library behavior (oapi-codegen, rate limiters)
- 100% code coverage for the sake of coverage
- Hypothetical edge cases never observed on real hardware

## Running Tests

### All Tests

```bash
go test ./...
```

### Specific Package

```bash
# Site Manager API
go test ./api/sitemanager/

# Network API
go test ./api/network/

# Internal packages
go test ./internal/...
```

### With Coverage

```bash
go test ./... -cover
```

### Verbose Output

```bash
go test ./... -v
```

### Parallel Execution

Tests run in parallel by default where safe:

```bash
# Use specific number of parallel tests
go test ./... -parallel 4
```

## Mock Strategy

### HTTP Mocking

We use `httptest.NewServer()` to create mock HTTP servers that return realistic API responses:

```go
func TestListHosts(t *testing.T) {
    tests := []struct {
        name           string
        mockResponse   string
        mockStatusCode int
        wantErr        bool
    }{
        {
            name: "success",
            mockResponse: `{
                "httpStatusCode": 200,
                "traceId": "test-trace",
                "data": [{"id": "host1", "type": "unifi-os"}]
            }`,
            mockStatusCode: http.StatusOK,
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(tt.mockStatusCode)
                w.Write([]byte(tt.mockResponse))
            }))
            defer server.Close()

            // Test implementation...
        })
    }
}
```

### Mock Response Accuracy

**Critical**: Mock responses must match real API responses exactly:

- Field names and types must be correct
- Enum values must match observed values
- Status codes must match actual API behavior
- Null fields should be null, not omitted or empty

This is why we validate against real hardware first, then create matching mocks.

## Validation Against Real API

Before writing tests, we validate behavior against real UniFi controllers:

### Manual Testing

Create test scripts to verify actual API behavior:

```bash
# Example: Test DELETE status codes
UNIFI_API_KEY=key go run /tmp/test_delete.go
```

### Test Reality Tool

The project includes a tool to validate types against real API responses:

```bash
go run ./cmd/test-reality -api-key your-key
```

This ensures our types match actual API responses.

## Coverage Goals

### Current Coverage (as of 2025-11-12)

- **Network API**: 63.1%
- **Site Manager API**: (to be measured)

### Coverage Philosophy

We don't chase arbitrary coverage numbers. Coverage increases when:

1. We add support for new API endpoints
2. We discover new error conditions on real hardware
3. We add features that need testing (rate limiting, retries, etc.)

**Coverage decreases are acceptable** if:
- Generated code increases without corresponding test value
- Internal implementation changes make tests redundant
- We remove dead code

### What Affects Coverage

- ✅ **Client methods** - Always tested
- ✅ **Error handling** - Tested for observed errors
- ✅ **Rate limiting** - Tested for behavior, not internals
- ✅ **Retry logic** - Tested for observable behavior
- ❌ **Generated code** - Not always worth testing
- ❌ **Simple getters/setters** - Usually not tested
- ❌ **Constants and types** - Self-documenting, no tests needed

## Test Organization

### File Structure

```
api/
├── sitemanager/
│   ├── client.go          # Implementation
│   ├── client_test.go     # Tests
│   └── generated.go       # Generated code (not tested directly)
└── network/
    ├── client.go          # Implementation
    ├── client_test.go     # Tests
    └── generated.go       # Generated code (not tested directly)
```

### Test Naming

- Test files: `*_test.go`
- Test functions: `TestFunctionName(t *testing.T)`
- Table-driven subtests: Use descriptive names

```go
func TestListHosts(t *testing.T) {
    tests := []struct {
        name string
        // ...
    }{
        {name: "success"},
        {name: "unauthorized"},
        {name: "rate_limited"},
    }
}
```

## When to Add Tests

### ✅ Add Tests When:

1. **Adding new API endpoint support**
   - Test successful response parsing
   - Test error conditions (401, 404, 500, etc.)
   - Test rate limiting if applicable

2. **Fixing bugs discovered on real hardware**
   - Reproduce the bug in a test
   - Fix the bug
   - Verify test passes

3. **Adding complex logic**
   - Rate limiting changes
   - Retry logic modifications
   - Custom error handling

### ❌ Don't Add Tests For:

1. **Generated code** - Already validated by oapi-codegen
2. **Simple wrappers** - No logic to test
3. **Theoretical scenarios** - Only test what we've seen happen
4. **100% coverage goals** - Not our objective

## Integration Testing

### Real Controller Testing

When validating against real UniFi controllers, maintain separate environments:

- **Production controllers**: **READ-ONLY** access, no destructive operations
- **Test/Development controllers**: Destructive operations allowed for validation

### Testing Destructive Operations

**NEVER run DELETE/UPDATE tests on production controllers**:

```bash
# ❌ WRONG - Don't do this on production
UNIFI_CONTROLLER=https://unifi.local go run test_delete.go

# ✅ CORRECT - Use dedicated test environment
UNIFI_CONTROLLER=https://test-unifi.local go run test_delete.go
```

Always use a separate test controller (VM or spare hardware) for testing operations that modify configuration.

## Continuous Integration

Tests run automatically on:
- Pull requests
- Commits to main branch
- Release tags

CI validates:
- All tests pass
- Linters pass (golangci-lint)
- Code compiles for all supported platforms

## Contributing Tests

When contributing, ensure:

1. ✅ Tests reflect real API behavior (not assumptions)
2. ✅ Mock responses match actual API responses exactly
3. ✅ Tests use table-driven design where applicable
4. ✅ Tests run with `t.Parallel()` where safe
5. ✅ Error messages are clear and actionable
6. ❌ Don't add tests just to increase coverage percentage

## Test Maintenance

### When API Behavior Changes

If UniFi updates their API and behavior changes:

1. Verify new behavior on real hardware
2. Update mock responses to match new behavior
3. Update tests to expect new behavior
4. Document the change in commit message

### When Tests Fail

If tests start failing:

1. Check if UniFi API behavior changed (validate on real hardware)
2. Check if mock responses need updating
3. Check if our assumptions were wrong
4. Update tests to match reality

## Example Test

Here's a complete example showing our testing approach:

```go
func TestCreateFirewallPolicy(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name           string
        mockResponse   string
        mockStatusCode int
        wantErr        bool
        checkResponse  bool
    }{
        {
            name: "success",
            mockResponse: `{
                "httpStatusCode": 200,
                "traceId": "abc123",
                "data": {
                    "_id": "policy123",
                    "name": "Test Policy",
                    "action": "ALLOW",
                    "enabled": true
                }
            }`,
            mockStatusCode: http.StatusOK,
            wantErr:        false,
            checkResponse:  true,
        },
        {
            name:           "unauthorized",
            mockResponse:   `{"code": "UNAUTHORIZED", "message": "Invalid API key"}`,
            mockStatusCode: http.StatusUnauthorized,
            wantErr:        true,
            checkResponse:  false,
        },
        {
            name:           "bad_request",
            mockResponse:   `{"code": "BAD_REQUEST", "message": "Invalid policy"}`,
            mockStatusCode: http.StatusBadRequest,
            wantErr:        true,
            checkResponse:  false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                // Validate request
                if r.Method != http.MethodPost {
                    t.Errorf("Expected POST, got %s", r.Method)
                }

                w.WriteHeader(tt.mockStatusCode)
                w.Write([]byte(tt.mockResponse))
            }))
            defer server.Close()

            client, _ := network.NewWithConfig(&network.ClientConfig{
                ControllerURL:      server.URL,
                APIKey:             "test-key",
                InsecureSkipVerify: true,
            })

            policy := &network.FirewallPolicyInput{
                Name:    "Test Policy",
                Action:  network.FirewallPolicyInputActionALLOW,
                Enabled: true,
            }

            result, err := client.CreateFirewallPolicy(context.Background(), "default", policy)

            if tt.wantErr {
                if err == nil {
                    t.Error("Expected error, got nil")
                }
                return
            }

            if err != nil {
                t.Errorf("Unexpected error: %v", err)
                return
            }

            if tt.checkResponse {
                if result.UnderscoreId != "policy123" {
                    t.Errorf("Expected ID 'policy123', got '%s'", result.UnderscoreId)
                }
                if result.Name != "Test Policy" {
                    t.Errorf("Expected name 'Test Policy', got '%s'", result.Name)
                }
            }
        })
    }
}
```

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Table-Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [httptest Package](https://pkg.go.dev/net/http/httptest)
- [UniFi Site Manager API Docs](https://developer.ui.com/site-manager-api/gettingstarted)
- [UniFi Network API Docs](https://developer.ui.com/network-api/unifi-network-api)

---

**Remember**: Tests are documentation of how the API actually works, not how we think it should work.
