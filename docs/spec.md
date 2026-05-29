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
- Persistent route storage across router restarts.

## Key concepts

### Router

A small persistent local service that:

- Listens on loopback HTTP port 80.
- Routes requests by `Host` header.
- Reverse-proxies registered hosts to local target ports.
- Exposes a small REST registration API.
- Serves a very small HTML route table at the reserved control hosts.
- Can be controlled by a CLI, but is not specific to CLI callers.
- Marks or removes stale non-pinned routes based on heartbeat misses.

### Registered route

A route maps a required name to a local target port.

Required registration values:

- `name`: the subdomain label, such as `demo` for `demo.localhost`.
- `port`: the local target port to proxy to.

Optional registration values:

- `title`: a slightly longer display title.
- `targetHost`: defaults to `127.0.0.1`.
- `pinned`: whether the route should remain reserved while the router is running, even when no target is currently responding.
- `exec`: small display/debug metadata describing what started or owns the target.
- `heartbeatPath`: target path used for health checks; defaults to `/`.

The router does not choose generated names. Callers must provide the route name they want.

### Target server

A target server is any local HTTP server that another tool or user starts separately. It should bind to loopback. It may register/unregister itself through the API or rely on a wrapper script/CLI to do that.

## Default URL model

Use `.localhost` by default:

```text
http://dev.localhost     primary router control page
http://router.localhost  secondary router control page
http://demo.localhost    route named demo
http://plot.localhost    route named plot
```

Rationale:

- `.localhost` is reserved for local/loopback use.
- It avoids hosts-file edits for common browser behavior.
- It is more predictable than `.local`, which is associated with mDNS/Bonjour and LAN discovery.
- Custom short domains are possible later but require resolver/DNS/hosts setup beyond the reverse proxy itself.

Reserved route names:

- `dev`
- `router`

These names cannot be registered as normal routes.

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

For successful registration, stdout should print only the final URL. Other commands should produce useful but conservative output and should not spam. Proxied request logs should not be printed by default, especially not successful `200` requests.

Human-readable logs should use `charmbracelet/log` with its default nice/colorful terminal output. There is no CLI `--json` requirement for v1.

If the router is not running, the CLI should fail clearly and say that the router service is not running.

Relevant command shape:

```bash
local-router status
local-router register demo --port 5173 --title "Demo app"
local-router register demo --port 5173 --title "Demo app" --pinned
local-router register demo --port 3000 --force
local-router unregister demo
local-router routes
local-router pin demo
local-router unpin demo
```

`--force` is the generic override flag for replacing/updating an existing route when the operation would otherwise conflict.

## Control page requirements

The router should serve a very simple pure HTML control page at `dev.localhost`, also available at `router.localhost`. `dev.localhost` is the primary name.

The page should show a table of current registrations. No styling framework is needed; plain HTML is enough. Use a small amount of JavaScript to periodically refresh the table in the background with clean updates and no visible page flicker.

Suggested columns:

- Name
- URL
- Title, if provided
- Status: live, unavailable, stale, pinned
- Target host/port
- Heartbeat path
- Miss count
- Exec metadata, if provided

Suggested actions, if easy and still minimal:

- Open route
- Copy URL
- Unregister route
- Pin/unpin route

Pinned routes stay reserved for the lifetime of the router process even if nothing is currently backing the target port. When a pinned route has no reachable target, the router should serve its own helpful unavailable page for that hostname instead of treating the name as unregistered. If something later starts listening on that port, the route should work on the next browser request/reload.

## Registration API draft

Routes are managed through reserved router paths, not proxied target paths. Use paths under `/router/...`; do not use an underscore-prefixed path.

The API is exposed on the control hosts:

```text
http://dev.localhost/router/...
http://router.localhost/router/...
```

### Register route

```http
PUT /router/routes/{name}
Content-Type: application/json

{
  "port": 5173,
  "title": "Demo app",
  "targetHost": "127.0.0.1",
  "pinned": false,
  "exec": "npm run dev",
  "heartbeatPath": "/"
}
```

The route host is derived from the normalized route name:

```text
name = demo
host = demo.localhost
```

If a route already exists, registration fails with `409 Conflict` unless the request uses the generic force/override mechanism. For the REST API, use:

```http
PUT /router/routes/{name}?force=true
```

Pinned routes also require `force=true` to replace/update.

### Heartbeat / health check target

The route registration may include a custom `heartbeatPath`. If omitted or empty, `/` is used.

The router checks the target with a lightweight request to that path. `HEAD` is preferred where practical; if a target does not handle `HEAD` usefully, the router may fall back to `GET`.

Any `2xx` response counts as a hit. Anything else, including timeout or connection failure, counts as a miss.

Default interval: 30 seconds.

Default state transitions:

- after 3 consecutive misses: route becomes unavailable;
- after 5 consecutive misses: non-pinned route is removed;
- pinned routes are never removed by heartbeat misses.

There is no separate health-check system beyond this heartbeat/health path behavior.

### Unregister

```http
DELETE /router/routes/{name}
```

### List routes

```http
GET /router/routes
```

### Control page data

The control page should periodically refresh route data in the background. This can use the route list endpoint directly; no event stream is required for v1.

## Route name normalization

Route names are lowercase labels using `-` as the delimiter.

The router/CLI should massage practical user input into a useful route name where possible:

- convert to lowercase;
- replace common separators such as spaces, underscores, and commas with `-`;
- collapse repeated dashes;
- trim leading/trailing dashes;
- replace remaining invalid symbols with `0`.

If normalization cannot produce a useful valid name, reject the input with a clear error.

This normalization behavior is implementation behavior, not user-facing conceptual complexity. The regular user-facing explanation should simply say that route names are lowercase words separated by dashes.

Final route names must be valid single `.localhost` labels: lowercase letters, numbers, and hyphens only; no dots; no empty name; no reserved control name.

## Routing behavior

For normal browser requests:

1. Strip any port from `req.Host`.
2. If host is `dev.localhost` or `router.localhost`, serve the control page/API.
3. Else look up host in the route table.
4. If the route exists, attempt to reverse proxy to its target.
5. If proxying fails because no target is reachable, serve a helpful route-unavailable page.
6. If the route is missing, return a helpful 404 page explaining that no route is registered for the host.

Unavailable and missing responses:

- unregistered host: `404 Not Found`;
- registered but unavailable host, including pinned routes with no backing target: `523 Origin Is Unreachable`;
- both pages should mention `dev.localhost` as the place to inspect current registrations.

Reverse proxy should support:

- Normal HTTP methods.
- Streaming responses without unwanted buffering.
- WebSocket upgrades.
- SSE passthrough.
- Path and query preservation.

Proxy behavior:

- Rewrite upstream `Host` to the target host.
- Set `X-Forwarded-Host` to the original requested host.
- Set `X-Forwarded-Proto: http`.
- Set or append `X-Forwarded-For`.
- Pass ordinary request/response headers through normally.
- Use a 2 minute timeout.
- Do not impose a small application-level body limit; this is local tooling, so the practical limit should be large/default rather than restrictive.

## Lifecycle and cleanup

All route state is in memory. Nothing persists across router restart, including pinned routes.

Use layered cleanup:

- Callers may unregister routes on normal exit.
- The router checks each target's heartbeat path every 30 seconds.
- Non-pinned routes become unavailable after 3 misses and are removed after 5 misses.
- Pinned routes become unavailable after misses but remain registered for the lifetime of the router process.
- Each browser request still attempts to proxy a registered route, so a previously unavailable route can recover as soon as the target is reachable again.

Stale non-pinned entries should not permanently block names.

## Naming behavior

A route name is required. The host is derived from its normalized form:

```text
name = demo
host = demo.localhost
```

Collision behavior:

- Registering an existing route fails with `409 Conflict` unless force is explicitly requested.
- Replacing or updating a pinned route also requires force.
- Stale/dead non-pinned routes may be replaced with force or after removal.

## Security requirements

- Bind router and targets to loopback only by default.
- Do not listen on `0.0.0.0` unless explicitly requested.
- Do not expose the admin API beyond local access.
- Reject requests that do not come from broadly local/Docker-style local development addresses; the exact allowlist can be refined during implementation.
- Do not add token/auth complexity for v1.
- Keep CORS restrictive/off by default; avoid browser-origin exposure rather than adding remote access conveniences.
- Treat route registration as local-only control plane access.
- Avoid serving arbitrary files from the router itself.

## Linux / WSL model

Primary user environment:

- The router service runs on Linux/WSL and owns loopback port 80.
- Target servers and CLI commands usually run in the same Linux/WSL environment.
- The browser may run on Windows and reach WSL loopback services through `localhost` / `.localhost` behavior when available.

Expected split:

- The router service is a long-running Linux/WSL process.
- Later, setup should support installing/running it as a Linux/WSL service, for example through a systemd user/system unit where available.
- The CLI talks to that router service over localhost and registers target ports.

No special target-address translation is required for v1; this is expected to work like same-host localhost development.

## Port 80 behavior

Nice URLs without `:port` are a hard requirement. The router should listen on HTTP port 80 on loopback.

If the router service cannot bind port 80, setup/startup should fail clearly and explain the conflict or permission problem. Falling back to URLs with explicit ports is not a v1 feature.

## Command shape

The CLI controls and registers with the router service. It does not start target servers.

Example commands:

```bash
local-router status
local-router register demo --port 5173 --title "Demo app"
local-router register demo --port 5173 --title "Demo app" --pinned
local-router register demo --port 3000 --force
local-router unregister demo
local-router routes
local-router pin demo
local-router unpin demo
```

The router service may be a separate long-running process or installed Linux/WSL service later, but v1 does not need to implement service installation. The CLI should just interact with the service API like any other client.

## Implementation notes for Go

Core reverse proxy can be built with:

```go
net/http
net/http/httputil
net/url
sync
```

Logging should use `github.com/charmbracelet/log` for conservative human-readable output.

Route table shape:

```go
type Route struct {
    Name          string
    Host          string
    TargetHost    string
    Port          int
    Title         string
    Pinned        bool
    Exec          string
    HeartbeatPath string
    CreatedAt     time.Time
    LastCheckAt   time.Time
    Misses        int
}
```

Router lookup should be protected by a mutex or other concurrency-safe structure.

## Proposed v1 scope

- Project/tool name: `local-router`.
- Router service listening on loopback port 80.
- CLI that talks to the router service API.
- In-memory route registry only; no route persistence across restart.
- Required route registration by name and port.
- Route name normalization to lowercase dash-delimited labels.
- Optional title, exec metadata, heartbeat path, and pinned route metadata.
- Register, unregister, and list API.
- `--force` override for replace/update conflicts.
- Host-header reverse proxy.
- `dev.localhost` primary control page and `router.localhost` secondary control page.
- Helpful unavailable page for pinned or registered routes with no reachable target.
- Heartbeat-path cleanup for non-pinned routes.
- Clear setup/startup error when port 80 cannot be used.
- Conservative human-readable logging through `charmbracelet/log`.
- Documentation for Linux/WSL usage and later Linux/WSL service setup.

Initial implementation lives under `cmd/local-router` and `internal/`; this spec remains the v1 behavioral reference.
