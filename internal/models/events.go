package models

import "time"

// Event represents the payload received from the client.
type Event struct {
	Source    string    `json:"source"`
	UserID    string    `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}
