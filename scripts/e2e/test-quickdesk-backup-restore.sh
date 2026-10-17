#!/bin/bash

set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
server_dir="$repo_root/SignalingServer"
root="$(mktemp -d "${TMPDIR:-/tmp}/quickdesk-backup.XXXXXX")"
port="${QD_BACKUP_TEST_PORT:-18081}"
source_data="$root/source"
restore_data="$root/restore"
archive="$root/quickdesk-backup.tar.gz"
signaling_bin="$root/signaling"
backup_bin="$root/quickdesk-backup"
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
    exit 1
}

if lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
    fail "port $port is already in use; set QD_BACKUP_TEST_PORT"
fi

start_server() {
    data_dir="$1"
    mkdir -p "$data_dir"
    (
        cd "$data_dir"
        SERVER_HOST=127.0.0.1 \
        SERVER_PORT="$port" \
        DB_DRIVER=sqlite \
        DB_PATH="$data_dir/quickdesk.db" \
        DB_HOST=127.0.0.1 \
        DB_PORT=1 \
        REDIS_HOST=127.0.0.1 \
        REDIS_PORT=1 \
        RECORDING_DIR="$data_dir/recordings" \
        UPDATE_ARTIFACT_DIR="$data_dir/updates" \
        RUN_AUTO_MIGRATE=false \
        LOG_DIR="$data_dir/logs" \
        "$signaling_bin" >"$data_dir/server.log" 2>&1 &
        echo $! >"$data_dir/server.pid"
    )
    pid="$(cat "$data_dir/server.pid")"
}

wait_for_health() {
    deadline=$((SECONDS + 60))
    while [ "$SECONDS" -lt "$deadline" ]; do
        if curl -fsS "http://127.0.0.1:$port/health" 2>/dev/null |
            jq -e '.status == "ok" and .components.sqlite == "ok"' >/dev/null; then
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
    CGO_ENABLED=0 go build -o "$signaling_bin" ./cmd/signaling
    CGO_ENABLED=0 go build -o "$backup_bin" ./cmd/quickdesk-backup
) || fail "failed to build signaling or backup command"

start_server "$source_data"
wait_for_health

mkdir -p "$source_data/recordings/7" "$source_data/updates/9"
printf 'recording-bytes' >"$source_data/recordings/7/rec-42.mp4"
printf 'installer-bytes' >"$source_data/updates/9/QuickDesk-4.4.0.exe"
stop_server

"$backup_bin" backup \
    -db "$source_data/quickdesk.db" \
    -recordings "$source_data/recordings" \
    -updates "$source_data/updates" \
    -output "$archive" >"$root/backup.json" ||
    fail "backup command failed"

tar -tzf "$archive" | grep -qx 'manifest.json' ||
    fail "backup archive has no manifest"
tar -tzf "$archive" | grep -qx 'database/quickdesk.db' ||
    fail "backup archive has no database snapshot"

"$backup_bin" restore \
    -archive "$archive" \
    -db "$restore_data/quickdesk.db" \
    -recordings "$restore_data/recordings" \
    -updates "$restore_data/updates" >"$root/restore.json" ||
    fail "restore command failed"

cmp -s "$source_data/recordings/7/rec-42.mp4" "$restore_data/recordings/7/rec-42.mp4" ||
    fail "restored recording differs from the source"
cmp -s "$source_data/updates/9/QuickDesk-4.4.0.exe" "$restore_data/updates/9/QuickDesk-4.4.0.exe" ||
    fail "restored installer differs from the source"

start_server "$restore_data"
wait_for_health
stop_server

echo "backup/restore E2E passed"
