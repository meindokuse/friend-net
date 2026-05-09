package messagebus

import (
	"context"

	"github.com/IBM/sarama"
	"github.com/meindokuse/cloud-drive/auth-service-new/config"
)

// Registry contains message bus components
type Registry struct {
	Producer *Producer
}

// NewRegistry creates a new messagebus registry
func NewRegistry(cfg config.KafkaConfig) *Registry {
	var producer *Producer
	if cfg.Enabled {
		producer = NewProducer(cfg)
	}
	return &Registry{
		Producer: producer,
	}
}

// Producer implements Kafka producer
type Producer struct {
	producer sarama.SyncProducer
	topic    string
}

// NewProducer creates a new Kafka producer
func NewProducer(cfg config.KafkaConfig) *Producer {
	return &Producer{
		producer: MustSyncProducer(cfg),
		topic:    cfg.Topic,
	}
}

// Close closes the producer
func (p *Producer) Close(_ context.Context) error {
	if p.producer == nil {
		return nil
	}
	return p.producer.Close()
}

// SyncProducer exposes underlying Sarama producer for adapters.
func (p *Producer) SyncProducer() sarama.SyncProducer {
	if p == nil {
		return nil
	}
	return p.producer
}
