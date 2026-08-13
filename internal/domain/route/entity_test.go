package route_test

import (
	"testing"
	"time"

	"transport-app/internal/domain/route"
	"transport-app/internal/domain/types"
)

func TestRoute_BidirectionalLookup(t *testing.T) {
	now := time.Now()
	revDist := 155.0
	revFare := 4800.0

	r := route.Route{
		ID:                  types.RouteID("rt-1"),
		Source:              "Mumbai",
		Destination:         "Pune",
		Distance:            150.0,
		EstimatedHours:      3.5,
		StandardFare:        4500.0,
		ReverseDistance:     &revDist,
		ReverseStandardFare: &revFare,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	// Forward lookup Mumbai -> Pune
	dist, fare, isRev := r.GetDistanceAndFare("Mumbai", "Pune")
	if dist != 150.0 || fare != 4500.0 || isRev {
		t.Fatalf("expected forward route metrics dist=150 fare=4500 isRev=false, got dist=%f fare=%f isRev=%v", dist, fare, isRev)
	}

	// Reverse lookup Pune -> Mumbai
	distRev, fareRev, isRev2 := r.GetDistanceAndFare("Pune", "Mumbai")
	if distRev != 155.0 || fareRev != 4800.0 || !isRev2 {
		t.Fatalf("expected reverse route metrics dist=155 fare=4800 isRev=true, got dist=%f fare=%f isRev=%v", distRev, fareRev, isRev2)
	}

	// Unknown route fallback
	dUnk, fUnk, _ := r.GetDistanceAndFare("Delhi", "Agra")
	if dUnk != 150.0 || fUnk != 4500.0 {
		t.Fatalf("expected fallback to default distance/fare")
	}
}
