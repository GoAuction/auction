package events

import (
	"context"
)

// Publisher defines the interface for publishing domain events
type Publisher interface {
	// Publish publishes an event to the message broker
	Publish(ctx context.Context, exchange string, event *Event, headers Headers) error

	// Close closes the publisher connection
	Close() error
}
