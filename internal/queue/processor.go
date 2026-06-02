package queue

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rabobank/papi/internal/domain/operation"
	"github.com/rabobank/papi/pkg/rei"
)

type Processor struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	stop   chan struct{}
}

func NewProcessor(pool *pgxpool.Pool, logger *slog.Logger) *Processor {
	return &Processor{
		pool:   pool,
		logger: logger,
		stop:   make(chan struct{}),
	}
}

func (p *Processor) Start(ctx context.Context) {
	go p.run(ctx)
}

func (p *Processor) Stop() {
	close(p.stop)
}

func (p *Processor) run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stop:
			return
		case <-ticker.C:
			p.processNext(ctx)
		}
	}
}

func (p *Processor) processNext(ctx context.Context) {
	row := p.pool.QueryRow(ctx, `
		SELECT o.id, o.namespace_id, o.runtime_environment_id, o.operation_type, o.payload,
		       re.type, re.connection_config
		FROM operations o
		JOIN runtime_environments re ON re.id = o.runtime_environment_id
		WHERE o.status = $1 AND re.health_status = 'healthy'
		ORDER BY o.created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`, operation.StatusPending)

	var opID, nsID, envID, opType string
	var payloadJSON, configJSON []byte
	var envType string

	if err := row.Scan(&opID, &nsID, &envID, &opType, &payloadJSON, &envType, &configJSON); err != nil {
		return
	}

	p.pool.Exec(ctx, "UPDATE operations SET status = $2, updated_at = now() WHERE id = $1", opID, operation.StatusInProgress)

	factory, ok := rei.Registry[envType]
	if !ok {
		p.failOperation(ctx, opID, "no REI implementation for type: "+envType)
		return
	}

	var connConfig map[string]interface{}
	json.Unmarshal(configJSON, &connConfig)

	runtime, err := factory(connConfig)
	if err != nil {
		p.failOperation(ctx, opID, "failed to create runtime: "+err.Error())
		return
	}

	switch operation.Type(opType) {
	case operation.TypeCreateNamespace:
		var payload map[string]string
		json.Unmarshal(payloadJSON, &payload)

		var components map[string]string
		if compJSON, ok := payload["name_components"]; ok {
			json.Unmarshal([]byte(compJSON), &components)
		}

		result, err := runtime.CreateNamespace(ctx, rei.CreateNamespaceRequest{
			NamespaceID:    nsID,
			CompositeName:  payload["composite_name"],
			NameComponents: components,
		})
		if err != nil {
			p.failOperation(ctx, opID, err.Error())
			return
		}

		if envType == "cloudfoundry" {
			p.pool.Exec(ctx,
				`INSERT INTO cf_namespace_mappings (namespace_id, runtime_environment_id, cf_org_name, cf_space_name, cf_org_guid, cf_space_guid)
				 VALUES ($1, $2, $3, $4, $5, $6)
				 ON CONFLICT (namespace_id, runtime_environment_id) DO UPDATE SET cf_org_guid = $5, cf_space_guid = $6`,
				nsID, envID, result.ExtensionData["cf_org_name"], result.ExtensionData["cf_space_name"],
				result.ExtensionData["cf_org_guid"], result.ExtensionData["cf_space_guid"])
		}

		p.completeOperation(ctx, opID)

	case operation.TypeDeleteNamespace:
		err := runtime.DeleteNamespace(ctx, rei.DeleteNamespaceRequest{
			NamespaceID: nsID,
		})
		if err != nil {
			p.failOperation(ctx, opID, err.Error())
			return
		}
		p.completeOperation(ctx, opID)
	}

	p.logger.Info("operation processed", "op_id", opID, "type", opType, "env_type", envType)
}

func (p *Processor) failOperation(ctx context.Context, opID, errMsg string) {
	p.pool.Exec(ctx,
		"UPDATE operations SET status = $2, error_message = $3, attempts = attempts + 1, last_attempted_at = now(), updated_at = now() WHERE id = $1",
		opID, operation.StatusFailed, errMsg)
	p.logger.Error("operation failed", "op_id", opID, "error", errMsg)
}

func (p *Processor) completeOperation(ctx context.Context, opID string) {
	p.pool.Exec(ctx,
		"UPDATE operations SET status = $2, updated_at = now() WHERE id = $1",
		opID, operation.StatusCompleted)

	p.pool.Exec(ctx, `
		UPDATE namespaces SET status = 'active', updated_at = now()
		WHERE id = (SELECT namespace_id FROM operations WHERE id = $1)
		AND NOT EXISTS (
			SELECT 1 FROM operations
			WHERE namespace_id = (SELECT namespace_id FROM operations WHERE id = $1)
			AND status != 'completed'
		)
	`, opID)
}
