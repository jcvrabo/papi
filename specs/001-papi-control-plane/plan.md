# Implementation Plan: PAPI Control Plane

**Branch**: `001-namespace-handling` | **Date**: 2026-05-28 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/001-papi-control-plane/spec.md`

## Summary

Build PAPI (Platform API), a Go-based control plane that abstracts interactions with runtime
environments through a pluggable interface (REI). V1 exposes versioned REST endpoints for
namespace management (list/create) and runtime environment group CRUD, with async provisioning
across grouped environments, OIDC authentication, role-based authorization, and PostgreSQL
persistence. Ships with a Cloud Foundry REI implementation.

## Technical Context

**Language/Version**: Go 1.22+
**Primary Dependencies**: chi (HTTP router), sqlc or pgx (database), golang-jwt (JWT validation), testify (assertions), oapi-codegen (OpenAPI codegen)
**Storage**: PostgreSQL 15+ with JSONB for extensible metadata and REII-specific extensions
**Testing**: go test, testcontainers-go (integration), httptest (contract tests)
**Target Platform**: Linux server (containerized, Docker/Kubernetes)
**Project Type**: web-service (REST API)
**Performance Goals**: <2s namespace list for 1000 items, <500ms health/info, 100 concurrent users
**Constraints**: <200ms p95 for synchronous endpoints, eventual consistency for async operations
**Scale/Scope**: 100 concurrent users, 1000s of namespaces, 5-10 runtime environments per group

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**API Versioning Check**:
- [x] API version determined: v1
- [x] Breaking vs. non-breaking changes identified: all v1 operations are additive (initial version)
- [x] If breaking: N/A (initial version)
- [x] OpenAPI contract specification planned: yes, generated via oapi-codegen
- [x] Contract tests planned before implementation: yes, using httptest against OpenAPI spec
- [x] Backward compatibility strategy defined: additive-only changes within v1
- [x] Deprecation plan: N/A (no prior versions)

**Common Constitution Violations to Avoid**:
- [x] Breaking changes without version increment: N/A (v1 initial)
- [x] Implementation before tests written: TDD enforced via contract tests first
- [x] Missing API contracts/OpenAPI specs: OpenAPI 3.1 spec generated in Phase 1
- [x] Insufficient observability: structured logging (slog), metrics (prometheus), tracing (OpenTelemetry)

**All gates pass. No violations.**

## Project Structure

### Documentation (this feature)

```text
specs/001-papi-control-plane/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (OpenAPI specs)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
cmd/
└── papi/
    └── main.go              # Application entry point

internal/
├── api/
│   └── v1/
│       ├── handlers/        # HTTP handlers per resource
│       ├── middleware/       # Auth, logging, metrics middleware
│       └── router.go        # v1 route registration
├── config/                  # Application configuration
├── domain/
│   ├── namespace/           # Namespace domain logic
│   ├── group/               # Runtime environment group domain logic
│   └── operation/           # Operation queue domain logic
├── platform/
│   ├── roles.go             # Platform-level role definitions
│   └── auth.go              # OIDC token validation, authorization
├── rei/
│   ├── interface.go         # REI specification (importable package)
│   └── cloudfoundry/        # Cloud Foundry REII implementation
├── queue/                   # Internal operation queue processor
└── store/
    ├── postgres/            # PostgreSQL repository implementations
    └── migrations/          # Database migration files

pkg/
└── rei/                     # Public REI package (importable by external REIIs)
    └── rei.go               # Interface definitions

tests/
├── contract/                # API contract tests against OpenAPI spec
├── integration/             # Integration tests (testcontainers)
└── e2e/                     # End-to-end tests

api/
└── openapi/
    └── v1.yaml              # OpenAPI 3.1 specification
```

**Structure Decision**: Standard Go project layout with `cmd/` for entrypoints, `internal/` for
private application code, `pkg/` for the public REI interface package, and `api/` for OpenAPI
specs. The REI is both an internal interface and a public importable package (`pkg/rei/`) so
external REII implementations can depend on it.

## Complexity Tracking

> No constitution violations to justify.
