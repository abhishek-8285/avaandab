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

	if cookie, err := r.Cookie("flash_error"); err == nil {
		if pd.Extra == nil {
			pd.Extra = map[string]interface{}{}
		}
		pd.Extra["Error"] = cookie.Value
	}

	if cookie, err := r.Cookie("auth_email"); err == nil {
		if pd.Extra == nil {
			pd.Extra = map[string]interface{}{}
		}
		pd.Extra["Email"] = cookie.Value
	}

	h.renderAuthPage(w, "login_form.html", pd)
}

// RegisterPage renders the user onboarding registration page.
func (h *AuthHandlers) RegisterPage(w http.ResponseWriter, r *http.Request) {
	if isDatastarRequest(r) {
		h.renderFragment(w, "register_form.html", nil)
		return
	}
	pd := PageData{Title: "Create Account"}
	if cookie, err := r.Cookie("flash_error"); err == nil {
		if pd.Extra == nil {
			pd.Extra = map[string]interface{}{}
		}
		pd.Extra["Error"] = cookie.Value
	}
	h.renderAuthPage(w, "register_form.html", pd)
}

// Register handles self-onboarding account creation.
func (h *AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	name := r.PostFormValue("name")
	email := r.PostFormValue("email")
	phone := r.PostFormValue("phone")
	password := r.PostFormValue("password")
	confirm := r.PostFormValue("confirm_password")

	if password != confirm {
		h.renderRegisterError(w, r, "Passwords do not match", email, name, phone)
		return
	}

	// New onboarded user is set as Admin (Role ID 1)
	user, err := h.Services.Users.CreateUserWithPassword(r.Context(), email, name, phone, password, 1, domain.UserStatusActive)
	if err != nil {
		h.renderRegisterError(w, r, err.Error(), email, name, phone)
		return
	}

	// Update Casbin policy so user gets admin permissions immediately
	_ = h.AuthSrv.AddRoleForUser(user.ID.String(), "admin")

	// Automatically log in the user upon onboarding
	h.AuthStore.CreateSession(w, user.ID.String(), "admin", user.Name)

	targetURL := "/dashboard"
	// Check if company settings are configured, if not redirect to company onboarding
	if company, err := h.Services.Settings.GetSettings(r.Context()); err == nil && company.CompanyName == "" {
		targetURL = "/company/onboard"
	}

	if isDatastarRequest(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<script>window.location.href='" + targetURL + "'</script>"))
		return
	}

	http.Redirect(w, r, targetURL, http.StatusSeeOther)
}

func (h *AuthHandlers) renderRegisterError(w http.ResponseWriter, r *http.Request, errMsg, email, name, phone string) {
	if isDatastarRequest(r) {
		h.renderFragment(w, "register_form.html", map[string]interface{}{
			"Title": "Create Account",
			"Error": errMsg,
			"Email": email,
			"Name":  name,
			"Phone": phone,
		})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "flash_error",
		Value:    errMsg,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   30,
	})
	http.Redirect(w, r, "/register", http.StatusSeeOther)
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
		_, _ = w.Write([]byte("<script>window.location.href='/dashboard'</script>"))
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
