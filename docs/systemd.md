# systemd service setup

`local-router` is intended to run as a long-lived Linux/WSL service that owns `127.0.0.1:80`. The CLI then registers already-running target servers with that service.

This machine appears to use systemd (`ps -p 1 -o comm=` reports `systemd`).

## Install the binary

From the repo root:

```bash
go build -o bin/local-router ./cmd/local-router
sudo install -m 0755 bin/local-router /usr/local/bin/local-router
```

## Install the system service

The packaged unit is a template so the service can run as your normal Linux user while systemd grants only the low-port bind capability.

```bash
sudo install -m 0644 packaging/systemd/local-router@.service /etc/systemd/system/local-router@.service
sudo systemctl daemon-reload
sudo systemctl enable --now local-router@$USER.service
```

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

## Update after rebuilding

```bash
go build -o bin/local-router ./cmd/local-router
sudo install -m 0755 bin/local-router /usr/local/bin/local-router
sudo systemctl restart local-router@$USER.service
```

Route state is in memory, so restarting the service clears registered routes, including pinned routes.

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

That should let the non-root service bind `127.0.0.1:80`. If you run the binary manually without systemd, use `sudo local-router serve` or grant the binary the bind capability:

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
