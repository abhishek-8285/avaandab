package user_test

import (
	"testing"
	"time"

	"transport-app/internal/domain/types"
	"transport-app/internal/domain/user"
)

func TestDefaultRoleID(t *testing.T) {
	tests := []struct {
		role     user.RoleName
		expected int64
	}{
		{user.RoleAdmin, 1},
		{user.RoleDispatcher, 2},
		{user.RoleAccountant, 3},
		{user.RoleViewer, 4},
		{user.RoleName("unknown"), 2},
	}

	for _, tt := range tests {
		got := user.DefaultRoleID(tt.role)
		if got != tt.expected {
			t.Errorf("expected role %s to have ID %d, got %d", tt.role, tt.expected, got)
		}
	}
}

func TestUser_Struct(t *testing.T) {
	now := time.Now()
	phone := "+91-9876543210"

	u := user.User{
		ID:           types.UserID("usr-1"),
		Email:        "admin@avandab.com",
		PasswordHash: "hashed_pass",
		Name:         "System Admin",
		Phone:        &phone,
		Role:         user.Role{ID: 1, Name: user.RoleAdmin},
		Status:       user.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if u.Email != "admin@avandab.com" || u.Role.Name != user.RoleAdmin || u.Status != user.UserStatusActive {
		t.Fatalf("user struct mismatch")
	}
}
