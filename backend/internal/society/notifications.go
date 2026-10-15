package society

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// EnqueueNotificationOutbox writes a durable notification event into the PostgreSQL outbox within a transaction.
func (r *Repository) EnqueueNotificationOutbox(ctx context.Context, tx pgx.Tx, eventType, idempotencyKey string, payload map[string]interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize notification payload: %w", err)
	}

	query := `
		INSERT INTO notification_outbox (event_type, idempotency_key, payload, status, attempt_count, next_attempt_at)
		VALUES ($1, $2, $3::jsonb, 'PENDING', 0, NOW())
		ON CONFLICT (idempotency_key) DO NOTHING
	`

	if tx != nil {
		_, err = tx.Exec(ctx, query, eventType, idempotencyKey, string(payloadJSON))
	} else if r.pool != nil {
		_, err = r.pool.Exec(ctx, query, eventType, idempotencyKey, string(payloadJSON))
	} else {
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to insert notification outbox: %w", err)
	}
	return nil
}
