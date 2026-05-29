package cli

import (
	"fmt"
	"io"
)

const helpText = `local-router gives local web servers friendly .localhost URLs.

USAGE
  local-router <command> [options]

SERVICE COMMAND
  serve [--addr 127.0.0.1:80]
      Start the long-running local reverse proxy. This is the actual router.
      Keep it running while you use registered routes.

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

  unregister <name>
      Remove a route.

  pin <name>
      Keep a route reserved even when its target is down.

  unpin <name>
      Allow heartbeat cleanup to remove a stale route.

EXAMPLES
  # Terminal 1: start the router service
  sudo env "PATH=$PATH" go run ./cmd/local-router serve

  # Terminal 2: start any target web server yourself
  python3 -m http.server 5173 --bind 127.0.0.1

  # Terminal 3: register the target port
  ./local-router register demo --port 5173 --title "Demo app"

  # Open these in a browser
  http://demo.localhost
  http://dev.localhost

NOTES
  local-router does not start your app/server. It only proxies to ports you register.
  The service binds port 80 for nice URLs, so Linux/WSL usually needs sudo or setcap.
  The CLI talks to http://127.0.0.1 by default and sends Host: dev.localhost, so it does not depend on dev.localhost DNS resolution.
  If sudo cannot find go, preserve PATH: sudo env "PATH=$PATH" go run ./cmd/local-router serve.
  On this machine, an absolute path should also work: sudo /usr/local/go/bin/go run ./cmd/local-router serve.
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
  ./local-router register demo --port 5173 --title "Demo app"
`)
	case "serve":
		fmt.Fprint(w, `USAGE
  local-router serve [--addr 127.0.0.1:80]

Start the long-running router/reverse-proxy service.
Keep this process running while you use registered routes.

EXAMPLES
  sudo env "PATH=$PATH" go run ./cmd/local-router serve
  sudo /usr/local/go/bin/go run ./cmd/local-router serve
  ./local-router serve --addr 127.0.0.1:8080   # testing only; URLs need :8080
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
