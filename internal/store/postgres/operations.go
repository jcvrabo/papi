package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rabobank/papi/internal/domain/operation"
)

type OperationStore struct {
	pool *pgxpool.Pool
}

func NewOperationStore(pool *pgxpool.Pool) *OperationStore {
	return &OperationStore{pool: pool}
}

func (s *OperationStore) Create(ctx context.Context, op *operation.Operation) error {
	payloadJSON, _ := json.Marshal(op.Payload)
	_, err := s.pool.Exec(ctx,
		`INSERT INTO operations (namespace_id, runtime_environment_id, operation_type, status, idempotency_key, payload)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		op.NamespaceID, op.RuntimeEnvironmentID, op.OperationType, op.Status, op.IdempotencyKey, payloadJSON)
	return err
}

func (s *OperationStore) ListByNamespace(ctx context.Context, namespaceID string) ([]operation.Operation, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, namespace_id, runtime_environment_id, operation_type, status, idempotency_key,
				payload, attempts, last_attempted_at, error_message, created_at, updated_at
		 FROM operations WHERE namespace_id = $1 ORDER BY created_at`,
		namespaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ops []operation.Operation
	for rows.Next() {
		var op operation.Operation
		var payloadJSON []byte
		if err := rows.Scan(&op.ID, &op.NamespaceID, &op.RuntimeEnvironmentID, &op.OperationType,
			&op.Status, &op.IdempotencyKey, &payloadJSON, &op.Attempts, &op.LastAttemptedAt,
			&op.ErrorMessage, &op.CreatedAt, &op.UpdatedAt); err != nil {
			return nil, err
		}
		if payloadJSON != nil {
			json.Unmarshal(payloadJSON, &op.Payload)
		}
		ops = append(ops, op)
	}
	return ops, rows.Err()
}

func (s *OperationStore) UpdateStatus(ctx context.Context, id string, status operation.Status, errorMsg string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE operations SET status = $2, error_message = $3, attempts = attempts + 1,
		 last_attempted_at = now(), updated_at = now() WHERE id = $1`,
		id, status, errorMsg)
	return err
}
