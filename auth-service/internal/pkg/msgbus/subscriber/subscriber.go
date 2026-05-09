package subscriber

import "github.com/IBM/sarama"

// Handler processes a consumed Kafka message.
type Handler interface {
	Handle(session sarama.ConsumerGroupSession, msg *sarama.ConsumerMessage) error
}
