package storage

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	eventstorage "github.com/meindokuse/cloud-drive/analytic-service/internal/infrastructure/storage/event"
	"github.com/meindokuse/cloud-drive/analytic-service/config"
)

type Registry struct {
	Event *eventstorage.Storage
}

func NewRegistry(ctx context.Context, conn driver.Conn, cfg config.BatcherConfig) (*Registry, error) {
	if err := eventstorage.EnsureSchema(ctx, conn); err != nil {
		return nil, fmt.Errorf("ensure schema: %w", err)
	}

	eventStorage, err := eventstorage.NewStorage(conn, cfg.Size, cfg.FlushInterval, cfg.ChannelBuffer)
	if err != nil {
		return nil, fmt.Errorf("event storage: %w", err)
	}

	return &Registry{Event: eventStorage}, nil
}
