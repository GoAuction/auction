package main

import (
	"auction/infra/postgres"
	"auction/infra/rabbitmq"
	"auction/internal/consumers"
	"auction/pkg/config"
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	zapConfig := zap.NewDevelopmentConfig()
	zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, _ := zapConfig.Build()
	zap.ReplaceGlobals(logger)
	defer logger.Sync()

	zap.L().Info("Auction Worker Service starting...")

	// Load application config
	appConfig := config.Read()
	zap.L().Info("Worker config loaded",
		zap.String("serviceName", appConfig.ServiceName),
		zap.String("rabbitMQURL", appConfig.RabbitMQURL),
	)

	// Validate RabbitMQ URL
	if appConfig.RabbitMQURL == "" {
		zap.L().Fatal("RABBITMQ_URL is required for worker service")
	}

	pgRepository := postgres.NewPgRepository(
		appConfig.PostgresHost,
		appConfig.PostgresDatabase,
		appConfig.PostgresUsername,
		appConfig.PostgresPassword,
		appConfig.PostgresPort,
	)

	// Initialize bid event handler
	bidHandler := consumers.NewBidEventHandler(
		pgRepository,
		zap.L(),
	)

	// Configure bid consumer
	// This consumes events from the "bid" service
	bidConsumerConfig := rabbitmq.ConsumerConfig{
		Exchange:       "auction.bid",         // Exchange where bid service publishes
		QueueName:      "auction.bid.all.v1",  // Queue name: {service}.{domain}.{events}.{version}
		RoutingKeys:    []string{"bid.*.v1"},  // Consume all bid events (placed, cancelled, won)
		ServiceName:    appConfig.ServiceName, // "auction"
		PrefetchCount:  10,                    // Prefetch 10 messages from queue
		WorkerPoolSize: 20,                    // Process up to 20 messages concurrently
	}

	bidConsumer, err := rabbitmq.NewConsumer(appConfig.RabbitMQURL, bidConsumerConfig)
	if err != nil {
		zap.L().Fatal("Failed to create bid consumer", zap.Error(err))
	}
	defer bidConsumer.Close()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start bid consumer in goroutine
	go func() {
		zap.L().Info("Starting bid event consumer...")
		if err := bidConsumer.Consume(ctx, bidHandler.HandleEvent); err != nil {
			if err != context.Canceled {
				zap.L().Error("Bid consumer error", zap.Error(err))
			}
		}
	}()

	// Start connection pool monitoring
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats := pgRepository.GetPoolStats()
				zap.L().Info("Connection pool stats",
					zap.Int("max_open", stats["max_open_connections"].(int)),
					zap.Int("open", stats["open_connections"].(int)),
					zap.Int("in_use", stats["in_use"].(int)),
					zap.Int("idle", stats["idle"].(int)),
					zap.Int64("wait_count", stats["wait_count"].(int64)),
					zap.Int64("wait_duration_ms", stats["wait_duration_ms"].(int64)),
				)
			}
		}
	}()

	zap.L().Info("Worker service started successfully. Waiting for events...")
	zap.L().Info("Consuming from exchanges",
		zap.String("bidExchange", "auction.bid"),
	)
	zap.L().Info("Press Ctrl+C to stop...")

	// Wait for shutdown signal
	<-sigChan
	zap.L().Info("Shutdown signal received, stopping worker service...")
	cancel()

	zap.L().Info("Worker service stopped gracefully")
}
