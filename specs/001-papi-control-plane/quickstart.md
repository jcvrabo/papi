# Quickstart: PAPI Control Plane

## Prerequisites

- Go 1.22+
- PostgreSQL 15+
- A running OIDC provider (UAA, Keycloak, or similar)
- A Cloud Foundry instance (for REII integration testing)

## Setup

```bash
# Clone and enter the project
git clone <repository-url>
cd papi

# Install dependencies
go mod download

# Set environment variables
export PAPI_DB_URL="postgres://papi:papi@localhost:5432/papi?sslmode=disable"
export PAPI_OIDC_ISSUER="https://uaa.example.com/oauth/token"
export PAPI_LISTEN_ADDR=":8080"

# Run database migrations
go run cmd/papi/main.go migrate up

# Start the server
go run cmd/papi/main.go serve
```

## Verify

```bash
# Health check (no auth required)
curl http://localhost:8080/api/v1/health

# Platform info (no auth required)
curl http://localhost:8080/api/v1/info

# List namespaces (requires auth)
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/namespaces

# Create a namespace (requires namespace-creator role)
curl -X POST http://localhost:8080/api/v1/namespaces \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name_components": {"team": "payments", "project": "checkout", "env": "dev"},
    "members": [{"identifier": "jane.doe", "type": "user", "role": "read_write"}]
  }'

# List runtime environment groups (any authenticated user)
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/groups
```

## Running Tests

```bash
# Unit tests
go test ./internal/... -short

# Integration tests (requires Docker for testcontainers)
go test ./tests/integration/... -v

# Contract tests (validates API against OpenAPI spec)
go test ./tests/contract/... -v

# All tests
go test ./...
```

## Configuration

| Variable | Required | Description |
|----------|----------|-------------|
| PAPI_DB_URL | Yes | PostgreSQL connection string |
| PAPI_OIDC_ISSUER | Yes | OIDC issuer URL for token validation |
| PAPI_LISTEN_ADDR | No | Listen address (default: `:8080`) |
| PAPI_LOG_LEVEL | No | Log level: debug, info, warn, error (default: `info`) |
| PAPI_METRICS_ADDR | No | Prometheus metrics address (default: `:9090`) |
| PAPI_IDENTITY_CACHE_TTL | No | TTL for identity group cache in seconds (default: `300`) |
