package notification_test

import (
	"testing"
	"time"

	"transport-app/internal/domain/notification"
	"transport-app/internal/domain/types"
)

func TestNotification_MarkRead(t *testing.T) {
	n := notification.Notification{
		ID:        "notif-1",
		UserID:    types.UserID("usr-1"),
		Title:     "Dispatch Update",
		Message:   "Driver assigned to Trip #101",
		Channel:   notification.ChannelInApp,
		Status:    notification.NotificationUnread,
		CreatedAt: time.Now(),
	}

	if n.Status != notification.NotificationUnread {
		t.Fatalf("expected unread status")
	}

	n.MarkRead()

	if n.Status != notification.NotificationRead {
		t.Fatalf("expected read status, got %s", n.Status)
	}
	if n.ReadAt == nil {
		t.Fatalf("expected ReadAt timestamp to be set")
	}
}
