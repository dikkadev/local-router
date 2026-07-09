package router

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStateFileRoundTripShieldsImportedRoutes(t *testing.T) {
	created := time.Date(2026, 6, 1, 12, 34, 56, 0, time.UTC)
	store := NewStore()
	_, err := store.Register("demo", RegisterRequest{
		Port:          5173,
		Title:         "Demo app",
		TargetHost:    "127.0.0.1",
		Exec:          "vp dev",
		HeartbeatPath: "/health",
		CreatedAt:     created,
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "state", "routes.jsonl")
	if err := store.SaveStateFile(path); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "shielded") || strings.Contains(string(data), "misses") || strings.Contains(string(data), "lastCheckAt") {
		t.Fatalf("state includes transient fields: %s", data)
	}

	restored := NewStore()
	count, err := restored.LoadStateFile(path, true)
	if err != nil {
		t.Fatal(err)
	}
	route, err := restored.Get("demo")
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || !route.Shielded || route.Port != 5173 || route.Title != "Demo app" || route.Exec != "vp dev" || route.HeartbeatPath != "/health" || !route.CreatedAt.Equal(created) {
		t.Fatalf("unexpected restored route: count=%d route=%+v", count, route)
	}
}

func TestLoadStateFileMissingAndMalformed(t *testing.T) {
	store := NewStore()
	missing := filepath.Join(t.TempDir(), "missing.jsonl")
	count, err := store.LoadStateFile(missing, true)
	if err != nil || count != 0 {
		t.Fatalf("missing file: count=%d err=%v", count, err)
	}

	path := filepath.Join(t.TempDir(), "bad.jsonl")
	if err := os.WriteFile(path, []byte("not-json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadStateFile(path, true); err == nil || !strings.Contains(err.Error(), "line 1") {
		t.Fatalf("malformed state error=%v", err)
	}
}

func TestLoadStateFileDoesNotReplaceStoreOnValidationFailure(t *testing.T) {
	store := NewStore()
	_, _ = store.Register("existing", RegisterRequest{Port: 1111}, false)
	path := filepath.Join(t.TempDir(), "routes.jsonl")
	data := "{\"name\":\"valid\",\"port\":2222}\n{\"name\":\"invalid\",\"port\":99999}\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadStateFile(path, true); err == nil {
		t.Fatal("expected validation failure")
	}
	if _, err := store.Get("existing"); err != nil {
		t.Fatalf("failed load replaced existing store: %v", err)
	}
}
