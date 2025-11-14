# Releasing Guide

This document describes the release process for go-unifi.

## Release Strategy

go-unifi follows a **single module** strategy with **semantic versioning** (semver).

### Single Module Structure

- **One module**: `github.com/lexfrei/go-unifi`
- **One version**: All packages share the same version
- **One tag**: Simple `vX.Y.Z` tags (no prefixes)

All packages (`api/sitemanager`, `api/network`, `observability`, `internal/*`) are part of the same module and released together.

## Semantic Versioning

Follows [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR** (`vX.0.0`): Breaking changes (incompatible API changes)
- **MINOR** (`v0.X.0`): New features (backward compatible)
- **PATCH** (`v0.0.X`): Bug fixes (backward compatible)

### Breaking Changes

Breaking changes include:

- Removing or renaming public types, functions, or methods
- Changing function signatures (parameters or return types)
- Removing or renaming exported constants or variables
- Changing struct field types or removing fields
- Modifying OpenAPI schema in a way that changes generated code

**Before 1.0.0**: Breaking changes are allowed in MINOR versions (v0.X.0).

**After 1.0.0**: Breaking changes require MAJOR version bump.

## Release Process

### 1. Prepare Release

1. Ensure `main` branch is clean and all tests pass:

   ```bash
   git checkout main
   git pull origin main
   golangci-lint run ./...
   go test ./...
   ```

2. Review changes since last release:

   ```bash
   # List commits since last tag
   git log $(git describe --tags --abbrev=0)..HEAD --oneline
   ```

3. Determine version bump:
   - Breaking changes → MAJOR
   - New features → MINOR
   - Bug fixes → PATCH

### 2. Create Release Tag

Create annotated tag with release notes:

```bash
git tag vX.Y.Z --message "Release vX.Y.Z - Short description

Detailed description of changes:
- Feature 1: description
- Feature 2: description
- Bug fix 1: description

Breaking changes (if any):
- Breaking change description

Co-Authored-By: Claude <noreply@anthropic.com>"
```

**Examples:**

```bash
# Minor release (new feature)
git tag v0.2.0 --message "Release v0.2.0 - Add Network API support

Add support for UniFi Network API v1:
- api/network: New package for Network API client
- examples/network: Example programs for common operations

Co-Authored-By: Claude <noreply@anthropic.com>"

# Patch release (bug fix)
git tag v0.1.2 --message "Release v0.1.2 - Fix rate limiting

Fix rate limiter token bucket initialization:
- Fix: Correct initial burst size calculation
- Fix: Handle edge case with zero rate limit

Co-Authored-By: Claude <noreply@anthropic.com>"

# Major release (breaking change)
git tag v1.0.0 --message "Release v1.0.0 - Stable API

First stable release with guaranteed API stability:
- Stabilize all public APIs
- Comprehensive test coverage
- Production-ready observability

Breaking changes:
- ClientConfig: Remove deprecated RetryCount field (use MaxRetries)
- Logger: Rename Debug method to Debugf for consistency

Co-Authored-By: Claude <noreply@anthropic.com>"
```

### 3. Push Release

Push tag to GitHub:

```bash
git push origin vX.Y.Z
```

**Important**: Once pushed, tags are **immutable** in Go module proxy system. Never delete or recreate pushed tags.

### 4. Verify Release

Wait 5-10 minutes for Go proxy to index, then verify:

```bash
# Check module is available
go list -m github.com/lexfrei/go-unifi@vX.Y.Z

# Verify module info
go list -m -json github.com/lexfrei/go-unifi@vX.Y.Z

# Test in fresh project
cd /tmp
mkdir test-release && cd test-release
go mod init test
go get github.com/lexfrei/go-unifi@vX.Y.Z
```

### 5. Create GitHub Release

Create GitHub release from tag:

```bash
gh release create vX.Y.Z \
  --title "vX.Y.Z - Short description" \
  --notes "Detailed release notes..."
```

## Versioning Guidelines

### When to Bump Version

- **PATCH**: Bug fixes, documentation updates, internal refactoring
- **MINOR**: New APIs, new features, new packages (backward compatible)
- **MAJOR**: Breaking changes, removing deprecated features

### Pre-1.0.0 Versions

Current status: **v0.x.x** (unstable API)

- API may change between minor versions
- Breaking changes allowed in v0.X.0 releases
- Use with caution in production

### Post-1.0.0 Versions

Once v1.0.0 is released:

- API stability guarantee
- Breaking changes only in major versions
- Deprecation warnings before removal

## Module Proxy and Caching

### Go Module Proxy

Go modules are cached by [proxy.golang.org](https://proxy.golang.org) and [sum.golang.org](https://sum.golang.org).

**Key facts:**

- Once published, versions are **immutable**
- Tags cannot be deleted or changed
- Modules are cached for 24+ hours
- GOPROXY can be bypassed with `GOPROXY=direct`

### If Release Goes Wrong

If you push a broken tag:

1. **DO NOT** delete the tag
2. **DO NOT** recreate the tag
3. **Fix** the issue in code
4. **Release** new PATCH version (X.Y.Z+1)

**Why?** Go proxy has likely cached the broken version. Deleting the tag creates inconsistencies.

### Testing Before Release

Always test locally before pushing tags:

```bash
# Create local test project
cd /tmp
rm -rf test-local && mkdir test-local && cd test-local
go mod init test-local

# Point to local module for testing
go mod edit -replace github.com/lexfrei/go-unifi=/path/to/local/go-unifi

# Write test code
cat > main.go <<EOF
package main
import "github.com/lexfrei/go-unifi/api/sitemanager"
func main() {
    client, _ := sitemanager.NewWithConfig(&sitemanager.ClientConfig{
        APIKey: "test",
    })
    _ = client
}
EOF

# Build and test
go mod tidy
go build .
```

## Troubleshooting

### Tag Already Exists

```bash
# Delete local tag
git tag --delete vX.Y.Z

# Delete remote tag (only if not yet published!)
git push origin :refs/tags/vX.Y.Z
```

### Go Proxy Cached Old Version

Wait 24 hours or use direct mode:

```bash
GOPROXY=direct GOSUMDB=off go get github.com/lexfrei/go-unifi@vX.Y.Z
```

### Module Not Found

1. Check tag exists: `git ls-remote --tags origin`
2. Check tag format: Must be `vX.Y.Z` (not `X.Y.Z` or `version-X.Y.Z`)
3. Wait 5-10 minutes for proxy indexing
4. Verify tag points to correct commit: `git show vX.Y.Z`

## Changelog

Maintain CHANGELOG.md with:

- Version number and release date
- Breaking changes (highlighted)
- New features
- Bug fixes
- Deprecations

Follow [Keep a Changelog](https://keepachangelog.com/) format.

## References

- [Go Modules Reference](https://go.dev/ref/mod)
- [Semantic Versioning](https://semver.org/)
- [Go Module Proxy](https://proxy.golang.org)
- [Go Sum Database](https://sum.golang.org)
