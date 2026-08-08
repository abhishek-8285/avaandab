package audit

import (
	"context"
	"sync"
	"time"

	"transport-app/internal/shared/ports"
)

type LoginRecord struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	UserEmail string    `json:"user_email"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Success   bool      `json:"success"`
	Timestamp time.Time `json:"timestamp"`
}

type SecurityPolicy struct {
	AlwaysNotifyOnLogin bool
	NotifyOnNewDevice   bool
	NotifyOnNewIP       bool
}

type LoginAuditService struct {
	mu           sync.RWMutex
	history      []LoginRecord
	knownDevices map[string]map[string]bool // userID -> userAgent -> true
	notifSvc     ports.NotificationService
	policy       SecurityPolicy
}

func NewLoginAuditService(notifSvc ports.NotificationService, policy SecurityPolicy) *LoginAuditService {
	return &LoginAuditService{
		history:      make([]LoginRecord, 0),
		knownDevices: make(map[string]map[string]bool),
		notifSvc:     notifSvc,
		policy:       policy,
	}
}

func (s *LoginAuditService) RecordLogin(ctx context.Context, record LoginRecord) error {
	s.mu.Lock()
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}
	s.history = append(s.history, record)

	if !record.Success {
		s.mu.Unlock()
		return nil
	}

	userDevices, exists := s.knownDevices[record.UserID]
	if !exists {
		userDevices = make(map[string]bool)
		s.knownDevices[record.UserID] = userDevices
	}

	isNewDevice := !userDevices[record.UserAgent]
	userDevices[record.UserAgent] = true
	s.mu.Unlock()

	// Adaptive security notification check
	shouldNotify := s.policy.AlwaysNotifyOnLogin || (s.policy.NotifyOnNewDevice && isNewDevice)

	if shouldNotify && s.notifSvc != nil {
		body := "New login detected on your FlyFleet account.\n\n" +
			"Time: " + record.Timestamp.Format(time.RFC1123) + "\n" +
			"IP Address: " + record.IPAddress + "\n" +
			"Browser/Device: " + record.UserAgent + "\n\n" +
			"If this wasn't you, contact support immediately."

		_ = s.notifSvc.SendEmail(ctx, ports.NotificationMessage{
			UserID:    record.UserID,
			Recipient: record.UserEmail,
			Subject:   "Security Alert: New Login Detected",
			Body:      body,
			Type:      ports.NotificationTypeEmail,
		})
	}

	return nil
}

func (s *LoginAuditService) GetLoginHistory(userID string) []LoginRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var records []LoginRecord
	for _, r := range s.history {
		if r.UserID == userID {
			records = append(records, r)
		}
	}
	return records
}
