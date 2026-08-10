package graphqlservice

import (
	"encoding/json"
	"net/http"
	"time"
)

type TripSchema struct {
	ID          string  `json:"id"`
	DriverName  string  `json:"driver_name"`
	Origin      string  `json:"origin"`
	Destination string  `json:"destination"`
	Status      string  `json:"status"`
	CargoWeight float64 `json:"cargo_weight_kg"`
}

type GraphQLQuery struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func GraphQLHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var q GraphQLQuery
	_ = json.NewDecoder(r.Body).Decode(&q)

	// Mock GraphQL Query Response
	response := map[string]interface{}{
		"data": map[string]interface{}{
			"activeTrips": []TripSchema{
				{
					ID:          "TRIP-8842",
					DriverName:  "Vikram Singh",
					Origin:      "Mumbai Port Terminal 2",
					Destination: "Pune Logistics Hub Hub B",
					Status:      "IN_TRANSIT",
					CargoWeight: 4200.5,
				},
				{
					ID:          "TRIP-9921",
					DriverName:  "Rajesh Kumar",
					Origin:      "Bhiwandi Warehouse A",
					Destination: "Nashik Distribution Depot",
					Status:      "DISPATCHED",
					CargoWeight: 1850.0,
				},
			},
			"serverTime": time.Now().Format(time.RFC3339),
		},
	}

	json.NewEncoder(w).Encode(response)
}
