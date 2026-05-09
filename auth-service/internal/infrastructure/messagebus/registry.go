package messagebus

import (
	"github.com/meindokuse/cloud-drive/auth-service-new/config"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/infrastructure/messagebus/producer"
)

// Registry contains message bus components.
type Registry struct {
	Producer *producer.Producer
}

// NewRegistry creates a new messagebus registry.
func NewRegistry(cfg config.KafkaConfig) *Registry {
	var p *producer.Producer
	if cfg.Enabled {
		p = producer.New(cfg)
	}
	return &Registry{Producer: p}
}
