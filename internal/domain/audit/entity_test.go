package audit_test

import (
	"testing"
	"time"

	"transport-app/internal/domain/audit"
	"transport-app/internal/domain/types"
)

func TestAuditLog_Struct(t *testing.T) {
	now := time.Now()
	uid := types.UserID("usr-1")
	rec := "trp-101"
	ip := "192.168.1.1"

	al := audit.AuditLog{
		ID:        types.FileID("log-1"),
		UserID:    &uid,
		Action:    "UPDATE_TRIP_STATUS",
		TableName: "trips",
		RecordID:  &rec,
		IPAddress: &ip,
		CreatedAt: now,
	}

	if al.Action != "UPDATE_TRIP_STATUS" || *al.RecordID != "trp-101" || *al.UserID != uid {
		t.Fatalf("audit log struct mismatch")
	}
}
