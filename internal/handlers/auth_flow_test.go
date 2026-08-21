package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"transport-app/internal/auth"
	"transport-app/internal/config"
	"transport-app/internal/domain"
)

func newAuthTestApp(t *testing.T) *AuthHandlers {
	t.Helper()
	if cwd, _ := os.Getwd(); filepath.Base(cwd) == "handlers" {
		_ = os.Chdir("../..")
	}
	tmpl, err := parseTemplates(&mockAuthSvc{})
	require.NoError(t, err)

	authStore := auth.NewSessionStore("test-secret-key-that-is-at-least-32-chars-long", false)
	cfg := &config.Config{
		AppEnv:       "development",
		CookieSecure: false,
	}

	app := &App{
		Templates:   tmpl,
		AuthStore:   authStore,
		Config:      cfg,
		ResetTokens: auth.NewResetTokenStore(0),
	}
	return NewAuthHandlers(app)
}

func TestLoginPage_FlashSuccessAndError(t *testing.T) {
	h := newAuthTestApp(t)

	// Test with flash_success cookie
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	req.AddCookie(&http.Cookie{Name: "flash_success", Value: "Password reset successful."})
	w := httptest.NewRecorder()

	h.LoginPage(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Password reset successful.")

	// Verify cookie clearing
	cookies := w.Result().Cookies()
	var clearedSuccess bool
	for _, c := range cookies {
		if c.Name == "flash_success" && c.MaxAge == -1 {
			clearedSuccess = true
		}
	}
	assert.True(t, clearedSuccess, "flash_success cookie should be cleared")

	// Test with flash_error cookie
	req2 := httptest.NewRequest(http.MethodGet, "/login", nil)
	req2.AddCookie(&http.Cookie{Name: "flash_error", Value: "Invalid credentials."})
	w2 := httptest.NewRecorder()

	h.LoginPage(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	body2 := w2.Body.String()
	assert.Contains(t, body2, "Invalid credentials.")
}

func TestChangePassword_MismatchRendersHTML(t *testing.T) {
	h := newAuthTestApp(t)

	form := url.Values{}
	form.Set("old_password", "oldpass123")
	form.Set("new_password", "newpass123")
	form.Set("confirm_password", "differentpass")

	req := httptest.NewRequest(http.MethodPost, "/change-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Add authenticated session context
	sess := &auth.SessionData{
		UserID: "user-1",
		Role:   string(domain.RoleAdmin),
		Name:   "Test User",
	}
	ctx := context.WithValue(req.Context(), auth.ContextUser, sess)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.ChangePassword(w, req)

	// Should return 200 OK and render change_password.html with error alert, NOT raw plain text 400
	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Passwords do not match")
	assert.Contains(t, body, "Change Password")
}
