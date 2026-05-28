# Research: PAPI Control Plane

**Date**: 2026-05-28
**Feature**: PAPI Control Plane
**Purpose**: Resolve technical decisions and document rationale

## 1. HTTP Router Selection

**Decision**: chi (go-chi/chi)

**Rationale**: Lightweight, idiomatic Go, stdlib-compatible (net/http), excellent middleware
ecosystem, supports route groups (ideal for `/api/v1/` versioning), no code generation overhead.

**Alternatives considered**:
- **gin**: Popular but uses custom context, less stdlib-compatible
- **echo**: Similar to chi but slightly heavier, custom context
- **stdlib only (Go 1.22 ServeMux)**: Now supports method routing but lacks middleware chaining and route groups
- **fiber**: Fast but based on fasthttp (not net/http compatible)

## 2. Database Access Pattern

**Decision**: pgx (direct driver) + sqlc (type-safe SQL generation)

**Rationale**: pgx is the most performant and feature-rich PostgreSQL driver for Go. sqlc
generates type-safe Go code from SQL queries, catching errors at compile time. Together they
provide excellent developer experience without ORM overhead. Supports JSONB, LISTEN/NOTIFY,
and connection pooling natively.

**Alternatives considered**:
- **GORM**: Full ORM, heavier, magic behavior, harder to optimize complex queries
- **sqlx**: Good but manual scanning; sqlc automates this with better type safety
- **ent**: Schema-as-code ORM, powerful but adds abstraction layer not needed here

## 3. Database Migrations

**Decision**: golang-migrate/migrate

**Rationale**: Well-maintained, supports PostgreSQL, can be embedded in binary or run as CLI,
integrates with testcontainers for testing. Simple up/down SQL files.

**Alternatives considered**:
- **goose**: Similar capability, slightly less active community
- **atlas**: Schema-as-code approach, more complex than needed for this project

## 4. Authentication / JWT Validation

**Decision**: golang-jwt/jwt + OIDC discovery

**Rationale**: Standard library for JWT validation in Go. Combined with OIDC discovery
(fetching JWKS from `.well-known/openid-configuration`), supports any compliant provider
(UAA, Keycloak, Azure AD, etc.) without vendor lock-in.

**Alternatives considered**:
- **coreos/go-oidc**: Higher-level OIDC library; good but adds dependency for something achievable with golang-jwt + HTTP client
- **Custom validation**: Too risky for security-critical code

**Decision update**: Use `coreos/go-oidc/v3` for OIDC discovery + JWKS caching, combined with
`golang-jwt` for token parsing. This provides robust JWKS key rotation handling.

## 5. API Contract / OpenAPI Generation

**Decision**: Hand-written OpenAPI 3.1 YAML + oapi-codegen for server stubs

**Rationale**: Contract-first approach aligns with constitution (Principle IV). Hand-writing
the spec ensures it's the source of truth. oapi-codegen generates Go interfaces and types,
ensuring implementation matches contract. Supports chi router directly.

**Alternatives considered**:
- **swaggo/swag**: Code-first (annotations → spec), violates contract-first principle
- **buf/connect**: gRPC-based, not REST
- **huma**: Interesting but less mature ecosystem

## 6. Observability Stack

**Decision**: slog (structured logging) + OpenTelemetry (tracing) + Prometheus (metrics)

**Rationale**: slog is stdlib (Go 1.21+), zero-dependency structured logging. OpenTelemetry
is the industry standard for distributed tracing. Prometheus metrics are de facto standard
for Kubernetes deployments. All three integrate cleanly.

**Alternatives considered**:
- **zerolog/zap**: Fast but slog is now stdlib and sufficient
- **Jaeger directly**: OTel is vendor-neutral superset
- **StatsD**: Less common in cloud-native deployments

## 7. Operation Queue Implementation

**Decision**: PostgreSQL-backed job queue (internal implementation using pg LISTEN/NOTIFY + polling)

**Rationale**: Keeps infrastructure minimal (no Redis/RabbitMQ dependency). PostgreSQL
LISTEN/NOTIFY provides near-real-time notification of new jobs. Polling provides reliability
guarantee. Operations are already persisted in PostgreSQL. Supports at-least-once delivery
with idempotency keys.

**Alternatives considered**:
- **Redis + worker**: Additional infrastructure dependency
- **RabbitMQ/NATS**: Overkill for this scale (5-10 environments per group)
- **go channels (in-memory)**: Not durable across restarts

## 8. Identity Group Membership Caching

**Decision**: In-memory cache with TTL (built-in sync.Map or go-cache) + database as fallback

**Rationale**: Resolved members are stored in database with last-refresh timestamp. In-memory
cache provides fast access for authorization checks. TTL refresh triggers async background
update. If identity provider is unreachable during refresh, stale cache is served with
degraded-status logging.

**Alternatives considered**:
- **Redis cache**: Additional infrastructure for something that fits in memory at this scale
- **Database-only**: Too many queries for high-frequency authorization checks
- **No cache (live resolution)**: Too slow and creates identity provider dependency for every request

## 9. Project Layout Decisions

**Decision**: `pkg/rei/` as public importable package, `internal/` for all private code

**Rationale**: External REII implementors need to import the REI interface definitions. Using
`pkg/rei/` makes this explicitly public and versionable via Go modules. All PAPI-internal
code stays in `internal/` to prevent accidental external imports.

## 10. Configuration Management

**Decision**: Environment variables + YAML config file (viper or envconfig)

**Rationale**: 12-factor app compliance. Environment variables for secrets and deployment
config. Optional YAML for local development. No runtime config reload needed for v1.

**Alternatives considered**:
- **viper**: Full-featured but heavy; koanf is lighter alternative
- **envconfig**: Simple struct tag-based env parsing, minimal

**Final choice**: kelseyhightower/envconfig for simplicity. YAML file support deferred to
future version if needed.
