# go-unifi

Almost pure Go client libraries for UniFi APIs with type-safe code generation from OpenAPI specifications.

## 📦 Available APIs

### [UniFi Site Manager API](./api/sitemanager/)

Cloud-based API for managing multiple UniFi sites from a central location.

```go
import "github.com/lexfrei/go-unifi/api/sitemanager"

client, _ := sitemanager.New("your-api-key")
hosts, _ := client.ListHosts(context.Background(), nil)
```

**Features:**
- Hosts, Sites, Devices management
- ISP Metrics and monitoring
- SD-WAN configuration
- Dual rate limiting (10K req/min v1, 100 req/min EA)

**Documentation:** [api/sitemanager/README.md](./api/sitemanager/)

---

### [UniFi Network API](./api/network/)

Local controller API for detailed device and client management on your network.

```go
import "github.com/lexfrei/go-unifi/api/network"

client, _ := network.New("https://unifi.local", "your-api-key")
sites, _ := client.ListSites(context.Background(), nil)
```

**Features:**
- Sites, Devices, Clients management
- Real-time device status
- Port and radio interface details
- Support for self-signed certificates

**Documentation:** [api/network/doc.go](./api/network/doc.go)

---

## 🚀 Quick Start

### Installation

```bash
go get github.com/lexfrei/go-unifi
```

### Choose Your API

**Use Site Manager API when:**
- Managing multiple sites remotely
- Accessing cloud-hosted controllers
- Need ISP metrics and SD-WAN features
- Want centralized monitoring

**Use Network API when:**
- Managing a single local controller
- Need detailed device/client information
- Working with on-premises hardware
- Require real-time port/radio status

## 📖 Examples

- [Site Manager Examples](./examples/sitemanager/) - Cloud API usage
- [Network API Examples](./examples/network/) - Local controller usage

## ✨ Features

- ✅ **Type-safe** - Generated from OpenAPI specifications
- ✅ **Rate limiting** - Automatic with configurable limits
- ✅ **Retry logic** - Exponential backoff for failures
- ✅ **Error handling** - Using `github.com/cockroachdb/errors`
- ✅ **Context support** - All operations support cancellation
- ✅ **Well documented** - Extensive examples and godoc

## 🧪 Tested Hardware

Both API clients have been tested and validated against:

- **UniFi Dream Router (UDR7)** running:
  - UniFi OS **4.3.87** with Network Application **9.4.19**
  - UniFi OS **4.3.6** with Network Application **9.4.19**
  - UniFi OS **4.3.6** with Network Application **9.5.21**

**Site Manager API:** Validated against official UniFi Site Manager API documentation.

**Network API:** Derived from controller endpoint documentation, official docs, and reverse engineering.

**Note:** API responses may vary depending on hardware models, firmware versions, and UniFi OS releases.

## 🏗️ Project Structure

```
go-unifi/
├── api/
│   ├── sitemanager/    # Cloud-based Site Manager API
│   └── network/        # Local Network API
├── internal/           # Shared infrastructure (rate limiting, retry)
├── examples/           # Working examples for both APIs
└── cmd/                # Command-line tools
```

## 🛠️ Development

### Generate Code from OpenAPI

```bash
# Site Manager API
cd api/sitemanager && oapi-codegen -config .oapi-codegen.yaml openapi.yaml

# Network API
cd api/network && oapi-codegen -config .oapi-codegen.yaml openapi.yaml
```

### Test Against Reality

Validate types against real API responses:

```bash
go run github.com/lexfrei/go-unifi/cmd/test-reality@latest -api-key your-key
```

See [cmd/test-reality/README.md](./cmd/test-reality/README.md) for details.

### Run Linters

```bash
golangci-lint run ./...
```

## 📚 API Documentation

- [UniFi Site Manager API](https://developer.ui.com/site-manager-api/gettingstarted) - Official docs
- [UniFi Network API](https://developer.ui.com/network-api/unifi-network-api) - Official docs
- [OpenAPI Specifications](./api/) - Local specs for both APIs

## 🔑 Authentication

### Site Manager API

1. Go to [unifi.ui.com](https://unifi.ui.com)
2. Navigate to **API** section
3. Click **Create API Key**
4. Store securely and pass to client

### Network API

1. Open UniFi Network controller
2. Go to **Settings > Control Plane > Integrations**
3. Create API key
4. Use with local controller URL

## 📄 License

BSD-3-Clause - see [LICENSE](./LICENSE) file for details

## 👤 Maintainer

**Aleksei Sviridkin**

- Email: <f@lex.la>
- GPG: F57F 85FC 7975 F22B BC3F 2504 9C17 3EB1 B531 AA1F

---

**Note:** This library provides unofficial Go clients for UniFi APIs. Not affiliated with or endorsed by Ubiquiti Inc.
