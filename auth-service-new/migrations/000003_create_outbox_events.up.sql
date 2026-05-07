-- Outbox table for CDC pattern (without Debezium, with polling flusher)
CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type TEXT NOT NULL,           -- 'account', 'oauth_account', etc.
    aggregate_id UUID NOT NULL,             -- ID of the entity (account.id)
    event_type TEXT NOT NULL,               -- 'account.created', 'account.updated', etc.
    payload JSONB NOT NULL,                 -- JSON with event data
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ                -- NULL = not processed, NOT NULL = processed
);

-- Index for fast search of unprocessed events (polling)
CREATE INDEX IF NOT EXISTS idx_outbox_events_processed ON outbox_events (processed_at) WHERE processed_at IS NULL;

-- Index for search by aggregate
CREATE INDEX IF NOT EXISTS idx_outbox_events_aggregate ON outbox_events (aggregate_type, aggregate_id);

-- Index for sorting by creation time
CREATE INDEX IF NOT EXISTS idx_outbox_events_created_at ON outbox_events (created_at);

COMMENT ON TABLE outbox_events IS 'Outbox pattern table for CDC with polling flusher';
COMMENT ON COLUMN outbox_events.aggregate_type IS 'Type of aggregate (account, oauth_account, etc.)';
COMMENT ON COLUMN outbox_events.aggregate_id IS 'ID of the aggregate entity';
COMMENT ON COLUMN outbox_events.event_type IS 'Event type (account.created, account.updated, etc.)';
COMMENT ON COLUMN outbox_events.payload IS 'JSON payload with event data';
COMMENT ON COLUMN outbox_events.processed_at IS 'Timestamp when event was processed (NULL = pending)';
