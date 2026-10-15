#!/usr/bin/env bash
# Start the Parallels Windows VM if needed and wait until the guest is reachable.
# Safe by design: never shuts down, deletes, suspends or reconfigures the VM.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=pd-lib.sh
source "$SCRIPT_DIR/pd-lib.sh"

WAIT_SEC=180
while [ $# -gt 0 ]; do
  case "$1" in
    --wait) WAIT_SEC="$2"; shift 2 ;;
    -h|--help)
      cat <<'USAGE'
Usage: scripts/pd/pd-vm-up.sh [--wait SECONDS]

Starts the configured Parallels Windows VM (default: "Windows 11") when it is
stopped and waits until `prlctl exec` works inside the guest.
USAGE
      exit 0 ;;
    *) pd_die "unknown argument: $1" ;;
  esac
done

pd_require_prlctl
pd_vm_exists || pd_die "VM not found: $PD_VM_UUID ($PD_VM_NAME)"

status="$(pd_vm_status)"
case "$status" in
  running)
    pd_log "VM already running"
    ;;
  stopped|suspended)
    pd_log "starting VM $PD_VM_NAME ($status -> running)"
    prlctl start "$PD_VM_UUID" >/dev/null || pd_die "prlctl start failed"
    ;;
  *)
    pd_die "unexpected VM status: $status"
    ;;
esac

pd_log "waiting for guest (max ${WAIT_SEC}s)"
if ! pd_wait_guest_ready "$WAIT_SEC"; then
  pd_warn "guest not reachable within ${WAIT_SEC}s"
  pd_warn "check that the Windows user is logged in and Parallels Tools is running"
  exit 1
fi

ip="$(pd_vm_ip)"
pd_log "guest ready: ip=${ip:-unknown} host=${PD_HOST_ADDR}"
pd_guest_cmd "echo PDARCH=%PROCESSOR_ARCHITECTURE%" | grep -i "PDARCH" || true
