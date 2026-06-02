package queue

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Notifier struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewNotifier(pool *pgxpool.Pool, logger *slog.Logger) *Notifier {
	return &Notifier{pool: pool, logger: logger}
}

// Listen starts listening for NOTIFY events on the 'new_operation' channel.
// When notified, it triggers immediate processing.
func (n *Notifier) Listen(ctx context.Context, processor *Processor) {
	conn, err := n.pool.Acquire(ctx)
	if err != nil {
		n.logger.Error("failed to acquire connection for LISTEN", "error", err)
		return
	}
	defer conn.Release()

	_, err = conn.Exec(ctx, "LISTEN new_operation")
	if err != nil {
		n.logger.Error("failed to LISTEN", "error", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			notification, err := conn.Conn().WaitForNotification(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				n.logger.Error("notification error", "error", err)
				continue
			}
			n.logger.Debug("received notification", "channel", notification.Channel, "payload", notification.Payload)
			processor.processNext(ctx)
		}
	}
}
