package producer

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/event"
)

// Producer sends domain events to Kafka.
type Producer struct {
	sp sarama.SyncProducer
	
}

// New wraps a sarama.SyncProducer.
func New(sp sarama.SyncProducer) *Producer {
	return &Producer{sp: sp}
}

// Flush implements event.Flusher by publishing all events to Kafka.
func (p *Producer) Flush(_ context.Context, events event.Events) error {
	msgs := make([]*sarama.ProducerMessage, 0, len(events))
	for _, e := range events {
		msg := &sarama.ProducerMessage{
			Topic: e.Topic,
			Key:   sarama.StringEncoder(e.Key),
			Value: sarama.ByteEncoder(e.Payload),
		}
		for k, v := range e.Headers {
			msg.Headers = append(msg.Headers, sarama.RecordHeader{
				Key:   []byte(k),
				Value: []byte(v),
			})
		}
		msgs = append(msgs, msg)
	}
	if err := p.sp.SendMessages(msgs); err != nil {
		return fmt.Errorf("send messages: %w", err)
	}
	return nil
}
