package event

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"

	"github.com/meindokuse/cloud-drive/analytic-service/internal/application/service/analytic/get_stats"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/application/service/analytic/list_events"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/domain/entity"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/infrastructure/storage/event/dao"
)

const createTableSQL = `
CREATE TABLE IF NOT EXISTS analytic_events (
    event_id    UUID,
    event_type  String,
    service     String,
    user_id     UUID,
    payload     String,
    timestamp   DateTime64(3, 'UTC'),
    created_at  DateTime64(3, 'UTC') DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (event_type, service, timestamp, event_id)
SETTINGS index_granularity = 8192`

type Storage struct {
	conn          driver.Conn
	ch            chan *entity.Event
	batchSize     int
	flushInterval time.Duration

	done      chan struct{}
	closeOnce sync.Once
}

func NewStorage(conn driver.Conn, batchSize int, flushInterval time.Duration, channelBuffer int) (*Storage, error) {
	if batchSize <= 0 {
		batchSize = 500
	}
	if flushInterval <= 0 {
		flushInterval = 5 * time.Second
	}
	if channelBuffer <= 0 {
		channelBuffer = 10000
	}
	s := &Storage{
		conn:          conn,
		ch:            make(chan *entity.Event, channelBuffer),
		batchSize:     batchSize,
		flushInterval: flushInterval,
		done:          make(chan struct{}),
	}
	return s, nil
}

func EnsureSchema(ctx context.Context, conn driver.Conn) error {
	return conn.Exec(ctx, createTableSQL)
}

// Enqueue adds an event to the in-memory buffer. Non-blocking by design;
// if the channel is full the event is dropped and a warning is logged.
func (s *Storage) Enqueue(e *entity.Event) {
	select {
	case s.ch <- e:
	default:
		slog.Warn("analytic batcher channel full; event dropped",
			"event_type", e.EventType(),
			"service", e.Service(),
		)
	}
}

// Start runs the batcher loop. Call it in a dedicated goroutine.
// It flushes to ClickHouse when either batchSize is reached or flushInterval elapses.
func (s *Storage) Start(ctx context.Context) {
	defer close(s.done)

	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()

	batch := make([]*entity.Event, 0, s.batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		s.flush(context.Background(), batch)
		batch = batch[:0]
	}

	for {
		select {
		case e, ok := <-s.ch:
			if !ok {
				flush()
				return
			}
			batch = append(batch, e)
			if len(batch) >= s.batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-ctx.Done():
			// Drain remaining events before exit.
			for {
				select {
				case e := <-s.ch:
					batch = append(batch, e)
					if len(batch) >= s.batchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		}
	}
}

// Stop closes the event channel so the batcher drains and exits,
// then waits until it has finished.
func (s *Storage) Stop(ctx context.Context) error {
	s.closeOnce.Do(func() { close(s.ch) })
	select {
	case <-s.done:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

func (s *Storage) flush(ctx context.Context, events []*entity.Event) {
	b, err := s.conn.PrepareBatch(ctx, "INSERT INTO analytic_events")
	if err != nil {
		slog.Error("clickhouse prepare batch", "error", err)
		return
	}
	for _, e := range events {
		if err := b.AppendStruct(dao.FromEntity(e)); err != nil {
			slog.Error("clickhouse append struct", "error", err)
		}
	}
	if err := b.Send(); err != nil {
		slog.Error("clickhouse batch send", "error", err, "batch_size", len(events))
		return
	}
	slog.Debug("flushed batch to clickhouse", "count", len(events))
}

// Insert writes a single event synchronously (used by the manual HTTP endpoint).
func (s *Storage) Insert(ctx context.Context, e *entity.Event) error {
	b, err := s.conn.PrepareBatch(ctx, "INSERT INTO analytic_events")
	if err != nil {
		return fmt.Errorf("prepare batch: %w", err)
	}
	if err := b.AppendStruct(dao.FromEntity(e)); err != nil {
		return fmt.Errorf("append: %w", err)
	}
	return b.Send()
}

func (s *Storage) GetStats(ctx context.Context, from, to *time.Time) (*get_stats.Output, error) {
	where, args := buildTimeWhere(from, to)

	var total uint64
	if err := s.conn.QueryRow(ctx,
		"SELECT count() FROM analytic_events WHERE "+where, args...,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("stats total: %w", err)
	}

	byType, err := s.queryGrouped(ctx, "event_type", where, args)
	if err != nil {
		return nil, fmt.Errorf("stats by event_type: %w", err)
	}

	bySvc, err := s.queryGrouped(ctx, "service", where, args)
	if err != nil {
		return nil, fmt.Errorf("stats by service: %w", err)
	}

	out := &get_stats.Output{Total: total}
	for _, r := range byType {
		out.ByEventType = append(out.ByEventType, get_stats.EventTypeCount{EventType: r.key, Count: r.count})
	}
	for _, r := range bySvc {
		out.ByService = append(out.ByService, get_stats.ServiceCount{Service: r.key, Count: r.count})
	}
	return out, nil
}

type groupRow struct {
	key   string
	count uint64
}

func (s *Storage) queryGrouped(ctx context.Context, col, where string, args []interface{}) ([]groupRow, error) {
	query := fmt.Sprintf(
		"SELECT %s, count() FROM analytic_events WHERE %s GROUP BY %s ORDER BY count() DESC LIMIT 100",
		col, where, col,
	)
	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []groupRow
	for rows.Next() {
		var r groupRow
		if err := rows.Scan(&r.key, &r.count); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (s *Storage) List(ctx context.Context, f list_events.Filter) ([]*entity.Event, int64, error) {
	where, args := buildListWhere(f)

	var total uint64
	if err := s.conn.QueryRow(ctx,
		"SELECT count() FROM analytic_events WHERE "+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("list count: %w", err)
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}

	query := fmt.Sprintf(
		"SELECT event_id, event_type, service, user_id, payload, timestamp, created_at"+
			" FROM analytic_events WHERE %s ORDER BY timestamp DESC LIMIT %d OFFSET %d",
		where, limit, f.Offset,
	)
	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list query: %w", err)
	}
	defer rows.Close()

	var events []*entity.Event
	for rows.Next() {
		d := &dao.EventDAO{}
		if err := rows.Scan(&d.EventID, &d.EventType, &d.Service, &d.UserID, &d.Payload, &d.Timestamp, &d.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan: %w", err)
		}
		events = append(events, dao.ToEntity(d))
	}
	return events, int64(total), rows.Err()
}

func (s *Storage) Delete(ctx context.Context, id uuid.UUID) error {
	return s.conn.Exec(ctx, "DELETE FROM analytic_events WHERE event_id = ?", id)
}

// buildTimeWhere builds a WHERE clause for time-range filtering only.
func buildTimeWhere(from, to *time.Time) (string, []interface{}) {
	var conds []string
	var args []interface{}
	if from != nil {
		conds = append(conds, "timestamp >= ?")
		args = append(args, *from)
	}
	if to != nil {
		conds = append(conds, "timestamp <= ?")
		args = append(args, *to)
	}
	if len(conds) == 0 {
		return "1=1", nil
	}
	return strings.Join(conds, " AND "), args
}

func buildListWhere(f list_events.Filter) (string, []interface{}) {
	var conds []string
	var args []interface{}
	if f.EventType != nil {
		conds = append(conds, "event_type = ?")
		args = append(args, *f.EventType)
	}
	if f.Service != nil {
		conds = append(conds, "service = ?")
		args = append(args, *f.Service)
	}
	if f.UserID != nil {
		conds = append(conds, "user_id = ?")
		args = append(args, *f.UserID)
	}
	if f.From != nil {
		conds = append(conds, "timestamp >= ?")
		args = append(args, *f.From)
	}
	if f.To != nil {
		conds = append(conds, "timestamp <= ?")
		args = append(args, *f.To)
	}
	if len(conds) == 0 {
		return "1=1", nil
	}
	return strings.Join(conds, " AND "), args
}
