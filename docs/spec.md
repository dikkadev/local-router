# Local Router Specification

## Purpose

Build a Go-based local development tool that makes short-lived local web servers easy to open from a Windows browser while the actual server runs ad hoc, likely from WSL, with logs/stdout still attached to the command that started it.

The tool should avoid requiring the user to remember ports, manually pick non-conflicting ports, or edit proxy configuration for each run.

## Problem statement

A command starts a temporary web server on WSL. The user accesses it from a Windows browser. Raw ports are inconvenient and unclear:

```text
http://localhost:5173
```

They also create a choice between living with port collisions on familiar ports or using random/high ports that are hard to remember and identify.

The desired experience is a stable, meaningful URL:

```text
http://some-thing.localhost
```

without per-run Windows hosts-file edits and without hand-editing/reloading Caddy/nginx config.

## Non-goals for v1

- General-purpose replacement for Caddy, nginx, or Traefik.
- Required custom DNS setup.
- Required Windows hosts-file mutation.
- Internet/LAN exposure; this is loopback-only.
- HTTPS as a v1 requirement, though it should remain possible later.

## Key concepts

### Router

A small persistent local daemon that:

- Listens on loopback HTTP, ideally `127.0.0.1:80`.
- Routes requests by `Host` header.
- Reverse-proxies registered hosts to target ports.
- Serves its own dashboard/control plane.
- Exposes a small registration API.
- Removes stale routes when registered targets die or stop heartbeating.

### Routed server

An ephemeral local web server. It should:

- Bind to loopback, usually `127.0.0.1:0`, letting the OS choose a free port.
- Register a friendly hostname with the router.
- Print/open the stable URL.
- Keep stdout/log output attached to the command that started it.
- Unregister on normal exit.
- Heartbeat while alive so crashes are cleaned up.

## Default URL model

Use `.localhost` by default:

```text
http://dev.localhost       router dashboard
http://demo.localhost      routed server named demo
http://plot.localhost      routed server named plot
http://run-8f3a.localhost  generated route name
```

Rationale:

- `.localhost` is reserved for local/loopback use.
- It avoids hosts-file edits for common browser behavior.
- It is more predictable than `.local`, which is associated with mDNS/Bonjour and LAN discovery.
- Custom short domains are possible later but require resolver/DNS/hosts setup beyond the reverse proxy itself.

## Optional naming extensions

Future optional install steps may support:

- One fixed hosts alias for the dashboard, e.g. `http://art`.
- A local wildcard DNS resolver for suffixes like `*.x` or `*.art`.

These should not be required for v1 because a normal hosts file generally cannot express wildcard subdomains.

## Expected UX

Example serve command:

```bash
local-router serve ./demo --name demo
```

Output:

```text
Serving:
  http://demo.localhost

Dashboard:
  http://dev.localhost

Logs:
  GET /                    200
  GET /assets/app.js       200
```

If the router is not running, `serve` should ideally try to start it or give a clear instruction.

If port 80 is unavailable or requires privileges, the tool may fall back to a configured non-privileged port such as `7777`, producing URLs like:

```text
http://demo.localhost:7777
```

## Dashboard requirements

The router should serve a small dashboard at `dev.localhost` showing active and recently stale routes.

Suggested columns:

- Name
- URL
- Status: live, unhealthy, stale, removing
- Age / started time
- Last heartbeat
- Target URL
- PID, if provided
- CWD / command, if provided

Suggested actions:

- Open route
- Copy URL
- Unregister/stop route
- Rename route, optional later
- Pin route, optional later
- View logs, optional later if logs are exposed by the routed server or captured intentionally

The dashboard should live-update. Prefer Server-Sent Events for route table changes:

```text
GET /_router/events
```

Use WebSockets only if later bidirectional terminal-like interaction is needed.

## Registration API draft

Routes are managed through reserved router paths, not proxied target paths.

### Register or replace route

```http
PUT /_router/routes/{name}
Content-Type: application/json

{
  "host": "demo.localhost",
  "target": "http://127.0.0.1:49173",
  "title": "Demo",
  "pid": 12345,
  "cwd": "/home/me/project",
  "startedBy": "local-router serve ./demo --name demo",
  "ttlSeconds": 10
}
```

### Heartbeat

```http
POST /_router/routes/{name}/heartbeat
```

### Unregister

```http
DELETE /_router/routes/{name}
```

### List routes

```http
GET /_router/routes
```

### Event stream

```http
GET /_router/events
```

Example events:

```json
{ "type": "route_added", "name": "demo" }
{ "type": "route_changed", "name": "demo" }
{ "type": "route_unhealthy", "name": "demo" }
{ "type": "route_removed", "name": "demo" }
```

## Routing behavior

For normal browser requests:

1. Strip any port from `req.Host`.
2. If host is the router dashboard host, serve the dashboard/API.
3. Else look up host in the route table.
4. If found, reverse proxy to its target.
5. If missing, return a helpful 404 page explaining that no route is registered for the host and link to the dashboard.

Reverse proxy should support:

- Normal HTTP methods.
- Streaming responses without unwanted buffering.
- WebSocket upgrades for interactive routes.
- SSE passthrough.

## Lifecycle and cleanup

Use layered cleanup:

- Registering process unregisters route on normal exit.
- Registering process heartbeats periodically while alive.
- Router expires routes whose heartbeat TTL elapses.
- Router may periodically health-check targets and mark/remove dead routes.

Stale entries should not permanently block names.

## Naming behavior

A route has both a short `name` and a full `host`.

Default host derivation:

```text
name = demo
host = demo.localhost
```

If a generated name is needed:

```text
run-8f3a.localhost
```

Suggested collision behavior:

- Explicit `--name demo`: fail clearly if live route exists unless `--replace` is passed.
- Generated/default names: auto-suffix or generate a fresh name.
- Stale/dead route: allow replacement after verification or TTL expiry.

## Security requirements

- Bind router and targets to loopback only by default.
- Do not listen on `0.0.0.0` unless explicitly requested.
- Do not expose the admin API beyond loopback.
- Treat route registration as local-only control plane access.
- Avoid serving arbitrary files from the router itself.
- Be careful with future log capture because logs may contain secrets.

## WSL / Windows considerations

Primary user environment:

- Tool runs in WSL.
- Browser runs on Windows.
- Windows commonly reaches WSL servers through `localhost:<port>`.

Open implementation question:

- Should the router run in WSL for simplicity, or on Windows for more robust Windows-facing behavior?

Initial recommendation:

- Implement and test WSL-side router first because it is easiest for a Go CLI used from WSL.
- Keep the design portable enough for a future Windows-side router binary or service.

## Port 80 behavior

Nice URLs without `:port` require the router to listen on HTTP port 80.

v1 should handle this explicitly:

- Try configured router address, default `127.0.0.1:80`.
- If binding fails, report the reason clearly.
- Optional fallback to `127.0.0.1:7777` for development mode.
- Document that port 80 may need privileges or may conflict with IIS, Docker, other dev tools, etc.

## Possible command shape

One binary can have multiple modes:

```bash
local-router router start
local-router router status
local-router router stop
local-router router routes
local-router serve ./demo --name demo
```

Alternative binary split:

```bash
local-router-daemon
local-router serve ./demo
```

The one-binary mode is likely simpler for installation and discovery.

## Implementation notes for Go

Core reverse proxy can be built with:

```go
net/http
net/http/httputil
net/url
sync
```

Route table shape:

```go
type Route struct {
    Name       string
    Host       string
    Target     *url.URL
    Title      string
    PID        int
    CWD        string
    StartedBy  string
    CreatedAt  time.Time
    LastSeenAt time.Time
    TTL        time.Duration
}
```

Router lookup should be protected by a mutex or other concurrency-safe structure.

## Open questions

1. Project/tool name: should the command be `local-router`, `router`, or something else?
2. Should v1 include the server-starting command itself, or only the reusable router and registration client?
3. Should the router auto-start from `serve`, and if so how should it daemonize under WSL?
4. What should happen when port 80 is unavailable: hard fail, fallback port, or guided install?
5. Should the dashboard be plain server-rendered HTML first, or a small bundled frontend?
6. Should route definitions persist across router restart, or should all routes be ephemeral only?
7. Is Windows-side router support required before implementation, or can it be a later compatibility target?

## Proposed v1 scope

- Go module and CLI skeleton.
- Router process listening on loopback.
- In-memory route registry.
- Register, heartbeat, unregister, list API.
- Host-header reverse proxy.
- Dashboard host at `dev.localhost`.
- SSE events for dashboard live updates.
- Basic stale route cleanup.
- Clear port 80/fallback behavior.
- Documentation for WSL/Windows browser usage.

No actual implementation has been started yet.
