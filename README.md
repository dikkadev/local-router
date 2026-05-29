# Local Router

A Go tool for giving temporary local web servers stable, friendly browser URLs such as `http://demo.localhost`, while the actual target server remains separately owned by the tool or shell that started it.

See [`docs/spec.md`](docs/spec.md) for the v1 specification.

## Quick start

There are two parts:

1. **Start the router service** — this is the long-running reverse proxy on `127.0.0.1:80`.
2. **Register routes** — tell the router which already-running local server/port should answer for a `.localhost` name.

Build the binary first:

```bash
go build ./cmd/local-router
```

Start the actual router service:

```bash
# Terminal 1: start the router service
sudo ./local-router serve
```

Keep that process running. Then start any local web server separately. For example:

```bash
# Terminal 2: example target server
python3 -m http.server 5173 --bind 127.0.0.1
```

Register that target port with the router:

```bash
# Terminal 3: register a friendly URL for the target server
./local-router register demo --port 5173 --title "Demo app"
```

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

The binary has built-in help:

```bash
./local-router --help
./local-router serve --help
./local-router register --help
```

## Why `sudo go run` may fail

If `sudo go run ./cmd/local-router serve` says it cannot find `go`, that is usually because `sudo` uses root's restricted `PATH`, not your normal user shell setup. This is common when Go is installed through a user-level tool such as `mise`, `asdf`, or a custom `$HOME` path.

Prefer building as your normal user and only using `sudo` for the built binary:

```bash
go build ./cmd/local-router
sudo ./local-router serve
```

That is also closer to how the tool will run as a Linux/WSL service later.

## Port 80 note

The service intentionally binds `127.0.0.1:80` so route URLs do not need `:port`.

If startup fails, either another process already owns port 80 or your OS requires elevated permission for low ports. For a built binary, a less broad Linux option than `sudo` is granting only the bind capability:

```bash
sudo setcap 'cap_net_bind_service=+ep' ./local-router
./local-router serve
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

Initial v1 implementation exists: in-memory route registry, REST API, control page, reverse proxy, heartbeat cleanup, and CLI commands. Service installation is not implemented yet; the intended direction is a Linux/WSL service, not a Windows service.
