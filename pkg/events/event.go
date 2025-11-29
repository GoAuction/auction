package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	Event         string      `json:"event"`         // e.g., "item.created"
	Version       string      `json:"version"`       // e.g., "v1"
	Timestamp     time.Time   `json:"timestamp"`     // Event occurrence time
	Payload       interface{} `json:"payload"`       // The actual event data
	TraceID       string      `json:"traceId"`       // For distributed tracing
	CorrelationID string      `json:"correlationId"` // For request correlation
}

type Headers struct {
	TraceID       string
	CorrelationID string
	Service       string
}

func NewEvent(eventName, version string, payload interface{}, headers Headers) *Event {
	return &Event{
		Event:         eventName,
		Version:       version,
		Timestamp:     time.Now().UTC(),
		Payload:       payload,
		TraceID:       headers.TraceID,
		CorrelationID: headers.CorrelationID,
	}
}

func (e *Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

func (e *Event) GetRoutingKey() string {
	return e.Event + "." + e.Version
}

func GenerateTraceID() string {
	return uuid.New().String()
}

func GenerateCorrelationID() string {
	return uuid.New().String()
}
