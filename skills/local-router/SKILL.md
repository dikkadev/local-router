---
name: local-router
description: Use the local-router CLI to manage friendly .localhost URLs for already-running local development servers. Use when the user asks about local-router, demo/dev/router.localhost URLs, registering or unregistering local ports, starting the router service, or troubleshooting local-router route status.
---

# Local Router CLI

## Scope

Use `local-router` for local loopback development routing: mapping an already-running local HTTP server on a port to a friendly URL like `http://demo.localhost`.

Do not treat it as public deployment, DNS management, Windows hosts-file management, Caddy/nginx configuration, or a tool that starts the target app server. The target server must already be running separately.

## Source of truth

Prefer installed CLI help and repo docs over memory:

```bash
local-router --help
local-router serve --help
local-router register --help
```

If working inside the source repo before installation, use `go run ./cmd/local-router ...`. The longer behavioral reference is `docs/spec.md`; service setup notes are in `docs/systemd.md`.

## Light preflight

Start with the least invasive checks:

```bash
command -v local-router
local-router status
local-router routes
```

If the binary is not installed but the current directory is the repo, use `go run ./cmd/local-router ...` for read-only checks. If `status` says the service is not running, explain that the router service must be started before routes can be registered.

## Normal workflow

1. Confirm or ask which already-running target port should be routed.
2. Check `local-router routes` when conflicts matter.
3. Register the route:
   ```bash
   local-router register demo --port 5173 --title "Demo app"
   ```
4. Report the printed route URL plus the dashboard URL `http://dev.localhost`.

Useful commands:

```bash
local-router status
local-router routes
local-router pin demo
local-router unpin demo
local-router unregister demo
```

Use `--force` only when replacing an existing route is clearly intended or after explaining the conflict.

## Starting the service

The service binds `127.0.0.1:80` so nice URLs do not need a port. On Linux/WSL that normally requires elevated privileges, a systemd unit with `CAP_NET_BIND_SERVICE`, or a binary granted that capability.

Source-repo quick test:

```bash
sudo env "PATH=$PATH" go run ./cmd/local-router serve
```

Installed binary quick test:

```bash
sudo local-router serve
```

Preferred persistent setup on systemd systems is documented in `docs/systemd.md`.

## Safety and mutations

Registering a clearly requested route is fine. Ask or explain before:

- using `--force` to replace an existing route;
- unregistering a route when the target name is ambiguous;
- pinning/unpinning a route if the user did not ask for that behavior;
- changing service installation or systemd state.

This is local-only tooling, but service changes can affect port 80 and current browser workflows.

## Troubleshooting

- `router service is not running`: start `local-router serve` or the systemd service.
- bind failure on `127.0.0.1:80`: check port 80 conflicts or bind permission.
- `sudo` cannot find Go: use `sudo env "PATH=$PATH" go run ./cmd/local-router serve` or install the binary.
- Browser `.localhost` works but CLI DNS does not: the CLI talks to `http://127.0.0.1` and sends `Host: dev.localhost`; do not require CLI DNS resolution.
- route URL is unavailable: the route exists, but the target server is not reachable on its registered host/port.

## Current limitations

Routes are in-memory and do not survive router service restarts. The router does not start, stop, or supervise target app servers.
