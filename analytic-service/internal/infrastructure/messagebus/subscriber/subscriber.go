package subscriber

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"log/slog"
	"sync"

	"github.com/segmentio/kafka-go"

	analyticevents "github.com/meindokuse/cloud-drive/common/events/analytic"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/application/service/analytic/ingest_event"
)

// EventIngester is the application-layer port for ingesting a single event.
type EventIngester interface {
	Execute(ctx context.Context, in ingest_event.Input) error
}

// Consumer fans out Kafka messages to a pool of virtual workers.
// Workers are hash-routed by EventID for stable per-event ordering.
// Delivery semantics: at-least-once (auto-commit every second).
type Consumer struct {
	reader   *kafka.Reader
	ingester EventIngester
	workers  []chan kafka.Message
	wg       sync.WaitGroup

	cancelMu sync.Mutex
	cancel   context.CancelFunc
}

func NewConsumer(
	brokers []string,
	topic, groupID string,
	ingester EventIngester,
	workersCount int,
) *Consumer {
	if workersCount <= 0 {
		workersCount = 8
	}

	workers := make([]chan kafka.Message, workersCount)
	for i := range workers {
		workers[i] = make(chan kafka.Message, 64)
	}

	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     brokers,
			Topic:       topic,
			GroupID:     groupID,
			StartOffset: kafka.FirstOffset,
			// Auto-commit every second; analytics tolerates at-least-once delivery.
			CommitInterval: 1000000000, // 1s in nanoseconds
		}),
		ingester: ingester,
		workers:  workers,
	}
}

// Start launches the worker pool and the fetch loop. Blocks until ctx is cancelled or Stop is called.
func (c *Consumer) Start(parentCtx context.Context) {
	ctx, cancel := context.WithCancel(parentCtx)
	c.cancelMu.Lock()
	c.cancel = cancel
	c.cancelMu.Unlock()
	defer cancel()

	for _, ch := range c.workers {
		c.wg.Add(1)
		go c.workerLoop(ctx, ch)
	}
	defer func() {
		for _, ch := range c.workers {
			close(ch)
		}
	}()

	slog.InfoContext(ctx, "analytic kafka consumer started",
		"topic", c.reader.Config().Topic,
		"workers", len(c.workers),
	)

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			slog.ErrorContext(ctx, "fetch kafka message failed", "error", err)
			continue
		}

		workerID := hashKey(msg.Key) % uint32(len(c.workers))
		c.workers[workerID] <- msg
	}

	c.wg.Wait()
	slog.InfoContext(ctx, "analytic kafka consumer stopped")
}

func (c *Consumer) Stop(ctx context.Context) error {
	c.cancelMu.Lock()
	if c.cancel != nil {
		c.cancel()
	}
	c.cancelMu.Unlock()

	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		slog.ErrorContext(ctx, "analytic consumer shutdown timed out")
	}
	return c.reader.Close()
}

func (c *Consumer) workerLoop(ctx context.Context, msgs <-chan kafka.Message) {
	defer c.wg.Done()
	for msg := range msgs {
		c.handle(ctx, msg)
	}
}

func (c *Consumer) handle(ctx context.Context, msg kafka.Message) {
	var event analyticevents.AnalyticEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		slog.ErrorContext(ctx, "analytic: poison pill",
			"topic", msg.Topic, "offset", msg.Offset, "error", err)
		return
	}

	if err := c.ingester.Execute(ctx, ingest_event.Input{
		EventID:   event.EventID,
		EventType: event.EventType,
		Service:   event.Service,
		UserID:    event.UserID,
		Payload:   event.Payload,
		Timestamp: event.Timestamp,
	}); err != nil {
		slog.ErrorContext(ctx, "analytic: ingest failed",
			"event_type", event.EventType, "error", err)
	}
}

func hashKey(key []byte) uint32 {
	if len(key) == 0 {
		return 0
	}
	h := fnv.New32a()
	h.Write(key)
	return h.Sum32()
}
