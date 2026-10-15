#!/usr/bin/env bash
# Collect the Parallels Desktop Windows verification baseline and write a report.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=pd-lib.sh
source "$SCRIPT_DIR/pd-lib.sh"

OUT_DIR="$PD_REPORT_DIR"
RAW_JSON=""
while [ $# -gt 0 ]; do
  case "$1" in
    --out-dir) OUT_DIR="$2"; shift 2 ;;
    --raw-json) RAW_JSON="$2"; shift 2 ;;
    -h|--help)
      cat <<'USAGE'
Usage: scripts/pd/pd-probe.sh [--out-dir DIR] [--raw-json FILE]

Probes the Windows guest and the macOS host, then writes a markdown baseline
report (and the raw guest JSON) under docs/pd/reports by default.
USAGE
      exit 0 ;;
    *) pd_die "unknown argument: $1" ;;
  esac
done

pd_require_prlctl
pd_vm_exists || pd_die "VM not found: $PD_VM_UUID ($PD_VM_NAME)"
pd_vm_status | grep -qi running || pd_die "VM is not running; run scripts/pd/pd-vm-up.sh first"
pd_guest_ready || pd_die "guest is not reachable via prlctl exec"

mkdir -p "$OUT_DIR"
slug="$(pd_date_slug)"
[ -n "$RAW_JSON" ] || RAW_JSON="$OUT_DIR/${slug}-pd-probe.json"

pd_sync_guest_scripts
pd_log "probing guest"
pd_guest_ps -Action probe -HostAddr "$PD_HOST_ADDR" > "$RAW_JSON" || pd_die "guest probe failed"

ip="$(pd_vm_ip)"
report="$OUT_DIR/${slug}-pd-probe.md"

{
  echo "# Parallels Windows 探针报告（${slug}）"
  echo
  echo "> 由 \`scripts/pd/pd-probe.sh\` 生成，仅做只读探测。"
  echo
  echo "## 宿主机（macOS）"
  echo
  echo "| 项目 | 值 |"
  echo "|---|---|"
  echo "| 生成时间 | $(pd_timestamp) |"
  echo "| VM 名称 | $PD_VM_NAME |"
  echo "| VM UUID | $PD_VM_UUID |"
  echo "| VM 状态 | $(pd_vm_status) |"
  echo "| Guest IP | ${ip:-unknown} |"
  echo "| 宿主机回连地址 | $PD_HOST_ADDR |"
  echo "| 共享目录（宿主机） | $PD_SHARE_DIR |"
  echo "| 共享目录（Guest） | ${PD_SHARE_UNC} |"
  echo
  echo "## Guest（Windows）"
  echo
  echo "| 项目 | 值 |"
  echo "|---|---|"
  echo "| 主机名 | $(pd_json_field "$RAW_JSON" hostname) |"
  echo "| 用户 | $(pd_json_field "$RAW_JSON" user) |"
  echo "| 系统版本 | $(pd_json_field "$RAW_JSON" os.caption) |"
  echo "| 版本号 / 构建号 | $(pd_json_field "$RAW_JSON" os.version) / $(pd_json_field "$RAW_JSON" os.build) |"
  echo "| 架构 | $(pd_json_field "$RAW_JSON" os.arch) |"
  echo "| 机型 | $(pd_json_field "$RAW_JSON" computer.model) |"
  echo "| CPU/系统类型 | $(pd_json_field "$RAW_JSON" computer.systemType) |"
  echo "| 内存(GB) | $(pd_json_field "$RAW_JSON" computer.ramGB) |"
  echo "| 交互会话 | name=$(pd_json_field "$RAW_JSON" session.name) id=$(pd_json_field "$RAW_JSON" session.id) state=$(pd_json_field "$RAW_JSON" session.state) |"
  echo "| 当前是否锁屏 | $(pd_json_field "$RAW_JSON" locked)（判定方式：$(pd_json_field "$RAW_JSON" lockMethod)） |"
  echo "| 前台窗口 | $(pd_json_field "$RAW_JSON" foreground.title)（pid=$(pd_json_field "$RAW_JSON" foreground.pid), proc=$(pd_json_field "$RAW_JSON" foreground.process)） |"
  echo "| PowerShell | $(pd_json_field "$RAW_JSON" powershell.version) |"
  echo
  echo "## 显示器"
  echo
  echo "| 适配器 | 宽度 | 高度 |"
  echo "|---|---|---|"
  pd_json_columns "$RAW_JSON" displays "name,width,height" | sed 's/^/| /; s/$/ |/'
  echo
  echo "## 共享目录可见性（Guest）"
  echo
  pd_json_rows "$RAW_JSON" shared | sed 's/^/- /'
  echo
  echo "## 已安装的 QuickDesk / Go-Remote-Next 相关路径"
  echo
  pd_json_rows "$RAW_JSON" installed | sed 's/^/- /'
  echo
  echo "## 宿主机到 Guest 的连通性"
  echo
  echo "- Guest 侧 ping/TCP 到宿主机 \`$PD_HOST_ADDR\`：$(pd_json_field "$RAW_JSON" hostReachability.ok)（$(pd_json_field "$RAW_JSON" hostReachability.detail)）"
  echo "- 原始 JSON：\`$RAW_JSON\`"
} > "$report"

pd_log "report: $report"
pd_log "raw json: $RAW_JSON"
