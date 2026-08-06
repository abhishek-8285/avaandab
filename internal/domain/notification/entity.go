package notification

import (
	"time"

	"transport-app/internal/domain/types"
)

type NotificationChannel string

const (
	ChannelInApp NotificationChannel = "in_app"
	ChannelEmail NotificationChannel = "email"
	ChannelSMS   NotificationChannel = "sms"
	ChannelPush  NotificationChannel = "push"
)

type NotificationStatus string

const (
	NotificationUnread NotificationStatus = "unread"
	NotificationRead   NotificationStatus = "read"
)

// Notification represents a user notification.
type Notification struct {
	ID        string              `json:"id"`
	UserID    types.UserID        `json:"user_id"`
	Title     string              `json:"title"`
	Message   string              `json:"message"`
	Channel   NotificationChannel `json:"channel"`
	Status    NotificationStatus  `json:"status"`
	Link      *string             `json:"link,omitempty"`
	CreatedAt time.Time           `json:"created_at"`
	ReadAt    *time.Time          `json:"read_at,omitempty"`
}

func (n *Notification) MarkRead() {
	n.Status = NotificationRead
	now := time.Now()
	n.ReadAt = &now
}
