package ports

import "context"

type NotificationType string

const (
	NotificationTypeEmail   NotificationType = "EMAIL"
	NotificationTypeInApp   NotificationType = "IN_APP"
	NotificationTypeSMS     NotificationType = "SMS"
	NotificationTypePush    NotificationType = "PUSH"
	NotificationTypeWebhook NotificationType = "WEBHOOK"
)

type NotificationMessage struct {
	TenantID  string                 `json:"tenant_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	Recipient string                 `json:"recipient"`
	Subject   string                 `json:"subject"`
	Body      string                 `json:"body"`
	Type      NotificationType       `json:"type"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type NotificationService interface {
	SendEmail(ctx context.Context, msg NotificationMessage) error
	SendInApp(ctx context.Context, msg NotificationMessage) error
	SendSMS(ctx context.Context, msg NotificationMessage) error
	SendPush(ctx context.Context, msg NotificationMessage) error
	SendWebhook(ctx context.Context, msg NotificationMessage) error
}
