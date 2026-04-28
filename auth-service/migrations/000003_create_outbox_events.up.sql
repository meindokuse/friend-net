-- Outbox таблица для CDC pattern (Debezium)
CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type TEXT NOT NULL,           -- 'account', 'oauth_account', etc.
    aggregate_id UUID NOT NULL,             -- ID сущности (account.id)
    event_type TEXT NOT NULL,               -- 'account.created', 'account.updated', etc.
    payload JSONB NOT NULL,                 -- JSON с данными события
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ                -- NULL = не обработано, NOT NULL = обработано
);

-- Индекс для быстрого поиска необработанных событий (если будет polling вместо CDC)
CREATE INDEX IF NOT EXISTS idx_outbox_events_processed ON outbox_events (processed_at) WHERE processed_at IS NULL;

-- Индекс для поиска по aggregate
CREATE INDEX IF NOT EXISTS idx_outbox_events_aggregate ON outbox_events (aggregate_type, aggregate_id);

-- Индекс для сортировки по времени создания
CREATE INDEX IF NOT EXISTS idx_outbox_events_created_at ON outbox_events (created_at);

COMMENT ON TABLE outbox_events IS 'Outbox pattern table for CDC with Debezium';
COMMENT ON COLUMN outbox_events.aggregate_type IS 'Type of aggregate (account, oauth_account, etc.)';
COMMENT ON COLUMN outbox_events.aggregate_id IS 'ID of the aggregate entity';
COMMENT ON COLUMN outbox_events.event_type IS 'Event type (account.created, account.updated, etc.)';
COMMENT ON COLUMN outbox_events.payload IS 'JSON payload with event data';
COMMENT ON COLUMN outbox_events.processed_at IS 'Timestamp when event was processed (NULL = pending)';
