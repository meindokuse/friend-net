package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
)

// Storage implements outbox repository
type Storage struct {
	pool *pgxpool.Pool
}

// NewStorage creates a new outbox storage
func NewStorage(pool *pgxpool.Pool) *Storage {
	return &Storage{pool: pool}
}

// Create creates a new outbox event
func (s *Storage) Create(ctx context.Context, event *entity.OutboxEvent) error {
	const query = `
		INSERT INTO outbox_events (
			id, aggregate_type, aggregate_id, event_type, payload, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := s.pool.Exec(ctx, query,
		event.ID,
		event.AggregateType,
		event.AggregateID,
		event.EventType,
		event.Payload,
		event.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("insert outbox: %w", err)
	}

	return nil
}

// GetPending retrieves pending events for flusher
func (s *Storage) GetPending(ctx context.Context, batchSize int) ([]*entity.OutboxEvent, error) {
	const query = `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, created_at, processed_at
		FROM outbox_events
		WHERE processed_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1
	`

	rows, err := s.pool.Query(ctx, query, batchSize)
	if err != nil {
		return nil, fmt.Errorf("query pending: %w", err)
	}
	defer rows.Close()

	var events []*entity.OutboxEvent
	for rows.Next() {
		var event entity.OutboxEvent
		if err := rows.Scan(
			&event.ID,
			&event.AggregateType,
			&event.AggregateID,
			&event.EventType,
			&event.Payload,
			&event.CreatedAt,
			&event.ProcessedAt,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		events = append(events, &event)
	}

	return events, nil
}

// MarkProcessed marks an event as processed
func (s *Storage) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	const query = `
		UPDATE outbox_events
		SET processed_at = $1
		WHERE id = $2
	`

	_, err := s.pool.Exec(ctx, query, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("mark processed: %w", err)
	}

	return nil
}

// OutboxEventDAO for database mapping
type OutboxEventDAO struct {
	ID            uuid.UUID       `db:"id"`
	AggregateType string          `db:"aggregate_type"`
	AggregateID   uuid.UUID       `db:"aggregate_id"`
	EventType     string          `db:"event_type"`
	Payload       json.RawMessage `db:"payload"`
	CreatedAt     time.Time       `db:"created_at"`
	ProcessedAt   *time.Time      `db:"processed_at"`
}

// ToEntity converts DAO to entity
func (dao *OutboxEventDAO) ToEntity() *entity.OutboxEvent {
	return &entity.OutboxEvent{
		ID:            dao.ID,
		AggregateType: dao.AggregateType,
		AggregateID:   dao.AggregateID,
		EventType:     dao.EventType,
		Payload:       dao.Payload,
		CreatedAt:     dao.CreatedAt,
		ProcessedAt:   dao.ProcessedAt,
	}
}
