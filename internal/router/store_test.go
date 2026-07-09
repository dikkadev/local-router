package router

import (
	"errors"
	"testing"
	"time"
)

func TestStoreRegisterDefaultsConflictAndForce(t *testing.T) {
	store := NewStore()
	view, err := store.Register("Demo App", RegisterRequest{Port: 5173}, false)
	if err != nil {
		t.Fatal(err)
	}
	if view.Name != "demo-app" || view.Host != "demo-app.localhost" || view.URL != "http://demo-app.localhost" {
		t.Fatalf("unexpected view: %+v", view)
	}
	if view.TargetHost != DefaultTargetHost || view.HeartbeatPath != DefaultHeartbeatPath || view.CreatedAt.IsZero() {
		t.Fatalf("defaults not applied: %+v", view.Route)
	}
	_, err = store.Register("demo app", RegisterRequest{Port: 3000}, false)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("conflict error=%v", err)
	}
	forced, err := store.Register("demo app", RegisterRequest{Port: 3000, Pinned: true}, true)
	if err != nil {
		t.Fatalf("force replace failed: %v", err)
	}
	if forced.Port != 3000 || !forced.Pinned {
		t.Fatalf("force replace got %+v", forced.Route)
	}
}

func TestStoreListOrdersPinnedFirstThenOldest(t *testing.T) {
	store := NewStore()
	oldest := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	middle := oldest.Add(time.Hour)
	newest := oldest.Add(2 * time.Hour)
	_, _ = store.Register("z-old-unpinned", RegisterRequest{Port: 1, CreatedAt: oldest}, false)
	_, _ = store.Register("a-middle-unpinned", RegisterRequest{Port: 2, CreatedAt: middle}, false)
	_, _ = store.Register("a-new-pinned", RegisterRequest{Port: 3, Pinned: true, CreatedAt: newest}, false)
	_, _ = store.Register("b-old-pinned", RegisterRequest{Port: 4, Pinned: true, CreatedAt: oldest}, false)

	list := store.List()
	got := []string{list[0].Name, list[1].Name, list[2].Name, list[3].Name}
	want := []string{"b-old-pinned", "a-new-pinned", "z-old-unpinned", "a-middle-unpinned"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("list order = %v, want %v", got, want)
		}
	}
}

func TestStoreDeletePinAndHeartbeat(t *testing.T) {
	store := NewStore()
	_, _ = store.Register("b", RegisterRequest{Port: 2}, false)
	_, _ = store.Register("a", RegisterRequest{Port: 1, Pinned: true}, false)
	if _, err := store.SetPinned("b", true); err != nil {
		t.Fatal(err)
	}
	checked, _ := store.Get("b")
	store.updateHeartbeat(checked, false, time.Now())
	store.updateHeartbeat(checked, false, time.Now())
	store.updateHeartbeat(checked, false, time.Now())
	route, err := store.Get("b")
	if err != nil {
		t.Fatal(err)
	}
	if got := statusFor(route); got != "pinned unavailable" {
		t.Fatalf("status=%q", got)
	}
	for range 3 {
		store.updateHeartbeat(checked, false, time.Now())
	}
	if _, ok := store.GetByHost("b.localhost"); !ok {
		t.Fatal("pinned route should remain")
	}
	if err := store.Delete("b"); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete("b"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete missing error=%v", err)
	}
}

func TestHeartbeatRemovesNonPinnedAfterFiveMisses(t *testing.T) {
	store := NewStore()
	_, _ = store.Register("demo", RegisterRequest{Port: 5173}, false)
	checked, _ := store.Get("demo")
	for range 5 {
		store.updateHeartbeat(checked, false, time.Now())
	}
	if _, ok := store.GetByHost("demo.localhost"); ok {
		t.Fatal("non-pinned route should be removed after five misses")
	}
}

func TestShieldedRouteSurvivesUntilFirstHitThenDecaysNormally(t *testing.T) {
	store := NewStore()
	_, _ = store.Register("demo", RegisterRequest{Port: 5173, Shielded: true}, false)
	checked, _ := store.Get("demo")
	for range 8 {
		store.updateHeartbeat(checked, false, time.Now())
	}
	route, err := store.Get("demo")
	if err != nil || !route.Shielded || statusFor(route) != "restored unavailable" {
		t.Fatalf("shielded route did not survive: route=%+v err=%v", route, err)
	}

	store.updateHeartbeat(route, true, time.Now())
	route, _ = store.Get("demo")
	if route.Shielded || route.Misses != 0 || statusFor(route) != "live" {
		t.Fatalf("successful heartbeat did not remove shield: %+v", route)
	}
	for range 5 {
		store.updateHeartbeat(route, false, time.Now())
	}
	if _, err := store.Get("demo"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unshielded route should decay normally: %v", err)
	}
}

func TestStaleHeartbeatDoesNotMutateReplacement(t *testing.T) {
	store := NewStore()
	_, _ = store.Register("demo", RegisterRequest{Port: 5173, Shielded: true}, false)
	stale, _ := store.Get("demo")
	_, _ = store.Register("demo", RegisterRequest{Port: 3000}, true)
	store.updateHeartbeat(stale, false, time.Now())
	replacement, _ := store.Get("demo")
	if replacement.Port != 3000 || replacement.Misses != 0 {
		t.Fatalf("stale heartbeat mutated replacement: %+v", replacement)
	}
}
