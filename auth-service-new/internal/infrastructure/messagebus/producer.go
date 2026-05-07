package messagebus

import (
	"log"

	"github.com/IBM/sarama"
	"github.com/meindokuse/cloud-drive/auth-service-new/config"
)

// MustSyncProducer creates a sync producer or panics
func MustSyncProducer(cfg config.KafkaConfig) sarama.SyncProducer {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Partitioner = sarama.NewRoundRobinPartitioner

	producer, err := sarama.NewSyncProducer(cfg.Brokers, saramaConfig)
	if err != nil {
		log.Fatalf("failed to create kafka producer: %v", err)
	}

	return producer
}
