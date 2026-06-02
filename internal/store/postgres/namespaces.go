package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rabobank/papi/internal/domain/namespace"
)

type NamespaceStore struct {
	pool *pgxpool.Pool
}

func NewNamespaceStore(pool *pgxpool.Pool) *NamespaceStore {
	return &NamespaceStore{pool: pool}
}

func (s *NamespaceStore) ListByUser(ctx context.Context, userIdentifier string, page, pageSize int) ([]namespace.Namespace, int, error) {
	var total int
	countQuery := `
		SELECT COUNT(DISTINCT n.id) FROM namespaces n
		LEFT JOIN members m ON m.namespace_id = n.id
		LEFT JOIN identity_group_cache igc ON igc.group_identifier = m.member_identifier AND m.member_type = 'identity_group'
		WHERE (m.member_type = 'user' AND m.member_identifier = $1)
		   OR (m.member_type = 'identity_group' AND igc.resolved_members @> to_jsonb($1::text))
	`
	if err := s.pool.QueryRow(ctx, countQuery, userIdentifier).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query := `
		SELECT DISTINCT n.id, n.composite_name, n.name_components, n.metadata,
		       n.runtime_environment_group_id, n.status, n.created_at, n.updated_at
		FROM namespaces n
		LEFT JOIN members m ON m.namespace_id = n.id
		LEFT JOIN identity_group_cache igc ON igc.group_identifier = m.member_identifier AND m.member_type = 'identity_group'
		WHERE (m.member_type = 'user' AND m.member_identifier = $1)
		   OR (m.member_type = 'identity_group' AND igc.resolved_members @> to_jsonb($1::text))
		ORDER BY n.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := s.pool.Query(ctx, query, userIdentifier, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var namespaces []namespace.Namespace
	for rows.Next() {
		var ns namespace.Namespace
		var componentsJSON, metadataJSON []byte
		if err := rows.Scan(&ns.ID, &ns.CompositeName, &componentsJSON, &metadataJSON,
			&ns.RuntimeEnvironmentGroupID, &ns.Status, &ns.CreatedAt, &ns.UpdatedAt); err != nil {
			return nil, 0, err
		}
		json.Unmarshal(componentsJSON, &ns.NameComponents)
		if metadataJSON != nil {
			json.Unmarshal(metadataJSON, &ns.Metadata)
		}
		namespaces = append(namespaces, ns)
	}
	return namespaces, total, rows.Err()
}

func (s *NamespaceStore) NameExists(ctx context.Context, compositeName string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM namespaces WHERE composite_name = $1)",
		compositeName).Scan(&exists)
	return exists, err
}

func (s *NamespaceStore) Create(ctx context.Context, ns *namespace.Namespace, members []namespace.Member) error {
	componentsJSON, _ := json.Marshal(ns.NameComponents)
	metadataJSON, _ := json.Marshal(ns.Metadata)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO namespaces (id, composite_name, name_components, metadata, runtime_environment_group_id, status)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		ns.ID, ns.CompositeName, componentsJSON, metadataJSON, ns.RuntimeEnvironmentGroupID, ns.Status)
	if err != nil {
		return err
	}

	for _, m := range members {
		_, err = tx.Exec(ctx,
			`INSERT INTO members (id, namespace_id, member_type, member_identifier, role)
			 VALUES ($1, $2, $3, $4, $5)`,
			m.ID, m.NamespaceID, m.MemberType, m.MemberIdentifier, m.Role)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
