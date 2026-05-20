package models

import (
	"encoding/json"
	"time"
)

type Channel string

const (
	ChannelEmail Channel = "email"
	ChannelSMS   Channel = "sms"
	ChannelPush  Channel = "push"
)

type Status string

const (
	StatusQueued    Status = "queued"
	StatusInFlight  Status = "in_flight"
	StatusDelivered Status = "delivered"
	StatusFailed    Status = "failed"
	StatusDeadLet   Status = "dead_lettered"
)

type Notification struct {
	ID         string          `json:"id"`
	Channel    Channel         `json:"channel"`
	To         string          `json:"to"`
	Subject    string          `json:"subject,omitempty"`
	Body       string          `json:"body"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	Status     Status          `json:"status"`
	Attempts   int             `json:"attempts"`
	QueuedAt   time.Time       `json:"queued_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	LastError  string          `json:"last_error,omitempty"`
}

type DeliveryLog struct {
	NotificationID string    `json:"notification_id"`
	Attempt        int       `json:"attempt"`
	Status         Status    `json:"status"`
	Error          string    `json:"error,omitempty"`
	At             time.Time `json:"at"`
}
