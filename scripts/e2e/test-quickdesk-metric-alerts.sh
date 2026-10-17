#!/bin/bash

set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
server_dir="$repo_root/SignalingServer"
root="$(mktemp -d "${TMPDIR:-/tmp}/quickdesk-metric-alert.XXXXXX")"
server_port="${QD_METRIC_SERVER_PORT:-18084}"
webhook_port="${QD_METRIC_WEBHOOK_PORT:-19184}"
api_key="metric-e2e-key"
binary="$root/signaling"
work_dir="$root/run"
server_pid=""
receiver_pid=""

cleanup() {
    if [ -n "$receiver_pid" ] && kill -0 "$receiver_pid" 2>/dev/null; then
        kill "$receiver_pid" 2>/dev/null || true
        wait "$receiver_pid" 2>/dev/null || true
    fi
    if [ -n "$server_pid" ] && kill -0 "$server_pid" 2>/dev/null; then
        kill "$server_pid" 2>/dev/null || true
        wait "$server_pid" 2>/dev/null || true
    fi
    rm -rf "$root"
}
trap cleanup EXIT

fail() {
    echo "FAIL: $*" >&2
    if [ -f "$work_dir/server.log" ]; then
        tail -n 100 "$work_dir/server.log" >&2
    fi
    if [ -f "$work_dir/receiver.log" ]; then
        tail -n 40 "$work_dir/receiver.log" >&2
    fi
    exit 1
}

for port in "$server_port" "$webhook_port"; do
    if lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
        fail "port $port is already in use; override QD_METRIC_SERVER_PORT / QD_METRIC_WEBHOOK_PORT"
    fi
done

mkdir -p "$work_dir"
base_url="http://127.0.0.1:$server_port"

(
    cd "$server_dir"
    CGO_ENABLED=0 go build -o "$binary" ./cmd/signaling
) || fail "failed to build signaling server"

(
    cd "$work_dir"
    SERVER_HOST=127.0.0.1 \
    SERVER_PORT="$server_port" \
    DB_DRIVER=sqlite \
    DB_PATH="$work_dir/data/quickdesk.db" \
    REDIS_DRIVER=embedded \
    RUN_AUTO_MIGRATE=false \
    API_KEY="$api_key" \
    ADMIN_USER=admin \
    ADMIN_PASSWORD=admin \
    LOG_DIR="$work_dir/logs" \
    "$binary" >>"$work_dir/server.log" 2>&1 &
    echo $! >"$work_dir/server.pid"
)
server_pid="$(cat "$work_dir/server.pid")"

deadline=$((SECONDS + 60))
while [ "$SECONDS" -lt "$deadline" ]; do
    if curl -fsS "$base_url/health" >/dev/null 2>&1; then
        break
    fi
    if ! kill -0 "$server_pid" 2>/dev/null; then
        fail "signaling server exited before becoming healthy"
    fi
    sleep 0.25
done
curl -fsS "$base_url/health" >/dev/null || fail "server did not become healthy"

QD_LISTEN="$webhook_port" \
QD_SECRET=metric-e2e-secret \
QD_EXPECT=1 \
QD_TIMEOUT_MS=20000 \
node "$script_dir/notification-receiver.mjs" >"$work_dir/receiver.log" 2>&1 &
receiver_pid=$!
sleep 0.25
kill -0 "$receiver_pid" 2>/dev/null || fail "webhook receiver exited before the request"

suffix="${SECONDS}-$$"
username="metric-e2e-$suffix"
password="Metric-E2E-$suffix!"

register="$(curl -fsS -X POST "$base_url/v1/auth/register" \
    -H "X-API-Key: $api_key" \
    -H 'Content-Type: application/json' \
    -d "{\"username\":\"$username\",\"password\":\"$password\"}")"
token="$(printf '%s' "$register" | jq -r '.access_token')"
[ -n "$token" ] && [ "$token" != "null" ] || fail "register did not return an access token"

provision="$(curl -fsS -X POST "$base_url/v1/devices:provision" \
    -H "X-API-Key: $api_key" \
    -H 'Content-Type: application/json' \
    -d "{\"device_uuid\":\"metric-e2e-$suffix\",\"os\":\"test\",\"app_version\":\"e2e\"}")"
device_id="$(printf '%s' "$provision" | jq -r '.device_id')"
device_secret="$(printf '%s' "$provision" | jq -r '.device_secret')"
[ -n "$device_id" ] && [ -n "$device_secret" ] || fail "provision did not return a device id and secret"

curl -fsS -X POST "$base_url/v1/me/devices" \
    -H "X-API-Key: $api_key" \
    -H "Authorization: Bearer $token" \
    -H 'Content-Type: application/json' \
    -d "{\"device_id\":\"$device_id\"}" >/dev/null

curl -fsS -X POST "$base_url/v1/me/notification-channels" \
    -H "X-API-Key: $api_key" \
    -H "Authorization: Bearer $token" \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"metric e2e\",\"url\":\"http://127.0.0.1:$webhook_port/\",\"secret\":\"metric-e2e-secret\",\"events\":[\"device.metric.alert\"],\"enabled\":true}" >/dev/null

curl -fsS -X PUT "$base_url/v1/me/devices/$device_id/metric-alert" \
    -H "X-API-Key: $api_key" \
    -H "Authorization: Bearer $token" \
    -H 'Content-Type: application/json' \
    -d '{"enabled":true,"cpu_percent":1,"memory_percent":0,"disk_percent":0,"sustained_minutes":0,"repeat_minutes":0,"snoozed_until":null}' >/dev/null

captured_at="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
curl -fsS -X POST "$base_url/v1/devices/$device_id/metrics" \
    -H "X-API-Key: $api_key" \
    -H "Authorization: Bearer $device_secret" \
    -H 'Content-Type: application/json' \
    -d "{\"captured_at\":\"$captured_at\",\"cpu_percent\":91.5,\"memory_percent\":24,\"disk_percent\":33,\"disk_mount\":\"/\"}" >/dev/null

if ! wait "$receiver_pid"; then
    fail "metric alert was not delivered to the webhook"
fi
receiver_pid=""

view="$(curl -fsS "$base_url/v1/me/devices/$device_id/metric-alert" \
    -H "X-API-Key: $api_key" \
    -H "Authorization: Bearer $token")"
printf '%s' "$view" | jq -e '
    .enabled == true and
    .cpu_percent == 1 and
    .current.cpu_percent == 91.5 and
    (.states.cpu.exceeded_since | type) == "string" and
    (.states.cpu.last_alerted_at | type) == "string"
' >/dev/null || fail "metric-alert view did not expose the sample and fired episode"

grep -q '"event":"device.metric.alert"' "$work_dir/receiver.log" ||
    fail "webhook payload did not identify device.metric.alert"
grep -q '"metric":"cpu"' "$work_dir/receiver.log" ||
    fail "webhook payload did not identify the CPU metric"

echo "metric-alert E2E passed"
