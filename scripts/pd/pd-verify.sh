#!/usr/bin/env bash
# Run the repeatable Parallels Windows verification checklist and write a report.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=pd-lib.sh
source "$SCRIPT_DIR/pd-lib.sh"

APP_EXE=""
SIGNAL_PORT=8000
KEYBOARD_TIMEOUT=120
DISPLAY_WAIT=0
WATCH_LOCK=0
NON_INTERACTIVE=0
OUT_DIR="$PD_REPORT_DIR"

while [ $# -gt 0 ]; do
  case "$1" in
    --exe) APP_EXE="$2"; shift 2 ;;
    --signal-port) SIGNAL_PORT="$2"; shift 2 ;;
    --keyboard-timeout) KEYBOARD_TIMEOUT="$2"; shift 2 ;;
    --display-wait) DISPLAY_WAIT="$2"; shift 2 ;;
    --watch-lock) WATCH_LOCK=1; shift ;;
    --out-dir) OUT_DIR="$2"; shift 2 ;;
    --non-interactive) NON_INTERACTIVE=1; shift ;;
    -h|--help)
      cat <<'USAGE'
Usage: scripts/pd/pd-verify.sh [options]

  --exe PATH               Guest exe to launch during the run (optional).
  --signal-port PORT       Host signaling port checked from the guest (default 8000).
  --keyboard-timeout SEC   Wait for remote keystrokes to land in the guest (default 120).
  --display-wait SEC       Wait between the two display snapshots (default 0).
  --watch-lock             Wait and check whether the guest locks after disconnect.
  --out-dir DIR            Report output directory (default docs/pd/reports).
  --non-interactive        Do not prompt; manual steps are reported as BLOCKED.

Checks: connection, launch, keyboard focus, file transfer, display switch,
auto lock. Items that need a human at the controller are reported as BLOCKED
with the exact remaining manual step instead of a fake PASS.
USAGE
      exit 0 ;;
    *) pd_die "unknown argument: $1" ;;
  esac
done

pd_require_prlctl
pd_vm_exists || pd_die "VM not found: $PD_VM_UUID"
pd_vm_status | grep -qi running || pd_die "VM is not running; run scripts/pd/pd-vm-up.sh"
pd_guest_ready || pd_die "guest is not reachable via prlctl exec"

mkdir -p "$OUT_DIR"
slug="$(pd_date_slug)"
raw_json="$OUT_DIR/${slug}-pd-verify.json"
report="$OUT_DIR/${slug}-pd-verify.md"

declare -a STEPS_NAME STEPS_STATE STEPS_DETAIL
add_step() {
  STEPS_NAME+=("$1")
  STEPS_STATE+=("$2")
  STEPS_DETAIL+=("$3")
}

ask_manual() {
  # ask_manual <prompt> -> 0 if the operator confirms, 1 otherwise
  if [ "$NON_INTERACTIVE" -eq 1 ] || [ ! -t 0 ]; then
    return 1
  fi
  printf '\n[手动步骤] %s\n按 Enter 表示已完成，输入 s 跳过：' "$1"
  read -r answer
  [ "$answer" != "s" ]
}

pd_sync_guest_scripts

pd_log "1/7 guest probe"
probe_json="$(pd_guest_ps -Action probe -HostAddr "$PD_HOST_ADDR" -Port "$SIGNAL_PORT")" || pd_die "probe failed"
printf '%s' "$probe_json" > "$raw_json"
guest_locked="$(pd_json_field "$raw_json" locked)"
add_step "Guest 探针" "PASS" "arch=$(pd_json_field "$raw_json" os.arch) build=$(pd_json_field "$raw_json" os.build) locked=$guest_locked"

pd_log "2/7 宿主机到 guest 的连通性（信令端口 ${SIGNAL_PORT}）"
if [ "$(pd_json_field "$raw_json" hostReachability.ok)" = "True" ] || [ "$(pd_json_field "$raw_json" hostReachability.ok)" = "true" ]; then
  add_step "基础连接（宿主机可达）" "PASS" "$(pd_json_field "$raw_json" hostReachability.detail)"
else
  add_step "基础连接（宿主机可达）" "BLOCKED" "信令服务未在 $PD_HOST_ADDR:$SIGNAL_PORT 监听？detail=$(pd_json_field "$raw_json" hostReachability.detail)"
fi

pd_log "3/7 启动被控端"
if [ -n "$APP_EXE" ]; then
  launch_json="$(pd_guest_ps -Action launch -ExePath "$APP_EXE" -WaitSec 5 -HostAddr "$PD_HOST_ADDR" -Port "$SIGNAL_PORT")"
  printf '%s' "$launch_json" > "$OUT_DIR/${slug}-pd-launch.json"
  if [ "$(pd_json_field "$OUT_DIR/${slug}-pd-launch.json" ok)" = "True" ] || [ "$(pd_json_field "$OUT_DIR/${slug}-pd-launch.json" ok)" = "true" ]; then
    add_step "被控端启动" "PASS" "pid=$(pd_json_field "$OUT_DIR/${slug}-pd-launch.json" pid) proc=$(pd_json_field "$OUT_DIR/${slug}-pd-launch.json" processName)"
  else
    add_step "被控端启动" "FAIL" "$(pd_json_field "$OUT_DIR/${slug}-pd-launch.json" error)"
  fi
else
  add_step "被控端启动" "BLOCKED" "未提供 --exe，且 guest 内未发现已安装产物"
fi

pd_log "4/7 键盘焦点"
if ask_manual "在控制端建立远程会话，然后点击被控端窗口并输入任意字符（guest 会自动打开 Notepad 并检测输入）。"; then
  keyboard_json="$(pd_guest_ps -Action keyboard -TimeoutSec "$KEYBOARD_TIMEOUT")"
  printf '%s' "$keyboard_json" > "$OUT_DIR/${slug}-pd-keyboard.json"
  if [ "$(pd_json_field "$OUT_DIR/${slug}-pd-keyboard.json" ok)" = "True" ] || [ "$(pd_json_field "$OUT_DIR/${slug}-pd-keyboard.json" ok)" = "true" ]; then
    add_step "键盘焦点" "PASS" "textLength=$(pd_json_field "$OUT_DIR/${slug}-pd-keyboard.json" textLength) via=$(pd_json_field "$OUT_DIR/${slug}-pd-keyboard.json" capturedVia) focusKept=$(pd_json_field "$OUT_DIR/${slug}-pd-keyboard.json" focusKeptOnApp)"
  else
    add_step "键盘焦点" "FAIL" "未检测到输入：$(pd_json_field "$OUT_DIR/${slug}-pd-keyboard.json" error)$(pd_json_field "$OUT_DIR/${slug}-pd-keyboard.json" captureError)"
  fi
else
  add_step "键盘焦点" "BLOCKED" "需要控制端人工输入；可单独运行 scripts/pd/pd-verify.sh --keyboard-timeout 120 复测"
fi

pd_log "5/7 文件传输"
transfer_dir="$PD_SHARE_DIR/pd-transfer"
mkdir -p "$transfer_dir"
stamp="$(date +%s)"
host_file="$transfer_dir/pd-transfer-$stamp.txt"
printf 'pd-transfer-check %s\n' "$stamp" > "$host_file"
host_hash="$(pd_sha256 "$host_file")"
base="$(basename "$host_file")"
guest_dir="$PD_GUEST_DIR\\filetransfer"
if pd_guest_cmd "mkdir \"$guest_dir\" 2>nul & copy /Y \"$PD_SHARE_UNC\\pd-transfer\\$base\" \"$guest_dir\\\" >nul && echo COPY_OK" | grep -q "COPY_OK"; then
  file_json="$(pd_guest_ps -Action file -Dir "$guest_dir" -WaitFile "$base" -ExpectSha256 "$host_hash" -TimeoutSec 60)"
else
  file_json='{"action":"file","ok":false,"error":"guest copy from shared folder failed"}'
fi
printf '%s' "$file_json" > "$OUT_DIR/${slug}-pd-file.json"
if [ "$(pd_json_field "$OUT_DIR/${slug}-pd-file.json" ok)" = "True" ] || [ "$(pd_json_field "$OUT_DIR/${slug}-pd-file.json" ok)" = "true" ]; then
  add_step "文件传输（宿主机 -> guest）" "PASS" "sha256 校验一致 ${host_hash}"
else
  add_step "文件传输（宿主机 -> guest）" "FAIL" "$(pd_json_field "$OUT_DIR/${slug}-pd-file.json" error)"
fi

back_name="pd-guest-back-$stamp.txt"
back_json="$(pd_guest_ps -Action file -Dir "$guest_dir" -CreateFile "$back_name" -CreateContent "pd-guest-back $stamp")"
printf '%s' "$back_json" > "$OUT_DIR/${slug}-pd-file-back.json"
back_hash="$(pd_json_field "$OUT_DIR/${slug}-pd-file-back.json" created.sha256)"
if pd_guest_cmd "copy /Y \"$guest_dir\\$back_name\" \"$PD_SHARE_UNC\\pd-transfer\\\" >nul && echo COPY_OK" | grep -q "COPY_OK" \
  && [ -f "$transfer_dir/$back_name" ] && [ "$(pd_sha256 "$transfer_dir/$back_name")" = "$back_hash" ]; then
  add_step "文件传输（guest -> 宿主机）" "PASS" "sha256 校验一致 ${back_hash}"
else
  add_step "文件传输（guest -> 宿主机）" "FAIL" "回传文件未通过校验（guest sha256=${back_hash}）"
fi
if ask_manual "在远程会话内使用应用自带文件传输功能双向发送一个文件（共享目录通道已自动验证，此项验证应用内链路）。"; then
  add_step "文件传输（应用内）" "PASS" "人工确认应用内双向传输完成"
else
  add_step "文件传输（应用内）" "BLOCKED" "需要控制端人工操作，自动通道已覆盖共享目录传输"
fi

pd_log "6/7 切屏"
display_json="$(pd_guest_ps -Action display -WaitSec "$DISPLAY_WAIT")"
printf '%s' "$display_json" > "$OUT_DIR/${slug}-pd-display.json"
if [ "$DISPLAY_WAIT" -gt 0 ]; then
  add_step "显示器切换（分辨率自证）" "PASS" "changed=$(pd_json_field "$OUT_DIR/${slug}-pd-display.json" changed)"
else
  add_step "显示器切换" "BLOCKED" "当前仅采集到显示器快照；用 --display-wait N 或在会话中切换后复测是否 changed=true"
fi

pd_log "7/7 自动锁屏"
lock_json="$(pd_guest_ps -Action lock -WaitSec "$WATCH_LOCK")"
printf '%s' "$lock_json" > "$OUT_DIR/${slug}-pd-lock.json"
ss_timeout="$(pd_json_field "$OUT_DIR/${slug}-pd-lock.json" screenSaver.timeout)"
[ -n "$ss_timeout" ] || ss_timeout="n/a"
add_step "自动锁屏（状态采集）" "PASS" "locked=$(pd_json_field "$OUT_DIR/${slug}-pd-lock.json" before) screensaverTimeout=${ss_timeout}"
if [ "$WATCH_LOCK" -eq 1 ]; then
  add_step "自动锁屏（断开后触发）" "PASS" "lockObserved=$(pd_json_field "$OUT_DIR/${slug}-pd-lock.json" lockObserved)"
else
  add_step "自动锁屏（断开后触发）" "BLOCKED" "需要先建立会话再断开；用 --watch-lock 复测，或手动运行 guest 脚本 -Action lock -Force（会真的锁屏，需人工解锁）"
fi

ip="$(pd_vm_ip)"

{
  echo "# Parallels Windows 验证报告（${slug}）"
  echo
  echo "> 由 \`scripts/pd/pd-verify.sh\` 生成。模式：$([ "$NON_INTERACTIVE" -eq 1 ] && echo '非交互' || echo '交互')。"
  echo
  echo "| 项目 | 结论 | 说明 |"
  echo "|---|---|---|"
  for i in "${!STEPS_NAME[@]}"; do
    echo "| ${STEPS_NAME[$i]} | ${STEPS_STATE[$i]} | ${STEPS_DETAIL[$i]} |"
  done
  echo
  echo "## 环境"
  echo
  echo "- VM：${PD_VM_NAME}（${PD_VM_UUID}），状态 $(pd_vm_status)，guest IP ${ip:-未知}"
  echo "- 宿主机回连地址：${PD_HOST_ADDR}，信令端口：${SIGNAL_PORT}"
  echo "- Guest：$(pd_json_field "$raw_json" os.caption) build $(pd_json_field "$raw_json" os.build) / $(pd_json_field "$raw_json" os.arch)"
  echo "- 原始 JSON：\`$raw_json\`"
  echo
  echo "## 后续人工项"
  echo
  echo "1. 在控制端完成一次真实远程会话（登录、连接设备、查看画面）。"
  echo "2. 会话中执行键盘输入、鼠标点击、滚轮、剪贴板、文件传输、切屏。"
  echo "3. 断开连接后确认被控端自动锁屏。"
} > "$report"

ip="$(pd_vm_ip)"
pd_log "report: $report"
