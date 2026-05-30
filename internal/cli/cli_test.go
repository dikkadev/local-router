package cli

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"forge.dikka.dev/lab/local-router/internal/router"
)

func TestRegisterPrintsOnlyURL(t *testing.T) {
	s := router.NewServer()
	server := httptest.NewServer(s)
	defer server.Close()
	var stdout, stderr bytes.Buffer
	code := Run([]string{"register", "Demo", "--port", "5173", "--title", "Demo app", "--pinned"}, Config{BaseURL: server.URL, Stdout: &stdout, Stderr: &stderr})
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if stdout.String() != "http://demo.localhost\n" {
		t.Fatalf("stdout=%q", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestRouterNotRunningError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	var stdout, stderr bytes.Buffer
	code := Run([]string{"status"}, Config{BaseURL: "http://" + addr, Stdout: &stdout, Stderr: &stderr})
	if code == 0 {
		t.Fatal("expected failure")
	}
	if !strings.Contains(stderr.String(), "router service is not running") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestRoutesPinUnpinAndUnregister(t *testing.T) {
	s := router.NewServer()
	server := httptest.NewServer(s)
	defer server.Close()
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"register", "demo", "--port", "5173"}, Config{BaseURL: server.URL, Stdout: &stdout, Stderr: &stderr}); code != 0 {
		t.Fatalf("register failed: %s", stderr.String())
	}
	stdout.Reset()
	if code := Run([]string{"pin", "demo"}, Config{BaseURL: server.URL, Stdout: &stdout, Stderr: &stderr}); code != 0 || !strings.Contains(stdout.String(), "pinned=true") {
		t.Fatalf("pin code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	if code := Run([]string{"routes"}, Config{BaseURL: server.URL, Stdout: &stdout, Stderr: &stderr}); code != 0 || !strings.Contains(stdout.String(), "demo") {
		t.Fatalf("routes code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	if code := Run([]string{"unpin", "demo"}, Config{BaseURL: server.URL, Stdout: &stdout, Stderr: &stderr}); code != 0 || !strings.Contains(stdout.String(), "pinned=false") {
		t.Fatalf("unpin code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	if code := Run([]string{"unregister", "demo"}, Config{BaseURL: server.URL, Stdout: &stdout, Stderr: &stderr}); code != 0 || !strings.Contains(stdout.String(), "unregistered demo") {
		t.Fatalf("unregister code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestHelpCommands(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--help"}, Config{Stdout: &stdout, Stderr: &stderr})
	if code != 0 || !strings.Contains(stdout.String(), "SERVICE COMMAND") || !strings.Contains(stdout.String(), `sudo env "PATH=$PATH" go run`) {
		t.Fatalf("global help code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	code = Run([]string{"register", "--help"}, Config{Stdout: &stdout, Stderr: &stderr})
	if code != 0 || !strings.Contains(stdout.String(), "--heartbeat-path") {
		t.Fatalf("register help code=%d stdout=%q", code, stdout.String())
	}
}

func TestAPIClientReportsServerErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"boom"}`, http.StatusConflict)
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	code := Run([]string{"register", "demo", "--port", "5173"}, Config{BaseURL: server.URL, Stdout: &stdout, Stderr: &stderr})
	if code == 0 || stderr.String() == "" {
		t.Fatalf("expected error")
	}
}
