# Local Router

A Go tool for giving temporary local web servers stable, friendly browser URLs such as `http://demo.localhost`, while the actual target server remains separately owned by the tool or shell that started it.

See [`docs/spec.md`](docs/spec.md) for the original v1 specification.

## Quick start

Install the CLI directly from the Git source:

```bash
go install forge.dikka.dev/lab/local-router@latest
```

Make sure Go's install directory is on your `PATH`:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

Allow the installed CLI to bind local port 80 without running the router as root:

```bash
sudo setcap 'cap_net_bind_service=+ep' "$(go env GOPATH)/bin/local-router"
```

There are two runtime parts:

1. **Start the router service** — the long-running reverse proxy on `127.0.0.1:80`.
2. **Register routes** — tell the router which already-running local server/port should answer for a `.localhost` name.

Start the router service:

```bash
# Terminal 1: start the router service
local-router serve

# Or use structured JSON logs
local-router serve --json
```

Keep that process running. Then start any local web server separately. For example:

```bash
# Terminal 2: example target server
python3 -m http.server 5173 --bind 127.0.0.1
```

Register that target port with the router:

```bash
# Terminal 3: register a friendly URL for the target server
local-router register demo --port 5173 --title "Demo app"
```

The CLI talks to `http://127.0.0.1` by default and sends `Host: dev.localhost` internally. That matters because some command-line tools do not resolve `dev.localhost` even when browsers handle `.localhost` correctly.

The register command prints the URL:

```text
http://demo.localhost
```

Open that URL in the browser. The dashboard is available at:

```text
http://dev.localhost
http://router.localhost
```

## Help

The CLI has built-in help:

```bash
local-router --help
local-router serve --help
local-router register --help
```

## Agent skill installation

This repo is also a Pi package. It exposes the `local-router` skill from [`skills/local-router/SKILL.md`](skills/local-router/SKILL.md), so Pi agents can learn the CLI workflow from the same repo as the tool.

For Pi, install the package from the remote repo after pushing a commit or tag:

```bash
pi install ssh://git@git.dikka.dev:2222/lab/local-router.git@<ref>
```

For local Pi skill development before pushing:

```bash
pi install /home/dikka/projs/local-router
```

The open skills CLI can also install the bundled skill for Pi or other supported agents:

```bash
# From a local checkout of this repository
npx skills add . --skill local-router -a pi -g -y

# Or from a reachable git clone URL
npx skills add ssh://git@git.dikka.dev:2222/lab/local-router.git --skill local-router -a pi -g -y
```

Omit `-g` to install into the current project instead of the global agent skill directory. For non-Pi agents, replace `-a pi` with the agent name supported by `npx skills`.

Other agent harnesses that understand `SKILL.md` files can install or copy the skill from:

```text
skills/local-router/SKILL.md
```

## Service setup

For persistent Linux/WSL systemd setup, see [`docs/systemd.md`](docs/systemd.md). The packaged unit template runs the router as your normal Linux user while granting only the low-port bind capability needed for `127.0.0.1:80`.

## Installing from a checkout

If you already cloned the repository and want to install that exact checkout instead of `@latest`, run this from the repo root:

```bash
go install .
```

## Port 80 note

The service intentionally binds `127.0.0.1:80` so route URLs do not need `:port`.

If startup fails, either another process already owns port 80 or your OS requires elevated permission for low ports. The quick start uses `setcap` to grant only the bind capability to the installed CLI:

```bash
sudo setcap 'cap_net_bind_service=+ep' "$(go env GOPATH)/bin/local-router"
local-router serve
```

## Core idea

- Run one small persistent loopback-only router/reverse proxy on port 80.
- Let ad hoc local servers register and unregister routes dynamically.
- Use `.localhost` hostnames by default to avoid hosts-file edits.
- Provide a live dashboard at `http://dev.localhost` and `http://router.localhost`.

## CLI reference

```bash
local-router status
local-router register demo --port 5173 --title "Demo app"
local-router register demo --port 5173 --title "Demo app" --pinned
local-router register demo --port 3000 --force
local-router routes
local-router export routes.jsonl
local-router import routes.jsonl --mode set
local-router pin demo
local-router unpin demo
local-router unregister demo
```

Successful `register` prints only the final URL, for example:

```text
http://demo.localhost
```

Important: `local-router` does **not** start your target web server. Start your app/dev server yourself first, then register its port.

## Status

Initial v1 implementation exists: in-memory route registry, JSONL route snapshot import/export, REST API, control page, reverse proxy, heartbeat cleanup, CLI commands, and a Linux/WSL systemd unit template.
