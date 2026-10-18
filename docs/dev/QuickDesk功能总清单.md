# QuickDesk 功能总清单

> 最后对账：2026-10-18 13:21 CST。上游与 fork 状态都是动态的，实施前必须重新执行
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
- 本地代码基线：`20d02cc`（QD-610 IANA timezone-aware cron schedules）；`origin/dev` 与
  `origin/master` 仍在 `9837194`，本地领先数量以 `git rev-list --count origin/dev..dev`
  的实时值为准
- 上游基线：`barry-ran/QuickDesk` 的 `dev` 和 `master` 当前均为 `f6651f0`
- 上游维护口径：持续维护但暂缓更新，下一阶段重点偏 MCP；外部贡献从 `dev` 切分支，完成后合入 `dev`，维护者定期合入 `master`
- Remoting 边界：`quickdesk-remoting` 需要编译整个 Chromium，涉及 Remoting 的改动优先向上游提需求，不要求贡献者本地编译
- 浏览器技能桥边界：预编译 Remoting 已带 `qd-skill-bridge`，WebClient 可通过
  control DataChannel 发送扩展消息，经 Host 转发到 `quickdesk-skill-host`；Web 文件
  管理器不需要重新编译 Chromium
- 当前主要增量：QD-214 的 SPAKE2 兼容修复与 Windows Host 文件传输根因定位、
  QD-215 Windows 发布依赖校验（`67bf11f`、`556f86e`）、QD-607 批量远控清场（`5028068`）、QD-204 地址簿与设备分组
  （`f270487`、`b1a1155`、`162ac6e`、`b013a62`）、QD-508 Qt 录像回放
  （`74c7022`）、三端管理面实时消费、`a982a10` 的 Windows 更新闭环，以及
  `c77dcae` 的 SQLite 轻量单进程模式和 `64ab93a` 的备份/恢复、容器备份二进制、
  录像/安装包总量配额及管理存储视图；Web 屏幕墙已补单瓦片音频独占；QD-604 已补齐
  Host 指标上报、阈值 episode、通知 outbox、Web/Qt 规则入口和七套 SDK；QD-911 的
  根目录 `SECURITY.md`、服务端 QD-604 stacked 候选，以及 QD-115 的上游安全边界文档
- QD-002 / QD-403 Web 文件管理器已补齐：利用现有 `qd-skill-bridge` 浏览目录、新建、
  重命名、版本保留软删除、文本预览及 64 KiB 分块上传下载；host `file-ops` 新增
  `read_file_chunk`、`write_file_chunk`、`trash_path` 和 `get_default_directory`
- QD-610 已补控制面事件订阅：Qt `CloudDeviceManager` 只向本机 MCP 广播白名单事件，
  MCP 支持 `event_type` / `event_device_id` / `event_filter`、设备 ID 归一化、
  claim-before-run 和事件 ID 幂等；SQLite 模式重启后不依赖会重置的 `server_rev`
- QD-610 已补 cron：六段/七段标准表达式和 `@hourly` 等简写由 `cron` crate 校验，
  `chrono-tz` 提供 IANA 时区和 DST；`interval_seconds` 与 `cron_expression` 互斥，
  时间规则可与事件规则共存，错过槽位不补跑
- fork PR：<https://github.com/ilh-github/go-remote-next/pull/1>，OPEN / MERGEABLE /
  CLEAN，7 项检查全部成功；它的 head 是早期最小分支，不等于当前完整 `dev`
- 服务端：Go SignalingServer
- 桌面客户端：Qt 6 + QML
- Web 客户端：Vue 3 外壳 + 原生 `remote.html`
- 核心控制引擎：QuickDesk / Chromium Remoting
- 持久化：PostgreSQL（多实例）/ SQLite（轻量单进程）
- 实时状态与令牌：Redis（多实例）/ 进程内 Redis（轻量单进程，重启即失）

## 3. 当前迭代

| ID | 优先级 | 功能 | 状态 | 上游 | 验收标准 |
|---|---|---|---|---|---|
| QD-001 | P0 | 建立独立功能总清单 | `[x]` | Fork-only | 文件覆盖全部范围、状态、优先级和验收口径 |
| QD-002 | P0 | Web 控制端完整文件管理器 | `[x]` | UP-Ready | 经 `qd-skill-bridge` 完成目录浏览、新建、重命名、版本保留软删除、文本预览、分块上传下载；62 项 WebClient 测试、生产构建和桌面/移动浏览器烟测通过 |
| QD-003 | P0 | 多标签控制与音频抢占 | `[x]` | UP-Ready | 同一设备多个同源标签页只有一个控制者/音频播放者；新标签接管后旧标签立即释放 |
| QD-004 | P0 | Host/Relay 权威自动录制 | `[ ]` | UP-Remoting | 被控端或中继自动录制，断线自动收尾并上传索引；断线收尾的控制端半边已由 QD-505 补齐 |
| QD-005 | P0 | 客户端自动更新消费 | `[~]` | UP-PR#25 + Upstream-Open | Windows 已通过检查、下载、SHA-256 校验、静默安装、自动重启、版本落盘和服务存活真机闭环；macOS 已实现受校验的 DMG 挂载、暂存替换、失败回退和重开，并通过 ad-hoc 真实 DMG E2E；剩余 Developer ID 签名、Apple 公证和公证后替换回归 |
| QD-006 | P1 | SQLite 轻量服务端部署 | `[x]` | UP-PR#25 | 单进程、单文件 SQLite、默认进程内 Redis；`/health` 区分 sqlite，空目录启动和重启持久化 E2E 通过 |
| QD-007 | P1 | 图片剪贴板 | `[ ]` | UP-Remoting | 双向图片格式协商、分片传输、权限和边界测试 |

最近验证：

- 本轮代码提交：`71b899e`（QD-316 会话内文字聊天）、`2e09d5b`（QD-610 控制面事件触发器）、`9f7be0f`（QD-610 HMAC 结果回调）、`e8ce807`（QD-610 异步任务队列/取消）、`e518194`（QD-610 Idempotency-Key）、`3f94911`（QD-610 持久化/定时触发器）、`bca8149`（QD-610 HTTP trigger）、`de88f07`（QD-506 Web 录像分块续传）、`001a76c`（QD-510 工作流核心）、`e688869`（访问码同步重试）、`5be14da`（SPAKE2 与传输错误码）、
  `a982a10`（Windows 自动更新重启与缓存清理）、`c77dcae`（SQLite 轻量模式）、
  `64ab93a`（备份恢复与存储配额）、`37dd8aa`（Host 指标告警）、`3021ca9`
  （私密安全披露策略）、`3474203`（信令信任边界）；
  `dev` / `master` 在其后同步本轮验证；
-  当前领先数量以 `git rev-list --count origin/dev..dev` 为准；
  QD-215 主线和上游最小候选均已提交并推送。
- 上游核对：`upstream/dev` 已前进到 `5db316c`（skill-host 邻近运行时修复）；
  fork `dev` 已合并该基线；当前上游 open issues 10 个、
  open PR 6 个（新增本轮的 #26 file-ops 分块与版本保留软删除、#27 WebClient 文件管理器，
  两者为 stacked 关系）。
- 上游维护：issue #21 已由维护者回复“持续维护、暂缓更新、重点偏 MCP、欢迎从
  `dev` 切分支贡献、Remoting 改动由维护者处理”；不能再把“暂缓”理解成停止维护。
- 上游 PR：#22、#23、#24、#25 均为 OPEN / MERGEABLE，但 GitHub 返回的
  `mergeStateStatus=UNSTABLE` 且没有 reported checks；不能把 MERGEABLE 写成 CI 已通过。
- 清单统计：117 个唯一 QD ID；82 个 `[x]`、20 个 `[~]`、11 个 `[ ]`、
  1 个 `[!]`、2 个 `[>]`、1 个 `[-]`。
- 上游待办已全部映射到本清单：10 个 open issue、6 个 open PR、3 个未完成 Roadmap
  项均有对应 QD、处理边界和上游贡献方式；上游已完成的 Roadmap 项只作为继承基线，
  不重复开发。
- 本次合并后的实施结论：上游没有新增必须立即重复开发的 issue；`QD-608`
  （C#/PHP/Kotlin SDK）和 `QD-002` / `QD-403`（Web 文件管理器）已完成。可独立回贡的
  QD-002/QD-003/QD-114/QD-212/QD-213/QD-214/QD-215/QD-316/QD-911 继续按最小 PR 保持候选；
  QD-004、QD-007 和 QD-214 的 Windows 用户上下文根因继续归入 Remoting/宿主需求；
  QD-505 的控制端断开收尾已完成。
- 融合结论：本清单的 3.1 决策台账负责固定“做/不做/等待/交 Remoting”，3.2 执行队列负责
  排序，第 16、17 节负责逐项追溯上游来源和 PR 交付方式。三者共同作为后续开发和回贡的
  唯一工作口径，不再另建一套 fork 待办。
- 上游 `docs/dev/*技术方案.md` 里有部分历史未勾选实现清单，抽查当前代码后多数已经落地；
  这些文件是设计历史，不直接当新增待办。权威待办仍以 README Roadmap、open issue、open PR
  和维护者最新回复为准。
- file-ops `cargo test --offline -p file-ops -- --nocapture`：4/4 通过，覆盖分块往返、
  Base64 错误、最大 256 KiB 截断、重复软删除和默认目录。
- file-ops `cargo clippy --offline --workspace --all-targets -- -D warnings`：通过，
  显式关闭自动 truncate，首块仍由 `set_len(0)` 控制。
- WebClient `npm test`：62/62 通过，新增 skill bridge 请求关联/错误/超时、路径模型、
  分块上传下载和重命名测试。
- WebClient `npm run build`：通过，JS 378.74 kB，gzip 119.96 kB。
- QD-511 状态校验：`cargo test` 46/46、`cargo clippy -D warnings` 通过；新增
  `set_step_expectation` MCP 工具和屏幕断言翻译/解析测试。
- QD-505 断线收尾：新增 `RecordingLifecycle` 判定与 `quickdesk_recording_lifecycle_test`，
  覆盖终态/非终态、无关设备和重复事件；CTest 14/14 通过；`MainController` 在信令进入
  终态时调用 `stopRecording()`，QML 改走 `toggleSessionRecording()`。
- Web 文件管理器 Chrome 烟测：桌面 1280x720 与移动 390x844 均打开成功；目录行、
  操作按钮、文本预览和响应式布局无重叠，截图见 `docs/e2e/e2e-web-file-manager*.png`。
- QD-505 编译验证：`MainController.cpp` 编译成功，最终链接仍因既有 `framework 'AGL' not found` 失败。
- `remote-main` 接线烟测：收到完整 `capabilitiesReady` 后 `_supportsFileManager=true`，
  浮动菜单“远程文件”从隐藏变为可点击；Qt `RemoteFileManager.cpp` 编译通过，
  `ctest` 13/13 通过；完整 App 最终链接仍因预编译 Remoting 引用已移除的 macOS `AGL`
  framework 失败，属于既有外部依赖兼容问题。
- Chrome 同源双标签实测：后开标签接管控制后，前一个标签释放控制；接管音频后，前一个标签释放音频。
- QD-114 SPKI 纯 C++ 单测：通过；覆盖标准/URL-safe Base64、无 padding、重复项、
  非法项和 Chromium 参数构造。
- QD-114 上游候选：`codex/upstream-qd-114-spki` = `13559e6`，基于 `upstream/dev`，
  17 个文件；fork 侧 Qt 6.8.3 构建环境已可用，但该上游候选自身的完整 Qt 构建、
  Windows 安装器编译和服务回归待补。
- QD-005 功能提交：`cae77ff`。Qt 客户端已有启动后自动检查和每 6 小时复查、手动检查、
  更新提示、下载进度/取消、大小与 SHA-256 校验、可选版本推迟、强制更新、同源 API key
  下载和安装器启动；非同一 origin 的下载不会泄漏 API key。
- QD-005 Windows 收口提交：`a982a10`。应用内更新改为传 `/LAUNCHAPP`，安装器在静默升级
  完成时显式启动新版本；新版本启动时清理不高于当前版本的下载目录，保留尚未安装的更高
  版本缓存。
- QD-005 macOS 更新助手：`QuickDesk/scripts/macos_update_helper.sh` 通过资源打包，校验
  bundle identifier、可执行文件、版本、TeamIdentifier、内嵌签名和 SHA-256 已下载产物；
  先挂载只读 DMG，
  再把新应用暂存到更新目录，使用同目录备份完成替换，打开失败时回退旧版本。无写权限时
  通过 macOS 管理员授权执行，限制清理路径只能位于更新工作目录或目标 `.app` 备份。
- QD-005 macOS 测试：`scripts/e2e/test-quickdesk-macos-update-helper.sh` 使用真实 UDZO
  DMG 和两个临时 `.app` 验证正常替换、版本落盘、DMG 清理，以及 bundle identifier
  不匹配时拒绝替换；`quickdesk_update_manager_test` 同时断言更新助手已嵌入 Qt 资源。
- QD-005 发布门禁：`publish_qd_mac.sh` 支持 `MACOS_SIGN_IDENTITY` hardened runtime
  签名，`notarize_qd_mac.sh` 支持 keychain profile 或 Apple ID 凭据完成 notarytool、
  staple 和 Gatekeeper 验证；`QUICKDESK_REQUIRE_MACOS_NOTARIZATION=1` 时缺少证书或
  公证凭据会直接失败。
- QD-005 测试：`quickdesk_update_support_test` 与 `quickdesk_update_manager_test` 通过；
  后者使用真实本地 HTTP 服务覆盖检查、下载、摘要不匹配拒绝、残件清理和跨源凭据隔离。
- QD-006 / QD-809 功能提交：`c77dcae`。`DB_DRIVER=sqlite` 使用单文件 SQLite；
  `REDIS_DRIVER` 留空时自动切到进程内 Redis，不再依赖外部 PostgreSQL/Redis。轻量模式
  强制迁移 SQLite schema，关闭计划任务/跨实例日志等不适合单进程的后台能力；通知投递
  worker 保留运行，因为它只消费同一 SQLite 中的持久 outbox，QD-604 指标告警依赖该路径。
  `/health` 会返回 `sqlite` 而不是 `postgres`。
- QD-006 / QD-809 验证：`scripts/e2e/test-quickdesk-minimal-server.sh` 从空临时目录构建
  并启动真实二进制，使用不可达的 PostgreSQL/Redis 地址证明二者未被依赖，验证
  `/health`、SQLite 文件创建、进程重启后数据库仍存在；全量 `go test -p 1 -count=1 ./...`、
  `go test -race ./internal/database`、`go vet ./...` 和 `git diff --check` 均通过。
- QD-006 上游候选：`codex/upstream-qd-006-sqlite` = `7758456`，基于 PR #25 的
  `feat/signaling-control-plane@8998466`，只含 SignalingServer SQLite/embedded Redis、
  配置、健康检查、测试和 E2E 脚本；候选自身完整 Go 测试、database race、`go vet`、
  Web 生产构建和最小服务 E2E 均通过。PR #25 合并后必须 rebase 到 `upstream/dev` 再开 PR。
- QD-006 边界：进程内 Redis 是易失的，重启会清空 refresh token、presence、限流和实时流
  续传状态；SQLite 模式只支持一个信令进程。此边界已写入 `SignalingServer/README.md`，
  不能把本地最小模式描述成多实例生产部署。
- QD-810 功能提交：`64ab93a`。新增 `quickdesk-backup backup|restore`：SQLite 使用
  `VACUUM INTO` 生成一致性快照，与 `RECORDING_DIR`、`UPDATE_ARTIFACT_DIR` 一起写入
  gzip tar；manifest 记录格式版本、文件集合、大小和逐文件 SHA-256。恢复拒绝绝对路径、
  `..`、符号链接和重复文件，摘要不符不落盘；`-force` 会把旧数据库/媒体目录改名为
  `*.pre-restore-*`，任一后续目标安装失败时已替换目标回滚。
- QD-810 配额：`RECORDING_STORAGE_QUOTA_GB` 和 `UPDATE_STORAGE_QUOTA_GB` 统计真实普通
  文件（含 `.part`），单次、可恢复上传和管理端安装包上传都接同一配额闸门。
  `GET /v1/admin/recordings:storage` 同时返回索引字节与磁盘字节；新增
  `GET /v1/admin/update-releases:storage`；管理端升级通道页展示已用、配额、剩余和超限。
- QD-810 验证：备份包内数据库行、录像字节和安装包字节完整往返；路径穿越、manifest
  SHA-256 篡改、强制恢复旧副本、后续媒体目标不可写时数据库回滚均有真实文件系统测试；
  `bash scripts/e2e/test-quickdesk-backup-restore.sh` 用真实轻量服务生成数据、恢复后重新
  启动健康；全量 Go 测试、backup/service/handler race、`go vet`、管理端生产构建和
  Linux `CGO_ENABLED=0` 两个二进制的 amd64 交叉编译均通过。
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
- WebClient 的 SPAKE2 Alice 现在复刻 BoringSSL password-scalar 的低三位兼容修正。
  固定字节向量与 Chromium 140 对齐；真实 macOS Host 和 Windows Host 均通过验证哈希，
  DataChannel、视频和数据通道不再被 `Verification hash mismatched` 阻断。
- QD-214 根因已定位，不再归因于 WebClient：
  - macOS Host 上传 37 字节 probe 成功，落盘文件字节和 SHA-256 与发送内容一致；
  - Windows Host 以普通交互用户启动时，`WTSQueryUserToken(WTS_CURRENT_SESSION)` 返回
    `0x522`，Chromium 映射为 `type=2 UNEXPECTED_ERROR`；
  - Windows Host 以 SYSTEM/session 0 启动时没有交互桌面用户，Chromium 返回
    `type=7 NOT_LOGGED_IN`。
  - 正确修复点位于 Host 服务/宿主生命周期或 Remoting 的用户上下文解析，不应在
    WebClient 文件传输协议里伪造成功。
- WebClient 文件传输错误现在覆盖 Chromium 协议 0~7：取消、保存失败、协议错误、
  权限拒绝、磁盘不足、IO 错误和未登录。上传取消继续使用协议值 1。
- QD-214 SPAKE2 上游候选：`codex/upstream-qd-214-spake2-webclient` = `ed5437b`，基于
  `upstream/dev`；固定向量 1/1、生产构建通过。
- QD-214 错误码上游候选：`codex/upstream-qd-214-file-transfer-errors` = `60837af`，
  基于 `upstream/dev`；0~7 错误解释测试 6/6 通过。该候选改善诊断，不代表上传根因已修。
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
  `openapi-smoke.mjs` 输出 `PASS: 388 documented assertions`；WebClient 37/37 与生产构建
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
- Windows 更新闭环：Parallels Windows 11 中重新构建并安装带 `a982a10` 修复的 4.2.8.0
  基线包（90,095,272 字节，SHA-256
  `0249d1d3573e26b4bba9064e0cc8879bf94301c29c50c71c11ed426a1c1830a3`）；
  客户端真实发现并下载 4.2.9.0 升级包（90,084,629 字节，SHA-256
  `8e5e32cb038f5f6757bef58e2ed0907c5918e131e0d118d61de41f3090379383`），
  下载文件摘要一致。点击“安装并重启”后，安装器完成升级，客户端进程从 PID `12460`
  退出并自动启动新 PID `6412`；注册表、关于页均为 `4.2.9.0`，QuickDeskHost 服务继续
  RUNNING，更新临时目录已清理。QD-005 / QD-808 的 Windows 半边关闭，状态仍因 macOS
  DMG 替换、签名和公证未验收而保持 `[~]`。
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
- QD-702 墙级音频：屏幕墙新增单瓦片音频按钮和 `WallAudioSelector`，同一时刻只允许
  一个 iframe 播放声音；切换、移除、重连、设备离线或会话失败时会先向旧瓦片发送关闭
  消息，活动瓦片在画面恢复后会自动重新应用音频状态。`remote-main.js` 在墙模式下默认
  关闭音频，仅在墙页选中后开启远端音频并解除 `<video>` 静音；晚到的音频轨道会合并到
  已挂载的媒体流，且不会重复添加。
- QD-702 验证：严格 TDD 先新增 `wallAudio.test.mjs` 和 `mediaStream.test.mjs`，分别
  确认模块缺失 RED；实现后两组共 9/9 通过。WebClient 全量 `npm test` 46/46 通过，
  `npm run build` 通过，
  生产构建 JS 363.42 kB、gzip 117.06 kB。
- QD-604 功能提交：`37dd8aa`。新增 device-secret 上报接口
  `POST /v1/devices/:id/metrics` 和 Owner 规则
  `GET|PUT|DELETE /v1/me/devices/:id/metric-alert`；每台设备只保留一个最新快照，
  CPU、内存、磁盘各自维护 `exceeded_since` / `last_alerted_at` episode，避免一个指标
  恢复时重置其他指标。Host 通过内部 `get_system_info` 请求采集，响应被
  `SkillHostManager` 截流，不泄漏给远端控制端。
- QD-604 客户端：Qt 通知页和 Web `/notifications` 都显示最新 CPU/内存/磁盘、挂载点、
  采集时间与 episode 状态，并可设置 0~100 阈值、`sustained_minutes=0` 首样本立即告警、
  重复提醒和 snooze。Web 同时校验 0~10080 分钟上限。
- QD-604 验证：SignalingServer 全量 `go test -p 1 -count=1 ./...`、相关包 race、
  `go vet ./...` 通过；真实 SQLite 服务 E2E 创建账号、设备和 device secret，上报 91.5%
  CPU 样本，确认持久 outbox 把 `device.metric.alert` 投递到真实 Webhook；WebClient
  47/47、生产构建、七套 SDK freshness、QuickDesk Qt 6.8.3 Debug 完整构建、
  `QuickDesk_qmllint`、离屏启动和 12/12 CTest 通过。
- QD-608 功能提交：`f8927d7`（`feat(sdk): add C# PHP and Kotlin clients`）。OpenAPI 描述更新为
  七套生成客户端；新增 `sdk/csharp`、`sdk/php`、`sdk/kotlin`，生成器支持
  `csharp`/`php`/`kotlin`，新鲜度检查覆盖七种语言。C# 项目固定 solution GUID，PHP
  使用 Composer/Guzzle 包并移除 TODO 测试，Kotlin 使用 Gradle/JVM OkHttp4 客户端。
- QD-608 验证：C# `dotnet build -c Release` 通过（3 个生成代码弃用警告，0 error）；
  PHP `composer validate`、`composer install --no-dev` 和 76 个生成 PHP 文件 `php -l`
  通过；Kotlin `gradle --no-daemon build` 通过；七种语言 freshness 全部
  `matches a fresh ... generation`；workflow YAML 解析和脚本 `bash -n` 通过。
- QD-608 上游候选：`codex/upstream-qd-608-extra-sdks` = `cca9e44`，基于 PR #25 的
  `feat/signaling-control-plane@8998466` 重新生成全部七套客户端；候选自身七种语言
  freshness、C# build、PHP Composer/lint（61 个生成文件）和 Kotlin Gradle build 均通过。
- QD-510 功能提交：`001a76c`。步骤会记录 `remote_api` 或 `skill_host` 执行通道，旧工作流
  缺少通道字段时默认 `skill_host`；远程鼠标、键盘、剪贴板和文本点击可按原通道回放。
  工作流新增 `format_version=1` 兼容读取，参数支持递归 `{{name}}` 占位符和旧版顶层
  同名覆盖。每次回放持久化成功、失败、跳过和耗时，安全失败始终中止，普通失败由
  `stop_on_failure` 控制；新增 `list_workflow_runs` / `get_workflow_run` 查询入口。
  `cargo test` 19/19 通过。预录/后录屏幕状态校验和智能回放另列 QD-511。
- QD-510 上游候选：`codex/upstream-qd-510-workflow` = `0a9ecd5`，直接基于
  `upstream/dev@f6651f0`，包含 5 个 MCP 源文件和 3 个通用 MCP 文档，不含 fork 清单、
  自托管记录或业务控制面；候选自身 19/19 测试、Clippy 零告警和
  `git diff --check upstream/dev...HEAD` 均通过。
- QD-506 功能提交：`de88f07`。在线会话录像改为按 4 MiB 分块 PATCH 上传，服务端返回的
  `Upload-Offset` 是唯一权威进度；409 会从服务端已有 offset 续传，瞬时网络错误和
  429/5xx 重试但不推进本地 offset，4xx 立即失败。上传按钮显示进度并在上传期间禁用，
  失败继续把索引标成 `failed`。WebClient 50/50 单测和生产构建通过；真实 SQLite
  SignalingServer smoke 上传 196625 字节、4 个分片，完成状态和逐字节回读一致。
- QD-506 smoke 自举提交：`eba7ae2`。真实服务脚本在源码树没有 `web/dist` 时创建临时嵌入文件，
  构建后清理，不要求先构建管理端；脚本自身通过 `bash -n`。
- QD-506 上游候选：`codex/upstream-qd-506-recording-upload` = `de45538`，基于 PR #23
  当前 head `00a7cb8`。候选只含 Web 上传器、录制页接线、测试、E2E 脚本和候选基线所需
  的 `type: module` / ESM copy-legacy 配置；候选自身 3/3 单测、生产构建和 diff check 通过。
  PR #23 基线早于 QD-006，没有 SQLite 驱动，因此真实分片 E2E 在 fork 主线执行，候选
  不复用 fork 后续 SQLite 代码。
- QD-610 第一阶段功能提交：`bca8149`。`quickdesk-mcp --transport http` 新增
  `POST /v1/workflows/{workflow_id}/run` 和 `GET /v1/workflow-runs/{run_id}`，使用独立
  `QUICKDESK_TRIGGER_TOKEN`，执行前检查设备白名单，继续经过 trust/emergency 层，并返回
  已持久化的运行报告；进程内限制单条工作流并发，忙碌返回 429。fake QuickDesk WS 的
  HTTP→replay→report E2E、错误 token、设备白名单和 429 测试均通过，MCP 全量 23/23、
  Clippy 零告警和 release 构建通过。定时触发、控制面事件订阅、幂等键、出站回调和委托
  权限仍未完成，QD-610 保持 `[~]`。
- QD-610 上游 stacked 候选：`codex/upstream-qd-610-business-trigger` = `1be154c`，
  基于 `codex/upstream-qd-510-workflow@0a9ecd5`。候选现已叠加 HTTP trigger、持久化
  interval/IANA 时区 cron trigger、Idempotency-Key、异步 job 队列/取消/单进程重启恢复、
  HMAC 持久 outbox/重试/死信/重放/ack、Qt 白名单事件桥和 MCP event trigger；候选自身
  47/47 测试、Clippy、release 构建和 `git diff --check upstream/dev...HEAD` 通过。
  候选保留通用 MCP 代码和契约文档，不含 fork 清单、QD-506 或控制面业务记录。
- QD-610 第二阶段功能提交：`3f94911`。新增 `~/.quickdesk/workflow-triggers/` JSON store、
  `GET|POST /v1/triggers`、`DELETE /v1/triggers/{id}`、`POST /v1/triggers/{id}/run`；
  触发器保存工作流、设备、默认参数和可选 60 秒~365 天 interval。HTTP worker 每 5 秒
  最多领取一条到期触发器，执行前推进 `next_run_at`，执行后回写 run ID/status/error。
  手动和定时触发共用单执行槽。MCP 全量 31/31、Clippy 零告警、release 构建通过。
- QD-610 第三阶段功能提交：`e518194`。`POST /v1/workflows/{workflow_id}/run` 支持可选
  `Idempotency-Key`，相同 key、工作流、设备和参数会返回第一次持久化的 run report，
  不会再次执行；不同请求复用同一 key 或请求仍在执行时返回 409。映射写入
  `~/.quickdesk/workflow-idempotency.json`，成功、失败和进程重启后的重放均有测试。
- QD-610 第四阶段功能提交：`e8ce807`。新增 `POST /v1/workflows/{workflow_id}/run-async`、
  `GET /v1/workflow-runs/{run_id}/status` 和 `POST /v1/workflow-runs/{run_id}/cancel`；
  job 状态持久化到 `~/.quickdesk/workflow-jobs/`，进程内队列容量 32，异步 worker 与
  同步/定时触发共用单执行槽。运行中取消会中止当前 WebSocket 请求并写入
  `cancelled=true` 的报告。MCP 全量 34/34、Clippy 零告警和 release 构建通过。
- QD-610 第五阶段功能提交：`9f7be0f`。同步、异步和持久化 trigger 支持 `callback_url`；
  `QUICKDESK_CALLBACK_SECRET` 存在时发送 `workflow.run.completed` 或
  `workflow.run.failed`，带 timestamp、delivery ID 和 HMAC-SHA1 签名。真实本地 HTTP
  接收端测试覆盖同步、异步和定时 trigger；HMAC 构造使用 RFC 2202 固定向量校验。
  MCP 全量 36/36、Clippy 零告警和 release 构建通过。回调当前只允许 HTTP，是单次
  best-effort，尚无持久 outbox/重试。cron/日历、控制面事件、跨进程队列恢复和委托权限
  仍未完成，QD-610 保持 `[~]`。
- QD-610 第六阶段功能提交：`2e09d5b`。Qt `CloudDeviceManager` 将已通过
  snapshot/replay 与 `server_rev` 顺序校验的控制面事件收窄为白名单，并转发为本机
  `controlPlaneEvent`；桥接层只保留 `id/type/device_id/data/ts/server_rev`，并从
  `data.device_id` 归一化实时信封没有稳定提供的设备字段。MCP 新增事件 trigger 字段、
  顶层过滤器、claim-before-run、设备/用户级事件区分和白名单校验。事件 `id` 是跨重启
  幂等键，没有 ID 时才使用 `server_rev`，避免 SQLite 进程内 Redis 重启后永久停摆。
  MCP 39/39、Clippy 零告警、release 构建通过；Qt 事件桥单测通过，完整 Qt Debug
  编译到 100%，最终链接受预编译 Remoting 依赖缺失旧 `AGL` framework 阻塞，不是本轮
  源码错误。该阶段结束时回调仍是 best-effort，cron、事件停机补发、多进程共享 worker、
  确认操作者归属和委托权限仍未完成，QD-610 保持 `[~]`。
- QD-610 第七至第十阶段功能提交：`1176a92` 补持久回调 outbox、同一 delivery id 重试、
  重启恢复和死信状态；`9f75331` 补受保护的死信查询与人工重放；`a94bba1` 补 delivered/dead
  的幂等 ack；`d3dc4cf` 补 queued job 单进程重启恢复和 running fail-closed。
- QD-610 第十一阶段功能提交：`3c31d9a`。新增 UTC `cron_expression`，使用 `cron`
  crate 解析六段/七段表达式和常用简写；`interval_seconds` 与 cron 互斥，任一规则可与
  `event_type` 同时使用。`claim_due` 先推进 `next_run_at` 再执行，仍为 at-most-once；
  MCP 停机期间错过的槽位不补跑。MCP 全量 53/53、Clippy 零告警、release 构建和
  diff check 通过。IANA 时区、DST、节假日、事件停机补发、多进程共享 worker、确认
  操作者归属和委托权限仍未完成，QD-610 保持 `[~]`。
- QD-610 第十二阶段功能提交：`20d02cc`。cron trigger 新增可选 IANA `timezone`，
  省略时规范化为 UTC；使用 `chrono-tz` 计算本地日历与真实 DST 偏移。春季不存在的
  本地时间跳到下一有效槽位，秋季重复时间只执行第一次，继续保证 at-most-once。非法
  时区、以及没有 cron 却提供 timezone 的请求返回 `400`。MCP 全量 54/54、Clippy、
  release 构建和 diff check 通过。法定节假日、事件停机补发、多进程共享 worker、确认
  操作者归属和委托权限仍未完成，QD-610 保持 `[~]`。
- QD-911 当前状态：上游 issue #6 仍 OPEN，GitHub API 返回
  `private-vulnerability-reporting.enabled=false`，维护者最新回复仍只是旧 release 链接。
  fork 与上游候选已新增根目录 `SECURITY.md`，覆盖支持版本、公开 issue 禁令、GitHub 私密
  报告优先路径、维护者公开邮箱兜底、范围内外、协调披露和 maintainer 检查项。
- QD-911 候选：`codex/upstream-qd-911-security` = `9862717`，直接基于
  `upstream/dev@f6651f0`，只有 `SECURITY.md` 1 个文件；结论不能标记上游已启用，
  因此在清单中保持 `[~]`，等 PR 合并且仓库启用 Private Vulnerability Reporting 后才可 `[x]`。
- QD-115 设计提交：`3474203`。新增
  `docs/security/signaling-trust-model.md`，明确 `device_secret` 只有 Argon2id 哈希、
  服务器不能从数据库恢复；但 `access_code` 明文存储、signal token/审批/设备归属都由
  服务器控制，因此恶意或被攻陷的信令服务器仍是设备访问信任方，当前协议不能保证其
  无法冒充 Controller。
- QD-115 目标方案：Host 生成独立设备身份密钥，Controller 使用公钥，Owner 给
  device/permissions/expiry/nonce 签名，Host 在建立媒体/数据通道前本地验证 Owner 签名
  与 Controller challenge。服务器降级为发现、排队和路由；token 只是路由凭证。
- QD-115 上游候选：`codex/upstream-qd-115-trust-model` = `1ca2543`，直接基于
  `upstream/dev@f6651f0`，只含安全边界文档；候选 `git diff --check` 通过。QD-115
  保持 `[~]`，直到 8 条协议验收项全部完成。
- `git diff --check -- ':!sdk'`：通过；生成 SDK 保留 openapi-generator 自带的文档尾随
  空格，不能手改，否则 freshness 校验会失败。
- 远端推送仍被当前执行策略拒绝，本地 `dev` 尚未同步到 GitHub。

### 3.1 统一需求决策台账

这张表是本轮对“我们自己的清单 + 旧 Go-Remote 缺口 + 上游待办”的合并结论。
它不替代后面的完整功能表，只固定优先级、依赖和上游贡献方式。

| 队列 | 工作项 | 来源 | 当前判断 | 下一步 |
|---|---|---|---|---|
| 上游同步 | PR #22 自动审批线程亲和性 | `Upstream` / QD-113 | 上游 PR 已覆盖 issue #17，目标分支为 `master`；本 fork 不重复实现 | 等 PR #22 合并后先同步上游 `master`，再按 `master <= dev` 的线序回灌 |
| 上游同步 | PR #23 会话录像与回放 | `Upstream` / QD-501~507 | 已覆盖服务端索引、Web/Qt 手动录制、预览、生命周期；Host 权威录制不在其中 | 不重复提交；合并后回灌 fork，再单独推进 Host 录制 |
| 上游同步 | PR #24 skill-host Windows 运行库 | `Upstream` / QD-402、QD-609 | 修复技能进程找不到 MSVC runtime；与 fork 同向 | 合并后同步，不维护第二套 PATH/stderr 逻辑 |
| 上游同步 | PR #25 自托管控制面、OpenAPI、SDK | `Upstream` / QD-101~110、201~211、501、601~608、801~808 | 当前控制面总集；维护者可接受继续拆 PR | 等审查；若维护者要求，按 access / alerting / recording / operations / update 拆 |
| 已完成 | QD-708 Qt 通知规则页 | `Fork` / `Upstream` 通用能力 | 通知通道、静默时段、设备离线告警、投递重试/确认和实时刷新已完成，提交 `dbd828f` | PR #25 合并后拆成通用 Qt 管理面候选并提交上游 |
| 已完成 | QD-508 Qt 录像回放页 | `Legacy` / `Upstream` PR #23 | Qt 录像列表、分页、缩略图、应用内播放、外部打开和删除已完成，提交 `74c7022` | PR #23 合并后作为通用 Qt 管理面候选提交上游 |
| 已完成 | QD-204 地址簿与设备分组用户入口 | `Legacy` / `Upstream` Roadmap | owner-scoped API、四套 SDK、WebClient 和 Qt 已完成；真实 service smoke 388 条断言通过 | 本地候选已准备：API `codex/upstream-qd-204-api@d2321b9`，客户端 `codex/upstream-qd-204-client@b259d03`；等远端写入恢复后建 PR |
| 已完成 / 上游候选 | QD-604 指标型告警 | `Legacy` / `Fork` / `Upstream-Open` | Host 上报、服务端 episode、通知 outbox、Web/Qt 和 SDK 已由 `37dd8aa` 完整闭环；服务端候选 `codex/upstream-qd-604-metrics@0ea9e7c` 已基于 QD-006 `7758456` 独立验证 | 推送服务端候选并作为 `#25 -> SQLite -> metrics` stacked PR；Web/Qt 待 #25 客户端基线确认后单独拆 PR |
| 本地收口 | QD-607 计划任务执行动作 | `Fork` / `Legacy` | 服务端调度、Web/Qt 管理面、重试已完成；新增 `revoke_all_remote_sessions`，可按执行时设备归属批量关闭活动远控会话（最多 200 台）；纯服务端候选 `codex/upstream-qd-607-batch-revoke@badf990` 已独立验证 | “定时连接”仍需控制端到点发起连接的协议；更广义的批量 agent 指令仍需执行与回执通道；远端写入恢复后推送候选并作为 PR #25 的 stacked PR |
| 本地收口 | QD-005 / QD-808 自动更新真机验证 | `Fork` / `Legacy` / `Upstream` PR #25 | Windows 已完成 4.2.8.0→4.2.9.0 自动下载、校验、静默安装、自动重启、版本和服务闭环，提交 `a982a10`；macOS 已实现 DMG 自动替换和失败回退，并通过 ad-hoc 真实 DMG E2E | 取得 Apple Developer ID 和公证凭据，执行 hardened runtime 签名、notarytool、staple、Gatekeeper 与真实升级回归；上游候选仍应在 PR #25/更新通道基线确认后重做最小增量 |
| 已完成 | QD-006 / QD-809 SQLite 轻量服务端 | `Fork` / `Upstream-Open` | fork 提交 `c77dcae`；候选 `codex/upstream-qd-006-sqlite@7758456` 已在 PR #25 基线上通过完整测试、race、vet、构建和 E2E | PR #25 合并后 rebase 到 `upstream/dev` 并创建正式 PR；明确单实例和易失 Redis 边界，不和多实例生产配置混在同一 PR |
| 已完成 | QD-810 备份恢复与存储配额 | `Fork` / `Upstream-Open` | 提交 `64ab93a`；SQLite 一致性归档、恢复校验/回滚、PostgreSQL 运维文档、录像与安装包总量配额、两个管理存储接口和升级通道页状态均完成 | PR #25 合并后作为自托管运维增量拆最小候选；备份 CLI 与配额边界应独立于控制面功能 PR |
| 上游候选 | QD-003 多标签控制与音频抢占 | `Legacy` / `Upstream` Roadmap 补强 | 候选 `codex/upstream-cross-tab-coordination` = `6047615`，仅 WebClient 改动 | 远端写入恢复后推送并建 PR；补截图和同源双标签验证说明 |
| 上游候选 | QD-114 自托管 WSS SPKI 兼容 | `Upstream` issue #20 / `Fork` | 候选 `codex/upstream-qd-114-spki` = `13559e6`；临时方案不替代真实证书校验 | 推送并建 issue #20 PR；补 Qt 构建、Windows 服务和真实 `wss://` |
| 上游候选 | QD-212 设备列表本机标识 | `Upstream` issue #19 / `Legacy` | 候选 `f1ce096` 已在干净上游基线构建和测试 | 推送并建 PR；与 #19 其余问题拆开 |
| 上游候选 | QD-213 缺失访问码回源 | `Upstream` issue #19 / `Legacy` | 候选 `6928d24` 已通过集成测试 | 推送并建 PR；补两真机换码时序回归 |
| 上游候选 | QD-214 SPAKE2 与文件传输错误诊断 | `Upstream` issue #19 / `Legacy` | SPAKE2 候选 `ed5437b` 修通真实 Host；错误码候选 `60837af` 覆盖 0~7；真实日志已确认 Windows Host 用户上下文失败 | 分别推送 SPAKE2 和错误码诊断 PR；根因修复作为 Host/Remoting 需求，不在 WebClient 伪装修复 |
| 已完成 / 上游候选 | QD-002 / QD-403 Web 完整文件管理器 | `Legacy` / `Fork` / `Upstream` 通用能力 | 预编译 Remoting 的 `qd-skill-bridge` 已可承载浏览器工具调用；WebClient 完成目录、新建、重命名、软删除、预览、分块上传下载，host `file-ops` 新增对应工具 | 作为 `UP-Ready` 从 `upstream/dev` 准备最小候选：`file-ops` 工具与 WebClient 页面可拆成两个相邻 PR，不携带 fork 清单和品牌 |
| Remoting 需求 | QD-004 Host 权威录制 | `Legacy` / `Fork` | PR #23 只录控制端已解码画面；Host/Relay 没有权威媒体落盘。控制端断线收尾已由 QD-505 独立完成 | 提 Remoting/宿主生命周期需求；fork 不另造第二套媒体链路 |
| Remoting 需求 | QD-007 图片剪贴板 | `Legacy` / `Fork` | Remoting `ClipboardEvent` 只承载 UTF-8 文本 | 提图片格式、分片和权限边界需求；等协议方案后再实现 |
| Remoting 需求 | QD-114 真实 CertVerifier | `Upstream` issue #20 | SPKI 只是临时放行；根因是 `quickdesk_network_service.cc` 未绑定真实证书校验器 | 向 Remoting 维护者提正式修复需求和回归用例 |
| 验证/排障 | QD-215 Windows Qt platform plugin | `Upstream` issue #7 | 已补发布依赖校验和缺 DLL/Qt 插件负向回归，Windows publish/package 均通过；上游候选 `f449b87` 已准备 | 远端写入恢复后推送候选并开 issue #7 PR；取得原始坏包后做直接复现 |
| 验证/排障 | QD-216 主机连接失败 | `Upstream` issue #9 | 缺真实 host/signaling/TLS 日志，不能直接猜实现 | 先做支持 triage；能拆出确定 bug 后再进入实现 |
| 安全设计 | QD-115 服务端设备密钥信任边界 | `Upstream` issue #10 / `Fork` | 威胁模型已完成：device_secret 不可逆，但明文 access_code + server token/审批使当前服务器仍是设备访问信任方；候选 `codex/upstream-qd-115-trust-model@1ca2543` 已准备 | 按文档分阶段推进 Host 身份密钥、Owner 签名 grant、Host 本地验证和恶意服务器 E2E；8 条验收项完成前保持 `[~]` |
| 上游候选 | QD-911 私密安全披露入口 | `Upstream` issue #6 | fork 与独立候选已有 `SECURITY.md`，候选 `codex/upstream-qd-911-security@9862717` 直接基于 `upstream/dev`，只含 1 个文件；上游 issue 仍开放且 Private Vulnerability Reporting 仍 disabled | 远端写入恢复后推送并开 PR；维护者合并且启用私密报告后才转为 `[x]` |
| 后续增强 | QD-610 业务工作流 | `Legacy` / `Fork` / `Upstream-Open` | QD-510 已完成录制/回放/报告；`bca8149` 补 HTTP trigger；`3f94911` 补持久化 CRUD、固定 interval worker 与运行结果回写；`e518194` 补 Idempotency-Key；`e8ce807` 补异步队列、状态查询和取消；`9f7be0f` 补 HMAC 出站回调；`2e09d5b` 补白名单控制面事件订阅、过滤和幂等 claim；`1176a92`~`a94bba1` 补持久 outbox、死信、重放和 ack；`d3dc4cf` 补 queued job 重启恢复；`3c31d9a` 补 UTC cron；`20d02cc` 补 IANA 时区和 DST；法定节假日、事件停机补发、多进程共享 worker、确认操作者归属和委托权限仍缺 | 继续做不依赖外部凭据的业务编排 |
| 质量收口 | QD-906、QD-907、QD-908、QD-909 | `Legacy` / `Fork` | 真机矩阵、异常路径、2h+ 长稳、文件/剪贴板审计仍不完整 | 每完成一个主功能就补对应边界，不集中到最后一次性补测 |
| 明确暂缓 | QD-318 Linux、QD-319 原生移动端 | `Upstream` Roadmap | 上游已列计划，当前 fork 不做实现承诺 | 保持接口和构建兼容，等上游方向明确后接 |
| 明确不做 | RDP、重写 WebRTC/Remoting、多租户/SSO/计费、通用插件市场 | `Fork` 决策 | 与当前“复用成熟远控内核 + 补业务控制面”的路线冲突或收益不足 | 不进入待办池，除非用户重新定义产品目标 |

### 3.2 合并后的执行队列

这张表把 fork 自己的未完成项与上游 10 个 open issue、4 个 open PR、3 个未完成
Roadmap 项放到同一条实施路径中。详细追溯仍以第 16 节为准。

| 顺序 | 队列 | 包含工作 | 为什么现在放这里 | 完成条件 |
|---|---|---|---|---|
| 1 | Fork 真机收口 | QD-005 / QD-808 macOS 签名与公证、QD-215 Windows Qt 插件、QD-214 Windows Host 用户上下文、QD-607 定时连接边界 | Windows 自动更新已闭环，macOS 自动替换和回退也已通过 ad-hoc 真实 DMG E2E；当前只缺正式签名、公证链和公证包升级回归 | 使用 Developer ID 完成 hardened runtime 签名、notarytool、staple、Gatekeeper 评估和 4.2.8→4.2.9 真实替换；Windows Host 上传和 Qt 插件问题有真实证据 |
| 2 | 可独立回贡上游 | QD-002/QD-403 Web 文件管理、QD-003、QD-114 第一阶段、QD-212、QD-213、QD-214 SPAKE2/诊断、QD-215 Windows 发布校验、QD-316 会话聊天、QD-510 MCP 工作流、QD-911 私密披露策略 | 改动边界清楚，可从 `upstream/dev` 独立审查；Web 文件管理和聊天只依赖现有 `qd-skill-bridge`，QD-510 候选不含 fork 业务和清单，QD-911 候选只含根目录 `SECURITY.md` | 每个主题一个基于 `upstream/dev` 的最小分支；Web 文件管理拆成 `file-ops` 与 WebClient 两个 PR；远端写入恢复后推送 |
| 3 | 等 PR #25 基线 | QD-204 API/客户端、QD-607 服务端、QD-108 / QD-508 / QD-706 / QD-708 管理面、QD-005 客户端 | 这些能力依赖控制面、OpenAPI、录像或管理页基线；直接塞进 #25 会变成巨型 PR | 维护者确认 #25 合并或 stacked base；再按服务端/Web/Qt/更新拆 PR |
| 4 | 跟踪上游已有 PR | PR #22 自动审批、#23 录像、#24 skill-host 运行库、#25 控制面 | 上游已有实现，重复开发只会制造分叉 | PR 合并后同步 `upstream/dev`，重跑受影响测试；删除重复本地实现 |
| 5 | 提 Remoting 需求 | QD-004 Host 权威录制、QD-007 图片剪贴板、QD-114 真实 CertVerifier、QD-214 Windows 用户上下文 | 必须改 Chromium Remoting 或宿主生命周期，维护者已明确这类改动由他处理 | 每项提交协议草案、权限边界、兼容性和验收用例；不在 fork 伪造第二套媒体链路 |
| 6 | fork 自主增强 | QD-610 业务集成、QD-511 智能回放、QD-904 / QD-907 / QD-908 / QD-909 | QD-510、HTTP trigger、持久化 interval/IANA 时区 cron trigger、Idempotency-Key、异步队列/取消、HMAC 回调、持久 outbox、死信查询/重放、幂等 ack 和 queued job 重启恢复已落地；剩余业务编排仍缺法定节假日、事件停机补发、多进程共享 worker 和确认操作者归属 | 有明确用户入口、失败路径和对应自动化或真机证据 |
| 7 | 暂缓跟随上游 | QD-318 Linux、QD-319 iOS/Android | 上游 Roadmap 已列出，本 fork 当前不承诺实现 | 保持接口和构建兼容；上游方向明确后再排期 |
| 8 | 不进入待办 | RDP、重写 WebRTC/Remoting、多租户/SSO/计费、通用插件市场 | 与“复用成熟远控内核 + 补业务控制面”的路线冲突或收益不足 | 只有用户重新定义产品目标才重新评估 |

当前第一执行项是 `QD-005 / QD-808` 的 macOS Developer ID 签名、Apple 公证和公证包
替换回归。自动替换链路已经用 ad-hoc DMG 真机验证；整体仍保持 `[~]`，不能把本地
ad-hoc 成功写成正式跨平台发布完成。

QD-604 和 QD-608 已完成；QD-911 与 QD-115 的上游候选均已准备，前者等上游启用
Private Vulnerability Reporting，后者等协议实现。当前继续做不依赖外部凭据的完整
功能项：QD-610 业务工作流。

回贡上游不等到 fork 全部做完再一次性提交。每个已经独立验收、能从 `upstream/dev`
干净切出的主题都应立即形成最小 PR；依赖 PR #25 的功能只准备 stacked 候选，不继续
扩大现有巨型 PR。

### 3.3 上游待办融合后的回贡批次

上游待办不是另一套产品路线，而是本清单的交付约束。当前上游 `dev` / `master` 都是
`f6651f0`，有 10 个 open issue、4 个 open PR；其中 #22 基于 `master`，#23、#24、#25
基于 `dev`，四个 PR 都是 `mergeable=true`、`mergeable_state=unstable`，不能把
`unstable` 写成 CI 已通过。合并后的回贡顺序固定如下，避免重复实现或把 fork 整体
塞成一个巨型 PR。

| 批次 | 包含工作 | 当前动作 | 上游交付方式 | 明确不做 |
|---|---|---|---|---|
| U0 同步闸门 | PR #22 自动审批线程亲和性、#23 录像、#24 skill-host 运行库、#25 自托管控制面 | 先审查和同步上游已有 PR；本 fork 不重复实现 | `UP-PR#22` ~ `UP-PR#25` | 不把已有 PR 再写一遍，不抢跑 #25 控制面 |
| U1 立即回贡 | QD-002/QD-403 Web 文件管理、QD-003、QD-114 SPKI、QD-212、QD-213、QD-214 SPAKE2/错误诊断、QD-215、QD-316 会话聊天、QD-510、QD-610 通用触发器、QD-911、QD-115 文档 | 远端写入恢复后按主题各推一个最小分支；Web 文件管理拆为 `file-ops` 工具和 WebClient 页面两个相邻 PR；QD-316 候选已在本地验证但 push 被策略拒绝 | `UP-Ready` | 不带 fork 清单、品牌、部署地址或无关业务代码 |
| U2 等 #25 基线 | QD-204、QD-607、QD-006、QD-604、QD-810、QD-005、QD-108/QD-508/QD-706/QD-708 管理面 | 先把 fork 功能做完整，保留 stacked 候选；#25 合并或维护者确认基线后 rebase | `UP-Stacked` | 不直接 cherry-pick 整个 fork 提交；不把服务端、Web、Qt 塞进同一个 PR |
| U3 交 Remoting | QD-004、QD-007、QD-114 真实 CertVerifier、QD-214 Windows 用户上下文、QD-111/QD-408 所需的文件事件 | 写协议草案、权限边界、兼容性和验收用例，先向上游提需求 | `UP-Remoting` | 不重写媒体/输入链路，不在 fork 造第二套 RDP 或伪协议 |
| U4 fork 产品层 | 业务系统专属 provider、委托权限策略、运营页面、品牌和部署配置；QD-610 中依赖具体业务凭据的部分 | 在 fork 主干完成产品化，通用底层抽成 U1/U2 的 PR | `Fork-only` | 不把私有业务规则伪装成上游通用能力 |
| U5 质量底座 | QD-904、QD-906、QD-907、QD-908、QD-909、QD-910 | 每完成一个主功能同步补测试、真机或长稳证据；上游通用测试随后回贡 | `UP-Ready` 或 `Fork-only` | 不集中到最后补测，不用单次成功代替稳定性 |
| U6 跟随上游 | QD-318 Linux、QD-319 原生移动端 | 保持接口和构建兼容，等待上游路线明确 | `Upstream-Open` | 不提前承诺实现，不把 Web 触摸或跨平台构建冒充原生客户端 |

“现在要做”的 fork 工作按以下顺序执行：

1. **QD-610 后续阶段**：`Idempotency-Key`、异步队列/取消、HMAC 回调、控制面事件订阅、
   UTC cron、IANA 时区和 DST 已完成；下一步做法定节假日、事件停机补发和委托权限。通用触发链路能从当前的 MCP
   trigger 候选继续 stacked 回贡，后几项若绑定 fork 业务数据库则留在 `Fork-only`。
2. **QD-511 智能回放**：预录屏幕断言与状态不匹配暂停已完成；剩余 OCR 关键词、AI 介入和人工接管；
   通用部分在 QD-510 候选合并后回贡。
3. **QD-907 异常路径**：按“协议错误、权限拒绝、重连耗尽、上传失败、设备离线”补
   可重复的自动化或真机证据，避免把 Windows 上传根因误判成 WebClient 协议错误。
4. **QD-904 / QD-908**：补 Web E2E、2 小时桌面长稳和几百路观看基线，作为发布门禁。
5. **Remoting 需求单**：QD-004、QD-007、QD-114 根因和 QD-214 用户上下文
   分别写清协议、权限、兼容性和验收标准，交给上游维护者评估。QD-002 已由现有
   `qd-skill-bridge` 解除阻塞，不再列入 Remoting 需求。
6. **#25 后置队列**：QD-204、QD-607、QD-006、QD-604、QD-810 和通用 Web/Qt 管理面
   在 #25 合并/基线确认后按服务端、Web、Qt 拆分回贡。

上游不合并时，fork 不会被阻塞：`fork dev` 继续作为我们的产品主干；每个 U1/U2 候选
保持独立最小分支，按上游变化定期 rebase；上游合并后再把 `upstream/dev` 或
`upstream/master` 合回 fork，删除重复实现并重跑受影响测试。不能让上游审查状态决定
fork 是否能继续开发，也不能因为 fork 自己有实现就再次向 upstream 提交同一主题。

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
| QD-115 | 信令服务器持有设备密钥的信任边界 | `[~]` | `3474203` 已明确当前服务器仍是信任方：device_secret 只有 Argon2id 哈希，但 access_code 明文、signal token/审批/归属由服务器控制。目标方案为 Host 身份密钥 + Owner 签名授权 + Host 本地验证；协议验收项尚未实现 |

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
| QD-214 | Windows 文件上传用户上下文 | `[!]` | 上游 issue #19；SPAKE2 已修通，macOS 上传成功；Windows 普通用户为 `type=2`（`0x522`），SYSTEM/session 0 为 `type=7 NOT_LOGGED_IN`，根因是 Host 服务未在可解析的交互用户上下文运行 |
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
| QD-316 | 文字聊天 | `[x]` | 复用 `qd-skill-bridge` 的内建 `session_chat`；Qt/Web 支持双向短消息、ACK、未读和 4000 字边界，不持久化 |
| QD-317 | 自定义采集/编码插件 | `[ ]` | 暂不优先 |
| QD-318 | Linux 被控端 | `[>]` | 上游 Roadmap 已列出；本 fork 暂缓 |
| QD-319 | iOS/Android 原生控制端 | `[>]` | 上游 Roadmap 已列出；Web 触摸输入不等同原生客户端 |

## 7. 文件、剪贴板与传输

| ID | 功能 | 状态 | 说明 |
|---|---|---|---|
| QD-401 | 文件上传、下载和分片传输 | `[x]` | Remoting 数据通道已具备 |
| QD-402 | Qt 完整文件管理器 | `[x]` | `RemoteFileManager` + `file-ops` skill bridge |
| QD-403 | Web 完整文件管理器 | `[x]` | QD-002 已完成；`SkillBridgeClient` + `RemoteFileManager` + `FileManagerPanel` 经 `qd-skill-bridge` 调用 host `file-ops` |
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
| QD-505 | 断线自动收尾 | `[x]` | 控制端录制在 `disconnected` / `failed` / `closed` / `error` 时自动 `stopRecording()` 并上传；`connected` / `reconnecting` 不中断。Host 侧权威录制的收尾仍属 QD-004 |
| QD-506 | 录像上传、断点续传 | `[x]` | `de88f07`：Web 在线录像按 4 MiB 分块上传，409 按服务端权威 offset 续传，瞬时错误重试、进度展示和失败收口已完成；WebClient 50/50、生产构建和真实 SQLite 4 分片逐字节 E2E 通过 |
| QD-507 | Web 录像列表、播放、下载、删除 | `[x]` | `RecordingsPage.vue` |
| QD-508 | Qt 录像回放页 | `[x]` | `74c7022` 已覆盖列表、分页、缩略图、应用内播放、外部打开、删除和实时刷新 |
| QD-509 | 缩略图、保留期、孤儿回收 | `[x]` | 服务端生命周期已完成 |
| QD-510 | 工作流录制与回放核心 | `[x]` | 远程 UI/skill-host 双通道录制回放、旧格式兼容、递归参数替换、失败策略、持久化报告和查询工具已完成；`cargo test` 19/19 通过 |
| QD-511 | 工作流状态校验与智能回放 | `[~]` | 预录屏幕断言与状态不匹配暂停已完成：`screen_hash` / `screen_contains` 可挂到任一步骤，回放前经 host `assertScreenState` 校验，不匹配即 park 并写入原因，新增 `set_step_expectation` 工具；OCR 关键词和 AI 介入仍未实现 |

## 9. 通知、任务与集成

| ID | 功能 | 状态 | 说明 |
|---|---|---|---|
| QD-601 | 用户通知通道与 Webhook | `[x]` | HMAC、outbox、重试、死信重放 |
| QD-602 | 投递确认与静默时段 | `[x]` | ack、一次性和周期静默 |
| QD-603 | 设备离线持续告警 | `[x]` | 阈值、重复提醒、维护窗口 |
| QD-604 | 指标型告警 | `[x]` | `37dd8aa`：Host 60 秒上报、单设备最新快照、三项独立 episode、持久 outbox、Web/Qt 规则入口和四套 SDK；`sustained_minutes=0` 首样本告警，真实 SQLite+Webhook E2E 通过 |
| QD-605 | WebClient 通知管理页 | `[x]` | `/notifications` |
| QD-606 | Qt 通知规则管理页 | `[x]` | `dbd828f` 已覆盖通道 CRUD/test/启停、静默时段、离线告警和投递运维 |
| QD-607 | 计划任务 | `[~]` | 服务端、WebClient、Qt 管理页和重试已完成；`revoke_all_remote_sessions` 已补齐控制面批量清场（最多 200 台）。定时连接、广义批量 agent 指令及回执仍未做 |
| QD-608 | OpenAPI 与生成 SDK | `[x]` | TypeScript/Go/Python/Java/C#/PHP/Kotlin 七套客户端已提交，生成器、freshness 门禁和 CI 编译覆盖全部语言；C# `dotnet build`、PHP Composer/lint、Kotlin Gradle build 通过 |
| QD-609 | MCP / AI 自动化 | `[x]` | QuickDesk 原生能力；也是上游维护者当前最优先方向 |
| QD-610 | 业务系统集成与自定义工作流 | `[~]` | `bca8149` + `3f94911` + `e518194` + `e8ce807` + `9f7be0f` + `2e09d5b` 已提供同步/异步 HTTP trigger、持久化 trigger CRUD、60 秒~365 天 interval worker、next_run_at at-most-once、运行结果回写、Idempotency-Key 持久化重放、异步 job 状态/取消、HMAC 结果回调、白名单控制面事件订阅、事件过滤/幂等 claim、设备白名单、trust/emergency、单并发和 429；`1176a92`~`a94bba1` 补持久回调 outbox、重试、死信、人工重放和幂等 ack；`d3dc4cf` 补 queued job 重启恢复；`3c31d9a` 补 UTC cron；`20d02cc` 补 IANA 时区和 DST，上游候选推进到 `1be154c`；法定节假日、事件停机补发、多进程共享 worker、确认操作者归属和委托权限仍未完成 |

## 10. Web、Qt 与管理端

| ID | 功能 | 状态 | 说明 |
|---|---|---|---|
| QD-701 | Web 登录与设备接入 | `[x]` | Vue shell + remote.html |
| QD-702 | Web 屏幕墙 | `[x]` | 多设备、轮巡、单砖输入解锁、单瓦片音频独占和晚到音频轨道合并；新增测试 9/9、WebClient 46/46 通过 |
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
| QD-808 | 客户端自动更新消费 | `[~]` | 当前迭代 QD-005；Windows 用 `/SILENT /CLOSEAPPLICATIONS /LAUNCHAPP` 启动安装器，已验证 4.2.8.0→4.2.9.0 全闭环。macOS 更新助手完成 DMG 挂载、bundle/version/签名校验、暂存替换、失败回退、重开和临时包清理，ad-hoc 真实 DMG E2E 通过。剩余：Developer ID 签名、Apple 公证和公证包升级回归 |
| QD-809 | SQLite 轻量服务端 | `[x]` | QD-006 已收口；单文件 SQLite + 默认进程内 Redis，健康检查、启动和重启 E2E 通过；仅支持单进程，Redis 状态重启即失 |
| QD-810 | 正式备份、恢复和容量配额 | `[x]` | `64ab93a`：SQLite 一致性归档与安全恢复、`-force` 旧副本和失败回滚；PostgreSQL `pg_dump` + 媒体目录运维路径；录像/安装包总配额、管理接口和升级通道页状态；真实备份恢复 E2E 通过 |

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
| QD-911 | 上游私密安全披露入口 | `[~]` | fork 提交 `3021ca9` 与上游候选 `9862717` 已新增 `SECURITY.md`，覆盖私密邮箱兜底、公开 issue 禁令、范围、协调披露和 maintainer 检查项；issue #6 仍 OPEN，GitHub Private Vulnerability Reporting 仍 disabled |

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
   远端写入恢复后，先推送已准备好的 QD-002/QD-403、QD-003、QD-114、QD-212、QD-213、
   QD-214、QD-215 最小候选。
2. **把 QD-204 回馈上游**：PR #25 的控制面/ API 基线确认后，先推送 owner-scoped
   服务端、OpenAPI 和 SDK 候选，再推送 Web/Qt 客户端候选；不把 fork 文档带入 PR。
3. **补任务和更新真机链路**：QD-607 的批量远控清场已完成；继续定义定时连接和广义
   批量 agent 指令/回执，补 QD-005 / QD-808 Windows 与 macOS 安装、重启、签名和
   公证验证。QD-215 已补发布依赖校验、正负向回归和上游候选 `f449b87`，剩余原始坏包复现与远端 PR。
4. **准备 Remoting 需求单**：QD-004 Host 权威录制、
   QD-007 图片剪贴板、QD-114 真实 CertVerifier。每项写清协议、宿主生命周期、权限、
   兼容性和验收用例，先交上游维护者评估。
5. **安全与可观测性**：QD-911 候选等上游启用 Private Vulnerability Reporting；
   QD-115 威胁模型已完成，下一阶段是 Host 身份密钥和 Owner 签名授权；之后补 QD-909
   文件/剪贴板审计。
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
- 当前 10 个 open issue：#6、#7、#9、#10、#13、#17、#18、#19、#20、#21；其中
  #13、#18、#19、#20 是用户问题或 Roadmap 入口，#6、#10、#17、#19、#20 已经拆入
  QD-911、QD-115、QD-113、QD-212/213/214、QD-114 的交付队列。
- 当前 4 个 open PR：#22 自动审批线程亲和性（base `master`）、#23 录像与回放、
  #24 skill-host Windows 运行库、#25 自托管控制面/OpenAPI/SDK（后三者 base `dev`）。
- 2026-10-17 23:04 复核：上游两个分支仍为 `f6651f0`，10 个 open issue、4 个 open PR
  无漂移；四个 PR 均为 `mergeable=true`、`mergeable_state=unstable`，不能把
  “可合并”写成 CI 已通过。上游待办与本 fork 的统一拆解以 3.3 为执行入口。
- QD-006 是通用自托管增强，但当前上游没有对应 open issue/PR；候选
  `codex/upstream-qd-006-sqlite@7758456` 已基于 PR #25 基线准备并验证，等 #25 合并后
  rebase 到 `upstream/dev` 才能开不重复 #25 提交的正式 PR。
- 本次融合结论：当前上游没有“必须由本 fork 立即重复开发”的 open issue；真正需要决策的是
  哪些工作作为独立小 PR 先贡献、哪些等 #25 合并后再拆、哪些只能向 Remoting 提需求。
  QD-608 已完成，立即实施顺序改为 QD-610；独立上游候选和 Remoting 需求按 3.3 的批次执行。

### 16.2 上游待办与本 fork 的统一决策矩阵

这张表覆盖当前所有上游 open issue、open PR 和未完成 Roadmap 项。`现在做` 表示不依赖上游
合并即可在 fork 完成；`等待` 表示上游已有实现或 PR，不能重复提交；`边界` 表示需要
Remoting 或协议改造；`不重复` 表示上游已经具备，只需验证或回复 issue。

| 上游项 | 本 fork 对应项 | 当前决定 | 交付方式 | 下一步 |
|---|---|---|---|---|
| #21 维护计划与参与方式 | QD-001、QD-910 | 立即执行 | `Upstream-Open` | 所有新功能从 `upstream/dev` 切最小分支；持续同步清单 |
| #20 自托管 WSS 失败 | QD-114 | 双轨推进 | `UP-Ready` + `UP-Remoting` | Qt/Windows SPKI 先做独立 PR；真实 CertVerifier 交 Remoting |
| #19 设备列表问题 | QD-212、QD-213、QD-214 | 拆分处理 | `UP-Ready` + `UP-Remoting` | 本机标识、访问码回源分别提；上传错误诊断可独立提，Windows 用户上下文根因交 Host/Remoting |
| #18 连接历史 | QD-206 | 不重复实现 | `Upstream-Open` | 核验现有 WebClient/Qt 入口，整理版本证据后回复 issue |
| #17 Auto Approve 失效 | QD-113 | 等待上游 | `UP-PR#22` | #22 合并后同步；不在 fork 重写第二套线程亲和性修复 |
| #13 Linux 支持 | QD-318 | 暂缓 | `Upstream-Open` | 保留跨平台接口，不承诺当前迭代实现 |
| #10 信令服务设备密钥信任 | QD-115 | 设计完成，待协议 | `UP-Ready` + `Upstream-Open` | 安全边界候选 `1ca2543` 可直接提 PR；实现侧需要 Host 身份密钥、Owner 签名 grant 和 Host 本地验证 |
| #9 主机连接失败 | QD-216 | 先排障 | `Upstream-Open` | 收集 host/signaling/TLS 日志，确认可复现 bug 后再提单 |
| #7 Windows Qt 插件启动失败 | QD-215 | 本地回归完成 | `UP-Ready` | 候选 `f449b87` 已包含 x64/VC Runtime/Qt 插件/直接依赖校验和失败关闭；远端写入恢复后推送并开 PR |
| #6 私密安全披露入口 | QD-911 | 候选已准备 | `UP-Ready` | `codex/upstream-qd-911-security@9862717` 只新增 `SECURITY.md`；推送后开 PR，并请维护者启用 Private Vulnerability Reporting |
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
| #19 设备列表相关问题 | Bug cluster | QD-212、QD-213、QD-214 | 本机标识和访问码回源已完成；上传日志确认 Windows Host 无法取得交互用户 token，普通用户为 `0x522/type=2`，SYSTEM/session 0 为 `type=7` | 推送三个独立诊断候选；QD-214 根因按 Host 服务或 Remoting 用户上下文单独提交，不混成一个“设备列表”大修 |
| #17 自动审批被忽略 | Bug | QD-113 | 已由 PR #22 覆盖 | 跟踪 PR #22，不重复实现 |
| #13 Linux 支持 | Roadmap | QD-318 | 上游 Roadmap 已列出，本 fork 暂缓 | 保持跨平台接口，不在当前迭代投入 Linux 被控端 |
| #10 信令服务器设备密钥信任 | Security | QD-115 | device_secret 只有哈希；但明文 access_code + server token/审批意味着被攻陷服务器仍可冒充 Controller | 推送 `signal-trust-model` 候选并在 issue 讨论协议升级；不能把哈希化 device_secret 当作已修复 |
| #9 主机连接失败 | Support / Bug | QD-216 | 需要真实环境日志，不宜直接猜实现 | 收集 host/signaling/TLS 日志后再拆 bug |
| #7 Windows 启动 Qt platform plugin 错误 | Packaging | QD-215 | Qt 插件/运行库打包问题；Windows publish/package 与缺 DLL 负向测试已通过，候选 `f449b87` 仅 3 个文件 | 推送候选并开 issue #7 PR；取得原始坏包后做直接复现 |
| #6 私密安全披露入口 | Process | QD-911 | 候选 `9862717` 已新增根目录 `SECURITY.md`；上游仍无该文件且 Private Vulnerability Reporting 关闭 | 推送 fork 分支并开 PR；请维护者启用 GitHub 私密报告，不在公开 issue 贴利用细节 |

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
| 0a | `feat(file-ops): add browser-safe chunking and versioned trash` | QD-002 / QD-403 | 本轮实现已完成；从 `upstream/dev` 只取 `file-ops` crate、SKILL.md 和 `Cargo.lock`，先独立提交 |
| 0b | `feat(webclient): add remote file manager over skill bridge` | QD-002 / QD-403 | stacked 在 0a 之后；只取 `SkillBridgeClient`、文件管理器模块、remote-main 接线、样式、双语和 Node 测试 |
| 1 | `feat(webclient): coordinate cross-tab control and audio` | QD-003 | 当前功能已测试；补上游 WebClient 约定和截图 |
| 2 | `fix(self-host): allow pinned SPKI for self-hosted WSS` | QD-114 / issue #20 | 候选已准备；需推送、创建 PR，并补 Qt 构建和 Windows service 实机验证 |
| 3 | `feat(qt): mark the local device in device lists` | QD-212 / issue #19 | 候选已准备并通过上游基线构建；等远端写入后直接创建 PR |
| 4 | `fix(qt): refresh missing access codes before connecting` | QD-213 / issue #19 | 候选已准备并通过上游基线构建；等远端写入后直接创建 PR |
| 5 | `fix(webclient): match BoringSSL SPAKE2 password scalar` | QD-214 / issue #19 | 候选 `ed5437b`；Chromium 140 固定向量、真实 macOS/Windows Host 均通过 |
| 6 | `fix(webclient): explain host file transfer failures` | QD-214 / issue #19 | 0~7 错误码候选 `60837af`；Windows 根因需要 Host/Remoting 用户上下文修复 |
| 7 | `fix(win): reject incomplete Qt runtime releases` | QD-215 / issue #7 | 候选 `f449b87` 只含 verifier 和 publish/package 失败关闭；校验器正负向测试通过 |
| 8 | `feat(scheduled): add batch remote-session revoke` | QD-607 | 纯服务端候选 `badf990` 已基于 PR #25 独立验证；远端写入恢复后作为 stacked PR 提交 |
| 9 | `feat(webclient): control-plane management pages` | QD-108/605/607/703 | 依赖 PR #25 合并；按页面拆成多个 PR |
| 10 | `feat(qt): remote session and management pages` | QD-706/708 | 依赖 PR #25 合并；审批页 `7c23f13`、会话实时消费 `ea5a65f` 和计划任务页 `42b6c61` 可先拆，再补通知页 |
| 11 | `feat(qt): consume self-hosted update channel` | QD-005 / QD-808 | 依赖 PR #25 合并；Windows 重启修复已包含在 fork `a982a10`。直接 cherry-pick `cae77ff` 到 `upstream/dev` 已实测冲突，必须在更新通道基线上重做最小增量，不携带 fork 私有文档 |
| 12 | `feat(signaling): support SQLite minimal mode` | QD-006 / QD-809 | 候选 `codex/upstream-qd-006-sqlite@7758456` 已基于 PR #25 基线通过完整验证；等 #25 合并后 rebase 到 `upstream/dev`，不带 fork 清单 |
| 13 | `feat(ops): backup restore and storage quotas` | QD-810 | fork 提交 `64ab93a` 已完成；依赖 PR #25 与 QD-006 SQLite 候选。等 #25 合并且 SQLite 候选先落地后，再拆备份 CLI、配额与存储视图最小增量 |
| 14 | `feat(signaling): report host metrics and alert on sustained thresholds` | QD-604 | 服务端候选 `codex/upstream-qd-604-metrics@0ea9e7c` 已基于 `codex/upstream-qd-006-sqlite@7758456` 独立验证；含 SignalingServer、OpenAPI、四套 SDK、smoke 和真实 SQLite+Webhook E2E。fork 功能提交为 `37dd8aa` | 远端写入恢复后推送并作为 QD-006 的 stacked PR；Web/Qt 等 PR #25 客户端基线确认后单独拆 |
| 15 | `docs: add private security disclosure policy` | QD-911 / issue #6 | 候选 `codex/upstream-qd-911-security@9862717` 直接基于 `upstream/dev@f6651f0`，只新增根目录 `SECURITY.md`；上游 Private Vulnerability Reporting 仍 disabled | 推送后创建 issue #6 PR；请维护者同时启用 GitHub Private Vulnerability Reporting |
| 16 | `docs(security): define signaling trust boundary` | QD-115 / issue #10 | 候选 `codex/upstream-qd-115-trust-model@1ca2543` 直接基于 `upstream/dev@f6651f0`，只新增 `docs/security/signaling-trust-model.md` | 推送并关联 issue #10；文档批准后再按 Host 身份密钥与 Owner 签名授权拆协议 PR |

QD-002 / QD-403 的上游候选已经推送并开 PR：

- `codex/upstream-qd-002-file-ops` = `e106be4`：基于 `upstream/dev@f6651f0`，只含 host
  `file-ops` 的分块读写、默认目录、版本保留软删除、SKILL.md 和 Rust 测试；
  上游 PR <https://github.com/barry-ran/QuickDesk/pull/26>，OPEN / MERGEABLE / CLEAN。
- `codex/upstream-qd-403-webclient` = `9ff00af`：stacked 在 file-ops 分支上，只含浏览器
  `qd-skill-bridge` 客户端、Web 文件管理器 UI、remote-main 接线、样式、双语和 Node 测试；
  上游 PR <https://github.com/barry-ran/QuickDesk/pull/27>，base 为 PR #26 的 head。
- 验证基线：候选分支上 file-ops 4/4、Clippy `-D warnings`、WebClient 12/12、生产构建、
  接线检查（完整能力集打开菜单入口）和 diff check 全部通过；fork 主线另跑 62/62 与
  桌面/移动 Chrome 烟测。
- 上游 `#22`~`#25` 的既有候选分支已全部推送到 fork，等维护者审查。

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

QD-214 的 SPAKE2 上游候选已经单独准备：

- 分支：`codex/upstream-qd-214-spake2-webclient`
- 基线：`upstream/dev` = `f6651f0`
- 本地提交：`ed5437b`
- 范围：BoringSSL password-scalar 兼容修正、Chromium 140 固定向量和 Node 测试入口
- 验证：固定向量 1/1、`npm run build`、真实 macOS/Windows Host SPAKE2 均通过
- 未完成：远端推送仍被策略拒绝，尚未创建 upstream PR

QD-214 的文件传输错误码上游候选已经单独准备：

- 分支：`codex/upstream-qd-214-file-transfer-errors`
- 基线：`upstream/dev` = `f6651f0`
- 本地提交：`94d3ec3` + `60837af`
- 范围：宿主错误码 0~7 解释、上传取消协议值、Node 测试入口
- 验证：错误码测试 6/6、`npm run build`、`git diff --check` 通过
- 根因证据：Windows 普通用户 Host 报 `Failed to get current user token: 0x522`，
  SYSTEM/session 0 报 `type=7 NOT_LOGGED_IN`；macOS Host 同文件上传成功且字节一致
- 未完成：诊断 PR 待远端写入恢复；根因修复需要 QuickDesk Windows 服务在活动交互
  session 中启动 Host worker，或由 Remoting 解析活动用户 token

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
| Host 权威录制 | QD-004 | 当前 PR #23 只做控制端录制；Host 侧录制需要 Remoting/宿主生命周期。断线收尾的控制端部分已完成（QD-505） |
| 双向图片剪贴板 | QD-007 | `ClipboardEvent` 当前只承载 UTF-8 文本，需要新格式/分片协议 |
| 真实证书校验 | QD-114 | `quickdesk_network_service.cc` 的 CertVerifier 管道在 Remoting 内 |
| Windows 文件传输用户上下文 | QD-214 | `EnsureUserContext()` 依赖 `WTSQueryUserToken(WTS_CURRENT_SESSION)`；服务/session 0 或普通用户 Host 都无法满足 |

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
- 把 QD-214 的 SPAKE2 兼容修复和 0~7 错误诊断分别作为独立增量；Windows 上传根因作为 Host 服务/
  Remoting 用户上下文需求提交，不在 WebClient 内绕过；
- 把 QD-005 的 Qt 更新消费作为 PR #25 的后续增量，先保证跨平台安装行为和签名链；
- 把 QD-006 的 SQLite 单进程模式作为 PR #25 的独立 stacked 候选，明确“单实例、进程内
  Redis 易失”的边界；候选已在 `7758456` 验证，等 #25 合并后 rebase 再提交；
- 把 QD-604 指标告警作为 PR #25 后的 stacked 增量：服务端保持“单设备最新快照 + 独立
  episode + 持久 outbox”，客户端只消费稳定 API，不在上游重建第二套监控系统；
- 把 QD-911 的 `SECURITY.md` 作为独立文档 PR，明确 GitHub 私密报告优先、公开 issue
  禁止贴利用细节，同时请维护者启用 Private Vulnerability Reporting；
- 把 QD-115 的信任边界文档单独提 PR，先把“服务器不能恢复 device_secret”与“服务器
  仍可冒充 Controller”区分清楚，再推进签名授权协议；
- 把 Host 自动录制、图片剪贴板整理成 Remoting 需求，由维护者评估；Web 文件管理器已经
  通过现有 `qd-skill-bridge` 完成，并作为两个最小上游增量准备；
- 把 PR #25 的控制面能力先等上游审查，避免重复提交；
- fork 的 Web/Qt 业务页面在 #25 合并后按页面拆成增量 PR，而不是塞进一个巨型控制面 PR。
