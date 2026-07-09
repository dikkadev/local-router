package cli

import (
	"fmt"
	"io"
)

const helpText = `local-router gives local web servers friendly .localhost URLs.

USAGE
  local-router <command> [options]

SERVICE COMMAND
  serve [--addr 127.0.0.1:80] [--external-host <host>] [--state-file <path>] [--shield-imported] [--json]
      Start the long-running local reverse proxy. This is the actual router.
      Keep it running while you use registered routes.
      Use --external-host to also serve a tailnet/domain dashboard and subdomain routes.
      Use --state-file to restore routes at startup and save them on graceful shutdown.
      Use --shield-imported to keep restored routes until they become healthy once.
      Use --json for structured JSON logs.

CLIENT COMMANDS
  status
      Check whether the router service is reachable.

  register <name> --port <port> [options]
      Register an already-running local server as http://<name>.localhost.
      Prints only the final URL on success.

      Options:
        --title <text>           Display title for the dashboard
        --target-host <host>     Target host, default 127.0.0.1
        --heartbeat-path <path>  Health-check path, default /
        --exec <text>            Display/debug metadata for what started it
        --pinned                 Keep the route reserved while router runs
        --force                  Replace an existing route

  routes
      List registered routes.

  export <path|->
      Write current routes as JSONL, one route per line.

  import <path|-> [--mode merge|set] [--force] [--shielded]
      Read routes from JSONL. Default mode is merge; set replaces current routes.
      Shielded imports survive misses until their first successful heartbeat.

  unregister <name>
      Remove a route.

  pin <name>
      Keep a route reserved even when its target is down.

  unpin <name>
      Allow heartbeat cleanup to remove a stale route.

EXAMPLES
  # One-time install from Git source
  go install forge.dikka.dev/lab/local-router@latest

  # One-time low-port bind permission for the installed CLI
  sudo setcap 'cap_net_bind_service=+ep' "$(go env GOPATH)/bin/local-router"

  # Terminal 1: start the router service
  local-router serve

  # Terminal 2: start any target web server yourself
  python3 -m http.server 5173 --bind 127.0.0.1

  # Terminal 3: register the target port
  local-router register demo --port 5173 --title "Demo app"

  # Open these in a browser
  http://demo.localhost
  http://dev.localhost

NOTES
  local-router does not start your app/server. It only proxies to ports you register.
  The service binds port 80 for nice URLs, so Linux/WSL usually needs setcap, systemd capabilities, or sudo.
  The CLI talks to http://127.0.0.1 by default and sends Host: dev.localhost, so it does not depend on dev.localhost DNS resolution.
`

func printHelp(w io.Writer) {
	fmt.Fprint(w, helpText)
}

func printCommandHelp(w io.Writer, command string) bool {
	switch command {
	case "register":
		fmt.Fprint(w, `USAGE
  local-router register <name> --port <port> [options]

Register an already-running local server as http://<name>.localhost.

OPTIONS
  --port <port>             Required target port
  --title <text>            Display title for dashboard
  --target-host <host>      Target host, default 127.0.0.1
  --heartbeat-path <path>   Health-check path, default /
  --exec <text>             Display/debug metadata
  --pinned                  Keep route reserved while router runs
  --force                   Replace existing route

EXAMPLE
  local-router register demo --port 5173 --title "Demo app"
`)
	case "serve":
		fmt.Fprint(w, `USAGE
  local-router serve [--addr 127.0.0.1:80] [--external-host <host>] [--state-file <path>] [--shield-imported] [--json]

Start the long-running router/reverse-proxy service.
Keep this process running while you use registered routes.

OPTIONS
  --addr <addr>            Listen address, default 127.0.0.1:80 or LOCAL_ROUTER_ADDR
  --external-host <host>   Also serve dashboard at host and routes at <name>.<host>
                           Can also be set with LOCAL_ROUTER_EXTERNAL_HOST
  --state-file <path>      Restore on startup and save on graceful shutdown
  --shield-imported        Keep restored routes until they become healthy once
  --json                   Emit structured JSON logs using charmbracelet/log

EXAMPLES
  local-router serve
  local-router serve --json
  local-router serve --addr 0.0.0.0:80 --external-host ppc.dikka.dev
  local-router serve --state-file /var/lib/local-router/routes.jsonl --shield-imported
  local-router serve --addr 127.0.0.1:8080   # testing only; URLs need :8080
`)
	case "export":
		fmt.Fprint(w, `USAGE
  local-router export <path|->

Write current routes as JSONL, one route per line. Excludes transient health fields
such as misses and lastCheckAt.

EXAMPLE
  local-router export routes.jsonl
`)
	case "import":
		fmt.Fprint(w, `USAGE
  local-router import <path|-> [--mode merge|set] [--force] [--shielded]

Read routes from JSONL. Default mode is merge. Set mode removes current routes
before importing. Merge mode reports conflicts unless --force is used. Shielded
routes survive heartbeat misses until they respond successfully once.

EXAMPLE
  local-router import routes.jsonl
  local-router import routes.jsonl --mode set --shielded
`)
	case "status", "routes", "unregister", "pin", "unpin":
		fmt.Fprintf(w, "USAGE\n  local-router %s", command)
		if command == "unregister" || command == "pin" || command == "unpin" {
			fmt.Fprint(w, " <name>")
		}
		fmt.Fprint(w, "\n")
	default:
		return false
	}
	return true
}

func isHelpArg(arg string) bool {
	return arg == "help" || arg == "--help" || arg == "-h"
}
