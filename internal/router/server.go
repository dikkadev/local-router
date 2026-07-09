package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

type Server struct {
	Store             *Store
	HeartbeatInterval time.Duration
	HTTPClient        *http.Client
	KillProcessByPort func(port int) ([]int, error)
	ExternalHost      string
}

func NewServer() *Server {
	return &Server{
		Store:             NewStore(),
		HeartbeatInterval: 10 * time.Second,
		HTTPClient:        &http.Client{Timeout: 3 * time.Second},
		KillProcessByPort: killProcessByPort,
	}
}

func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	lc := net.ListenConfig{}
	ln, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("cannot start local-router on %s: %w", addr, err)
	}
	log.Info("local-router listening", "addr", addr, "url", "http://"+addr, "externalHost", s.externalHost())
	s.StartHeartbeat(ctx)
	httpServer := &http.Server{Handler: s, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()
	if err := httpServer.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !isAllowedRemoteAddr(r.RemoteAddr) {
		http.Error(w, "local-router only accepts local requests", http.StatusForbidden)
		return
	}
	host := stripHostPort(r.Host)
	if s.isControlHost(host) {
		s.serveControl(w, r)
		return
	}
	if route, ok := s.routeByHost(host); ok {
		s.proxyRoute(w, r, route)
		return
	}
	missingRoutePage(w, host, s.dashboardURLForHost(host))
}

func (s *Server) serveControl(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/router/") {
		s.serveAPI(w, r)
		return
	}
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, controlHTML)
}

func (s *Server) serveAPI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/router/routes" && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, s.Store.List())
		return
	}
	name, ok := strings.CutPrefix(r.URL.Path, "/router/routes/")
	if !ok || name == "" || (strings.Contains(name, "/") && !(r.Method == http.MethodPost && strings.HasSuffix(name, "/kill"))) {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON request")
			return
		}
		force := r.URL.Query().Get("force") == "true"
		view, err := s.Store.Register(name, req, force)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		log.Info("route registered", "name", view.Name, "url", view.URL, "target", net.JoinHostPort(view.TargetHost, strconv.Itoa(view.Port)), "pinned", view.Pinned, "force", force)
		writeJSON(w, http.StatusOK, view)
	case http.MethodDelete:
		normalizedName, _ := NormalizeName(name)
		if err := s.Store.Delete(name); err != nil {
			writeStoreError(w, err)
			return
		}
		log.Info("route unregistered", "name", normalizedName, "url", urlForName(normalizedName))
		w.WriteHeader(http.StatusNoContent)
	case http.MethodPatch:
		var req struct {
			Pinned bool `json:"pinned"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON request")
			return
		}
		view, err := s.Store.SetPinned(name, req.Pinned)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if view.Pinned {
			log.Info("route pinned", "name", view.Name, "url", view.URL)
		} else {
			log.Info("route unpinned", "name", view.Name, "url", view.URL)
		}
		writeJSON(w, http.StatusOK, view)
	case http.MethodPost:
		if !strings.HasSuffix(name, "/kill") {
			w.Header().Set("Allow", "GET, PUT, DELETE, PATCH")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		routeName := strings.TrimSuffix(name, "/kill")
		route, err := s.Store.Get(routeName)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		kill := s.KillProcessByPort
		if kill == nil {
			kill = killProcessByPort
		}
		pids, err := kill(route.Port)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		log.Info("killed process for route", "name", route.Name, "port", route.Port, "pids", pids)
		writeJSON(w, http.StatusOK, map[string]any{"pids": pids})
	default:
		w.Header().Set("Allow", "GET, PUT, DELETE, PATCH")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) proxyRoute(w http.ResponseWriter, r *http.Request, route Route) {
	target := &url.URL{Scheme: "http", Host: net.JoinHostPort(route.TargetHost, strconv.Itoa(route.Port))}
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		origHost := r.Host
		forwardedProto := r.Header.Get("X-Forwarded-Proto")
		if forwardedProto == "" {
			forwardedProto = "http"
		}
		originalDirector(req)
		req.Host = route.TargetHost
		req.Header.Set("X-Forwarded-Host", stripHostPort(origHost))
		req.Header.Set("X-Forwarded-Proto", forwardedProto)
	}
	proxy.Transport = &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		DialContext:         (&net.Dialer{Timeout: 2 * time.Minute, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:   true,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		unavailableRoutePage(w, route, err, s.dashboardURLForHost(stripHostPort(r.Host)))
	}
	proxy.FlushInterval = -1
	proxy.ServeHTTP(w, r)
}

func (s *Server) StartHeartbeat(ctx context.Context) {
	interval := s.HeartbeatInterval
	if interval <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.CheckAllHeartbeats(ctx)
			}
		}
	}()
}

func (s *Server) CheckAllHeartbeats(ctx context.Context) {
	for _, route := range s.Store.snapshot() {
		hit := s.checkRoute(ctx, route)
		s.Store.updateHeartbeat(route, hit, time.Now())
	}
}

func (s *Server) checkRoute(ctx context.Context, route Route) bool {
	target := "http://" + net.JoinHostPort(route.TargetHost, strconv.Itoa(route.Port)) + route.HeartbeatPath
	for _, method := range []string{http.MethodHead, http.MethodGet} {
		req, err := http.NewRequestWithContext(ctx, method, target, nil)
		if err != nil {
			return false
		}
		resp, err := s.HTTPClient.Do(req)
		if err != nil {
			return false
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return true
		}
		if method == http.MethodHead && (resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotImplemented) {
			continue
		}
		return false
	}
	return false
}

func (s *Server) isControlHost(host string) bool {
	host = stripHostPort(host)
	return host == PrimaryControlHost || host == SecondaryControlHost || (s.externalHost() != "" && host == s.externalHost())
}

func (s *Server) routeByHost(host string) (Route, bool) {
	name, ok := s.routeNameForHost(host)
	if !ok {
		return Route{}, false
	}
	route, err := s.Store.Get(name)
	return route, err == nil
}

func (s *Server) routeNameForHost(host string) (string, bool) {
	host = stripHostPort(host)
	if stringsHasLocalhostSuffix(host) {
		name := host[:len(host)-len(LocalhostSuffix)]
		if _, err := NormalizeName(name); err == nil {
			return name, true
		}
		return "", false
	}
	externalHost := s.externalHost()
	if externalHost == "" || !strings.HasSuffix(host, "."+externalHost) {
		return "", false
	}
	name := strings.TrimSuffix(host, "."+externalHost)
	if _, err := NormalizeName(name); err != nil {
		return "", false
	}
	return name, true
}

func (s *Server) dashboardURLForHost(host string) string {
	host = stripHostPort(host)
	if externalHost := s.externalHost(); externalHost != "" && (host == externalHost || strings.HasSuffix(host, "."+externalHost)) {
		return "http://" + externalHost
	}
	return "http://" + PrimaryControlHost
}

func (s *Server) externalHost() string {
	return normalizeExternalHost(s.ExternalHost)
}

func normalizeExternalHost(raw string) string {
	host := strings.ToLower(strings.TrimSpace(raw))
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	if before, _, ok := strings.Cut(host, "/"); ok {
		host = before
	}
	host = strings.TrimSuffix(host, ".")
	return stripHostPort(host)
}

func killProcessByPort(port int) ([]int, error) {
	pids, err := listeningPIDs(port)
	if err != nil {
		return nil, err
	}
	if len(pids) == 0 {
		return nil, fmt.Errorf("no listening process found on port %d", port)
	}
	for _, pid := range pids {
		proc, err := os.FindProcess(pid)
		if err != nil {
			return pids, err
		}
		if err := proc.Kill(); err != nil {
			return pids, err
		}
	}
	return pids, nil
}

func listeningPIDs(port int) ([]int, error) {
	commands := [][]string{
		{"lsof", "-tiTCP:" + strconv.Itoa(port), "-sTCP:LISTEN"},
		{"fuser", strconv.Itoa(port) + "/tcp"},
	}
	var lastErr error
	seen := map[int]bool{}
	var pids []int
	for _, args := range commands {
		out, err := exec.Command(args[0], args[1:]...).Output()
		if err != nil {
			lastErr = err
			continue
		}
		for _, field := range strings.Fields(string(out)) {
			pid, err := strconv.Atoi(field)
			if err != nil || seen[pid] {
				continue
			}
			seen[pid] = true
			pids = append(pids, pid)
		}
		if len(pids) > 0 {
			return pids, nil
		}
	}
	return pids, lastErr
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrConflict):
		writeError(w, http.StatusConflict, "route already exists; use force=true to replace it")
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "route not found")
	case errors.Is(err, ErrInvalidName), errors.Is(err, ErrReservedName), errors.Is(err, errInvalidPort), errors.Is(err, errInvalidTargetHost):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func missingRoutePage(w http.ResponseWriter, host, dashboardURL string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = fmt.Fprintf(w, "<h1>No route registered for %s</h1><p>Open <a href=\"%s\">%s</a> to inspect current registrations.</p>", html.EscapeString(host), html.EscapeString(dashboardURL), html.EscapeString(strings.TrimPrefix(dashboardURL, "http://")))
}

func unavailableRoutePage(w http.ResponseWriter, route Route, err error, dashboardURL string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(523)
	_, _ = fmt.Fprintf(w, "<h1>%s is unavailable</h1><p>The route is registered, but local-router cannot reach %s:%d.</p><p>Open <a href=\"%s\">%s</a> to inspect registrations.</p><pre>%s</pre>", html.EscapeString(route.Host), html.EscapeString(route.TargetHost), route.Port, html.EscapeString(dashboardURL), html.EscapeString(strings.TrimPrefix(dashboardURL, "http://")), html.EscapeString(err.Error()))
}

func isAllowedRemoteAddr(remoteAddr string) bool {
	if remoteAddr == "" {
		return true
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	if isAllowedClientAddr(addr) {
		return true
	}
	if addr.Is4In6() {
		return isAllowedClientAddr(addr.Unmap())
	}
	return false
}

func isAllowedClientAddr(addr netip.Addr) bool {
	return addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() || isCGNATAddr(addr)
}

func isCGNATAddr(addr netip.Addr) bool {
	prefix := netip.MustParsePrefix("100.64.0.0/10")
	return addr.Is4() && prefix.Contains(addr)
}
