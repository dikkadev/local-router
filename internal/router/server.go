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
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

type Server struct {
	Store             *Store
	HeartbeatInterval time.Duration
	HTTPClient        *http.Client
}

func NewServer() *Server {
	return &Server{
		Store:             NewStore(),
		HeartbeatInterval: 30 * time.Second,
		HTTPClient:        &http.Client{Timeout: 3 * time.Second},
	}
}

func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	lc := net.ListenConfig{}
	ln, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("cannot start local-router on %s: %w", addr, err)
	}
	log.Infof("local-router listening on http://%s", addr)
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
	if host == PrimaryControlHost || host == SecondaryControlHost {
		s.serveControl(w, r)
		return
	}
	if route, ok := s.Store.GetByHost(host); ok {
		s.proxyRoute(w, r, route)
		return
	}
	missingRoutePage(w, host)
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
	if !ok || name == "" || strings.Contains(name, "/") {
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
		view, err := s.Store.Register(name, req, r.URL.Query().Get("force") == "true")
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, view)
	case http.MethodDelete:
		if err := s.Store.Delete(name); err != nil {
			writeStoreError(w, err)
			return
		}
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
		writeJSON(w, http.StatusOK, view)
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
		originalDirector(req)
		req.Host = route.TargetHost
		req.Header.Set("X-Forwarded-Host", stripHostPort(origHost))
		req.Header.Set("X-Forwarded-Proto", "http")
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
		unavailableRoutePage(w, route, err)
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
		s.Store.updateHeartbeat(route.Name, hit, time.Now())
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

func missingRoutePage(w http.ResponseWriter, host string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = fmt.Fprintf(w, "<h1>No route registered for %s</h1><p>Open <a href=\"http://dev.localhost\">dev.localhost</a> to inspect current registrations.</p>", html.EscapeString(host))
}

func unavailableRoutePage(w http.ResponseWriter, route Route, err error) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(523)
	_, _ = fmt.Fprintf(w, "<h1>%s is unavailable</h1><p>The route is registered, but local-router cannot reach %s:%d.</p><p>Open <a href=\"http://dev.localhost\">dev.localhost</a> to inspect registrations.</p><pre>%s</pre>", html.EscapeString(route.Host), html.EscapeString(route.TargetHost), route.Port, html.EscapeString(err.Error()))
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
	if addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() {
		return true
	}
	if addr.Is4In6() {
		v4 := addr.Unmap()
		return v4.IsLoopback() || v4.IsPrivate() || v4.IsLinkLocalUnicast()
	}
	return false
}
