#!/usr/bin/env bash
# Shared helpers for the Parallels Desktop (PD) Windows verification baseline.
# Source this file from other scripts/pd/*.sh scripts. It is not meant to be run directly.

PD_VM_NAME="${PD_VM_NAME:-Windows 11}"
PD_VM_UUID="${PD_VM_UUID:-{e987387b-9ef0-4b39-b510-4a505a6212bc}}"

# Guest work directory for pushed artifacts, scripts and verification output.
PD_GUEST_DIR="${PD_GUEST_DIR:-C:\\QuickDeskTest}"

# Host directory that is shared into the guest. Parallels maps the macOS home
# directory as \\Mac\Home, so anything under $HOME is reachable from the guest.
PD_SHARE_DIR="${PD_SHARE_DIR:-$HOME/PDShare}"
PD_SHARE_UNC="${PD_SHARE_UNC:-\\\\Mac\\Home\\PDShare}"

# Host-side address the guest uses to reach this Mac on the Parallels shared net.
PD_HOST_ADDR="${PD_HOST_ADDR:-10.211.55.2}"

PD_REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PD_REPORT_DIR="${PD_REPORT_DIR:-$PD_REPO_ROOT/docs/pd/reports}"
PD_GUEST_SCRIPT_DIR="$PD_GUEST_DIR\\scripts"
PD_GUEST_AGENT="$PD_GUEST_SCRIPT_DIR\\pd-guest-agent.ps1"

pd_log() {
  printf '[pd] %s\n' "$*"
}

pd_warn() {
  printf '[pd][warn] %s\n' "$*" >&2
}

pd_die() {
  printf '[pd][error] %s\n' "$*" >&2
  exit 1
}

pd_require_prlctl() {
  command -v prlctl >/dev/null 2>&1 || pd_die "prlctl not found; install Parallels Desktop CLI tools first"
}

pd_vm_exists() {
  prlctl list -a --no-header 2>/dev/null | grep -Fq "$PD_VM_UUID"
}

pd_vm_status() {
  prlctl list -o status "$PD_VM_UUID" 2>/dev/null | tail -n 1 | tr -d '[:space:]'
}

pd_vm_ip() {
  prlctl list -f "$PD_VM_UUID" 2>/dev/null | tail -n 1 | awk '{print $3}'
}

pd_guest_cmd() {
  prlctl exec "$PD_VM_UUID" --current-user cmd.exe /c "$1" 2>&1
}

pd_guest_ready() {
  pd_guest_cmd "echo PD_READY" | grep -q "PD_READY"
}

pd_wait_guest_ready() {
  local max_sec="${1:-180}"
  local waited=0
  while [ "$waited" -lt "$max_sec" ]; do
    if pd_guest_ready; then
      return 0
    fi
    sleep 5
    waited=$((waited + 5))
  done
  return 1
}

pd_sha256() {
  shasum -a 256 "$1" | awk '{print $1}'
}

pd_timestamp() {
  date +%Y-%m-%dT%H:%M:%S%z
}

pd_date_slug() {
  date +%Y-%m-%d
}

pd_to_guest_path() {
  # Convert a host path under $HOME into the guest UNC path.
  local host_path="$1"
  case "$host_path" in
    "$HOME"/*)
      local rel="${host_path#"$HOME"/}"
      printf '%s\\%s' "$PD_SHARE_UNC" "$(printf '%s' "$rel" | tr '/' '\\')"
      ;;
    *)
      return 1
      ;;
  esac
}

pd_stage_file() {
  # Copy a host file into the shared directory and echo its guest path.
  local src="$1"
  [ -f "$src" ] || pd_die "file not found: $src"
  local guest_path
  if guest_path="$(pd_to_guest_path "$src")"; then
    printf '%s' "$guest_path"
    return 0
  fi
  mkdir -p "$PD_SHARE_DIR"
  local base
  base="$(basename "$src")"
  cp -f "$src" "$PD_SHARE_DIR/$base" || pd_die "failed to stage $src into $PD_SHARE_DIR"
  printf '%s\\%s' "$PD_SHARE_UNC" "$base"
}

pd_sync_guest_scripts() {
  # Copy scripts/pd/guest/* into the guest work directory.
  mkdir -p "$PD_SHARE_DIR/pd-guest"
  cp -f "$PD_REPO_ROOT"/scripts/pd/guest/*.ps1 "$PD_SHARE_DIR/pd-guest/" \
    || pd_die "failed to stage guest scripts"
  pd_guest_cmd "mkdir \"$PD_GUEST_SCRIPT_DIR\" 2>nul & copy /Y \"$PD_SHARE_UNC\\pd-guest\\*.ps1\" \"$PD_GUEST_SCRIPT_DIR\\\" >nul" >/dev/null
  pd_guest_cmd "dir /b \"$PD_GUEST_SCRIPT_DIR\"" | grep -qi "pd-guest-agent.ps1" \
    || pd_die "guest agent script was not copied into $PD_GUEST_SCRIPT_DIR"
}

pd_decode_guest_json_output() {
  # Read combined guest output on stdin, decode the PDJSON_B64 payload and echo JSON.
  local raw
  raw="$(cat)"
  local b64
  b64="$(printf '%s' "$raw" | tr -d '\r' | sed -n 's/.*PDJSON_B64:\([A-Za-z0-9+/=]\{1,\}\).*/\1/p' | tail -n 1)"
  if [ -z "$b64" ]; then
    printf '%s\n' "$raw" >&2
    return 1
  fi
  printf '%s' "$b64" | base64 -d
}

pd_guest_ps() {
  # Run the guest agent script and echo decoded JSON on success.
  prlctl exec "$PD_VM_UUID" --current-user powershell.exe -NoProfile -ExecutionPolicy Bypass \
    -File "$PD_GUEST_AGENT" "$@" 2>&1 | pd_decode_guest_json_output
}

pd_json_pretty() {
  python3 -m json.tool
}

pd_json_field() {
  # pd_json_field <json-file> <dotted.path>
  python3 - "$1" "$2" <<'PY'
import json, sys

path = sys.argv[2]
with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)
value = data
for part in path.split("."):
    if part == "":
        continue
    if isinstance(value, list):
        value = value[int(part)]
    else:
        value = value.get(part)
    if value is None:
        print("")
        sys.exit(0)
if isinstance(value, (dict, list)):
    print(json.dumps(value, ensure_ascii=False))
else:
    print(value)
PY
}

pd_json_rows() {
  # pd_json_rows <json-file> <dotted.path> [separator]
  python3 - "$1" "$2" "${3:- , }" <<'PY'
import json, sys

path = sys.argv[2]
sep = sys.argv[3]
with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)
value = data
for part in path.split("."):
    if not part:
        continue
    value = value[int(part)] if isinstance(value, list) else value.get(part)
    if value is None:
        print("")
        sys.exit(0)
if isinstance(value, list):
    for item in value:
        if isinstance(item, dict):
            print(sep.join(str(v) for v in item.values()))
        else:
            print(item)
elif isinstance(value, dict):
    for key, item in value.items():
        print(f"{key}={item}")
else:
    print(value)
PY
}

pd_json_columns() {
  # pd_json_columns <json-file> <dotted.path> <field1,field2,...>
  python3 - "$1" "$2" "$3" <<'PY'
import json, sys

path, fields = sys.argv[2], sys.argv[3].split(",")
with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)
value = data
for part in path.split("."):
    if part:
        value = value[int(part)] if isinstance(value, list) else value.get(part)
items = value if isinstance(value, list) else [value]
for item in items:
    if isinstance(item, dict):
        print(" | ".join(str(item.get(f, "")) for f in fields))
    else:
        print(item)
PY
}
