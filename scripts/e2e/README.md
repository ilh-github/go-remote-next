# 端到端联调脚本（自托管信令 + 真实被控端）

这套脚本用来在没有构建本仓库 Windows 产物的情况下，验证
「WebClient → 自托管 SignalingServer → Windows 被控端」的完整链路：
画面、鼠标、键盘三条路都要拿到证据，不允许只看截图就下结论。

## 组成

| 文件 | 作用 |
|---|---|
| `test-quickdesk-minimal-server.sh` | 从临时目录构建并启动轻量信令（SQLite + 进程内 Redis），断言 `/health` 不依赖外部 PostgreSQL/Redis，并在重启后确认 SQLite 数据文件仍在 |
| `test-quickdesk-backup-restore.sh` | 启动真实轻量信令生成 SQLite/媒体数据，运行 `quickdesk-backup` 备份并恢复，校验 manifest、数据库快照、录像和安装包字节后用恢复库重新启动 |
| `remote-smoke.mjs` | 用真实 Chrome 打开 WebClient，连接设备，断言解码后的视频帧在推进；可选向远端打字 |
| `record.mjs` | 连上之后点录制按钮，录一段再停止，断言上传日志出现（录 → 存 → 索引） |
| `record-failure.mjs` | 录制时拦截上传请求制造失败，断言索引行最终是 `failed` 且没有产物（不允许"假完成"） |
| `record-retry.mjs` | 中断前两次产物上传与第一次预览上传，断言退避重试后仍然落地，并且最终行是 `completed` **且带 `thumbnail_url`** |
| `record-concurrent.mjs` | 同时开 N 个会话录同一个设备（`QD_SESSIONS`，默认 2），断言每路都独立落库为 `completed` 且各自有产物与预览图 |
| `wall.mjs` | 屏幕墙：把一台在线设备和一台离线设备放上墙，断言在线砖真的在解码（`videoWidth>0` 且时钟推进）、**默认没有创建鼠标/键盘/触摸处理器**（只看不控）、并隐藏了砖内的连接条与工具按钮；离线砖只显示占位与「重新连接」；断言单砖放大后宽度覆盖整行、轮巡能移动焦点且停止后不再变化。给 `QD_GUEST_UUID` 后还会解锁砖内输入、键入一串 token 并从输入靶子 spool 里读回，再锁定并断言处理器被移除 |
| `playback.mjs` | 在录像页点「播放」，断言签名 URL 打开的播放器 `currentTime` 在推进 |
| `guest-input-target.ps1` | 被控端侧的单窗口输入靶子（WinForms 文本框，内容变化即落盘），比 Win11 记事本可靠 |
| `guest-qt-ui.ps1` | 用 UI Automation 驱动 Windows 上的 QuickDesk 桌面客户端（填设备号/接入码、点连接、发快捷键），用于验证 Qt 侧录制 |
| `guest-notepad.ps1` | 备选靶子：用 UI Automation 读写记事本，适合快速人工核对 |
| `qd-api-probe.mjs` | 直接打 QuickDesk 本地 WebSocket API（默认 `ws://127.0.0.1:9600`）：`listConnections` / `connectToHost` / `skillListTools` / `skillExec`，用来绕开界面单独验证 skill bridge |
| `qd-host-stdio-harness.py` | 直接用 Chromium 原生消息协议启动 `quickdesk_host`，绕开 GUI/安全桌面；可指定独立 `--config-dir` 和 `--log-dir`，打印 `hostReady`、信令状态和 Host stderr，用于复现 SPAKE2、文件传输和用户上下文问题 |
| `realtime-access-events.mjs` | 连上用户的实时流（`/v1/realtime/events`）等 `device.access.*` 事件，`QD_WAIT_MS` 内没等到就退出码 3，可直接当验证闸门 |
| `notification-receiver.mjs` | 本地 webhook 接收器：打印每一条投递并校验 `X-Webhook-Signature` 的 HMAC，收满 `QD_EXPECT` 条就退出（签名不对退 4、超时退 3），用来验证用户级通知通道 |
| `signal-client-probe.mjs` | 开一条控制端信令连接并报告服务端的裁决：退 0 = `auth_ok`，退 3 = 收到 `error`（例如 `SESSION_LIMIT_REACHED`），退 4 = 没等到 auth_ok 就断开；`QD_HOLD_MS` 可以让连上的 socket 占住名额，用来测并发上限 |
| `signal-load.mjs` | 信令层并发压测台：`QD_CLIENTS`/`QD_ROUNDS` 控制突发与轮转，`QD_HOLD_MS` 占住会话，`QD_EXPECT_ACCEPTED`/`QD_EXPECT_REJECTED` 可直接断言并发上限；输出连接延迟 p50/p95/max 与拒绝码分布（不符合预期退 3，传输错误退 4） |
| `turn-capacity.mjs` | coturn 压测台：容量（并发 allocation）、中继质量（两路 allocation 真发数据）、soak（长稳 + 定期 refresh）三种模式；URL scheme 决定传输（`turn:` = UDP/TCP，`turns:` = TLS），凭据可本地用共享密钥签或从 `/v1/ice-config` 取。`QD_TURN_CHANNEL=1` 走真实客户端用的 **ChannelData** 路径（ChannelBind + 4 字节分帧；soak 会按 600 秒寿命重发绑定），默认仍是 Send/Data indication。TLS 默认做证书校验，`QD_TURN_CA` 指定 CA，`QD_TURN_INSECURE=1` 只给本地自签用。端口池耗尽可以用 `QD_TURN_EXPECT_FAILURES` + `QD_TURN_EXPECT_ERROR` 断言（默认会因"没有全部成功"判 FAIL），`QD_TURN_POOL_RECYCLE=1` 则测"池排空后多久能再用" |
| `turn-dtls.sh` | DTLS/TLS 冒烟：coturn 自带的 `turnutils_uclient` 跑 `-S`（无 `-t` 即 DTLS over UDP，加 `-t` 即 TLS over TCP），断言两个 leg 都 0 丢包。**Node 没有 DTLS 栈**（`node:tls` 只做 TCP 上的 TLS），所以 DTLS 只能这么验；脚本注释里也写了为什么 |
| `turn-dtls-soak.sh` | DTLS **小时级**长跑：同样用 `turnutils_uclient -S`，但按 `QD_TURN_SOAK_SECONDS`（默认 3600）与每消息间隔算出消息数，启动 echo peer，断言 `sent == recv`、`lost == 0`、**实际墙钟时长达到请求时长**（防止静默早退），并统计 coturn 日志里本次运行期间的 `allocation refreshed ... method=DTLSv1` 条数——600 秒 lifetime 之上还能全量中继，才算真的跨过了 refresh 路径 |
| `coturn-local.conf.example` | 本地 coturn 配置示例：`use-auth-secret` 模式、可选 `max-bps` 限速、TLS/DTLS 证书与端口。注意 coturn 4.18 必须显式写 `dtls`，否则只开 TLS/TCP，日志会说 "DTLS listeners are not started" |
| `coturn-tiny-pool.conf.example` | 第二个 coturn 实例，中继端口池故意只留 4 个（`min-port`/`max-port`），与主实例共用 `static-auth-secret`。用来验两件事：同一份 `/v1/ice-config` 凭据能同时认证到两个实例；以及端口池排空时的真实行为（`508`，reason 为空串） |
| `guest-uia-invoke.ps1` | 用 UI Automation 的 `InvokePattern` 点按钮（比合成鼠标可靠）：`-Hwnd` 指定顶层窗口，`-Name` 或 `-AutomationIdSuffix` 定位元素 |
| `guest-uia-buttons.ps1` | 列出 QuickDesk 主窗口里所有按钮/分组元素的 `AutomationId`、名称和屏幕坐标，用来找没有名字的 QML 按钮 |
| `guest-uia-setedit.ps1` | 按顺序设置某个顶层窗口里的第 N 个输入框（QML 输入框没有可访问名字） |
| `guest-list-windows.ps1` | 按进程列出顶层窗口（句柄、可见性、矩形、标题），确认"窗口到底开没开" |
| `guest-close-hard-error.ps1` | 关掉 `csrss` 持有的 `xxx.exe - 系统错误` 僵尸弹窗（会吞掉合成鼠标点击），按标题片段匹配 |
| `guest-verify-qd-win-release.ps1` | 验证 Windows 发布目录完整通过检查，并确认缺 VC Runtime、`qwindows.dll` 或 Qt 直接依赖时会被拒绝 |

## 依赖

- Node 18+，`npm i playwright`（用本机已安装的 Chrome，`channel: 'chrome'`）
- Python 3.10+（仅 `qd-host-stdio-harness.py` 需要）
- 一个跑起来的 SignalingServer，以及一个已经绑定到该用户的在线设备
- WebClient 构建产物用静态服务器暴露，例如：

```bash
cd WebClient && npm run build
python3 -m http.server 5173 --bind 127.0.0.1 --directory dist
```

## 跑一次

```bash
QD_TOKEN=<access_token> \
QD_DEVICE=<device_id> \
QD_SERVER=ws://10.211.55.2:8000 \
QD_APP=http://127.0.0.1:5173 \
node remote-smoke.mjs
```

可选环境变量：

- `QD_TYPE_TEXT`：连上之后向远端输入的字符串
- `QD_CLEAR_FIRST=1`：输入前先 Ctrl+A / Delete
- `QD_CLICK_FRAC=x,y`：点击远端画面的相对坐标（默认 `0.12,0.20`，避开 Windows
  首次连接时的防火墙弹窗）
- `QD_OUT`：截图与 `e2e-result.json` 的输出目录

退出码 0 表示视频帧真实到达（`videoWidth > 0` 且 `currentTime` 在推进）。

### 直启 Host 做协议联调

```bash
python3 scripts/e2e/qd-host-stdio-harness.py \
  --host /path/to/quickdesk_host \
  --config-dir /tmp/qd-host/config \
  --log-dir /tmp/qd-host/logs \
  --server ws://127.0.0.1:8000 \
  --api-key "$QD_API_KEY"
```

Windows 上应分别验证两种上下文：普通交互用户用于确认 `WTSQueryUserToken` 是否被
`0x522` 拒绝；SYSTEM/session 0 用于确认是否返回 `NOT_LOGGED_IN`。正常服务模式必须在
有权访问活动交互用户 token 的 session 中启动 Host worker。

## 被控端准备（Parallels Windows 虚拟机）

```bash
# 1. 安装上游产物；SYSTEM 上下文可以免 UAC 静默安装
prlctl exec <uuid> --current-user cmd /c "copy /Y \\Mac\Home\...\QuickDesk-win-x64-*-setup.exe C:\QuickDeskTest\setup.exe"
prlctl exec <uuid> cmd /c "C:\QuickDeskTest\setup.exe /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /SP-"

# 2. 把被控端指向自托管信令：写 %APPDATA%\QuickDesk\db\appconfig.db
#    signalingServerUrl=ws://<host>:8000 且 apiKey=<服务端 API_KEY>

# 3. 输入靶子
prlctl exec <uuid> --current-user powershell -File C:\QuickDeskTest\guest-input-target.ps1 -Action show
prlctl exec <uuid> --current-user powershell -File C:\QuickDeskTest\guest-input-target.ps1 -Action focus
```

实测结论与两个容易踩的坑见 `docs/dev/自托管端到端联调记录.md`。
