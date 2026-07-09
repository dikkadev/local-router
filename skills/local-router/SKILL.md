---
name: local-router
description: Use the local-router CLI to manage friendly .localhost URLs for already-running local development servers. Use when the user asks about local-router, demo/dev/router.localhost URLs, registering or unregistering local ports, starting the router service, or troubleshooting local-router route status.
---

# Local Router CLI

## Scope

Use `local-router` for local development routing: mapping an already-running local HTTP server on a port to a friendly URL like `http://demo.localhost`. It can optionally expose the same dashboard/routes through a configured tailnet/domain host such as `http://ppc.dikka.dev` and `http://demo.ppc.dikka.dev` when DNS and binding are set up separately.

Do not treat it as public deployment, DNS management, Windows hosts-file management, Caddy/nginx configuration, or a tool that starts the target app server. The target server must already be running separately.

## Source of truth

Prefer installed CLI help and repo docs over memory:

```bash
local-router --help
local-router serve --help
local-router register --help
```

Install with `go install forge.dikka.dev/lab/local-router@latest`, or from a checkout with `go install .`. The longer behavioral reference is `docs/spec.md`; service setup notes are in `docs/systemd.md`.

## Light preflight

Start with the least invasive checks:

```bash
command -v local-router
local-router status
local-router routes
```

If the CLI is not installed but the current directory is the repo, prefer `go install .` before continuing. If `status` says the service is not running, explain that the router service must be started before routes can be registered.

## Normal workflow

1. Confirm or ask which already-running target port should be routed.
2. Check `local-router routes` when conflicts matter.
3. Register the route:
   ```bash
   local-router register demo --port 5173 --title "Demo app"
   ```
4. Report the printed route URL plus the dashboard URL `http://dev.localhost`. If the service is configured with `--external-host <host>`, also report `http://<host>` and `http://<route>.<host>` when useful.

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

The service binds `127.0.0.1:80` by default so nice URLs do not need a port. For tailnet/domain access, start it with `--addr 0.0.0.0:80 --external-host <host>` or set `LOCAL_ROUTER_ADDR` / `LOCAL_ROUTER_EXTERNAL_HOST` in the systemd service. On Linux/WSL, binding port 80 normally requires elevated privileges, a systemd unit with `CAP_NET_BIND_SERVICE`, or a binary granted that capability.

Foreground quick test after granting low-port bind capability:

```bash
sudo setcap 'cap_net_bind_service=+ep' "$(go env GOPATH)/bin/local-router"
local-router serve
```

Preferred persistent setup on systemd systems is documented in `docs/systemd.md`.

## Safety and mutations

Registering a clearly requested route is fine. Ask or explain before:

- using `--force` to replace an existing route;
- unregistering a route when the target name is ambiguous;
- pinning/unpinning a route if the user did not ask for that behavior;
- changing service installation or systemd state.

This is local-first tooling, but service changes can affect port 80, current browser workflows, and any tailnet/domain clients that can reach the bound address.

## Troubleshooting

- `router service is not running`: start `local-router serve` or the systemd service.
- bind failure on `127.0.0.1:80` or `0.0.0.0:80`: check port 80 conflicts or bind permission.
- `local-router` cannot be found: make sure Go's install directory is on `PATH`, or follow `docs/systemd.md` to copy the installed CLI to `/usr/local/bin/local-router` for service use.
- Browser `.localhost` works but CLI DNS does not: the CLI talks to `http://127.0.0.1` and sends `Host: dev.localhost`; do not require CLI DNS resolution.
- route URL is unavailable: the route exists, but the target server is not reachable on its registered host/port.
- external route like `http://demo.ppc.dikka.dev` fails DNS: verify both the base record and wildcard record point to the machine's Tailscale IP and are DNS-only when using Cloudflare.

## Persistence and limitations

The packaged systemd unit restores route definitions at startup and saves them on graceful shutdown using `/var/lib/local-router-<user>/routes.jsonl`. Restored routes are shielded from heartbeat removal until their target responds successfully once. Outside systemd, use `local-router serve --state-file <path> --shield-imported`, or use `local-router export routes.jsonl` and `local-router import routes.jsonl [--mode merge|set] [--shielded]` manually. Abrupt termination may lose changes since the last graceful snapshot. The router does not start, stop, or supervise target app servers.
