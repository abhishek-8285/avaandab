package optimizer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OSRMClient calls OSRM route/table for distance matrix, then greedy assigns.
// Pluggable URL: public demo http://router.project-osrm.org or self-host http://osrm.internal:5000
type OSRMClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func (o *OSRMClient) Name() string {
	if strings.Contains(o.BaseURL, "project-osrm.org") {
		return "osrm-public"
	}
	if o.BaseURL != "" {
		return "osrm-selfhost"
	}
	return "osrm"
}

func (o *OSRMClient) client() *http.Client {
	if o.HTTPClient != nil {
		return o.HTTPClient
	}
	return &http.Client{Timeout: 10 * time.Second}
}

// Solve uses OSRM table for distance matrix; falls back to haversine on error.
func (o *OSRMClient) Solve(ctx context.Context, in OptimizationInput) (OptimizationOutput, error) {
	if err := in.Validate(); err != nil {
		return OptimizationOutput{}, err
	}
	start := time.Now()

	// If no URL configured, delegate to mock.
	if o.BaseURL == "" {
		m := &MockOptimizer{}
		out, err := m.Solve(ctx, in)
		if err == nil {
			out.ProviderName = o.Name()
		}
		return out, err
	}

	// Build coordinate list: vehicles starts + shipments
	coords := []string{}
	for _, v := range in.Vehicles {
		coords = append(coords, fmt.Sprintf("%f,%f", v.StartLng, v.StartLat))
	}
	for _, s := range in.Shipments {
		coords = append(coords, fmt.Sprintf("%f,%f", s.Longitude, s.Latitude))
	}
	coordStr := strings.Join(coords, ";")
	nVeh := len(in.Vehicles)

	// OSRM table: sources = vehicles, destinations = shipments
	// GET /table/v1/driving/{coords}?sources=0..nVeh-1&destinations=nVeh..end&annotations=distance,duration
	u, _ := url.Parse(strings.TrimRight(o.BaseURL, "/") + "/table/v1/driving/" + coordStr)
	q := u.Query()
	srcIdx := make([]string, nVeh)
	for i := 0; i < nVeh; i++ {
		srcIdx[i] = fmt.Sprintf("%d", i)
	}
	dstIdx := make([]string, len(in.Shipments))
	for i := range in.Shipments {
		dstIdx[i] = fmt.Sprintf("%d", nVeh+i)
	}
	q.Set("sources", strings.Join(srcIdx, ";"))
	q.Set("destinations", strings.Join(dstIdx, ";"))
	q.Set("annotations", "distance,duration")
	u.RawQuery = q.Encode()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	resp, err := o.client().Do(req)
	if err != nil {
		// Fallback to mock on network error
		m := &MockOptimizer{}
		out, _ := m.Solve(ctx, in)
		out.ProviderName = o.Name() + "-fallback"
		return out, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		_ = body
		m := &MockOptimizer{}
		out, _ := m.Solve(ctx, in)
		out.ProviderName = o.Name() + "-fallback"
		return out, nil
	}
	var tbl struct {
		Code      string      `json:"code"`
		Distances [][]float64 `json:"distances"`
		Durations [][]float64 `json:"durations"`
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err := json.Unmarshal(body, &tbl); err != nil || tbl.Code != "Ok" {
		m := &MockOptimizer{}
		out, _ := m.Solve(ctx, in)
		out.ProviderName = o.Name() + "-fallback"
		return out, nil
	}

	// Greedy assignment using OSRM distances
	routes := make([]OptimizedRoute, nVeh)
	for i, v := range in.Vehicles {
		routes[i] = OptimizedRoute{VehicleID: v.ID}
	}
	assigned := make([]bool, len(in.Shipments))
	totalKM := 0.0
	for k := 0; k < len(in.Shipments); k++ {
		bestS, bestV := -1, -1
		bestD := 1e12
		for si := range in.Shipments {
			if assigned[si] {
				continue
			}
			for vi := 0; vi < nVeh; vi++ {
				if vi >= len(tbl.Distances) || si >= len(tbl.Distances[vi]) {
					continue
				}
				dM := tbl.Distances[vi][si] // meters
				if dM <= 0 {
					continue
				}
				dKM := dM / 1000.0
				if dKM < bestD {
					bestD = dKM
					bestS = si
					bestV = vi
				}
			}
		}
		if bestS == -1 {
			break
		}
		assigned[bestS] = true
		durMin := 0.0
		if bestV < len(tbl.Durations) && bestS < len(tbl.Durations[bestV]) {
			durMin = tbl.Durations[bestV][bestS] / 60.0
		}
		if durMin <= 0 {
			durMin = bestD / 40.0 * 60.0
		}
		leg := RouteLeg{
			ShipmentID:  in.Shipments[bestS].ID,
			DistanceKM:  bestD,
			DurationMin: durMin,
			Sequence:    len(routes[bestV].Legs) + 1,
		}
		routes[bestV].Legs = append(routes[bestV].Legs, leg)
		routes[bestV].TotalKM += bestD
		routes[bestV].TotalMin += durMin
		totalKM += bestD
	}

	out := OptimizationOutput{
		Routes:       routes,
		TotalKM:      totalKM,
		TotalCost:    totalKM,
		SolveTimeMS:  time.Since(start).Milliseconds(),
		ProviderName: o.Name(),
		CreatedAt:    time.Now().UTC(),
	}
	return out, nil
}
