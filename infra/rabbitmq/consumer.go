package rabbitmq

import (
	"auction/pkg/events"
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// EventHandler is a function that processes events
type EventHandler func(ctx context.Context, event *events.Event) error

// Consumer represents a RabbitMQ consumer
type Consumer struct {
	conn        *amqp.Connection
	channel     *amqp.Channel
	queueName   string
	serviceName string
}

// ConsumerConfig holds configuration for setting up a consumer
type ConsumerConfig struct {
	Exchange     string   // e.g., "auction.item"
	QueueName    string   // e.g., "payment.item.created.v1"
	RoutingKeys  []string // e.g., ["item.created.v1"]
	ServiceName  string   // e.g., "payment"
	PrefetchCount int     // Number of messages to prefetch (0 = unlimited)
}

// NewConsumer creates a new RabbitMQ consumer
func NewConsumer(url string, config ConsumerConfig) (*Consumer, error) {
	// Connect to RabbitMQ with retry logic
	var conn *amqp.Connection
	var err error

	for i := 0; i < 5; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		zap.L().Warn("Failed to connect to RabbitMQ, retrying...",
			zap.Int("attempt", i+1),
			zap.Error(err))
		time.Sleep(time.Second * time.Duration(i+1))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ after retries: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Set QoS (prefetch count)
	prefetchCount := config.PrefetchCount
	if prefetchCount == 0 {
		prefetchCount = 10 // Default prefetch
	}
	if err := channel.Qos(prefetchCount, 0, false); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	// Declare exchange
	if err := channel.ExchangeDeclare(
		config.Exchange,
		"topic",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare dead letter exchange for this queue
	dlxName := config.Exchange + ".dlx"
	if err := channel.ExchangeDeclare(
		dlxName,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare DLX: %w", err)
	}

	// Declare main queue with DLQ configuration
	queueArgs := amqp.Table{
		"x-dead-letter-exchange": dlxName,
	}
	queue, err := channel.QueueDeclare(
		config.QueueName,
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		queueArgs, // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Declare dead letter queue
	dlqName := config.QueueName + ".dlq"
	_, err = channel.QueueDeclare(
		dlqName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare DLQ: %w", err)
	}

	// Bind DLQ to DLX
	for _, routingKey := range config.RoutingKeys {
		if err := channel.QueueBind(
			dlqName,
			routingKey,
			dlxName,
			false,
			nil,
		); err != nil {
			channel.Close()
			conn.Close()
			return nil, fmt.Errorf("failed to bind DLQ: %w", err)
		}
	}

	// Bind queue to exchange with routing keys
	for _, routingKey := range config.RoutingKeys {
		if err := channel.QueueBind(
			queue.Name,
			routingKey,
			config.Exchange,
			false,
			nil,
		); err != nil {
			channel.Close()
			conn.Close()
			return nil, fmt.Errorf("failed to bind queue: %w", err)
		}
	}

	zap.L().Info("RabbitMQ consumer created successfully",
		zap.String("queue", config.QueueName),
		zap.String("exchange", config.Exchange),
		zap.Strings("routingKeys", config.RoutingKeys),
	)

	return &Consumer{
		conn:        conn,
		channel:     channel,
		queueName:   config.QueueName,
		serviceName: config.ServiceName,
	}, nil
}

// Consume starts consuming messages from the queue
func (c *Consumer) Consume(ctx context.Context, handler EventHandler) error {
	msgs, err := c.channel.Consume(
		c.queueName,
		c.serviceName, // consumer tag
		false,         // auto-ack (false = manual ack)
		false,         // exclusive
		false,         // no-local
		false,         // no-wait
		nil,           // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	zap.L().Info("Started consuming messages", zap.String("queue", c.queueName))

	for {
		select {
		case <-ctx.Done():
			zap.L().Info("Consumer context cancelled, stopping...")
			return ctx.Err()
		case msg, ok := <-msgs:
			if !ok {
				zap.L().Warn("Message channel closed")
				return fmt.Errorf("message channel closed")
			}

			c.handleMessage(ctx, msg, handler)
		}
	}
}

// handleMessage processes a single message
func (c *Consumer) handleMessage(ctx context.Context, msg amqp.Delivery, handler EventHandler) {
	// Extract headers for logging
	traceID, _ := msg.Headers["x-trace-id"].(string)
	correlationID, _ := msg.Headers["x-correlation-id"].(string)
	service, _ := msg.Headers["x-service"].(string)

	zap.L().Info("Received message",
		zap.String("queue", c.queueName),
		zap.String("routingKey", msg.RoutingKey),
		zap.String("traceId", traceID),
		zap.String("correlationId", correlationID),
		zap.String("sourceService", service),
	)

	// Parse the event
	var event events.Event
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		zap.L().Error("Failed to unmarshal event",
			zap.Error(err),
			zap.String("traceId", traceID),
		)
		// Reject and don't requeue - malformed messages go to DLQ
		msg.Nack(false, false)
		return
	}

	// Process the event with timeout
	processCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := handler(processCtx, &event); err != nil {
		zap.L().Error("Failed to process event",
			zap.Error(err),
			zap.String("event", event.Event),
			zap.String("traceId", traceID),
		)
		// Reject and don't requeue - failed processing goes to DLQ
		msg.Nack(false, false)
		return
	}

	// Acknowledge successful processing
	if err := msg.Ack(false); err != nil {
		zap.L().Error("Failed to acknowledge message",
			zap.Error(err),
			zap.String("traceId", traceID),
		)
	} else {
		zap.L().Info("Successfully processed event",
			zap.String("event", event.Event),
			zap.String("traceId", traceID),
		)
	}
}

// Close closes the consumer connection
func (c *Consumer) Close() error {
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			zap.L().Error("Failed to close channel", zap.Error(err))
		}
	}
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			zap.L().Error("Failed to close connection", zap.Error(err))
			return err
		}
	}
	zap.L().Info("RabbitMQ consumer closed")
	return nil
}
