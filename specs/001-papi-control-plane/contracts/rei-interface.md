# REI - Runtime Environment Interface Contract

**Date**: 2026-05-28
**Package**: `pkg/rei`

## Overview

The REI defines the Go interface that any Runtime Environment Interface Implementation (REII)
must satisfy. It is published as an importable Go package so third-party implementations can
depend on it.

## Interface Definition

```go
package rei

import "context"

// RuntimeEnvironment represents the interface that all REII implementations must satisfy.
// Each method corresponds to a platform operation that PAPI delegates to the runtime.
type RuntimeEnvironment interface {
    // CreateNamespace provisions the runtime-specific equivalent of a PAPI namespace.
    // For Cloud Foundry, this creates an org/space pair.
    // Returns a ProvisionResult with runtime-specific identifiers.
    CreateNamespace(ctx context.Context, req CreateNamespaceRequest) (CreateNamespaceResult, error)

    // DeleteNamespace removes the runtime-specific equivalent of a PAPI namespace.
    DeleteNamespace(ctx context.Context, req DeleteNamespaceRequest) error

    // HealthCheck verifies connectivity and operational status of the runtime environment.
    HealthCheck(ctx context.Context) (HealthStatus, error)
}

// CreateNamespaceRequest contains the information needed to provision a namespace
// in a runtime environment.
type CreateNamespaceRequest struct {
    // NamespaceID is the PAPI-internal namespace identifier.
    NamespaceID string

    // CompositeName is the resolved PAPI namespace name.
    CompositeName string

    // NameComponents are the individual template segment values.
    NameComponents map[string]string

    // Metadata contains namespace tags/labels.
    Metadata map[string]string
}

// CreateNamespaceResult contains runtime-specific identifiers created during provisioning.
// Implementations store additional data via the ExtensionData field.
type CreateNamespaceResult struct {
    // ExtensionData contains runtime-specific key-value data to be persisted
    // (e.g., cf_org_guid, cf_space_guid for Cloud Foundry).
    ExtensionData map[string]string
}

// DeleteNamespaceRequest contains the information needed to deprovision a namespace.
type DeleteNamespaceRequest struct {
    NamespaceID   string
    CompositeName string
    // ExtensionData contains runtime-specific identifiers from the original provisioning.
    ExtensionData map[string]string
}

// HealthStatus represents the health of a runtime environment.
type HealthStatus struct {
    Status  Status
    Message string
}

// Status represents health states.
type Status string

const (
    StatusHealthy     Status = "healthy"
    StatusDegraded    Status = "degraded"
    StatusUnavailable Status = "unavailable"
)
```

## REII Registration

REII implementations register themselves via a factory function:

```go
package rei

// Factory creates a RuntimeEnvironment instance from connection configuration.
type Factory func(connectionConfig map[string]interface{}) (RuntimeEnvironment, error)

// Registry maps runtime environment type names to their factories.
// The PAPI application populates this at startup.
var Registry = map[string]Factory{}
```

## Cloud Foundry REII Contract

The bundled Cloud Foundry implementation satisfies `rei.RuntimeEnvironment` and:

- Maps `CreateNamespace` → create CF org + create CF space within that org
- Derives org name and space name from namespace `NameComponents` (configurable mapping)
- Returns `cf_org_guid` and `cf_space_guid` in `ExtensionData`
- `HealthCheck` → calls CF `/v3/info` endpoint
- Requires `connection_config`: `{"api_url": "...", "client_id": "...", "client_secret": "..."}`
