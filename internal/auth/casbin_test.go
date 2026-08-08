package auth

import (
	"database/sql"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func TestCasbinAuthorization(t *testing.T) {
	// Create test DB
	db, err := sql.Open("sqlite", ":memory:?cache=shared&_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Apply migrations
	_ = goose.SetDialect("sqlite")
	if err := goose.Up(db, "../../db/migrations"); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Initialize authorization service
	authSvc, err := NewCasbinAuthorizationService(db)
	if err != nil {
		t.Fatalf("failed to create authorization service: %v", err)
	}

	// Seed users with roles
	_, err = db.Exec(`
		INSERT INTO users (id, email, password_hash, name, role_id, status)
		VALUES 
			('user-admin-1', 'admin@test.com', 'hash', 'Admin', 1, 'active'),
			('user-disp-1', 'disp@test.com', 'hash', 'Dispatcher', 2, 'active'),
			('user-viewer-1', 'viewer@test.com', 'hash', 'Viewer', 4, 'active')
	`)
	if err != nil {
		t.Fatalf("failed to seed users: %v", err)
	}

	// Note: user_roles are populated automatically by SQLite triggers on insert into users

	// Reload policies
	if err := authSvc.Reload(); err != nil {
		t.Fatalf("failed to reload policies: %v", err)
	}

	tests := []struct {
		userID   string
		resource string
		action   string
		allowed  bool
	}{
		// Admin permissions
		{"user-admin-1", "drivers", "create", true},
		{"user-admin-1", "users", "delete", true},
		{"user-admin-1", "settings", "update", true},

		// Dispatcher permissions
		{"user-disp-1", "drivers", "create", true},
		{"user-disp-1", "drivers", "read", true},
		{"user-disp-1", "users", "delete", false},    // Dispatcher cannot delete users
		{"user-disp-1", "payments", "create", false}, // Dispatcher cannot manage payments

		// Viewer permissions
		{"user-viewer-1", "drivers", "read", true},
		{"user-viewer-1", "drivers", "create", false}, // Viewer cannot create
		{"user-viewer-1", "users", "read", false},     // Viewer cannot read users table
	}

	for _, tt := range tests {
		t.Run(tt.userID+"-"+tt.resource+"-"+tt.action, func(t *testing.T) {
			allowed := authSvc.Can(tt.userID, tt.resource, tt.action)
			if allowed != tt.allowed {
				t.Errorf("expected allowed=%t for %s on %s:%s, got %t", tt.allowed, tt.userID, tt.resource, tt.action, allowed)
			}
		})
	}
}
