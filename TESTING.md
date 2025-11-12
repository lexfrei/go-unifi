# Testing Guide

## Philosophy

**This project does NOT aim for 100% test coverage or follow strict TDD.**

Tests reflect real-world API behavior observed on actual UniFi hardware. We write tests only when we are confident in the expected behavior, not for theoretical scenarios or arbitrary coverage goals.

## Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Run Go linters
golangci-lint run ./...

# Run Markdown linters
markdownlint *.md **/*.md
```

## Testing Approach

- **Framework**: Standard Go `testing` package with `httptest` for mocks
- **Style**: Table-driven tests with `t.Parallel()`
- **Mock Responses**: Must match actual API responses exactly
- **Validation**: Test against real controllers first, then write mocks

## When to Write Tests

Write tests when:

- Adding new API endpoint support
- Fixing bugs discovered on real hardware
- Adding complex logic (rate limiting, retry logic, etc.)

Do NOT write tests for:

- Generated code (already validated by oapi-codegen)
- Simple wrappers with no logic
- Theoretical scenarios never observed on real hardware
- Arbitrary coverage percentage goals

Coverage is NOT a goal. It increases when we add features, decreases when we remove dead code. Both are acceptable.
