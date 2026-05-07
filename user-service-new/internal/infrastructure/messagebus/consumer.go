package messagebus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	authevents "github.com/meindokuse/cloud-drive/common/events/auth-service"
	"github.com/segmentio/kafka-go"
)

type AccountCreatedHandler interface {
	HandleAccountCreated(ctx context.Context, event *authevents.AccountCreated) error
}

type Consumer struct {
	reader  *kafka.Reader
	handler AccountCreatedHandler
	logger  *slog.Logger
}

func NewConsumer(brokers []string, topic, groupID string, handler AccountCreatedHandler, logger *slog.Logger) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers, Topic: topic, GroupID: groupID, CommitInterval: 0, StartOffset: kafka.FirstOffset,
		}),
		handler: handler,
		logger:  logger,
	}
}

func (c *Consumer) Start(ctx context.Context) {
	c.logger.InfoContext(ctx, "kafka consumer started")
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			c.logger.ErrorContext(ctx, "fetch kafka message failed", "error", err)
			continue
		}
		var event authevents.AccountCreated
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			c.logger.ErrorContext(ctx, "unmarshal account.created failed", "error", err)
			_ = c.reader.CommitMessages(ctx, msg)
			continue
		}
		if err := c.handler.HandleAccountCreated(ctx, &event); err != nil {
			c.logger.ErrorContext(ctx, "handle account.created failed", "error", err, "account_id", event.AccountID)
			continue
		}
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.ErrorContext(ctx, "commit kafka message failed", "error", err)
		}
	}
}

func (c *Consumer) Stop(ctx context.Context) error {
	if c.reader == nil {
		return nil
	}
	if err := c.reader.Close(); err != nil {
		return fmt.Errorf("close kafka reader: %w", err)
	}
	return nil
}
