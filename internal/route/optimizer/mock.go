package optimizer

import (
	"context"
	"math"
	"time"
)

// MockOptimizer is a deterministic greedy nearest-neighbor solver.
// Zero external deps, used for dev/test and as fallback when OSRM fails.
type MockOptimizer struct{}

func (m *MockOptimizer) Name() string { return "mock" }

func (m *MockOptimizer) Solve(ctx context.Context, in OptimizationInput) (OptimizationOutput, error) {
	if err := in.Validate(); err != nil {
		return OptimizationOutput{}, err
	}
	start := time.Now()

	// Round-robin greedy: assign shipments to vehicles by nearest current position.
	routes := make([]OptimizedRoute, len(in.Vehicles))
	for i, v := range in.Vehicles {
		routes[i] = OptimizedRoute{VehicleID: v.ID}
	}
	// Current pos per vehicle
	curLat := make([]float64, len(in.Vehicles))
	curLng := make([]float64, len(in.Vehicles))
	for i, v := range in.Vehicles {
		curLat[i] = v.StartLat
		curLng[i] = v.StartLng
	}

	totalKM := 0.0
	assigned := make([]bool, len(in.Shipments))

	for k := 0; k < len(in.Shipments); k++ {
		// pick next unassigned globally closest to any vehicle head
		bestS, bestV := -1, -1
		bestD := math.MaxFloat64
		for si, s := range in.Shipments {
			if assigned[si] {
				continue
			}
			for vi := range in.Vehicles {
				d := haversine(curLat[vi], curLng[vi], s.Latitude, s.Longitude)
				if d < bestD {
					bestD = d
					bestS = si
					bestV = vi
				}
			}
		}
		if bestS == -1 {
			break
		}
		assigned[bestS] = true
		s := in.Shipments[bestS]
		// Assume 40 km/h avg for duration estimate
		dist := bestD
		dur := dist / 40.0 * 60.0 // minutes
		leg := RouteLeg{
			ShipmentID:  s.ID,
			DistanceKM:  dist,
			DurationMin: dur,
			Sequence:    len(routes[bestV].Legs) + 1,
		}
		routes[bestV].Legs = append(routes[bestV].Legs, leg)
		routes[bestV].TotalKM += dist
		routes[bestV].TotalMin += dur
		totalKM += dist
		curLat[bestV] = s.Latitude
		curLng[bestV] = s.Longitude
		select {
		case <-ctx.Done():
			return OptimizationOutput{}, ctx.Err()
		default:
		}
	}

	out := OptimizationOutput{
		Routes:       routes,
		TotalKM:      totalKM,
		TotalCost:    totalKM, // cost = km for mock
		SolveTimeMS:  time.Since(start).Milliseconds(),
		ProviderName: m.Name(),
		CreatedAt:    time.Now().UTC(),
	}
	return out, nil
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
