package router

import (
	"errors"
	"net"
	"regexp"
	"strings"
	"time"
)

const (
	PrimaryControlHost   = "dev.localhost"
	SecondaryControlHost = "router.localhost"
	LocalhostSuffix      = ".localhost"
	DefaultTargetHost    = "127.0.0.1"
	DefaultHeartbeatPath = "/"

	UnavailableAfterMisses = 3
	RemoveAfterMisses      = 5
)

var validNamePattern = regexp.MustCompile(`^[a-z0-9-]+$`)

var (
	ErrInvalidName  = errors.New("invalid route name")
	ErrReservedName = errors.New("reserved route name")
	ErrConflict     = errors.New("route already exists")
	ErrNotFound     = errors.New("route not found")
)

type Route struct {
	Name          string    `json:"name"`
	Host          string    `json:"host"`
	TargetHost    string    `json:"targetHost"`
	Port          int       `json:"port"`
	Title         string    `json:"title,omitempty"`
	Pinned        bool      `json:"pinned"`
	Exec          string    `json:"exec,omitempty"`
	HeartbeatPath string    `json:"heartbeatPath"`
	CreatedAt     time.Time `json:"createdAt"`
	LastCheckAt   time.Time `json:"lastCheckAt,omitempty"`
	Misses        int       `json:"misses"`
}

type RouteView struct {
	Route
	URL    string `json:"url"`
	Status string `json:"status"`
}

type RegisterRequest struct {
	Port          int    `json:"port"`
	Title         string `json:"title,omitempty"`
	TargetHost    string `json:"targetHost,omitempty"`
	Pinned        bool   `json:"pinned,omitempty"`
	Exec          string `json:"exec,omitempty"`
	HeartbeatPath string `json:"heartbeatPath,omitempty"`
}

func NormalizeName(input string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(input))
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ' || r == ',' || r == '\t' || r == '\n':
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		default:
			b.WriteByte('0')
			lastDash = false
		}
	}
	name = strings.Trim(b.String(), "-")
	name = collapseDashes(name)
	if name == "" || strings.Contains(name, ".") || !validNamePattern.MatchString(name) {
		return "", ErrInvalidName
	}
	if name == "dev" || name == "router" {
		return "", ErrReservedName
	}
	return name, nil
}

func collapseDashes(s string) string {
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return s
}

func stripHostPort(host string) string {
	if strings.Contains(host, ":") {
		if h, _, err := net.SplitHostPort(host); err == nil {
			return strings.ToLower(h)
		}
	}
	return strings.ToLower(host)
}

func hostForName(name string) string { return name + LocalhostSuffix }

func urlForName(name string) string { return "http://" + hostForName(name) }

func statusFor(route Route) string {
	if route.Misses >= UnavailableAfterMisses {
		if route.Pinned {
			return "pinned unavailable"
		}
		return "unavailable"
	}
	if route.Pinned {
		return "pinned"
	}
	return "live"
}
