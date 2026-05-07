package flusher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"

	"github.com/meindokuse/cloud-drive/auth-service-new/config"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
)

// OutboxRepository interface for outbox operations
type OutboxRepository interface {
	GetPending(ctx context.Context, batchSize int) ([]*entity.OutboxEvent, error)
	MarkProcessed(ctx context.Context, id uuid.UUID) error
}

// Flusher implements outbox pattern flusher
type Flusher struct {
	repo     OutboxRepository
	producer sarama.SyncProducer
	topic    string
	cfg      config.OutboxConfig
}

// NewFlusher creates a new outbox flusher
func NewFlusher(
	repo OutboxRepository,
	producer sarama.SyncProducer,
	cfg config.OutboxConfig,
	topic string,
) *Flusher {
	return &Flusher{
		repo:     repo,
		producer: producer,
		topic:    topic,
		cfg:      cfg,
	}
}

// Start starts the flusher loop
func (f *Flusher) Start(ctx context.Context) {
	if !f.cfg.FlushEnabled {
		slog.InfoContext(ctx, "outbox flusher is disabled")
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
			slog.InfoContext(ctx, "outbox flusher stopped")
			return
		case <-ticker.C:
			if err := f.flush(ctx); err != nil {
				slog.ErrorContext(ctx, "flush failed", "error", err)
			}
		}
	}
}

func (f *Flusher) flush(ctx context.Context) error {
	events, err := f.repo.GetPending(ctx, f.cfg.BatchSize)
	if err != nil {
		return fmt.Errorf("get pending events: %w", err)
	}

	if len(events) == 0 {
		return nil
	}

	slog.DebugContext(ctx, "flushing outbox events", "count", len(events))

	for _, event := range events {
		if err := f.sendEvent(ctx, event); err != nil {
			slog.ErrorContext(ctx, "failed to send event",
				"event_id", event.ID,
				"error", err,
			)
			continue
		}

		if err := f.repo.MarkProcessed(ctx, event.ID); err != nil {
			slog.ErrorContext(ctx, "failed to mark event as processed",
				"event_id", event.ID,
				"error", err,
			)
		}
	}

	return nil
}

func (f *Flusher) sendEvent(ctx context.Context, event *entity.OutboxEvent) error {
	if f.producer == nil {
		return fmt.Errorf("kafka producer not initialized")
	}

	key := event.AggregateID.String()
	value, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: f.topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(value),
		Headers: []sarama.RecordHeader{
			{Key: []byte("event_type"), Value: []byte(event.EventType)},
			{Key: []byte("aggregate_type"), Value: []byte(event.AggregateType)},
		},
	}

	_, _, err = f.producer.SendMessage(msg)
	return err
}
