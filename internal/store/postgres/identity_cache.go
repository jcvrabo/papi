package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rabobank/papi/internal/domain/namespace"
)

type IdentityCacheStore struct {
	pool *pgxpool.Pool
}

func NewIdentityCacheStore(pool *pgxpool.Pool) *IdentityCacheStore {
	return &IdentityCacheStore{pool: pool}
}

func (s *IdentityCacheStore) GetByGroupIdentifier(ctx context.Context, groupIdentifier string) (*namespace.IdentityGroupCache, error) {
	var cache namespace.IdentityGroupCache
	var membersJSON []byte
	err := s.pool.QueryRow(ctx,
		"SELECT id, group_identifier, resolved_members, last_refreshed_at, ttl_seconds FROM identity_group_cache WHERE group_identifier = $1",
		groupIdentifier).Scan(&cache.ID, &cache.GroupIdentifier, &membersJSON, &cache.LastRefreshedAt, &cache.TTLSeconds)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(membersJSON, &cache.ResolvedMembers)
	return &cache, nil
}

func (s *IdentityCacheStore) Upsert(ctx context.Context, cache *namespace.IdentityGroupCache) error {
	membersJSON, _ := json.Marshal(cache.ResolvedMembers)
	_, err := s.pool.Exec(ctx,
		`INSERT INTO identity_group_cache (group_identifier, resolved_members, last_refreshed_at, ttl_seconds)
		 VALUES ($1, $2, now(), $3)
		 ON CONFLICT (group_identifier) DO UPDATE SET resolved_members = $2, last_refreshed_at = now()`,
		cache.GroupIdentifier, membersJSON, cache.TTLSeconds)
	return err
}

func (s *IdentityCacheStore) IsUserInAnyGroup(ctx context.Context, userIdentifier string) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		"SELECT group_identifier FROM identity_group_cache WHERE resolved_members @> to_jsonb($1::text)",
		userIdentifier)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []string
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}
