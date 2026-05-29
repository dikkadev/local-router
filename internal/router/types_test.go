package router

import (
	"errors"
	"testing"
)

func TestNormalizeName(t *testing.T) {
	tests := map[string]string{
		"Demo App":        "demo-app",
		"a__b,,c":         "a-b-c",
		" --Hello---You ": "hello-you",
		"wow!":            "wow0",
		"MiXeD_Thing":     "mixed-thing",
	}
	for input, want := range tests {
		got, err := NormalizeName(input)
		if err != nil {
			t.Fatalf("NormalizeName(%q) returned error: %v", input, err)
		}
		if got != want {
			t.Fatalf("NormalizeName(%q)=%q, want %q", input, got, want)
		}
	}
}

func TestNormalizeNameRejectsInvalidAndReserved(t *testing.T) {
	for _, input := range []string{"", "---", "   ", "dev", "router"} {
		_, err := NormalizeName(input)
		if err == nil {
			t.Fatalf("NormalizeName(%q) expected error", input)
		}
	}
	_, err := NormalizeName("dev")
	if !errors.Is(err, ErrReservedName) {
		t.Fatalf("dev error=%v, want ErrReservedName", err)
	}
}

func TestHostURLAndStatus(t *testing.T) {
	if hostForName("demo") != "demo.localhost" {
		t.Fatal("unexpected host")
	}
	if urlForName("demo") != "http://demo.localhost" {
		t.Fatal("unexpected URL")
	}
	if statusFor(Route{}) != "live" || statusFor(Route{Pinned: true}) != "pinned" || statusFor(Route{Misses: 3}) != "unavailable" || statusFor(Route{Pinned: true, Misses: 3}) != "pinned unavailable" {
		t.Fatal("unexpected status mapping")
	}
}
