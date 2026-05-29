package router

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/log"
)

func TestAPIMutationsLogRouteEvents(t *testing.T) {
	var buf bytes.Buffer
	oldLogger := log.Default()
	log.SetDefault(log.New(&buf))
	defer log.SetDefault(oldLogger)

	s := NewServer()
	rec := doRouterRequest(s, "PUT", "http://dev.localhost/router/routes/demo", strings.NewReader(`{"port":5173}`))
	if rec.Code != 200 {
		t.Fatalf("register status=%d body=%s", rec.Code, rec.Body.String())
	}
	rec = doRouterRequest(s, "PATCH", "http://dev.localhost/router/routes/demo", strings.NewReader(`{"pinned":true}`))
	if rec.Code != 200 {
		t.Fatalf("pin status=%d body=%s", rec.Code, rec.Body.String())
	}
	rec = doRouterRequest(s, "PATCH", "http://dev.localhost/router/routes/demo", strings.NewReader(`{"pinned":false}`))
	if rec.Code != 200 {
		t.Fatalf("unpin status=%d body=%s", rec.Code, rec.Body.String())
	}
	rec = doRouterRequest(s, "DELETE", "http://dev.localhost/router/routes/demo", nil)
	if rec.Code != 204 {
		t.Fatalf("delete status=%d body=%s", rec.Code, rec.Body.String())
	}

	logs := buf.String()
	for _, want := range []string{"route registered", "route pinned", "route unpinned", "route unregistered"} {
		if !strings.Contains(logs, want) {
			t.Fatalf("logs missing %q in:\n%s", want, logs)
		}
	}
}
