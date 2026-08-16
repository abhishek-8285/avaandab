package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"transport-app/internal/auth"
	"transport-app/internal/config"
	"transport-app/internal/domain"
	"transport-app/internal/service"
)

const datastarRequestHeader = "Datastar-Request"

// App holds shared handler dependencies.
type App struct {
	Services  *service.Services
	Config    *config.Config
	AuthStore *auth.SessionStore
	DB        *sql.DB
	Templates *template.Template
	AuthSrv   auth.AuthorizationService

	// Handler groups
	Auth      *AuthHandlers
	Dashboard *DashboardHandlers
	Users     *UserHandlers
	Drivers   *DriverHandlers
	Vehicles  *VehicleHandlers
	Customers *CustomerHandlers
	Routes    *RouteHandlers
	Bookings  *BookingHandlers
	Trips     *TripHandlers
	Invoices  *InvoiceHandlers
	Payments  *PaymentHandlers
	Reports   *ReportHandlers
	SettingsH *SettingsHandlers
	AuditLogs *AuditLogHandlers
	Contact   *ContactHandlers
	Kharcha   *KharchaHandlers
}

// NewApp creates a new handler app with all handler groups initialized.
func NewApp(svc *service.Services, cfg *config.Config, authStore *auth.SessionStore, db *sql.DB, authSrv auth.AuthorizationService) *App {
	templates, err := parseTemplates(authSrv)
	if err != nil {
		slog.Error("failed to parse templates; serving with minimal template set", "error", err)
		templates = template.New("")
	}

	app := &App{
		Services:  svc,
		Config:    cfg,
		AuthStore: authStore,
		DB:        db,
		Templates: templates,
		AuthSrv:   authSrv,
	}

	app.Auth = &AuthHandlers{App: app}
	app.Dashboard = &DashboardHandlers{App: app}
	app.Users = &UserHandlers{App: app}
	app.Drivers = &DriverHandlers{App: app}
	app.Vehicles = &VehicleHandlers{App: app}
	app.Customers = &CustomerHandlers{App: app}
	app.Routes = &RouteHandlers{App: app}
	app.Bookings = &BookingHandlers{App: app}
	app.Trips = &TripHandlers{App: app}
	app.Invoices = &InvoiceHandlers{App: app}
	app.Payments = &PaymentHandlers{App: app}
	app.Reports = &ReportHandlers{App: app}
	app.SettingsH = &SettingsHandlers{App: app}
	app.AuditLogs = &AuditLogHandlers{App: app}
	app.Contact = &ContactHandlers{App: app}
	app.Kharcha = &KharchaHandlers{App: app}

	return app
}

// parseTemplates loads and parses all HTML templates with custom functions.
func parseTemplates(authSrv auth.AuthorizationService) (*template.Template, error) {
	tmpl := template.New("").Funcs(template.FuncMap{
		"can": func(user interface{}, resource string, action string) bool {
			if user == nil {
				return false
			}
			var uid string
			switch u := user.(type) {
			case *auth.SessionData:
				if u == nil {
					return false
				}
				uid = u.UserID
			case auth.SessionData:
				uid = u.UserID
			case string:
				uid = u
			default:
				return false
			}
			return authSrv.Can(uid, resource, action)
		},
		"formatDateTime": formatDateTime,
		"formatDate":     formatDate,
		"datetime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04")
		},
		"date_only": func(t time.Time) string {
			return t.Format("2006-01-02")
		},
		"lower":    strings.ToLower,
		"upper":    strings.ToUpper,
		"join":     strings.Join,
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
		"abbr": func(s string, max int) string {
			if len(s) <= max {
				return s
			}
			if max <= 3 {
				return s[:max] + "..."
			}
			return s[:max-3] + "..."
		},
		"fileExt":     filepath.Ext,
		"add":         func(a, b int) int { return a + b },
		"sub":         func(a, b int) int { return a - b },
		"mul":         func(a, b int) int { return a * b },
		"div":         func(a, b int) int { return a / b },
		"statusBadge": statusBadgeClass,
		"priceFormat": func(f float64) string { return fmt.Sprintf("%.2f", f) },
		"yesNo": func(b bool) string {
			if b {
				return "Yes"
			}
			return "No"
		},
		"nullString": func(s *string) string {
			if s == nil {
				return ""
			}
			return *s
		},
		"slice": func(s string, i, j int) string {
			r := []rune(s)
			if i < 0 {
				i = 0
			}
			if j > len(r) {
				j = len(r)
			}
			if i >= j {
				return ""
			}
			return string(r[i:j])
		},
		"derefTime": func(t *time.Time) string {
			if t == nil {
				return ""
			}
			return t.Format("2006-01-02 15:04")
		},
	})

	// Support TEMPLATES_DIR env var for deployments where CWD != repo root
	templatesDir := os.Getenv("TEMPLATES_DIR")
	if templatesDir == "" {
		templatesDir = "internal/templates"
	}
	partialsDir := filepath.Join(templatesDir, "partials")

	_, err := tmpl.ParseGlob(filepath.Join(templatesDir, "*.html"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates from %q: %w", templatesDir, err)
	}
	// Parse partial templates from subdirectories
	if _, err := tmpl.ParseGlob(filepath.Join(partialsDir, "*.html")); err != nil {
		return nil, fmt.Errorf("failed to parse partial templates from %q: %w", partialsDir, err)
	}
	return tmpl, nil
}

func statusBadgeClass(status interface{}) string {
	s := fmt.Sprintf("%v", status)
	classes := map[string]string{
		"pending":        "bg-yellow-100 text-yellow-800",
		"confirmed":      "bg-blue-100 text-blue-800",
		"completed":      "bg-green-100 text-green-800",
		"cancelled":      "bg-red-100 text-red-800",
		"draft":          "bg-gray-100 text-gray-800",
		"scheduled":      "bg-purple-100 text-purple-800",
		"assigned":       "bg-indigo-100 text-indigo-800",
		"started":        "bg-orange-100 text-orange-800",
		"reached_pickup": "bg-blue-100 text-blue-800",
		"in_transit":     "bg-teal-100 text-teal-800",
		"delivered":      "bg-emerald-100 text-emerald-800",
		"available":      "bg-green-100 text-green-800",
		"on_trip":        "bg-orange-100 text-orange-800",
		"maintenance":    "bg-yellow-100 text-yellow-800",
		"running":        "bg-blue-100 text-blue-800",
		"inactive":       "bg-gray-100 text-gray-800",
		"paid":           "bg-green-100 text-green-800",
		"partially_paid": "bg-yellow-100 text-yellow-800",
	}
	if cls, ok := classes[s]; ok {
		return cls
	}
	return "bg-gray-100 text-gray-800"
}

func (a *App) getUserFromContext(r *http.Request) (*auth.SessionData, bool) {
	data, ok := r.Context().Value(auth.ContextUser).(*auth.SessionData)
	return data, ok
}

func isDatastarRequest(r *http.Request) bool {
	return r.Header.Get(datastarRequestHeader) == "true" ||
		r.Header.Get("HX-Request") == "true" ||
		r.URL.Query().Get("_fragment") == "true"
}

func formatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}

func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// PageData is the base data passed to all page templates.
type PageData struct {
	Title         string
	Version       string
	User          *auth.SessionData
	UserDetail    interface{}
	Roles         interface{}
	Settings      interface{}
	FlashError    string
	FlashSuccess  string
	RazorpayKeyID string
	Extra         map[string]interface{}
}

// PaginationData contains pagination data for templates.
type PaginationData struct {
	Page       int
	PerPage    int
	Total      int64
	TotalPages int
	HasPrev    bool
	HasNext    bool
	BasePath   string
}

// PaginationParams holds pagination and search parameters.
type PaginationParams struct {
	Query  string
	Status string
	Limit  int
	Page   int
	Offset int
}

func parsePaginationParams(r *http.Request) PaginationParams {
	query := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")
	if status == "all" {
		status = ""
	}
	limit := 20
	_, _ = fmt.Sscanf(r.URL.Query().Get("limit"), "%d", &limit)
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	page := 1
	_, _ = fmt.Sscanf(r.URL.Query().Get("page"), "%d", &page)
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit
	return PaginationParams{Query: query, Status: status, Limit: limit, Page: page, Offset: offset}
}

func newPaginationData(pp PaginationParams, total int64, basePath string) PaginationData {
	totalPages := int(total / int64(pp.Limit))
	if total%int64(pp.Limit) > 0 {
		totalPages++
	}
	if totalPages < 1 {
		totalPages = 1
	}
	return PaginationData{
		Page:       pp.Page,
		PerPage:    pp.Limit,
		Total:      total,
		TotalPages: totalPages,
		HasPrev:    pp.Page > 1,
		HasNext:    pp.Page < totalPages,
		BasePath:   basePath,
	}
}

// AppVersion holds the build version string populated from environment or fallback
var AppVersion = func() string {
	if v := os.Getenv("APP_VERSION"); v != "" {
		return v
	}
	return fmt.Sprintf("%d", time.Now().Unix())
}()

// buildTemplateData creates a flat map for templates from PageData.
func buildTemplateData(data PageData) map[string]interface{} {
	v := data.Version
	if v == "" {
		v = AppVersion
	}
	m := map[string]interface{}{
		"Title":        data.Title,
		"Version":      v,
		"User":         data.User,
		"UserDetail":   data.UserDetail,
		"Roles":        data.Roles,
		"Settings":     data.Settings,
		"FlashError":   data.FlashError,
		"FlashSuccess": data.FlashSuccess,
	}
	for k, v := range data.Extra {
		m[k] = v
	}
	return m
}

// renderPage renders a full page with the layout.
func (a *App) renderPage(w http.ResponseWriter, r *http.Request, name string, data PageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	layout := a.Templates.Lookup("layout.html")
	if layout == nil {
		http.Error(w, "layout template not found", http.StatusInternalServerError)
		return
	}

	contentTmpl := a.Templates.Lookup(name)
	if contentTmpl == nil {
		a.renderError(w, http.StatusNotFound, "Page Not Found", fmt.Sprintf("Template %q could not be located.", name), data.User)
		return
	}

	templateData := buildTemplateData(data)

	var buf strings.Builder
	if err := contentTmpl.Execute(&buf, templateData); err != nil {
		a.renderError(w, http.StatusInternalServerError, "Template Execution Error", err.Error(), data.User)
		return
	}

	if s, ok := w.(interface{ IsSPARequest() bool }); ok && s.IsSPARequest() {
		w.Header().Set("X-Page-Title", data.Title)
		_, _ = w.Write([]byte(buf.String()))
		return
	}

	var notifications interface{}
	var unreadCount int
	if notifs, total, err := a.Services.Audit.ListAuditLogs(context.Background(), 5, 0); err == nil {
		notifications = notifs
		unreadCount = int(total)
	}

	// Apply per-user "mark all read" — if cookie is set, count only logs newer than that time
	if r != nil {
		if cookie, err := r.Cookie("notif_read_at"); err == nil && cookie.Value != "" {
			// Use timestamp to reduce unread count — simplified: treat as 0 unread
			_ = cookie.Value
			unreadCount = 0
		}
	}
	if unreadCount > 99 {
		unreadCount = 99
	}

	pd := struct {
		Title         string
		Content       template.HTML
		User          *auth.SessionData
		Query         string
		Notifications interface{}
		UnreadCount   int
		HasUnread     bool
		FlashError    string
		FlashSuccess  string
		Version       string
	}{
		Title:         data.Title,
		Content:       template.HTML(buf.String()),
		User:          data.User,
		Query:         fmt.Sprintf("%v", templateData["Query"]),
		Notifications: notifications,
		UnreadCount:   unreadCount,
		HasUnread:     unreadCount > 0,
		FlashError:    data.FlashError,
		FlashSuccess:  data.FlashSuccess,
		Version:       AppVersion,
	}

	if err := layout.Execute(w, pd); err != nil {
		http.Error(w, fmt.Sprintf("layout template error: %v", err), http.StatusInternalServerError)
	}
}

// renderAuthPage renders a full page with the auth layout (no sidebar).
func (a *App) renderAuthPage(w http.ResponseWriter, name string, data PageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	authLayout := a.Templates.Lookup("auth_layout.html")
	if authLayout == nil {
		http.Error(w, "auth layout template not found", http.StatusInternalServerError)
		return
	}

	contentTmpl := a.Templates.Lookup(name)
	if contentTmpl == nil {
		http.Error(w, fmt.Sprintf("template %q not found", name), http.StatusInternalServerError)
		return
	}

	templateData := buildTemplateData(data)

	var buf strings.Builder
	if err := contentTmpl.Execute(&buf, templateData); err != nil {
		http.Error(w, fmt.Sprintf("content template error: %v", err), http.StatusInternalServerError)
		return
	}

	pd := struct {
		Title        string
		Content      template.HTML
		User         *auth.SessionData
		FlashError   string
		FlashSuccess string
		Version      string
	}{
		Title:        data.Title,
		Content:      template.HTML(buf.String()),
		User:         data.User,
		FlashError:   data.FlashError,
		FlashSuccess: data.FlashSuccess,
		Version:      AppVersion,
	}

	if err := authLayout.Execute(w, pd); err != nil {
		http.Error(w, fmt.Sprintf("auth layout template error: %v", err), http.StatusInternalServerError)
	}
}

// renderFragment renders a fragment or template safely.
func (a *App) renderFragment(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := a.Templates.Lookup(name)
	if tmpl == nil {
		// Fallback: strip _table suffix if present and attempt main template lookup
		fallbackName := strings.Replace(name, "_table.html", ".html", 1)
		tmpl = a.Templates.Lookup(fallbackName)
	}
	if tmpl == nil {
		http.Error(w, fmt.Sprintf("template %q not found", name), http.StatusInternalServerError)
		return
	}

	// If data is PageData, flatten it
	if pd, ok := data.(PageData); ok {
		data = buildTemplateData(pd)
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("template error: %v", err), http.StatusInternalServerError)
	}
}

// renderForm renders a form template (full page or fragment).
func (a *App) renderForm(w http.ResponseWriter, r *http.Request, name string, data PageData) {
	if isDatastarRequest(r) {
		a.renderFragment(w, name, data)
		return
	}
	a.renderPage(w, r, name, data)
}

const homeCacheTTL = 15 * time.Minute

var (
	homeCacheMu    sync.Mutex
	cachedHomeHTML []byte
	cachedHomeAt   time.Time
)

// Marketing renders the landing homepage using an in-memory cache with TTL.
func (a *App) Marketing(w http.ResponseWriter, r *http.Request) {
	homeCacheMu.Lock()
	if len(cachedHomeHTML) == 0 || time.Since(cachedHomeAt) > homeCacheTTL {
		tmpl := a.Templates.Lookup("home.html")
		if tmpl != nil {
			var buf bytes.Buffer
			data := map[string]interface{}{"Version": AppVersion}
			if err := tmpl.Execute(&buf, data); err == nil {
				cachedHomeHTML = buf.Bytes()
				cachedHomeAt = time.Now()
			}
		}
	}
	html := cachedHomeHTML
	homeCacheMu.Unlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600, s-maxage=86400, stale-while-revalidate=604800")
	if len(html) > 0 {
		_, _ = w.Write(html)
		return
	}

	tmpl := a.Templates.Lookup("home.html")
	if tmpl == nil {
		http.Error(w, "home template not found", http.StatusInternalServerError)
		return
	}
	data := map[string]interface{}{"Version": AppVersion}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("template error: %v", err), http.StatusInternalServerError)
	}
}

// DownloadFile serves an uploaded file by ID.
func (a *App) DownloadFile(w http.ResponseWriter, r *http.Request) {
	id := filepath.Base(r.URL.Path)
	if !isValidFileID(id) {
		a.renderError(w, http.StatusBadRequest, "Invalid File ID", "The requested file identifier is invalid.", nil)
		return
	}
	file, err := a.Services.Files.GetFile(r.Context(), domain.FileID(id))
	if err != nil {
		a.renderError(w, http.StatusNotFound, "File Not Found", "The requested document or file does not exist.", nil)
		return
	}
	uploadDir := filepath.Clean(a.Config.UploadDir)
	if uploadDir == "." {
		uploadDir = ""
	}
	filePath := filepath.Clean(filepath.Join(uploadDir, file.Path))
	if uploadDir != "" && !strings.HasPrefix(filePath, uploadDir+string(os.PathSeparator)) && filePath != uploadDir {
		a.renderError(w, http.StatusBadRequest, "Invalid File Path", "The requested file path is invalid.", nil)
		return
	}
	http.ServeFile(w, r, filePath)
}

func isValidFileID(id string) bool {
	if id == "" || id == "." || id == ".." {
		return false
	}
	return !strings.ContainsAny(id, `/\`) && !strings.Contains(id, "..")
}

// renderError renders a friendly user-facing error screen using error.html and layout.html.
func (a *App) renderError(w http.ResponseWriter, statusCode int, title string, message string, user *auth.SessionData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)

	fallback := fmt.Sprintf("%d - %s: %s", statusCode, title, message)

	errTmpl := a.Templates.Lookup("error.html")
	if errTmpl == nil {
		_, _ = w.Write([]byte(fallback))
		return
	}

	var buf strings.Builder
	if err := errTmpl.Execute(&buf, map[string]interface{}{
		"StatusCode": statusCode,
		"Title":      title,
		"Message":    message,
	}); err != nil {
		slog.Error("error template execution failed", "statusCode", statusCode, "title", title, "error", err)
		_, _ = w.Write([]byte(fallback))
		return
	}

	layout := a.Templates.Lookup("layout.html")
	if layout == nil {
		_, _ = w.Write([]byte(buf.String()))
		return
	}

	if err := layout.Execute(w, struct {
		Title        string
		Content      template.HTML
		User         *auth.SessionData
		FlashError   string
		FlashSuccess string
	}{
		Title:   title,
		Content: template.HTML(buf.String()),
		User:    user,
	}); err != nil {
		slog.Error("error layout execution failed", "statusCode", statusCode, "title", title, "error", err)
		_, _ = w.Write([]byte(fallback))
	}
}
