package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"transport-app/internal/domain"
	"transport-app/internal/service"
)

// createTestAdmin provisions an admin user directly through the service,
// mirroring the env-based bootstrap flow (migrations no longer seed one).
func createTestAdmin(t *testing.T, svc *service.Services) domain.User {
	t.Helper()
	ctx := context.Background()

	created, err := svc.Users.CreateUserWithPassword(ctx, "admin@transport.local", "Admin User", "555-0100", "admin12345", 1, domain.UserStatusActive)
	require.NoError(t, err)
	return created
}

func TestAuthService_Login(t *testing.T) {
	db := NewTestDB(t)
	svc := NewTestServices(t, db)
	createTestAdmin(t, svc)
	ctx := context.Background()

	result, err := svc.Auth.Login(ctx, service.LoginRequest{
		Email:    "admin@transport.local",
		Password: "admin12345",
	})

	require.NoError(t, err)
	assert.Equal(t, "admin@transport.local", result.User.Email)
	assert.Equal(t, domain.RoleAdmin, result.User.Role.Name)
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	db := NewTestDB(t)
	svc := NewTestServices(t, db)
	createTestAdmin(t, svc)

	_, err := svc.Auth.Login(context.Background(), service.LoginRequest{
		Email:    "admin@transport.local",
		Password: "wrongpassword",
	})

	assert.Error(t, err)
}

func TestAuthService_GetProfile(t *testing.T) {
	db := NewTestDB(t)
	svc := NewTestServices(t, db)
	admin := createTestAdmin(t, svc)

	user, err := svc.Auth.GetProfile(context.Background(), admin.ID)
	require.NoError(t, err)
	assert.Equal(t, "admin@transport.local", user.Email)
	assert.Equal(t, "Admin User", user.Name)
}
