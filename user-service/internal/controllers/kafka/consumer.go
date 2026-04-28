package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	authevents "github.com/meindokuse/cloud-drive/common/events/auth-service"
	domainuser "github.com/meindokuse/cloud-drive/user-service/internal/domain/user"
	usecase "github.com/meindokuse/cloud-drive/user-service/internal/usecase/user"
)

// UserService — интерфейс usecase для создания пользователей.
type UserService interface {
	CreateUser(ctx context.Context, in usecase.CreateUserInput) (*usecase.UserOutput, error)
}

// Consumer обрабатывает события из Kafka топика accounts.events.
type Consumer struct {
	reader      *kafka.Reader
	userService UserService
	logger      *slog.Logger
}

// NewConsumer создаёт новый Kafka consumer.
func NewConsumer(brokers []string, topic, groupID string, userService UserService, logger *slog.Logger) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1e3,  // 1KB
		MaxBytes:       10e6, // 10MB
		MaxWait:        500 * time.Millisecond,
		CommitInterval: 0, // manual commit
		StartOffset:    kafka.FirstOffset,
	})

	return &Consumer{
		reader:      reader,
		userService: userService,
		logger:      logger,
	}
}

// Start запускает consumer loop.
func (c *Consumer) Start(ctx context.Context) error {
	c.logger.InfoContext(ctx, "starting kafka consumer",
		"topic", c.reader.Config().Topic,
		"group_id", c.reader.Config().GroupID,
	)

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				c.logger.InfoContext(ctx, "context canceled, stopping consumer")
				return nil
			}
			c.logger.ErrorContext(ctx, "failed to fetch message", "error", err)
			continue
		}

		if err := c.handleMessage(ctx, msg); err != nil {
			c.logger.ErrorContext(ctx, "failed to handle message",
				"error", err,
				"partition", msg.Partition,
				"offset", msg.Offset,
			)
			// Не коммитим — сообщение будет перечитано
			continue
		}

		// Успешно обработали — коммитим offset
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.ErrorContext(ctx, "failed to commit offset",
				"error", err,
				"partition", msg.Partition,
				"offset", msg.Offset,
			)
		}
	}
}

// handleMessage обрабатывает одно сообщение из Kafka.
func (c *Consumer) handleMessage(ctx context.Context, msg kafka.Message) error {
	// Десериализуем событие
	var event authevents.AccountCreated
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		c.logger.ErrorContext(ctx, "failed to unmarshal event",
			"error", err,
			"partition", msg.Partition,
			"offset", msg.Offset,
		)
		// Poison pill — логируем и коммитим чтобы не застрять
		return nil
	}

	c.logger.InfoContext(ctx, "received account.created event",
		"account_id", event.AccountID,
		"username", event.Username,
		"email", event.Email,
	)

	// Вызываем usecase для создания User
	_, err := c.userService.CreateUser(ctx, usecase.CreateUserInput{
		ID:          &event.AccountID, // Используем тот же ID что и Account
		Username:    event.Username,
		Email:       &event.Email,
		Phone:       nil,
		DisplayName: event.DisplayName,
	})

	if err != nil {
		// Idempotency: если пользователь уже существует — это OK
		if errors.Is(err, domainuser.ErrUsernameAlreadyTaken) {
			c.logger.WarnContext(ctx, "user already exists (idempotent retry)",
				"account_id", event.AccountID,
				"username", event.Username,
			)
			return nil // Считаем успешной обработкой
		}

		// Другие ошибки — retry
		return fmt.Errorf("create user: %w", err)
	}

	c.logger.InfoContext(ctx, "user created successfully",
		"account_id", event.AccountID,
		"username", event.Username,
	)

	return nil
}

// Stop останавливает consumer.
func (c *Consumer) Stop(ctx context.Context) error {
	c.logger.InfoContext(ctx, "stopping kafka consumer")
	return c.reader.Close()
}
