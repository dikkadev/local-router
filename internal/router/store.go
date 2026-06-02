package router

import (
	"net"
	"sort"
	"sync"
	"time"
)

type Store struct {
	mu     sync.RWMutex
	routes map[string]Route
}

func NewStore() *Store {
	return &Store{routes: make(map[string]Route)}
}

func (s *Store) Register(rawName string, req RegisterRequest, force bool) (RouteView, error) {
	name, err := NormalizeName(rawName)
	if err != nil {
		return RouteView{}, err
	}
	if req.Port < 1 || req.Port > 65535 {
		return RouteView{}, errInvalidPort
	}
	targetHost := req.TargetHost
	if targetHost == "" {
		targetHost = DefaultTargetHost
	}
	if net.ParseIP(targetHost) == nil && targetHost != "localhost" {
		return RouteView{}, errInvalidTargetHost
	}
	heartbeatPath := req.HeartbeatPath
	if heartbeatPath == "" {
		heartbeatPath = DefaultHeartbeatPath
	}
	if heartbeatPath[0] != '/' {
		heartbeatPath = "/" + heartbeatPath
	}
	createdAt := req.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.routes[name]; ok && !force {
		return RouteView{}, ErrConflict
	}
	route := Route{
		Name:          name,
		Host:          hostForName(name),
		TargetHost:    targetHost,
		Port:          req.Port,
		Title:         req.Title,
		Pinned:        req.Pinned,
		Exec:          req.Exec,
		HeartbeatPath: heartbeatPath,
		CreatedAt:     createdAt,
	}
	s.routes[name] = route
	return viewFor(route), nil
}

func (s *Store) Delete(rawName string) error {
	name, err := NormalizeName(rawName)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.routes[name]; !ok {
		return ErrNotFound
	}
	delete(s.routes, name)
	return nil
}

func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.routes = make(map[string]Route)
}

func (s *Store) Get(rawName string) (Route, error) {
	name, err := NormalizeName(rawName)
	if err != nil {
		return Route{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	route, ok := s.routes[name]
	if !ok {
		return Route{}, ErrNotFound
	}
	return route, nil
}

func (s *Store) GetByHost(host string) (Route, bool) {
	host = stripHostPort(host)
	if !stringsHasLocalhostSuffix(host) {
		return Route{}, false
	}
	name := host[:len(host)-len(LocalhostSuffix)]
	s.mu.RLock()
	defer s.mu.RUnlock()
	route, ok := s.routes[name]
	return route, ok
}

func (s *Store) List() []RouteView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	views := make([]RouteView, 0, len(s.routes))
	for _, route := range s.routes {
		views = append(views, viewFor(route))
	}
	sort.Slice(views, func(i, j int) bool { return views[i].Name < views[j].Name })
	return views
}

func (s *Store) SetPinned(rawName string, pinned bool) (RouteView, error) {
	name, err := NormalizeName(rawName)
	if err != nil {
		return RouteView{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	route, ok := s.routes[name]
	if !ok {
		return RouteView{}, ErrNotFound
	}
	route.Pinned = pinned
	s.routes[name] = route
	return viewFor(route), nil
}

func (s *Store) updateHeartbeat(name string, hit bool, checkedAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	route, ok := s.routes[name]
	if !ok {
		return
	}
	route.LastCheckAt = checkedAt
	if hit {
		route.Misses = 0
	} else {
		route.Misses++
	}
	if route.Misses >= RemoveAfterMisses && !route.Pinned {
		delete(s.routes, name)
		return
	}
	s.routes[name] = route
}

func (s *Store) snapshot() []Route {
	s.mu.RLock()
	defer s.mu.RUnlock()
	routes := make([]Route, 0, len(s.routes))
	for _, route := range s.routes {
		routes = append(routes, route)
	}
	return routes
}

func viewFor(route Route) RouteView {
	return RouteView{Route: route, URL: urlForName(route.Name), Status: statusFor(route)}
}

func stringsHasLocalhostSuffix(host string) bool {
	return len(host) > len(LocalhostSuffix) && host[len(host)-len(LocalhostSuffix):] == LocalhostSuffix
}
