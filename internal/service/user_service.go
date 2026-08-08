package service

import (
	"context"
	"fmt"

	"transport-app/internal/auth"
	"transport-app/internal/domain"
	"transport-app/internal/repository"
)

// UserService handles user management.
type UserService struct {
	baseService
}

// CreateUser creates a new user with a default password.
func (s *UserService) CreateUser(ctx context.Context, email, name, phone string, roleID int64, status domain.UserStatus) (domain.User, error) {
	if email == "" {
		return domain.User{}, domain.ErrUserEmailRequired
	}
	if phone == "" {
		return domain.User{}, domain.ErrUserPhoneRequired
	}

	if _, err := s.store.GetUserByEmail(ctx, email); err == nil {
		return domain.User{}, domain.ErrUserEmailExists
	}

	if _, err := s.store.GetRoleByID(ctx, roleID); err != nil {
		return domain.User{}, fmt.Errorf("invalid role")
	}

	hashed, err := auth.HashPassword(email + "-changeme")
	if err != nil {
		return domain.User{}, err
	}

	user := domain.User{
		ID:           domain.UserID(generateID()),
		Email:        email,
		PasswordHash: hashed,
		Name:         sanitizeName(name),
		Phone:        &phone,
		Role:         domain.Role{ID: roleID},
		Status:       status,
	}

	created, err := s.store.CreateUser(ctx, user)
	if err != nil {
		return domain.User{}, err
	}

	s.log.Info("user created", "user_id", created.ID, "email", created.Email)
	return created, nil
}

// CreateUserWithPassword creates a user with a specific password.
func (s *UserService) CreateUserWithPassword(ctx context.Context, email, name, phone, password string, roleID int64, status domain.UserStatus) (domain.User, error) {
	if email == "" {
		return domain.User{}, domain.ErrUserEmailRequired
	}
	if phone == "" {
		return domain.User{}, domain.ErrUserPhoneRequired
	}

	if _, err := s.store.GetUserByEmail(ctx, email); err == nil {
		return domain.User{}, domain.ErrUserEmailExists
	}

	if _, err := s.store.GetRoleByID(ctx, roleID); err != nil {
		return domain.User{}, fmt.Errorf("invalid role")
	}

	hashed, err := auth.HashPassword(password)
	if err != nil {
		return domain.User{}, err
	}

	user := domain.User{
		ID:           domain.UserID(generateID()),
		Email:        email,
		PasswordHash: hashed,
		Name:         sanitizeName(name),
		Role:         domain.Role{ID: roleID},
		Status:       status,
	}

	if phone != "" {
		user.Phone = &phone
	}

	created, err := s.store.CreateUser(ctx, user)
	if err != nil {
		return domain.User{}, err
	}

	s.log.Info("user created", "user_id", created.ID, "email", created.Email)
	return created, nil
}

// GetUser retrieves a user by ID.
func (s *UserService) GetUser(ctx context.Context, id domain.UserID) (domain.User, error) {
	return s.store.GetUserByID(ctx, id)
}

// ListUsers retrieves users with search and pagination.
func (s *UserService) ListUsers(ctx context.Context, query, status string, limit, offset int) ([]repository.UserWithRole, int64, error) {
	users, err := s.store.SearchUsers(ctx, query, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.store.CountUsers(ctx, query, status)
	if err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// UpdateUser updates an existing user.
func (s *UserService) UpdateUser(ctx context.Context, id domain.UserID, email, name, phone string, roleID int64, status domain.UserStatus) (domain.User, error) {
	user, err := s.store.GetUserByID(ctx, id)
	if err != nil {
		return domain.User{}, domain.ErrUserNotFound
	}

	existing, _ := s.store.GetUserByEmail(ctx, email)
	if existing.ID != user.ID && existing.Email == email {
		return domain.User{}, domain.ErrUserEmailExists
	}

	user.Email = email
	user.Name = sanitizeName(name)
	user.Role.ID = roleID
	user.Status = status
	if phone != "" {
		user.Phone = &phone
	} else {
		user.Phone = nil
	}

	updated, err := s.store.UpdateUser(ctx, user)
	if err != nil {
		return domain.User{}, err
	}

	s.log.Info("user updated", "user_id", id)
	return updated, nil
}

// DeleteUser deletes a user by ID.
func (s *UserService) DeleteUser(ctx context.Context, id domain.UserID) error {
	if err := s.store.DeleteUser(ctx, id); err != nil {
		return err
	}
	s.log.Info("user deleted", "user_id", id)
	return nil
}

// ListRoles returns all roles.
func (s *UserService) ListRoles(ctx context.Context) ([]domain.Role, error) {
	return s.store.ListRoles(ctx)
}

// ResetPassword resets a user's password to a default value.
func (s *UserService) ResetPassword(ctx context.Context, id domain.UserID) error {
	hashed, err := auth.HashPassword("changeme")
	if err != nil {
		return err
	}
	_, err = s.store.UpdateUserPassword(ctx, id, hashed)
	return err
}
