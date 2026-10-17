# QuickDesk 功能总清单

> 最后对账：2026-10-17 18:20 CST。上游与 fork 状态都是动态的，实施前必须重新执行
> `git fetch --all --prune` 并核对 GitHub issue / PR。
>
> 本文件是 QuickDesk fork 后续产品开发的唯一功能清单。它合并三类输入：
> 1. QuickDesk fork 当前代码、测试和已验收能力；
> 2. 旧 Go-Remote 仍值得迁移的业务语义和遗留缺口；
> 3. `barry-ran/QuickDesk` 的 Roadmap、open issue、open PR 和维护者贡献口径。
>
> 旧 Go-Remote 仓库不再作为实现基线；某个能力是否需要做，以本清单的当前状态、上游映射和验收标准为准。

## 1. 状态与规则

| 状态 | 含义 |
|---|---|
| `[x]` | 已完成，代码已落地并有测试、构建或真机验证证据 |
| `[~]` | 部分完成，底层已具备但产品入口、跨端消费或边界验证仍缺 |
| `[ ]` | 未开始 |
| `[!]` | 被外部依赖或协议改造阻塞 |
| `[>]` | 上游 Roadmap 已列出，本 fork 当前暂缓 |
| `[-]` | 明确不做 |

需求来源：

| 来源 | 含义 |
|---|---|
| `Fork` | 当前 QuickDesk fork 自己必须完成的产品、部署或质量问题 |
| `Legacy` | 旧 Go-Remote 有、当前主线还缺或有明显差距的业务能力 |
| `Upstream` | 上游 Roadmap、issue 或 PR 已明确提出的能力 |

维护规则：

- 功能完成时同步更新状态、验收标准和证据路径。
- 不以“底层已经有接口”等同于“产品已完成”；必须把用户入口、错误处理和验证一起收口。
- 核心媒体与输入不重写，继续复用 Chromium Remoting、WebRTC、ICE、STUN、TURN。
- Windows 和 macOS 当前优先；Linux、iOS/Android 属于上游 Roadmap，本 fork 暂缓但不删除贡献接口；RDP 明确不做。
- 每项功能都要判断上游贡献方式：`UP-PR`、`UP-Ready`、`UP-Remoting`、`Fork-only` 或 `Upstream-Open`。
- 上游已有 open PR 的功能先同步审查，不重复提交；上游合并后把 `upstream/dev` 合回 fork，再继续增量。
- 任何新功能先写失败测试，再实现，最后补构建或端到端验证。

## 2. 当前主线

- 仓库：`go-remote-next`
- 本地主分支：`dev`、`master`
- 本地功能基线：`83d8caf`；`dev` 与 `master` 指向同一提交
- 远端 fork：`origin/dev` 仍为 `f6651f0`，本地 `dev` 领先 232 个提交；远端写入恢复前不能把本地验收当成已发布
- 上游基线：`barry-ran/QuickDesk` 的 `dev` 和 `master` 当前均为 `f6651f0`
- 上游维护口径：持续维护但暂缓更新，下一阶段重点偏 MCP；外部贡献从 `dev` 切分支，完成后合入 `dev`，维护者定期合入 `master`
- Remoting 边界：`quickdesk-remoting` 需要编译整个 Chromium，涉及 Remoting 的改动优先向上游提需求，不要求贡献者本地编译
- 当前主要增量：QD-215 Windows 发布依赖校验（`67bf11f`、`556f86e`）、QD-607 批量远控清场（`5028068`）、QD-204 地址簿与设备分组
  （`f270487`、`b1a1155`、`162ac6e`、`b013a62`）、QD-508 Qt 录像回放
  （`74c7022`）、三端管理面实时消费，以及 `eca3c8f` 的上游待办融合文档
- fork PR：<https://github.com/ilh-github/go-remote-next/pull/1>，OPEN / MERGEABLE /
  CLEAN，7 项检查全部成功；它的 head 是早期最小分支，不等于当前完整 `dev`
- 服务端：Go SignalingServer
- 桌面客户端：Qt 6 + QML
- Web 客户端：Vue 3 外壳 + 原生 `remote.html`
- 核心控制引擎：QuickDesk / Chromium Remoting
- 持久化：PostgreSQL
- 实时状态与令牌：Redis

## 3. 当前迭代

| ID | 优先级 | 功能 | 状态 | 上游 | 验收标准 |
|---|---|---|---|---|---|
| QD-001 | P0 | 建立独立功能总清单 | `[x]` | Fork-only | 文件覆盖全部范围、状态、优先级和验收口径 |
| QD-002 | P0 | Web 控制端完整文件管理器 | `[!]` | UP-Remoting | 目录浏览、新建、重命名、软删除、上传下载；需要先补浏览器可用的 host 文件操作通道 |
| QD-003 | P0 | 多标签控制与音频抢占 | `[x]` | UP-Ready | 同一设备多个同源标签页只有一个控制者/音频播放者；新标签接管后旧标签立即释放 |
| QD-004 | P0 | Host/Relay 权威自动录制 | `[ ]` | UP-Remoting | 被控端或中继自动录制，断线自动收尾并上传索引 |
| QD-005 | P0 | 客户端自动更新消费 | `[~]` | UP-PR#25 + Upstream-Open | 客户端检查、下载、校验、安装和重启闭环；Windows 静默安装与 macOS DMG 打开已实现，真机安装/重启验收待补 |
| QD-006 | P1 | SQLite 轻量服务端部署 | `[ ]` | Upstream-Open | 单进程、单文件数据库、无 Redis 的最小自托管模式 |
| QD-007 | P1 | 图片剪贴板 | `[ ]` | UP-Remoting | 双向图片格式协商、分片传输、权限和边界测试 |

最近验证：

- 本地基线：`dev` / `master` = `83d8caf`，相对 `origin/dev` 领先 232 个提交；
  QD-215 主线和上游最小候选均已提交，远端写入仍被策略拒绝。
- 上游核对：`upstream/dev` 与 `upstream/master` 均为 `f6651f0`；当前上游 open issues 10 个、open PR 4 个。
- 上游维护：issue #21 已由维护者回复“持续维护、暂缓更新、重点偏 MCP、欢迎从
  `dev` 切分支贡献、Remoting 改动由维护者处理”；不能再把“暂缓”理解成停止维护。
- 上游 PR：#22、#23、#24、#25 均为 OPEN / MERGEABLE，但 GitHub 返回的
  `mergeStateStatus=UNSTABLE` 且没有 reported checks；不能把 MERGEABLE 写成 CI 已通过。
- 清单统计：116 个唯一 QD ID；70 个 `[x]`、22 个 `[~]`、19 个 `[ ]`、
  2 个 `[!]`、2 个 `[>]`、1 个 `[-]`。
- 上游待办已全部映射到本清单：10 个 open issue、4 个 open PR、3 个未完成 Roadmap
  项均有对应 QD、处理边界和上游贡献方式；上游已完成的 Roadmap 项只作为继承基线，
  不重复开发。
- 融合结论：本清单的 3.1 决策台账负责固定“做/不做/等待/交 Remoting”，3.2 执行队列负责
  排序，第 16、17 节负责逐项追溯上游来源和 PR 交付方式。三者共同作为后续开发和回贡的
  唯一工作口径，不再另建一套 fork 待办。
- 上游 `docs/dev/*技术方案.md` 里有部分历史未勾选实现清单，抽查当前代码后多数已经落地；
  这些文件是设计历史，不直接当新增待办。权威待办仍以 README Roadmap、open issue、open PR
  和维护者最新回复为准。
- WebClient `npm test`：34/34 通过。
- WebClient `npm run build`：通过，JS 362.03 kB，gzip 116.68 kB。
- Chrome 同源双标签实测：后开标签接管控制后，前一个标签释放控制；接管音频后，前一个标签释放音频。
- QD-114 SPKI 纯 C++ 单测：通过；覆盖标准/URL-safe Base64、无 padding、重复项、
  非法项和 Chromium 参数构造。
- QD-114 上游候选：`codex/upstream-qd-114-spki` = `13559e6`，基于 `upstream/dev`，
  17 个文件；fork 侧 Qt 6.8.3 构建环境已可用，但该上游候选自身的完整 Qt 构建、
  Windows 安装器编译和服务回归待补。
- QD-005 功能提交：`cae77ff`。Qt 客户端已有启动后自动检查和每 6 小时复查、手动检查、
  更新提示、下载进度/取消、大小与 SHA-256 校验、可选版本推迟、强制更新、同源 API key
  下载和安装器启动；非同一 origin 的下载不会泄漏 API key。
- QD-005 测试：`quickdesk_update_support_test` 与 `quickdesk_update_manager_test` 通过；
  后者使用真实本地 HTTP 服务覆盖检查、下载、摘要不匹配拒绝、残件清理和跨源凭据隔离。
- QuickDesk 完整 Qt 6.8.3 Debug 构建通过，`QuickDesk_qmllint` 退出 0；现有旧 QML
  未解析类型和布局告警仍在，本轮新增更新对话框没有阻断告警。
- QD-212 功能提交：`4e8ea5a`。Qt 我的设备与收藏列表会用当前 Host 设备 ID 标记
  “本机”，同时保留备注/设备名；空设备 ID 或不匹配时不会误标。
- QD-212 上游候选：`codex/upstream-qd-212-local-device` = `f1ce096`，基于
  `upstream/dev`；在干净上游基线上完整构建通过，QML 测试 1/1 通过。
- QD-213 功能提交：`9a2608b`。Qt 点击设备时先读最新 `access_code`（再退到收藏密码），
  为空时按单设备回源并自动继续连接；并发点击同一设备只发一次请求，失败显示真实错误。
- QD-213 上游候选：`codex/upstream-qd-213-access-code-refresh` = `6928d24`，基于
  `upstream/dev`；干净基线完整构建、设备管理集成测试 1/1、QML lint 均通过。
- QD-214 功能提交：`cea23ce`。WebClient 不再只显示 `Host error type=2`，会解释为宿主
  保存失败并提示权限/目标目录/磁盘空间；上传取消统一使用协议取消值 1。
- QD-214 上游候选：`codex/upstream-qd-214-file-transfer-errors` = `94d3ec3`，基于
  `upstream/dev`；WebClient 测试 4/4、生产构建通过。该候选改善诊断，不代表上传根因已修。
- QD-108 功能提交：`7c23f13`。Qt 新增“访问审批”页，支持按设备查看待审批/已决定/
  过期/已兑换/全部、五状态汇总、重复拒绝来源、批准/拒绝原因、15 秒断线兜底刷新，
  并消费 `device.access_approval.requested/decided` 实时事件。
- QD-108 测试：`quickdesk_access_approval_manager_test` 使用真实本地 HTTP 夹具覆盖列表/
  汇总请求、批准与拒绝、可选原因、409 已决定冲突和失败状态；QuickDesk 离屏启动确认页面
  可实例化，审批页 QML lint 无新增告警。
- QD-706 功能提交：`ea5a65f`。Qt 会话页消费 `remote_session.started/ended`，匹配当前设备
  的 150ms 防抖刷新取代常态轮询；消费 `device.access_policy.changed` 直接更新重复封禁来源，
  WebSocket `bootstrap_done` 后停止 10 秒轮询，断线时自动恢复轮询。
- QD-706 测试：`quickdesk_remote_session_manager_test` 覆盖匹配/非匹配会话事件与策略事件；
  `quickdesk_cloud_device_manager_test` 用真实本地 WebSocket 服务器验证
  `realtimeConnected` 在 `bootstrap_done` 后为真、`stopSync()` 后为假。
- QD-708 计划任务功能提交：`42b6c61`。Qt 新增计划任务页，支持一次、固定间隔和五种
  日历排期，展示服务端支持的 action/排期、下一次执行、尝试次数、最后错误，并支持删除；
  消费 `scheduled_task.failed` 实时提示和防抖刷新，30 秒轮询作为断线兜底。
- QD-708 计划任务测试：`quickdesk_scheduled_task_manager_test` 使用真实本地 HTTP 夹具
  覆盖列表解析、一次性/日历 payload 分离、删除路径和失败事件信号。
- QD-708 通知规则功能提交：`dbd828f`。Qt 新增通知页，支持 Webhook 通道 CRUD/test/
  启停、事件订阅、静默时段与周期窗口、设备离线阈值/重复提醒/维护窗口、投递历史、
  单条重试/确认、按通道和跨通道失败重投，并消费设备实时事件触发防抖刷新。
- QD-708 通知规则测试：`quickdesk_notification_manager_test` 覆盖通道读取/创建/编辑/
  删除/测试、静默时段保存、离线告警读取/保存/删除、投递记录/重试/确认、失败重投和
  实时刷新信号。
- QD-508 录像回放功能提交：`74c7022`。Qt 新增录像页，支持游标分页、缩略图票据、
  应用内播放/暂停/停止、外部打开、删除、实时事件刷新和分页加载更多。
- QD-508 测试：`quickdesk_recording_manager_test` 覆盖录像列表和游标、播放/缩略图
  票据 URL 解析、删除、分页追加和录像实时刷新信号。
- QD-204 功能提交：`f270487`、`b1a1155`、`162ac6e`、`b013a62`。服务端新增
  `user_device_groups` / `user_device_group_members` 和 `/v1/me/device-groups`
  五个接口；WebClient 和 Qt 均有分组筛选、CRUD、颜色/描述、成员整体替换和失败回滚；
  分组只包含当前用户名下设备，同一用户内名称唯一、跨用户可同名，单组成员上限 500。
- QD-204 验证：全新 PostgreSQL 从零执行 `001_init.sql` 通过；全量 Go 测试
  `-p 1 -count=1` 通过；四套 SDK freshness 和编译检查通过；真实 SignalingServer
  `openapi-smoke.mjs` 输出 `PASS: 388 documented assertions`；WebClient 34/34 与生产构建
  通过；Qt 完整构建、11/11 CTest、新弹窗 lint、双语 764/764 和离屏启动通过。
- QD-607 控制面批量动作：新增 `revoke_all_remote_sessions`，按执行时的设备归属关闭
  owner 所有设备上的活动远控会话，最多一次处理 200 台；不注销账号登录会话，也不会把
  “没有活动会话”当失败。服务端调度 worker 在完整动作注册后启动，`supported_kinds`
  由服务端注册表返回，WebClient 和 Qt 直接消费，不在客户端硬编码能力表。
- QD-607 验证：服务层真实 PostgreSQL 覆盖 owner 过滤、已结束会话排除、单台失败不拖死其余
  设备；handler 适配器复用现有跨实例 `revokeDeviceSessions` 路径；空测试库按 CI 口径执行
  `go test -p 1 -count=1 ./...` 和 `go vet ./...` 通过；本地服务启动日志确认
  `kinds=[revoke_all_remote_sessions sign_out_all_sessions]`。
- QD-607 上游候选：`codex/upstream-qd-607-batch-revoke` = `badf990`，基于 PR #25 的
  `feat/signaling-control-plane@8998466`，只含服务端、handler、测试和用户 API 文档
  7 个文件。候选基线尚无 Web/Qt 计划任务页，所以三端标签与 fallback 留在 fork，
  不把不存在的页面强塞进同一个上游 PR。
- 上游对账：本次重新 fetch 后 `upstream/dev` / `upstream/master` 仍为 `f6651f0`，open
  issue 仍为 10 个，open PR 仍为 #22、#23、#24、#25；没有新增上游项需要插队。QD-607
  属于 `Fork` / `Legacy` 的控制面补强，不是复制上游已有 PR；等 PR #25 基线确认后作为
  管理面增量准备，不把它包装成上游已经提出的需求。
- QuickDesk 当前 11/11 CTest 通过（SPKI、更新解析、更新 HTTP 集成、设备显示 QML、
  访问码回源集成、访问审批管理集成、远程会话实时事件、计划任务管理、通知规则管理、
  录像回放管理、设备分组管理）；双语 `.ts` 各 765 条完成、0 unfinished。
- Windows 更新基线：在 Parallels Windows 11 中从干净的 `b283c7f` 源码完成
  QuickDesk 4.2.8.0 Release 编译和静默安装；注册表显示 4.2.8.0，QuickDeskHost
  服务 RUNNING，基线安装包 89,678,233 字节。随后构建 4.2.9.0 升级包
  （89,676,841 字节，SHA-256
  `67dbc4e5f56dec06d20b0feeef7d70545f087af4b8673025adba987cfcf9eeb6`），
  客户端已真实发现更新、下载并通过摘要校验。尚未点击“安装并重启”，因此安装器启动、
  进程退出/重启、版本落盘和服务重建仍待确认，QD-005 / QD-808 继续维持 `[~]`。
- QD-215 Windows 发布依赖回归：新增 release 校验器，检查 `QuickDesk.exe` 与
  `platforms/qwindows.dll` 的 PE 架构、VC Runtime、Qt 平台插件及直接导入依赖；
  `publish_qd_win.bat` 和 `package_qd_win.bat` 都会在继续前失败关闭。Windows 11 中
  `dumpbin` 确认两个二进制均为 x64（`0x8664`），`qwindows.dll` 直接依赖
  `Qt6Gui.dll`、`Qt6Core.dll`、`MSVCP140.dll`、`VCRUNTIME140.dll` 和
  `VCRUNTIME140_1.dll`。完整 publish 和 Inno Setup 打包均以退出码 0 通过；负向测试
  证明缺 `vcruntime140_1.dll`、`platforms/qwindows.dll` 或 `Qt6Network.dll` 时会被拒绝。
  issue #7 的原始坏安装包尚未取得，因此还不能声称直接复现了用户环境。
- QD-215 上游候选：`codex/upstream-qd-215-win-runtime-check` = `f449b87`，基于
  `upstream/dev`，只含 release 校验器、publish 检查和 package 检查 3 个文件；
  已用该候选校验器跑完同一组正负向测试。远端 push 仍被执行策略拒绝。
- WebClient 当前 34/34 测试通过；生产构建 JS 362.03 kB、gzip 116.68 kB。
- `git diff --check`：通过。
- 远端推送仍被当前执行策略拒绝，本地 `dev` 尚未同步到 GitHub。

### 3.1 统一需求决策台账

这张表是本轮对“我们自己的清单 + 旧 Go-Remote 缺口 + 上游待办”的合并结论。
它不替代后面的完整功能表，只固定优先级、依赖和上游贡献方式。

| 队列 | 工作项 | 来源 | 当前判断 | 下一步 |
|---|---|---|---|---|
| 上游同步 | PR #22 自动审批线程亲和性 | `Upstream` / QD-113 | 上游 PR 已覆盖 issue #17；本 fork 不重复实现 | 等 PR #22 合并后同步；当前 `master <= dev`，合并时确认目标分支 |
| 上游同步 | PR #23 会话录像与回放 | `Upstream` / QD-501~507 | 已覆盖服务端索引、Web/Qt 手动录制、预览、生命周期；Host 权威录制不在其中 | 不重复提交；合并后回灌 fork，再单独推进 Host 录制 |
| 上游同步 | PR #24 skill-host Windows 运行库 | `Upstream` / QD-402、QD-609 | 修复技能进程找不到 MSVC runtime；与 fork 同向 | 合并后同步，不维护第二套 PATH/stderr 逻辑 |
| 上游同步 | PR #25 自托管控制面、OpenAPI、SDK | `Upstream` / QD-101~110、201~211、501、601~608、801~808 | 当前控制面总集；维护者可接受继续拆 PR | 等审查；若维护者要求，按 access / alerting / recording / operations / update 拆 |
| 已完成 | QD-708 Qt 通知规则页 | `Fork` / `Upstream` 通用能力 | 通知通道、静默时段、设备离线告警、投递重试/确认和实时刷新已完成，提交 `dbd828f` | PR #25 合并后拆成通用 Qt 管理面候选并提交上游 |
| 已完成 | QD-508 Qt 录像回放页 | `Legacy` / `Upstream` PR #23 | Qt 录像列表、分页、缩略图、应用内播放、外部打开和删除已完成，提交 `74c7022` | PR #23 合并后作为通用 Qt 管理面候选提交上游 |
| 已完成 | QD-204 地址簿与设备分组用户入口 | `Legacy` / `Upstream` Roadmap | owner-scoped API、四套 SDK、WebClient 和 Qt 已完成；真实 service smoke 388 条断言通过 | 本地候选已准备：API `codex/upstream-qd-204-api@d2321b9`，客户端 `codex/upstream-qd-204-client@b259d03`；等远端写入恢复后建 PR |
| 本地收口 | QD-607 计划任务执行动作 | `Fork` / `Legacy` | 服务端调度、Web/Qt 管理面、重试已完成；新增 `revoke_all_remote_sessions`，可按执行时设备归属批量关闭活动远控会话（最多 200 台）；纯服务端候选 `codex/upstream-qd-607-batch-revoke@badf990` 已独立验证 | “定时连接”仍需控制端到点发起连接的协议；更广义的批量 agent 指令仍需执行与回执通道；远端写入恢复后推送候选并作为 PR #25 的 stacked PR |
| 本地收口 | QD-005 / QD-808 自动更新真机验证 | `Fork` / `Legacy` / `Upstream` PR #25 | Windows 4.2.8.0 已安装，4.2.9.0 已由客户端发现、下载并校验；安装与重启尚未执行 | 明确确认后完成 Windows 升级/重启与版本/服务证据；再补 macOS DMG 替换、签名和公证回归 |
| 上游候选 | QD-003 多标签控制与音频抢占 | `Legacy` / `Upstream` Roadmap 补强 | 候选 `codex/upstream-cross-tab-coordination` = `6047615`，仅 WebClient 改动 | 远端写入恢复后推送并建 PR；补截图和同源双标签验证说明 |
| 上游候选 | QD-114 自托管 WSS SPKI 兼容 | `Upstream` issue #20 / `Fork` | 候选 `codex/upstream-qd-114-spki` = `13559e6`；临时方案不替代真实证书校验 | 推送并建 issue #20 PR；补 Qt 构建、Windows 服务和真实 `wss://` |
| 上游候选 | QD-212 设备列表本机标识 | `Upstream` issue #19 / `Legacy` | 候选 `f1ce096` 已在干净上游基线构建和测试 | 推送并建 PR；与 #19 其余问题拆开 |
| 上游候选 | QD-213 缺失访问码回源 | `Upstream` issue #19 / `Legacy` | 候选 `6928d24` 已通过集成测试 | 推送并建 PR；补两真机换码时序回归 |
| 上游候选 | QD-214 文件传输错误诊断 | `Upstream` issue #19 / `Legacy` | 候选 `94d3ec3` 改善 `Host error:type=2` 提示；根因未定位 | 推送诊断 PR；另收 Windows host 日志再做根因修复 |
| Remoting 需求 | QD-002 Web 完整文件管理器 | `Legacy` / `Fork` | 当前 skill bridge 只在 Qt 与 host 间可用，浏览器没有等价通道 | 向 Remoting 提 host 文件操作通道需求；在此之前不把上传下载冒充完整管理器 |
| Remoting 需求 | QD-004 / QD-505 Host 权威录制与断线收尾 | `Legacy` / `Fork` | PR #23 只录控制端已解码画面；Host/Relay 没有权威媒体落盘 | 提 Remoting/宿主生命周期需求；fork 不另造第二套媒体链路 |
| Remoting 需求 | QD-007 图片剪贴板 | `Legacy` / `Fork` | Remoting `ClipboardEvent` 只承载 UTF-8 文本 | 提图片格式、分片和权限边界需求；等协议方案后再实现 |
| Remoting 需求 | QD-114 真实 CertVerifier | `Upstream` issue #20 | SPKI 只是临时放行；根因是 `quickdesk_network_service.cc` 未绑定真实证书校验器 | 向 Remoting 维护者提正式修复需求和回归用例 |
| 验证/排障 | QD-215 Windows Qt platform plugin | `Upstream` issue #7 | 已补发布依赖校验和缺 DLL/Qt 插件负向回归，Windows publish/package 均通过；上游候选 `f449b87` 已准备 | 远端写入恢复后推送候选并开 issue #7 PR；取得原始坏包后做直接复现 |
| 验证/排障 | QD-216 主机连接失败 | `Upstream` issue #9 | 缺真实 host/signaling/TLS 日志，不能直接猜实现 | 先做支持 triage；能拆出确定 bug 后再进入实现 |
| 安全设计 | QD-115 服务端设备密钥信任边界 | `Upstream` issue #10 / `Fork` | 信令服务端是否能冒充设备仍缺威胁模型 | 先写安全边界和攻击面；需要协议改动时与维护者协作 |
| 上游流程 | QD-911 私密安全披露入口 | `Upstream` issue #6 | 上游没有 `SECURITY.md`，issue 要求私密漏洞报告渠道 | 在 fork 准备最小文档候选，并建议上游启用 GitHub Private Vulnerability Reporting |
| 后续增强 | QD-604 指标型告警、QD-702 墙级音频、QD-610 业务工作流 | `Legacy` / `Fork` | 均有基础或设计，但没有完整产品闭环 | 等当前 Qt/录像/更新收口后逐项做 |
| 质量收口 | QD-906、QD-907、QD-908、QD-909 | `Legacy` / `Fork` | 真机矩阵、异常路径、2h+ 长稳、文件/剪贴板审计仍不完整 | 每完成一个主功能就补对应边界，不集中到最后一次性补测 |
| 明确暂缓 | QD-318 Linux、QD-319 原生移动端 | `Upstream` Roadmap | 上游已列计划，当前 fork 不做实现承诺 | 保持接口和构建兼容，等上游方向明确后接 |
| 明确不做 | RDP、重写 WebRTC/Remoting、多租户/SSO/计费、通用插件市场 | `Fork` 决策 | 与当前“复用成熟远控内核 + 补业务控制面”的路线冲突或收益不足 | 不进入待办池，除非用户重新定义产品目标 |

### 3.2 合并后的执行队列

这张表把 fork 自己的未完成项与上游 10 个 open issue、4 个 open PR、3 个未完成
Roadmap 项放到同一条实施路径中。详细追溯仍以第 16 节为准。

| 顺序 | 队列 | 包含工作 | 为什么现在放这里 | 完成条件 |
|---|---|---|---|---|
| 1 | Fork 真机收口 | QD-005 / QD-808 Windows 自动更新、QD-215 Windows Qt 插件、QD-214 上传根因、QD-607 定时连接边界 | 不依赖上游合并，且有 Windows VM 可复现；先证明客户端和后端闭环，而不是继续堆页面 | 安装包升级、校验、静默安装、重启和版本证据齐全；不能伪造成功 |
| 2 | 可独立回贡上游 | QD-003、QD-114 第一阶段、QD-212、QD-213、QD-214 诊断、QD-215 Windows 发布校验、QD-911 | 改动边界清楚，可从 `upstream/dev` 独立审查；其中 QD-114 只解决 SPKI 临时兼容 | 每个主题一个基于 `upstream/dev` 的最小分支；远端写入恢复后推送并开 PR |
| 3 | 等 PR #25 基线 | QD-204 API/客户端、QD-607 服务端、QD-108 / QD-508 / QD-706 / QD-708 管理面、QD-005 客户端 | 这些能力依赖控制面、OpenAPI、录像或管理页基线；直接塞进 #25 会变成巨型 PR | 维护者确认 #25 合并或 stacked base；再按服务端/Web/Qt/更新拆 PR |
| 4 | 跟踪上游已有 PR | PR #22 自动审批、#23 录像、#24 skill-host 运行库、#25 控制面 | 上游已有实现，重复开发只会制造分叉 | PR 合并后同步 `upstream/dev`，重跑受影响测试；删除重复本地实现 |
| 5 | 提 Remoting 需求 | QD-002 Web 文件通道、QD-004 / QD-505 Host 权威录制、QD-007 图片剪贴板、QD-114 真实 CertVerifier | 必须改 Chromium Remoting 或宿主生命周期，维护者已明确这类改动由他处理 | 每项提交协议草案、权限边界、兼容性和验收用例；不在 fork 伪造第二套媒体链路 |
| 6 | fork 自主增强 | QD-006 / QD-809 SQLite、QD-810 备份恢复、QD-604 指标告警、QD-702 墙级音频、QD-610 工作流、QD-608 更多 SDK、QD-904 / QD-907 / QD-908 / QD-909 | 有产品价值但不阻塞上游，也不是当前最小可回贡补丁 | 有明确用户入口、失败路径和对应自动化或真机证据 |
| 7 | 暂缓跟随上游 | QD-318 Linux、QD-319 iOS/Android | 上游 Roadmap 已列出，本 fork 当前不承诺实现 | 保持接口和构建兼容；上游方向明确后再排期 |
| 8 | 不进入待办 | RDP、重写 WebRTC/Remoting、多租户/SSO/计费、通用插件市场 | 与“复用成熟远控内核 + 补业务控制面”的路线冲突或收益不足 | 只有用户重新定义产品目标才重新评估 |

当前第一执行项仍是 `QD-005 / QD-808` 的 Windows 真机升级验证。4.2.8.0 已安装，
4.2.9.0 已被客户端发现、下载并通过摘要校验；剩下的是需要明确确认的“安装并重启”
以及安装后的版本、进程和服务证据，因此状态继续维持 `[~]`。

回贡上游不等到 fork 全部做完再一次性提交。每个已经独立验收、能从 `upstream/dev`
干净切出的主题都应立即形成最小 PR；依赖 PR #25 的功能只准备 stacked 候选，不继续
扩大现有巨型 PR。

## 4. 账号、认证与安全

| ID | 功能 | 状态 | 说明 |
|---|---|---|---|
| QD-101 | 用户注册、登录、退出 | `[x]` | 已有 access/refresh token |
| QD-102 | 短信验证码与账号恢复流程 | `[x]` | SignalingServer 已有短信服务与限流 |
| QD-103 | JWT 刷新、会话撤销 | `[x]` | access token、refresh token、session revoke |
| QD-104 | 管理员登录与 TOTP | `[x]` | 管理端双重验证 |
| QD-105 | 设备配额与用户设备隔离 | `[x]` | 设备归属和 owner-scoped API |
| QD-106 | 设备访问策略 | `[x]` | 开关、IP/CIDR、允许时段、例外日 |
| QD-107 | 临时访客授权 | `[x]` | 限时、限次、可撤销、IP/client 绑定 |
| QD-108 | 二次访问审批 | `[x]` | 服务端、WebClient 和 Qt 均已完成；Qt 提交 `7c23f13` 覆盖入口、决定、汇总、冲突处理和实时消费 |
| QD-109 | 来源封禁与解封 | `[~]` | 服务端、WebClient、Qt、管理端和 SDK 已完成；真实 Windows 踢人回归待补 |
| QD-110 | 审计、归档、导出、保留策略 | `[x]` | 管理端和用户侧事件已覆盖 |
| QD-111 | 文件传输审计 | `[ ]` | 目前文件走 Remoting 数据通道，控制面没有事件 |
| QD-112 | SSO、多租户、计费 | `[-]` | 当前不做 |
| QD-113 | Trust 自动审批线程亲和性 | `[~]` | 上游 issue #17 / PR #22 已有修复；本 fork 未重复实现，PR 合并后同步 |
| QD-114 | 自托管 HTTPS/WSS 证书校验 | `[~]` | 上游 issue #20；Qt 子进程支持 SPKI 列表，Windows 服务安装支持 `QUICKDESK_SPKI`；彻底修复仍需 Remoting 接入真实 CertVerifier，Qt/Windows 实机构建待补 |
| QD-115 | 信令服务器持有设备密钥的信任边界 | `[ ]` | 上游 issue #10；需要明确服务端是否可代替设备连接的安全模型 |

## 5. 设备与会话

| ID | 功能 | 状态 | 说明 |
|---|---|---|---|
| QD-201 | 设备 ID、设备密钥、自动 provisioning | `[x]` | Host 自动注册 |
| QD-202 | Pair Code / 设备绑定 | `[x]` | 用户绑定与解绑 |
| QD-203 | 设备列表、收藏、最近连接 | `[x]` | WebClient 和 Qt 均有入口 |
| QD-204 | 地址簿与设备分组 | `[x]` | 上游 Roadmap 的 Address book & device groups；owner-scoped API/SDK、WebClient、Qt、成员整体替换、500 上限和绑定生命周期清理均已完成，提交 `f270487`、`b1a1155`、`162ac6e`、`b013a62` |
| QD-205 | 在线状态与心跳 | `[x]` | Redis presence + heartbeat |
| QD-206 | 连接历史与快速连接 | `[x]` | 上游 `dev` 与 fork 都已有 `ConnectionHistory` 和连接历史 UI；issue #18 看起来已过期，先验证再关闭，不重复实现 |
| QD-207 | 会话创建、运行、结束、清理 | `[x]` | `remote_sessions` 控制面已落地 |
| QD-208 | 会话详情、录像、访问日志聚合 | `[x]` | WebClient 和 API 已完成 |
| QD-209 | 单会话踢人、按设备批量清场 | `[~]` | 服务端和 Web 已完成；Qt 真机回归待补 |
| QD-210 | 跨实例会话撤销 | `[~]` | 已按实例寻址；Redis pub/sub 不是可靠队列 |
| QD-211 | 撤销冷却期 | `[x]` | 支持窗口和递增冷却 |
| QD-212 | 设备列表标识本机 | `[x]` | 上游 issue #19；Qt 我的设备/收藏已标记“本机”，候选 `f1ce096` 已独立验证 |
| QD-213 | 访问码不可用回归 | `[x]` | 上游 issue #19；Qt 缺失/过期访问码会按单设备回源并自动继续连接，候选 `6928d24` 已独立验证 |
| QD-214 | 文件上传 Host error type=2 | `[~]` | 上游 issue #19；错误码已解释为宿主保存失败并给出排查方向，上传取消码已修正；真实 Windows 宿主日志和失败路径仍未定位 |
| QD-215 | Windows 启动 Qt platform plugin 错误 | `[~]` | 上游 issue #7；发布链新增 x64、VC Runtime、`qwindows.dll` 和 Qt 直接依赖校验，Windows 正负向测试通过；上游候选 `f449b87` 已准备，原始坏安装包未复现，push/PR 被策略阻塞 |
| QD-216 | 主机连接与设备 ID/访问码加载失败 | `[~]` | 上游 issue #9；作为支持 triage 跟踪，不直接等同产品需求 |

## 6. 远程控制核心

| ID | 功能 | 状态 | 说明 |
|---|---|---|---|
| QD-301 | 视频画面与低延迟观看 | `[x]` | Chromium Remoting |
| QD-302 | H.264/VP8/VP9/AV1 | `[x]` | 编码偏好可配置 |
| QD-303 | 硬件编码 | `[x]` | 继承 Remoting 各平台后端 |
| QD-304 | Windows/macOS 输入注入 | `[x]` | 鼠标、键盘、滚轮、文本输入 |
| QD-305 | iOS/Safari 触摸输入 | `[x]` | WebClient touch handler |
| QD-306 | 多显示器切换 | `[x]` | 当前代码已接线 |
| QD-307 | 虚拟显示器 | `[x]` | Windows IDD 驱动与 UI |
| QD-308 | 隐私屏 | `[x]` | Windows 被控端 |
| QD-309 | 锁屏控制与安全桌面输入 | `[x]` | Windows Service |
| QD-310 | 会话结束自动锁屏 | `[x]` | v4.2 能力 |
| QD-311 | 自适应帧率、码率和质量 | `[x]` | 已有画质/延迟指标 |
| QD-312 | 远端音频播放和静音 | `[x]` | WebClient 和 Qt 已有 |
| QD-313 | 多标签控制与音频抢占 | `[x]` | BroadcastChannel + localStorage fallback；浏览器双标签实测通过 |
| QD-314 | 多人同时控制 | `[ ]` | 需要控制权模型和冲突策略 |
| QD-315 | 画笔标注 | `[ ]` | 需要共享标注画布 |
| QD-316 | 文字聊天 | `[ ]` | 需要会话内消息通道 |
| QD-317 | 自定义采集/编码插件 | `[ ]` | 暂不优先 |
| QD-318 | Linux 被控端 | `[>]` | 上游 Roadmap 已列出；本 fork 暂缓 |
| QD-319 | iOS/Android 原生控制端 | `[>]` | 上游 Roadmap 已列出；Web 触摸输入不等同原生客户端 |

## 7. 文件、剪贴板与传输

| ID | 功能 | 状态 | 说明 |
|---|---|---|---|
| QD-401 | 文件上传、下载和分片传输 | `[x]` | Remoting 数据通道已具备 |
| QD-402 | Qt 完整文件管理器 | `[x]` | `RemoteFileManager` + `file-ops` skill bridge |
| QD-403 | Web 完整文件管理器 | `[!]` | 当前迭代 QD-002，缺浏览器 host 文件通道 |
| QD-404 | 文件传输进度、取消、重试 | `[x]` | WebClient 和 Qt 已有基础 |
| QD-405 | 双向文本剪贴板 | `[x]` | UTF-8 文本 |
| QD-406 | 双向图片剪贴板 | `[ ]` | 当前迭代 QD-007，需要 Remoting 协议扩展 |
| QD-407 | 系统级文件剪贴板 | `[ ]` | 当前不做主链路承诺 |
| QD-408 | 文件传输审计 | `[ ]` | 同 QD-111 |

## 8. 录制、回放与文件生命周期

| ID | 功能 | 状态 | 说明 |
|---|---|---|---|
| QD-501 | 录像索引、归属、分页 | `[x]` | `session_recordings` |
| QD-502 | Web 控制端手动录制 | `[x]` | MediaRecorder |
| QD-503 | Qt 控制端手动录制 | `[x]` | DeviceRecorder |
| QD-504 | Host/Relay 自动录制 | `[ ]` | 当前迭代 QD-004 |
| QD-505 | 断线自动收尾 | `[ ]` | 仅控制端录制时没有 Host 侧收尾 |
| QD-506 | 录像上传、断点续传 | `[~]` | 服务端和 SDK 已完成；Web UI 消费待补 |
| QD-507 | Web 录像列表、播放、下载、删除 | `[x]` | `RecordingsPage.vue` |
| QD-508 | Qt 录像回放页 | `[x]` | `74c7022` 已覆盖列表、分页、缩略图、应用内播放、外部打开、删除和实时刷新 |
| QD-509 | 缩略图、保留期、孤儿回收 | `[x]` | 服务端生命周期已完成 |
| QD-510 | 工作流录制与回放 | `[~]` | 有设计文档，未形成完整产品 |

## 9. 通知、任务与集成

| ID | 功能 | 状态 | 说明 |
|---|---|---|---|
| QD-601 | 用户通知通道与 Webhook | `[x]` | HMAC、outbox、重试、死信重放 |
| QD-602 | 投递确认与静默时段 | `[x]` | ack、一次性和周期静默 |
| QD-603 | 设备离线持续告警 | `[x]` | 阈值、重复提醒、维护窗口 |
| QD-604 | 指标型告警 | `[ ]` | CPU/内存等依赖被控端上报 |
| QD-605 | WebClient 通知管理页 | `[x]` | `/notifications` |
| QD-606 | Qt 通知规则管理页 | `[x]` | `dbd828f` 已覆盖通道 CRUD/test/启停、静默时段、离线告警和投递运维 |
| QD-607 | 计划任务 | `[~]` | 服务端、WebClient、Qt 管理页和重试已完成；`revoke_all_remote_sessions` 已补齐控制面批量清场（最多 200 台）。定时连接、广义批量 agent 指令及回执仍未做 |
| QD-608 | OpenAPI 与生成 SDK | `[~]` | Go/TS/Python/Java；C#/PHP/Kotlin 未提供 |
| QD-609 | MCP / AI 自动化 | `[x]` | QuickDesk 原生能力；也是上游维护者当前最优先方向 |
| QD-610 | 业务系统集成与自定义工作流 | `[~]` | 设计和工作流方向已有，完整产品未收口 |

## 10. Web、Qt 与管理端

| ID | 功能 | 状态 | 说明 |
|---|---|---|---|
| QD-701 | Web 登录与设备接入 | `[x]` | Vue shell + remote.html |
| QD-702 | Web 屏幕墙 | `[~]` | 多设备、轮巡、单砖解锁已有；墙级音频待补 |
| QD-703 | Web 会话页 | `[x]` | 进行中、已结束、详情、踢人、清场 |
| QD-704 | Web 录像页 | `[x]` | 列表、播放、下载、删除 |
| QD-705 | Qt 主界面与远程窗口 | `[x]` | Windows/macOS |
| QD-706 | Qt 会话页 | `[x]` | 页面已有；`ea5a65f` 已消费会话生命周期和策略变更事件，轮询只在 WebSocket 未连接时兜底 |
| QD-707 | Qt 文件管理器 | `[x]` | host `file-ops` skill bridge |
| QD-708 | Qt 通知/计划/审批页 | `[x]` | 审批页 `7c23f13`、计划任务页 `42b6c61`、通知规则页 `dbd828f` 均已完成 |
| QD-709 | 管理后台设备、会话、审计、发布 | `[x]` | SignalingServer web |
| QD-710 | 中英文与移动布局 | `[x]` | Web 端已有回归 |

## 11. 部署、运维与发布

| ID | 功能 | 状态 | 说明 |
|---|---|---|---|
| QD-801 | PostgreSQL + Redis 自托管 | `[x]` | 生产主路径 |
| QD-802 | Docker / Nginx / TLS 部署 | `[x]` | 已有部署脚本和手册 |
| QD-803 | coturn P2P/TURN | `[x]` | UDP/TLS/DTLS 和小时级 soak |
| QD-804 | Prometheus 指标和告警规则 | `[x]` | `/metrics` + rules |
| QD-805 | 日志检索和跨实例聚合 | `[x]` | 本地/聚合搜索 |
| QD-806 | 多实例会话和租约 | `[~]` | 已有实例寻址；Redis 仍是单点 |
| QD-807 | 安装包、升级发布服务端 | `[x]` | 上传、Range、断点续传、管理端 UI |
| QD-808 | 客户端自动更新消费 | `[~]` | 当前迭代 QD-005；客户端检查/下载/校验/安装入口已完成。Windows 用 `/SILENT /CLOSEAPPLICATIONS /RESTARTAPPLICATIONS` 启动安装器后退出；macOS 打开 DMG 后退出。剩余：Windows 安装包真机升级/自动重启、macOS DMG 真机替换与签名/公证回归 |
| QD-809 | SQLite 轻量服务端 | `[ ]` | 当前迭代 QD-006 |
| QD-810 | 正式备份、恢复和容量配额 | `[ ]` | 还需完整演练和文档 |

## 12. 质量与验证

| ID | 功能 | 状态 | 说明 |
|---|---|---|---|
| QD-901 | Go 全包测试 | `[x]` | 已有 `go test ./...` |
| QD-902 | WebClient 单测 | `[x]` | Node test |
| QD-903 | WebClient 生产构建 | `[x]` | Vite build |
| QD-904 | Web End-to-End / Playwright | `[~]` | 已有部分 smoke，覆盖面待扩 |
| QD-905 | 信令、TURN 并发压测 | `[x]` | 已有负载和 soak 脚本 |
| QD-906 | Windows/macOS 真机矩阵 | `[~]` | Windows 主链已验证；输入、音频、剪贴板边界待补 |
| QD-907 | 协议异常、权限异常、重连耗尽 | `[~]` | 需要继续扩边界 |
| QD-908 | 2h+ 桌面长稳和几百路观看 | `[ ]` | 当前缺业务级长稳基线 |
| QD-909 | 安全审计：文件、剪贴板、权限 | `[~]` | 访问和会话已有；文件/剪贴板未进入控制面 |
| QD-910 | 需求、上游与实现持续对账 | `[~]` | 每次上游合并、issue/PR 变化或功能完成时，同步本清单、状态证据和上游贡献决策 |
| QD-911 | 上游私密安全披露入口 | `[ ]` | 上游 issue #6 仍开放；需准备 `SECURITY.md` 或推动启用 GitHub Private Vulnerability Reporting，并给出不公开利用细节的报告流程 |

## 13. 明确不做

- `[-]` RDP：会锁被控端本机会话。
- `[-]` 重写 Chromium Remoting 媒体内核。
- `[-]` 自研 WebRTC / ICE / STUN / TURN。
- `[>]` Linux 被控端：上游 Roadmap 已列出；本 fork 暂缓，接口和验证基线保留。
- `[>]` iOS/Android 原生控制端：上游 Roadmap 已列出；本 fork 暂缓。
- `[-]` 多租户、SSO、计费。
- `[-]` 通用插件市场和完整插件权限模型。

## 14. 推荐执行顺序

执行顺序按“先解除上游重复投入，再完成本地闭环，最后做协议扩展”排列：

1. **上游同步闸门**：持续跟踪 PR #22 / #23 / #24 / #25；已有 PR 的功能不重复实现。
   远端写入恢复后，先推送已准备好的 QD-003、QD-114、QD-212、QD-213、QD-214、QD-215
   最小候选。
2. **把 QD-204 回馈上游**：PR #25 的控制面/ API 基线确认后，先推送 owner-scoped
   服务端、OpenAPI 和 SDK 候选，再推送 Web/Qt 客户端候选；不把 fork 文档带入 PR。
3. **补任务和更新真机链路**：QD-607 的批量远控清场已完成；继续定义定时连接和广义
   批量 agent 指令/回执，补 QD-005 / QD-808 Windows 与 macOS 安装、重启、签名和
   公证验证。QD-215 已补发布依赖校验、正负向回归和上游候选 `f449b87`，剩余原始坏包复现与远端 PR。
4. **准备 Remoting 需求单**：QD-002 Web 文件操作通道、QD-004 / QD-505 Host 权威录制、
   QD-007 图片剪贴板、QD-114 真实 CertVerifier。每项写清协议、宿主生命周期、权限、
   兼容性和验收用例，先交上游维护者评估。
5. **安全与可观测性**：QD-115 设备密钥信任边界、QD-604 指标告警、QD-909 文件/剪贴板审计。
6. **规模与长稳**：QD-906 真机矩阵、QD-907 异常路径、QD-908 2h+ 和几百路观看，
   以实际发布门槛补证据，不把压测数字当宣传口径。
7. **Roadmap 暂缓项**：QD-318 Linux、QD-319 iOS/Android 跟随上游，不提前承诺。
8. 每次完成或合并后更新 QD-910：重新 fetch 上游、核对 issue/PR、更新本清单状态与证据。

## 15. 完成定义

一个功能只有同时满足以下条件才可标记为 `[x]`：

- 有用户可发现的入口，不是只有内部 API。
- 有失败路径和权限边界，不只验证成功路径。
- 有自动化测试或明确的只读/真机验收记录。
- 不破坏已有会话、审计、通知和权限语义。
- 文档、API 契约、SDK 或客户端类型在涉及接口时同步更新。
- 变更已记录到本清单，并留下验证命令或证据路径。

## 16. 上游 QuickDesk 待办映射

### 16.1 上游来源

- 上游仓库：<https://github.com/barry-ran/QuickDesk>
- 上游基线：`dev` / `master` = `f6651f0`
- 上游 Roadmap：Windows、macOS、P2P/TURN、MCP、虚拟屏、隐私屏、文件传输、音频、无人值守已完成；Linux、iOS/Android、Address book & device groups 未完成
- `Multi-tab multi-connection` 在上游已勾选；QD-003 补的是同一设备多个标签的控制/音频互斥，不重复“多连接”能力
- 维护计划：issue #21，维护者明确“持续维护、暂缓更新、重点偏 MCP、欢迎从 dev 切分支贡献、Remoting 改动由维护者处理”
- 当前 open PR：#22、#23、#24、#25
- 2026-10-17 复核：四个 PR 均显示 MERGEABLE / `mergeStateStatus=UNSTABLE`，但 GitHub
  没有 reported checks；不能把“MERGEABLE”写成 CI 已通过
- 本次融合结论：当前上游没有“必须由本 fork 立即重复开发”的 open issue；真正需要决策的是
  哪些工作作为独立小 PR 先贡献、哪些等 #25 合并后再拆、哪些只能向 Remoting 提需求。

### 16.2 上游待办与本 fork 的统一决策矩阵

这张表覆盖当前所有上游 open issue、open PR 和未完成 Roadmap 项。`现在做` 表示不依赖上游
合并即可在 fork 完成；`等待` 表示上游已有实现或 PR，不能重复提交；`边界` 表示需要
Remoting 或协议改造；`不重复` 表示上游已经具备，只需验证或回复 issue。

| 上游项 | 本 fork 对应项 | 当前决定 | 交付方式 | 下一步 |
|---|---|---|---|---|
| #21 维护计划与参与方式 | QD-001、QD-910 | 立即执行 | `Upstream-Open` | 所有新功能从 `upstream/dev` 切最小分支；持续同步清单 |
| #20 自托管 WSS 失败 | QD-114 | 双轨推进 | `UP-Ready` + `UP-Remoting` | Qt/Windows SPKI 先做独立 PR；真实 CertVerifier 交 Remoting |
| #19 设备列表问题 | QD-212、QD-213、QD-214 | 拆分处理 | `UP-Ready` | 本机标识、访问码回源分别提；上传错误先做诊断，根因另立项 |
| #18 连接历史 | QD-206 | 不重复实现 | `Upstream-Open` | 核验现有 WebClient/Qt 入口，整理版本证据后回复 issue |
| #17 Auto Approve 失效 | QD-113 | 等待上游 | `UP-PR#22` | #22 合并后同步；不在 fork 重写第二套线程亲和性修复 |
| #13 Linux 支持 | QD-318 | 暂缓 | `Upstream-Open` | 保留跨平台接口，不承诺当前迭代实现 |
| #10 信令服务设备密钥信任 | QD-115 | 先设计 | `Upstream-Open` | 写威胁模型和安全边界，再决定是否改协议 |
| #9 主机连接失败 | QD-216 | 先排障 | `Upstream-Open` | 收集 host/signaling/TLS 日志，确认可复现 bug 后再提单 |
| #7 Windows Qt 插件启动失败 | QD-215 | 本地回归完成 | `UP-Ready` | 候选 `f449b87` 已包含 x64/VC Runtime/Qt 插件/直接依赖校验和失败关闭；远端写入恢复后推送并开 PR |
| #6 私密安全披露入口 | QD-911 | 流程改进 | `Upstream-Open` | 推动 `SECURITY.md` 或私密披露渠道，不在公开 issue 贴利用细节 |
| PR #22 自动审批线程亲和性 | QD-113 | 等待同步 | `UP-PR#22` | 合并后 cherry-pick，确认 base 是 `master` 时的回灌顺序 |
| PR #23 会话录像与回放 | QD-501~509 | 等待同步 | `UP-PR#23` | 不重复提交；合并后回灌，Host 权威录制另走 Remoting |
| PR #24 skill-host Windows 运行库 | QD-402、QD-609 | 等待同步 | `UP-PR#24` | 合并后同步，避免维护第二套 PATH/stderr 逻辑 |
| PR #25 自托管控制面/OpenAPI/SDK | QD-101~110、201~211、501、601~608、801~808 | 等待审查 | `UP-PR#25` | 维护者要求时按 access/alerting/recording/ops/update 拆分 |
| Roadmap: Address book & device groups | QD-204 | 已完成，准备上游 | `UP-Ready` | API/SDK 与 Web/Qt 已验收；等 PR #25 基线确认后拆两个最小 PR |
| Roadmap: Linux support | QD-318 | 暂缓 | `Upstream-Open` | 跟随上游，不把 Web 触摸或跨平台构建冒充 Linux 被控端 |
| Roadmap: iOS / Android clients | QD-319 | 暂缓 | `Upstream-Open` | 当前 Web 触摸输入不等于原生客户端 |

### 16.3 上游已完成能力的继承边界

上游 README 的 `[x]` 表示“上游项目已经具备”，不等于本 fork 的产品入口、权限、测试和
发布证据都已完成。以下能力不重复开发，只做 fork 侧验证和产品化补口：

| 上游已完成能力 | 本 fork 处理 | 不能重复做什么 |
|---|---|---|
| Windows/macOS 远控主链路 | QD-301、QD-304、QD-705 | 不重写采集、编码、WebRTC 或输入注入 |
| P2P、STUN、TURN | QD-803 | 不自研 NAT 穿透或 TURN 协议 |
| 多标签多连接 | QD-313 | 不重新发明连接协议；只补控制/音频抢占和产品边界 |
| 访问码自动刷新 | QD-213 | 不在 Qt 和 Web 各写一套不同回源逻辑 |
| MCP、技能宿主、工具与安全确认 | QD-402、QD-609、QD-113 | 不复制 MCP 工具集；只在管理面和业务接入层扩展 |
| 设备记忆、工作流、虚拟屏、隐私屏、文件传输、音频 | QD-203、QD-307、QD-308、QD-312、QD-401~405、QD-510 | 不另造媒体/文件通道；只补本 fork 的页面、审计和验收 |

### 16.4 上游贡献状态的统一口径

上游贡献状态：

| 标记 | 含义 |
|---|---|
| `UP-PR#n` | 已经在上游 open PR 中，先同步审查，不重复提交 |
| `UP-Ready` | 可作为独立最小 PR 提交到 `upstream/dev` |
| `UP-Remoting` | 需要修改 `quickdesk-remoting`，优先提需求给维护者 |
| `Fork-only` | 产品品牌、部署配置或本 fork 专属业务，不进入上游 |
| `Upstream-Open` | 上游 issue/Roadmap 仍开放，当前 fork 可有本地实现，但需要单独拆 PR |

### 16.5 上游 PR 映射

| 上游项 | 状态 | 涉及 QD | 当前处理 | 后续动作 |
|---|---|---|---|---|
| PR #25 self-hosted control plane | OPEN，MERGEABLE / UNSTABLE，无 reported checks | QD-101~110、QD-201~211、QD-501、QD-601~608、QD-801~808 | 服务端、契约、SDK、smoke 和 Qt 更新消费已在本 fork；本地不重复造一套 | 等维护者审查；若要求拆分，按 access / alerting / recording / ops / update 拆 |
| PR #23 session recordings | OPEN，MERGEABLE / UNSTABLE，无 reported checks | QD-501~507 | 控制端手动录制、索引、回放、生命周期已具备 | 先等 PR #23；Host 权威录制另走 `UP-Remoting` |
| PR #24 skill-host runtime path | OPEN，MERGEABLE / UNSTABLE，无 reported checks | QD-402、QD-609 | 本 fork 已包含 PATH/stderr 修复，和上游补丁同向 | 上游合并后同步回 fork，避免二套实现 |
| PR #22 auto-approve thread affinity | OPEN，MERGEABLE / UNSTABLE，无 reported checks，基于 master | QD-113 | 本 fork 尚未重复实现 | 等 PR #22 合并后 cherry-pick；不要重复提交 issue #17 |

### 16.6 上游 issue / Roadmap 映射

| 上游项 | 类型 | 涉及 QD | 判断 | 动作 |
|---|---|---|---|---|
| #21 维护计划与参与维护意向 | 维护策略 | QD-001、QD-609 | 上游优先 MCP、从 dev 贡献、Remoting 由维护者改 | 所有新功能按 dev 最小分支准备；涉及 Remoting 先提需求 |
| #20 自托管 HTTPS/WSS 失败 | Bug / 自托管 | QD-114 | 证书验证器在 Remoting 内未绑定；Qt-only SPKI 已支持子进程和服务安装参数 | 推送候选并补 Windows/Qt 回归；长期向 Remoting 提 CertVerifier 修复 |
| #18 连接历史列表 | Feature | QD-206 | WebClient 已有 `ConnectionHistory`，Qt 已有 `RemoteDeviceManager` 历史下拉、删除和重连；issue 仍开放 | 在 issue 回复现有入口和版本证据，建议关闭；不再重复实现 |
| #19 设备列表相关问题 | Bug cluster | QD-212、QD-213、QD-214 | 本机标识和访问码回源已完成；上传错误诊断已改善，但根因仍需宿主日志 | 推送三个独立候选；QD-214 在复现后继续补真实修复，不混成一个“设备列表”大修 |
| #17 自动审批被忽略 | Bug | QD-113 | 已由 PR #22 覆盖 | 跟踪 PR #22，不重复实现 |
| #13 Linux 支持 | Roadmap | QD-318 | 上游 Roadmap 已列出，本 fork 暂缓 | 保持跨平台接口，不在当前迭代投入 Linux 被控端 |
| #10 信令服务器设备密钥信任 | Security | QD-115 | 服务端是否可冒充设备需要明确威胁模型 | 与上游安全维护者讨论；先写安全边界，不改协议 |
| #9 主机连接失败 | Support / Bug | QD-216 | 需要真实环境日志，不宜直接猜实现 | 收集 host/signaling/TLS 日志后再拆 bug |
| #7 Windows 启动 Qt platform plugin 错误 | Packaging | QD-215 | Qt 插件/运行库打包问题；Windows publish/package 与缺 DLL 负向测试已通过，候选 `f449b87` 仅 3 个文件 | 推送候选并开 issue #7 PR；取得原始坏包后做直接复现 |
| #6 私密安全披露入口 | Process | QD-911 | 不是产品功能，但影响上游贡献可信度 | 推动 SECURITY.md 或私密披露渠道，不在公开 issue 贴利用细节 |

### 16.7 上游 Roadmap 映射

| 上游 Roadmap | 当前 fork | 状态 | 备注 |
|---|---|---|---|
| Address book & device groups | QD-204 | `[x]` | fork 已完成 owner-scoped API/SDK、Web/Qt 和真实 smoke；上游 Roadmap 仍未勾选，等待拆分 PR 回馈 |
| Linux support | QD-318 | `[>]` | 上游已列计划，fork 暂缓；不能把 Linux 写成永久不做 |
| iOS / Android clients | QD-319 | `[>]` | 上游已列计划，fork 暂缓；现有 Web 触摸输入不等于原生客户端 |
| MCP / AI 自动化 | QD-609 | `[x]` | 上游当前最高优先级；我们的新增业务能力尽量不破坏 MCP 兼容 |

## 17. 上游贡献与 PR 策略

### 17.1 分支与合并

- 从 `upstream/dev` 创建最小功能分支，不把整个 fork `dev` 直接推成一个巨型 PR。
- 不直接向 `barry-ran/QuickDesk` 的 `dev` 推送；先推到自己的 fork，再以
  `upstream/dev` 为目标分支创建 PR。
- 每个 PR 只包含一个功能或一个 bug；服务端、OpenAPI、生成 SDK、smoke 必须同 PR。
- 上游合并后，把 `upstream/dev` 合回 fork `dev`，重新跑受影响测试，不在两边维护不同实现。
- 具有本 fork 品牌、部署环境、私有业务规则的页面和文档留在 fork；通用能力优先上游。

### 17.2 可直接准备的上游 PR

| 顺序 | PR 主题 | 来源 | 前置条件 |
|---|---|---|---|
| 1 | `feat(webclient): coordinate cross-tab control and audio` | QD-003 | 当前功能已测试；补上游 WebClient 约定和截图 |
| 2 | `fix(self-host): allow pinned SPKI for self-hosted WSS` | QD-114 / issue #20 | 候选已准备；需推送、创建 PR，并补 Qt 构建和 Windows service 实机验证 |
| 3 | `feat(qt): mark the local device in device lists` | QD-212 / issue #19 | 候选已准备并通过上游基线构建；等远端写入后直接创建 PR |
| 4 | `fix(qt): refresh missing access codes before connecting` | QD-213 / issue #19 | 候选已准备并通过上游基线构建；等远端写入后直接创建 PR |
| 5 | `fix(webclient): explain host file transfer failures` | QD-214 / issue #19 | 诊断改进候选已准备；根因 PR 需等真实宿主日志 |
| 6 | `fix(win): reject incomplete Qt runtime releases` | QD-215 / issue #7 | 候选 `f449b87` 只含 verifier 和 publish/package 失败关闭；校验器正负向测试通过 |
| 7 | `feat(scheduled): add batch remote-session revoke` | QD-607 | 纯服务端候选 `badf990` 已基于 PR #25 独立验证；远端写入恢复后作为 stacked PR 提交 |
| 8 | `feat(webclient): control-plane management pages` | QD-108/605/607/703 | 依赖 PR #25 合并；按页面拆成多个 PR |
| 9 | `feat(qt): remote session and management pages` | QD-706/708 | 依赖 PR #25 合并；审批页 `7c23f13`、会话实时消费 `ea5a65f` 和计划任务页 `42b6c61` 可先拆，再补通知页 |
| 10 | `feat(qt): consume self-hosted update channel` | QD-005 / QD-808 | 依赖 PR #25 合并；保留检查、下载、摘要校验、凭据隔离和测试，不携带 fork 私有文档 |

QD-003 的上游候选已经单独准备：

- 分支：`codex/upstream-cross-tab-coordination`
- 基线：`upstream/dev` = `f6651f0`
- 本地提交：`6047615`
- 范围：仅 `TabCoordinator`、WebClient 接入、双语文案、Node 测试、ESM 构建兼容
- 验证：`npm test` 6/6、`npm run build`、真实 Chrome 同源双标签接管均通过
- 未完成：远端推送仍被策略拒绝，尚未创建 upstream PR

QD-114 的上游候选已经单独准备：

- 分支：`codex/upstream-qd-114-spki`
- 基线：`upstream/dev` = `f6651f0`
- 本地提交：`13559e6`
- 范围：SPKI 解析/校验与单测、Qt 设置页、Host/Client 子进程参数、
  Windows 服务安装参数、英文/中文部署文档
- 验证：纯 C++ 单测通过、翻译 `.ts` XML 校验通过、`git diff --check` 通过
- 未完成：该候选自身的完整 Qt 构建、Inno Setup 编译、Windows 服务和真实 `wss://` 回归；
  远端推送仍被策略拒绝，尚未创建 upstream PR

QD-212 的上游候选已经单独准备：

- 分支：`codex/upstream-qd-212-local-device`
- 基线：`upstream/dev` = `f6651f0`
- 本地提交：`f1ce096`
- 范围：设备显示 JS、我的设备/收藏接线、中英文、QML QuickTest 与最小 CMake 测试入口
- 验证：干净上游基线完整 Qt 构建通过、QML 测试 1/1、`QuickDesk_qmllint` 退出 0、
  `git diff --check upstream/dev...HEAD` 通过
- 未完成：远端推送仍被策略拒绝，尚未创建 upstream PR

QD-213 的上游候选已经单独准备：

- 分支：`codex/upstream-qd-213-access-code-refresh`
- 基线：`upstream/dev` = `f6651f0`
- 本地提交：`6928d24`
- 范围：单设备访问码回源、Qt 点击自动续连、并发请求合并、失败提示与集成测试
- 验证：干净上游基线完整 Qt 构建通过、设备管理集成测试 1/1、
  `QuickDesk_qmllint` 退出 0、`git diff --check upstream/dev...HEAD` 通过
- 未完成：远端推送仍被策略拒绝，尚未创建 upstream PR

QD-214 的上游候选已经单独准备：

- 分支：`codex/upstream-qd-214-file-transfer-errors`
- 基线：`upstream/dev` = `f6651f0`
- 本地提交：`94d3ec3`
- 范围：宿主错误码解释、type=2 排查提示、上传取消协议值、Node 测试入口
- 验证：WebClient `npm test` 4/4、`npm run build`、`git diff --check` 通过
- 未完成：真实 Windows 宿主日志、失败目录/磁盘/权限复现和根因修复；
  远端推送仍被策略拒绝，尚未创建 upstream PR

QD-215 的上游候选已经单独准备：

- 分支：`codex/upstream-qd-215-win-runtime-check`
- 基线：`upstream/dev` = `f6651f0`
- 本地提交：`f449b87`
- 范围：x64 PE、Qt 插件、VC Runtime、Qt 直接依赖校验，以及 publish/package 缺依赖失败关闭
- 验证：Windows 11 正常发布包通过；缺 `vcruntime140_1.dll`、`platforms/qwindows.dll`
  或 `Qt6Network.dll` 时均拒绝；候选只有 3 个文件
- 未完成：远端推送仍被策略拒绝，尚未创建 issue #7 PR；原始坏安装包未取得

### 17.3 需要上游 Remoting 处理的 PR

以下能力不在本 fork 强行修改预编译 Remoting，先准备需求、协议草案、验证用例，再交给维护者：

| 能力 | 来源 | 为什么不能只改 Qt/Web |
|---|---|---|
| 浏览器可用 host 文件操作通道 | QD-002 | WebClient 目前只能走 `filetransfer-*`，完整目录管理需要 Remoting 数据通道或宿主命令协议 |
| Host/Relay 权威自动录制 | QD-004 | 当前 PR #23 只做控制端录制；Host 侧录制需要 Remoting/宿主生命周期 |
| 双向图片剪贴板 | QD-007 | `ClipboardEvent` 当前只承载 UTF-8 文本，需要新格式/分片协议 |
| 真实证书校验 | QD-114 | `quickdesk_network_service.cc` 的 CertVerifier 管道在 Remoting 内 |

### 17.4 明确留在 fork 的内容

- 本 fork 的品牌、产品名称、部署地址、默认服务器和运营配置。
- 上游不接受的产品化页面，若依赖 fork 专属业务数据或品牌，先保留在 fork，再把可复用的底层 API 抽成上游 PR。
- 旧 Go-Remote 历史迁移文档、回退说明和本仓库的运营清单，不进入上游。

### 17.5 当前判断

上游不是要我们另起一套远程桌面协议，而是接受“控制面、MCP、可靠性和通用客户端能力”的增量贡献。当前最合理的组合是：

- 继续使用上游 Chromium Remoting 作为核心，不在 fork 里重写媒体栈；
- 把 QD-003、连接历史、TLS 临时修复先做成上游可审查的小 PR；
- 把 QD-212 本机标识作为 issue #19 的第一个独立增量，先合这个再做访问码/上传问题；
- 把 QD-213 访问码回源自愈作为 issue #19 的第二个独立增量；
- 把 QD-214 错误诊断作为独立增量合并，同时收集宿主日志继续追真实上传失败；
- 把 QD-005 的 Qt 更新消费作为 PR #25 的后续增量，先保证跨平台安装行为和签名链；
- 把 Web 文件管理器、Host 自动录制、图片剪贴板整理成 Remoting 需求，由维护者评估；
- 把 PR #25 的控制面能力先等上游审查，避免重复提交；
- fork 的 Web/Qt 业务页面在 #25 合并后按页面拆成增量 PR，而不是塞进一个巨型控制面 PR。
