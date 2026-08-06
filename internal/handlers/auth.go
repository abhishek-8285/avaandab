package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/domain"
	"transport-app/internal/service"
)

// AuthHandlers handles authentication-related HTTP requests.
type AuthHandlers struct {
	*App
}

// NewAuthHandlers creates auth handlers.
func NewAuthHandlers(app *App) *AuthHandlers {
	return &AuthHandlers{App: app}
}

// LoginPage renders the login page.
func (h *AuthHandlers) LoginPage(w http.ResponseWriter, r *http.Request) {
	if isDatastarRequest(r) {
		h.renderFragment(w, "login_form.html", nil)
		return
	}

	pd := PageData{Title: "Login"}

	// Read flash error cookie and pass to template (do not clear —
	// allows error to persist through browser back/forward navigation)
	if cookie, err := r.Cookie("flash_error"); err == nil {
		if pd.Extra == nil {
			pd.Extra = map[string]interface{}{}
		}
		pd.Extra["Error"] = cookie.Value
	}

	// Preserve email input on error
	if cookie, err := r.Cookie("auth_email"); err == nil {
		if pd.Extra == nil {
			pd.Extra = map[string]interface{}{}
		}
		pd.Extra["Email"] = cookie.Value
	}

	h.renderAuthPage(w, "login_form.html", pd)
}

// Login processes the login form submission.
func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	email := r.PostFormValue("email")
	password := r.PostFormValue("password")

	result, err := h.Services.Auth.Login(r.Context(), service.LoginRequest{
		Email:    email,
		Password: password,
	})

	if err != nil {
		if isDatastarRequest(r) {
			h.renderFragment(w, "login_form.html", map[string]interface{}{
				"Title": "Login",
				"Error": err.Error(),
			})
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "flash_error",
			Value:    err.Error(),
			Path:     "/",
			HttpOnly: true,
			MaxAge:   30,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "auth_email",
			Value:    email,
			Path:     "/",
			HttpOnly: true,
			MaxAge:   30,
		})
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	h.AuthStore.CreateSession(w, result.User.ID.String(), string(result.User.Role.Name), result.User.Name)

	// Clear flash cookies so old errors don't show after successful login
	http.SetCookie(w, &http.Cookie{
		Name:     "flash_error",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_email",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	if isDatastarRequest(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("<script>window.location.href='/dashboard'</script>"))
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// Logout handles user logout.
func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	h.AuthStore.ClearSession(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// ProfilePage renders the user profile page.
func (h *AuthHandlers) ProfilePage(w http.ResponseWriter, r *http.Request) {
	session, ok := h.getUserFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, err := h.Services.Auth.GetProfile(r.Context(), domain.UserID(session.UserID))
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	roles, _ := h.Services.Users.ListRoles(r.Context())

	h.renderPage(w, "profile_page.html", PageData{
		Title:      "My Profile",
		User:       session,
		UserDetail: user,
		Roles:      roles,
	})
}

// ChangePasswordPage renders the change password page.
func (h *AuthHandlers) ChangePasswordPage(w http.ResponseWriter, r *http.Request) {
	h.renderAuthPage(w, "change_password.html", PageData{
		Title: "Change Password",
	})
}

// ChangePassword processes password change.
func (h *AuthHandlers) ChangePassword(w http.ResponseWriter, r *http.Request) {
	session, ok := h.getUserFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	userID := domain.UserID(session.UserID)
	oldPassword := r.PostFormValue("old_password")
	newPassword := r.PostFormValue("new_password")
	confirmPassword := r.PostFormValue("confirm_password")

	if newPassword != confirmPassword {
		http.Error(w, "Passwords do not match", http.StatusBadRequest)
		return
	}

	if err := h.Services.Auth.ChangePassword(r.Context(), userID, oldPassword, newPassword); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "flash_success",
		Value:    "Password changed successfully",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   5,
	})
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

// RegisterRoutes registers auth routes.
func (h *AuthHandlers) RegisterRoutes(r chi.Router) {
	r.Get("/login", h.LoginPage)
	r.Post("/login", h.Login)
	r.Post("/logout", h.Logout)
	r.Get("/profile", h.ProfilePage)
	r.Post("/profile", h.UpdateProfile)
	r.Get("/change-password", h.ChangePasswordPage)
	r.Post("/change-password", h.ChangePassword)
}

// UpdateProfile handles profile updates.
func (h *AuthHandlers) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	session, ok := h.getUserFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	userID := domain.UserID(session.UserID)
	name := r.PostFormValue("name")
	phone := r.PostFormValue("phone")

	updated, err := h.Services.Auth.UpdateProfile(r.Context(), userID, name, phone)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if isDatastarRequest(r) {
		h.renderFragment(w, "profile_page.html", PageData{
			Title:      "My Profile",
			User:       session,
			UserDetail: updated,
		})
		return
	}

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}
