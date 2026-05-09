package subscriber

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"github.com/segmentio/kafka-go"

	authevents "github.com/meindokuse/cloud-drive/common/events/auth-service"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/create_user"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/domain/entity"
)

type UserCreator interface {
	Execute(ctx context.Context, in create_user.Input) (*entity.User, error)
}

type Consumer struct {
	reader  *kafka.Reader
	creator UserCreator
	logger  *slog.Logger
}

func NewConsumer(brokers []string, topic, groupID string, creator UserCreator, logger *slog.Logger) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			CommitInterval: 0,
			StartOffset:    kafka.FirstOffset,
		}),
		creator: creator,
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
		if err := c.handle(ctx, msg); err != nil {
			c.logger.ErrorContext(ctx, "handle message failed", "error", err)
			continue
		}
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.ErrorContext(ctx, "commit kafka message failed", "error", err)
		}
	}
}

func (c *Consumer) Stop(_ context.Context) error {
	if c.reader == nil {
		return nil
	}
	return c.reader.Close()
}

func (c *Consumer) handle(ctx context.Context, msg kafka.Message) error {
	var event authevents.AccountCreated
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		c.logger.ErrorContext(ctx, "unmarshal account.created failed", "error", err)
		return nil // skip malformed messages
	}

	username := strings.TrimSpace(event.Username)
	if username == "" {
		username = strings.Split(event.Email, "@")[0]
	}
	displayName := strings.TrimSpace(event.DisplayName)
	if displayName == "" {
		displayName = username
	}
	email := event.Email

	_, err := c.creator.Execute(ctx, create_user.Input{
		ID:          &event.AccountID,
		Username:    username,
		Email:       &email,
		DisplayName: displayName,
	})
	if err != nil {
		if errors.Is(err, entity.ErrUsernameAlreadyTaken) || errors.Is(err, entity.ErrEmailAlreadyTaken) {
			return nil // idempotent
		}
		return err
	}
	return nil
}
