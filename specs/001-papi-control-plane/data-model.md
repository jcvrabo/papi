# Data Model: PAPI Control Plane

**Date**: 2026-05-28
**Source**: spec.md, research.md

## Entity Relationship Overview

```
Namespace ──────── hasManyMembers ──────── Member
    │                                         │
    │ belongsTo                               │ referencesOptional
    ▼                                         ▼
RuntimeEnvironmentGroup              IdentityGroupCache
    │
    │ hasMany
    ▼
RuntimeEnvironment
    │
    │ hasMany
    ▼
Operation (queue)

NamespaceNameTemplate (system-level, one active)
MetadataConfiguration (system-level, many rules)
PlatformRole (system-level)
```

## Entities

### Namespace

| Field | Type | Constraints |
|-------|------|-------------|
| id | UUID | PK, auto-generated |
| composite_name | TEXT | UNIQUE, NOT NULL, built from template |
| name_components | JSONB | NOT NULL, stores individual template segment values |
| metadata | JSONB | nullable, freeform tags/labels |
| runtime_environment_group_id | UUID | FK → RuntimeEnvironmentGroup, NOT NULL |
| status | ENUM | NOT NULL: active, provisioning, failed |
| created_at | TIMESTAMPTZ | NOT NULL, auto |
| updated_at | TIMESTAMPTZ | NOT NULL, auto |

**State transitions**: provisioning → active | failed

### Member

| Field | Type | Constraints |
|-------|------|-------------|
| id | UUID | PK, auto-generated |
| namespace_id | UUID | FK → Namespace, NOT NULL |
| member_type | ENUM | NOT NULL: user, identity_group |
| member_identifier | TEXT | NOT NULL (username or group name) |
| role | ENUM | NOT NULL: read, write, read_write |
| created_at | TIMESTAMPTZ | NOT NULL, auto |

**Unique constraint**: (namespace_id, member_type, member_identifier)

### IdentityGroupCache

| Field | Type | Constraints |
|-------|------|-------------|
| id | UUID | PK, auto-generated |
| group_identifier | TEXT | UNIQUE, NOT NULL (e.g., Azure Entra ID group name) |
| resolved_members | JSONB | NOT NULL, array of usernames |
| last_refreshed_at | TIMESTAMPTZ | NOT NULL |
| ttl_seconds | INTEGER | NOT NULL, from system config |

### RuntimeEnvironmentGroup

| Field | Type | Constraints |
|-------|------|-------------|
| id | UUID | PK, auto-generated |
| name | TEXT | UNIQUE, NOT NULL |
| is_default | BOOLEAN | NOT NULL, default false (exactly one true) |
| orchestration_strategy | ENUM | NOT NULL: replicate, canary |
| canary_environment_id | UUID | nullable, FK → RuntimeEnvironment (when strategy=canary) |
| created_at | TIMESTAMPTZ | NOT NULL, auto |
| updated_at | TIMESTAMPTZ | NOT NULL, auto |

### RuntimeEnvironment

| Field | Type | Constraints |
|-------|------|-------------|
| id | UUID | PK, auto-generated |
| group_id | UUID | FK → RuntimeEnvironmentGroup, NOT NULL |
| name | TEXT | NOT NULL |
| type | TEXT | NOT NULL (e.g., "cloudfoundry") |
| connection_config | JSONB | NOT NULL, encrypted at rest |
| health_status | ENUM | NOT NULL: healthy, degraded, unavailable, maintenance |
| last_health_check_at | TIMESTAMPTZ | nullable |
| created_at | TIMESTAMPTZ | NOT NULL, auto |
| updated_at | TIMESTAMPTZ | NOT NULL, auto |

**Unique constraint**: (group_id, name)

### Operation

| Field | Type | Constraints |
|-------|------|-------------|
| id | UUID | PK, auto-generated |
| namespace_id | UUID | FK → Namespace, NOT NULL |
| runtime_environment_id | UUID | FK → RuntimeEnvironment, NOT NULL |
| operation_type | ENUM | NOT NULL: create_namespace, delete_namespace |
| status | ENUM | NOT NULL: pending, in_progress, completed, failed |
| idempotency_key | TEXT | UNIQUE, NOT NULL |
| payload | JSONB | NOT NULL, operation-specific data |
| attempts | INTEGER | NOT NULL, default 0 |
| last_attempted_at | TIMESTAMPTZ | nullable |
| error_message | TEXT | nullable |
| created_at | TIMESTAMPTZ | NOT NULL, auto |
| updated_at | TIMESTAMPTZ | NOT NULL, auto |

**State transitions**: pending → in_progress → completed | failed → pending (retry)

### NamespaceNameTemplate

| Field | Type | Constraints |
|-------|------|-------------|
| id | UUID | PK, auto-generated |
| template_pattern | TEXT | NOT NULL (e.g., "{team}-{project}-{env}") |
| segments | JSONB | NOT NULL, array of {name, required, validation_regex, allowed_values} |
| is_active | BOOLEAN | NOT NULL, default true (exactly one active) |
| created_at | TIMESTAMPTZ | NOT NULL, auto |

### MetadataRule

| Field | Type | Constraints |
|-------|------|-------------|
| id | UUID | PK, auto-generated |
| field_name | TEXT | UNIQUE, NOT NULL |
| is_mandatory | BOOLEAN | NOT NULL |
| validation_regex | TEXT | nullable |
| allowed_values | JSONB | nullable, array of strings |
| description | TEXT | nullable |
| created_at | TIMESTAMPTZ | NOT NULL, auto |

### PlatformUserRole

| Field | Type | Constraints |
|-------|------|-------------|
| id | UUID | PK, auto-generated |
| user_identifier | TEXT | NOT NULL (username from OIDC token) |
| role | ENUM | NOT NULL: admin, namespace_creator |
| created_at | TIMESTAMPTZ | NOT NULL, auto |

**Unique constraint**: (user_identifier, role)

## REII Extension Table (Cloud Foundry)

### CFNamespaceMapping

| Field | Type | Constraints |
|-------|------|-------------|
| id | UUID | PK, auto-generated |
| namespace_id | UUID | FK → Namespace, NOT NULL |
| runtime_environment_id | UUID | FK → RuntimeEnvironment, NOT NULL |
| cf_org_name | TEXT | NOT NULL |
| cf_space_name | TEXT | NOT NULL |
| cf_org_guid | TEXT | nullable (populated after provisioning) |
| cf_space_guid | TEXT | nullable (populated after provisioning) |
| created_at | TIMESTAMPTZ | NOT NULL, auto |

**Unique constraint**: (namespace_id, runtime_environment_id)

## Indexes

- `namespace.composite_name` — unique index for conflict detection
- `member.namespace_id` — for listing members of a namespace
- `member.member_identifier, member.member_type` — for finding namespaces a user belongs to
- `operation.status, operation.runtime_environment_id` — for queue processing
- `operation.namespace_id` — for status reporting
- `identity_group_cache.group_identifier` — for fast lookup
- `platform_user_role.user_identifier` — for authorization checks
