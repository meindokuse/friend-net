package messagebus

import (
	"github.com/meindokuse/cloud-drive/analytic-service/config"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/application/service/analytic/ingest_event"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/infrastructure/messagebus/subscriber"
)

type Registry struct {
	Consumer *subscriber.Consumer
}

func NewRegistry(cfg config.KafkaConfig, ingester *ingest_event.Service) *Registry {
	if !cfg.Enabled {
		return &Registry{}
	}
	return &Registry{
		Consumer: subscriber.NewConsumer(
			cfg.Brokers,
			cfg.Topic,
			cfg.GroupID,
			ingester,
			cfg.WorkersCount,
		),
	}
}
