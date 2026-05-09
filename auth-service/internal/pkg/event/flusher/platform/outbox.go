package platform

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/meindokuse/cloud-drive/auth-service-new/config"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
)

// outboxAdvisoryLockKey is a stable per-service key for pg_try_advisory_xact_lock.
// Prevents concurrent flushes across pods without a schema change.
// The lock is transaction-scoped: released automatically on COMMIT or ROLLBACK.
const outboxAdvisoryLockKey = int64(0x617574685f6f7574) // "auth_out"

// Flusher polls the outbox table and publishes pending events to Kafka.
type Flusher struct {
	pool     *pgxpool.Pool
	producer sarama.SyncProducer
	topic    string
	cfg      config.OutboxConfig
	done     chan struct{}
}

// NewFlusher creates a new Flusher.
func NewFlusher(
	pool *pgxpool.Pool,
	producer sarama.SyncProducer,
	cfg config.OutboxConfig,
	topic string,
) *Flusher {
	return &Flusher{
		pool:     pool,
		producer: producer,
		topic:    topic,
		cfg:      cfg,
		done:     make(chan struct{}),
	}
}

// Done returns a channel closed when the flusher has fully stopped.
func (f *Flusher) Done() <-chan struct{} {
	return f.done
}

// Start runs the polling loop until ctx is cancelled.
// Call Done() to wait for the final drain to complete after cancellation.
func (f *Flusher) Start(ctx context.Context) {
	defer close(f.done)

	if !f.cfg.FlushEnabled {
		slog.InfoContext(ctx, "outbox flusher disabled")
		return
	}

	ticker := time.NewTicker(f.cfg.FlushInterval)
	defer ticker.Stop()

	slog.InfoContext(ctx, "outbox flusher started",
		"interval", f.cfg.FlushInterval,
		"batch_size", f.cfg.BatchSize,
	)

	for {
		select {
		case <-ctx.Done():
			// Drain one final batch before stopping so no events are lost on
			// graceful shutdown. Uses a fresh context so the pool/producer are
			// still reachable even if the parent context has been cancelled.
			drainCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := f.flush(drainCtx); err != nil {
				slog.ErrorContext(drainCtx, "drain flush failed", "error", err)
			}
			cancel()
			slog.InfoContext(ctx, "outbox flusher stopped")
			return
		case <-ticker.C:
			if err := f.flush(ctx); err != nil {
				slog.ErrorContext(ctx, "flush error", "error", err)
			}
		}
	}
}

// ─── flush ────────────────────────────────────────────────────────────────────

const queryFetchPending = `
    SELECT id, aggregate_type, aggregate_id, event_type, payload, created_at
    FROM outbox_events
    WHERE processed_at IS NULL
    ORDER BY created_at ASC
    LIMIT $1
`

func (f *Flusher) flush(ctx context.Context) error {
	if f.producer == nil {
		return fmt.Errorf("kafka producer not initialized")
	}

	tx, err := f.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Transaction-level advisory lock: only one pod flushes at a time.
	// pg_try_advisory_xact_lock returns false (non-blocking) if another
	// session already holds the lock; we simply skip this cycle.
	var locked bool
	if err := tx.QueryRow(ctx,
		"SELECT pg_try_advisory_xact_lock($1)", outboxAdvisoryLockKey,
	).Scan(&locked); err != nil {
		return fmt.Errorf("advisory lock: %w", err)
	}
	if !locked {
		return nil
	}

	events, err := f.fetchPending(ctx, tx)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	slog.DebugContext(ctx, "flushing outbox batch", "count", len(events))

	successIDs, sendErr := f.sendWithRetry(ctx, events)
	if len(successIDs) == 0 {
		return sendErr
	}

	if err := f.markProcessed(ctx, tx, successIDs); err != nil {
		return fmt.Errorf("mark processed: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	slog.InfoContext(ctx, "outbox batch flushed",
		"sent", len(successIDs),
		"total", len(events),
	)

	return sendErr // non-nil only on a partial batch failure
}

func (f *Flusher) fetchPending(ctx context.Context, tx pgx.Tx) ([]*entity.OutboxEvent, error) {
	rows, err := tx.Query(ctx, queryFetchPending, f.cfg.BatchSize)
	if err != nil {
		return nil, fmt.Errorf("query pending: %w", err)
	}
	defer rows.Close()

	var events []*entity.OutboxEvent
	for rows.Next() {
		var e entity.OutboxEvent
		if err := rows.Scan(
			&e.ID, &e.AggregateType, &e.AggregateID,
			&e.EventType, &e.Payload, &e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		events = append(events, &e)
	}
	return events, rows.Err()
}

// ─── Kafka send with retry ────────────────────────────────────────────────────

func (f *Flusher) sendWithRetry(ctx context.Context, events []*entity.OutboxEvent) ([]uuid.UUID, error) {
	msgs, msgToEventID := f.buildMessages(events)

	maxAttempts := f.cfg.RetryAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(f.cfg.RetryDelay):
			}
			slog.WarnContext(ctx, "retrying kafka batch send",
				"attempt", attempt,
				"max", maxAttempts,
				"error", lastErr,
			)
		}

		err := f.producer.SendMessages(msgs)
		if err == nil {
			ids := make([]uuid.UUID, len(events))
			for i, e := range events {
				ids[i] = e.ID
			}
			return ids, nil
		}

		// sarama.ProducerErrors: partial batch failure — some messages were
		// rejected (e.g. message-too-large). Return the successes immediately
		// rather than retrying, since the same messages would fail again.
		if prodErrs, ok := err.(sarama.ProducerErrors); ok {
			failed := make(map[*sarama.ProducerMessage]struct{}, len(prodErrs))
			for _, pe := range prodErrs {
				failed[pe.Msg] = struct{}{}
				slog.ErrorContext(ctx, "kafka message rejected",
					"event_id", msgToEventID[pe.Msg],
					"error", pe.Err,
				)
			}
			var successIDs []uuid.UUID
			for i, e := range events {
				if _, bad := failed[msgs[i]]; !bad {
					successIDs = append(successIDs, e.ID)
				}
			}
			return successIDs, err
		}

		// Whole-batch failure (broker unreachable, network error, etc.) — retry.
		lastErr = err
	}

	return nil, lastErr
}

func (f *Flusher) buildMessages(events []*entity.OutboxEvent) ([]*sarama.ProducerMessage, map[*sarama.ProducerMessage]string) {
	msgs := make([]*sarama.ProducerMessage, len(events))
	msgToEventID := make(map[*sarama.ProducerMessage]string, len(events))

	for i, e := range events {
		msg := &sarama.ProducerMessage{
			Topic: f.topic,
			// Partition by aggregate ID to preserve per-entity ordering.
			Key: sarama.StringEncoder(e.AggregateID.String()),
			// Payload is already marshalled JSON — send the raw bytes directly.
			Value: sarama.ByteEncoder(e.Payload),
			Headers: []sarama.RecordHeader{
				{Key: []byte("event_id"), Value: []byte(e.ID.String())},
				{Key: []byte("event_type"), Value: []byte(e.EventType)},
				{Key: []byte("aggregate_type"), Value: []byte(e.AggregateType)},
			},
		}
		msgs[i] = msg
		msgToEventID[msg] = e.ID.String()
	}
	return msgs, msgToEventID
}

// ─── batch mark processed ─────────────────────────────────────────────────────

func (f *Flusher) markProcessed(ctx context.Context, tx pgx.Tx, ids []uuid.UUID) error {
	batch := &pgx.Batch{}
	for _, id := range ids {
		batch.Queue("UPDATE outbox_events SET processed_at = NOW() WHERE id = $1", id)
	}
	br := tx.SendBatch(ctx, batch)
	for range ids {
		if _, err := br.Exec(); err != nil {
			_ = br.Close()
			return err
		}
	}
	return br.Close()
}
