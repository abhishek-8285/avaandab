package user

import (
	"time"

	"transport-app/internal/domain/types"
)

// UserCreated is emitted when a new user is created.
type UserCreated struct {
	UserID     types.UserID
	Email      string
	Name       string
	RoleID     int64
	OccurredAt time.Time
}

// UserUpdated is emitted when a user is updated.
type UserUpdated struct {
	UserID     types.UserID
	OccurredAt time.Time
}

// UserDeleted is emitted when a user is deleted.
type UserDeleted struct {
	UserID     types.UserID
	OccurredAt time.Time
}

// UserLoggedIn is emitted when a user logs in.
type UserLoggedIn struct {
	UserID     types.UserID
	OccurredAt time.Time
}

// UserLoggedOut is emitted when a user logs out.
type UserLoggedOut struct {
	UserID     types.UserID
	OccurredAt time.Time
}

// PasswordChanged is emitted when a user changes their password.
type PasswordChanged struct {
	UserID     types.UserID
	OccurredAt time.Time
}
