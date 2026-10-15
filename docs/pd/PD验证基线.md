# Parallels Windows 真机验证基线

本目录用于固化 Go-Remote-Next 在 macOS + Parallels Desktop（PD）Windows 虚拟机上的
真机验证方式，避免每次重新摸索 `prlctl` 命令、共享目录、产物推送和验证顺序。

配套脚本位于 `scripts/pd/`，本轮实测证据位于 `docs/pd/reports/`。

## 1. 当前环境事实（本轮实测）

| 项目 | 值 |
|---|---|
| 宿主机 | macOS，`prlctl` 位于 `/usr/local/bin/prlctl` |
| VM 名称 | `Windows 11` |
| VM UUID | `{e987387b-9ef0-4b39-b510-4a505a6212bc}` |
| Guest IP | `10.211.55.4`（Parallels Shared 网络） |
| 宿主机回连地址 | `10.211.55.2`（bridge100） |
| Guest 系统 | Microsoft Windows 11 专业版，版本 `10.0.26200`，构建 `26200` |
| Guest 架构 | ARM64（`ARM 64 位处理器`，机型 `Parallels ARM Virtual Machine`，8 GB 内存） |
| Guest 会话 | 交互会话 `console`，已登录用户 `TIMBLD899\timbl` |
| PowerShell | 5.1.26100.9457 |
| Parallels Tools | `state=possibly_installed version=26.4.1-57516`，`prlctl exec` 可用 |
| 共享目录 | Guest `Z:` -> `\\Mac\Home`（`net use` 可能显示 `Disconnected`，但 UNC 路径可用） |
| 显示器 | 单屏 `Parallels Display Adapter (WDDM)`，`1701x768` |
| Guest 工作目录 | `C:\QuickDeskTest`（脚本创建，`scripts/`、`artifacts/`、`filetransfer/`） |
| 宿主机暂存目录 | `~/PDShare`（Guest 侧为 `\\Mac\Home\PDShare`） |

重要限制：这是 **Windows 11 ARM64** 虚拟机，不是 x64 真机。ARM64 上运行 x64 构建属于
模拟执行，功能可以验证，但编码器、性能、资源占用结论必须标注为“ARM VM 结果”，不能
直接等同于 x64 发布验证。

## 2. 前置条件

- macOS 上已安装 Parallels Desktop，`prlctl` 可用。
- VM 已安装 Parallels Tools，且 Windows 用户已登录到交互会话。
  `prlctl exec --current-user` 需要交互会话，否则采集类验证会落到服务会话。
- 宿主机需要 `python3`、`base64`、`shasum`（脚本用于解析 JSON、比较哈希）。
- Guest 通过 `\\Mac\Home` 访问宿主机用户目录，脚本据此推送产物和脚本。

## 3. 脚本清单

| 文件 | 作用 |
|---|---|
| `scripts/pd/pd-lib.sh` | 公共配置与函数：VM 解析、共享路径转换、Guest 执行、JSON 解析 |
| `scripts/pd/pd-vm-up.sh` | 启动 VM（若已停止）并等待 Guest 可执行；不会关机/删除/重装 |
| `scripts/pd/pd-probe.sh` | 只读探针，输出 `docs/pd/reports/<日期>-pd-probe.md/.json` |
| `scripts/pd/pd-deploy.sh` | 把产物经共享目录推送到 Guest 并做 SHA256 双向校验 |
| `scripts/pd/pd-verify.sh` | 7 项验证清单编排，输出 `docs/pd/reports/<日期>-pd-verify.md/.json` |
| `scripts/pd/guest/pd-guest-agent.ps1` | Guest 侧动作：`probe/launch/keyboard/file/display/lock/netstat` |

Guest 侧脚本以 Base64(UTF-8 JSON) 单行输出（前缀 `PDJSON_B64:`），避免中文在
`prlctl exec` 控制台代码页下乱码。

## 4. 使用流程

```bash
cd /Users/timbl/timbl_work/go-remote-next

# 1) 启动并等待 VM
./scripts/pd/pd-vm-up.sh

# 2) 采集环境基线（只读）
./scripts/pd/pd-probe.sh

# 3) 推送 Windows 产物（先拿到安装包或 exe）
./scripts/pd/pd-deploy.sh --artifact ~/Downloads/QuickDesk-win-x64-setup.exe

# 4) 交互式验证（会提示人工步骤；--non-interactive 则把人工项标为 BLOCKED）
./scripts/pd/pd-verify.sh --signal-port 8000 --keyboard-timeout 120
```

### 被控端产物的三种来源

1. 本 fork 的 GitHub Actions：`.github/workflows/quickdesk-windows.yml` 支持
   `workflow_dispatch`，产物为 `publish/QuickDesk-win-x64-setup.exe`。
   目前 `ilh-github/go-remote-next` 尚无运行记录，需要手动触发一次构建。
2. 上游 release 回退冒烟：`barry-ran/QuickDesk` 的 `v4.2.0.0` 提供
   `QuickDesk-win-x64-v4.2.0.0-setup.exe`，可在 ARM VM 上以 x64 模拟方式跑通基础链路。
3. 在 Windows 机器上本地构建：`scripts/build_qd_win.bat Release` 需要
   MSVC 2022 + Qt 6.8.x（`ENV_QT_PATH`，`msvc2022_64`）+ Inno Setup，
   宿主机 macOS 无法交叉编译，不建议把 Qt/MSVC 装进本 VM。

### 信令服务

`pd-verify.sh` 只检查宿主机端口可达性，不会自动启动服务。需要时先起信令：

```bash
cd SignalingServer && cp -n .env.example .env && docker compose up -d
cd .. && ./scripts/pd/pd-verify.sh --signal-port 8000
```

## 5. 验证项与自动化程度

| 验证项 | 自动化程度 | 说明 |
|---|---|---|
| VM 启动与 Guest 可达 | 全自动 | `pd-vm-up.sh` + 探针 |
| Guest 系统/架构/显示器/会话/锁屏状态 | 全自动 | `pd-probe.sh` |
| 宿主机到 Guest 连通性 | 全自动 | Guest 侧 ping + TCP 端口探测，可指定 `--signal-port` |
| 被控端启动 | 半自动 | `pd-verify.sh --exe '<guest exe>'` 启动并回报进程/Foreground/连接 |
| 键盘焦点 | 半自动 | Guest 自动开 Notepad，人工在控制端输入，脚本检测标题变更与控件文本 |
| 文件传输（宿主机 <-> Guest） | 全自动 | 经共享目录双向传输 + SHA256 双向校验 |
| 文件传输（应用内链路） | 人工 | 需在真实会话内使用应用自带传输功能 |
| 切屏 / 多显示器 | 半自动 | 快照采集自动；`--display-wait N` 可在切换后判断 `changed` |
| 自动锁屏（状态采集） | 全自动 | 读取 LogonUI 存在性与屏保配置 |
| 自动锁屏（断开后触发） | 半自动 | `--watch-lock` 等待锁屏事件；`-Force` 会真的锁屏，需人工解锁 |

键盘焦点检查会用三种信号互相印证：窗口标题脏标记、UI Automation 控件文本
（`ValuePattern`/`TextPattern`）、以及 `Ctrl+A`/`Ctrl+C` 剪贴板兜底读取。

## 6. 本轮已知阻塞与下一步人工操作

| 阻塞项 | 原因 | 下一步 |
|---|---|---|
| 无本 fork 的 Windows 产物 | fork 仓库 Actions 无运行记录，guest 内也没有安装包 | 手动触发 `quickdesk-windows.yml`，或用上游 v4.2.0.0 安装包做首次冒烟 |
| 宿主机 8000 端口无信令服务 | 本轮未启动信令栈，避免改变宿主机服务状态 | `cd SignalingServer && docker compose up -d` 后重跑 `pd-verify.sh` |
| 键盘焦点 / 切屏 / 断开自动锁屏 / 应用内文件传输 | 需要真实控制端会话和人工操作 | 在控制端完成一次完整会话后重跑交互模式 `pd-verify.sh` |
| x64 真机性能数据 | 当前 VM 为 ARM64，x64 构建走模拟 | 需要 x64 真机或 x64 VM 再做编码/性能专项 |

## 7. 安全边界

- 脚本不会关机、挂起、删除、重装或重配置 VM；`prlctl start` 是本基线中唯一改变 VM 状态的操作。
- 不修改应用源码；写入范围限定为 `scripts/pd/`、`docs/pd/`、宿主机 `~/PDShare`
  与 Guest 的 `C:\QuickDeskTest`。
- 自动锁屏验证默认只读；显式传 `-Force` 才会真的锁屏，且需要人工解锁。
- 共享目录映射使用宿主机用户目录，请勿在此之前把敏感文件放进 `~/PDShare`。

## 8. 本轮实测证据

- `docs/pd/reports/2026-10-15-pd-probe.md` / `.json`：Guest 与宿主机环境基线。
- `docs/pd/reports/2026-10-15-pd-verify.md` / `.json`：7 项验证结论。
- 键盘焦点探针自检：Guest 内 Notepad 收到 `pd-selftest`，`ok=true`，
  抓取方式 `ValuePattern`。
- 文件传输：宿主机 -> Guest 与 Guest -> 宿主机两个方向 SHA256 均一致。
- 连通性探针：宿主机临时监听 `10.211.55.2:18099` 时 Guest 侧返回
  `ping=True tcp/18099=True`，证明端口探测链路有效。
