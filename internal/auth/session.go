package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/securecookie"

	"transport-app/internal/domain"
)

// ContextKey is a typed key for context values.
type ContextKey string

const (
	ContextUser     ContextKey = "user"
	ContextRole     ContextKey = "role"
	ContextReqID    ContextKey = "request_id"
	ContextIP       ContextKey = "ip_address"
	ContextLocation ContextKey = "location"
)

// SessionStore manages secure cookie-based sessions.
type SessionStore struct {
	cookieName string
	signer     *securecookie.SecureCookie
}

// NewSessionStore creates a new session store with the given secret.
func NewSessionStore(cookieSecret string) *SessionStore {
	return &SessionStore{
		cookieName: "session",
		signer:     securecookie.New([]byte(cookieSecret), nil),
	}
}

// SessionData holds the data stored in a session cookie.
type SessionData struct {
	UserID  string `json:"user_id"`
	Role    string `json:"role"`
	Name    string `json:"name"`
	Expires int64  `json:"expires"`
}

// CreateSession creates and signs a session cookie for a user.
func (s *SessionStore) CreateSession(w http.ResponseWriter, userID, roleName, name string) {
	data := &SessionData{
		UserID:  userID,
		Role:    roleName,
		Name:    name,
		Expires: time.Now().Add(24 * time.Hour).Unix(),
	}

	http.SetCookie(w, &http.Cookie{
		Name:     s.cookieName,
		Value:    s.mustEncode(data),
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})
}

func (s *SessionStore) mustEncode(data *SessionData) string {
	encoded, err := s.signer.Encode(s.cookieName, data)
	if err != nil {
		return ""
	}
	return encoded
}

// ValidateSession validates and decodes the session cookie.
func (s *SessionStore) ValidateSession(r *http.Request) (*SessionData, bool) {
	cookie, err := r.Cookie(s.cookieName)
	if err != nil {
		return nil, false
	}

	var data SessionData
	if err := s.signer.Decode(s.cookieName, cookie.Value, &data); err != nil {
		return nil, false
	}

	if time.Now().Unix() > data.Expires {
		return nil, false
	}

	return &data, true
}

// ClearSession removes the session cookie.
func (s *SessionStore) ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// GenerateSecureToken generates a cryptographically secure random token.
func GenerateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ClientIP extracts the client IP from the request, checking Cloudflare headers
// (CF-Connecting-IP, X-Real-IP, X-Forwarded-For) and falling back to RemoteAddr safely.
func ClientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); ip != "" {
		return ip
	}
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx >= 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if ip := strings.TrimSpace(r.RemoteAddr); ip != "" {
		if host, _, err := net.SplitHostPort(ip); err == nil {
			return host
		}
		return ip
	}
	return "Unknown"
}

// ClientLocation safely extracts country and city from Cloudflare proxy headers (CF-IPCountry, CF-IPCity).
// Returns non-empty string or fallback without requiring client-side permissions or blocking behavior.
func ClientLocation(r *http.Request) string {
	country := strings.TrimSpace(r.Header.Get("CF-IPCountry"))
	city := strings.TrimSpace(r.Header.Get("CF-IPCity"))

	if city != "" && country != "" {
		return city + ", " + country
	}
	if country != "" {
		return country
	}
	return "Unknown"
}

// HashToken hashes a session token for storage.
func HashToken(token string) string {
	return hex.EncodeToString([]byte(token))
}

// CompareToken compares a token against its hash.
func CompareToken(token, hash string) bool {
	return token == hash
}

// Role hierarchy: lower number = more privilege
// admin(1) > dispatcher(2) > accountant(3) > viewer(4)

// HasPermission checks if a role can perform a given action on a resource.
func HasPermission(roleID int64, resource string, action string) bool {
	permissions := map[string]map[int64]bool{
		"users":     {1: true, 2: false, 3: false, 4: false},
		"drivers":   {1: true, 2: true, 3: false, 4: false},
		"vehicles":  {1: true, 2: true, 3: false, 4: false},
		"customers": {1: true, 2: true, 3: false, 4: false},
		"routes":    {1: true, 2: true, 3: false, 4: false},
		"bookings":  {1: true, 2: true, 3: false, 4: false},
		"trips":     {1: true, 2: true, 3: false, 4: false},
		"invoices":  {1: true, 2: false, 3: true, 4: false},
		"payments":  {1: true, 2: false, 3: true, 4: false},
		"reports":   {1: true, 2: true, 3: true, 4: false},
	}

	if roleID == 4 {
		readResources := map[string]bool{
			"drivers": true, "vehicles": true, "customers": true, "routes": true,
			"bookings": true, "trips": true, "invoices": true, "payments": true, "reports": true,
		}
		if readResources[resource] && action == "read" {
			return true
		}
	}

	if resourcePerms, ok := permissions[resource]; ok {
		if allowed, ok := resourcePerms[roleID]; ok {
			return allowed
		}
	}

	return false
}

// RoleNameForID maps a role ID to its domain.RoleName.
func RoleNameForID(roleID int64) domain.RoleName {
	switch roleID {
	case 1:
		return domain.RoleAdmin
	case 2:
		return domain.RoleDispatcher
	case 3:
		return domain.RoleAccountant
	case 4:
		return domain.RoleViewer
	default:
		return domain.RoleDispatcher
	}
}

// RoleID for action checks (lower = more privilege)
func RoleIDForName(role domain.RoleName) int64 {
	return domain.DefaultRoleID(role)
}
