package route

import (
	"strings"
	"time"

	"transport-app/internal/domain/types"
)

// Route represents a bidirectional transport route between a source and destination.
type Route struct {
	ID                  types.RouteID
	TenantID            string
	Source              string
	Destination         string
	SourceNormalized    string
	DestNormalized      string
	Distance            float64
	EstimatedHours      float64
	StandardFare        float64
	ReverseDistance     *float64
	ReverseStandardFare *float64
	Direction           string // "oneway" | "bidirectional"
	IsActive            bool
	Remarks             *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// GetDistanceAndFare returns distance and fare considering bidirectional route lookup.
func (r Route) GetDistanceAndFare(from, to string) (distance float64, fare float64, isReverse bool) {
	if strings.EqualFold(r.Source, from) && strings.EqualFold(r.Destination, to) {
		return r.Distance, r.StandardFare, false
	}
	if strings.EqualFold(r.Source, to) && strings.EqualFold(r.Destination, from) {
		dist := r.Distance
		if r.ReverseDistance != nil {
			dist = *r.ReverseDistance
		}
		f := r.StandardFare
		if r.ReverseStandardFare != nil {
			f = *r.ReverseStandardFare
		}
		return dist, f, true
	}
	return r.Distance, r.StandardFare, false
}
