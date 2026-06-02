package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rabobank/papi/internal/domain/namespace"
)

type MetadataRuleStore struct {
	pool *pgxpool.Pool
}

func NewMetadataRuleStore(pool *pgxpool.Pool) *MetadataRuleStore {
	return &MetadataRuleStore{pool: pool}
}

func (s *MetadataRuleStore) GetMandatoryRules(ctx context.Context) ([]namespace.MetadataRule, error) {
	rows, err := s.pool.Query(ctx,
		"SELECT field_name, is_mandatory, validation_regex, allowed_values, description FROM metadata_rules WHERE is_mandatory = true")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []namespace.MetadataRule
	for rows.Next() {
		var r namespace.MetadataRule
		var allowedJSON []byte
		var validationRegex, description *string
		if err := rows.Scan(&r.FieldName, &r.IsMandatory, &validationRegex, &allowedJSON, &description); err != nil {
			return nil, err
		}
		if validationRegex != nil {
			r.ValidationRegex = *validationRegex
		}
		if allowedJSON != nil {
			json.Unmarshal(allowedJSON, &r.AllowedValues)
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}
