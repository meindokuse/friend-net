package producer

import (
	"context"

	"github.com/IBM/sarama"

	"github.com/meindokuse/cloud-drive/auth-service-new/config"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/connector/kafka"
)

// Producer wraps a Kafka SyncProducer for the auth service.
type Producer struct {
	producer sarama.SyncProducer
	topic    string
}

// New creates a new Kafka Producer using the given config.
func New(cfg config.KafkaConfig) *Producer {
	return &Producer{
		producer: kafka.MustSyncProducer(cfg),
		topic:    cfg.Topic,
	}
}

// SyncProducer exposes the underlying sarama producer.
func (p *Producer) SyncProducer() sarama.SyncProducer {
	if p == nil {
		return nil
	}
	return p.producer
}

// Close closes the producer connection.
func (p *Producer) Close(_ context.Context) error {
	if p.producer == nil {
		return nil
	}
	return p.producer.Close()
}
