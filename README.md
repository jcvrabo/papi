# PAPI - Platform API

**Control plane for runtime environment abstraction with Cloud Foundry REI implementation**

PAPI (Platform API) is a Go-based control plane that provides a unified REST API for managing namespaces across multiple runtime environments (Cloud Foundry, Kubernetes, etc.) through a pluggable Runtime Environment Interface (REI).

[![Go Version](https://img.shields.io/badge/go-1.22+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

## Features

- **Namespace Management**: Create and list namespaces with template-based naming and metadata validation
- **Multi-Environment Groups**: Organize runtime environments into groups with orchestrated provisioning
- **Async Provisioning**: Queue-based operation execution across grouped environments with eventual consistency
- **OIDC Authentication**: Secure endpoints with OpenID Connect token validation
- **Role-Based Access Control**: Platform-level roles (admin, namespace-creator) and namespace-level permissions
- **Identity Group Integration**: Cache-based membership resolution with TTL refresh
- **Pluggable REI**: Extensible Runtime Environment Interface for supporting multiple platforms
- **Cloud Foundry Support**: Built-in REII (REI Implementation) for Cloud Foundry (org/space provisioning)
- **API Versioning**: Contract-first development with OpenAPI 3.1 specifications
- **Observability**: Structured logging (slog), Prometheus metrics, OpenTelemetry tracing

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        PAPI REST API (v1)                   │
│  /namespaces  /groups  /health  /info                       │
└───────────────────┬─────────────────────────────────────────┘
                    │
        ┌───────────┴───────────┐
        │   Domain Services     │
        │  (Namespace, Groups)  │
        └───────────┬───────────┘
                    │
    ┌───────────────┼────────────────────┐
    │               │                    │
┌───▼────┐    ┌────▼────┐        ┌──────▼──────┐
│  OIDC  │    │  Queue  │        │  PostgreSQL │
│  Auth  │    │ Worker  │        │   Storage   │
└────────┘    └────┬────┘        └─────────────┘
                   │
                   │ REI Interface
              ┌────┴────┬────────────┐
              │         │            │
         ┌────▼───┐ ┌───▼────┐  ┌───▼────┐
         │   CF   │ │  k8s   │  │  ...   │
         │  REII  │ │  REII  │  │  REII  │
         └────────┘ └────────┘  └────────┘
```

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- PostgreSQL 15+

### Local Development

1. **Clone the repository**
   ```bash
   git clone https://github.com/jcvrabo/papi.git
   cd papi
   ```

2. **Start dependencies**
   ```bash
   docker-compose up -d
   ```
   This starts:
   - PostgreSQL (port 5432)
   - Mock OAuth2 Server (port 8888)

3. **Configure environment**
   ```bash
   export PAPI_DB_URL="postgres://papi:papi@localhost:5432/papi?sslmode=disable"
   export PAPI_OIDC_ISSUER="http://localhost:8888/default"
   export PAPI_LISTEN_ADDR=":8080"
   export PAPI_LOG_LEVEL="info"
   ```

4. **Run migrations**
   ```bash
   go build -o papi ./cmd/papi
   ./papi migrate up
   ```

5. **Start the server**
   ```bash
   ./papi serve
   ```

6. **Verify it's running**
   ```bash
   curl http://localhost:8080/api/v1/health
   curl http://localhost:8080/api/v1/info
   ```

### Building

```bash
# Build binary
make build

# Run tests
make test

# Run linter
make lint

# Build Docker image
docker build -t papi:latest .
```

## API Documentation

PAPI exposes a versioned REST API (`/api/v1/`) with the following endpoints:

### Public Endpoints (No Auth)
- `GET /api/v1/health` - Health check with environment status
- `GET /api/v1/info` - Platform version and OIDC issuer info

### Authenticated Endpoints
- `GET /api/v1/namespaces` - List namespaces (filtered by user access)
- `POST /api/v1/namespaces` - Create namespace (requires `namespace-creator` role)
- `GET /api/v1/namespaces/{id}/status` - Get namespace provisioning status
- `GET /api/v1/groups` - List runtime environment groups
- `POST /api/v1/groups` - Create group (admin only)
- `PUT /api/v1/groups/{id}` - Update group (admin only)
- `DELETE /api/v1/groups/{id}` - Delete group (admin only)

Full OpenAPI specification: [`api/openapi/v1.yaml`](api/openapi/v1.yaml)

## Configuration

PAPI is configured via environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PAPI_DB_URL` | Yes | - | PostgreSQL connection string |
| `PAPI_OIDC_ISSUER` | Yes | - | OIDC provider issuer URL |
| `PAPI_LISTEN_ADDR` | No | `:8080` | HTTP server listen address |
| `PAPI_LOG_LEVEL` | No | `info` | Log level (debug, info, warn, error) |
| `PAPI_METRICS_ADDR` | No | `:9090` | Prometheus metrics endpoint address |
| `PAPI_IDENTITY_CACHE_TTL` | No | `300` | Identity group cache TTL (seconds) |

## Runtime Environment Interface (REI)

PAPI abstracts runtime environments through a pluggable REI. The interface is defined in `pkg/rei/rei.go`:

```go
type RuntimeEnvironment interface {
    CreateNamespace(ctx context.Context, req CreateNamespaceRequest) (*CreateNamespaceResult, error)
    DeleteNamespace(ctx context.Context, req DeleteNamespaceRequest) error
    HealthCheck(ctx context.Context) error
}
```

### Built-in Implementations

- **Cloud Foundry REII** (`internal/rei/cloudfoundry/`) - Creates CF orgs and spaces

### Creating Custom REIIs

1. Implement the `rei.RuntimeEnvironment` interface
2. Register a factory function in the REI registry
3. Configure runtime environments in PostgreSQL to use your REII type

See `pkg/rei/rei.go` and `internal/rei/cloudfoundry/cloudfoundry.go` for examples.

## Project Structure

```
.
├── cmd/papi/              # Application entry point
├── internal/
│   ├── api/v1/            # HTTP handlers, middleware, router
│   ├── config/            # Configuration, logging, metrics, tracing
│   ├── domain/            # Business logic (namespace, group, operation)
│   ├── platform/          # Platform-level auth and roles
│   ├── queue/             # Async operation processor
│   ├── rei/               # REI implementations (Cloud Foundry)
│   └── store/             # PostgreSQL repositories and migrations
├── pkg/rei/               # Public REI interface (importable)
├── tests/
│   ├── contract/          # API contract tests
│   ├── integration/       # Integration tests (testcontainers)
│   └── e2e/               # End-to-end tests
├── api/openapi/           # OpenAPI 3.1 specifications
└── specs/                 # Feature specifications and design docs
```

## Development Workflow

PAPI follows TDD and contract-first development:

1. **Specification** - Write user stories and acceptance criteria
2. **Contract** - Define OpenAPI spec for API changes
3. **Tests** - Write contract and integration tests (failing)
4. **Implementation** - Implement features to pass tests
5. **Verification** - Run full test suite and linter

### Running Tests

```bash
# All tests
go test ./...

# Contract tests only
go test ./tests/contract/...

# With coverage
go test -cover ./...
```

### Code Quality

```bash
# Run linter
golangci-lint run

# Format code
gofumpt -w .
```

## Constitution & Principles

PAPI development follows strict engineering principles defined in `.specify/memory/constitution.md`:

1. **API Versioning** - URI-based (`/api/v1/`), breaking changes require version bump
2. **Contract-First Development** - OpenAPI specs before implementation
3. **Test-First Development** - TDD mandatory, tests written before code
4. **Observability** - Structured logging, metrics, and tracing on all operations
5. **Governance** - Constitution changes require explicit approval

## License

MIT License - see [LICENSE](LICENSE) for details

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Follow TDD and contract-first development
4. Ensure tests pass and linter is happy
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## Support

- **Documentation**: See [`specs/`](specs/) directory for detailed design docs
- **Issues**: https://github.com/jcvrabo/papi/issues
- **Discussions**: https://github.com/jcvrabo/papi/discussions

## Roadmap

See [`specs/001-papi-control-plane/tasks.md`](specs/001-papi-control-plane/tasks.md) for the full implementation plan.

**Current Status**: ✅ Phase 1-8 Complete (80/81 tasks)
- ✅ MVP features (namespace list/create, health/info)
- ✅ Group management CRUD
- ✅ Async provisioning with Cloud Foundry REII
- ✅ OIDC auth, RBAC, rate limiting, observability
- ⏳ Manual smoke testing pending

---

Built with ❤️ using Go, PostgreSQL, and contract-first development
