package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RuntimeGroupStore struct {
	pool *pgxpool.Pool
}

func NewRuntimeGroupStore(pool *pgxpool.Pool) *RuntimeGroupStore {
	return &RuntimeGroupStore{pool: pool}
}

func (s *RuntimeGroupStore) GetDefaultGroup(ctx context.Context) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx,
		"SELECT id FROM runtime_environment_groups WHERE is_default = true LIMIT 1",
	).Scan(&id)
	return id, err
}

func (s *RuntimeGroupStore) GetEnvironmentsByGroup(ctx context.Context, groupID string) ([]string, error) {
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
