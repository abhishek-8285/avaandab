package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// Geofence kinds (Spec 02 DDL CHECK constraint).
const (
	KindPickup     = "pickup"
	KindDrop       = "drop"
	KindDepot      = "depot"
	KindRestricted = "restricted"
	KindNoEntry    = "no_entry"
)

// Geofence shapes.
const (
	ShapeCircle  = "circle"
	ShapePolygon = "polygon"
)

// Point is a geographic coordinate (degrees).
type Point struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// Fix is a single vehicle telemetry fix consumed by the dwell engine.
type Fix struct {
	VehicleID string
	TripID    *string
	Timestamp time.Time
	Latitude  float64
	Longitude float64
	Speed     float64
}

// Geofence is a named zone definition (Spec 02 §2).
type Geofence struct {
	ID        string
	TenantID  string
	Name      string
	Kind      string // pickup | drop | depot | restricted | no_entry
	Shape     string // circle | polygon
	CenterLat float64
	CenterLng float64
	RadiusM   float64
	Polygon   []Point
	RouteName string
	Priority  int
	IsActive  bool
	CreatedBy *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PolygonFromJSON parses the persisted polygon representation
// (`[[lat,lng],...]`) into a Point slice.
func PolygonFromJSON(raw string) ([]Point, error) {
	var coords [][]float64
	if err := json.Unmarshal([]byte(raw), &coords); err != nil {
		return nil, fmt.Errorf("parse polygon: %w", err)
	}
	pts := make([]Point, 0, len(coords))
	for _, c := range coords {
		if len(c) < 2 {
			return nil, fmt.Errorf("polygon point needs [lat,lng]")
		}
		pts = append(pts, Point{Lat: c[0], Lng: c[1]})
	}
	return pts, nil
}

// PolygonJSON serialises points into the persisted JSON form
// (`[[lat,lng],...]`).
func PolygonJSON(pts []Point) (string, error) {
	coords := make([][]float64, 0, len(pts))
	for _, p := range pts {
		coords = append(coords, []float64{p.Lat, p.Lng})
	}
	raw, err := json.Marshal(coords)
	if err != nil {
		return "", fmt.Errorf("serialize polygon: %w", err)
	}
	return string(raw), nil
}

// ContainsEntry reports whether the point falls inside the zone using the
// entry test: the zone is expanded by bufferM metres (spec: BufferMetres).
func (g *Geofence) ContainsEntry(lat, lng, bufferM float64) bool {
	switch g.Shape {
	case ShapeCircle:
		r := g.RadiusM + bufferM
		return r > 0 && CircleContains(g.CenterLat, g.CenterLng, r, lat, lng)
	case ShapePolygon:
		p := Point{Lat: lat, Lng: lng}
		if PointInPolygon(lat, lng, g.Polygon) {
			return true
		}
		return PointToPolygonDistance(p, g.Polygon) <= bufferM
	}
	return false
}

// ContainsExit reports whether the point falls inside the zone using the
// exit test: the zone is contracted by hysteresisM metres (spec:
// HysteresisMetres). Inside AND farther than hysteresis from the boundary.
func (g *Geofence) ContainsExit(lat, lng, hysteresisM float64) bool {
	switch g.Shape {
	case ShapeCircle:
		r := g.RadiusM - hysteresisM
		return r > 0 && CircleContains(g.CenterLat, g.CenterLng, r, lat, lng)
	case ShapePolygon:
		if !PointInPolygon(lat, lng, g.Polygon) {
			return false
		}
		return PointToPolygonDistance(Point{Lat: lat, Lng: lng}, g.Polygon) > hysteresisM
	}
	return false
}
