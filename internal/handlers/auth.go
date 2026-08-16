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

	pd := PageData{
		Title: "Login",
		Extra: map[string]interface{}{},
	}

	if cookie, err := r.Cookie("flash_error"); err == nil {
		pd.Extra["Error"] = cookie.Value
	}

	if cookie, err := r.Cookie("auth_email"); err == nil {
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

	// Self-onboarded users start with the least-privilege viewer role.
	// Privileged roles (admin, dispatcher, accountant) are assigned only
	// by an admin through the authenticated user management interface.
	roleID := domain.DefaultRoleID(domain.RoleViewer)
	user, err := h.Services.Users.CreateUserWithPassword(r.Context(), email, name, phone, password, roleID, domain.UserStatusActive)
	if err != nil {
		h.renderRegisterError(w, r, err.Error(), email, name, phone)
		return
	}

	// Update Casbin policy so user gets viewer permissions immediately
	_ = h.AuthSrv.AddRoleForUser(user.ID.String(), string(domain.RoleViewer))

	// Automatically log in the user upon onboarding with server-side session
	if sessResult, err := h.Services.Auth.CreateSessionForUser(r.Context(), user.ID); err == nil && sessResult != nil {
		h.AuthStore.CreateSessionWithToken(w, user.ID.String(), string(domain.RoleViewer), user.Name, sessResult.SessionToken)
	} else {
		h.AuthStore.CreateSession(w, user.ID.String(), string(domain.RoleViewer), user.Name)
	}

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

	h.AuthStore.CreateSessionWithToken(w, result.User.ID.String(), string(result.User.Role.Name), result.User.Name, result.SessionToken)

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

	targetURL := "/dashboard"
	// Check if user has incomplete setup profile (or company onboarding needed)
	if result.User.Phone == nil || *result.User.Phone == "" {
		targetURL = "/user/onboard"
	} else if company, err := h.Services.Settings.GetSettings(r.Context()); err == nil && company.CompanyName == "" {
		targetURL = "/company/onboard"
	}

	if isDatastarRequest(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<script>window.location.href='" + targetURL + "'</script>"))
		return
	}

	http.Redirect(w, r, targetURL, http.StatusSeeOther)
}

// Logout handles user logout with server-side revocation.
func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	h.AuthStore.RevokeSession(r, w)
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

	pd := PageData{
		Title:      "My Profile",
		User:       session,
		UserDetail: user,
		Roles:      roles,
	}

	// Read and clear flash cookies
	if c, err := r.Cookie("flash_success"); err == nil && c.Value != "" {
		pd.FlashSuccess = c.Value
		http.SetCookie(w, &http.Cookie{Name: "flash_success", Value: "", Path: "/", MaxAge: -1})
	}
	if c, err := r.Cookie("flash_error"); err == nil && c.Value != "" {
		pd.FlashError = c.Value
		http.SetCookie(w, &http.Cookie{Name: "flash_error", Value: "", Path: "/", MaxAge: -1})
	}

	h.renderPage(w, r, "profile_page.html", pd)
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

// ForgotPasswordPage renders the forgot password request page.
func (h *AuthHandlers) ForgotPasswordPage(w http.ResponseWriter, r *http.Request) {
	h.renderAuthPage(w, "forgot_password.html", PageData{
		Title: "Forgot Password",
	})
}

// SubmitForgotPassword processes password reset requests.
func (h *AuthHandlers) SubmitForgotPassword(w http.ResponseWriter, r *http.Request) {
	email := r.PostFormValue("email")
	pd := PageData{
		Title: "Forgot Password",
	}
	if pd.Extra == nil {
		pd.Extra = map[string]interface{}{}
	}
	if email == "" {
		pd.Extra["Error"] = "Please enter your email address"
	} else {
		pd.Extra["SuccessMsg"] = "If an account exists for " + email + ", password reset instructions have been sent."
	}
	h.renderAuthPage(w, "forgot_password.html", pd)
}

// UserOnboardingPage renders the post-login setup page when user has not completed details.
func (h *AuthHandlers) UserOnboardingPage(w http.ResponseWriter, r *http.Request) {
	session, ok := h.getUserFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, _ := h.Services.Auth.GetProfile(r.Context(), domain.UserID(session.UserID))

	pd := PageData{
		Title:      "Account Setup",
		User:       session,
		UserDetail: user,
	}

	h.renderPage(w, r, "user_onboarding.html", pd)
}

// RegisterRoutes registers auth routes.
func (h *AuthHandlers) RegisterRoutes(r chi.Router) {
	r.Get("/login", h.LoginPage)
	r.Post("/login", h.Login)
	r.Get("/register", h.RegisterPage)
	r.Post("/register", h.Register)
	r.Get("/forgot-password", h.ForgotPasswordPage)
	r.Post("/forgot-password", h.SubmitForgotPassword)
	r.Post("/logout", h.Logout)
	r.Get("/profile", h.ProfilePage)
	r.Post("/profile", h.UpdateProfile)
	r.Get("/change-password", h.ChangePasswordPage)
	r.Post("/change-password", h.ChangePassword)
	r.Get("/user/onboard", h.UserOnboardingPage)
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
	timezone := r.PostFormValue("timezone")

	updated, err := h.Services.Auth.UpdateProfile(r.Context(), userID, name, phone, timezone)
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

	http.SetCookie(w, &http.Cookie{
		Name:     "flash_success",
		Value:    "Profile updated successfully",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   10,
	})
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}
