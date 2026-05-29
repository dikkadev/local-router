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

func TestStoreListDeletePinAndHeartbeat(t *testing.T) {
	store := NewStore()
	_, _ = store.Register("b", RegisterRequest{Port: 2}, false)
	_, _ = store.Register("a", RegisterRequest{Port: 1, Pinned: true}, false)
	list := store.List()
	if len(list) != 2 || list[0].Name != "a" || list[1].Name != "b" {
		t.Fatalf("list not sorted: %+v", list)
	}
	if _, err := store.SetPinned("b", true); err != nil {
		t.Fatal(err)
	}
	store.updateHeartbeat("b", false, time.Now())
	store.updateHeartbeat("b", false, time.Now())
	store.updateHeartbeat("b", false, time.Now())
	if got := store.List()[1].Status; got != "pinned unavailable" {
		t.Fatalf("status=%q", got)
	}
	for range 3 {
		store.updateHeartbeat("b", false, time.Now())
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
	for range 5 {
		store.updateHeartbeat("demo", false, time.Now())
	}
	if _, ok := store.GetByHost("demo.localhost"); ok {
		t.Fatal("non-pinned route should be removed after five misses")
	}
}
