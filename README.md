# Local Router

A Go tool for giving temporary local web servers stable, friendly browser URLs such as `http://demo.localhost`, while the actual target server remains separately owned by the tool or shell that started it.

See [`docs/spec.md`](docs/spec.md) for the v1 specification.

## Core idea

- Run one small persistent loopback-only router/reverse proxy on port 80.
- Let ad hoc local servers register and unregister routes dynamically.
- Use `.localhost` hostnames by default to avoid Windows hosts-file edits.
- Provide a live dashboard at `http://dev.localhost` and `http://router.localhost`.

## Build and run

```bash
go test ./...
go run ./cmd/local-router serve
```

The v1 service intentionally binds `127.0.0.1:80` for nice URLs without a port. If binding fails, another process may already own port 80 or the OS may require elevated privileges.

## CLI examples

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

## Status

Initial v1 implementation exists: in-memory route registry, REST API, control page, reverse proxy, heartbeat cleanup, and CLI commands. Windows service installation and persistent route storage are intentionally out of scope for v1.
