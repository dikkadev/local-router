# Local Router Specification

## Purpose

Build a local development router that gives ad hoc local web servers stable, meaningful `.localhost` URLs without requiring the user to remember ports, manually avoid port collisions, edit hosts files, or hand-edit/reload reverse-proxy config.

The router does not start or own the web servers behind those URLs. A separate process starts whatever local server it wants, then registers a route by giving the router a required name and local target port, plus optional display metadata.

## Problem statement

Local development servers are usually opened through raw ports:

```text
http://localhost:5173
```

That is inconvenient because:

- familiar ports collide;
- random/high ports are hard to remember;
- the URL says nothing about what the server is;
- switching between multiple temporary servers becomes messy.

The desired experience is a stable, meaningful URL:

```text
http://some-thing.localhost
```

without per-run Windows hosts-file edits and without hand-editing/reloading Caddy/nginx config.

## Non-goals for v1

- General-purpose replacement for Caddy, nginx, or Traefik.
- Required custom DNS setup.
- Required Windows hosts-file mutation.
- Starting, supervising, or owning the target web servers.
- Internet/LAN exposure; this is loopback-only.
- HTTPS as a v1 requirement, though it should remain possible later.

## Key concepts

### Router

A small persistent local service that:

- Listens on loopback HTTP port 80.
- Routes requests by `Host` header.
- Reverse-proxies registered hosts to local target ports.
- Exposes a small REST registration API.
- Can be controlled by a CLI, but is not specific to CLI callers.
- Removes or marks stale routes when registered targets disappear or stop heartbeating.

### Registered route

A route maps a required name to a local target port.

Required registration values:

- `name`: the subdomain label, such as `demo` for `demo.localhost`.
- `port`: the local target port to proxy to.

Optional registration values:

- `title`: a slightly longer display title.
- `targetHost`: defaults to `127.0.0.1`.
- `pinned`: whether the route should remain reserved even when no target is currently responding.
- process metadata such as PID/CWD/command, if a caller wants to provide it.
- heartbeat/TTL settings, if the route should expire automatically.

The router does not choose generated names. Callers must provide the route name they want.

### Target server

A target server is any local HTTP server that another tool or user starts separately. It should bind to loopback. It may register/unregister itself through the API or rely on a wrapper script/CLI to do that.

## Default URL model

Use `.localhost` by default:

```text
http://local-router.localhost  router dashboard/control page
http://demo.localhost          route named demo
http://plot.localhost          route named plot
```

Rationale:

- `.localhost` is reserved for local/loopback use.
- It avoids hosts-file edits for common browser behavior.
- It is more predictable than `.local`, which is associated with mDNS/Bonjour and LAN discovery.
- Custom short domains are possible later but require resolver/DNS/hosts setup beyond the reverse proxy itself.

## Optional naming extensions

Future optional install steps may support:

- One fixed hosts alias for the dashboard.
- A local wildcard DNS resolver for suffixes like `*.x` or `*.art`.

These should not be required for v1 because a normal hosts file generally cannot express wildcard subdomains.

## Expected CLI UX

The CLI registers, lists, updates, and removes routes. It does not run the target server.

Example registration:

```bash
local-router register demo --port 5173 --title "Demo app"
```

Minimal successful output:

```text
http://demo.localhost
```

The first line should be the URL so it is easy to copy, pipe, parse, or open. After that, commands may emit normal conservative logs for important events, warnings, and errors. They should not print noisy per-request access logs by default, especially not successful `200` requests.

Human-readable logs should use `charmbracelet/log` with its default nice/colorful terminal output. Commands should also expose a `--json` option for machine-readable log output.

If the router is not running, the CLI should fail clearly and say that the router service is not running.

## Dashboard requirements

If included, the router dashboard should be small and focused. It should show active and pinned routes and make it easy to open/copy route URLs.

Suggested columns:

- Name
- URL
- Title, if provided
- Status: live, unavailable, stale, pinned
- Target host/port
- Last heartbeat, if applicable
- PID/CWD/command, if provided

Suggested actions:

- Open route
- Copy URL
- Unregister route
- Pin/unpin route

Pinned routes stay reserved even if nothing is currently backing the target port. When a pinned route has no reachable target, the router should serve its own helpful error page for that hostname instead of treating the name as unregistered.

If live dashboard updates are implemented, Server-Sent Events are a reasonable fit for route table changes:

```text
GET /_router/events
```

## Registration API draft

Routes are managed through reserved router paths, not proxied target paths.

### Register or replace route

```http
PUT /_router/routes/{name}
Content-Type: application/json

{
  "port": 5173,
  "title": "Demo app",
  "targetHost": "127.0.0.1",
  "pinned": false,
  "pid": 12345,
  "cwd": "/home/me/project",
  "startedBy": "npm run dev",
  "ttlSeconds": 10
}
```

The route host is derived from the route name:

```text
name = demo
host = demo.localhost
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
{ "type": "route_unavailable", "name": "demo" }
{ "type": "route_removed", "name": "demo" }
```

## Routing behavior

For normal browser requests:

1. Strip any port from `req.Host`.
2. If host is the router dashboard/control host, serve the router UI/API.
3. Else look up host in the route table.
4. If the route exists and its target is reachable, reverse proxy to the target.
5. If the route exists but no target is reachable, serve a helpful route-unavailable page.
6. If the route is missing, return a helpful 404 page explaining that no route is registered for the host.

Reverse proxy should support:

- Normal HTTP methods.
- Streaming responses without unwanted buffering.
- WebSocket upgrades.
- SSE passthrough.

## Lifecycle and cleanup

Use layered cleanup:

- Callers may unregister routes on normal exit.
- Callers may heartbeat periodically while alive.
- Non-pinned routes with a heartbeat TTL may expire when heartbeats stop.
- The router may health-check targets and mark them unavailable.
- Pinned routes remain registered even when unavailable.

Stale non-pinned entries should not permanently block names.

## Naming behavior

A route name is required. The host is derived from it:

```text
name = demo
host = demo.localhost
```

Collision behavior:

- Registering an already-live route should fail clearly unless replacement is explicitly requested.
- Replacing a pinned route should require explicit replacement.
- Stale/dead non-pinned routes may be replaceable after verification or TTL expiry.

## Security requirements

- Bind router and targets to loopback only by default.
- Do not listen on `0.0.0.0` unless explicitly requested.
- Do not expose the admin API beyond loopback.
- Treat route registration as local-only control plane access.
- Avoid serving arbitrary files from the router itself.
- Be careful with future log capture because logs may contain secrets.

## WSL / Windows model

Primary user environment:

- The browser runs on Windows.
- Many target servers and CLI commands run inside WSL.
- Windows commonly reaches WSL servers through `localhost:<port>`.

The router service and CLI do not have to be the same binary for the same OS target.

Expected split:

- A Windows router service can own the stable browser-facing port 80 and behave like a normal Windows background service.
- Once the router behavior is working, the Windows-side server should be installed/run as a service, for example through NSSM.
- A WSL CLI can talk to that router service over localhost and register WSL-hosted target ports.
- Shared code can still live in one Go codebase where practical.

This keeps the router close to the browser-facing side while allowing WSL tools to register routes without owning the router process.

## Port 80 behavior

Nice URLs without `:port` are a hard requirement. The router should listen on HTTP port 80 on loopback.

If the router service cannot bind port 80, setup/startup should fail clearly and explain the conflict or permission problem. Falling back to URLs with explicit ports is not a v1 feature.

## Command shape

The CLI controls and registers with the router service. It does not start target servers.

Example commands:

```bash
local-router status
local-router register demo --port 5173 --title "Demo app"
local-router unregister demo
local-router routes
local-router pin demo --port 5173 --title "Demo app"
local-router unpin demo
```

The router service may be a separate executable or installed service wrapper, especially on Windows. NSSM is a likely service wrapper once the router server is ready to run persistently. The CLI should just interact with the service API like any other client.

## Implementation notes for Go

Core reverse proxy can be built with:

```go
net/http
net/http/httputil
net/url
sync
```

Logging should use `github.com/charmbracelet/log` for human-readable output, with a `--json` option for structured output.

Route table shape:

```go
type Route struct {
    Name       string
    Host       string
    TargetHost string
    Port       int
    Title      string
    Pinned     bool
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

1. Should the dashboard be included in v1, or should v1 only expose CLI/API route inspection?
2. Should NSSM be the Windows service wrapper/install approach for the router service, or is there a better fit?
3. Should route definitions persist across router restart, and if so should only pinned routes persist?

## Proposed v1 scope

- Project/tool name: `local-router`.
- Router service listening on loopback port 80.
- CLI that talks to the router service API.
- In-memory route registry.
- Required route registration by name and port.
- Optional title and pinned route metadata.
- Register, heartbeat, unregister, list API.
- Host-header reverse proxy.
- Helpful unavailable page for pinned or registered routes with no reachable target.
- Basic stale route cleanup for non-pinned routes.
- Clear setup/startup error when port 80 cannot be used.
- Conservative human-readable logging through `charmbracelet/log`, plus a `--json` mode.
- Documentation for Windows browser, WSL CLI usage, and Windows service setup such as NSSM.

No actual implementation has been started yet.
