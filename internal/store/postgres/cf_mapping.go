package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CFNamespaceMapping struct {
	ID                   string
	NamespaceID          string
	RuntimeEnvironmentID string
	CFOrgName            string
	CFSpaceName          string
	CFOrgGUID            string
	CFSpaceGUID          string
}

type CFMappingStore struct {
	pool *pgxpool.Pool
}

func NewCFMappingStore(pool *pgxpool.Pool) *CFMappingStore {
	return &CFMappingStore{pool: pool}
}

func (s *CFMappingStore) Create(ctx context.Context, m *CFNamespaceMapping) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO cf_namespace_mappings (namespace_id, runtime_environment_id, cf_org_name, cf_space_name, cf_org_guid, cf_space_guid)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		m.NamespaceID, m.RuntimeEnvironmentID, m.CFOrgName, m.CFSpaceName, m.CFOrgGUID, m.CFSpaceGUID)
	return err
}

func (s *CFMappingStore) GetByNamespaceAndEnv(ctx context.Context, namespaceID, envID string) (*CFNamespaceMapping, error) {
	var m CFNamespaceMapping
	err := s.pool.QueryRow(ctx,
		`SELECT id, namespace_id, runtime_environment_id, cf_org_name, cf_space_name, cf_org_guid, cf_space_guid
		 FROM cf_namespace_mappings WHERE namespace_id = $1 AND runtime_environment_id = $2`,
		namespaceID, envID).Scan(&m.ID, &m.NamespaceID, &m.RuntimeEnvironmentID, &m.CFOrgName, &m.CFSpaceName, &m.CFOrgGUID, &m.CFSpaceGUID)
	return &m, err
}
