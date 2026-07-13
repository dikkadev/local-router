#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROUTE_NAME="${ROUTE_NAME:-local-router-website}"
ROUTE_TITLE="${ROUTE_TITLE:-Local Router website}"
SERVER_PID=""
ROUTE_REGISTERED=false

if [[ -z "${PORT:-}" ]]; then
  PORT="$(python3 - <<'PY'
import socket

with socket.socket() as sock:
    sock.bind(("127.0.0.1", 0))
    print(sock.getsockname()[1])
PY
)"
fi

cleanup() {
  local exit_code=$?
  trap - EXIT INT TERM

  if [[ "$ROUTE_REGISTERED" == true ]]; then
    local-router unregister "$ROUTE_NAME" >/dev/null 2>&1 || true
  fi

  if [[ -n "$SERVER_PID" ]] && kill -0 "$SERVER_PID" 2>/dev/null; then
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi

  exit "$exit_code"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

if command -v pnpm >/dev/null 2>&1; then
  server_command=(pnpm dlx live-server)
elif command -v npx >/dev/null 2>&1; then
  server_command=(npx --yes live-server)
else
  echo "Error: pnpm or npx is required to run the live-reload server." >&2
  exit 1
fi

"${server_command[@]}" "$HERE" \
  --host=127.0.0.1 \
  --port="$PORT" \
  --no-browser \
  --wait=100 &
SERVER_PID=$!

for _ in {1..100}; do
  if curl --silent --fail --output /dev/null "http://127.0.0.1:$PORT/"; then
    break
  fi
  if ! kill -0 "$SERVER_PID" 2>/dev/null; then
    wait "$SERVER_PID"
    exit $?
  fi
  sleep 0.1
done

if ! curl --silent --fail --output /dev/null "http://127.0.0.1:$PORT/"; then
  echo "Error: live-reload server did not become ready on port $PORT." >&2
  exit 1
fi

echo "Live reload: http://127.0.0.1:$PORT"

if command -v local-router >/dev/null 2>&1 && local-router status >/dev/null 2>&1; then
  if route_url="$(local-router register "$ROUTE_NAME" --port "$PORT" --title "$ROUTE_TITLE" --exec "./website/run.sh" --force 2>/dev/null)"; then
    ROUTE_REGISTERED=true
    echo "Friendly URL: $route_url"
  else
    echo "Note: local-router is running, but '$ROUTE_NAME' could not be registered; using the direct URL." >&2
  fi
fi

echo "Watching $HERE (Ctrl-C to stop)"
wait "$SERVER_PID"
