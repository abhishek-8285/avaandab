package handlers

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"transport-app/internal/middleware"
)

// SettingsHandlers handles company settings management.
type SettingsHandlers struct {
	*App
}

func (h *SettingsHandlers) Routes(r chi.Router) {
	r.With(middleware.ResourcePermission(h.AuthSrv, "settings", "read")).Get("/", h.Index)
	r.With(middleware.ResourcePermission(h.AuthSrv, "settings", "update")).Post("/update", h.Update)
	r.Get("/onboard", h.OnboardPage)
	r.Post("/onboard", h.SaveOnboard)
}

func (h *SettingsHandlers) OnboardPage(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)
	settings, err := h.Services.Settings.GetSettings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.renderPage(w, "company_onboard.html", PageData{
		Title: "Company Onboarding",
		User:  session,
		Extra: map[string]interface{}{"Settings": settings},
	})
}

func (h *SettingsHandlers) SaveOnboard(w http.ResponseWriter, r *http.Request) {
	h.Update(w, r)
}

func (h *SettingsHandlers) Index(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)

	settings, err := h.Services.Settings.GetSettings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.renderPage(w, "settings.html", PageData{
		Title: "Settings",
		User:  session,
		Extra: map[string]interface{}{"Settings": settings},
	})
}

func (h *SettingsHandlers) Update(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Preserve existing logo if no new file is uploaded
	var logoPath *string
	current, err := h.Services.Settings.GetSettings(r.Context())
	if err == nil && current.LogoPath != nil {
		logoPath = current.LogoPath
	}

	// Handle logo upload
	if file, header, err := r.FormFile("logo"); err == nil {
		defer file.Close()

		uploaded, upErr := saveLogo(file, header, h.Config.UploadDir)
		if upErr != nil {
			http.Error(w, upErr.Error(), http.StatusBadRequest)
			return
		}
		logoPath = &uploaded
	}

	gstEnabled := r.PostFormValue("gst_enabled") == "1"
	gstRate, _ := parseDecimal(r.PostFormValue("gst_rate"))

	_, err = h.Services.Settings.UpdateSettings(
		r.Context(),
		r.PostFormValue("company_name"),
		r.PostFormValue("currency"),
		r.PostFormValue("timezone"),
		gstEnabled,
		gstRate,
		r.PostFormValue("booking_prefix"),
		r.PostFormValue("trip_prefix"),
		r.PostFormValue("invoice_prefix"),
		r.PostFormValue("financial_year"),
		r.PostFormValue("address"),
		r.PostFormValue("phone"),
		r.PostFormValue("email"),
		r.PostFormValue("gst_number"),
		logoPath,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

// saveLogo writes an uploaded logo file to the uploads directory and returns
// the relative path (from UploadDir) that should be stored in logo_path.
func saveLogo(file io.Reader, header *multipart.FileHeader, uploadDir string) (string, error) {
	subdir := filepath.Join(uploadDir, "company")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		return "", err
	}

	ext := filepath.Ext(header.Filename)
	filename := uuid.NewString() + ext
	dest := filepath.Join(subdir, filename)

	out, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		return "", err
	}

	return filepath.Join("company", filename), nil
}

func parseDecimal(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
