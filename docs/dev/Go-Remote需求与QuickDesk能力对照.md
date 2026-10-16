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
| 录像索引 | 查询、删除、归属 | `session_recordings` 控制面 API 已完整：增删改查 + 归属 + 游标分页、媒体字节上传（整包 `POST` 与**分块断点续传 `PATCH`**，见第 90 节）、播放票据、缩略图、保留/清理生命周期（`RECORDING_RETENTION_DAYS` / `RECORDING_STALE_MINUTES` / 孤儿与**弃用暂存文件**回收） | 基本完成 | 还差**被控端/控制端自动上传**：目前媒体字节由客户端主动推（Web 用 MediaRecorder + Qt 用 DeviceRecorder），Host 侧无自动上传；断点续传也还没有客户端在 UI 上用起来 |
| 录像回放 | 在线播放、下载 | 当前没有完整录像回放页面 | 缺失 | Web 和 Qt 增加回放页面 |
| 屏幕墙 | 多设备集中查看、批量操作 | WebClient 已新增 `/wall` 屏幕墙（iframe 复用自己的 remote.html，默认只看不控，可单砖解锁输入） | 完成基础 | 继续补墙级音频、大屏轮巡 |
| MCP | AI 控制和自动化 | 内置 MCP、OCR、40+ 工具、工作流方向 | 已有 | 保留并作为特色能力 |
| 工作流录制回放 | AI 人工流程录制和重放 | 有设计文档，未形成完整实现 | 部分 | 后续按上游方案推进 |
| Web 控制端 | 浏览器访问 | `WebClient` 已有 | 已有 | 品牌化并补我们的功能入口 |
| 管理后台 | 用户、设备、审计、统计 | Go + Vue 管理后台已有 | 已有 | 保留并定制 |
| 审计 | 管理员和业务操作审计 | audit logs、导出、webhook；**已补设备访问事件**（拒绝的访问进审计表，授权/拒绝都推用户实时流）；**已补用户侧业务事件与 actor 归属**（`actor_user_id` 与 `admin_id` 两列区分"哪个用户"和"哪个管理员"，用户删除录像现在写 `recording.delete` 且可归属，见第 47 节）；**审计列表可筛选、导出与来源 IP 已补齐**（`?actor_user_id=` 按用户筛选、非数字 400，用户侧审计行记录 actor 的 `c.ClientIP()`，CSV 导出新增 `actor_user_id` 列，见第 54 节）；**录像媒体读取已审计**（用户侧认证路由与票据路由都写 `recording.download`/`recording.thumbnail.download`，按 `(actor, action, 录像)` 5 分钟去重以免被 range 请求刷屏，见第 55 节）；**保留/TTL 已补齐**（`AUDIT_RETENTION_DAYS` 默认 0=永久保留，`AUDIT_ARCHIVE_DIR` 时先归档再删、归档失败不删任何行，管理端 `POST /v1/admin/audit-logs:sweep` 可手动清理并写 `audit.sweep`，见第 56 节）；**归档可查询**（`GET /v1/admin/audit-archive:search` 按天读回保留期写入的 JSONL，支持 `q`/`action`/`admin`/`actor_user_id`/`limit` 过滤、反向扫描带 32 MiB 预算、`truncated` 与 `has_more` 如实上报，见第 64 节）；**行数封顶已完成**（`AUDIT_MAX_ROWS` 默认 0=不封顶，与时间窗相互独立、任一开启即跑 sweep，裁最旧的行且归档优先，见第 65 节）；**按 actor/设备/action 三个维度封顶已完成**（`AUDIT_MAX_ROWS_PER_ACTOR` + `AUDIT_MAX_ROWS_PER_DEVICE` + `AUDIT_MAX_ROWS_PER_ACTION` 默认均为 0=不封顶，定向规则先于全表封顶运行，仅裁旧行、归档优先、可经 sweep 接口按需执行，见第 76/83 节）；**归档目录可轮转且可进冷存**（`AUDIT_ARCHIVE_RETENTION_DAYS` 默认 0=永久保留；未配冷存时到期删除，配 `AUDIT_ARCHIVE_COLD_DIR` 时移动到冷存目录、移动失败则热文件保留并报错，见第 68/85 节） | 已有 | 文件传输在控制面**没有任何信号**（走 Remoting 数据通道，不经信令），要审计需先让客户端上报；没有按用户名筛选（表里不存用户名）；保留策略的时间窗仍是全局一个（没有按 actor 的时间窗），归档是 JSONL 文件、已可查询、已可轮转，行数已可按全表、actor、device、action 四个维度封顶（仅裁旧行、归档优先、可经 sweep 接口按需执行，见第 65/76/83 节），但没有按字节封顶（Postgres 不 VACUUM 不还空间，字节阈值会失真） |
| 权限角色 | 多角色、租户、权限 | 有 admin 角色，但没有我们的租户模型 | 部分 | 后续按业务需要扩展 |
| 2FA | 管理员双因素 | Admin TOTP 已有 | 已有 | 保留 |
| SSO | 企业单点登录 | 未发现完整接入 | 缺失 | 明确暂不做，后续评估 |
| 通知 | 设备上下线、异常、任务 | 管理端 webhook 已有；**用户级通知通道已完成**（`/v1/me/notification-channels`，订阅访问拒绝/会话开始/设备上下线等事件，HMAC 签名 + 持久 outbox、至少一次投递、接收端幂等键（`X-Webhook-Delivery-Id` / `X-Webhook-Event-Id`，见第 51 节）、投递公平（按通道份额领取，第 43 节）、单条/按通道/跨通道死信重放（可按事件类型/设备/通道筛选、按入队时间的闭区间窗口、支持 dry-run、批量上限与重投限速，见第 40/43/46 节）、保留策略含先归档再删、归档目录也可轮转且可进冷存（`NOTIFICATION_ARCHIVE_RETENTION_DAYS` 默认 0=永久；配 `NOTIFICATION_ARCHIVE_COLD_DIR` 时移动到冷存目录，未配时才删除 `notification-deliveries-YYYY-MM-DD.jsonl` 形状，同目录下的审计归档不受影响，见第 69/85 节），管理端有 `notification-deliveries:stats|sweep`；**聚合降噪已完成**（`NOTIFICATION_AGGREGATE_SECONDS` 默认 0=关闭，开启后按 `(通道, 事件类型, 设备)` 在持有窗口内合并为一行、body 带 `aggregated_count`，(窗口固定所以最坏延迟就是窗口本身，见第 57 节）；**投递确认闭环已完成**（`POST /v1/me/notification-channels/:id/deliveries/:deliveryId/ack` 记录人工确认 `acked_at`（与 `delivered_at` 并列，2xx 不等于有人看见）：首次 ack 定时间戳、之后幂等返回同一行，未终态 `pending`/`delivering` 是 409，跨通道是 404；列表回 `unacked`、管理端 stats 回 `acked`/`unacked`，见第 92 节）；**告警静默已完成**（`/v1/me/notification-quiet-hours`：`snoozed_until` 一次性静默 + 最多 20 条周期性窗口（IANA 时区），在投递 worker 侧生效——静默期内到期的行不投递、回落 `pending` 并打上 `held_until`，`attempts`/`last_status` 不动（静默不是一次失败尝试）；改设置会同步挪动被压住的行，清空即刻放行；只有静默压住的行被处理，退避中的行不受影响，见第 93 节）；**设备离线持续告警已完成**（`/v1/me/devices/:id/offline-alert`，阈值 1 分钟~7 天，越过阈值发 `device.offline.sustained`，恢复后自动重新武装，支持 `repeat_minutes` 重复提醒、`snoozed_until` 一次性静默与 `maintenance_windows` 周期性维护窗口，见第 44/52 节） | 部分 | 投递链路、运维入口、第一类规则（离线时长，含提醒/静默/周期窗口）、投递确认闭环与告警静默已具备；还缺指标型规则（CPU 等，需被控端上报），WebClient 的通道/规则管理页面仍缺 |
| 设备策略 | 限制设备、会话策略 | 设备配额、管理端 IP 白名单有基础；**设备级访问策略已完成**（`/v1/me/devices/:id/access-policy`：远程访问开关 + 来访 IP/CIDR 白名单 + 允许时段 + 例外日（固定日/区间/年度重复/法定节假日日历/农历重复），在 `access-code:verify` 前置拦截；`max_sessions` 并发上限在控制端 `auth_ok` 之前拦截），**临时授权已完成**（`/v1/me/devices/:id/access-grants`：限时/限次/可撤销的访客码 + `bound_client_id`/`bound_ips` 绑定 + 记录实际兑换者，策略仍优先），**二次审批已完成**（`approval_mode` = off/always/grant/untrusted_ip 决定何时要批，房主侧 `/v1/me/devices/:id/access-approvals` 批准/拒绝，控制端 claim 单次换 token），**审批队列已有闸门与冷却期**（同一来源已有等待中的请求即 429 + `Retry-After`，每设备未决定请求上限 5 条；被房主拒绝后同一 `(设备, 来源 IP)` 在 `APPROVAL_DENIAL_COOLDOWN_SECONDS`（默认 300）内一律 429，见第 49/61 节），**拒绝冷却可递增**（可选 `APPROVAL_DENIAL_COOLDOWN_MULTIPLIER`（默认 1 = 窗口恒定）与 `APPROVAL_DENIAL_COOLDOWN_MAX_SECONDS`（默认 0 = 不封顶）：连拒第 k 次窗口为 `基础 × multiplier^(k-1)`，一次批准或一次过期即清零，summary 回显两个设置，见第 89 节），**拒绝可见性已完成**（`GET /v1/me/devices/:id/access-approvals:summary`：按 `created_at` 窗口统计五状态计数，并把被拒来源按 `client_ip` 聚合、按拒绝次数与最近时间排序，回显线上真实冷却秒数，见第 63 节），**可信代理已可配置**（`TRUSTED_PROXIES`，默认只信 loopback 与私网；非可信对端的 `X-Forwarded-For` 被忽略，见第 50 节）；**多级代理取值已实测**（从右往左跳过可信跳、取最右的**非可信**跳；客户端前置伪造地址不会改变归属 IP，整链可信时取最左，见第 74 节） | 部分 | 还缺策略变更的多端同步；节假日日历只内置 2026，年度更新靠放文件（无自动拉取）；`bound_client_id` 只是客户端自述，真正的身份证明未做；summary 只有计数没有逐条明细与实时推送 |
| 计划任务 | 定时连接、批量运维 | **控制面调度框架已完成**（`/v1/me/scheduled-tasks`：`kind` + `run_at` + `repeat_seconds`，注册制的动作集合，`FOR UPDATE SKIP LOCKED` 领取且同一事务内推进排期保证多实例不重复执行，一次性任务执行后停用并保留结果字段，每次执行写 `scheduled_task.run` 审计；首个 kind `sign_out_all_sessions` 已在真机验证：到期后 refresh token 立即 `401 REFRESH_INVALID`，见第 60 节）；**失败重试已完成**（`max_attempts` 默认 1=不重试、允许 1~10；失败且预算未耗尽时按 30s/1m/2m/4m/8m/10m 退避重排，重排不会晚于该周期末；`attempts` 成功清零、一次性任务停用时保留，审计行带 `attempt=n/m`，见第 70 节）；**最终失败会告警**（预算耗尽时发布用户事件 `scheduled_task.failed`，带 task_id/kind/attempts/error，同时进 owner 的实时流与订阅了它的通知通道；每次重试**不**发事件，避免训练出"忽略告警"，见第 71 节）；**日历式排期已完成**（`schedule` 显式 DSL：`daily@HH:MM`、`weekly@<dow>@HH:MM`、`monthly@<dom>@HH:MM` + IANA `timezone`，首次触发与后续触发都按该时区的墙钟推出；`schedule` 与 `run_at`/`repeat_seconds` 互斥；`monthly@31` 按月夹取到月末；DST 跳过的墙钟时间按"缺口后第一个真实时刻"执行；解析不了的 schedule 会停车而不是每次 tick 重领；`GET` 增加 `supported_schedules`，见第 75 节） | 部分 | "定时连接"需要控制端到点发起连接、"批量运维"需要被控端执行指令并回执——两者都在客户端/agent 侧（受上游 PR #23 影响）；kind 目前只有定时全端下线；日历 DSL 没有"每隔 N 天/工作日"与节假日跳过 |
| 开放 API | 业务系统集成 | Signaling API 和管理 API 已有；**已有机器可校验的业务契约**（`SignalingServer/api/openapi.yaml`，OpenAPI 3.1，22 个 path / 30 个 schema：登录、当前用户、设备列表、验证码校验、访问策略读写、访客授权码增删查、审批列表/拒绝汇总/批准/拒绝/claim（成功路径见第 80/86 节冒烟）、离线告警规则读写、通知通道、录像索引（游标分页的 `GET /v1/me/recordings`）与录像全套（索引 `GET|POST /v1/me/recordings`、单条 `GET|PATCH|DELETE /v1/me/recordings/:id`、上传 `POST .../artifact|thumbnail` 的 multipart 与 `?ext=` 两种形状、断点续传 `PATCH .../artifact`（`Upload-Offset`/`Upload-Length`/`Upload-Complete`，第 90 节）、`GET .../artifact|thumbnail`、`POST .../artifact-ticket|thumbnail-ticket`、票据路由 `GET /v1/recordings/:id/artifact|thumbnail?t=`）；`scripts/sdk/openapi-smoke.mjs` 对运行中的服务逐项断言文档化字段，只读 136 条、含写操作 213~298 条（断言数随可选拒绝腿、主机掉线腿和账号残留数据浮动），另有实时冒烟 30~39 条，见第 59/62/63/66/67/72/79/80/81/86/87/88/89 节）；**已提交生成好的 SDK（两种语言）+ 已接 CI**（`sdk/ts` 是 TypeScript 客户端、`sdk/go` 是 Go 客户端（含唯一手写的 `examples/`：CI 在冒烟 job 里真跑一次登录+设备列表），两者都由 `scripts/sdk/generate-client.sh <目录> [语言]` 用钉死版本的 openapi-generator jar 生成，`scripts/sdk/check-generated-sdk.sh [语言]` 重新生成并逐文件比对，`.github/workflows/signaling-server-verify.yml` 三个 job：Go vet/test（真 Postgres+Redis）、契约与两个 SDK 的新鲜度+编译+vet、以及“起服务→播种账号与设备→跑冒烟（HTTP/实时/Go 客户端）”的 live 冒烟，见第 78/91 节） | 部分 | SDK 覆盖 TypeScript 与 Go 两种语言，Python/Java 等仍未提供；Go 客户端目前只有 `examples/` 在用，没有业务级示例脚本；**实时 WebSocket 已进契约并被实测**（`SignalingServer/api/asyncapi.json`，AsyncAPI 3.0 用 JSON 序列化——冒烟要读它来驱动断言，仓库不想为此引入 YAML 依赖：两条 WS 的地址、首帧认证帧、`auth_ok`/`snapshot_required`/`snapshot`/`bootstrap_done` 顺序、事件信封、事件类型目录、错误码；`scripts/sdk/realtime-smoke.mjs` 真连一次并按契约文件断言 39 条，见第 81 节）；审批的两条触发均已由冒烟覆盖：device_code（approve→claim 换到 signal_token、重复 claim 409、拒绝后 claim 403，第 80 节）与 grant（待批列表 `source=grant`、批准后 claim 才消耗单次授权，第 86 节）；审批队列上限与拒绝冷却也已进冒烟（第 87 节）；主机在审批期间掉线的 claim 也已由冒烟覆盖（第 88 节）；signal 通道只钉了握手与错误形状，Jingle 载荷是端到端透传、不在契约里 |
| 插件系统 | 自定义采集、编码、能力 | MCP 和 skill host 可扩展，不是通用插件系统 | 部分 | 先不做通用插件市场 |
| 移动端 | 手机控制 | Android 客户端已有，iOS 未承诺 | 部分 | Android 保留，iOS 暂不做 |
| Linux | Linux 被控端 | README 标注计划中 | 缺失 | Linux 暂不做，Windows/macOS 优先 |
| P2P | 直连降低延迟 | WebRTC ICE，P2P 优先、TURN 回退 | 已有 | 不重写，做自托管压测 |
| TURN | NAT 失败时中继 | coturn 自托管方案 | 已有 | 保留；容量/带宽/TLS/**DTLS**/**CA 证书轮换**/**小时级 soak（UDP、TLS 与 DTLS）**/**ChannelData 路径**/**多实例与中继端口池耗尽** 都已实测（第 25、28、29、39、41、42、45、58 节；DTLS 小时级为 3605s、7200/7200 包、0 丢包、247 次 allocation refresh），还缺公网/NAT 实测、公网 CA 自动续期与多天回归 |
| RDP | 系统远程桌面 | 不依赖 RDP，避免锁屏 | 已有替代 | 不做 RDP |
| 日志聚合 | 集中日志和告警 | 有系统日志、审计和应用日志；**设备离线告警规则已落地**（第 44 节）；**日志可在服务器上就地检索**（`GET /v1/admin/system/logs:search`：从文件尾部逆序扫描，支持 `q` / `component` 整词 / `since`+`until` 闭区间 / `limit` / `file=` 选择来源，8 MiB 预算超了会报 `truncated`，检索本身写审计 `log.search`；配置 `POSTGRES_LOG_FILE`/`REDIS_LOG_FILE` 后数据库与缓存日志同样可搜，见第 48/53 节）；**可一次搜全部来源**（`file=all` 按注册顺序扫每个已配置来源、共享同一份 8 MiB 预算，超预算仍返回已找到的条目并报 `truncated`，每条结果带 `file` 标注来源，见第 73 节）；**跨实例检索已完成**（`INSTANCE_LOG_SINK=true` 时每个实例把日志行投递到共享表 `instance_log_entries`，`GET /v1/admin/system/logs:search?scope=all` 合并所有实例、每条带 `instance`/`source`，`instance=` 可收窄，响应带 `instances` 与 `stored_lines`/`dropped_lines` 计量；投递不阻塞不报错、重发幂等、`INSTANCE_LOG_RETENTION_DAYS` 默认 7 天封顶，单实例默认关闭且关闭时 `scope=all` 明确 400，见第 77 节）；**管理端日志面板已完成**（`/admin/#/system-logs` 的 scope 本机/聚合切换、来源/子串/component/时间窗/limit/instance 过滤、结果计量、文件列表与下载，聚合未开启时会显示 400 原因而不是空表，见第 82 节） | 部分 | 还不是集中式日志栈（无统一查询语言/全文索引，`q` 仍是 ILIKE 子串）、没有指标型告警（CPU/内存等，依赖被控端上报）；跨实例只覆盖应用日志，Postgres/Redis 来源不投递 |
| Prometheus | 监控指标 | 后端有 metrics，管理后台有统计；**Prometheus 文本端点已完成**（`METRICS_TOKEN` 非空时开放 `/metrics`，必须用 `Authorization: Bearer`，未设置默认 404；暴露构建信息、进程运行时长、API 请求总数/当日数、WebSocket/signal/event 连接数、goroutine 与 Go 内存指标，见第 84 节） | 部分 | 仍是内存内指标，进程重启归零；没有按用户/设备/业务动作维度的持久化指标和告警规则 |
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
   拒绝类事件同时进管理员审计；**用户侧动作也补上了 actor**（见第 47 节：`admin_id` 与
   `actor_user_id` 两列恰好一列有值，用户删除录像写 `recording.delete`，与管理员同一动作名但可区分）。
   文件传输不在其中——它在控制面没有任何信号。
5. 用户通知和 webhook 模板。**服务端已完成**：`/v1/me/notification-channels` 用户级通道
   （HMAC 签名 + `last_status`），**投递 outbox 已完成**：持久队列、至少一次投递、
   多 worker 领取互斥（`SKIP LOCKED`，有并发单测）、单条/批量死信重放、保留/TTL
   （见第 21-24 节）；**重投运维也已补齐**（见第 40 节：按事件类型/设备筛选、`limit` 按最老优先、
   `dry_run` 预演）；**投递公平与限速也已补齐**（见第 43 节：两阶段按通道公平领取，杜绝一条通道
   的积压饿死其他通道；`spread_over_minutes` 把批量重投拆成约 20 个桶错开发送，端点刚恢复不会再被
   整个积压砸一遍）；**重投的作用域也补齐了**（见第 46 节：`channel_ids` 只动指定通道，
   `created_after` / `created_before` 按入队时间取闭区间窗口，窗口可回答"只重投最近一小时入队的积压"）；
   **第一类告警规则也已落地**（见第 44 节：`/v1/me/devices/:id/offline-alert`
   按"离线超过 N 分钟"发 `device.offline.sustained`，一次掉线只发一条、恢复后重新武装，可选
   `repeat_minutes` 重复提醒与 `snoozed_until` 静默窗口，投递复用上面的 outbox）；还缺 WebClient
   管理页面、指标型规则（CPU 等，需被控端上报）、周期性维护窗口与聚合降噪。
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
   **授权码绑定访客（见第 38 节）**：授权码可绑 `bound_client_id` 与 `bound_ips`
   （IP 侧是服务端实测地址），并记录 `last_used_client_id`/`last_used_ip`（真机已验证）。
   还缺节假日日历的自动更新。
7. WebClient 产品化定制。
8. 长稳、并发和 TURN 容量压测。**并发压测台已完成**（`scripts/e2e/signal-load.mjs`：
   突发/轮转两种模式，含并发上限断言与连接延迟分位，见第 19 节）；**TURN 容量与中继质量也已压测**
   （`scripts/e2e/turn-capacity.mjs`：800 并发 allocation 零失败零泄漏、1200B 中继零丢包，见第 25 节），
   **长稳与带宽上限也测了**（300s soak 零丢包 9/9 续期；限速在 coturn `max-bps`，见第 28 节）；
   **TURN over TLS 已打通**（`turns:` 全链路 + 信令凭证，见第 29 节）；
   **DTLS 与 CA 证书轮换已打通**（见第 39 节：coturn 4.18 需显式 `dtls`，
   本机 CA 签发 + 真校验 + 换 leaf 客户端无需改动、换 CA 立即被发现）；
   **小时级 soak 已完成**（见第 41 节：3600s、328196 包 / 394MB、0 丢包、119/119 续期与 permission 重发、
   allocation 计数归零；这次长跑还暴露并修掉了 harness 不刷新 permission 导致的 300 秒假通过）。
   **TLS 上的小时级 soak 与 ChannelData 路径也已完成**（见第 42 节：3600s TLS 0 丢包；
   ChannelBind + ChannelData 分帧在 UDP/TLS 上实测，700s soak 跨过 600 秒绑定寿命）。
   还缺 DTLS 上的小时级 soak、公网 CA 自动续期与多天回归。

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
