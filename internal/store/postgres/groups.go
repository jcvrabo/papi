package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rabobank/papi/internal/domain/group"
)

type GroupStore struct {
	pool *pgxpool.Pool
}

func NewGroupStore(pool *pgxpool.Pool) *GroupStore {
	return &GroupStore{pool: pool}
}

func (s *GroupStore) List(ctx context.Context) ([]group.Group, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, is_default, orchestration_strategy, canary_environment_id, created_at, updated_at 
         FROM runtime_environment_groups ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []group.Group
	for rows.Next() {
		var g group.Group
		var canaryID *string
		if err := rows.Scan(&g.ID, &g.Name, &g.IsDefault, &g.OrchestrationStrategy, &canaryID, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		if canaryID != nil {
			g.CanaryEnvironmentID = *canaryID
		}
		groups = append(groups, g)
	}

	for i := range groups {
		envs, err := s.getEnvironments(ctx, groups[i].ID)
		if err != nil {
			return nil, err
		}
		groups[i].Environments = envs
	}

	return groups, rows.Err()
}

func (s *GroupStore) getEnvironments(ctx context.Context, groupID string) ([]group.Environment, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, group_id, name, type, connection_config, health_status, created_at
         FROM runtime_environments WHERE group_id = $1 ORDER BY name`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var envs []group.Environment
	for rows.Next() {
		var e group.Environment
		var configJSON []byte
		if err := rows.Scan(&e.ID, &e.GroupID, &e.Name, &e.Type, &configJSON, &e.HealthStatus, &e.CreatedAt); err != nil {
			return nil, err
		}
		if configJSON != nil {
			json.Unmarshal(configJSON, &e.ConnectionConfig)
		}
		envs = append(envs, e)
	}
	return envs, rows.Err()
}

func (s *GroupStore) Create(ctx context.Context, g *group.Group) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO runtime_environment_groups (name, is_default, orchestration_strategy, canary_environment_id)
         VALUES ($1, $2, $3, $4)`,
		g.Name, g.IsDefault, g.OrchestrationStrategy, nilIfEmpty(g.CanaryEnvironmentID))
	return err
}

func (s *GroupStore) Update(ctx context.Context, g *group.Group) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE runtime_environment_groups SET name = $2, is_default = $3, orchestration_strategy = $4, 
         canary_environment_id = $5, updated_at = now() WHERE id = $1`,
		g.ID, g.Name, g.IsDefault, g.OrchestrationStrategy, nilIfEmpty(g.CanaryEnvironmentID))
	return err
}

func (s *GroupStore) Delete(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM runtime_environment_groups WHERE id = $1", id)
	return err
}

func (s *GroupStore) HasNamespaces(ctx context.Context, groupID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM namespaces WHERE runtime_environment_group_id = $1)", groupID).Scan(&exists)
	return exists, err
}

func (s *GroupStore) GetByID(ctx context.Context, id string) (*group.Group, error) {
	var g group.Group
	var canaryID *string
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, is_default, orchestration_strategy, canary_environment_id, created_at, updated_at 
         FROM runtime_environment_groups WHERE id = $1`, id).
		Scan(&g.ID, &g.Name, &g.IsDefault, &g.OrchestrationStrategy, &canaryID, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if canaryID != nil {
		g.CanaryEnvironmentID = *canaryID
	}
	envs, err := s.getEnvironments(ctx, g.ID)
	if err != nil {
		return nil, err
	}
	g.Environments = envs
	return &g, nil
}

func (s *GroupStore) GetDefaultGroupID(ctx context.Context) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx,
		"SELECT id FROM runtime_environment_groups WHERE is_default = true LIMIT 1").Scan(&id)
	return id, err
}

func (s *GroupStore) GetEnvironmentIDsByGroup(ctx context.Context, groupID string) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		"SELECT id FROM runtime_environments WHERE group_id = $1", groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
