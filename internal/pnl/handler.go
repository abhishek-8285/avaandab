package pnl

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/auth"
	"transport-app/internal/middleware"
)

func RegisterRoutes(r chi.Router, service *Service, authSrv auth.AuthorizationService) {
	r.With(middleware.RequirePermission(authSrv, "trips", "read")).Get("/api/v1/trips/{id}/pnl", func(w http.ResponseWriter, req *http.Request) {
		result, err := service.Calculate(req.Context(), chi.URLParam(req, "id"))
		if err != nil {
			http.Error(w, `{"error":"trip P&L unavailable"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	})
}
