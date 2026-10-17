#!/bin/bash

set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
server_dir="$repo_root/SignalingServer"
root="$(mktemp -d "${TMPDIR:-/tmp}/quickdesk-recording-upload.XXXXXX")"
port="${QD_RECORDING_UPLOAD_PORT:-18086}"
binary="$root/signaling"
work_dir="$root/run"
pid=""
dist_dir="$server_dir/web/dist"
created_dist_placeholder=0

cleanup() {
    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
        kill "$pid" 2>/dev/null || true
        wait "$pid" 2>/dev/null || true
    fi
    if [ "$created_dist_placeholder" = 1 ]; then
        rm -f "$dist_dir/index.html"
        rmdir "$dist_dir" 2>/dev/null || true
    fi
    rm -rf "$root"
}
trap cleanup EXIT

fail() {
    echo "FAIL: $*" >&2
    if [ -f "$work_dir/server.log" ]; then
        tail -n 100 "$work_dir/server.log" >&2
    fi
    exit 1
}

if lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
    fail "port $port is already in use; set QD_RECORDING_UPLOAD_PORT"
fi

mkdir -p "$work_dir"
base_url="http://127.0.0.1:$port"

if [ ! -f "$dist_dir/index.html" ]; then
    mkdir -p "$dist_dir"
    printf '<!doctype html><title>recording-upload-e2e</title>\n' >"$dist_dir/index.html"
    created_dist_placeholder=1
fi

(
    cd "$server_dir"
    CGO_ENABLED=0 go build -o "$binary" ./cmd/signaling
) || fail "failed to build signaling server"

(
    cd "$work_dir"
    SERVER_HOST=127.0.0.1 \
    SERVER_PORT="$port" \
    DB_DRIVER=sqlite \
    DB_PATH="$work_dir/data/quickdesk.db" \
    REDIS_DRIVER=embedded \
    RUN_AUTO_MIGRATE=false \
    API_KEY= \
    LOG_DIR="$work_dir/logs" \
    "$binary" >>"$work_dir/server.log" 2>&1 &
    echo $! >"$work_dir/server.pid"
)
pid="$(cat "$work_dir/server.pid")"

deadline=$((SECONDS + 60))
while [ "$SECONDS" -lt "$deadline" ]; do
    if curl -fsS "$base_url/health" >/dev/null 2>&1; then
        break
    fi
    if ! kill -0 "$pid" 2>/dev/null; then
        fail "signaling server exited before becoming healthy"
    fi
    sleep 0.25
done
curl -fsS "$base_url/health" >/dev/null || fail "server did not become healthy"

QD_BASE_URL="$base_url" node "$script_dir/recording-upload-smoke.mjs" ||
    fail "recording upload smoke failed"
