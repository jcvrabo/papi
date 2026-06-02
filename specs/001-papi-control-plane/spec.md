# Feature Specification: PAPI Control Plane

**Feature Branch**: `001-namespace-handling`
**Created**: 2026-05-28
**Status**: Draft
**Input**: User description: "develop PAPI (Platform API) in GO, an umbrella control plane which abstracts interactions with runtime environments..."

## Clarifications

### Session 2026-05-28

- Q: How is a namespace associated with a runtime environment group at creation? → A: A system-wide default group is auto-assigned; user may override with an explicit group selection.
- Q: What determines the composite namespace name structure? → A: Admin-configured template with segments (e.g., `{team}-{project}-{env}`) with optional validation rules per component.
- Q: What happens when group operations fail on some environments? → A: PAPI persists the intent, responds with IN_PROGRESS, and queues execution per environment. Unavailable environments receive the operation once they become healthy again (eventual consistency with internal queue).
- Q: Who is authorized to create namespaces? → A: Only users with a specific platform-level role ("namespace-creator") can create namespaces, separate from namespace-level read/write roles.
- Q: How does PAPI track identity group membership? → A: Store group reference AND cache resolved members with TTL-based refresh.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View Available Namespaces (Priority: P1)

As an authenticated platform user, I want to list namespaces I have access to so that I can see which environments and workspaces are available to me.

**Why this priority**: Viewing existing namespaces is the most fundamental operation — users need visibility into their available workspaces before they can take any action.

**Independent Test**: Can be fully tested by authenticating and calling the list endpoint; delivers immediate value by showing the user their available namespaces.

**Acceptance Scenarios**:

1. **Given** an authenticated user with read access to 3 namespaces, **When** they request the list of namespaces, **Then** they receive only the 3 namespaces they have access to with name and metadata.
2. **Given** an authenticated user with no namespace memberships, **When** they request the list of namespaces, **Then** they receive an empty list.
3. **Given** an unauthenticated request, **When** the list endpoint is called, **Then** the system rejects the request with an authentication error.
4. **Given** an authenticated user with read access to some namespaces and write access to others, **When** they request the list, **Then** all namespaces where they hold any role are returned.
5. **Given** a namespace with a member specified by identity group, **When** the cached member list includes the requesting user, **Then** that namespace is included in their list without querying the identity provider in real-time.

---

### User Story 2 - Create a New Namespace (Priority: P1)

As an authenticated platform user with the "namespace-creator" platform role, I want to create a new namespace with members so that my team can begin working in the platform's runtime environments.

**Why this priority**: Creating namespaces is the primary write operation and the gateway to all further platform usage. Without namespace creation, the platform provides no value beyond read-only visibility.

**Independent Test**: Can be fully tested by authenticating, submitting a namespace creation request with template-compliant name components and members, and verifying the namespace exists in the platform with an IN_PROGRESS provisioning status.

**Acceptance Scenarios**:

1. **Given** an authenticated user with the "namespace-creator" role, **When** they submit a valid namespace creation request with name components matching the configured template and at least one member, **Then** the namespace is created in PAPI, the system responds with IN_PROGRESS status, and provisioning is queued for all runtime environments in the assigned group.
2. **Given** a namespace creation request without specifying a runtime environment group, **When** submitted, **Then** the system-wide default group is automatically assigned.
3. **Given** a namespace creation request with an explicit group override, **When** submitted, **Then** the specified group is assigned instead of the default.
4. **Given** a namespace creation request with mandatory metadata fields missing, **When** the system has configured mandatory metadata rules, **Then** the request is rejected with a validation error listing the missing fields.
5. **Given** a namespace creation request with name components that fail per-component validation rules, **When** submitted, **Then** the request is rejected with specific validation errors per component.
6. **Given** a namespace creation request with optional metadata (tags/labels), **When** submitted, **Then** the namespace is created with the provided metadata attached.
7. **Given** a namespace creation request where the composite name already exists, **When** submitted, **Then** the system rejects it with a conflict error.
8. **Given** a namespace creation request with members specified by group name (e.g., Azure Entra ID group), **When** submitted, **Then** the group reference is stored and members are resolved and cached.
9. **Given** a user without the "namespace-creator" role, **When** they attempt to create a namespace, **Then** the request is rejected with an authorization error.
10. **Given** a successful namespace creation where a runtime environment is unavailable, **When** the environment becomes healthy, **Then** the queued provisioning operation is automatically executed.

---

### User Story 3 - Discover Platform Capabilities (Priority: P2)

As a platform consumer (human or system), I want to query the platform's health status and basic information (version, authentication endpoint) so that I can determine how to interact with the platform without prior configuration knowledge.

**Why this priority**: Discovery and health checking are prerequisites for any automated integration and operational monitoring, but users can still use the platform manually without this.

**Independent Test**: Can be tested by calling unauthenticated info/health endpoints and verifying version data and OIDC endpoint are returned.

**Acceptance Scenarios**:

1. **Given** the platform is running, **When** the health endpoint is called without authentication, **Then** a health status is returned indicating the platform is operational.
2. **Given** the platform is running, **When** the info endpoint is called without authentication, **Then** version information and the OIDC authentication endpoint URL are returned.
3. **Given** the platform's connection to a runtime environment is degraded, **When** the health endpoint is called, **Then** the response indicates degraded status with details.

---

### User Story 4 - Manage Runtime Environment Groups (Priority: P2)

As an authenticated user, I want to view available runtime environment groups, and as a platform administrator, I want to create, update, and delete groups so that I can organize runtime environments and control where namespaces are provisioned.

**Why this priority**: Runtime environment groups are a prerequisite for namespace provisioning orchestration. Administrators need CRUD operations to configure the platform, and users need visibility to select groups during namespace creation.

**Independent Test**: Can be tested by authenticating as an admin, creating a group with environments, updating its configuration, and verifying authenticated users can list groups.

**Acceptance Scenarios**:

1. **Given** an authenticated user (any role), **When** they request the list of runtime environment groups, **Then** they receive all available groups with their names, member environments, and default designation.
2. **Given** a platform administrator, **When** they submit a valid group creation request with a name and at least one runtime environment, **Then** the group is created.
3. **Given** a platform administrator, **When** they update a group's configuration (add/remove environments, change orchestration strategy), **Then** the changes are persisted.
4. **Given** a platform administrator, **When** they delete a group that has no namespaces assigned, **Then** the group is removed.
5. **Given** a platform administrator, **When** they attempt to delete a group that has namespaces assigned, **Then** the request is rejected with an error indicating active dependencies.
6. **Given** a non-admin authenticated user, **When** they attempt to create, update, or delete a group, **Then** the request is rejected with an authorization error.

---

### User Story 5 - Namespace Provisioning Across Runtime Environment Groups (Priority: P2)

As a platform administrator, I want namespace operations to be applied across all runtime environments in a group so that I do not need to manually replicate actions in each environment.

**Why this priority**: Group-level orchestration is what differentiates this platform from managing individual environments. It is critical for operational efficiency but depends on basic namespace operations (P1) working first.

**Independent Test**: Can be tested by creating a namespace assigned to a group with multiple runtime environments and verifying the namespace concept is eventually provisioned in each.

**Acceptance Scenarios**:

1. **Given** a runtime environment group with 3 environments, **When** a namespace is created for that group, **Then** PAPI queues provisioning for all 3 environments and the corresponding concept (e.g., org/space in Cloud Foundry) is created in each as they are processed.
2. **Given** a runtime environment group with an orchestrated deployment strategy (canary), **When** a deployment action is triggered, **Then** the action is first applied to the canary environment and only proceeds to the remaining environments upon success.
3. **Given** a group operation where one environment is unavailable (maintenance or recovery), **When** the operation is queued, **Then** it is held for that environment and executed automatically when the environment becomes healthy.
4. **Given** a queued operation, **When** a user queries the namespace status, **Then** per-environment provisioning status is reported (e.g., completed, in_progress, pending).

---

### Edge Cases

- How does the system handle concurrent namespace creation requests with the same composite name?
- What happens when a member specified by group name resolves to zero members?
- What happens when mandatory metadata validation rules are changed after namespaces already exist without those fields?
- How does the system behave when the identity provider is unavailable for token validation?
- What happens when the cached identity group membership TTL expires and the identity provider is temporarily unreachable?
- What happens when a queued operation becomes invalid (e.g., the namespace is deleted before the environment becomes available)?
- What happens when the admin-configured naming template is changed — are existing namespaces grandfathered?
- What happens when an administrator removes a runtime environment from a group that has active namespaces provisioned on it?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST expose a versioned REST API (starting at v1) for all platform operations
- **FR-002**: System MUST provide an unauthenticated health check endpoint returning operational status
- **FR-003**: System MUST provide an unauthenticated info endpoint returning version and OIDC authentication endpoint
- **FR-004**: System MUST require bearer token authentication for all endpoints except health and info
- **FR-005**: System MUST enforce authorization based on token claims and user-namespace permissions (read/write)
- **FR-006**: System MUST enforce a platform-level "namespace-creator" role for namespace creation, separate from namespace-level permissions
- **FR-007**: System MUST allow authenticated users to list namespaces they have access to (any role)
- **FR-008**: System MUST allow users with the "namespace-creator" role to create namespaces with name components and initial members
- **FR-009**: System MUST initialize all non-required namespace attributes with default values on creation
- **FR-010**: System MUST support admin-configured naming templates with segments (e.g., `{team}-{project}-{env}`) for composite namespace names
- **FR-011**: System MUST support optional per-component validation rules for namespace name templates
- **FR-012**: System MUST reject namespace creation if name components fail their configured validation rules
- **FR-013**: System MUST support freeform metadata (tags/labels) on namespaces
- **FR-014**: System MUST support configurable mandatory metadata fields with validation rules
- **FR-015**: System MUST reject namespace creation if mandatory metadata fields are missing or fail validation
- **FR-016**: System MUST support membership by individual username or identity group name
- **FR-017**: System MUST store identity group references and cache resolved members with configurable TTL-based refresh
- **FR-018**: System MUST translate namespace operations into corresponding runtime environment concepts via a pluggable interface
- **FR-019**: System MUST support multiple runtime environments grouped together
- **FR-020**: System MUST allow any authenticated user to list runtime environment groups
- **FR-021**: System MUST allow platform administrators to create runtime environment groups with a name and member environments
- **FR-022**: System MUST allow platform administrators to update runtime environment groups (add/remove environments, change orchestration strategy)
- **FR-023**: System MUST allow platform administrators to delete runtime environment groups that have no active namespace assignments
- **FR-024**: System MUST reject deletion of runtime environment groups that have namespaces assigned
- **FR-025**: System MUST assign a system-wide default runtime environment group to namespaces when no group is explicitly specified
- **FR-026**: System MUST allow users to override the default group with an explicit group selection at namespace creation
- **FR-027**: System MUST process group operations asynchronously — persisting intent, responding with IN_PROGRESS, and queuing per-environment execution
- **FR-028**: System MUST automatically execute queued operations on environments once they become healthy/available
- **FR-029**: System MUST support orchestrated deployment strategies (e.g., canary) for group actions
- **FR-030**: System MUST persist all state in a relational database
- **FR-031**: System MUST support extensible storage for runtime-environment-specific configurations
- **FR-032**: System MUST support any OIDC/OAuth2 identity provider for authentication
- **FR-033**: System MUST be bundled with a runtime environment implementation for Cloud Foundry in v1
- **FR-034**: System MUST define a runtime environment interface as a reusable specification that any implementation can satisfy
- **FR-035**: System MUST report per-environment provisioning status for namespace operations

### API Requirements *(include if feature involves REST endpoints)*

- **API-001**: API version: v1 (initial version)
- **API-002**: Endpoints follow versioning scheme `/api/v1/...`
- **API-003**: Future breaking changes (e.g., restructuring namespace model, changing auth flow) require v2
- **API-004**: Additive changes (new optional fields, new endpoints) remain within v1
- **API-005**: OpenAPI contract specification required for all endpoints
- **API-006**: Contract tests required before implementation

### Key Entities

- **Namespace**: The primary organizational unit in PAPI. Has a composite name built from an admin-configured template with per-component validation, metadata (tags/labels), members, and is associated with a runtime environment group (default or explicit). Represents a logical workspace that maps to platform-specific concepts (e.g., org/space in Cloud Foundry).
- **Namespace Name Template**: An admin-configured template defining the segments that compose a namespace name (e.g., `{team}-{project}-{env}`). Each segment may have optional validation rules (regex patterns, allowed values, etc.).
- **Member**: A user or identity group with a role (read, write, or both) within a namespace. Can be specified by username or identity group name. Identity groups are stored as references with cached resolved members (TTL-based refresh).
- **Runtime Environment**: A target platform (e.g., a Cloud Foundry instance) that PAPI manages through the Runtime Environment Interface. Has connection details, health status, and runtime-specific configuration.
- **Runtime Environment Group**: A logical grouping of one or more runtime environments. Has a designation as system-wide default (one group). Actions on a group are applied to all member environments asynchronously via an internal operation queue.
- **Operation Queue**: Internal queue tracking pending operations per environment. Operations are held for unavailable environments and executed when the environment becomes healthy.
- **Runtime Environment Interface (REI)**: A contract specifying all operations a runtime environment must support (create namespace, deploy application, etc.).
- **Runtime Environment Interface Implementation (REII)**: A concrete implementation of the REI for a specific platform (e.g., Cloud Foundry).
- **Metadata Configuration**: System-level rules defining mandatory metadata fields and their validation rules for namespaces.
- **Platform Role**: A system-wide permission (e.g., "namespace-creator", "admin") that governs platform-level actions, separate from namespace-level read/write permissions. Admins can manage runtime environment groups.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Authenticated users can list their accessible namespaces within 2 seconds for up to 1000 namespaces
- **SC-002**: Namespace creation request is accepted and IN_PROGRESS response returned within 2 seconds
- **SC-003**: Queued operations are executed on available environments within 10 seconds of being dequeued
- **SC-004**: Platform health and info endpoints respond within 500 milliseconds without authentication
- **SC-005**: System correctly enforces access control — users never see namespaces they have no role in (0% unauthorized access)
- **SC-006**: Users without "namespace-creator" role are rejected 100% of the time when attempting creation
- **SC-007**: Namespace creation with invalid mandatory metadata or invalid name components is rejected 100% of the time with actionable error messages
- **SC-008**: Adding a new runtime environment type requires only implementing the interface contract — no changes to core platform logic
- **SC-009**: Platform supports at least 100 concurrent authenticated users without degradation
- **SC-010**: All namespace operations are eventually executed across all environments in a group (0% drift once all environments are healthy)
- **SC-011**: Cached identity group memberships are refreshed within the configured TTL window

## Assumptions

- Users authenticate via an external OIDC/OAuth2 provider; PAPI does not manage user credentials directly
- The initial deployment will use a UAA server as the identity provider, but the design supports any compliant provider
- Namespace composite names are unique within the platform (no two namespaces share the same resolved composite name)
- Exactly one runtime environment group is designated as the system-wide default
- Runtime environment groups are preconfigured by administrators; end users may override the default at namespace creation
- The Cloud Foundry REII maps a single PAPI namespace to one Cloud Foundry organization/space pair
- Database extensions for runtime-specific data follow a convention that allows the core schema to remain stable
- Canary and other orchestrated deployment strategies are configurable per group and are not hard-coded
- Identity group references are stored persistently; resolved member lists are cached with a configurable TTL (default to be determined at planning)
- Rate limiting and abuse prevention follow standard practices for enterprise APIs
- Naming templates and per-component validation rules are configured at the system level by administrators and apply to all new namespace creations
- Existing namespaces are grandfathered when naming templates change (validation only applies at creation time)
