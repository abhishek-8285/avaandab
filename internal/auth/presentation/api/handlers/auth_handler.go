package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/auth"
	"transport-app/internal/domain"
	"transport-app/internal/service"
)

// APIAuthHandler handles REST authentication endpoints.
type APIAuthHandler struct {
	authSvc *service.AuthService
	userSvc *service.UserService
	secret  []byte
}

// NewAPIAuthHandler constructs an APIAuthHandler.
func NewAPIAuthHandler(authSvc *service.AuthService, userSvc *service.UserService, secret []byte) *APIAuthHandler {
	return &APIAuthHandler{authSvc: authSvc, userSvc: userSvc, secret: secret}
}

// Register mounts the auth endpoints onto a chi.Router.
func (h *APIAuthHandler) Register(r chi.Router) {
	r.Post("/api/v1/auth/token", h.IssueToken)
	r.Post("/api/v1/auth/register", h.RegisterUser)
}

// RegisterUser handles universal REST user registration across all client roles (Driver, Dispatcher, Admin).
func (h *APIAuthHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name          string `json:"name"`
		Email         string `json:"email"`
		Phone         string `json:"phone"`
		Password      string `json:"password"`
		Role          string `json:"role"`           // "driver", "dispatcher", "admin"
		VehicleNumber string `json:"vehicle_number"` // Optional metadata
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" || req.Name == "" {
		apiError(w, http.StatusBadRequest, "name, email, and password are required")
		return
	}

	// Map requested role to database Role ID (Default: Driver ID 5)
	var roleID int64 = 5
	roleName := "driver"

	switch req.Role {
	case "admin", "ADMIN":
		roleID = 1
		roleName = "admin"
	case "dispatcher", "DISPATCHER":
		roleID = 2
		roleName = "dispatcher"
	default:
		roleID = 5
		roleName = "driver"
	}

	// Register user in database
	user, err := h.userSvc.CreateUserWithPassword(r.Context(), req.Email, req.Name, req.Phone, req.Password, roleID, domain.UserStatusActive)
	if err != nil {
		apiError(w, http.StatusBadRequest, err.Error())
		return
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	token, err := auth.IssueAPIToken(h.secret, auth.APITokenClaims{
		UserID:    string(user.ID),
		Role:      roleName,
		TenantID:  "1",
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: expiresAt.Unix(),
	})
	if err != nil {
		apiError(w, http.StatusInternalServerError, "token generation failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"token":      token,
		"expires_at": expiresAt.UTC().Format(time.RFC3339),
		"user": map[string]string{
			"id":    string(user.ID),
			"name":  user.Name,
			"email": user.Email,
			"role":  roleName,
		},
	})
}

// IssueToken godoc
//
//	POST /api/v1/auth/token
//	Body: {"email":"user@example.com","password":"secret"}
//	Returns: {"token":"<signed-token>","expires_at":"<RFC3339>"}
func (h *APIAuthHandler) IssueToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		apiError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	result, err := h.authSvc.Login(r.Context(), service.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		// Don't distinguish "not found" from "wrong password" to prevent enumeration.
		apiError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	token, err := auth.IssueAPIToken(h.secret, auth.APITokenClaims{
		UserID:    string(result.User.ID),
		Role:      string(result.User.Role.Name),
		TenantID:  "1", // Single-tenant; extend when multi-tenancy is added.
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: expiresAt.Unix(),
	})
	if err != nil {
		apiError(w, http.StatusInternalServerError, "token generation failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"token":      token,
		"expires_at": expiresAt.UTC().Format(time.RFC3339),
		"user_id":    string(result.User.ID),
		"role":       string(result.User.Role.Name),
	})
}

// apiError writes a consistent JSON error response.
func apiError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
