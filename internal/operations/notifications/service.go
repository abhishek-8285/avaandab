package notifications

import (
	"context"
	"fmt"
	"log"
	"time"

	"transport-app/internal/shared/ports"
)

type Notification struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Type      string    `json:"type"`
	Recipient string    `json:"recipient"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

type Service struct {
	inAppStore map[string][]Notification
}

func NewService() *Service {
	return &Service{
		inAppStore: make(map[string][]Notification),
	}
}

func (s *Service) SendEmail(ctx context.Context, msg ports.NotificationMessage) error {
	// Logger / Email Adapter stub ready for SMTP/SendGrid driver
	log.Printf("[NOTIFICATION:EMAIL] To: %s | Subject: %s | Body: %s", msg.Recipient, msg.Subject, msg.Body)
	return nil
}

func (s *Service) SendInApp(ctx context.Context, msg ports.NotificationMessage) error {
	notif := Notification{
		ID:        fmt.Sprintf("notif_%d", time.Now().UnixNano()),
		TenantID:  msg.TenantID,
		UserID:    msg.UserID,
		Type:      string(ports.NotificationTypeInApp),
		Recipient: msg.Recipient,
		Subject:   msg.Subject,
		Body:      msg.Body,
		Read:      false,
		CreatedAt: time.Now(),
	}

	key := msg.UserID
	if key == "" {
		key = msg.TenantID
	}
	s.inAppStore[key] = append(s.inAppStore[key], notif)
	log.Printf("[NOTIFICATION:IN_APP] User/Tenant: %s | Subject: %s", key, msg.Subject)
	return nil
}

func (s *Service) SendSMS(ctx context.Context, msg ports.NotificationMessage) error {
	return fmt.Errorf("SMS notification channel not configured yet")
}

func (s *Service) SendPush(ctx context.Context, msg ports.NotificationMessage) error {
	return fmt.Errorf("Push notification channel not configured yet")
}

func (s *Service) SendWebhook(ctx context.Context, msg ports.NotificationMessage) error {
	return fmt.Errorf("Webhook notification channel not configured yet")
}

var _ ports.NotificationService = (*Service)(nil)
