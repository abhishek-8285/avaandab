package graphqlservice

import (
	"encoding/json"
	"net/http"
	"time"

	"transport-app/internal/shared"
	"transport-app/internal/trip/application"
)

type GraphQLHandler struct {
	listTripsUC *application.ListTripsUseCase
}

func NewGraphQLHandler(listTripsUC *application.ListTripsUseCase) *GraphQLHandler {
	return &GraphQLHandler{listTripsUC: listTripsUC}
}

type GraphQLQuery struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func (h *GraphQLHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var q GraphQLQuery
	_ = json.NewDecoder(r.Body).Decode(&q)

	tenantID := shared.TenantIDFromContext(r.Context())
	res, err := h.listTripsUC.Execute(r.Context(), application.ListTripsQuery{
		TenantID: tenantID,
		Page:     1,
		Limit:    50,
		Status:   "",
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"errors": []string{err.Error()}})
		return
	}

	activeTrips := make([]map[string]interface{}, 0, len(res.Trips))
	for _, t := range res.Trips {
		if t.Status == "completed" || t.Status == "cancelled" {
			continue
		}
		activeTrips = append(activeTrips, map[string]interface{}{
			"id":             t.ID,
			"trip_number":    t.TripNumber,
			"driver_name":    t.DriverFirstName + " " + t.DriverLastName,
			"origin":         t.RouteSource,
			"destination":    t.RouteDestination,
			"status":         t.Status,
			"departure_time": t.DepartureTime.Format(time.RFC3339),
			"vehicle_number": t.VehicleRegistrationNumber,
		})
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{
			"activeTrips": activeTrips,
			"serverTime":  time.Now().Format(time.RFC3339),
		},
	})
}
