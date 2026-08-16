package service

import (
	"context"
	"time"

	"transport-app/internal/auth"
	"transport-app/internal/domain"
	userdomain "transport-app/internal/domain/user"
)

// AuthService handles user authentication and session management.
type AuthService struct {
	baseService
}

// LoginRequest contains the credentials for login.
type LoginRequest struct {
	Email    string
	Password string
	Remember bool
}

// LoginResult contains the result of a successful login.
type LoginResult struct {
	User         domain.User
	SessionToken string
}

// Login authenticates a user by email and password, creating a session.
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*LoginResult, error) {
	user, err := s.store.GetUserByEmail(ctx, req.Email)
	if err != nil {
		s.log.Error("login failed: user not found", "email", req.Email)
		return nil, domain.ErrInvalidCredentials
	}

	if err := auth.CheckPassword(req.Password, user.PasswordHash); err != nil {
		s.log.Error("login failed: bad password", "email", req.Email)
		return nil, domain.ErrInvalidCredentials
	}

	if user.Status != domain.UserStatusActive {
		return nil, domain.ErrUnauthorized
	}

	// Update last login
	if _, err := s.store.UpdateUserLastLogin(ctx, user.ID); err != nil {
		s.log.Warn("failed to update last login", "user_id", user.ID, "error", err)
	}

	// Create session in database
	token, err := auth.GenerateSecureToken()
	if err != nil {
		return nil, err
	}
	tokenHash := auth.HashToken(token)

	session := domain.Session{
		ID:        domain.SessionID(generateID()),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if _, err := s.store.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	s.log.Info("user logged in", "user_id", user.ID, "email", user.Email)

	s.logAudit(ctx, &user.ID, "login", "users", string(user.ID), nil, nil)

	return &LoginResult{
		User:         user,
		SessionToken: token,
	}, nil
}

// Logout deletes a user's session.
func (s *AuthService) Logout(ctx context.Context, token string) error {
	tokenHash := auth.HashToken(token)
	return s.store.DeleteSession(ctx, tokenHash)
}

// ChangePassword changes the user's password after verifying the old one.
func (s *AuthService) ChangePassword(ctx context.Context, userID domain.UserID, oldPassword, newPassword string) error {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := auth.CheckPassword(oldPassword, user.PasswordHash); err != nil {
		return domain.ErrInvalidCredentials
	}

	if len(newPassword) < userdomain.MinPasswordLength {
		return domain.ErrWeakPassword
	}

	hashed, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}

	_, err = s.store.UpdateUserPassword(ctx, userID, hashed)
	return err
}

// GetProfile returns a user's profile.
func (s *AuthService) GetProfile(ctx context.Context, userID domain.UserID) (domain.User, error) {
	return s.store.GetUserByID(ctx, userID)
}

// UpdateProfile updates a user's profile information.
func (s *AuthService) UpdateProfile(ctx context.Context, userID domain.UserID, name, phone, timezone string) (domain.User, error) {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}

	user.Name = sanitizeName(name)
	if phone != "" {
		user.Phone = &phone
	} else {
		user.Phone = nil
	}
	if timezone != "" {
		user.Timezone = timezone
	}

	return s.store.UpdateUser(ctx, user)
}

// VerifySession verifies a session token and returns the associated user.
func (s *AuthService) VerifySession(ctx context.Context, token string) (*domain.User, error) {
	tokenHash := auth.HashToken(token)
	session, err := s.store.GetSessionByToken(ctx, tokenHash)
	if err != nil {
		return nil, domain.ErrSessionExpired
	}

	if time.Now().After(session.ExpiresAt) {
		_ = s.store.DeleteSession(ctx, tokenHash)
		return nil, domain.ErrSessionExpired
	}

	user, err := s.store.GetUserByID(ctx, session.UserID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	return &user, nil
}
