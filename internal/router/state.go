package router

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type snapshotRoute struct {
	Name          string    `json:"name"`
	Port          int       `json:"port"`
	Title         string    `json:"title,omitempty"`
	TargetHost    string    `json:"targetHost,omitempty"`
	Pinned        bool      `json:"pinned,omitempty"`
	Exec          string    `json:"exec,omitempty"`
	HeartbeatPath string    `json:"heartbeatPath,omitempty"`
	CreatedAt     time.Time `json:"createdAt,omitempty"`
}

// LoadStateFile replaces the store contents with routes from path. A missing file
// is treated as an empty initial state. Shielded routes survive heartbeat misses
// until their target responds successfully once.
func (s *Store) LoadStateFile(path string, shielded bool) (int, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	defer file.Close()

	routes, err := readState(file)
	if err != nil {
		return 0, err
	}

	loaded := NewStore()
	for _, route := range routes {
		_, err = loaded.Register(route.Name, RegisterRequest{
			Port:          route.Port,
			Title:         route.Title,
			TargetHost:    route.TargetHost,
			Pinned:        route.Pinned,
			Shielded:      shielded,
			Exec:          route.Exec,
			HeartbeatPath: route.HeartbeatPath,
			CreatedAt:     route.CreatedAt,
		}, false)
		if err != nil {
			return 0, fmt.Errorf("route %s: %w", route.Name, err)
		}
	}

	s.mu.Lock()
	s.routes = loaded.routes
	s.mu.Unlock()
	return len(routes), nil
}

// SaveStateFile atomically writes durable route definitions. Transient health
// fields, including the one-time shield, are intentionally not persisted.
func (s *Store) SaveStateFile(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, ".routes-*.jsonl")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}

	enc := json.NewEncoder(temp)
	routes := s.snapshot()
	sort.Slice(routes, func(i, j int) bool { return routes[i].Name < routes[j].Name })
	for _, route := range routes {
		if err := enc.Encode(snapshotFromRoute(route)); err != nil {
			_ = temp.Close()
			return err
		}
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func readState(r io.Reader) ([]snapshotRoute, error) {
	scanner := bufio.NewScanner(r)
	var routes []snapshotRoute
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var route snapshotRoute
		if err := json.Unmarshal([]byte(line), &route); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo, err)
		}
		if route.Name == "" {
			return nil, fmt.Errorf("line %d: missing name", lineNo)
		}
		if route.Port == 0 {
			return nil, fmt.Errorf("line %d: missing port", lineNo)
		}
		routes = append(routes, route)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return routes, nil
}

func snapshotFromRoute(route Route) snapshotRoute {
	return snapshotRoute{
		Name:          route.Name,
		Port:          route.Port,
		Title:         route.Title,
		TargetHost:    route.TargetHost,
		Pinned:        route.Pinned,
		Exec:          route.Exec,
		HeartbeatPath: route.HeartbeatPath,
		CreatedAt:     route.CreatedAt,
	}
}
