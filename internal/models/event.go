package models

import "encoding/json"

// EventEnvelope represents the data structure for an event.
type EventEnvelope struct {
	ID         string          `json:"id"`
	Source     string          `json:"source"`
	EventType  string          `json:"event_type"`
	SchemaType string          `json:"schema_type"` // Add this field
	Timestamp  int64           `json:"timestamp"`
	Data       json.RawMessage `json:"data"`
}
