# systemd service setup

`local-router` is intended to run as a long-lived Linux/WSL service that owns `127.0.0.1:80`. The CLI then registers already-running target servers with that service.

This machine appears to use systemd (`ps -p 1 -o comm=` reports `systemd`).

## Install the CLI

Install the CLI with Go:

```bash
go install forge.dikka.dev/lab/local-router@latest
```

The systemd unit uses a stable absolute path, so copy the installed CLI there:

```bash
sudo install -m 0755 "$(go env GOPATH)/bin/local-router" /usr/local/bin/local-router
```

If you are working from a local checkout and want that exact version instead of `@latest`, run this from the repo root first:

```bash
go install .
sudo install -m 0755 "$(go env GOPATH)/bin/local-router" /usr/local/bin/local-router
```

## Install the system service

The packaged unit is a template so the service can run as your normal Linux user while systemd grants only the low-port bind capability.

```bash
sudo install -m 0644 packaging/systemd/local-router@.service /etc/systemd/system/local-router@.service
sudo systemctl daemon-reload
sudo systemctl enable --now local-router@$USER.service
```

For tailnet/domain access, add a systemd override before starting or restart after adding it:

```bash
sudo systemctl edit local-router@$USER.service
```

Example override when `ppc.dikka.dev` and `*.ppc.dikka.dev` resolve to this
machine's Tailscale IP:

```ini
[Service]
Environment=LOCAL_ROUTER_ADDR=0.0.0.0:80
Environment=LOCAL_ROUTER_EXTERNAL_HOST=ppc.dikka.dev
```

`LOCAL_ROUTER_ADDR` can also be set to the machine's Tailscale IP plus `:80` if
binding only to the tailnet interface works in your environment.

Check it:

```bash
systemctl status local-router@$USER.service --no-pager
local-router status
local-router routes
```

Dashboard URLs:

```text
http://dev.localhost
http://router.localhost
```

With the external-host override above, the dashboard and route URLs are also:

```text
http://ppc.dikka.dev
http://demo.ppc.dikka.dev
```

## Update

The systemd service runs `/usr/local/bin/local-router`, so updating means building or downloading a new CLI binary, copying it to that path, and restarting the service. Stop the service before replacing the binary so the running service and the installed file are never out of sync.

From the remote source:

```bash
sudo systemctl stop local-router@$USER.service
go install forge.dikka.dev/lab/local-router@latest
sudo install -m 0755 "$(go env GOPATH)/bin/local-router" /usr/local/bin/local-router
sudo systemctl start local-router@$USER.service
local-router status
```

Or from a local checkout:

```bash
sudo systemctl stop local-router@$USER.service
go install .
sudo install -m 0755 "$(go env GOPATH)/bin/local-router" /usr/local/bin/local-router
sudo systemctl start local-router@$USER.service
local-router status
```

If the service is not running yet, skip the `stop` command and run the install/copy/start steps.

Route state is in memory, so stopping or restarting the service clears registered routes, including pinned routes. If you want a manual snapshot, run `local-router export routes.jsonl` before stopping and `local-router import routes.jsonl --mode set` after starting. Otherwise, re-register any routes you still need after the update.

## Stop or remove

```bash
sudo systemctl disable --now local-router@$USER.service
sudo rm -f /etc/systemd/system/local-router@.service
sudo systemctl daemon-reload
```

Optionally remove the installed binary:

```bash
sudo rm -f /usr/local/bin/local-router
```

## Troubleshooting

### Port 80 is already in use

```bash
sudo ss -ltnp 'sport = :80'
```

Stop the conflicting local service or choose which tool should own local port 80.

### Service cannot bind port 80

The unit uses:

```ini
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
```

That should let the non-root service bind `127.0.0.1:80`. If you run the CLI manually without systemd, grant the installed CLI the bind capability:

```bash
sudo setcap 'cap_net_bind_service=+ep' /usr/local/bin/local-router
local-router serve
```

### WSL notes

This setup expects systemd-enabled WSL. Verify with:

```bash
ps -p 1 -o comm=
```

If PID 1 is not `systemd`, use the foreground quick-start command from `README.md` or enable systemd in WSL before installing the service.
