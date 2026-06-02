package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rabobank/papi/internal/platform"
)

type PlatformRoleStore struct {
	pool *pgxpool.Pool
}

func NewPlatformRoleStore(pool *pgxpool.Pool) *PlatformRoleStore {
	return &PlatformRoleStore{pool: pool}
}

func (s *PlatformRoleStore) HasRole(ctx context.Context, userIdentifier string, role platform.Role) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM platform_user_roles WHERE user_identifier = $1 AND role = $2)",
		userIdentifier, string(role)).Scan(&exists)
	return exists, err
}

func (s *PlatformRoleStore) GetRoles(ctx context.Context, userIdentifier string) ([]platform.Role, error) {
	rows, err := s.pool.Query(ctx,
		"SELECT role FROM platform_user_roles WHERE user_identifier = $1",
		userIdentifier)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []platform.Role
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			return nil, err
		}
		roles = append(roles, platform.Role(r))
	}
	return roles, rows.Err()
}
