package messagebus

import (
	"github.com/meindokuse/cloud-drive/user-service-new/config"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/infrastructure/messagebus/subscriber"
)

type Registry struct {
	Consumer *subscriber.Consumer
}

func NewRegistry(
	cfg config.KafkaConfig,
	creator subscriber.UserCreator,
	idempotency subscriber.IdempotencyStore,
) *Registry {
	var consumer *subscriber.Consumer
	if cfg.Enabled {
		consumer = subscriber.NewConsumer(
			cfg.Brokers, cfg.Topic, cfg.GroupID,
			creator, idempotency,
			cfg.WorkersCount,
			subscriber.Options{
				MaxRetries:    cfg.MaxRetries,
				MaxDLQRetries: cfg.MaxDLQRetries,
			},
		)
	}
	return &Registry{Consumer: consumer}
}
