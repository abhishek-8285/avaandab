package application

import (
	"testing"
	"time"

	"transport-app/internal/geofence/domain"
	"transport-app/internal/shared"
)

// zone builds a circular geofence for engine tests.
func zone(id, kind string, centerLat, centerLng, radius float64) domain.Geofence {
	return domain.Geofence{
		ID: id, TenantID: string(shared.DefaultTenant), Name: id, Kind: kind, Shape: domain.ShapeCircle,
		CenterLat: centerLat, CenterLng: centerLng, RadiusM: radius, IsActive: true,
	}
}

func fix(vehicle string, t time.Time, lat, lng float64) domain.Fix {
	return domain.Fix{VehicleID: vehicle, Timestamp: t, Latitude: lat, Longitude: lng}
}

// insidePoint returns a point `northM` metres north of the centre.
func insidePoint(centerLat, centerLng, northM float64) (float64, float64) {
	return centerLat + northM/111320.0, centerLng
}

func state(s string) domain.EngineState {
	return domain.EngineState{VehicleID: "v1", TenantID: string(shared.DefaultTenant), State: s}
}

const (
	clat = 12.97
	clng = 77.59
)

func TestDwellEngine_JitterRejectionOutsideToEnteringToOutside(t *testing.T) {
	eng := NewDwellEngine(EngineConfig{Debounce: 60 * time.Second, BufferMetres: 20, HysteresisMetres: 25})
	zones := []domain.Geofence{zone("z1", domain.KindDepot, clat, clng, 100)}
	t0 := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)

	latIn, lngIn := insidePoint(clat, clng, 50) // inside the circle

	next, evs := eng.Evaluate(state(domain.StateOutside), fix("v1", t0, latIn, lngIn), zones)
	if next.State != domain.StateEntering {
		t.Fatalf("first inside fix: state = %s, want entering", next.State)
	}
	if len(evs) != 0 {
		t.Fatalf("entry probe must not emit events, got %d", len(evs))
	}
	if next.ZoneEnteredAt == nil || !next.ZoneEnteredAt.Equal(t0) {
		t.Fatal("zone_entered_at must be set to the first inside fix time")
	}

	// One miss < debounce reverts to outside (jitter ignored).
	latFar, lngFar := insidePoint(clat, clng, 1000)
	next2, evs2 := eng.Evaluate(next, fix("v1", t0.Add(10*time.Second), latFar, lngFar), zones)
	if next2.State != domain.StateOutside {
		t.Fatalf("jitter miss: state = %s, want outside", next2.State)
	}
	if next2.GeofenceID != nil || next2.ZoneKind != nil {
		t.Fatal("jitter miss must clear zone fields")
	}
	if len(evs2) != 0 {
		t.Fatalf("jitter miss must not emit events, got %d", len(evs2))
	}
}

func TestDwellEngine_DebounceConfirmation(t *testing.T) {
	eng := NewDwellEngine(EngineConfig{Debounce: 60 * time.Second, BufferMetres: 20, HysteresisMetres: 25})
	zones := []domain.Geofence{zone("z1", domain.KindDepot, clat, clng, 100)}
	t0 := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)
	latIn, lngIn := insidePoint(clat, clng, 50)

	// t0: entering probe.
	next, _ := eng.Evaluate(state(domain.StateOutside), fix("v1", t0, latIn, lngIn), zones)

	// t0+30s: still inside but < debounce → still entering.
	next2, evs := eng.Evaluate(next, fix("v1", t0.Add(30*time.Second), latIn, lngIn), zones)
	if next2.State != domain.StateEntering {
		t.Fatalf("30s inside: state = %s, want entering", next2.State)
	}
	if len(evs) != 0 {
		t.Fatalf("pre-debounce must not confirm, got %d events", len(evs))
	}

	// t0+60s: debounce satisfied → inside + entering event.
	next3, evs3 := eng.Evaluate(next2, fix("v1", t0.Add(60*time.Second), latIn, lngIn), zones)
	if next3.State != domain.StateInside {
		t.Fatalf("60s inside: state = %s, want inside", next3.State)
	}
	if next3.ConfirmedAt == nil || !next3.ConfirmedAt.Equal(t0.Add(60*time.Second)) {
		t.Fatal("confirmed_at must be set on confirmation")
	}
	if len(evs3) != 1 || evs3[0].EventType != domain.EventEntering {
		t.Fatalf("confirmation must emit one entering event, got %+v", evs3)
	}
}

func TestDwellEngine_HysteresisExitTest(t *testing.T) {
	eng := NewDwellEngine(EngineConfig{Debounce: 60 * time.Second, BufferMetres: 20, HysteresisMetres: 25})
	zones := []domain.Geofence{zone("z1", domain.KindDepot, clat, clng, 100)}
	t0 := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)

	// Confirmed inside state with zone attached.
	st := state(domain.StateInside)
	gid, kind := "z1", domain.KindDepot
	entered := t0.Add(-10 * time.Minute)
	st.GeofenceID = &gid
	st.ZoneKind = &kind
	st.ZoneEnteredAt = &entered
	st.ConfirmedAt = &entered

	// 80m from centre: inside the nominal 100m zone but BEYOND the
	// contracted 75m (100-25) → exit probe starts.
	lat80, lng80 := insidePoint(clat, clng, 80)
	next, evs := eng.Evaluate(st, fix("v1", t0, lat80, lng80), zones)
	if next.State != domain.StateLeaving {
		t.Fatalf("80m fix: state = %s, want leaving (hysteresis breach)", next.State)
	}
	if len(evs) != 0 {
		t.Fatalf("exit probe must not emit events, got %d", len(evs))
	}

	// 70m from centre: within contracted 75m → still inside.
	lat70, lng70 := insidePoint(clat, clng, 70)
	next2, _ := eng.Evaluate(state(domain.StateInside), fix("v1", t0, lat70, lng70), zones)
	if next2.State != domain.StateInside {
		t.Fatalf("70m fix: state = %s, want inside", next2.State)
	}
}

func TestDwellEngine_LeavingDebounceAndJitter(t *testing.T) {
	eng := NewDwellEngine(EngineConfig{Debounce: 60 * time.Second, BufferMetres: 20, HysteresisMetres: 25})
	zones := []domain.Geofence{zone("z1", domain.KindDepot, clat, clng, 100)}
	t0 := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)

	gid, kind := "z1", domain.KindDepot
	st := state(domain.StateLeaving)
	exitStart := t0.Add(-30 * time.Second)
	st.GeofenceID = &gid
	st.ZoneKind = &kind
	st.ExitStartedAt = &exitStart

	latFar, lngFar := insidePoint(clat, clng, 1000)

	// Still outside, < debounce → stays leaving.
	next, evs := eng.Evaluate(st, fix("v1", t0, latFar, lngFar), zones)
	if next.State != domain.StateLeaving {
		t.Fatalf("pre-debounce outside: state = %s, want leaving", next.State)
	}
	if len(evs) != 0 {
		t.Fatalf("pre-debounce exit must not emit, got %d", len(evs))
	}

	// Jitter: back inside → revert to inside, clear exit timer.
	latIn, lngIn := insidePoint(clat, clng, 50)
	next2, _ := eng.Evaluate(next, fix("v1", t0.Add(5*time.Second), latIn, lngIn), zones)
	if next2.State != domain.StateInside {
		t.Fatalf("jitter back inside: state = %s, want inside", next2.State)
	}
	if next2.ExitStartedAt != nil {
		t.Fatal("jitter revert must clear exit_started_at")
	}

	// Debounce satisfied: leaving → outside + leaving event.
	st2 := state(domain.StateLeaving)
	exitStart2 := t0.Add(-60 * time.Second)
	st2.GeofenceID = &gid
	st2.ZoneKind = &kind
	st2.ExitStartedAt = &exitStart2
	next3, evs3 := eng.Evaluate(st2, fix("v1", t0, latFar, lngFar), zones)
	if next3.State != domain.StateOutside {
		t.Fatalf("debounced exit: state = %s, want outside", next3.State)
	}
	if len(evs3) != 1 || evs3[0].EventType != domain.EventLeaving {
		t.Fatalf("debounced exit must emit one leaving event, got %+v", evs3)
	}
}

func TestDwellEngine_RestrictedZoneBreachAlert(t *testing.T) {
	eng := NewDwellEngine(EngineConfig{Debounce: 10 * time.Second, BufferMetres: 20, HysteresisMetres: 25})
	zones := []domain.Geofence{zone("zr", domain.KindRestricted, clat, clng, 100)}
	t0 := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)
	latIn, lngIn := insidePoint(clat, clng, 30)

	next, _ := eng.Evaluate(state(domain.StateOutside), fix("v1", t0, latIn, lngIn), zones)
	next2, evs := eng.Evaluate(next, fix("v1", t0.Add(10*time.Second), latIn, lngIn), zones)
	if next2.State != domain.StateInside {
		t.Fatalf("state = %s, want inside", next2.State)
	}
	if len(evs) != 2 {
		t.Fatalf("restricted confirmation must emit entering + breach, got %+v", evs)
	}
	if evs[0].EventType != domain.EventEntering || evs[1].EventType != domain.EventBreach {
		t.Fatalf("event order = %s, %s; want entering, breach", evs[0].EventType, evs[1].EventType)
	}
	if evs[1].Details == "" {
		t.Fatal("breach event needs details")
	}
}

func TestDwellEngine_ZoneSwitchRestartsEntryTimer(t *testing.T) {
	eng := NewDwellEngine(EngineConfig{Debounce: 60 * time.Second, BufferMetres: 20, HysteresisMetres: 25})
	zA := zone("a", domain.KindPickup, clat, clng, 200)
	zB := zone("b", domain.KindDrop, clat+0.002, clng, 150)
	zones := []domain.Geofence{zA, zB}
	t0 := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)

	latA, lngA := insidePoint(clat, clng, 50)
	latB, lngB := insidePoint(clat+0.002, clng, 30)

	next, _ := eng.Evaluate(state(domain.StateOutside), fix("v1", t0, latA, lngA), zones)
	if next.GeofenceID == nil || *next.GeofenceID != "a" {
		t.Fatalf("probe must attach to zone a, got %v", next.GeofenceID)
	}

	// Fix inside zone b before debounce on a → switch, restart timer.
	next2, _ := eng.Evaluate(next, fix("v1", t0.Add(10*time.Second), latB, lngB), zones)
	if next2.State != domain.StateEntering {
		t.Fatalf("switched probe: state = %s, want entering", next2.State)
	}
	if next2.GeofenceID == nil || *next2.GeofenceID != "b" {
		t.Fatalf("switched probe must attach to zone b, got %v", next2.GeofenceID)
	}
	if next2.ZoneEnteredAt == nil || !next2.ZoneEnteredAt.Equal(t0.Add(10*time.Second)) {
		t.Fatal("zone switch must restart the entry timer")
	}
}

func TestDwellEngine_LastFixAlwaysAdvances(t *testing.T) {
	eng := NewDwellEngine(EngineConfig{Debounce: 60 * time.Second, BufferMetres: 20, HysteresisMetres: 25})
	zones := []domain.Geofence{zone("z1", domain.KindDepot, clat, clng, 100)}
	t0 := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)
	latFar, lngFar := insidePoint(clat, clng, 1000)

	next, _ := eng.Evaluate(state(domain.StateOutside), fix("v1", t0, latFar, lngFar), zones)
	if next.State != domain.StateOutside {
		t.Fatalf("far fix must stay outside, got %s", next.State)
	}
	if !next.LastFixAt.Equal(t0) || next.LastLat != latFar || next.LastLng != lngFar {
		t.Fatal("last fix watermark must advance for non-zone fixes")
	}
}
