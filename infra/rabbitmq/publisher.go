package rabbitmq

import (
	"auction/pkg/events"
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// RabbitMQPublisher implements the events.Publisher interface
type RabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	service string
}

// NewRabbitMQPublisher creates a new RabbitMQ publisher
func NewRabbitMQPublisher(url, service string) (*RabbitMQPublisher, error) {
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

	// Enable publisher confirms for reliability
	if err := channel.Confirm(false); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to enable publisher confirms: %w", err)
	}

	zap.L().Info("RabbitMQ publisher connected successfully")

	return &RabbitMQPublisher{
		conn:    conn,
		channel: channel,
		service: service,
	}, nil
}

// DeclareExchange declares a topic exchange if it doesn't exist
func (p *RabbitMQPublisher) DeclareExchange(exchange string) error {
	return p.channel.ExchangeDeclare(
		exchange, // name
		"topic",  // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
}

// Publish publishes an event to the specified exchange
func (p *RabbitMQPublisher) Publish(ctx context.Context, exchange string, event *events.Event, headers events.Headers) error {
	// Ensure exchange exists
	if err := p.DeclareExchange(exchange); err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Serialize event to JSON
	body, err := event.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	// Prepare message headers
	messageHeaders := amqp.Table{
		"x-trace-id":       headers.TraceID,
		"x-correlation-id": headers.CorrelationID,
		"x-service":        p.service,
	}

	// Create the message
	msg := amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent, // Make message persistent
		Timestamp:    event.Timestamp,
		Headers:      messageHeaders,
	}

	// Get routing key from event
	routingKey := event.GetRoutingKey()

	// Create a dedicated channel for this publish operation to avoid confirmation conflicts
	publishCh, err := p.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to create publish channel: %w", err)
	}
	defer publishCh.Close()

	// Enable confirms on this channel
	if err := publishCh.Confirm(false); err != nil {
		return fmt.Errorf("failed to enable confirms: %w", err)
	}

	// Register for confirmations BEFORE publishing
	confirms := publishCh.NotifyPublish(make(chan amqp.Confirmation, 1))

	// Publish with context timeout
	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Publish the message
	if err := publishCh.PublishWithContext(
		publishCtx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		msg,
	); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	// Wait for confirmation
	select {
	case confirm := <-confirms:
		if !confirm.Ack {
			return fmt.Errorf("message was not acknowledged by broker")
		}
	case <-publishCtx.Done():
		return fmt.Errorf("publish confirmation timeout")
	}

	zap.L().Info("Event published successfully",
		zap.String("exchange", exchange),
		zap.String("routingKey", routingKey),
		zap.String("event", event.Event),
		zap.String("traceId", headers.TraceID),
	)

	return nil
}

// IsHealthy checks if the RabbitMQ connection is healthy
func (p *RabbitMQPublisher) IsHealthy() bool {
	if p == nil || p.conn == nil || p.channel == nil {
		return false
	}

	// Check if connection and channel are open
	return !p.conn.IsClosed() && !p.channel.IsClosed()
}

// Close closes the RabbitMQ connection
func (p *RabbitMQPublisher) Close() error {
	if p.channel != nil {
		if err := p.channel.Close(); err != nil {
			zap.L().Error("Failed to close channel", zap.Error(err))
		}
	}
	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			zap.L().Error("Failed to close connection", zap.Error(err))
			return err
		}
	}
	zap.L().Info("RabbitMQ publisher closed")
	return nil
}
