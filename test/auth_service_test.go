package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"transport-app/internal/domain"
	"transport-app/internal/service"
)

func TestAuthService_Login(t *testing.T) {
	db := NewTestDB(t)
	svc := NewTestServices(t, db)
	ctx := context.Background()

	result, err := svc.Auth.Login(ctx, service.LoginRequest{
		Email:    "admin@transport.local",
		Password: "admin123",
	})

	require.NoError(t, err)
	assert.Equal(t, "admin@transport.local", result.User.Email)
	assert.Equal(t, domain.RoleAdmin, result.User.Role.Name)
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	db := NewTestDB(t)
	svc := NewTestServices(t, db)

	_, err := svc.Auth.Login(context.Background(), service.LoginRequest{
		Email:    "admin@transport.local",
		Password: "wrongpassword",
	})

	assert.Error(t, err)
}

func TestAuthService_GetProfile(t *testing.T) {
	db := NewTestDB(t)
	svc := NewTestServices(t, db)

	user, err := svc.Auth.GetProfile(context.Background(), domain.UserID("765f6e4e-3b2a-4c1d-9e0f-1a2b3c4d5e6f"))
	require.NoError(t, err)
	assert.Equal(t, "admin@transport.local", user.Email)
	assert.Equal(t, "Admin User", user.Name)
}
