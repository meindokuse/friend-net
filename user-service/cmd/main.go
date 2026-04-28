package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	mongoadapter "github.com/meindokuse/cloud-drive/user-service/internal/adapters/mongo"
	"github.com/meindokuse/cloud-drive/user-service/internal/config"
	"github.com/meindokuse/cloud-drive/user-service/internal/controllers/handlers"
	"github.com/meindokuse/cloud-drive/user-service/internal/controllers/kafka"
	usecase "github.com/meindokuse/cloud-drive/user-service/internal/usecase/user"
)

func main() {
	// Загружаем конфиг
	cfg := config.MustLoad()

	// Инициализируем logger
	logger := setupLogger(cfg.Logger.Level)
	logger.Info("starting user-service", "env", cfg.Env)

	// Подключаемся к MongoDB
	mongoClient, err := connectMongo(cfg.Mongo)
	if err != nil {
		logger.Error("failed to connect to mongo", "error", err)
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := mongoClient.Disconnect(ctx); err != nil {
			logger.Error("failed to disconnect from mongo", "error", err)
		}
	}()

	// Инициализируем репозиторий и usecase
	userRepo, err := mongoadapter.NewUserRepository(mongoClient.Database(cfg.Mongo.Database))
	if err != nil {
		logger.Error("failed to create user repository", "error", err)
		os.Exit(1)
	}
	userService := usecase.NewService(userRepo)

	// Создаём HTTP handler и router
	userHandler := handlers.NewUserHandler(userService)
	router := handlers.NewRouter(userHandler)

	// Запускаем HTTP server
	httpServer := &http.Server{
		Addr:    cfg.Server.HTTPAddr,
		Handler: router,
	}

	// Запускаем Kafka consumer (если включен)
	var kafkaConsumer *kafka.Consumer
	if cfg.Kafka.Enabled {
		kafkaConsumer = kafka.NewConsumer(
			cfg.Kafka.Brokers,
			cfg.Kafka.Topic,
			cfg.Kafka.GroupID,
			userService,
			logger,
		)

		go func() {
			if err := kafkaConsumer.Start(context.Background()); err != nil {
				logger.Error("kafka consumer stopped with error", "error", err)
			}
		}()
	}

	// Запускаем HTTP server в отдельной горутине
	go func() {
		logger.Info("starting http server", "addr", cfg.Server.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down gracefully...")

	// Останавливаем HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server shutdown error", "error", err)
	}

	// Останавливаем Kafka consumer
	if kafkaConsumer != nil {
		if err := kafkaConsumer.Stop(context.Background()); err != nil {
			logger.Error("kafka consumer stop error", "error", err)
		}
	}

	logger.Info("user-service stopped")
}

// connectMongo подключается к MongoDB с retry логикой.
func connectMongo(cfg config.MongoConfig) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	clientOpts := options.Client().
		ApplyURI(cfg.URI).
		SetServerSelectionTimeout(cfg.Timeout)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	// Проверяем подключение
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}

	return client, nil
}

// setupLogger создаёт structured logger.
func setupLogger(level string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})

	return slog.New(handler)
}
