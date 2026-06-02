package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rabobank/papi/internal/domain/namespace"
)

type NameTemplateStore struct {
	pool *pgxpool.Pool
}

func NewNameTemplateStore(pool *pgxpool.Pool) *NameTemplateStore {
	return &NameTemplateStore{pool: pool}
}

func (s *NameTemplateStore) GetActiveTemplate(ctx context.Context) (*namespace.NamingConfig, error) {
	var pattern string
	var segmentsJSON []byte
	err := s.pool.QueryRow(ctx,
		"SELECT template_pattern, segments FROM namespace_name_templates WHERE is_active = true LIMIT 1",
	).Scan(&pattern, &segmentsJSON)
	if err != nil {
		return nil, err
	}

	var segments []namespace.TemplateSegment
	if err := json.Unmarshal(segmentsJSON, &segments); err != nil {
		return nil, err
	}

	return &namespace.NamingConfig{
		TemplatePattern: pattern,
		Segments:        segments,
	}, nil
}
