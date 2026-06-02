# Tasks: PAPI Control Plane

**Input**: Design documents from `specs/001-papi-control-plane/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Tests are included as the constitution mandates TDD (Principle III: Test-First Development NON-NEGOTIABLE).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Go project**: `cmd/papi/`, `internal/`, `pkg/rei/`, `tests/`, `api/openapi/`
- Paths assume standard Go layout per plan.md

---

## Phase 1: Setup

**Purpose**: Project initialization and basic structure

- [x] T001 Initialize Go module with `go mod init` and create `cmd/papi/main.go` entry point
- [x] T002 [P] Create project directory structure per plan.md (`internal/api/v1/handlers/`, `internal/api/v1/middleware/`, `internal/config/`, `internal/domain/namespace/`, `internal/domain/group/`, `internal/domain/operation/`, `internal/platform/`, `internal/rei/`, `internal/rei/cloudfoundry/`, `internal/queue/`, `internal/store/postgres/`, `internal/store/migrations/`, `pkg/rei/`, `tests/contract/`, `tests/integration/`, `tests/e2e/`, `api/openapi/`)
- [x] T003 [P] Add core dependencies to go.mod: chi, pgx, sqlc, golang-jwt, coreos/go-oidc, oapi-codegen, testify, testcontainers-go, envconfig, prometheus client, OpenTelemetry SDK
- [x] T004 [P] Copy OpenAPI v1 spec to `api/openapi/v1.yaml`
- [x] T005 [P] Configure oapi-codegen to generate server interfaces and types from `api/openapi/v1.yaml` into `internal/api/v1/generated.go`
- [x] T006 [P] Setup linting (golangci-lint config) and formatting (gofumpt) in `.golangci.yml`
- [x] T007 [P] Create `Makefile` with targets: build, test, lint, migrate, generate, run
- [x] T008 [P] Create `Dockerfile` for containerized builds

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T009 Implement application configuration loading via envconfig in `internal/config/config.go` (PAPI_DB_URL, PAPI_OIDC_ISSUER, PAPI_LISTEN_ADDR, PAPI_LOG_LEVEL, PAPI_METRICS_ADDR, PAPI_IDENTITY_CACHE_TTL)
- [x] T010 [P] Setup structured logging with slog in `internal/config/logging.go` (JSON output, configurable level)
- [x] T011 [P] Setup PostgreSQL connection pool with pgx in `internal/store/postgres/connection.go`
- [x] T012 Create database migration framework using golang-migrate in `internal/store/migrations/` with initial schema migration `000001_initial_schema.up.sql` and `000001_initial_schema.down.sql` (all tables from data-model.md)
- [x] T013 Implement migration CLI command in `cmd/papi/main.go` (`migrate up`/`migrate down` subcommands)
- [x] T014 [P] Implement OIDC token validation middleware using coreos/go-oidc in `internal/api/v1/middleware/auth.go` (JWKS discovery, token parsing, claims extraction)
- [x] T015 [P] Implement platform role authorization helper in `internal/platform/auth.go` (check user roles from DB against required role)
- [x] T016 [P] Implement platform role store in `internal/store/postgres/platform_roles.go` (GetRolesForUser, HasRole queries)
- [x] T017 [P] Setup chi router with API v1 route group, request logging middleware, and metrics middleware in `internal/api/v1/router.go`
- [x] T018 [P] Implement error response helpers in `internal/api/v1/handlers/errors.go` (ErrorResponse, ValidationErrorResponse JSON serialization)
- [x] T019 [P] Setup Prometheus metrics endpoint on separate port in `internal/config/metrics.go`
- [x] T020 [P] Setup OpenTelemetry tracing initialization in `internal/config/tracing.go`
- [x] T021 [P] Define REI interface package in `pkg/rei/rei.go` (RuntimeEnvironment interface, request/result types, Status constants, Factory type, Registry)
- [x] T022 Wire up main.go: config loading → DB connection → migrations check → OIDC provider init → router setup → HTTP server start with graceful shutdown in `cmd/papi/main.go`

**Checkpoint**: Foundation ready — user story implementation can now begin

---

## Phase 3: User Story 1 — View Available Namespaces (Priority: P1) 🎯 MVP

**Goal**: Authenticated users can list namespaces they have access to

**Independent Test**: Authenticate and call GET /api/v1/namespaces; verify only accessible namespaces returned

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T023 [P] [US1] Contract test for GET /api/v1/namespaces in `tests/contract/namespaces_list_test.go` (validates response matches OpenAPI schema, pagination, auth rejection)
- [x] T024 [P] [US1] Integration test for namespace listing in `tests/integration/namespace_list_test.go` (testcontainers PostgreSQL, seed data, verify access control filtering)

### Implementation for User Story 1

- [x] T025 [P] [US1] Create Namespace domain model in `internal/domain/namespace/model.go` (Namespace struct, Member struct, enums for status/role/member_type)
- [x] T026 [P] [US1] Create IdentityGroupCache model in `internal/domain/namespace/identity_cache.go` (cache struct, TTL logic)
- [x] T027 [US1] Implement namespace store in `internal/store/postgres/namespaces.go` (ListByUser query — joins members table, filters by user identifier or cached group membership, pagination)
- [x] T028 [US1] Implement identity group cache store in `internal/store/postgres/identity_cache.go` (GetCachedMembers, RefreshCache, IsUserInGroup queries)
- [x] T029 [US1] Implement namespace list service in `internal/domain/namespace/service.go` (ListForUser method — combines direct membership + identity group cache lookup)
- [x] T030 [US1] Implement GET /api/v1/namespaces handler in `internal/api/v1/handlers/namespaces.go` (auth required, calls service, returns paginated NamespaceListResponse)
- [x] T031 [US1] Register namespace list route in `internal/api/v1/router.go`
- [x] T032 [US1] Add structured logging for namespace list operations (user, result count, duration) in handler

**Checkpoint**: User Story 1 fully functional — users can list their namespaces

---

## Phase 4: User Story 2 — Create a New Namespace (Priority: P1) 🎯 MVP

**Goal**: Users with namespace-creator role can create namespaces with async provisioning

**Independent Test**: Authenticate as namespace-creator, POST namespace, verify 202 response and IN_PROGRESS status

### Tests for User Story 2

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T033 [P] [US2] Contract test for POST /api/v1/namespaces in `tests/contract/namespaces_create_test.go` (validates 202 response schema, 400 validation errors, 403 unauthorized, 409 conflict)
- [x] T034 [P] [US2] Contract test for GET /api/v1/namespaces/{id}/status in `tests/contract/namespaces_status_test.go` (validates status response schema)
- [x] T035 [P] [US2] Integration test for namespace creation in `tests/integration/namespace_create_test.go` (end-to-end: create namespace, verify DB state, verify operations queued)

### Implementation for User Story 2

- [x] T036 [P] [US2] Implement namespace name template store in `internal/store/postgres/name_template.go` (GetActiveTemplate, ValidateComponents)
- [x] T037 [P] [US2] Implement metadata rules store in `internal/store/postgres/metadata_rules.go` (GetMandatoryRules, ValidateMetadata)
- [x] T038 [US2] Implement namespace name validation service in `internal/domain/namespace/naming.go` (resolve template, validate each component against regex/allowed values, compose final name)
- [x] T039 [US2] Implement metadata validation service in `internal/domain/namespace/metadata.go` (check mandatory fields present, validate against rules)
- [x] T040 [US2] Implement namespace creation service in `internal/domain/namespace/service.go` (CreateNamespace method — validate name, validate metadata, check uniqueness, persist namespace + members, resolve default group, queue operations)
- [x] T041 [P] [US2] Implement Operation domain model in `internal/domain/operation/model.go` (Operation struct, status enum, operation type enum)
- [x] T042 [US2] Implement operation store in `internal/store/postgres/operations.go` (CreateOperation, ListByNamespace, UpdateStatus queries)
- [x] T043 [US2] Implement POST /api/v1/namespaces handler in `internal/api/v1/handlers/namespaces.go` (namespace-creator role check, validation, calls service, returns 202 with NamespaceResponse)
- [x] T044 [US2] Implement GET /api/v1/namespaces/{id}/status handler in `internal/api/v1/handlers/namespaces.go` (returns per-environment provisioning status)
- [x] T045 [US2] Register namespace create and status routes in `internal/api/v1/router.go`
- [x] T046 [US2] Add structured logging for namespace creation (user, namespace name, group, member count, duration)

**Checkpoint**: User Story 2 fully functional — namespaces can be created with async provisioning queued

---

## Phase 5: User Story 3 — Discover Platform Capabilities (Priority: P2)

**Goal**: Unauthenticated users can check health and discover platform info

**Independent Test**: Call GET /api/v1/health and GET /api/v1/info without auth; verify correct responses

### Tests for User Story 3

- [x] T047 [P] [US3] Contract test for GET /api/v1/health in `tests/contract/health_test.go` (validates schema, no auth required)
- [x] T048 [P] [US3] Contract test for GET /api/v1/info in `tests/contract/info_test.go` (validates schema, version, OIDC endpoint)

### Implementation for User Story 3

- [x] T049 [P] [US3] Implement health check handler in `internal/api/v1/handlers/system.go` (aggregates runtime environment health from DB, returns overall status)
- [x] T050 [P] [US3] Implement info handler in `internal/api/v1/handlers/system.go` (returns version, api_version, OIDC issuer URL)
- [x] T051 [US3] Register health and info routes (no auth middleware) in `internal/api/v1/router.go`

**Checkpoint**: Platform discovery endpoints operational

---

## Phase 6: User Story 4 — Manage Runtime Environment Groups (Priority: P2)

**Goal**: Admins can CRUD groups; any authenticated user can list groups

**Independent Test**: Authenticate as admin, create/update/delete groups; authenticate as user, list groups

### Tests for User Story 4

- [x] T052 [P] [US4] Contract test for GET /api/v1/groups in `tests/contract/groups_list_test.go`
- [x] T053 [P] [US4] Contract test for POST /api/v1/groups in `tests/contract/groups_create_test.go` (201, 400, 403)
- [x] T054 [P] [US4] Contract test for PUT /api/v1/groups/{id} in `tests/contract/groups_update_test.go`
- [x] T055 [P] [US4] Contract test for DELETE /api/v1/groups/{id} in `tests/contract/groups_delete_test.go` (204, 403, 409)
- [x] T056 [P] [US4] Integration test for group CRUD in `tests/integration/groups_test.go`

### Implementation for User Story 4

- [x] T057 [P] [US4] Create RuntimeEnvironmentGroup domain model in `internal/domain/group/model.go` (Group struct, Environment struct, orchestration strategy enum)
- [x] T058 [US4] Implement group store in `internal/store/postgres/groups.go` (List, Create, Update, Delete, HasNamespaces queries)
- [x] T059 [US4] Implement group service in `internal/domain/group/service.go` (ListAll, Create, Update, Delete with namespace dependency check)
- [x] T060 [US4] Implement GET /api/v1/groups handler in `internal/api/v1/handlers/groups.go` (any authenticated user)
- [x] T061 [US4] Implement POST /api/v1/groups handler in `internal/api/v1/handlers/groups.go` (admin role check)
- [x] T062 [US4] Implement PUT /api/v1/groups/{id} handler in `internal/api/v1/handlers/groups.go` (admin role check)
- [x] T063 [US4] Implement DELETE /api/v1/groups/{id} handler in `internal/api/v1/handlers/groups.go` (admin role check, 409 if namespaces assigned)
- [x] T064 [US4] Register group routes in `internal/api/v1/router.go`

**Checkpoint**: Group management fully operational

---

## Phase 7: User Story 5 — Namespace Provisioning Across Groups (Priority: P2)

**Goal**: Operations queued during namespace creation are executed against runtime environments

**Independent Test**: Create namespace for a group with multiple environments; verify provisioning executes on each

### Tests for User Story 5

- [x] T065 [P] [US5] Integration test for operation queue processing in `tests/integration/queue_test.go` (queue operation, process it, verify REI called)
- [x] T066 [P] [US5] Integration test for Cloud Foundry REII in `tests/integration/cloudfoundry_test.go` (mock CF API, verify org/space creation)

### Implementation for User Story 5

- [x] T067 [P] [US5] Implement Cloud Foundry REII in `internal/rei/cloudfoundry/cloudfoundry.go` (implements pkg/rei.RuntimeEnvironment — CreateNamespace creates org+space, HealthCheck calls /v3/info)
- [x] T068 [P] [US5] Implement CF namespace mapping store in `internal/store/postgres/cf_mapping.go` (CreateMapping, GetMapping for CFNamespaceMapping table)
- [x] T069 [US5] Implement operation queue processor in `internal/queue/processor.go` (polls pending operations, processes via REI, updates status, handles failures with retry)
- [x] T070 [US5] Implement PostgreSQL LISTEN/NOTIFY for new operations in `internal/queue/notifier.go` (notify on insert, processor listens for immediate processing)
- [x] T071 [US5] Implement environment health checker in `internal/queue/health.go` (periodic health check via REI.HealthCheck, updates environment status in DB)
- [x] T072 [US5] Register Cloud Foundry factory in REI Registry at application startup in `cmd/papi/main.go`
- [x] T073 [US5] Start queue processor goroutine in `cmd/papi/main.go` (graceful shutdown)
- [x] T074 [US5] Add structured logging and metrics for queue processing (operation type, environment, duration, success/failure counts)

**Checkpoint**: End-to-end provisioning operational — namespaces created in PAPI are provisioned in Cloud Foundry

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [x] T075 [P] Implement identity group cache background refresh worker in `internal/queue/identity_refresh.go` (periodic TTL check, re-resolve from identity provider, update cache)
- [x] T076 [P] Add request ID / correlation ID middleware in `internal/api/v1/middleware/request_id.go` (generate UUID, propagate via context, include in logs)
- [x] T077 [P] Add rate limiting middleware in `internal/api/v1/middleware/rate_limit.go`
- [x] T078 [P] Add pagination helper in `internal/api/v1/handlers/pagination.go` (parse page/page_size params, build Pagination response)
- [ ] T079 Run quickstart.md validation (manual smoke test against running instance)
- [x] T080 [P] Add OpenTelemetry trace spans to all handlers and queue processor
- [x] T081 [P] Create `docker-compose.yml` for local development (PostgreSQL, UAA mock)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational
- **User Story 2 (Phase 4)**: Depends on Foundational (can run in parallel with US1)
- **User Story 3 (Phase 5)**: Depends on Foundational (can run in parallel with US1/US2)
- **User Story 4 (Phase 6)**: Depends on Foundational (can run in parallel with US1/US2/US3)
- **User Story 5 (Phase 7)**: Depends on US2 (needs operations queued) + US4 (needs groups created)
- **Polish (Phase 8)**: Depends on all user stories complete

### User Story Dependencies

- **US1 (List Namespaces)**: Independent after Foundational
- **US2 (Create Namespace)**: Independent after Foundational (operations are queued but not processed until US5)
- **US3 (Health/Info)**: Independent after Foundational
- **US4 (Group CRUD)**: Independent after Foundational
- **US5 (Provisioning)**: Depends on US2 + US4 (needs both operation queue and group/environment data)

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Models before stores
- Stores before services
- Services before handlers
- Handlers before route registration

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel
- US1, US2, US3, US4 can all start in parallel after Foundational
- Within each story: tests [P], models [P] can run in parallel
- US5 must wait for US2 + US4

---

## Implementation Strategy

### MVP First (User Stories 1 + 2)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL — blocks all stories)
3. Complete Phase 3: User Story 1 (List Namespaces)
4. Complete Phase 4: User Story 2 (Create Namespace)
5. **STOP and VALIDATE**: Both P1 stories independently functional
6. Deploy/demo if ready (namespace CRUD works, provisioning is queued but not processed)

### Full Delivery

7. Complete Phase 5: User Story 3 (Health/Info)
8. Complete Phase 6: User Story 4 (Group CRUD)
9. Complete Phase 7: User Story 5 (Provisioning — connects everything end-to-end)
10. Complete Phase 8: Polish
11. Run quickstart.md validation

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- The REI interface (T021) is in Foundational because all stories eventually need it
- Operation queue processing (US5) is separate from creation (US2) — this is intentional for async decoupling
