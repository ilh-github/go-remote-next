#!/bin/bash

set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
server_dir="$repo_root/SignalingServer"
root="$(mktemp -d "${TMPDIR:-/tmp}/quickdesk-minimal.XXXXXX")"
port="${QD_MINIMAL_SERVER_PORT:-18080}"
binary="$root/signaling"
work_dir="$root/run"
pid=""

cleanup() {
    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
        kill "$pid" 2>/dev/null || true
        wait "$pid" 2>/dev/null || true
    fi
    rm -rf "$root"
}
trap cleanup EXIT

fail() {
    echo "FAIL: $*" >&2
    if [ -f "$work_dir/server.log" ]; then
        tail -n 80 "$work_dir/server.log" >&2
    fi
    exit 1
}

if lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
    fail "port $port is already in use; set QD_MINIMAL_SERVER_PORT"
fi

mkdir -p "$work_dir"
base_url="http://127.0.0.1:$port"

start_server() {
    (
        cd "$work_dir"
        SERVER_HOST=127.0.0.1 \
        SERVER_PORT="$port" \
        DB_DRIVER=sqlite \
        DB_PATH="$work_dir/data/quickdesk.db" \
        DB_HOST=127.0.0.1 \
        DB_PORT=1 \
        REDIS_HOST=127.0.0.1 \
        REDIS_PORT=1 \
        RUN_AUTO_MIGRATE=false \
        LOG_DIR="$work_dir/logs" \
        "$binary" >>"$work_dir/server.log" 2>&1 &
        echo $! >"$work_dir/server.pid"
    )
    pid="$(cat "$work_dir/server.pid")"
}

wait_for_health() {
    deadline=$((SECONDS + 60))
    while [ "$SECONDS" -lt "$deadline" ]; do
        if body="$(curl -fsS "$base_url/health" 2>/dev/null)"; then
            printf '%s' "$body" | jq -e '
                .status == "ok" and
                .components.sqlite == "ok" and
                .components.redis == "ok" and
                (.components.postgres == null)
            ' >/dev/null
            return 0
        fi
        if ! kill -0 "$pid" 2>/dev/null; then
            fail "signaling server exited before becoming healthy"
        fi
        sleep 0.25
    done
    fail "timed out waiting for /health"
}

stop_server() {
    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
        kill "$pid"
        wait "$pid" 2>/dev/null || true
    fi
    pid=""
}

(
    cd "$server_dir"
    CGO_ENABLED=0 go build -o "$binary" ./cmd/signaling
) || fail "failed to build signaling server"

start_server
wait_for_health

[ -s "$work_dir/data/quickdesk.db" ] ||
    fail "SQLite database was not created"

stop_server
start_server
wait_for_health

[ -s "$work_dir/data/quickdesk.db" ] ||
    fail "SQLite database disappeared after restart"

echo "minimal signaling E2E passed"
