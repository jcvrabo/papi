-- PAPI Initial Schema

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enums
CREATE TYPE namespace_status AS ENUM ('active', 'provisioning', 'failed');
CREATE TYPE member_type AS ENUM ('user', 'identity_group');
CREATE TYPE member_role AS ENUM ('read', 'write', 'read_write');
CREATE TYPE orchestration_strategy AS ENUM ('replicate', 'canary');
CREATE TYPE environment_health AS ENUM ('healthy', 'degraded', 'unavailable', 'maintenance');
CREATE TYPE operation_status AS ENUM ('pending', 'in_progress', 'completed', 'failed');
CREATE TYPE operation_type AS ENUM ('create_namespace', 'delete_namespace');
CREATE TYPE platform_role AS ENUM ('admin', 'namespace_creator');

-- Runtime Environment Groups
CREATE TABLE runtime_environment_groups (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT UNIQUE NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT false,
    orchestration_strategy orchestration_strategy NOT NULL DEFAULT 'replicate',
    canary_environment_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Runtime Environments
CREATE TABLE runtime_environments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    group_id UUID NOT NULL REFERENCES runtime_environment_groups(id),
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    connection_config JSONB NOT NULL,
    health_status environment_health NOT NULL DEFAULT 'healthy',
    last_health_check_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(group_id, name)
);

-- Add FK for canary after environments table exists
ALTER TABLE runtime_environment_groups
    ADD CONSTRAINT fk_canary_environment
    FOREIGN KEY (canary_environment_id) REFERENCES runtime_environments(id);

-- Namespaces
CREATE TABLE namespaces (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    composite_name TEXT UNIQUE NOT NULL,
    name_components JSONB NOT NULL,
    metadata JSONB,
    runtime_environment_group_id UUID NOT NULL REFERENCES runtime_environment_groups(id),
    status namespace_status NOT NULL DEFAULT 'provisioning',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Members
CREATE TABLE members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    namespace_id UUID NOT NULL REFERENCES namespaces(id) ON DELETE CASCADE,
    member_type member_type NOT NULL,
    member_identifier TEXT NOT NULL,
    role member_role NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(namespace_id, member_type, member_identifier)
);

-- Identity Group Cache
CREATE TABLE identity_group_cache (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    group_identifier TEXT UNIQUE NOT NULL,
    resolved_members JSONB NOT NULL DEFAULT '[]',
    last_refreshed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ttl_seconds INTEGER NOT NULL DEFAULT 300
);

-- Operations Queue
CREATE TABLE operations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    namespace_id UUID NOT NULL REFERENCES namespaces(id) ON DELETE CASCADE,
    runtime_environment_id UUID NOT NULL REFERENCES runtime_environments(id),
    operation_type operation_type NOT NULL,
    status operation_status NOT NULL DEFAULT 'pending',
    idempotency_key TEXT UNIQUE NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    attempts INTEGER NOT NULL DEFAULT 0,
    last_attempted_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Namespace Name Template
CREATE TABLE namespace_name_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    template_pattern TEXT NOT NULL,
    segments JSONB NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Metadata Rules
CREATE TABLE metadata_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    field_name TEXT UNIQUE NOT NULL,
    is_mandatory BOOLEAN NOT NULL,
    validation_regex TEXT,
    allowed_values JSONB,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Platform User Roles
CREATE TABLE platform_user_roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_identifier TEXT NOT NULL,
    role platform_role NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_identifier, role)
);

-- Cloud Foundry Extension
CREATE TABLE cf_namespace_mappings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    namespace_id UUID NOT NULL REFERENCES namespaces(id) ON DELETE CASCADE,
    runtime_environment_id UUID NOT NULL REFERENCES runtime_environments(id),
    cf_org_name TEXT NOT NULL,
    cf_space_name TEXT NOT NULL,
    cf_org_guid TEXT,
    cf_space_guid TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(namespace_id, runtime_environment_id)
);

-- Indexes
CREATE INDEX idx_members_namespace_id ON members(namespace_id);
CREATE INDEX idx_members_identifier ON members(member_identifier, member_type);
CREATE INDEX idx_operations_status_env ON operations(status, runtime_environment_id);
CREATE INDEX idx_operations_namespace ON operations(namespace_id);
CREATE INDEX idx_identity_cache_identifier ON identity_group_cache(group_identifier);
CREATE INDEX idx_platform_roles_user ON platform_user_roles(user_identifier);
