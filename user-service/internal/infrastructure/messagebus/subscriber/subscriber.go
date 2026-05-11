package subscriber

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"

	authevents "github.com/meindokuse/cloud-drive/common/events/auth-service"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/create_user"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/domain/entity"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/infrastructure/messagebus/watermark"
)

// UserCreator is the application-layer port for creating a user.
type UserCreator interface {
	Execute(ctx context.Context, in create_user.Input) (*entity.User, error)
}

// IdempotencyStore provides exactly-once delivery guarantees by persisting
// a record for every successfully processed event.
type IdempotencyStore interface {
	IsProcessed(ctx context.Context, key string) (bool, error)
	MarkProcessed(ctx context.Context, key, topic string, partition int, offset int64) error
}

// DLQWriter routes unprocessable messages to a dead-letter topic.
type DLQWriter interface {
	Write(ctx context.Context, reason error, msgs ...kafka.Message) error
}

// workerTask is dispatched from the fetch loop to a virtual-partition worker.
type workerTask struct {
	msg   kafka.Message
	event authevents.AccountCreated
}

// Consumer is a Kafka consumer that fans out messages to a fixed pool of virtual-partition
// workers. Hash routing on AccountID ensures all events for the same account always land
// on the same worker, preserving per-account ordering.
//
// Offset commits are gated by per-Kafka-partition watermarks: a partition offset is committed
// only when all preceding offsets have been processed, regardless of which virtual worker
// handled them.
type Consumer struct {
	reader      *kafka.Reader
	creator     UserCreator
	idempotency IdempotencyStore
	dlq         DLQWriter // optional; nil = log and skip on permanent failure

	maxRetries    int
	maxDLQRetries int

	workers []chan workerTask
	wg      sync.WaitGroup

	// watermarks maps Kafka partition → *watermark.PartitionWatermark via sync.Map.
	watermarks sync.Map

	cancelMu sync.Mutex
	cancel   context.CancelFunc
}

// Options holds optional constructor parameters.
type Options struct {
	DLQ           DLQWriter
	MaxRetries    int
	MaxDLQRetries int
}

// NewConsumer creates a Consumer with a pool of workersCount virtual-partition workers.
// workersCount <= 0 defaults to 16.
func NewConsumer(
	brokers []string,
	topic, groupID string,
	creator UserCreator,
	idempotency IdempotencyStore,
	workersCount int,
	opts Options,
) *Consumer {
	if workersCount <= 0 {
		workersCount = 16
	}
	if opts.MaxRetries <= 0 {
		opts.MaxRetries = 3
	}
	if opts.MaxDLQRetries <= 0 {
		opts.MaxDLQRetries = 3
	}

	workers := make([]chan workerTask, workersCount)
	for i := range workers {
		workers[i] = make(chan workerTask, 64)
	}

	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			CommitInterval: 0, // manual commit only
			StartOffset:    kafka.FirstOffset,
		}),
		creator:       creator,
		idempotency:   idempotency,
		dlq:           opts.DLQ,
		maxRetries:    opts.MaxRetries,
		maxDLQRetries: opts.MaxDLQRetries,
		workers:       workers,
	}
}

// Start launches the virtual-partition worker pool and the fetch loop.
// It blocks until ctx is cancelled or Stop is called.
func (c *Consumer) Start(parentCtx context.Context) {
	ctx, cancel := context.WithCancel(parentCtx)
	c.cancelMu.Lock()
	c.cancel = cancel
	c.cancelMu.Unlock()
	defer cancel()

	for i, ch := range c.workers {
		c.wg.Add(1)
		go c.workerLoop(ctx, i, ch)
	}

	// Closing all worker channels signals the workers to drain and exit.
	defer func() {
		for _, ch := range c.workers {
			close(ch)
		}
	}()

	slog.InfoContext(ctx, "kafka consumer started",
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

		slog.InfoContext(ctx, "kafka message received",
			"topic", msg.Topic,
			"partition", msg.Partition,
			"offset", msg.Offset,
			"consumer_group", c.reader.Config().GroupID,
		)

		tracker := c.getOrInitTracker(msg.Partition, msg.Offset)

		// Parse early so we can route by AccountID. Malformed messages are poison pills:
		// they cannot be retried, so we skip immediately and advance the watermark.
		var event authevents.AccountCreated
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			slog.ErrorContext(ctx, "poison pill: unmarshal failed",
				"topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset, "error", err)
			c.handlePoisonPill(ctx, msg, tracker, fmt.Errorf("unmarshal: %w", err))
			continue
		}

		workerID := hashAccountID(event.AccountID.String()) % uint32(len(c.workers))
		c.workers[workerID] <- workerTask{msg: msg, event: event}
	}

	c.wg.Wait()
	slog.InfoContext(ctx, "kafka consumer stopped")
}

// Stop cancels the fetch context and closes the reader. The closer in app.go calls
// this with a graceful-shutdown timeout context.
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
		slog.InfoContext(ctx, "all consumer workers drained")
	case <-ctx.Done():
		slog.ErrorContext(ctx, "consumer shutdown timed out; some in-flight messages dropped")
	}

	return c.reader.Close()
}

// workerLoop drains a virtual-partition channel sequentially, preserving per-account order.
func (c *Consumer) workerLoop(ctx context.Context, _ int, tasks <-chan workerTask) {
	defer c.wg.Done()
	for task := range tasks {
		c.process(ctx, task)
	}
}

// process runs the full pipeline: idempotency check → handle with retry → mark processed → commit.
func (c *Consumer) process(ctx context.Context, task workerTask) {
	start := time.Now()
	msg := task.msg
	key := idempotencyKey(msg)

	trackerIface, _ := c.watermarks.Load(msg.Partition)
	tracker := trackerIface.(*watermark.PartitionWatermark)

	already, err := c.idempotency.IsProcessed(ctx, key)
	if err != nil {
		// A failed lookup is not a reason to skip — fall through to processing.
		slog.ErrorContext(ctx, "idempotency lookup failed",
			"topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset, "error", err)
	}

	if !already {
		if processErr := c.handleWithRetry(ctx, task); processErr != nil {
			slog.ErrorContext(ctx, "message permanently failed",
				"topic", msg.Topic,
				"partition", msg.Partition,
				"offset", msg.Offset,
				"error", processErr,
				"retry_attempt", c.maxRetries,
			)
			c.sendToDLQ(ctx, processErr, msg)
		} else {
			slog.InfoContext(ctx, "kafka message processed",
				"topic", msg.Topic,
				"offset", msg.Offset,
				"duration_ms", time.Since(start).Milliseconds(),
			)
			if markErr := c.idempotency.MarkProcessed(ctx, key, msg.Topic, msg.Partition, msg.Offset); markErr != nil {
				// Non-fatal: successful processing with a failed mark means the message will
				// be re-processed on restart and skipped by the domain duplicate guards.
				slog.ErrorContext(ctx, "mark processed failed",
					"topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset, "error", markErr)
			}
		}
	}

	c.commitProgress(ctx, msg, tracker)
}

// handleWithRetry calls handle with up to maxRetries attempts and exponential backoff.
func (c *Consumer) handleWithRetry(ctx context.Context, task workerTask) error {
	var lastErr error
	delay := 100 * time.Millisecond

	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
			slog.WarnContext(ctx, "retrying kafka message",
				"topic", task.msg.Topic,
				"partition", task.msg.Partition,
				"offset", task.msg.Offset,
				"attempt_number", attempt+1,
				"reason", lastErr,
			)
			delay *= 2
		}
		if lastErr = c.handle(ctx, task); lastErr == nil {
			return nil
		}
	}
	return fmt.Errorf("all %d attempts failed: %w", c.maxRetries, lastErr)
}

// handle calls the application service for a single AccountCreated event.
func (c *Consumer) handle(ctx context.Context, task workerTask) error {
	event := task.event

	username := event.Username
	if username == "" {
		if idx := len(event.Email); idx > 0 {
			for i, ch := range event.Email {
				if ch == '@' {
					username = event.Email[:i]
					break
				}
			}
		}
		if username == "" {
			username = event.Email
		}
	}
	displayName := event.DisplayName
	if displayName == "" {
		displayName = username
	}
	email := event.Email

	_, err := c.creator.Execute(ctx, create_user.Input{
		ID:          &event.AccountID,
		Username:    username,
		Email:       &email,
		DisplayName: displayName,
	})
	if err != nil {
		if errors.Is(err, entity.ErrUsernameAlreadyTaken) || errors.Is(err, entity.ErrEmailAlreadyTaken) {
			return nil // idempotent at the domain level
		}
		return err
	}
	return nil
}

// commitProgress marks offset done in the partition watermark and commits to Kafka when
// the watermark advances (i.e., all preceding offsets are also done).
func (c *Consumer) commitProgress(ctx context.Context, msg kafka.Message, tracker *watermark.PartitionWatermark) {
	_, shouldCommit := tracker.MarkDone(msg.Offset)
	if !shouldCommit {
		return
	}
	if err := c.reader.CommitMessages(ctx, msg); err != nil && ctx.Err() == nil {
		slog.ErrorContext(ctx, "commit offset failed",
			"partition", msg.Partition, "offset", msg.Offset, "error", err)
	}
}

// handlePoisonPill attempts to route the malformed message to the DLQ, then advances
// the watermark so subsequent messages in the partition are not blocked.
func (c *Consumer) handlePoisonPill(ctx context.Context, msg kafka.Message, tracker *watermark.PartitionWatermark, reason error) {
	c.sendToDLQ(ctx, reason, msg)
	c.commitProgress(ctx, msg, tracker)
}

// sendToDLQ writes msg to the DLQ with retries. Errors are logged; callers always
// advance the watermark regardless so the partition is not blocked permanently.
func (c *Consumer) sendToDLQ(ctx context.Context, reason error, msg kafka.Message) {
	if c.dlq == nil {
		slog.ErrorContext(ctx, "DLQ not configured; message discarded",
			"partition", msg.Partition, "offset", msg.Offset, "reason", reason)
		return
	}
	delay := 2 * time.Second
	for i := 0; i < c.maxDLQRetries; i++ {
		if err := c.dlq.Write(ctx, reason, msg); err == nil {
			return
		}
		slog.ErrorContext(ctx, "DLQ write failed",
			"attempt", i+1, "partition", msg.Partition, "offset", msg.Offset)
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return
		}
	}
	slog.ErrorContext(ctx, "DLQ exhausted; message permanently lost",
		"partition", msg.Partition, "offset", msg.Offset, "reason", reason)
}

// getOrInitTracker returns the existing PartitionWatermark for a partition, or creates
// one starting at startOffset if this is the first message seen for that partition.
func (c *Consumer) getOrInitTracker(partition int, startOffset int64) *watermark.PartitionWatermark {
	actual, _ := c.watermarks.LoadOrStore(partition, watermark.NewPartitionWatermark(startOffset))
	return actual.(*watermark.PartitionWatermark)
}

// hashAccountID maps an account ID string to a stable uint32 for virtual-worker routing.
func hashAccountID(id string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(id))
	return h.Sum32()
}

// idempotencyKey produces a stable unique key from the Kafka message position.
// Business-level duplicate events (same AccountID at two offsets) are handled
// by domain errors in the application layer.
func idempotencyKey(msg kafka.Message) string {
	return fmt.Sprintf("%s:%d:%d", msg.Topic, msg.Partition, msg.Offset)
}
