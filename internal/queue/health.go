package queue

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rabobank/papi/pkg/rei"
)

type HealthChecker struct {
	pool     *pgxpool.Pool
	logger   *slog.Logger
	interval time.Duration
	stop     chan struct{}
}

func NewHealthChecker(pool *pgxpool.Pool, logger *slog.Logger, interval time.Duration) *HealthChecker {
	return &HealthChecker{
		pool:     pool,
		logger:   logger,
		interval: interval,
		stop:     make(chan struct{}),
	}
}

func (h *HealthChecker) Start(ctx context.Context) {
	go h.run(ctx)
}

func (h *HealthChecker) Stop() {
	close(h.stop)
}

func (h *HealthChecker) run(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.stop:
			return
		case <-ticker.C:
			h.checkAll(ctx)
		}
	}
}

func (h *HealthChecker) checkAll(ctx context.Context) {
	rows, err := h.pool.Query(ctx,
		"SELECT id, type, connection_config FROM runtime_environments")
	if err != nil {
		h.logger.Error("failed to query environments", "error", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, envType string
		var configJSON []byte
		if err := rows.Scan(&id, &envType, &configJSON); err != nil {
			continue
		}

		factory, ok := rei.Registry[envType]
		if !ok {
			continue
		}

		var connConfig map[string]interface{}
		json.Unmarshal(configJSON, &connConfig)

		runtime, err := factory(connConfig)
		if err != nil {
			h.updateStatus(ctx, id, "unavailable")
			continue
		}

		status, err := runtime.HealthCheck(ctx)
		if err != nil {
			h.updateStatus(ctx, id, "unavailable")
			continue
		}

		h.updateStatus(ctx, id, string(status.Status))
	}
}

func (h *HealthChecker) updateStatus(ctx context.Context, envID, status string) {
	h.pool.Exec(ctx,
		"UPDATE runtime_environments SET health_status = $2, last_health_check_at = now(), updated_at = now() WHERE id = $1",
		envID, status)
}
