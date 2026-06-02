package queue

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// IdentityRefresher periodically checks identity_group_cache for expired entries
// and refreshes them by updating last_refreshed_at to now().
type IdentityRefresher struct {
	pool     *pgxpool.Pool
	logger   *slog.Logger
	stop     chan struct{}
	interval time.Duration
}

func NewIdentityRefresher(pool *pgxpool.Pool, logger *slog.Logger, interval time.Duration) *IdentityRefresher {
	if interval == 0 {
		interval = 30 * time.Second
	}
	return &IdentityRefresher{
		pool:     pool,
		logger:   logger,
		stop:     make(chan struct{}),
		interval: interval,
	}
}

func (r *IdentityRefresher) Start(ctx context.Context) {
	go r.run(ctx)
}

func (r *IdentityRefresher) Stop() {
	close(r.stop)
}

func (r *IdentityRefresher) run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stop:
			return
		case <-ticker.C:
			r.refreshExpired(ctx)
		}
	}
}

func (r *IdentityRefresher) refreshExpired(ctx context.Context) {
	result, err := r.pool.Exec(ctx, `
		UPDATE identity_group_cache
		SET last_refreshed_at = now()
		WHERE last_refreshed_at + (ttl_seconds || ' seconds')::interval < now()
	`)
	if err != nil {
		r.logger.Error("failed to refresh identity cache entries", "error", err)
		return
	}

	if result.RowsAffected() > 0 {
		r.logger.Info("refreshed expired identity cache entries", "count", result.RowsAffected())
	}
}
