# CLAUDE.md — analytic-service

Module: `github.com/meindokuse/cloud-drive/analytic-service`

## Purpose

Consumes `analytic.event` Kafka topic, batches events into ClickHouse, and exposes HTTP endpoints for manual stats management.

## Build & Run

```bash
# Run locally (from analytic-service/)
go run ./cmd

# Run with Docker Compose (ClickHouse + service)
docker compose up --build

# Run tests
go test ./...
```

## Configuration

Config is loaded from `config/config.yaml` (override with `CONFIG_PATH` env var). Key env vars:

| Variable | Default | Description |
|---|---|---|
| `HTTP_ADDR` | `:8082` | HTTP listen address |
| `CLICKHOUSE_ADDRS` | `localhost:9000` | ClickHouse native TCP address(es), comma-separated |
| `CLICKHOUSE_DATABASE` | `analytics` | ClickHouse database name |
| `KAFKA_BROKERS` | `localhost:9092` | Kafka broker addresses |
| `KAFKA_TOPIC` | `analytic.event` | Topic to consume |
| `KAFKA_GROUP_ID` | `analytic-service` | Consumer group |
| `KAFKA_ENABLED` | `true` | Disable to run HTTP-only mode |
| `BATCHER_SIZE` | `500` | Flush to ClickHouse when batch reaches this size |
| `BATCHER_FLUSH_INTERVAL` | `5s` | Flush interval regardless of batch size |
| `BATCHER_CHANNEL_BUFFER` | `10000` | In-memory channel capacity before drops |

## Architecture: Clean Architecture, 4 layers

Same dependency rules as other services in this workspace.

```
Presentation (internal/app/analytic/v1/)
    → Application (internal/application/service/analytic/)
        → Domain (internal/domain/entity/)
            ← Infrastructure (internal/infrastructure/)
```

### Kafka → ClickHouse pipeline

```
Kafka topic analytic.event
    │
    ▼
subscriber.Consumer (8 virtual workers, hash-routed by msg.Key)
    │  at-least-once delivery (auto-commit every 1s)
    ▼
ingest_event.Service.Execute()   ← application layer
    │  validates event_type + service required
    ▼
event.Storage.Enqueue()          ← infrastructure, non-blocking
    │  buffered channel (10 000 cap)
    ▼
batcher goroutine (Start/Stop)
    │  flushes on size=500 OR every 5s
    ▼
ClickHouse PrepareBatch INSERT
```

### Batcher lifecycle

- `Storage.Start(ctx)` runs in a goroutine started by `app.go`
- On SIGTERM: HTTP shuts down → Kafka consumer drains → `Storage.Stop()` closes channel → batcher flushes remaining events → ClickHouse connection closes

### ClickHouse schema

Created automatically on first start (`EnsureSchema`):

```sql
CREATE TABLE IF NOT EXISTS analytic_events (
    event_id    UUID,
    event_type  String,
    service     String,
    user_id     UUID,        -- zero UUID = no user
    payload     String,      -- raw JSON string
    timestamp   DateTime64(3, 'UTC'),
    created_at  DateTime64(3, 'UTC') DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (event_type, service, timestamp, event_id)
```

## HTTP API

All routes are under `/analytics`. Traefik routes `Host(localhost) && PathPrefix(/analytics)` with `jwt-auth` middleware.

| Method | Path | Description |
|---|---|---|
| `GET` | `/analytics/stats` | Aggregated counts by event_type and service |
| `GET` | `/analytics/events` | Paginated event list with filters |
| `POST` | `/analytics/events` | Manually insert an event (enqueues to batcher) |
| `DELETE` | `/analytics/events/{id}` | Delete event by UUID (async ClickHouse mutation, returns 202) |

### Query params for GET /analytics/stats

`from`, `to` — RFC3339 timestamps (optional)

### Query params for GET /analytics/events

`event_type`, `service`, `user_id`, `from`, `to`, `limit` (default 50, max 1000), `offset` (default 0)

### POST /analytics/events body

```json
{
  "event_type": "user.login",
  "service": "auth-service",
  "user_id": "uuid (optional)",
  "payload": { "any": "json" },
  "timestamp": "2026-01-01T00:00:00Z (optional, defaults to now)"
}
```

## Producing events from other services

Other services publish to topic `analytic.event` using the shared event type from `common`:

```go
import analyticevents "github.com/meindokuse/cloud-drive/common/events/analytic"

event := analyticevents.AnalyticEvent{
    EventID:   uuid.New(),
    EventType: "user.login",
    Service:   "auth-service",
    UserID:    &userID,
    Payload:   json.RawMessage(`{"method":"password"}`),
    Timestamp: time.Now().UTC(),
}
// publish to Kafka topic "analytic.event"
```
