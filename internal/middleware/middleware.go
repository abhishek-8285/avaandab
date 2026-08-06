package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"

	"transport-app/internal/auth"
	"transport-app/internal/shared"
)

// RequestID adds a unique request ID to the context and response header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.NewString()
		}

		ctx := context.WithValue(r.Context(), auth.ContextReqID, reqID)
		w.Header().Set("X-Request-ID", reqID)

		slog.SetDefault(slog.New(slog.NewJSONHandler(nil, nil)))
		slog.LogAttrs(ctx, slog.LevelInfo, "request started",
			slog.String("request_id", reqID),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Logger logs request details and duration.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(ww, r)

		reqID, _ := r.Context().Value(auth.ContextReqID).(string)
		slog.LogAttrs(r.Context(), slog.LevelInfo, "request completed",
			slog.String("request_id", reqID),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", ww.statusCode),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

// Recoverer recovers from panics and returns a 500 error.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := debug.Stack()
				slog.Error("panic recovered",
					slog.Any("error", rec),
					slog.String("stack", string(stack)),
				)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Timeout wraps the handler with a request timeout.
func Timeout(duration time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), duration)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// RequireAuth is middleware that redirects unauthenticated users to login.
func RequireAuth(store *auth.SessionStore) func(http.Handler) http.Handler {
	return AuthRequired(store, "/login")
}

// AuthRequired is a simple middleware that redirects unauthenticated users to login.
func AuthRequired(store *auth.SessionStore, loginPath string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			data, ok := store.ValidateSession(r)
			if !ok {
				http.Redirect(w, r, loginPath, http.StatusSeeOther)
				return
			}

			ctx := context.WithValue(r.Context(), auth.ContextUser, data)
			ctx = context.WithValue(ctx, auth.ContextIP, auth.ClientIP(r))
			ctx = shared.ContextWithTenantID(ctx, "1")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RoleRequired checks that the authenticated user has one of the specified roles.
func RoleRequired(allowedRoles ...int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, ok := r.Context().Value(auth.ContextUser).(*auth.SessionData)
			if !ok || session == nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			roleID := roleIDFromName(session.Role)
			allowed := false
			for _, r := range allowedRoles {
				if roleID == r {
					allowed = true
					break
				}
			}

			if !allowed {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func roleIDFromName(role string) int64 {
	switch role {
	case "admin":
		return 1
	case "dispatcher":
		return 2
	case "accountant":
		return 3
	case "viewer":
		return 4
	default:
		return 4
	}
}

// ResourcePermission checks if the user has permission to access a resource with an action.
func ResourcePermission(authSrv auth.AuthorizationService, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, ok := r.Context().Value(auth.ContextUser).(*auth.SessionData)
			if !ok || session == nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			if !authSrv.Can(session.UserID, resource, action) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// NoCache sets headers to prevent caching of dynamic pages.
func NoCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		next.ServeHTTP(w, r)
	})
}

// SpaResponseWriter wraps http.ResponseWriter to track if it's an SPA request.
type SpaResponseWriter struct {
	http.ResponseWriter
	isSPA bool
}

func (s *SpaResponseWriter) IsSPARequest() bool {
	return s.isSPA
}

// SPAMiddleware checks for X-SPA-Request header and wraps the response writer.
func SPAMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isSPA := r.Header.Get("X-SPA-Request") == "true"
		wrapped := &SpaResponseWriter{
			ResponseWriter: w,
			isSPA:          isSPA,
		}
		next.ServeHTTP(wrapped, r)
	})
}

