package router

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAPIRegisterListDeleteAndControlPage(t *testing.T) {
	s := NewServer()
	reqBody := strings.NewReader(`{"port":5173,"title":"Demo"}`)
	req := httptest.NewRequest(http.MethodPut, "http://dev.localhost/router/routes/Demo_App", reqBody)
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("register status=%d body=%s", rec.Code, rec.Body.String())
	}
	var view RouteView
	if err := json.NewDecoder(rec.Body).Decode(&view); err != nil {
		t.Fatal(err)
	}
	if view.Name != "demo-app" || view.URL != "http://demo-app.localhost" {
		t.Fatalf("unexpected view %+v", view)
	}

	rec = doRouterRequest(s, http.MethodGet, "http://router.localhost/router/routes", nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "demo-app") {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}
	rec = doRouterRequest(s, http.MethodPut, "http://dev.localhost/router/routes/demo-app", strings.NewReader(`{"port":3000}`))
	if rec.Code != http.StatusConflict {
		t.Fatalf("conflict status=%d", rec.Code)
	}
	rec = doRouterRequest(s, http.MethodPut, "http://dev.localhost/router/routes/demo-app?force=true", strings.NewReader(`{"port":3000}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("force status=%d", rec.Code)
	}
	rec = doRouterRequest(s, http.MethodGet, "http://dev.localhost/", nil)
	controlBody := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(controlBody, "/router/routes") {
		t.Fatalf("control page status=%d", rec.Code)
	}
	for _, want := range []string{"id=\"add-route\"", "id=\"register-dialog\"", "tr.pinned", "tr.missed", "miss-mark", "status-badge.missed"} {
		if !strings.Contains(controlBody, want) {
			t.Fatalf("control page missing %q", want)
		}
	}
	rec = doRouterRequest(s, http.MethodDelete, "http://dev.localhost/router/routes/demo-app", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d", rec.Code)
	}
}

func TestKillRouteProcessByPort(t *testing.T) {
	s := NewServer()
	_, _ = s.Store.Register("demo", RegisterRequest{Port: 5173}, false)
	var killedPort int
	s.KillProcessByPort = func(port int) ([]int, error) {
		killedPort = port
		return []int{1234}, nil
	}
	rec := doRouterRequest(s, http.MethodPost, "http://dev.localhost/router/routes/demo/kill", nil)
	if rec.Code != http.StatusOK || killedPort != 5173 || !strings.Contains(rec.Body.String(), "1234") {
		t.Fatalf("kill status=%d port=%d body=%s", rec.Code, killedPort, rec.Body.String())
	}
}

func TestDefaultHeartbeatIntervalIsTenSeconds(t *testing.T) {
	s := NewServer()
	if s.HeartbeatInterval != 10*time.Second {
		t.Fatalf("heartbeat interval = %s", s.HeartbeatInterval)
	}
}

func TestMissingAndUnavailablePages(t *testing.T) {
	s := NewServer()
	rec := doRouterRequest(s, http.MethodGet, "http://missing.localhost/", nil)
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "dev.localhost") {
		t.Fatalf("missing status/body: %d %s", rec.Code, rec.Body.String())
	}
	_, deadPort := unusedLocalPort(t)
	_, _ = s.Store.Register("demo", RegisterRequest{Port: deadPort, Pinned: true}, false)
	rec = doRouterRequest(s, http.MethodGet, "http://demo.localhost/", nil)
	if rec.Code != 523 || !strings.Contains(rec.Body.String(), "dev.localhost") {
		t.Fatalf("unavailable status/body: %d %s", rec.Code, rec.Body.String())
	}
}

func TestProxyPreservesRequestAndForwardedHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hello" || r.URL.RawQuery != "x=1" {
			t.Fatalf("path/query = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "payload" || r.Header.Get("X-Test") != "yes" {
			t.Fatalf("body/header mismatch")
		}
		if r.Header.Get("X-Forwarded-Host") != "demo.localhost" || r.Header.Get("X-Forwarded-Proto") != "http" || r.Header.Get("X-Forwarded-For") == "" {
			t.Fatalf("forwarded headers missing: %+v", r.Header)
		}
		w.Header().Set("X-Upstream", "ok")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(r.Host))
	}))
	defer upstream.Close()
	host, port := hostPort(t, upstream.URL)
	s := NewServer()
	_, err := s.Store.Register("demo", RegisterRequest{Port: port, TargetHost: host}, false)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://demo.localhost/hello?x=1", bytes.NewBufferString("payload"))
	req.RemoteAddr = "127.0.0.1:7777"
	req.Header.Set("X-Test", "yes")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted || rec.Header().Get("X-Upstream") != "ok" || strings.TrimSpace(rec.Body.String()) != host {
		t.Fatalf("proxy response code=%d headers=%v body=%q", rec.Code, rec.Header(), rec.Body.String())
	}
}

func TestHeartbeatHeadFallbackAndRemoval(t *testing.T) {
	hits := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/health" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		hits++
		w.WriteHeader(http.StatusNoContent)
	}))
	host, port := hostPort(t, upstream.URL)
	s := NewServer()
	_, _ = s.Store.Register("demo", RegisterRequest{Port: port, TargetHost: host, HeartbeatPath: "/health"}, false)
	s.CheckAllHeartbeats(context.Background())
	if hits != 1 || s.Store.List()[0].Misses != 0 {
		t.Fatalf("fallback GET did not hit")
	}
	upstream.Close()
	for range 5 {
		s.CheckAllHeartbeats(context.Background())
	}
	if len(s.Store.List()) != 0 {
		t.Fatal("dead non-pinned route should be removed")
	}
}

func TestRejectsNonLocalRemoteAddr(t *testing.T) {
	s := NewServer()
	req := httptest.NewRequest(http.MethodGet, "http://dev.localhost/", nil)
	req.RemoteAddr = "8.8.8.8:1234"
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d", rec.Code)
	}
}

func doRouterRequest(s *Server, method, target string, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, body)
	req.RemoteAddr = "127.0.0.1:1234"
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	return rec
}

func hostPort(t *testing.T, rawURL string) (string, int) {
	t.Helper()
	trimmed := strings.TrimPrefix(rawURL, "http://")
	host, portString, err := net.SplitHostPort(trimmed)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		t.Fatal(err)
	}
	return host, port
}

func unusedLocalPort(t *testing.T) (string, int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	host, portString, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		_ = ln.Close()
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		_ = ln.Close()
		t.Fatal(err)
	}
	_ = ln.Close()
	return host, port
}
