# Go-Remote 需求与 QuickDesk 能力对照

## 1. 结论

QuickDesk 已经覆盖了远程控制最核心、最难实现的画面、输入、编码、WebRTC、P2P/TURN
和 Windows 用户会话控制。Go-Remote 后续不再自研这部分，而是在 QuickDesk 基础上
补齐业务控制面、录像、屏幕墙、文件管理、审计和自定义扩展。

总体策略：

- 直接继承 QuickDesk 的远程控制和媒体引擎。
- 复用 QuickDesk 的账号、设备、信令、审计和管理后台。
- 在 QuickDesk 上补齐 Go-Remote 原有业务能力。
- 不重新实现采集、编码、WebRTC、P2P、输入注入和 Chromium Remoting。
- 不为普通业务功能编译 Chromium；只有修改 Remoting 内核时才编译。

## 2. 状态定义

| 状态 | 含义 |
|---|---|
| 已有 | QuickDesk 已经实现，Go-Remote 可以直接继承 |
| 可用但要改 | QuickDesk 有基础能力，但需要适配我们的界面、接口或业务流程 |
| 部分 | 只有底层能力或字段，缺少完整产品功能 |
| 缺失 | QuickDesk 当前没有，需要新增 |
| 不做 | 明确不纳入本次改造 |

## 3. 核心能力对照

| 模块 | Go-Remote 需求 | QuickDesk 现状 | 结论 | 改造动作 |
|---|---|---|---|---|
| 用户注册 | 注册、登录、账号管理 | Signaling Server 已有注册、登录、短信、Token | 已有 | 保留，按我们的注册策略配置 |
| 登录鉴权 | JWT/CSRF/会话控制 | access/refresh token、admin token、session revoke | 已有 | 保留 |
| 设备配对 | Pair Code、设备注册 | 9 位 device ID、device secret、自动 provisioning | 已有 | 保留 |
| 设备绑定 | 用户和设备的归属关系 | 用户设备、收藏、解绑、转移、设备配额 | 已有 | 保留 |
| 设备分组 | 分组、筛选、批量操作 | 管理后台 `device_groups` 和设备批量接口 | 已有 | 保留并补用户端入口 |
| 设备在线状态 | 心跳和在线检测 | Redis presence、heartbeat、WS 派生 online | 已有 | 保留 |
| 设备历史 | 最近连接、快速重连 | 连接历史、收藏、最近设备 | 已有 | 保留 |
| 会话创建 | 用户发起远程会话 | 访问码校验、signal token、WebRTC 信令 | 已有 | 保留 |
| 会话生命周期 | 创建、运行、结束、清理 | Client/Host WebRTC 生命周期，连接历史；**设备访问审计已完成**（`/v1/me/devices/:id/access-logs` + `device.access.*` 事件） | 部分 | 统一 session 模型仍待补；访问尝试这一侧已可查、可订阅 |
| Quick Connect | 设备 ID 快速连接 | WebClient 输入 device ID + access code | 已有 | 保留 |
| 视频画面 | 低延迟观看 | Chromium Remoting，H.264/VP8/VP9/AV1，共享内存 | 已有 | 不重写 |
| 硬件编码 | GPU 编码 | Chromium Remoting 各平台编码后端 | 已有 | 不重写 |
| 远程输入 | 鼠标、键盘、滚轮 | Chromium Remoting Host 输入注入 | 已有 | 不重写 |
| 文本剪贴板 | 双向文本同步 | Host/Client 剪贴板同步 | 已有 | 保留并补边界测试 |
| 图片剪贴板 | 图片复制粘贴 | 已专项核对：剪贴板在 Remoting 内核里（`remoting.protocol.ClipboardEvent`、`clipboard_win.cc`），消息体只认 UTF-8 文本，二进制会被丢掉 | 缺失 | 要做得改 Remoting 协议（新增图片格式/分片），按"不为普通业务功能编译 Chromium"暂时搁置 |
| 文件传输 | 文件发送 | 支持远程文件传输 | 已有 | 保留 |
| 文件管理 | 浏览目录、新建、重命名、删除、分片上传下载 | 分片上传/下载已具备（两端）；host 侧 `file-ops` 技能提供浏览/读写/改名/新建，但**只有 Qt 控制端调用得到** | 部分 | 先在 Qt 侧做文件管理器；WebClient 需要先补一条浏览器可用的技能通道 |
| 音频 | 远端声音播放 | WebClient 有音频轨道和播放控制 | 已有 | 保留 |
| 多显示器 | 多屏切换 | 多显示器、虚拟屏 | 已有 | 保留 |
| 隐私屏 | 被控端隐私保护 | Windows 隐私屏 | 已有 | 保留 |
| 锁屏控制 | 锁屏、登录界面输入 | Windows Service + 安全桌面输入 | 已有 | 重点做真机验证 |
| 自动锁屏 | 会话结束后锁屏 | v4.2 已加入远程断开后自动锁屏 | 已有 | 保留 |
| 画质和帧率 | 手动质量、目标 FPS、自动降帧 | 自适应帧率、码率、编码偏好 | 已有 | 保留并补业务策略 |
| 延迟指标 | RTT、捕获/编码/网络/解码/渲染 | 已有延迟分解和实时统计 | 已有 | 保留 |
| 录制 | 会话录像 | 当前缺少完整媒体录像链路 | 缺失 | Host/Client 增加录制引擎 |
| 录像索引 | 查询、删除、归属 | 已新增 `session_recordings` 控制面 API | 完成基础 | 继续接 Host/Client 上传和状态更新 |
| 录像回放 | 在线播放、下载 | 当前没有完整录像回放页面 | 缺失 | Web 和 Qt 增加回放页面 |
| 屏幕墙 | 多设备集中查看、批量操作 | WebClient 已新增 `/wall` 屏幕墙（iframe 复用自己的 remote.html，默认只看不控，可单砖解锁输入） | 完成基础 | 继续补墙级音频、大屏轮巡 |
| MCP | AI 控制和自动化 | 内置 MCP、OCR、40+ 工具、工作流方向 | 已有 | 保留并作为特色能力 |
| 工作流录制回放 | AI 人工流程录制和重放 | 有设计文档，未形成完整实现 | 部分 | 后续按上游方案推进 |
| Web 控制端 | 浏览器访问 | `WebClient` 已有 | 已有 | 品牌化并补我们的功能入口 |
| 管理后台 | 用户、设备、审计、统计 | Go + Vue 管理后台已有 | 已有 | 保留并定制 |
| 审计 | 管理员和业务操作审计 | audit logs、导出、webhook；**已补设备访问事件**（拒绝的访问进审计表，授权/拒绝都推用户实时流） | 已有 | 继续补文件传输、录像等业务事件 |
| 权限角色 | 多角色、租户、权限 | 有 admin 角色，但没有我们的租户模型 | 部分 | 后续按业务需要扩展 |
| 2FA | 管理员双因素 | Admin TOTP 已有 | 已有 | 保留 |
| SSO | 企业单点登录 | 未发现完整接入 | 缺失 | 明确暂不做，后续评估 |
| 通知 | 设备上下线、异常、任务 | 管理端 webhook 已有；**用户级通知通道已完成**（`/v1/me/notification-channels`，订阅访问拒绝/会话开始/设备上下线等事件，HMAC 签名 + 持久 outbox、至少一次投递、单条/按通道/跨通道死信重放、保留策略含先归档再删），管理端有 `notification-deliveries:stats|sweep` | 部分 | 服务端投递链路与运维入口已具备；还缺 WebClient 的通道管理页面与聚合降噪 |
| 设备策略 | 限制设备、会话策略 | 设备配额、管理端 IP 白名单有基础；**设备级访问策略已完成**（`/v1/me/devices/:id/access-policy`：远程访问开关 + 来访 IP/CIDR 白名单 + 允许时段 + 例外日（固定日/区间/年度重复/法定节假日日历/农历重复），在 `access-code:verify` 前置拦截；`max_sessions` 并发上限在控制端 `auth_ok` 之前拦截），**临时授权已完成**（`/v1/me/devices/:id/access-grants`：限时/限次/可撤销的访客码，策略仍优先），**二次审批已完成**（`approval_mode` = off/always/grant/untrusted_ip 决定何时要批，房主侧 `/v1/me/devices/:id/access-approvals` 批准/拒绝，控制端 claim 单次换 token） | 部分 | 还缺策略变更的多端同步；节假日日历只内置 2026，年度更新靠放文件（无自动拉取） |
| 计划任务 | 定时连接、批量运维 | MCP/工作流可扩展，但无完整调度 | 缺失 | 放到 P2 |
| 开放 API | 业务系统集成 | Signaling API 和管理 API 已有 | 部分 | 补稳定的业务 API 和 SDK |
| 插件系统 | 自定义采集、编码、能力 | MCP 和 skill host 可扩展，不是通用插件系统 | 部分 | 先不做通用插件市场 |
| 移动端 | 手机控制 | Android 客户端已有，iOS 未承诺 | 部分 | Android 保留，iOS 暂不做 |
| Linux | Linux 被控端 | README 标注计划中 | 缺失 | Linux 暂不做，Windows/macOS 优先 |
| P2P | 直连降低延迟 | WebRTC ICE，P2P 优先、TURN 回退 | 已有 | 不重写，做自托管压测 |
| TURN | NAT 失败时中继 | coturn 自托管方案 | 已有 | 保留；容量/带宽已压测（第 25 节），还缺长稳与 TLS |
| RDP | 系统远程桌面 | 不依赖 RDP，避免锁屏 | 已有替代 | 不做 RDP |
| 日志聚合 | 集中日志和告警 | 有系统日志、审计和应用日志 | 部分 | 增加聚合、告警和日志面板 |
| Prometheus | 监控指标 | 后端有 metrics，管理后台有统计 | 部分 | 按业务指标扩展 |
| 安装升级 | 安装、自动更新 | 有安装包、发布链路 | 部分 | 增加自有升级通道 |
| 多租户 | 多个组织隔离 | 当前不是多租户模型 | 缺失 | 暂不做 |
| 计费和配额 | 商业计费 | 设备配额存在，计费不存在 | 缺失 | 暂不做 |

## 4. 直接继承

以下能力不再重写：

- Chromium Remoting 采集、编码、解码、输入和 WebRTC。
- P2P 直连和 TURN 回退。
- Windows 隐私屏、安全桌面、虚拟显示器和自动锁屏。
- Go Signaling Server 的账号、设备、Token、presence 和审计。
- WebClient 的远程观看和基础输入。
- MCP、OCR、远程文件和 skill host 基础能力。
- PostgreSQL、Redis、coturn 和 Docker 部署基础。

## 5. 需要在 QuickDesk 上改造

### P0：先把基础稳定和业务差异补上

1. 自托管 HTTPS/WSS 专项修复。
   - 当前 issue 已指出 Remoting 证书校验接口只适合 localhost 开发。
   - 这是公网自托管的关键 blocker。
2. 品牌和产品入口。
   - 应用名、Logo、关于页、Web 标题、管理后台文案。
3. 会话录像控制面。**已完成**：索引 API + 产物/预览存储 + 签名票据 + 生命周期 GC（含孤儿回收）。
4. 录像媒体链路。**已完成（控制端录制）**：Qt 与 Web 两个控制端各自录制自己已解码的画面并上传，
   不做 Host 侧权威录制（那需要 fork `quickdesk-remoting`，见第 6 节）。
5. 录像回放。**已完成**：WebClient 录像列表/播放/下载/删除 + 预览图，管理端跨用户检索与统计。
6. Windows 安装和连接稳定性。
   - 处理 Qt 插件、SSL、设备发现、访问码和连接初始化问题。
7. 上游同步和自有构建。
   - 固定 QuickDesk/Remoting 版本，建立自己的 release 流程。

### P1：补 Go-Remote 原有产品能力

1. 屏幕墙。**已完成**：WebClient `/wall`，支持多设备同屏、单砖放大、轮巡、默认只看不控、
   单砖解锁输入、离线/未登录占位。
2. 完整远程文件管理器。**Qt 侧已完成 / 浏览器端仍缺**：
   - 传输已具备：两个控制端都有分片上传/下载（WebClient 走 `filetransfer-*` 数据通道）；
   - **Qt 控制端已完成**：`RemoteWindow.qml` 的「远程文件」抽屉走 **skill bridge**
     调用 host 侧 `file-ops` 技能，浏览 / 新建 / 重命名 / 预览 / 软删除（建
     `.quickdesk-trash` 再 `move_file`）都已在真机跑通，详见
     `docs/dev/自托管端到端联调记录.md` 第 14 节；
   - **浏览器端目前没有这条路**：skill bridge 是 Qt↔host 的原生消息通道
     （`skillBridgeSend` / `skillBridgeResponse`），不在 WebRTC 数据通道上。
     所以 WebClient 文件管理器要么先补一条浏览器可用的技能通道（涉及 Remoting），
     要么先把文件管理器做在 Qt 侧。
3. 图片剪贴板。**已专项核对，暂缓**：剪贴板在 Remoting 内核里且只认 UTF-8 文本，
   要做图片必须改 Remoting 协议（见第 3 节）。
4. 业务审计事件。**已完成**：设备访问尝试/真实会话/策略拦截都进审计表与访问时间线，
   拒绝类事件同时进管理员审计。
5. 用户通知和 webhook 模板。**服务端已完成**：`/v1/me/notification-channels` 用户级通道
   （HMAC 签名 + `last_status`），**投递 outbox 已完成**：持久队列、至少一次投递、
   多 worker 领取互斥（`SKIP LOCKED`，有并发单测）、单条/批量死信重放、保留/TTL
   （见第 21-24 节）；还缺 WebClient 管理页面与聚合降噪。
6. 设备策略和会话策略。**已完成设备侧**：远程开关、来访 IP/CIDR 白名单、控制端并发上限、
   允许时段（IANA 时区 + 跨午夜窗口）、**一次性例外日（节假日 / 临时加班，见第 26 节）**；
   例外日支持区间与年度重复（见第 27 节）、**临时授权（限时/限次/可撤销访客码，见第 32 节）**；
   **二次审批（见第 35 节）**：策略打开 `approval_required` 后，正确验证码只发 202 待审批，
   房主 `approve/deny`，控制端 `access-approvals:claim` 单次兑换 signal_token（真机已验证）。
   **节假日日历与农历重复（见第 36 节）**：例外日可直接写 `{"holiday":"春节","mode":"deny"}`
   （内嵌 2026 法定安排 + `HOLIDAY_CALENDAR_DIR` 年度覆盖，能区分放假与调休上班），
   或用 `{"repeat":"lunar","lunar_month":8,"lunar_day":15}` 按农历自动换算（真机已验证）。
   **审批触发条件细化（见第 37 节）**：`approval_mode` 支持"仅访客码要批""仅陌生 IP 要批"，
   房主自己的设备码不再被打扰（真机已验证）。
   还缺节假日日历的自动更新。
7. WebClient 产品化定制。
8. 长稳、并发和 TURN 容量压测。**并发压测台已完成**（`scripts/e2e/signal-load.mjs`：
   突发/轮转两种模式，含并发上限断言与连接延迟分位，见第 19 节）；**TURN 容量与中继质量也已压测**
   （`scripts/e2e/turn-capacity.mjs`：800 并发 allocation 零失败零泄漏、1200B 中继零丢包，见第 25 节），
   **长稳与带宽上限也测了**（300s soak 零丢包 9/9 续期；限速在 coturn `max-bps`，见第 28 节）；
   **TURN over TLS 已打通**（`turns:` 全链路 + 信令凭证，见第 29 节）；还缺小时级 soak、DTLS 与真实证书。

### P2：业务扩展

1. 计划任务。
2. 开放 API 和 SDK。
3. 业务系统集成。
4. 集中日志和告警。
5. 自有自动更新。

## 6. 明确不做

- 不做 RDP，避免被控端本地会话被锁定。
- 不重写 Chromium Remoting。
- 不重写 Windows 屏幕采集、输入注入和硬件编码。
- 不重写 WebRTC、ICE、STUN、TURN。
- 不继续维护旧 Go-Remote 的 Control/Relay/Agent 媒体栈。
- 不为普通 UI 或业务功能编译整个 Chromium。
- 当前不做 iOS。
- 当前不做 Linux 被控端。
- 当前不做多租户、SSO、计费和通用插件市场。

## 7. 开发顺序

```text
P0
  1. 修复自托管 HTTPS/WSS
  2. 品牌和自有构建
  3. 录像元数据 API
  4. 录像媒体引擎
  5. Web 回放页面
  6. Windows 稳定性验证

P1
  1. 屏幕墙
  2. 文件管理器
  3. 图片剪贴板
  4. 业务审计和通知
  5. 压力和长稳测试

P2
  1. 计划任务
  2. 开放 API/SDK
  3. 日志聚合和告警
  4. 自动更新
```

## 8. 风险

- QuickDesk 维护者很少，上游更新节奏可能变慢。
- Remoting 内核修改需要编译整个 Chromium，成本很高。
- 自托管 TLS 问题目前没有看到完整修复。
- 没有公开的并发压测和容量数据。
- 需要尽快建立自己的上游同步、测试和发布流程。
