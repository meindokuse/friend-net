package outbox

import (
	"context"
	"database/sql"
	"fmt"
)

// Repository для работы с outbox_events таблицей.
type Repository struct {
	db *sql.DB
}

// NewRepository создаёт новый outbox репозиторий.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Save сохраняет событие в outbox таблицу.
// ВАЖНО: должно вызываться в той же транзакции что и основная операция!
func (r *Repository) Save(ctx context.Context, tx *sql.Tx, event *OutboxEvent) error {
	const query = `
		INSERT INTO outbox_events (
			id,
			aggregate_type,
			aggregate_id,
			event_type,
			payload,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query,
			event.ID,
			event.AggregateType,
			event.AggregateID,
			event.EventType,
			event.Payload,
			event.CreatedAt,
		)
	} else {
		_, err = r.db.ExecContext(ctx, query,
			event.ID,
			event.AggregateType,
			event.AggregateID,
			event.EventType,
			event.Payload,
			event.CreatedAt,
		)
	}

	if err != nil {
		return fmt.Errorf("save outbox event: %w", err)
	}

	return nil
}
