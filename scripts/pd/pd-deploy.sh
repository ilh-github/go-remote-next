#!/usr/bin/env bash
# Copy verification artifacts (installer / exe / zip) from the Mac into the guest.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=pd-lib.sh
source "$SCRIPT_DIR/pd-lib.sh"

ARTIFACTS=()
GUEST_DEST="$PD_GUEST_DIR\\artifacts"
VERIFY_HASH=1

while [ $# -gt 0 ]; do
  case "$1" in
    --artifact) ARTIFACTS+=("$2"); shift 2 ;;
    --guest-dir) GUEST_DEST="$2"; shift 2 ;;
    --no-hash) VERIFY_HASH=0; shift ;;
    -h|--help)
      cat <<'USAGE'
Usage: scripts/pd/pd-deploy.sh --artifact PATH [--artifact PATH ...] [--guest-dir DIR] [--no-hash]

Stages each artifact through the Parallels shared folder and copies it into the
guest work directory. SHA256 is compared on both sides unless --no-hash is set.

Examples:
  scripts/pd/pd-deploy.sh --artifact publish/QuickDesk-win-x64-setup.exe
  scripts/pd/pd-deploy.sh --artifact bin/QuickDesk.exe --guest-dir 'C:\QuickDeskTest\bin'
USAGE
      exit 0 ;;
    *) pd_die "unknown argument: $1" ;;
  esac
done

[ "${#ARTIFACTS[@]}" -gt 0 ] || pd_die "at least one --artifact is required"
pd_require_prlctl
pd_vm_exists || pd_die "VM not found: $PD_VM_UUID"
pd_guest_ready || pd_die "guest is not reachable; run scripts/pd/pd-vm-up.sh"

mkdir -p "$PD_SHARE_DIR/pd-artifacts"
pd_guest_cmd "mkdir \"$GUEST_DEST\" 2>nul" >/dev/null

for src in "${ARTIFACTS[@]}"; do
  [ -f "$src" ] || pd_die "artifact not found: $src"
  base="$(basename "$src")"
  staged="$PD_SHARE_DIR/pd-artifacts/$base"
  cp -f "$src" "$staged" || pd_die "failed to stage $src"
  host_hash="$(pd_sha256 "$staged")"
  guest_src="$PD_SHARE_UNC\\pd-artifacts\\$base"
  guest_dst="$GUEST_DEST\\$base"
  pd_log "copying $base -> $guest_dst"
  pd_guest_cmd "copy /Y \"$guest_src\" \"$guest_dst\" >nul && echo COPY_OK" | grep -q "COPY_OK" \
    || pd_die "guest copy failed for $base"
  if [ "$VERIFY_HASH" -eq 1 ]; then
    json="$(pd_guest_ps -Action file -Dir "$GUEST_DEST" -WaitFile "$base" -ExpectSha256 "$host_hash" -TimeoutSec 30)"
    printf '%s' "$json" > /tmp/pd-deploy-hash.json
    got="$(pd_json_field /tmp/pd-deploy-hash.json waitFile.sha256)"
    matched="$(pd_json_field /tmp/pd-deploy-hash.json waitFile.hashMatched)"
    if [ "$got" != "$host_hash" ]; then
      pd_die "hash mismatch for $base: host=$host_hash guest=$got"
    fi
    pd_log "$base ok: sha256=$host_hash matched=$matched bytes=$(pd_json_field /tmp/pd-deploy-hash.json waitFile.size)"
  fi
done

pd_log "guest artifacts ready in $GUEST_DEST"
