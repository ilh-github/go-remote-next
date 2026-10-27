# QuickDesk 功能总清单

> 最后对账：2026-10-21 01:00 CST。上游与 fork 状态都是动态的，实施前必须重新执行
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
- 本地代码基线：`fce1bf5`（QD-004 artifact 摘要校验：上传字节的 SHA-256 与声明值不符即拒绝且不落库；`2c3ca9b` 的 QD-005/QD-808 打包门禁：Release 默认强制公证，Debug 保持宽松；`2062b95` 的 QD-004 服务端半边补齐：一次性上传票据可实际用于上传，且绑 session 防跨会话写入；`f365e1b` 的一次性上传票据；`1ad0b85` 的 QD-511 人工接管：park 的 run 留下 resume point，operator 可查询、接管和清除；`97b8619` 的 QD-610 Owner 委托权限：Owner 可签发带 workflow/设备范围、步骤上限和禁权工具的委托 grant；`2848be5` 的公平调度：异步工作流按提交方轮转；`a1d34c5` 的账号会话撤销事件全部迁入事务 outbox；本轮之前的
  `314c73b` 先让改密码路径的 `session.revoked` 可恢复；更早的 `cddc0db` 为管理端设备、
  来源封禁与 TURN 配置事务 outbox 及在线状态离线事件迁移；QD-907 重连 Playwright E2E；
  QD-904 设备页、屏幕墙、文件管理器、真实多标签和真实自托管 SignalingServer E2E；
  QD-908 多观看端 soak 门禁）；`origin/dev` 与 `origin/master` 已同步到 `fce1bf5`，
  本地领先数量以 `git rev-list --count origin/dev..dev` 的实时值为准
- 上游基线：`barry-ran/QuickDesk` 的 `dev` = `5db316c`，`master` = `f6651f0`
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
- QD-904 Web 文件管理器浏览器 E2E 已补：真实 Chrome 加载 `RemoteFileManager` 和
  `FileManagerPanel`，用假 Host 覆盖目录导航、新建、重命名、截断文本预览、下载、
  4 字节分块上传和软删除；同一测试已追加到 QD-403 WebClient 上游候选
- QD-904 多标签浏览器 E2E 已补：真实 Chrome 打开三个同源标签，验证新标签抢占 control
  会撤销旧标签、抢占 audio 只撤销音频、释放后可重新 claim、不同 device 消息隔离；
  同一测试已追加到 QD-003 上游候选
- QD-904 屏幕墙浏览器 E2E 已补：Vite 加载真实 `WallPage.vue`，模拟 owner-scoped
  设备接口和同源被控端 iframe，覆盖在线/离线瓦片、音频独占、输入独占、聚焦、
  单瓦片重连、列设置、移除和刷新
- QD-904 设备页浏览器 E2E 已补：Vite 加载真实 `DevicesPage.vue`，覆盖分组筛选、
  创建分组、成员保存、备注、无访问码提示、连接 token/控制码 handoff、收藏移除、
  连接记录、解绑和刷新
- QD-904 真实自托管 SignalingServer E2E 已补：临时 Redis + SQLite 服务、管理员播种、
  账号和设备、真实登录、Realtime snapshot、私有分组写库和 access-code 校验全部走真实
  HTTP/WebSocket；生产 WebClient 构建由 Chrome 驱动
- QD-908 多观看端 soak 门禁已补：真实 `WallPage.vue` 加载可配置数量的动画会话 iframe，
  采样瓦片存活、iframe 帧推进、主线程 rAF、JS heap、音频独占和页面错误；200 路
  30 秒烟测通过，正式 2 小时门禁通过环境变量启用
- QD-908 本轮补齐「周期性分配源」，这是上一轮 2 小时长跑留下的唯一未答问题。先加测量：
  `js/soak/allocationProfile.js` 用 CDP `HeapProfiler.startSampling` 全程采样分配（结束时的
  堆快照只能说活着多少，说不出谁分配的，已被回收的周期性分配源在快照里不留痕迹），
  汇总出自分配最大的调用路径及其占比；peak tracker 原本就在按间隔读 heap，序列直接复用
  （上限 600 点，2 小时跑每秒一点是 7200 个点，不可读也未必更快），周期用去均值后的自相关，
  且必须超过阈值才报告——heap 曾被测得在均值 111% 幅度上振荡而一切正常，弱相关报成发现
  比不报更糟，`periodMs` 为 0 是「未建立周期」这个答案而非缺字段。40 路 90 秒首跑即定位：
  自分配第一名是 vue-i18n 的 `compileMessageFormat`，占 29.6%，heap 周期 2008ms、
  strength 0.762——与被控端 2000ms `getStats` 间隔吻合而非帧循环，与「分配源不依赖主线程
  帧循环」这一既有结论对上了。根因是三处无故全墙重渲：① iframe 每 500ms 上报帧计数，
  `onWallMessage` 用整个替换 `live[deviceId]` 对象来回应，替换即重渲，而瓦片模板带 8 个
  `$t()` 绑定各编译一次 i18n 消息；② `resolveTiles` 每次重算都返回全新 row 对象，Vue 于是
  认为每个瓦片都变了；③ `stateLabel` 在模板表达式里调 `t()`，每瓦片徽章每次渲染一次。
  修法分别是 `mergeLiveState`（合并且告知是否有可见变化，计数原地推进）、row 复用
  （`sameRow` 排除计数，先解析再比较）、`stateLabel` 按状态 memo 化并随 locale 重建。
  40 路 90 秒复测：总采样分配 -24.6%，i18n 头部路径 -21%，`compileMessageFormat` 已换成
  便宜得多的 `resolveMessageFormat`，说明编译器不再走 tick 路径。2008ms 周期仍在，
  这是对的——那是 `getStats` 在做真事，剩下的是瓦片状态变化时模板理应偿付的消息解析。
  验证：新增 16 个单测，`npm test` 207/207，wall E2E 绿，生产构建绿，smoke soak 通过且
  profile 非空、无页面错误。一处已记录的反面：row 复用的第一版拿半成品 row 与上一个比较，
  于是每个瓦片都像变过，测试直接红，改成先解析再比两个成品 row 才对
- QD-610 已补控制面事件订阅：Qt `CloudDeviceManager` 只向本机 MCP 广播白名单事件，
  MCP 支持 `event_type` / `event_device_id` / `event_filter`、设备 ID 归一化、
  claim-before-run 和事件 ID 幂等；SQLite 模式重启后不依赖会重置的 `server_rev`
- QD-610 已补 cron：六段/七段标准表达式和 `@hourly` 等简写由 `cron` crate 校验，
  `chrono-tz` 提供 IANA 时区和 DST；`interval_seconds` 与 `cron_expression` 互斥，
  时间规则可与事件规则共存，错过槽位不补跑
- QD-610 已补批量回调确认：最多 200 个去重 id，返回 acknowledged/skipped/not_found；
  非终态记录不会拖垮整批。`QUICKDESK_OPERATOR_TOKENS` 可配置命名 Bearer token，
  首次 ack 持久化 `acknowledged_by` / `acknowledged_via`，后续确认不能覆盖操作者
- QD-610 已补同目录多进程协调：回调 delivery 和 queued job 都通过原子 claim 选举单一
  worker；claim 在正常结束后释放，异常遗留 30 秒后可回收；有效 job claim 会阻止另一个
  实例把 running 误判成崩溃遗留。该能力不等同于跨机器数据库队列或公平调度
- QD-610 服务端 Redis 每用户事件 replay 窗口已可配置：`EVENT_REPLAY_MAX_LEN` 默认
  1000，`EVENT_REPLAY_TTL_SECONDS` 默认 3600，正常 publish、retry 补写和 replay 读取
  共用同一上限；这是有界窗口，不冒充 durable event outbox
- QD-610 服务端事件 retry 已补跨 worker claim：`LMOVE` 原子领取到 per-worker
  processing list，heartbeat 过期后其他实例恢复死 worker 遗留事件；正常 drain 也会先恢复
  自身 processing。语义为 at-least-once，完成 XADD 后崩溃可能重复但不会静默丢失
- QD-610 数据库事务 event outbox 已落地：业务事务可通过 `EventOutbox.EnqueueTx`
  原子写入事件；worker 使用条件 claim、2 分钟过期恢复和退避重试发布，发布成功后标记
  delivered。`scheduled_task.failed` 已迁移，outbox 写入失败会回滚任务终态
- QD-610 设备访问事件已迁入事务 outbox：`device.access.granted/denied` 与
  `device_access_logs` 同事务提交；outbox 插入失败会回滚访问日志，不再出现“审计有记录、
  实时事件丢失”
- QD-610 访问策略变更已迁入事务 outbox：策略 PUT、封禁和解封与
  `device.access_policy.changed` 同事务提交；outbox 失败会回滚策略写入
- QD-610 远端会话生命周期已迁入事务 outbox：`remote_session.started/ended` 与
  `remote_sessions` 同事务提交；进程重启 orphan sweep 现在会为每个被关闭会话产生结束事件
- QD-610 录像生命周期已迁入事务 outbox：recording create/update/delete、
  artifact 和 thumbnail 更新与 `session_recordings` 同事务；删除文件仍在数据库提交后执行
- QD-610 访问审批生命周期已迁入事务 outbox：`device.access_approval.requested/decided`
  与 `device_access_approvals` 同事务提交；outbox 插入失败会回滚审批创建或决定，
  handler 不再二次 publish
- QD-610 共享访问码生命周期已迁入事务 outbox：`device.access_grant.changed/used`
  与 `device_access_grants` 同事务提交；创建、核销、撤销失败都会回滚业务行
- QD-610 离线告警已迁入事务 outbox：`device.offline.sustained` 与
  `device_offline_alert_rules.alerted_at` claim 同事务；outbox 失败会回滚 claim
- QD-610 指标阈值告警已迁入事务 outbox：`device.metric.alert` 与
  `device_metric_alert_rules` episode claim 同事务；高频 `device.metrics.updated`
  仍保持 best-effort
- QD-610 设备控制面变更已迁入事务 outbox：bind/takeover、unbind、clear session、
  name/remark patch 和 access-code 同步与设备/绑定行同事务；takeover 的旧主人
  ownership-lost 事件也一起提交
- QD-610 收藏设备生命周期已迁入事务 outbox：favorite added/updated/removed 与
  `user_favorites` 同事务；收藏列表不再出现数据库已变、实时 UI 未收到通知
- QD-610 管理端设备变更已迁入事务 outbox：admin force-unbind、单条/批量硬删除和
  device secret rotation 与对应事件同事务提交；outbox 失败会回滚设备状态或密钥哈希，
  不再出现“操作已成功、审计/Webhook 事件却永久丢失”的窗口
- QD-610 管理员跨账号来源封禁/解封已迁入事务 outbox：管理员 ban/unban 复用策略服务
  的 `BanIPWithEvent` / `UnbanIPWithEvent`，策略行与 `device.access_policy.changed`
  同事务；SQLite handler E2E 已断言两次操作各产生一条 outbox 事件。该能力依赖
  QD-109 策略基线，因此留在 fork，不塞入缺少该功能的 PR #25 outbox 候选
- QD-610 TURN/STUN 配置变更已迁入事务 outbox：`turn.config.changed` 与 settings 行
  同事务提交，缓存仅在提交成功后刷新；outbox 失败时配置版本和数据库都保持旧值
- QD-610 已补 MCP 事件重连和持久化：QuickDesk WebSocket API 默认保留最近 2048 条白名单
  事件，可用 `QUICKDESK_CONTROL_PLANE_REPLAY_CAPACITY` 配置为 1~10000；按用户 ID
  原子持久化，线程重启/QuickDesk 重启后重新载入；MCP auth 携带
  `since_server_rev`，认证后补发更大的 revision
- QD-111 / QD-408 文件传输审计已补齐：控制端在 Remoting DataChannel 完成或失败后上报
  `POST /v1/me/file-transfer-audits`，服务端校验设备归属并按
  `(actor, direction, transfer_id)` 幂等写入 `file.transfer.upload|download` 审计；
  WebClient 和 Qt 均已接线，OpenAPI 与七套 SDK 同步。该记录是控制端上报，不是 Host
  对实际文件系统操作的加密证明
- QD-111 / QD-408 上游候选已准备：`codex/upstream-qd-111-file-transfer-audit` =
  `c57a2f4`，基于 PR #25 的 `feat/signaling-control-plane@8998466`。候选只带
  SignalingServer、OpenAPI、WebClient、Qt 接线、contract smoke 和 PR #25 基线已有的
  TypeScript/Go/Python/Java 四套 SDK，不带 fork 清单、品牌或七套 SDK 扩展
- QD-907 信令认证边界已补：`WebSocketTransport` 的 auth_ok 超时可配置，默认仍为
  6000ms；修复超时错误先被同步 `onclose` 覆盖成通用断线错误的问题；XML、JSON
  payload 和业务 `send()` 在 auth_ok 前全部拒绝，认证完成后恢复
- QD-907 访问码校验失败已分级：`ACCESS_POLICY_DENIED` / `FORBIDDEN` 不再误报为参数
  缺失，离线、错误访问码、限流、审批等待和控制器签名失败分别显示稳定中文/英文原因
- QD-907 录制上传重试耗尽已有证据：连续传输错误和最终 503 都保留最后错误，所有尝试
  使用同一个 `Upload-Offset`，不会把失败上传误报成完成
- QD-907 WebSocket 重连已有有界策略：信令断开后重新签发一次性 signal_token，按
  `0/1/3/7s` 四次尝试重连；不可重试的访问失败立即停止，尝试耗尽后显示稳定失败状态
- QD-907 浏览器 E2E 已补：真实 Chrome 加载 `remote.html`，用本地静态服务器、假信令
  WebSocket 和可控 `RTCPeerConnection` 覆盖第 1~3 次恢复、四次耗尽、策略拒绝、上传
  并发和旧 Session 迟到事件；上游最小候选同步包含该测试，但录像上传用例在候选基线
  缺少该 fork-only 模块时显式跳过
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
| QD-002 | P0 | Web 控制端完整文件管理器 | `[x]` | UP-Ready | 经 `qd-skill-bridge` 完成目录浏览、新建、重命名、版本保留软删除、文本预览、分块上传下载；62 项 WebClient 测试、生产构建和桌面/移动浏览器烟测通过。本轮复核 host 侧 `file-ops` 时**抓到一个真缺陷**（`c8b9cb3`）：`trash_path` 的工具描述写着「file or directory」，而测试只覆盖了「文件删掉再删一次」——那是另一种情况，因为第二个文件的父目录是真实目录。**已经在回收站里的路径**会被静默嵌套：它的父目录本身就是 `.quickdesk-trash`，于是目的地变成 `.quickdesk-trash/.quickdesk-trash/...`，每重复一次就往里埋一层，而任何清理回收站的逻辑都不会往那看。现在按**父目录**判定并拒绝。同时补上描述隐含的两个用例：目录连同内容一起移动、同一毫秒内两次删除两份都在（这才真正压到碰撞循环，而不是指望时钟足够慢）。**skill-host 测试也首次进入 CI**：它和 MCP bridge 一样只被 build 不被 test，而这里是全场最不该省的——`file-ops` 就是在远端机器上删东西的那个技能。证据：Windows run `40312641669`（head `c8b9cb3`）的 `Test Skill Host` 报 `7 passed; 0 failed`，`Test MCP Bridge` 报 `129 passed; 0 failed`；macOS run `40312641640` 的 arm64 job 同样两步全过（129 / 7）。两个平台仍只有 `Package DMG` 失败，原因是缺 `MACOS_SIGN_IDENTITY`，与代码无关。。`12b91bd` 再修两个路径助手缺陷：`parentPath('/home//user///docs')` 返回 `/home//user//`，而 `breadcrumbs` 对同一输入返回折叠后的 `/home/user/docs`——两个助手对同一条路径各说一套，而带双分隔符算出来的 parent 并不是服务端收到的那串，重命名就会落到别处；`breadcrumbs` 本来就靠 split+filter 折叠，`parentPath` 是按最后一个分隔符切片、把连续分隔符留下了。另一处：`parentPath('C:docs')` 返回 `''`，驱动器相对路径压根没有分隔符，索引查找落空，而 `''` 被调用方读作「没有父目录」、于是往文件系统根写；`basename` 有镜像 bug，返回整串而不是 `docs`。修法是两个助手都折叠连续分隔符，驱动器相对情形在索引查找之前处理（而不是靠只覆盖其中一种形状的 index===2 判断）。另 `425a067` 修好三个自 QD-111 起就一直红着的审计 E2E：direction 已从 action 名里移出，行带的是 `clipboard.sync` 而 direction 在 details 里，那三个浏览器测试还在断言旧名 |
| QD-003 | P0 | 多标签控制与音频抢占 | `[x]` | UP-Ready | 同一设备多个同源标签页只有一个控制者/音频播放者；新标签接管后旧标签立即释放；真实 Chrome 三标签 E2E 覆盖抢占、释放、音频独立撤销和设备隔离 |
| QD-004 | P0 | Host/Relay 权威自动录制 | `[~]` | UP-Remoting | 被控端或中继自动录制，断线自动收尾并上传索引；断线收尾的控制端半边已由 QD-505 补齐；`f365e1b` + `2062b95` 补服务端半边：一次性上传票据可经 `PUT /v1/recordings/:id/artifact?t=` 实际使用（Redis 标记 nonce，满足验收第 5 条「票据过期、会话错配、recording ID 错配全部被拒绝」），且票据绑 recording 的 session，跨会话写入返回 403 而非落到错误行；`a390735` 再补 artifact 摘要校验：上传字节的 SHA-256 与 `X-Artifact-Sha256` 声明值不符即拒绝并删除临时文件，行不被标记；`f19db63` 收口需求第 4 节「不能留下永久 `recording`」的另一半——僵死行进 `failed` 时同时写入 `ended_at`（此前只改状态，详情页没有结束时间、保留清理只能按 `updated_at` 计龄，而 sweep 每次都刷新它），并删掉一份没有调用方的重复 sweep 方法；Host 侧采集与编码仍待 `docs/dev/Host权威录制Remoting需求.md` 的 Remoting 协议 |
| QD-005 | P0 | 客户端自动更新消费 | `[~]` | UP-PR#25 + Upstream-Open | Windows 已通过检查、下载、SHA-256 校验、静默安装、自动重启、版本落盘和服务存活真机闭环；macOS 已实现受校验的 DMG 挂载、暂存替换、失败回退和重开，并通过 ad-hoc 真实 DMG E2E；`2c3ca9b` 补打包门禁：Release 现在默认强制公证（此前 `package_qd_mac.sh` 从不传该开关，「已签名未公证」的包会以退出码 0 通过），Debug 保持宽松，工具链命令可注入以便测试；`e702a19` 补消费端最后一道：helper 此前只跑 `codesign --verify`（只证明字节完好、由签名者签押），ad-hoc 签名没有 TeamIdentifier 因而绕过匹配检查，结果是更新器为 Gatekeeper 必拦的包报告成功；现在按「ad-hoc 签名永不可能带公证票据」与「`spctl --assess` 只在 Gatekeeper 启用时采信」两条独立信号判定，默认严格，`QUICKDESK_ALLOW_UNNOTARIZED_UPDATE` 显式放宽。E2E 修掉一个掩盖这一切的自身 bug：`make_dmg` 未加 `local`（/bin/sh 没有）赋值给全局 `source_app`，最后一次传入的 bundle 覆盖了所有后续场景。两个 macOS E2E 已接入 CI，Qt 资源测试断言内嵌 helper 仍带该检查。剩余：Apple Developer ID 签名与 Apple 公证凭据，以及用真实公证包做一次升级回归 |
| QD-006 | P1 | SQLite 轻量服务端部署 | `[x]` | UP-PR#25 | 单进程、单文件 SQLite、默认进程内 Redis；`/health` 区分 sqlite，空目录启动和重启持久化 E2E 通过 |
| QD-007 | P1 | 图片剪贴板 | `[ ]` | UP-Remoting | 需求单已成文 `docs/dev/图片剪贴板Remoting需求.md`：双向图片格式协商、独立 DataChannel 分片传输（沿用文件传输的 8 KB 分片与 `bufferedAmount` 背压）、权限与 10 条边界验收；实现待 Remoting |

最近验证：

- `f3719a39`（部署手册的默认值和代码不是一回事，`.env.example` 连数据库名都建不出来）：给两份部署
  手册补「完整环境变量参考」时逐条对 `viper.SetDefault` 核，抓到**一处会让首次部署直接起不来**
  和**四处会把操作员引向错误安全假设**的文档缺陷。真缺陷：`.env.example` 写着
  `DB_NAME=quickdesk_signaling`，而代码默认 `quickdesk`，`start.sh` 也只
  `CREATE DATABASE ${DB_NAME:-quickdesk}`——照模板复制的部署会去连一个没人建的库，报出来的
  是连接错误，指向的是服务而不是那份被要求复制的文件。四处文档假象：①
  `METRICS_TOKEN` 留空被写成「不校验，任何人都能读到指标」，实际是 `cfg.Metrics.Token != ""`
  才注册路由，**留空直接没有 `/metrics` 这个端点**——按文档配的部署会以为指标在裸奔，其实
  根本没有指标；②`TRUSTED_PROXIES` 留空被写成「除非在已知代理之后否则留空」，读作「默认不信任」，
  实际默认是 `DefaultTrustedProxies`（回环 + 私网段），默认**信任**内网来源的
  `X-Forwarded-For`；③`RECORDING_URL_SECRET` 被写成「生产必填，不填则媒体链接长期有效」，
  实际留空回退 `API_KEY`，两者都空时票据是**拒绝签发**而不是签发一个无签名长链；④
  `REVOKE_COOLDOWN_SECONDS` 没说 0 的含义，而 0 是「允许立即重连」（历史行为）。
  修法之外补了三层门禁（`SignalingServer/internal/config/docs_defaults_test.go`）：81 个
  `viper.SetDefault` 键两份手册必须全覆盖；文档默认值列必须和代码逐一相等（`(空)`/`(空)` 归一化）；
  `.env.example` 的 SERVER/DB/REDIS 坐标必须和 `start.sh` 的创建默认一致；以及四条行为断言。
  每层都做了双向验证：把 `LOG_MAX_BACKUPS` 改成 20、把 METRICS_TOKEN 那句改回去、把
  DB_NAME 改回旧值，各自对应门禁立刻红。上一轮的教训在这里又应验一次——文档缺陷和代码缺陷
  一样，都要走到「用户看到什么」才算数：这份手册的描述如果照做，得到的是一个连不上库的
  部署和一个以为在裸奔其实没有指标的运维。
- `0858c431`（重复键让批量重试按钮看起来像一句错误）：给 NotificationsPage 补浏览器
  E2E 的过程中抓到的真缺陷，两个 locale 都有。`notifications.deliveries.retryFailed`
  在同一个对象里声明了两次：第一处是**按钮标签**（「重试本通道失败投递」），第二处是
  **错误文案**（「重试投递失败」）。JS 对象字面量后者胜出，于是那个按钮——页面上唯一能把
  积压重新入队的控件——永久渲染成一句失败提示，哪怕什么都没失败。操作员看到的是一个写着
  「重试投递失败」的按钮，既不知道它是可以点的，也不敢点。修法是把错误那份改名
  `retryFailedError`，六处 `toastError` 调用点改指新键，按钮标签保持原键。同类隐患已全量
  扫过：两个 locale 各 690/691 个键，重复的各 1 个，就是这两处，现已归零。
- `7ad833f7`（AccessApprovalsPage 补上浏览器 E2E）：上一轮对账发现它是六个没有页面级
  E2E 的页面之一（912 行，只有模型层单测），而它正是「点错一下就有后果」的那一页——
  approve 放一个会话进屋，deny 把来源打进冷却期。断言选的是单测看不见的那几件事：
  ①**对话框必须对应被点的那一行**——两行形状完全相同，用页面级选择器会在决定了错误审批的
  情况下照样绿，所以点击限定在 `.approval-row` 的第 n 行内；②approve/deny 的按钮语义；
  ③**409 时必须报服务端回报的状态**而不是笼统失败。反向验证：把 `submitDecision` 里
  approve/deny 两个调用互换，测试立刻红。
- `7ad833f7`（AccessApprovalsPage 补上浏览器 E2E）：上一轮对账发现它是六个没有页面级
  E2E 的页面之一（912 行，只有模型层单测），而它正是「点错一下就有后果」的那一页——
  approve 放一个会话进屋，deny 把来源打进冷却期。按既有 fixture 形状（Vite + 真实
  `.vue` + mock owner-scoped 接口）补 `test/fixtures/access-approvals.html` 与
  `test/e2e/access-approvals.e2e.mjs`，E2E 门禁从 18 个文件增至 19 个。断言选的是单测
  看不见的那几件事：①**对话框必须对应被点的那一行**——两行形状完全相同，用页面级选择器
  会在决定了错误审批的情况下照样绿，所以点击限定在 `.approval-row` 的第 n 行内；
  ②approve/deny 的按钮语义（confirm 取 primary、deny 的危险样式与冷却期警告）；
  ③**409 时必须报服务端回报的状态**而不是笼统失败——「decision failed」会把操作员派去找
  一个根本不存在的网络问题。反向验证：把 `submitDecision` 里 approve/deny 两个调用互换，
  测试立刻红。
  过程中三次被自己的错误假设带偏，都已查伪并记录：曾据 6 空格缩进的正则漏看嵌套
  `status:` 块而误判「approvals.status.* 两个 locale 都缺键」——实际五个状态标签都在；
  曾据调试脚本自造的 mock 报出 `approvals.status.` 空键警告而怀疑页面缺陷——实际服务端
  `status` 列 `not null default pending`，空值不可达，警告来自我自己的假数据；
  曾据 `.approval-item` 选不中而怀疑页面没渲染——实际类名是 `approval-row`。
  三次共同点：都在「代码看着不对」停下来，没有先确认那个信号指向的对象是对的。
- QD-904 浏览器 E2E 覆盖对账（第 12 节，无代码改动，三条假设全部查伪）：按
  「哪些 `src/views/` 页面有真实浏览器 E2E」逐页核对。有 fixture 驱动真实 Vue 页面的是
  Devices、Recordings、ScheduledTasks、Wall 五个，加上 `session-chat.html` 驱动的
  ChatPanel/FloatingToolbar 与 `remote.html` 全页；**没有浏览器 E2E 的是 NotificationsPage
  （1652 行，最大）、AccessApprovalsPage（912 行）、AccountPage、SettingsPage、
  ResetPasswordPage、AboutPage**。后两者有模型层单测（`notificationModels` /
  `accessApprovalModels`）。对 AccessApprovalsPage 提了三个假设，**三个都被查伪**，记录判据
  以免下轮重复：①「确认按钮的 busy 文案是裸 `'...'` 而非 i18n」——`'...'` 是全仓库统一的
  busy 约定（DevicesPage 六处、NotificationsPage 两处同样形状），不是缺陷；②「关闭对话框后
  `decision.approve` 不清零会污染下一次」——`openDecision` 每次都写 `approve`，对话框在它被写
  之前不读它，陈旧值不可达；③「轮询刷新会替换列表而使打开中的对话框持有陈旧副本」——服务端对
  已决定的审批回 409，UI 有专门的 `approvals.alreadyDecided` 分支并按服务端回报的状态渲染，
  这条路径是收口的。**方法学**：这一轮三次据表面形状提出缺陷、三次被代码查伪。共同点是都停在
  「这段代码看着不对」而没有走到「这个不对会造成什么」。真正的缺陷前几轮全都满足后者：
  切标签页 host 按着键、drive-relative 重命名落到工作目录、进度条倒退——每一个都能说出用户
  看到什么。说不出来的，就该继续查而不是动手改。
- 事件覆盖对账（第 9 节通知与任务的 WebClient 消费侧，无代码改动，结论已查证）：
  33 个服务端事件常量中 12 个在 `SupportedUserNotificationEvents` 内，19 个用户域事件不在。
  逐条查证后**没有发现缺口**，其中三条此前看着像疏漏：①`device.secret.rotated` 由管理员
  触发（`/v1/admin/devices/:id/secret:rotate`），已在 admin webhook 白名单与审计链内，
  用户通知通道拿不到是划界而非遗漏；②`remote_session.started/ended` 无服务端订阅者、
  也不可订阅，但 WebClient 的 `userSync` 已把它们派发为 `remote-session-changed` 供会话页
  实时刷新——实时通路是通的，缺的只是「推送通知」这一条可选通道；③
  `device.access_grant.used` 不在审计开关里、审计行数为 0，这一点清单 QD-513 与
  `docs/dev/自托管端到端联调记录.md` 都已明确记录为**刻意设计**（它与
  `access_grant.changed` 的目标就是 webhook/通知，不是审计 trail）。本轮顺带纠正自己一个
  误判：曾以为 `SupportedUserNotificationEvents` 里 `EventScheduledTaskFailed` 与
  `EventDeviceConnectPrompted` 两条注释互换，逐行比对后确认两条各自匹配其常量，是我的读错。
  该对账的判据已由 `WebClient/test/eventCoverage.test.mjs` 固化为测试（驱动真实
  `userSync._onMessage` 逐条灌入 33 个事件，断言 10 个「有页面在看」的必须派发）。
- `8805474b`（首条 track 到达前会把猜出来的 1920x1080 焊进输入 handler）：
  `_setupInputHandlers` 的回退是 `this._remoteWidth || video.videoWidth || 1920`。两个回退都
  不诚实：`video.videoWidth` 在换流期间是**上一条流**的固有尺寸（正是此前修过的
  monitor-switch 缺陷的另一个入口），而 `1920x1080` 是个看似合理的桌面——什么都不像坏的，
  而每一次点击在真实 metadata 到达前都悄悄落错位置。同时 `_resetSessionForReconnect` 不清
  `_remoteWidth`/`_remoteHeight`，于是重连后用的是**上一个会话**的分辨率，而新会话可能接的是
  另一台显示器。修为：重连时清零，没有已知分辨率时直接 return 不建 handler。这样做的安全性
  依赖一个已被测住的性质——`_mapCoordinates` 在分辨率未知时返回 null，事件被丢弃而不是被注入
  到猜的位置；本轮把那条性质也钉成了测试（反向验证：把 `|| 1920` 加回去立刻红）。
- `3bc17071`（进度事件不带 total 会把进度条打回 0%）：两端对账的最后一类。host 在传输
  开始时宣告 total，之后的事件可以不带——审计状态早就为此保留着「最后一次已知的 total」
  （`state.totalBytes = Number(...) || state.totalBytes`），而进度条用的是**事件里的原始值**：
  `totalBytes > 0 ? ... : 0`。于是一个已经传到 80% 的传输，被下一个不带 total 的事件重画成
  0%，操作员看着进度条在剩余时间里一直倒退。上传与下载两条路径同病。修为
  `js/transferProgress.js`：total 被记住，缺失或为零的那个绝不撤销已知值。**零的二义性
  是这轮真正要决定的事**：「这个文件是空的」和「这个事件没说」都以 0 到达，而意思相反——
  前者是完成，后者是未知。区分靠零到达的时机：`announceTotal` 是传输开始，那里的零是对
  文件的陈述；`update` 里来的零只是事件没说。没有这个区分，把中途的零读成空文件会让一个
  传了一半的传输显示 100%。10 条单测，反向验证（改回用原始值）零值与中途零两条同时红。
- `fc7f715a`（没有显示器时旧的显示器条会一直挂着）：`_onVideoLayout` 在设完
  `_currentDisplayList` 之后用 `if (displays.length === 0) return;` 收口，于是 host 报
  「当前没有可选显示器」时，工具栏的显示器条**从未被更新**，继续显示上一批监视器，而点其中
  任意一个就是选中一个 host 再也不会发来的 stream id。空布局是一个状态，不是「什么都没发生」。
  修为先无条件把列表同步给工具栏，再走选择逻辑；选不到东西时把分辨率与 video 元素一起放过。
  选择规则本身（已选 stream 优先 → host 主屏 → 第一条）抽成 `js/videoLayout.js`，8 条单测
  含两个此前没人测过的边界：**已选 stream 不在新布局里**必须回落到主屏而不是静默什么都不显示
  （监视器被拔掉或重建，而操作员还在看那台机器），以及空布局解析为「无」而非「保持原样」。
- `fc7f715a`（没有显示器时旧的显示器条会一直挂着）：`_onVideoLayout` 在设完
  `_currentDisplayList` 之后用 `if (displays.length === 0) return;` 收口，于是 host 报
  「当前没有可选显示器」时，工具栏的显示器条**从未被更新**，继续显示上一批监视器，而点其中
  任意一个就是选中一个 host 再也不会发来的 stream id。空布局是一个状态，不是「什么都没发生」。
  修为先无条件把列表同步给工具栏，再走选择逻辑；选不到东西时把分辨率与 video 元素一起放过。
  选择规则本身（已选 stream 优先 → host 主屏 → 第一条）抽成 `js/videoLayout.js`，8 条单测
  含两个此前没人测过的边界：**已选 stream 不在新布局里**必须回落到主屏而不是静默什么都不显示
  （监视器被拔掉或重建，而操作员还在看那台机器），以及空布局解析为「无」而非「保持原样”。
- `0260dbb3`（切标签页或切应用会让 host 一直按着键）：把两端对账后发现的真实缺口。
  Qt 客户端守住四种情形——失去焦点、窗口失活、视图不可见、窗口本身失活——注释写明了
  理由：系统快捷键或焦点变化会让 QML 收不到配对的 key-up，而下一个坐到机器前的人会
  打进卡住的修饰键里。WebClient 只监听了**接收键盘的那个元素的 blur**，那只覆盖「在页面
  内点到别处」。切浏览器标签页、切应用程序、最小化、锁屏，四种全都不在那个元素上产生
  blur，host 于是持续拿着当时按着的东西，而操作员毫无线索。其中锁屏尤其值得单独说：
  它根本不产生 window blur，只有 `visibilitychange` 会报。修为补
  `js/input/visibilityRelease.js`：三个触发条件（`visibilitychange` 且 hidden、window
  `blur`、`pagehide`），事件集合是可断言的而不是散落在代码里。`pagehide` 而不是只靠
  `beforeunload`：后者在移动浏览器不可靠，且标签页被丢弃时根本不触发。8 条单测，
  反向验证（整个删掉 visibilitychange 触发条件）三条同时红。
- `0260dbb3`（切标签页或切应用会让 host 一直按着键）：把两端对账后发现的真实缺口。
  Qt 客户端守住四种情形——失去焦点、窗口失活、视图不可见、窗口本身失活——注释写明了
  理由：系统快捷键或焦点变化会让 QML 收不到配对的 key-up，而下一个坐到机器前的人会
  打进卡住的修饰键里。WebClient 只监听了**接收键盘的那个元素的 blur**，那只覆盖「在页面
  内点到别处」。切浏览器标签页、切应用程序、最小化、锁屏，四种全都不在那个元素上产生
  blur，host 于是持续拿着当时按着的东西，而操作员毫无线索。其中锁屏尤其值得单独说：
  它根本不产生 window blur，只有 `visibilitychange` 会报。修为补
  `js/input/visibilityRelease.js`：三个触发条件（`visibilitychange` 且 hidden、window
  `blur`、`pagehide`），事件集合是可断言的而不是散落在代码里。`pagehide` 而不是只靠
  `beforeunload`：后者在移动浏览器不可靠，且标签页被丢弃时根本不触发。8 条单测，
  反向验证（整个删掉 visibilitychange 触发条件）三条同时红。
- `7057d09c`（Qt 侧重命名 drive-relative 路径会把裸名送给 host）：WebClient 早前在
  `12b91bd` 修过同一个形状，Qt 侧的 helper 是内联复制的那一份，洞一字不差。
  `C:docs` 根本没有分隔符，`parentOf` 的索引查找落空返回空串，调用方读作「没有目录部分」，
  于是 `joinPath('', 'renamed.txt')` 产出裸名——一个按 host 进程工作目录解析的相对路径。
  文件落到没人看的地方，而操作员被告知重命名成功。修为抽出 `src/common/remotepath.h`
  （header-only，与 `common/urlencode.h` 同一取舍），drive-relative 的父目录取驱动器根。
  实现 `RemoteBaseName` 时又踩到并修掉自己的第二个问题：尾部分隔符没先剥掉，
  `/home/user/` 的名字会是空串，用空名重命名等于指向目录本身——这是跑测试才红的，
  不是从需求里读出来的。新测试 `tests/remotepath_test.cpp` 按 `urlencode_test.cpp` 的
  既有形状写成独立 CMake target；本地无 Qt 工具链，另用最小 QString stub 编译并跑通全部
  断言，两个守卫各自反向验证过。
- `7057d09c`（Qt 侧重命名 drive-relative 路径会把裸名送给 host）：WebClient 早前在
  `12b91bd` 修过同一个形状，Qt 侧的 helper 是内联复制的那一份，洞一模一样。
  `parentOf` 用 `lastIndexOf` 找最后一个分隔符，而 `C:docs` **根本没有分隔符**，
  查找落空返回空串；调用方把「没有父目录」读成「没有目录部分」，于是
  `joinPath('', 'renamed.txt')` 产出裸名 `renamed.txt`——一个相对路径，按 host 进程的
  工作目录解析。文件落到没人看的地方，而操作员被告知重命名成功。修为抽出
  `src/common/remotepath.h`（header-only，与 `common/urlencode.h` 同一取舍：规则是
  inline 的，值得测的是算术不是构建），drive-relative 情形的父目录取驱动器根。
  实现 `RemoteBaseName` 时又踩到并修掉自己的第二个问题：尾部分隔符没有先剥掉，
  `/home/user/` 的名字会是空串，用空名去重命名等于指向目录本身。新测试
  `tests/remotepath_test.cpp` 按 `urlencode_test.cpp` 的既有形状写成独立 CMake target。
  本地无 Qt 工具链，因此除语法检查外另用一份最小 QString stub 编译并跑通全部断言，
  两个守卫各自反向验证过（去掉尾分隔符剥除即红）；Qt 侧构建与 ctest 交 CI。
- `493a10ae`（批量上传里一个文件失败会掩盖已经落地的那些）：`_uploadSelected` 用裸循环
  逐个 await，第一个异常被 `_run` 捕获后只渲染错误——四个文件里第三个失败时，前两个已经
  在 host 上，而操作员看到的只有一句错误，既不知道到了几个，也不知道是哪个失败，重试则会把
  已上传的再传一遍。修为跑完整个批次、记住第一个失败、两半都报。**报第一个失败而不是最后
  一个**：后面的文件还会被尝试，报最后一个等于指向恰好排在最后的那个，而不是解释批次为何
  中断的那个。完全失败时重新抛出，交给既有错误路径——那种情况下没有「部分成功」可描述，
  渲染一句部分成功覆盖全盘失败是谎报。规则抽成 `js/files/uploadBatch.js`，7 条单测含非
  Error 抛出与空选择边界。
- `13d92005`（上传 100% 的读数被它自己触发的刷新抹掉）：面板状态行有三个写入者——上传
  进度、忙/加载生命周期、以及操作后到来的列表。它们按此顺序到达，于是最后一个赢，输的是
  进度：多分块上传报完 100%，紧接着上传后的刷新报「加载中」，再紧接着列表报条目数，
  操作员**从未看到**文件传输完成的确认，而取代它的条目数说的是「列出了多少行」不是
  「到了几个」。规则抽成 `js/files/statusLine.js`：消息声明自己谈的是什么，后来的消息只在
  同一主题上取代前一个——进度谈传输，加载与条目数谈列表，互不覆盖，传输的终态因此留下。
  错误是唯一总能透出的消息。6 条单测，破坏守卫后上传与下载两条同时红。
- `2c9e37d9`（WebClient 分块大小可以与 host 契约不一致）：host 的 `file-ops` 技能把
  `MAX_CHUNK_BYTES` 定为 256 KiB，超过即报「chunk is N bytes; maximum is 262144」，
  读请求则被静默钳到该值。`RemoteFileManager` 的 `chunkBytes` 却没有任何上界，
  配大了每一次上传都会以一个点名「调用方从未选过的数字」的错误失败。补上上界钳制。
  顺带抓到同一表达式里的第二个问题：`Math.max(1, Number(x) || DEFAULT)` 让 `-1`
  通过（`|| ` 只对 0/NaN/非数字生效），于是负值被钳成 1 字节——那是个合法大小，
  于是每次上传变成每字节一次往返。改为「非有限或非正一律回退默认值」。
  3 条新测试，两个守卫各自反向验证过（去掉上界则超限那条红）。
- `609ca99b`（上一个会话的音频轨可能在新会话的画面下播放）：`_pendingAudioTrack`、
  `_receivedStreams`、`_selectedStreamId` 三个字段在会话拆卸时从未被清空。音频轨是最险的
  一个：它单独缓存，并在之后每一个 combined stream 构建时被合并进去，而守卫只看
  `readyState === 'live'`——重连后旧会话的音轨**仍然**是 live，于是旧机器的音频在新画面下
  继续播。流映射按旧会话的 stream id 建键，新会话发自己的 id，旧条目最好不可达、id 相撞时
  则挂上错误的流。三者抽成 `js/sessionMediaState.js`（「哪些字段是会话级的」才是必须做对
  的那件事，而后加的字段没人清正是这类 bug 的形状），`usableAudioTrack()` 把「live」这一条
  收在模块内，`selectedStream()` 让旧 id 解析为 null 而不是解析到已不属于此处的流。
  `_resetSessionForReconnect()` 与 `_disconnect()` 都调用 `reset()`；控制权被撤销的三处
  **不**调——那三条路会话还活着，清掉会让还在播的画面失去音频。7 条单测，`npm test`
  358/358，重连 E2E 17/17，全量 E2E 门禁 17 passed / 0 failed / 1 skipped。
- `e6a508d7`（release 在「通道已消失但标志未清」时抛异常并被吞掉）：接上一轮给
  `_cleanupInputHandlers` 加 `releaseAllInput` 之后核对它能否真的到达主机。
  `sendAction` 只检查 `_actionsReady` 一个标志，而那个标志只由 `destroy()` 清除；
  对端关闭或只清一边的拆卸会留下「标志说就绪、通道已不在」的形状，此时往
  DataChannel 写会抛异常——而所有调用方都包了 try/catch，于是 release 静默丢失，
  主机继续拿着被告知要放开的按键。修为同时检查通道对象是否存在。**这一轮的测试
  教训值得记**：第一版测试伪造了一个自己的 `close()`，破坏产品守卫后它依然全绿，
  等于没有看守——因为被测行为全在我自己的假件里。改为驱动真实
  `DataChannelHandler` 后才红。另一次 `destroy()` 不清 `_actionsReady` 的破坏也没被
  抓到，原因是那条路有双重保护（标志 + 空通道），单破其一不红；这本身是信息：
  该边界由两个条件共同承担，测试必须打那个唯一失效的形状。
- `d3fd4a67`（切换显示器后统计浮层出现负帧率）：`video-stats.js` 的速率是从两个
  `getStats()` 样本的累计计数器相减得来的，而切换显示器会换掉媒体流、新流计数器从零开始。
  相减结果是大额负数，浮层于是显示负 fps 和负码率——操作员读作「解码器在倒着跑」。同时
  首个样本没有前值可减，`timeDelta` 为 0 时还会除出 Infinity。规则抽成
  `js/ui/videoRates.js`：计数器倒退一律报 0 而非负数（计数器变小只可能是被替换，不是时间
  倒流），`isComparableSample` 决定前值是否还值得留——一旦流被换掉，之后每次与旧样本比较
  都是无意义的，所以丢掉而不是留着每次切显示器产出一个假速率。8 条单测含两个边界（计数器
  倒退、零/负时间跨度），红绿双向。这也补上了 QD-311 自适应质量所依赖的统计层此前完全
  没有的覆盖。
- `db852e67`（「原始分辨率」一直给第一台显示器的尺寸）：该菜单项由 toolbar 第一次被告知的
  分辨率构建，而 `onloadedmetadata` 只为第一条流触发——切显示器后条目仍报第一台的几何尺寸，
  点它会把桌面改成与当前画面不一致的分辨率。修为「原始」跟随正在观看的显示器，规则抽成
  `js/ui/originalResolution.js`（toolbar 只能靠搭起整个菜单来测，而规则才是值得测的部分），
  并给两条会改变当前显示器的路径都补上 `setRemoteResolution` 调用。4 条单测含零值/负值/
  未知名边界。过程中先写过一版直接测 `FloatingToolbar` 的测试，被它构造函数里的 DOM 面积
  挡回，改为抽模块——这与 `js/wall/tileModel.js` 的既有取舍一致。
- `25f088f6` + `d5835131`（输入与光标链路补上此前完全不存在的覆盖）：接上一轮把三个
   handler 的真实缺陷抓出来之后，本轮把剩下两条没有测试的核心链路也钉住。**剪贴板同步环**：
   四条款式测试覆盖「主机来的文本不回传」「本地复制的文本只发一次」「本地写入被拒不会让
   上传循环重启」「文档失焦时不读剪贴板」。前三款原本就没有任何自动化守护，而它们守的正是
   `_lastClipboardText` 这个环闭点——环一开，会回声的主机每秒把同一段文本发回一次，持续整个
   会话。破坏守卫后三条同时红，证明它们是真门禁。**远程光标渲染**：四款覆盖 BGRA 通道交换
   （主机蓝在前、画布红在前）、截断光标不越界且缺像素留透明而非填脏值、hotspot 被钳进位图内
   （host 报出界外会让每次点击偏移那么多）、空形状被忽略而不是在 0x0 canvas 上崩掉把整个视频
   面板带走。四个守卫各自反向验证过。本轮共 8 条新测试，`npm test` 336/336。
- `2769c9bf`（切换显示器后每次点击都落在旧桌面的缩放位置上）：`_mapCoordinates` 里
   `videoWidth || remoteWidth` 的优先级是反的。切换显示器时 `<video>` 元素仍在上一条流的
   metadata 过期前一直报告**旧**监视器的固有尺寸，而输入 handler 已经被 `setRemoteResolution`
   告知了新尺寸——旧值赢，于是每次点击都按旧桌面的比例落到新桌面上。鼠标 handler 与触摸
   handler 的虚拟光标点（`_updateCursorElement`）同病，后者是手机上唯一告诉手指「远程指针
   在哪」的东西。修为「被告知的分辨率优先，固有尺寸只作兜底」。两个测试各自红绿双向验证：
   鼠标那条 revert 后 x 从 750 变 1000；触摸那条第一版探针取的是画面**中心**，而中心在两种
   缩放下都落在同一点，测试因此永远绿——那是个装饰性守卫不是门禁，改成取四分之一处后
   revert 立刻红（y 从 350 变 343.75）。这正是「一个从未见它失败的检查等于没有」的又一例。
- `2c72e87a`（WebClient 输入链路三处 stuck-state 缺陷，全部先红后绿两轮验证）：主线输入
   handlers 此前**零测试覆盖**——`rg` 过 `WebClient/test/`，mouse/keyboard/touch 三个
   handler 一个都没有对应测试文件，而它们正是「控制」这件事的全部。本轮补 15 条测试并
   当场抓到三个真缺陷。**①拖出画面不松键**：`MouseHandler` 监听的是 `mouseup`，而鼠标
   拖出 video 元素后那个事件落在别的元素上，handler 永远收不到——远程机器就这样保持按下
   状态，直到下一次点击恰好又落在画面上。修法是按下时 `setPointerCapture`，松键时
   `releasePointerCapture`，`disable()` 里也释放（捕获比 handler 活得长会把后续鼠标事件
   重定向到一个不再监听的元素上，远程光标会僵住而页面看着还正常）。**②捏合手指抬起产生
   幻影点击**：两指捏合时 `_onTouchStart` 把 `_moved` 置 true，而 `_onTouchEnd` 的
   「多于一指落到一指」分支为延续手势把它重置为 false——于是 surviving finger 抬手时走进
   单指分支、`_moved` 为 false，主机收到一次用户从没做过的左键单击。修法是引入
   `_fromPinch`：从捏合延续出来的手势永不产生点击。**③控制端消失后主机按键卡住**：
   Qt 客户端用 `releaseAllInput` 守住了同一类暴露（焦点丢失和窗口失活都调用），WebClient
   却从未发送过它，而 `_cleanupInputHandlers()` 会把还没出去的 key-up 一并丢掉——下一个
   坐到机器前的人，每次按键都带着上一个会话卡住的修饰键。`releaseAllInput` 的枚举值不是
   猜的：从预编译 host 二进制的 descriptor 表里读出
   `sendAttentionSequence=1 / lockWorkstation=2 / privacyScreen=3 / (off)=4 / releaseAllInput=5`，
   与既有 1~4 完全吻合后补 5。同时发现 actions channel 的创建条件只列了另外三个能力，
   一个只支持 releaseAllInput 的 host 将拿不到 channel——恰好是唯一能防 stuck input 的
   那个动作发不出去，已把该能力按自身资格加入条件。验证：15 条新测试（红绿双向各两轮：
   去掉 pointer capture、去掉 letterbox 守卫、去掉 deltaMode 归一化、去掉 `_fromPinch`
   守卫、去掉 release 的枚举项、去掉能力门禁，各自被对应测试抓住），`npm test` 326/326，
   全量 E2E 门禁 17 passed / 0 failed / 1 skipped（skipped 为两小时 wall-soak），
   生产构建绿，`git diff --check` 干净。
- `d44a9680`（11 个候选分支与 dev 的内容对账，结论：内容已在 dev）：上一轮发现这 11 个分支 rebase 时冲突集中在录像文件，一度判断为「我们的旧录像实现 vs 上游 PR #23」。逐层查证后**该判断被推翻**，真实情况是它们的内容早已全部落在 `dev`：①按签名比对录像服务，上游 16 个方法我们全有，只多 `WithOutbox`（outbox 集成）与三个会话维度查询（`ListForSession`/`ListForSessionIDs`/`ListForSessions`）——dev 的录像实现是上游功能的**超集**，不是另一套；②11 个分支相对共同基点 `f6651f03` 的改动并集共 1233 个文件，用 `-z` 正确解转义后逐一核对，**在 dev 中缺失的为 0**；③抽两个最有代表性的主题实证：SQLite 模式在 dev 有 `internal/database/sqlite.go` 且 5 个测试全过（覆盖比分枝的 131 行版本更多），踢人冷却在 dev 有 `internal/service/revoke_cooldown.go` 连同测试；④`dev` 自身 `go build ./...` 与 database/service 测试绿。**所以这些分支不是待拆分的工作，而是已完成工作的旧切片**。它们的冲突源于基于早期控制面提交，而 dev 早已演进到超越上游。U2 批次「把 PR #25 拆成小 PR」的前提因此改变：拆分的目的是让上游能逐个审查，而内容侧 dev 已经完备。过程中修掉自己两个测量错误：一是用提交对象判断「是否在 dev」（`merge-base --is-ancestor` 全返回 no），但 dev 是用不同提交对象实现同样功能，提交级比较毫无意义，必须比内容；二是 `git diff --name-only` 对中文路径加引号而我的 Python 没解转义，一度误报「2 个文档只在分支有」，改用 `-z` 后归零。另外清理了一次合并实验留下的未提交状态、临时分支 `union-analysis` 和两个 `__pycache__/`。
- `TODO8`（29 个上游候选分支的状态核对与 rebase）：清单多份条目写着「远端写入恢复后推送并建 PR」，本轮核实这条路**走不通**：`git push upstream` 返回 `Permission to barry-ran/QuickDesk.git denied to ilh-github`，且 `ilh-github/QuickDesk` 这个 fork 不存在。不是时机问题，是权限问题——我们既没有上游写权限，也没有 fork 可推。**能做且已做的是 rebase**：上游 dev 已从各分支的基点（`f6651f0`/`5db316c`）前进 7-8 个提交到 `88d4d36`（PR #23 录像落地）。15 个候选分支已全部 rebase 到最新上游、落后 0 提交，随时可提 PR 且不带冲突：`qd-911-security`、`qd-114-spki`、`qd-115-trust-model`、`qd-212-local-device`、`qd-213-access-code-refresh`、`qd-214-file-transfer-errors`、`qd-214-spake2-webclient`、`qd-215-win-runtime-check`、`qd-403-webclient`、`qd-506-recording-upload`、`qd-510-workflow`、`qd-002-file-ops`、`mcp-failure-lock`、`signaling-postgres-migrate`、`cross-tab-coordination`。rebase 后抽查 `qd-002-file-ops`：`cargo build -p file-ops` 通过、4 个测试全过。顺带清理了 8 个 prunable worktree 和一个我上次跑守卫时留下的 `scripts/__pycache__/`。**两类分支没能 rebase，原因与处置**：①10 个分支（`qd-006-sqlite`、`qd-111`、`qd-604-metrics`、`qd-607-batch-revoke`、`qd-204-api`、`qd-204-client`、`event-outbox`、`event-replay-retention`、`websocket-auth-timeout`、`access-bans`、`qd-608-extra-sdks`）全部基于我们 fork 的 PR #25 巨型提交，冲突集中在 8 个录像相关文件——那不是代码冲突，是「我们整个自托管控制面 vs 上游刚合并的 PR #23 录像重写」。解法仍是清单 U2 既有的结论：把 PR #25 拆成可独立审查的小 PR，逐个从 `upstream/dev` 切出。②`qd-316-session-chat-v2` 冲突在 `QuickDesk/qml/views/RemoteWindow.qml`，且是**同一个 UI 元素的用途冲突**：上游把该 Rectangle 改成录制按钮，我们的分支把它当聊天按钮。这需要产品判断（两个按钮如何共存），已 `rebase --abort` 保持原状，未擅自解。另发现 `qd-316-session-chat` 与 `cross-machine-queue` 两个分支相对上游 dev 是 0 提交/0 文件——停在旧基点且无自有提交，疑似曾推送并被合并，待核实后应删除以免混淆。
- `5fc8c42a`（smoke 还原守卫接入 CI）：接上一轮的静态守卫。原放在 `scripts/` 根下，接入时发现 `signaling-server-verify.yml` 的 `paths` 触发器只有 `SignalingServer/**`、`sdk/**`、`scripts/sdk/**`——**不含 `scripts/` 根**，也就是改守卫本身不会触发任何流水线，一个不会被触发的门禁等于没有。移到 `scripts/sdk/`（与它守护的 smoke 同目录），ROOT 计算相应改为上溯三级。守卫步落在 `contract` job、紧跟「Committed clients match the spec」之后，注释写清它为何是静态的：要回答的是 smoke **将会**发什么，不是活服务器答什么，所以不需要起服务。移动与改路径后**重新做了反向验证**（删掉 snooze 行 → exit 1 且指名结构体；恢复 → exit 0），并在仓库根与 `scripts/sdk/` 两个位置分别调用确认路径解析正确。验证：workflow YAML 解析通过，守卫步确认在 `contract` job 且 `scripts/sdk/**` 已在触发器内；`python3 scripts/sdk/check-smoke-restore-coverage.py` exit 0；`node --check scripts/sdk/openapi-smoke.mjs` 通过；`git diff --check` 干净。
- `1ff5ea83`（一致性 smoke 的还原体会静默清掉 snooze）：见 QD-910/QD-005 边界。`openapi.yaml` 开头承诺 smoke 碰过的规则「put back exactly as they were」，但 offline-alert 的还原体漏了 `snoozed_until`——而 `deviceOfflineAlertReq` 的注释明写「Omitted or null clears an existing snooze」。所以探测前若该规则有静默期，smoke 跑完会**把它抹掉并报告成功**。同一个文件两百行之下，metric-alert 的还原体却带着这个字段（`snoozed_until ?? null`）——同文件内两处不一致，正是漂移的形状。修好之后加了一道静态守卫 `scripts/check-smoke-restore-coverage.py`：把每个 PUT 请求结构体的 json 字段与 smoke 还原体实际提及的字段名对比，漏一个就失败并指名道姓。这类缺陷会复发（以后新增任何可清除字段的 PUT 都可能踩同一个坑），只修这一处不够。**守卫做过反向验证**：临时删掉 offline-alert 的 snooze 行，守卫立刻报 `restore body omits ['snoozed_until'] from deviceOfflineAlertReq` 并 exit 1；恢复后 exit 0。另外核对了守卫的结构体匹配不是偶然命中——两个路由各自精确匹配到唯一正确的结构体，字段列表逐条比对过。本轮另外把几个对撞维度扫到了干净，记录结论以免重复劳动：①请求体字段对撞（精确到「路由→handler→绑定结构体」）只剩 `machine_fingerprint`（Qt 被控端发）与 `challenge`（控制器质证链发），均非缺口；②服务端内部「同一规则两处实现」的扫描无果——`IPAllowed` 与 `NormalizeControlMode` 都是单一权威函数多处调用，`session_control.go` 的 `mode != exclusive` 比较安全（其入参已归一化）；③OpenAPI 路由对撞初看有 79 条缺失，实为脚本缺陷（spec 的 path 带 `/v1` 前缀、OpenAPI 用命名参数），修正后确认 spec 是**刻意**的子集——文件开头自述「deliberately covers a subset」，漏的路由由 `docs/user-api-docs.md` 覆盖并按 conformance test 逐步纳入，且 `access-grants` 确实已在 spec 内且被 smoke 的 GET/POST 打到。验证：`python3 scripts/check-smoke-restore-coverage.py` exit 0；`node --check scripts/sdk/openapi-smoke.mjs` 通过；管理端 `npm test` 48 pass / 3 skip / 0 fail；`git diff --check` 干净。
- `6d50bc1f`（QD-107 访客授权接入 Web 控制端）：承接上轮的模型层，把面板接进 `SessionsPage.vue`（访问策略 UI 所在页），并补 `listAccessGrants`/`createAccessGrant`/`deleteAccessGrant` 三个 api 方法。默认值刻意保守：一小时、一次会话，误提交不会开出开放式授权码。授权码只在创建响应里出现一次，所以直接进确认文案——owner 必须把它传给客人。**真实服务端 E2E 抓到自己两个契约缺陷**，这是本轮最该记的部分：①我把 `bound_ips` 序列化成 JSON **字符串**发送，而服务端请求结构体是 `[]string`——我看了存储列的 `type:text` 就推断线上形状，结果 400 报错只说 "cannot unmarshal string into Go struct field ... of type []string"，对调用方毫无信息量；②服务端**响应**的 `bound_ips` 是数组，而我的 `describeBinding` 按 JSON 字符串解析，于是渲染成 "(unreadable address list)"。两处都因为「存储形状 ≠ 线上形状」这一个误解。修好后 `describeBinding` 同时接受数组与字符串两种形状——响应走数组，而经过会把数组压平的存储读出来的行仍是字符串，拒掉那种会让有绑定的授权显示成没绑定。**还修一个渲染缺陷**：模板原先用 `$t('sessions.grantState' + grant.reason)` 拼接键名，而模型的 reason 是蛇形（`used_up`/`unknown`）、i18n 键是驼峰（`grantStateUsedUp`/`grantStateUnknown`）——两者不构成任何规则，漂移时静默渲染出原始键名。改为由模型直接携带自己状态的 i18n 键。验证：26 条模型测试（上轮 13 + 本轮 13：状态键逐条对应、活行不带状态键、地址按数组发送、字符串形状的地址仍可读、坏地址抛错），`npm test` 311/311 绿（原 298）；生产构建绿；E2E 全量门禁 17 passed / 1 skipped / 18 total；`git diff --check` 干净。真实服务端 E2E 全闭环：创建 201（`bound_ips` 以数组送达）→ 列表渲染为 `tab-7 · 203.0.113.5, 10.0.0.0/8` → 撤销 200 → 撤销行 `usable:false`/`stateKey:sessions.grantStateRevoked`，同设备另一条授权不受影响。红绿双向：`describeBinding` 改回只认字符串则数组形状那条红。
- `a6cf05fd`（QD-107 访客授权在两个控制端都没有入口）：用请求体字段对撞法扫出。服务端 `POST/GET/DELETE /v1/me/devices/:device_id/access-grants` 三个端点早已实现（QD-107 也标 `[x]`），但 **WebClient 与 Qt 两侧零调用**——`rg access-grants` 在 `QuickDesk/src`、`QuickDesk/qml`、`WebClient/src`、`WebClient/js` 全部零命中，只有 i18n 里一个无关的事件标签 `accessGranted`（通知用）和七套生成 SDK。也就是说清单标 `[x]` 的依据是「服务端 + SDK 完成」，而用户实际拿不到这个功能。这与本清单自己反复强调的「服务端有、界面无就是半成品」是同一形状，只是这次跨了两个控制端。本轮先做**表单序列化模型**（UI 接入留待下一步，不把半成品标成完成）：`src/api/accessGrantModels.js` 的 `buildAccessGrantPayload()`，把五个入参（`expires_in_minutes`/`max_uses`/`label`/`bound_client_id`/`bound_client_public_key`/`bound_ips`）的构造与校验固定下来。三处客户端校验的理由都是「服务端的拒绝正确但来得太晚」：密钥格式错误要等客人连不上才发现；坏地址要等兑换时才发现；公钥没有配 client id 会产生一个「绑定生效但 who 列为空」的授权。**实现时抓到自己一个真缺陷**：`bound_ips` 原先用 `.filter(Boolean)` 丢掉解析失败的条目——用户明明输了地址却被静默忽略，绑定在不告知的情况下变窄，而 owner 会以为泄露的码在某处没用（其实有用）。改为逐条校验、失败即抛错并点名是哪一条。验证：13 条新测试（寿命必填且必须为正、未填次数送 0 而非用户没输过的数字、负数次数拒绝、label 去空白与空值省略、client id 绑定、公钥必须配 client id、公钥 base64+32 字节形状、地址数组序列化、坏地址抛错、已过期窗口拒绝），`npm test` 298/298 绿（原 285）；红绿双向（模块不存在时 13 条全红；把 `filter` 改回去则坏地址那条红）。`git diff --check` 干净。**未做**：UI 接入、真实服务端 E2E、清单状态仍维持 `[x]` 之外的说明补充。
- `32e2a568`（审计列表六个过滤器只发了五个）：见 QD-110。`/v1/admin/audit-logs` 服务端接受 `action`/`admin`/`username`/`actor_user_id`/`date_from`/`date_to` 六个过滤，管理端 `getAuditLogs` 只发后五个之外的那五个——漏的正是 `username` 与 `actor_user_id`。其中 `username` 是关键：它一次匹配**管理员名与终端用户名**（不区分大小写），所以缺了它，「这个账号做了什么」在这块屏幕上根本没有答案，而这正是这块屏幕存在的理由。`actor_user_id` 则把管理员操作与终端用户自己的操作区分开。query 构造提成 `src/views/audit-filter.js` 以便测试，页面补两个输入框。两处客户端校验是刻意的：①**空值省略而非送空串**——服务端做子串匹配，空串等于「匹配全部」，会把结果悄悄放宽而不是收窄；②反序日期窗口直接抛错——服务端对空窗口回答空页，那会把人派去找一批不可能存在的行。`actor_user_id` 非数字或非正整数同样抛错，因为服务端对它回 400 而不是当作无过滤，一个手误会让整页空白。验证：8 条新测试（六个过滤全部可达、空值省略、纯空白 trim、actor id 非数字/负数抛错而数字字符串接受、反序日期抛错、同日窗口合法、分页参数同行），`npm test` 48 pass / 3 skip / 0 fail（3 skip 为需真实服务端的 E2E）；生产构建绿且四个新 i18n 键逐个 grep 确认进产物。真实服务端 E2E：`username=noisy`→2 行、`username=NOISY`→2 行（证明不区分大小写）、`username=admin`→4 行、无过滤 7 行、`actor_user_id=abc`→400。`actor_user_id=7` 返回 0 已核实为正确语义而非缺陷：该列是终端用户列，种子行的 `admin_id=7` 与 `actor_user_id=0` 是两个不同字段。`git diff --check` 干净，红绿双向（模块不存在时 8 条全红）。
- `a49be061`（通知重投的七个服务端参数一个都没接）：见 QD-909/QD-610。`POST /v1/me/notification-channels/:id/deliveries:retry-failed` 与跨通道的 `:retry-failed` 服务端接受七个参数（`event_type`/`device_id`/`limit`/`dry_run`/`spread_over_minutes`/`channel_ids`/`created_after`/`created_before`），而 WebClient 两处调用都传**空对象**——API 层早已透传 body，缺的是 UI 入口，于是七个参数全部不可达。其中两个是安全特性而不是便利功能，这是值得单独做一轮的理由：`dry_run` 让操作员在真正入队前看到会影响多少行；`spread_over_minutes` 把积压分摊到若干分钟，而这恰恰重要——**把行打成 failed 的那次故障，也正是端点此刻脆弱的原因**。默认值因此刻意设为「预览开启、不分摊」：误点一下不会把刚恢复的端点一次性打满。新增 `buildRetryFailedPayload()` 并把两处调用接上。三个设计点：①**空字段省略而不是送 0**——服务端把 0 读作「无上限/不分摊」，把用户没填的值送成 0 会让一次未输入看起来像一次明确选择；②时间窗在客户端就地校验（反序窗口、非法时间戳直接抛错），因为服务端对空窗口回答「matched: 0」，那会把操作员派去找一批不可能存在的行；③toast 按 `dry_run` 分成两句——「将影响 N 条，尚未入队」与「已重新入队 N 条」是两件不同的事，合成一句就是谎报。验证：11 条新测试（空载荷、dry_run 双向、分摊与上限的正值才发送、空值省略、过滤器去空白、时间窗 RFC3339、反序与非法时间戳抛错、channel_ids 校验），`npm test` 285/285 绿（原 274）；生产构建绿且六个新 i18n 键逐个 grep 确认进产物（中英文各三）；E2E 全量门禁 17 passed / 1 skipped / 18 total；`git diff --check` 干净；红绿双向（函数不存在时 11 条全红）。
- `16293225`（路由覆盖对撞工具，及两条客户端线的零缺口结论）：把前几轮人肉找到三个管理端缺口的方法固化成 `scripts/check-route-coverage.py`。它对比 `cmd/signaling/main.go` 的路由表与各客户端树里出现的字符串，按 `--group admin|me` 分别对撞（两组接收者不同：管理端挂在 `adminGuarded`，用户侧是 `me`；混在一起会把管理端报成缺少它从不该调用的 owner 端点）。**它刻意是报告而不是门禁**，理由写进 docstring：一个路由是否真能到达是代码路径问题，字符串搜索答不了——`storage` 一词会因完全无关的原因出现在客户端里，所以自动判定要么狼来了要么对真缺口沉默。它做的是把路由表和候选调用方并排列出，省掉每次重新推导路由表，判断仍由人做。`--exit-nonzero` 只在 **base 级**失败，那一档粗但安全：能抓住整个前缀无客户端提及的路由，对按动作分的细粒度保持沉默。因此**没有接入 CI**——一个会漏报的门禁比没有门禁更糟，它给的是虚假的安全感。对撞结果：管理端 53 条路由 / 20 个 base、用户侧 57 条路由 / 19 个 base，**全部有客户端可达，零缺口**。这解释了前几轮的体感：这个维度确实已经做完了，继续在这里找产出会越来越低。过程中修掉自己两个失误：正则只匹配字面接收者导致只扫到 3 条路由（真正的是 53 条）；以及 docstring 替换时把旧正文残留在新 docstring 之后造成语法错误，用「定位首尾引号、整段重建」修正。**验证是反向做的**：把 `SignalingServer/web/src` 整个移走，脚本立刻报出 53 条全部 MISSING 且 `--exit-nonzero` 返回 1；移回后 `npm run build` 绿、`npm test` 40 pass / 3 skip / 0 fail（3 条 skip 是需要真实服务端的 E2E）。另外核对并恢复了 `RemoteSessionsPage.vue` 的 import 顺序，使工作树回到干净状态。**下一步转向**：QD-908 剩余 200 路 7200 秒夜间正式跑（需 2.5 小时以上单窗口）；QD-004/QD-005/QD-511 分别卡在 Remoting 协议、Apple 凭据、外部 AI 服务。
- `1cd2eae8`（QD-907 最后一项「代理切换仍未验证」）：这是清单自己指定的下一步，也是 QD-907 五类异常里唯一没做的一条。它与已有两类都不同：socket 关闭会来一个事件，进程级断网至少有浏览器自己的 offline 状态，而**代理/VPN 切换下浏览器仍认为在线、socket 仍是 OPEN、什么事件都不触发**——包只是不再到达，因为路径在一条活连接下面移开了。`isConnected()` 只看 `readyState === OPEN`，所以这种链路会被永久报告成「已连接」，操作员只能靠「打字对方收不到」发现。修法没有新增心跳协议：信令服务器本来每 30 秒发一次 WS ping，健康链路 therefore 从不久静默，所以「下行静默过久」本身就是可用信号。`WebSocketTransport` 新增 `stalenessTimeoutMs`（默认 90s＝三个 ping 间隔，足够宽到不把慢链路当坏链路），任何下行帧都重新打点——忙碌会话靠帧续命，定时器根本不会响。触发时以 4000 主动关闭，复用 session 既有的 close→重连机制，而不是再加一条恢复路径。经 URL 参数 `staleness` 注入以便测试缩短。验证：`reconnect.e2e.mjs` 新增一条黑洞测试（连接建立、auth 成功、之后所有下行被静默吞掉且不 close），该测试**第一次跑就失败**，失败方式是等到超时也没出现第二个连接——缺陷因此现形，不是先写实现再补绿。红绿双向：摘掉打点则只有这一条红、其余 16 条不受影响。`reconnect.e2e.mjs` 17/17 绿（原 11），`npm test` 274/274 绿，E2E 全量门禁 17 passed / 1 skipped / 18 total，`git diff --check` 干净。过程中纠正自己两次：注释块被字符串替换打乱后反复修歪，最后用「定位起止索引重建整段」一次搞定；以及把 staleness 参数误插进 `_detectMobile()` 的作用域，运行时才发作用域不匹配，改为在使用处 `_connect()` 内读取（与文件既有习惯一致）。
- `0c1c4c2`（QD-107 与 QD-314 逐条核对，均未发现缺陷）：按「问条目自己的承诺」逐条查证两项标 `[x]`/`[~]` 的功能。QD-107 四条承诺全部有字段支撑且被真实使用——限时 `ExpiresAt`、限次 `MaxUses`/`Uses`、可撤销 `RevokedAt`、IP/client 绑定（`BoundClientID` 是摩擦、`BoundIPs` 按服务端观测地址强制、`BoundClientPublicKey` 升级为 Ed25519 签名质证）。质证链路查得很干净：空 challenge/signature 直接拒、challenge 绑定 device+grant+client 防重放、验签在 host-online 检查**之前**执行（否则在线状态会变成预言机），失败计入限流并写审计；`controller_challenge_test.go` 七条测试覆盖空值、绑定、过期、密钥形状。过程中一度以为抓到缺陷：`UsableAt` 用 `MaxUses <= 0` 判无上限，而 redeem SQL 用 `max_uses = 0 OR uses < max_uses`，负数下两者语义相反（前者可用、后者永不可达）。写测试确认分歧真实存在后才发现 `CreateWithEvent` 第 91 行早已拒绝 `MaxUses < 0`，且该拒绝**已有测试**（`MaxUses: -1` → `ErrInvalidAccessGrant`）。所以负数到不了那两个判断点，缺陷不成立——我那条测试断言的是一个不可达状态，已删除。真正的产出是把这道**隐含防线**写进 `UsableAt` 的注释：它说明 `<= 0` 是有意语义、SQL 那侧如何表述同一规则、以及为何 Create 必须拒绝负数，避免后来者「简化」比较符时重新打开这个分歧。QD-314 的核对结论同样是否决了一个可疑缺口：Qt 侧 `control_grant`/`control_revoked` 零引用，看似该补。但查证 `sendToClient` 寻址的是 `signalClient[deviceID][clientID]`——这两个帧**只发给 controller 连接**，而 Qt 的 `CloudDeviceManager` 用 `access_token` 连的是用户级事件通道（收 `device.*`/`favorite.*`/`session.revoked`）。Qt 作为被控端本来就不需要知道谁在控它；它作为 controller 时信令 WS 由预编译 Remoting 持有，代码层确实读不到。**原清单「剩余：Qt 客户端不消费这两个帧（需动 Remoting）」的判断是正确的**，不是待补项。验证：`go test ./internal/service/ ./internal/handler/` 全绿；`go build ./...` 与 `go vet` 干净；`git diff --check` 干净。本轮无新增测试（证明无缺陷），净改动为一条注释。
- `7a6564d9`（QD-316 的 4000 字边界双端都没有测试）：边界本身是真的——Qt 侧 `SessionChatManager::validateText` 与 Web 侧 `SessionChatModel::send` 都强制它，「不持久化」也确认成立（Web 侧聊天不碰任何存储）。但两侧各有一份模块私有常量，且**没有任何一条测试断言它**：把常量改大、或把强制逻辑删掉，不会有任何东西红。这正是最容易被「清理魔法数字」改坏而不被发现的那类限制。Web 侧把 `MAX_CHAT_TEXT_LENGTH` 导出并补 3 条测试：临界值发送成功、超一字符被拒且**不发出**、超长字符（多字节）按字符而非字节计。Qt 侧补同一组边界，断言刻意走 `errorOccurred` 信号而非只查返回值——拒绝必须对 UI 可见，一个返回 false 却不发任何信号的实现会让用户对着没反应的发送键白按。`sendToControllers(atLimit)` 一条**没有**写：它校验通过后还依赖 skillHost 的连接状态，返回值不确定，写了就是一条会偶发失败的测试。两侧都按 UTF-16 code unit 计长（Qt `QString::size()`、JS `.length`），这是两端能互信对方拒绝的原因，测试里用多字节字符钉住。验证：WebClient `npm test` 274/274 绿（原 271 + 3）；`git diff --check` 干净；红绿双向（把边界改小则新测试红）。**Qt 测试本地未运行**——本机无 Qt 工具链，只做了审阅核对：`errorOccurred(QString, QString)` 签名与 `QSignalSpy` 用法均与文件内既有 spy 一致，`QString(int, QChar)` 与 `QLatin1Char` 为标准 Qt，且 `sessionchatmanager_test.cpp` 确认在 `tests/CMakeLists.txt` 第 225 行的构建图内；首次运行在 Windows 与 macOS 的 CI job。
- `b4d1cdfb`（回放断言五类条件只有两类可达）：见 QD-511。Host 的 `VerificationService::evaluate` 一直实现五类条件（`text_present`/`text_absent`/`text_present_exact`/`window_title_contains`/`window_title_equals`），而工作流的 `StepExpectation.kind` 只认 `screen_hash` 和 `screen_contains`——另外三类无法从任何步骤表达。其中 `screen_absent` 是最常用的那类：「错误提示已消失」「弹窗已关闭」只能等超时。补三个文本类与两个窗口标题类 kind，各自映射到 Host 自己的条件名（沿用既定的「用 Host 的词汇说话」规则，`screen_contains`→`text_present` 那一套）。**接映射时抓到第二处、也更危险的半成品**：`classify_mismatch` 自己另有一份支持列表，仍只含那两个 kind。于是新 kind 能真的发到 Host 并通过，但只要不匹配就会被归类为 `UnsupportedKind`，park 报告告诉运维「该检查类型本 build 不支持」——而它完全支持。这比映射缺失更糟：它把一次正常的屏幕不匹配伪装成功能缺陷，把人送去找一个不存在的缺失。两份列表因此成了漂移点，补了一条守卫测试强制它们一致：本文件会发给 Host 的每个 kind，`workflow.rs` 必须也接受；反之本文件拒绝的每个 kind，分类器也必须拒绝。同时把工具参数描述与 `StepExpectation` 的文档注释从「两个 kind」改全——用户看不见的能力等于没做。验证：`cargo test --offline --bin quickdesk-mcp` 160/160 绿（原 119 条基础上新增 5 条：五类全可达、未知 kind 仍拒绝、Host 支持的 kind 不被误判、空值先于 kind 判定、两份列表一致）；`cargo clippy --all-targets -- -D warnings` 零告警；红绿双向两轮——摘掉 `screen_absent` 映射则 1 条红，收窄分类器支持列表则 3 条红（含漂移守卫），均确认破坏即红后恢复。WebClient 271/271 未触及仍绿，`git diff --check` 干净。
- `42426e58`（审计保留清理没有入口）：见 QD-110。`POST /v1/admin/audit-logs:sweep` 服务端早已实现，六个界限（age、per-actor、per-device、per-action、全表行数、归档文件保留）都可分别覆盖，管理端界面却没有任何调用方——运维只能等定时 worker，想立刻验证保留策略行为只能手工 curl。审计页补上「保留清理」按钮与两个对话框（参数输入 + 按界限拆分的结果）。结果按界限逐行列出的理由写在这里：合成一个「共移除 N 行」会藏掉哪个界限做了功，而「我的历史为什么没了」的答案正是那个。模型刻意区分两种空：**未配置的界限不出现在列表里**，而**配置了却没删掉任何行的界限以 0 列出**——「没设这个界限」和「设了但什么都没过期」是两个不同的事实，第二种恰恰是运维最需要看到的（它证明界限在生效）。验证：8 条模型测试（配置项逐条列出、未配置不出现、跑了零删除仍列出、总数跨界限汇总、冷存目录、cutoff），管理端 `npm test` 43/43 绿、生产构建绿且新键确认进产物（上一轮曾因构建绿掩盖缺键，这次逐个 grep 校对）；真实服务端 E2E——7 行种子（3 条 45 天前 + 4 条同一操作者近期），sweep 后 `purged:3`、`actor_cap_removed:2`，未配置的四界限全 0，列表 7→5 行，归档 JSONL 995 字节真实落盘；面板模型对真实响应渲染为 `[{age 30d, 0}, {per-actor 2, 3}]`，只列实际配置的两项；完全无配置时空 body 返回 400 且服务端原文通过 toast 显示，这正是 UI 需要写「至少设置一个」的原因。WebClient 271/271 未触及仍绿，`git diff --check` 干净，红绿双向（两类空分别验证）。
- `e066d37`（管理端能列录像、能删录像，却拿不到文件）：见 QD-810/QD-208。`GET /v1/admin/recordings/:id/artifact` 与 `/:id/thumbnail` 服务端早已实现并写审计，管理端界面却没有任何消费方——运维能看一个账号有几条录像、各多少字节、能删掉它们，但要看内容只能手工 curl，而「能统计不能取证」正是这类控制台最没用的形状。会话控制台的录像行补上下载按钮与缩略图。两个边界按「答错比没有按钮更糟」处理：①只在行真的持有媒体时给按钮——服务端还在写的录像没有字节，那里的下载是注定失败且不给解释；②只在行有缩略图路径时渲染 `<el-image>`，表格里一张破图读起来是 bug 而不是「没采集到」。下载走 `authFetch` + blob URL 而非裸链接：浏览器没法给 `<a href>` 或 `<img src>` 带 Bearer token，这是仓库里既有的下载模式（`api/system.js` 的 `downloadFile` 同一形状），没有另造一套短时签名 URL。缩略图 blob URL 按行缓存，重新渲染即回收会让运维正在看的图变黑。验证：7 条模型测试（路由归属、两类可得性守卫、文件名、id 转义），管理端 `npm test` 35/35 绿、生产构建绿；真实服务端 E2E——造一条 `completed` 且有真实 artifact 文件的录像，artifact 返回 200 + `Content-Disposition: inline; filename="seed-recording.h264"` + `video/h264`，46 字节内容逐字一致，thumbnail 200 返回 9 字节，列表行带 `thumbnail_path`；边界：无 token 401、不存在 id 404、非数字 id 400；审计表确认写入 `recording.download` 行（服务端注释说读用户屏幕捕获正是管理日志应能解释的那类访问）。WebClient 271/271 未触及仍绿，`git diff --check` 干净，红绿双向（两类守卫分别验证）。过程中删掉一个自己刚加的死 i18n 键（`remoteSessions.downloadFailed`，页面实际用 `common.failed`；`systemLogs.downloadFailed` 早已存在）。
- `20be33e`（通知投递在管理端没有任何入口）：`GET /v1/admin/notification-deliveries:stats` 与 `POST ...:sweep` 服务端早已实现并成文，管理端却没有客户端方法、没有页面——用户通知通道的投递队列状态只能手工 curl。新增 `/notification-deliveries` 页与 `src/api/notifications.js`。页面把 `unacked` 放在头条位置而不是脚注：端点回 2xx 只证明字节到了，不证明有人看到，ack 循环正是为补这个缺口而存在，所以「已投递未确认」是这里唯一真正重要的数字。两种故障分别报警而不是合成一个「异常」：有行从未被确认（通道在说谎）与队列有积压且零成功投递（worker 卡住），处置动作完全不同。两个窗口相互独立这点也在 UI 上如实呈现：行可以永久保留而它们被清理进的 JSONL 归档文件不行，所以 `older_than_days` 与 `archive_retention_days` 是两个可分别留空的输入，留空即用已配置值。验证：8 条模型测试（四态齐全含零值、缺态补零、unacked 头条、两类故障分别判定、保留窗口 0=关闭、归档目录空=未设置），管理端 `npm test` 28/28 绿、生产构建绿；真实服务端 E2E 用种子数据跑通全链路——`{total:27, delivered:15, acked:2, unacked:13, needsAttention:true}`，sweep 后 `purged:18`、统计降至 9 行、`unacked` 归零，且删除前归档文件真实落盘（`notification-deliveries-2026-10-27.jsonl`，7205 字节），证明「先归档再删」。两种 sweep 路径（仅归档窗口、空对象用配置窗口）均返回 200 且不误删近期行。WebClient 271/271 未触及仍绿，`git diff --check` 干净，红绿双向（缺态补零与两类故障分别验证）。**过程中纠正一处自己的测试错误**：断言「队列卡住」时把平铺的 `pending: 9` 传给了期望嵌套 `by_status` 的模型，测试红了三轮才定位；另有一起运算符优先级缺陷（`unacked > 0 || (pending+delivering) > 0 && delivered === 0`）由测试直接抓出。
- `099a45f`（录像存储在管理端没有任何入口）：见 QD-810。`GET /v1/admin/recordings:storage` 与 `POST /v1/admin/recordings:sweep` 服务端早已实现并成文，但管理端既没有 api 方法也没有页面——磁盘用量、配额、保留窗口这三个运维数字一直在 API 里躺着，界面上看不到。会话控制台页头补上存储摘要：索引字节、磁盘占用、文件数、录像数、配额、剩余、保留、停滞判定，以及索引未覆盖的差额和「立即清理」按钮。两处刻意的设计：①索引字节与磁盘字节**都显示并点名差额**，因为索引只认得行，而缩略图和 `.part` 上传暂存文件不产生行却真实占盘，而配额正是按盘算的；②`remaining_bytes` 为 -1 是服务端「未设配额」的标记，渲染成「未设置」而不是「0 B」——后者会在没有配额的部署上报一个满盘。清理按钮按当前策略执行（sweep 不接受参数，刻意如此：运维看到的必须与定时 worker 将要做的完全一致），回报 purged/freed/orphans/uploads 与 stale 标记数。顺带收敛四份 `formatSize`：`UpdateReleasesPage`、`SystemLogsPage`、`RemoteSessionsPage` 各有一份，小数规则还互不一致（`>=10` 取整 vs 一律一位），同一个字节数在两页会显示成两个值；抽成 `storage-size.js` 共用，并修正零值语义——0 字节是「真的是空」而不是「未知」，只有缺失和非数字才是 `-`。管理端测试基建继续沿用：新增 12 条用例（含 -1 语义、零值、负值、索引差额、无配额、满配额）。验证：`npm test` 20/20 绿，管理端生产构建绿；真实服务端 E2E 实测（`RECORDING_DIR` 指向种子目录 + 1 GB 配额）：`{file_count:10, total_bytes:40895044, quota_bytes:1073741824, remaining_bytes:1032846780}`，面板渲染为 `disk:39.0 MB / quota:1.0 GB / remaining:985.0 MB`，差额 40895044 被如实报出；sweep 真实返回 200 且五个计数字段齐全。WebClient 271/271 未触及仍绿，`git diff --check` 干净，红绿双向（-1 语义与索引差额分别验证）。
- `fa52f59`（归档检索的管理端入口，以及三态误判）：见 QD-110。`GET /v1/admin/audit-archive:search` 服务端早已实现并成文，管理端却没有客户端方法、更没有入口——归档那一半的链路一直是「服务端有、界面无」。补 `searchAuditArchive()` 与 AuditLogPage 的归档卡片：日期选择、action 与自由文本筛选、截断提示。接进去时发现一个真实缺陷：某一天没有归档文件时服务端返回 404 `no archive file for ...`，而界面原来用 `/archive/i` 匹配错误文本来判断「未配置」，这条消息里同样含 archive，于是健康部署被报成需要设 `AUDIT_ARCHIVE_DIR`。改为按结构化字段判定（404=当天无文件，400 且 detail 含 not configured=未配置），其余保持 error——坏日期也是 400，不能被算成配置问题。管理端此前没有任何测试基建（无 test script、无 runner），本轮一并补上：`SignalingServer/web` 新增 `npm test`（node:test，含 localStorage stub，因为 `src/api` 在 import 期就摸 localStorage），5 条纯判断用例 + 3 条打真实服务端的端到端。验证：8/8 绿，端到端实测 404→`no_file`、坏日期→`error`、有数据日期正常返回；管理端 `npm run build` 绿；红绿双向（去掉 404 分支则端到端红）。
- `c4d5389`（E2E 固化成单一门禁）：见 QD-910。上一轮发现三个审计 E2E 自 QD-111 起一直红着，根因是套件由一串逐个文件的 npm script 组成，**没有 script 的文件就是没人跑的文件**。新增 `WebClient/scripts/run-e2e.mjs`：自动发现 `test/e2e/`，每文件独立进程（它们各自起浏览器和服务器，一个崩不该带走整轮），汇总 verdict 并点名每个失败测试。两个坑：TAP 输出在 **stdout** 而非 stderr（读错流得到空串，报告只剩「exit code 1」而不说是哪条）；`wall-soak` 显式跳过并报成 skipped 而非 passed——大部分没跑的套件不能读作绿。CI 两个 job 都改走 runner。验证：17 passed / 1 skipped / 18 total，exit 0；红绿按动机验证——破坏一个**自己没有 npm script** 的 `owner-audit.e2e.mjs`，门禁 exit 1 并点名该测试。单测 271/271。
- `12b91bd`（两个路径助手对同一路径各说一套）：见 QD-002。`parentPath` 不折叠连续分隔符而 `breadcrumbs` 折叠，且 `parentPath('C:docs')` 返回 `''` 会被读成「没有父目录」而写到文件系统根；`basename` 有镜像 bug。8 条边界测试，WebClient 271/271，18 个 E2E 全绿，红绿双向（两处分别验证）。
- `425a067`（三个审计 E2E 自 QD-111 起一直红）：direction 已从 action 名移出，三个浏览器测试还在断言 `clipboard.sync.upload` 旧名；改为断言聚合名并检查 direction 在 details 里存活。修完全套 18 个 E2E 文件全绿（此前并非如此）。
- `58f49f5`（一个 transfer id 一行，并发也成立）：见 QD-909。`LogActorOnce` 的幂等此前只是 count+insert 无约束背书，两个并发上报会各写一行；补部分唯一索引（只覆盖 actor 行、只覆盖两个 transfer action、resource_id 非空），scope 改了两轮均由测试发现。Go 9 包绿、红绿双向。
- `9c38ce6`（会话表不再无限增长）：见 QD-207。QD-207 四段里前三段完整，「清理」是空的——已结束行无上界。补 `PurgeEndedSessions` 与两个环境变量（默认关），窗口按 `ended_at` 计、绝不碰未结束行。5 条服务测试，Go 9 包绿，红绿双向（WHERE 换成 started_at 则三条红）。
- `4b91c5d`（owner 可读自己的归档）：见 QD-110。保留把过期行按天写进 JSONL，管理端能搜而 owner 不能；补 `/v1/me/audit-logs/archive`，actor 传调用方自己的 id 并忽略请求携带的，方向限定 action 拒绝（归档无 details 过滤），客户端只在在线为空时回退并给归档行加标注。Go 9 包绿、WebClient 263/263、红绿双向。
- `30e51a6`（owner 可导自己的审计）：见 QD-110。审计/归档/保留都真实存在，缺的是导出的用户侧——owner 只能看着自己的历史而拿不出去。补 `/v1/me/audit-logs:export`，actor 作用域与读取完全一致并有专门测试；顺带修 CORS 从不 expose `Content-Disposition` 导致跨源下载丢服务端文件名的缺口。Go 9 包绿、WebClient 260/260、七套 SDK 门禁绿、红绿双向。
- `359cbf7`（会话详情列出录像）：见 QD-208。服务端一直返回完整 `recordings` 数组，UI 只读 `counts.recordings` 什么都不列；改为行渲染并可播，counts 与数组不一致时以数组为准，播放 URL 解析与录像页共享一份实现。10 条模型测试 + 会话详情 E2E，WebClient 260/260，构建绿，红绿双向。
- `489e215`（历史可删可清）：见 QD-206 与 issue #18。连接历史每次尝试长一行而没有任何删除手段，清单此前因「列表存在」就把它记成满足；补 `DELETE /v1/me/connections/:id` 与 `DELETE /v1/me/connections`，owner 写进 WHERE、别人的行回 404 而非 403，清空回报真实条数。至此 issue #18 四项诉求全部满足。4 条服务测试 + 3 条 handler 测试，Go 全量 8 包绿，WebClient 250/250，七套 SDK 门禁绿，红绿双向。
- `f56d88d`（历史行可直接重连）：见 QD-206。审「连接历史与快速连接」时发现历史行只有渲染没有回去的路，而它正是用户为了回到刚才那台机器才打开的列表。访问码按 device_id 与设备列表连接取得，不落进历史行（API 本就不存，与 issue #18 的「不持久化验证码」一致）。WebClient 250/250，生产构建绿，红绿双向。
- `8977f28`（连接记录可读）：见 QD-203。清单标 `[x]` 的项按「它自己的描述承诺了什么」
  再审一遍，QD-203 的「最近连接」承诺的是一个能查的记录列表，而实现只渲染了 device_id
  和时间戳，API 返回的 `device_name` / `connect_ip` / `error_msg` 全部未消费——
  `error_msg` 尤其致命：失败行不说明原因。行派生收进 `connectionLogModel.js` 并补兜底，
  E2E 从「只数行数」扩到断言内容。WebClient 245/245，生产构建绿，红绿双向。
- `8e39a3c`（Postgres 测试底座可重跑 + `control_mode` 补进 SQL）：Postgres 测试
  共用同一个 scratch 库且没有任何东西在两轮之间重置它，所以凡是裸
  `db.AutoMigrate` 建出来的表，第一次全绿、第二次必红——这个失败本来就在树里，
  `remote_session_records` 的两个用例第二次跑挂，另外四个访问策略用例在
  `users` 已被别的测试建出来之后挂。根因是同一条路径上的第二个坑，session flag
  修不了：GORM 把唯一索引命名为 `uni_<table>_<column>`，而
  `migrations/001_init.sql` 写的是裸 `UNIQUE`、名字交给 Postgres 定为
  `<table>_<column>_key`，于是只要表是 SQL 建的，AutoMigrate 的索引比对就去删一个
  从来没存在的约束。`migrateTestSchema` 只补掉了 PrepareStmt 那一半，所以现在
  两个 helper 按「表归谁」分工：SQL 定义的表走 `applyTestMigration`，只有 GORM
  定义的表走 `migrateTestSchema`；规则写进两个 helper 的注释，并用三个测试从两个方向
  钉住边界——包括钉住裸 `AutoMigrate` 确实会失败，免得这个分工哪天又塌回一个 helper。
  改走 SQL 重放之后暴露出一个不是测试问题的真缺口：`control_mode`（QD-314 多人同时控制）
  在 Go model 里、在每一次访问策略写入里，却不在 `001_init.sql` 里。生产没发现是因为
  `RUN_AUTO_MIGRATE` 会按 model 逐个补列，于是只靠 SQL 引导的服务端会得到一张写不进去的表，
  每个访问策略调用都回 500；按该文件既有的 `ADD COLUMN IF NOT EXISTS` 惯例补上。
  剩下的裸 `AutoMigrate` 调用全是 SQLite，不触发索引命名这个坑。
  证据：全新库全量 8 个包全绿，**同一库再跑一遍仍然全绿**（这正是此前被破坏的性质）；
  WebClient 239/239；改动文件 gofmt 干净。红绿双向：把 SQL 里的 `control_mode`
  改名，策略服务与封禁 handler 立刻以真实 SQLSTATE 42703 变红，改回即绿。
- QD-113 的 CI 首跑（`64c5523` 起）与 `270b7d3` 的修复：`QUICKDESK_BUILD_TESTS=ON`
  在两个 runner 上都能配置整个 Qt 工程；首跑抓到潜伏的 `recordinglifecycle`
  只进测试 target 未进 app target（`8120ed6` 已修）；链接修好后 Windows 侧
  报 16 个测试全部 `***Not Run`，根因是 multi-config 生成器下 `ctest` 缺少
  `-C`，macOS 侧同一命令因单配置生成器而看起来正常。两个平台已补 `-C Release`，
  并新增容器内 multi-config 回归检查（三种破坏均验证过会红）。
- 本轮代码提交：`cddc0db`（QD-610 在线状态离线事件事务 outbox）、`5000d2f`、`7929550`、`b6265e9`、`5704682`、`65e19e1`（QD-908 200 路长跑门禁的超时与按钮定位修正）、`d19c850`（QD-908 多观看端 soak 门禁）、`0930b27`（QD-904 真实自托管 SignalingServer E2E）、`3e0ab72`（QD-904 设备页浏览器 E2E）、`988238e`（QD-904 屏幕墙浏览器 E2E）、`28b1972`（QD-904 真实多标签浏览器 E2E）、`2d211c1`（QD-610 event retry claim 崩溃恢复）、`c39443c`（QD-610 服务端 replay retention 可配置）、`cd730d1`（QD-610 回调操作者归属）、`5f24ff7`（QD-610 同目录多进程回调与 job 协调）、`9ae401a`（QD-610 cron 漏跑补跑）、`cac2afa`（QD-610 replay 容量可配置）、`3c50004`（MCP 节假日日历随包分发）、`e30662d`（QD-610 cron 法定节假日跳过）、`b9f414c`（QD-111 / QD-408 文件传输审计）、`71b899e`（QD-316 会话内文字聊天）、`2e09d5b`（QD-610 控制面事件触发器）、`9f7be0f`（QD-610 HMAC 结果回调）、`e8ce807`（QD-610 异步任务队列/取消）、`e518194`（QD-610 Idempotency-Key）、`3f94911`（QD-610 持久化/定时触发器）、`bca8149`（QD-610 HTTP trigger）、`de88f07`（QD-506 Web 录像分块续传）、`001a76c`（QD-510 工作流核心）、`e688869`（访问码同步重试）、`5be14da`（SPAKE2 与传输错误码）、
  `a982a10`（Windows 自动更新重启与缓存清理）、`c77dcae`（SQLite 轻量模式）、
  `64ab93a`（备份恢复与存储配额）、`37dd8aa`（Host 指标告警）、`3021ca9`
  （私密安全披露策略）、`3474203`（信令信任边界）；
  `dev` / `master` 在其后同步本轮验证；
-  当前领先数量以 `git rev-list --count origin/dev..dev` 为准；
  QD-215 主线和上游最小候选均已提交并推送。
- 上游核对：`upstream/dev` 已前进到 `5db316c`（skill-host 邻近运行时修复）；
  fork `dev` 已合并该基线；当前上游 open issues 10 个、
  open PR 5 个（#22、#23、#25、#26、#27）；#24 已合并为 `5db316c`。
- 上游维护：issue #21 已由维护者回复“持续维护、暂缓更新、重点偏 MCP、欢迎从
  `dev` 切分支贡献、Remoting 改动由维护者处理”；不能再把“暂缓”理解成停止维护。
- 上游 PR：#22、#23、#25、#27 为 OPEN / `UNSTABLE`，#26 为 OPEN / `CLEAN`；
  多数仍没有 reported checks，不能把 MERGEABLE 或 CLEAN 写成 CI 已通过。
- 清单统计：119 个唯一 QD ID；91 个 `[x]`、19 个 `[~]`、5 个 `[ ]`、
  1 个 `[!]`、2 个 `[>]`、1 个 `[-]`。本轮 QD-216 六条验收全部完成后转 `[x]`，
  统计按第 3 节与第 4 节起各表分段解析（两处列序不同）重新点数。上一轮手写的 84/20/9 与实际标记不符，
  已按表格逐行重新统计更正；本轮 QD-113 转 `[x]` 后再次按同一方式统计
  （第 3 节表格列序与后续各节不同，须分段解析，否则会把功能名读成状态）。
  本轮再次重数：上一行的 118/89/20/5/1/2/1 与实际标记不符，已更正为
  119/91/19/5/1/2/1。判据是「按 QD ID 去重后取首次出现」——QD-204、QD-318、
  QD-319、QD-609 四个 ID 各出现在两张表里，四处状态各自一致，按行计数会把这
  四个多数一遍。
- 上游待办已全部映射到本清单：11 个 open issue、8 个 open PR、3 个未完成 Roadmap
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
- QD-111 / QD-408：TDD RED 覆盖 handler 缺失、重复上报和非法 direction；
  `go test ./...`、`go vet ./...`、WebClient 66/66、生产构建、Qt CTest 15/15 均通过；
  最小 SQLite 真服务验证同 transfer 两次 204、非法方向 400，数据库仅一行
  `file.transfer.upload`，中文文件名和字节数完整；QuickDesk 源码编译到最终链接，
  仍受既有 `AGL` framework 缺失限制。
- QD-907：TDD 覆盖可配置 auth_ok 超时、超时原始错误保留、认证前 XML/JSON payload
  丢弃和 `send()` 拒绝；WebClient 68/68 与生产构建通过，JS 379.50 kB、
  gzip 121.42 kB。
- QD-907 访问码错误归类：访问策略拒绝、访问码错误、离线、限流、审批等待、控制器签名
  和未知错误均有独立映射；WebClient 72/72 与生产构建通过，JS 380.30 kB、gzip 121.82 kB。
- QD-907 上传重试耗尽：连续三次 transport error 后抛最后一个错误，尝试 offset 均为 0；
  最终 503 保留 HTTP 状态和 detail；WebClient 74/74 通过。
- QD-610 数据库事务 event outbox：事务回滚同时撤销业务行和事件行；正常 drain 只发布一次；
  live claim 不被抢、过期 claim 可恢复、Redis 发布失败保留 pending 并退避；
  `scheduled_task.failed` 与任务终态同事务提交。SignalingServer `go test ./...`、
  `go vet ./...` 和 EventBus/Outbox/ScheduledTask race 测试通过。
- QD-610 设备访问事件 outbox：访问成功/拒绝日志与事件同事务；模拟 outbox 失败时
  `device_access_logs` 回滚为 0；SignalingServer 全量测试、vet、race 和构建通过。
- QD-610 访问策略 outbox：策略写入和 `device.access_policy.changed` 同事务，策略
  event/outbox 失败回滚为 0；SignalingServer 全量测试、vet、race 和构建通过。
- QD-610 远端会话 outbox：Open/Close 同一事务写会话和 started/ended 事件；orphan
  sweep 关闭会话时生成 ended 事件；outbox 失败回滚会话创建。SignalingServer 全量测试、
  vet、race 和构建通过。
- QD-610 录像生命周期 outbox：create/update/delete 三类事件依序生成；outbox 失败回滚
  录像创建；artifact/thumbnail 更新在同一事务内只生成一次最终事件。SignalingServer
  全量测试、vet、race 和构建通过。
- QD-610 访问审批生命周期 outbox：requested/decided 与审批行同事务；outbox 失败分别
  回滚创建和决定；重复决定不产生第二条事件。SignalingServer 全量测试、vet、race 和
  构建通过；上游 `codex/upstream-event-outbox@d9e231a` 同步通过全包测试、vet、race、
  构建和单提交 diff check。
- QD-610 共享访问码生命周期 outbox：changed/used 与 grant 行同事务；outbox 失败分别
  回滚创建、核销和撤销；重复撤销不产生第二条事件。SignalingServer 全量测试、vet、
  race 和构建通过；上游 `codex/upstream-event-outbox@16c00d4` 同步通过全包测试、
  vet、race、构建和单提交 diff check。
- QD-610 离线告警 outbox：首次/重复告警的 episode claim 与事件同事务；outbox 失败时
  `alerted_at` 保持为空，下一轮可安全重试。SignalingServer 全量测试、vet、race 和
  构建通过；上游 `codex/upstream-event-outbox@36af1ca` 同步通过全包测试、vet、race、
  构建和单提交 diff check。
- QD-610 指标阈值告警 outbox：CPU/memory/disk episode claim 与 `device.metric.alert`
  同事务；outbox 失败时对应 `*_alerted_at` 保持为空。SignalingServer 全量测试、vet、
  race 和构建通过；该提交依赖 QD-604，暂留 fork，不追加到缺少指标服务的 PR #25 堆叠
  候选。
- QD-610 设备控制面变更 outbox：bind、access-code、clear-session、name/remark 和
  unbind 六类事件与最终设备状态同事务；outbox 失败会回滚 bind/unbind。SignalingServer
  全量测试、vet、race 和构建通过；上游 `codex/upstream-event-outbox@dda240d` 同步通过，
  候选只保留 outbox 变更，不带后续分组清理或指标功能。
- QD-610 收藏设备 outbox：add/update/delete 三类事件与收藏行同事务；outbox 失败回滚
  新增收藏。SignalingServer 全量测试、vet、race 和构建通过；上游
  `codex/upstream-event-outbox@b14601b` 同步通过，候选不携带分组或指标功能。
- QD-610 管理端设备 outbox：force-unbind、硬删除、批量删除和 secret rotation 的
  业务行与 `device.ownership.lost` / `device.unbound` / `device.secret.rotated`
  同事务；模拟 outbox 失败时设备归属、删除和密钥哈希全部回滚。SignalingServer
  全包测试、vet、race 和构建通过；上游 `codex/upstream-event-outbox@fa18755`
  同步通过，候选不携带 fork-only 的分组清理和指标功能。
- QD-610 管理员来源封禁 outbox：跨账号 `access-bans` 和 `access-bans:remove`
  不再在 handler 里二次 publish，改为策略事务内 enqueue；新增 SQLite HTTP handler
  测试先 RED 为 0 条 outbox，再 GREEN 为 ban/unban 两条策略事件。SignalingServer
  全包测试、vet、handler/service race 和构建通过；因基线缺少访问策略功能，该提交
  只留在 fork。
- QD-610 TURN/STUN 设置 outbox：settings 创建/更新与 `turn.config.changed` 同事务，
  cache 只在提交后更新；模拟 outbox 失败时数据库不产生设置行、缓存版本不变。
  SignalingServer 全包测试、vet、race 和构建通过；上游
  `codex/upstream-event-outbox@4a3566f` 同步通过。
- QD-610 第十六至十八阶段：cron trigger 新增 `skip_holidays`，按 IANA 时区本地日期跳过
  内置中国法定放假日，调休上班日仍执行；支持 `QUICKDESK_HOLIDAY_CALENDAR_DIR`
  追加后续年份；interval/event-only 使用该标志返回 400。默认 replay buffer 提升到
  2048 并支持 1~10000 配置；`backfill_missed` 可在恢复后逐条补跑漏过 cron 槽位。
  MCP 61/61、Clippy
  `-D warnings` 和 release 构建通过。
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
- QD-111 / QD-408 上游候选：`codex/upstream-qd-111-file-transfer-audit` = `c57a2f4`，
  基于 PR #25 的 `feat/signaling-control-plane@8998466`，共 44 个文件。候选只含
  文件传输审计的 SignalingServer、OpenAPI、WebClient、Qt 接线、contract smoke 和
  PR #25 基线已有的 TypeScript/Go/Python/Java 四套 SDK；不携带 fork 清单、品牌、
  C#/PHP/Kotlin SDK 或 device-groups 等相邻功能。候选自身 Go 测试/vet、四套 SDK
  freshness、Go/Python/Java/TypeScript 编译、WebClient 单测和生产构建、管理端生产
  构建均通过；QuickDesk 源码编译经过新增的 `CloudDeviceManager` 和 `MainController`，
  最终链接仍因既有预编译 Remoting 依赖缺失 `AGL` framework 失败。
- QD-610 第一阶段功能提交：`bca8149`。`quickdesk-mcp --transport http` 新增
  `POST /v1/workflows/{workflow_id}/run` 和 `GET /v1/workflow-runs/{run_id}`，使用独立
  `QUICKDESK_TRIGGER_TOKEN`，执行前检查设备白名单，继续经过 trust/emergency 层，并返回
  已持久化的运行报告；进程内限制单条工作流并发，忙碌返回 429。fake QuickDesk WS 的
  HTTP→replay→report E2E、错误 token、设备白名单和 429 测试均通过，MCP 全量 23/23、
  Clippy 零告警和 release 构建通过。定时触发、控制面事件订阅、幂等键、出站回调和委托
  权限仍未完成，QD-610 保持 `[~]`。
- QD-610 上游 stacked 候选：`codex/upstream-qd-610-business-trigger` = `5fe849a`，
  基于 `codex/upstream-qd-510-workflow@0a9ecd5`。候选现已叠加 HTTP trigger、持久化
  interval/IANA 时区 cron trigger、Idempotency-Key、异步 job 队列/取消/重启恢复、
  HMAC 持久 outbox/重试/死信/重放/单条与批量 ack、Qt 白名单事件桥、MCP event trigger
  和持久事件重连补发；再叠加 `b0ef926` cron 法定节假日跳过、`340b463`
  MCP 内置日历资源、`908696f` 可配置 replay 容量、`0b4bbb9` cron 漏跑补跑，以及
  `a507aa6` 的同目录多进程回调/job claim 协调、`f90df36` 的回调操作者归属；
  通用文档收口为 `5fe849a`；
  候选自身 59/59 测试、Clippy、QuickDesk C++ 编译到既有 `AGL` 链接点、release 构建和
  `git diff --check upstream/dev...HEAD` 通过。
  候选保留通用 MCP 代码和契约文档，不含 fork 清单、QD-506 或控制面业务记录。
- QD-610 服务端 event reliability 独立候选：
  `codex/upstream-event-replay-retention` = `b5da22f`，直接基于 PR #25 的
  `feat/signaling-control-plane@8998466`；只包含 EventBus retention、retry claim、
  heartbeat/processing 恢复、配置、文档和回归测试，候选 Go 全包测试、`go vet`、EventBus race 与
  `git diff --check 8998466...HEAD` 均通过。
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
- QD-610 第十三阶段功能提交：`47d5e19`。新增 `POST /v1/callback-deliveries/ack`，
  接受去重后的 1~200 个 delivery id；delivered/dead 幂等确认，pending/retry 放入
  `skipped`，不存在或非法 UUID 放入 `not_found`，单个非终态记录不会导致整批失败。
  单条与批量 ack 共用 `acknowledged_at`。MCP 全量 54/54、Clippy、release 构建和
  diff check 通过。操作者身份、法定节假日、事件停机补发、多进程共享 worker 和委托
  权限仍未完成，QD-610 保持 `[~]`。
- QD-610 第十四阶段功能提交：`af8653f`。QuickDesk WebSocket API 新增最近 256 条
  白名单事件的有界内存 buffer，按事件 `id` 或 `server_rev` 去重；MCP auth 增加
  `last_control_plane_rev`，重连时发送 `since_server_rev`，认证后补发更大的 revision。
  该闭环覆盖 MCP 断开/重启但 QuickDesk 仍在运行的场景，trigger 继续按事件 ID 幂等。
  MCP 全量 55/55、Qt CTest 15/15、Clippy、release 构建和 diff check 通过；Qt 完整
  源码编译到链接阶段，仍因预编译 Remoting 依赖 `AGL` framework 不存在而失败。QuickDesk
  自身重启后的持久 replay、超过 256 条积压、跨实例事件 outbox、法定节假日、多进程
  worker、确认操作者归属和委托权限仍未完成，QD-610 保持 `[~]`。
- QD-610 第十五阶段功能提交：`5a4ff4a`。事件 replay buffer 新增按用户 ID 分区的
  `control-plane-event-replay.json` 原子持久化；`QSaveFile` 防止半写文件，登录态切换
  会重新加载对应 scope，其他用户的事件不会被当前账号回放。这样 MCP 和 QuickDesk
  任一短暂重启后都能从本地最近 256 条事件补发。Qt CTest 15/15、MCP 55/55、Clippy、
  release 构建和 diff check 通过；Qt 完整源码编译到链接阶段仍受既有 `AGL` 依赖限制。
  超过 256 条、跨实例持久 outbox、服务端 retention、法定节假日、多进程 worker、
  确认操作者归属和委托权限仍未完成，QD-610 保持 `[~]`。
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
| 上游同步 | PR #22 自动审批线程亲和性 | `Upstream` / QD-113 | 已完成：上游 PR 已覆盖 issue #17，`5969ae3` 已合并进 `upstream/master`，fork `3d44dfc` 已回灌且与上游无差异；本 fork 未重复实现 | 回归测试已在 Windows CI 实跑通过（17/17，含 `trust_handler_thread_affinity`），QD-113 转 `[x]`；阻塞它的三层原因（测试不在构建图、源文件只进测试 target、`ctest` 缺 `-C`）都已修掉并各有看守脚本 |
| 上游同步 | PR #23 会话录像与回放 | `Upstream` / QD-501~507 | 已覆盖服务端索引、Web/Qt 手动录制、预览、生命周期；Host 权威录制不在其中 | 不重复提交；合并后回灌 fork，再单独推进 Host 录制 |
| 上游同步 | PR #24 skill-host Windows 运行库 | `Upstream` / QD-402、QD-609 | 修复技能进程找不到 MSVC runtime；与 fork 同向 | 合并后同步，不维护第二套 PATH/stderr 逻辑 |
| 上游同步 | PR #25 自托管控制面、OpenAPI、SDK | `Upstream` / QD-101~110、201~211、501、601~608、801~808 | 当前控制面总集；维护者可接受继续拆 PR | 等审查；若维护者要求，按 access / alerting / recording / operations / update 拆 |
| 已完成 | QD-708 Qt 通知规则页 | `Fork` / `Upstream` 通用能力 | 通知通道、静默时段、设备离线告警、投递重试/确认和实时刷新已完成，提交 `dbd828f` | PR #25 合并后拆成通用 Qt 管理面候选并提交上游 |
| 已完成 | QD-508 Qt 录像回放页 | `Legacy` / `Upstream` PR #23 | Qt 录像列表、分页、缩略图、应用内播放、外部打开和删除已完成，提交 `74c7022` | PR #23 合并后作为通用 Qt 管理面候选提交上游 |
| 已完成 | QD-204 地址簿与设备分组用户入口 | `Legacy` / `Upstream` Roadmap | owner-scoped API、四套 SDK、WebClient 和 Qt 已完成；真实 service smoke 388 条断言通过 | 本地候选已准备：API `codex/upstream-qd-204-api@d2321b9`，客户端 `codex/upstream-qd-204-client@b259d03`；等远端写入恢复后建 PR |
| 已完成 / 上游候选 | QD-604 指标型告警 | `Legacy` / `Fork` / `Upstream-Open` | Host 上报、服务端 episode、通知 outbox、Web/Qt 和 SDK 已由 `37dd8aa` 完整闭环；服务端候选 `codex/upstream-qd-604-metrics@0ea9e7c` 已基于 QD-006 `7758456` 独立验证 | 推送服务端候选并作为 `#25 -> SQLite -> metrics` stacked PR；Web/Qt 待 #25 客户端基线确认后单独拆 PR |
| 本地收口 | QD-607 计划任务执行动作 | `Fork` / `Legacy` | 服务端调度、Web/Qt 管理面、重试已完成；新增 `revoke_all_remote_sessions`，可按执行时设备归属批量关闭活动远控会话（最多 200 台）；纯服务端候选 `codex/upstream-qd-607-batch-revoke@badf990` 已独立验证 | 「定时连接」已由 `connect_prompt` 提示 + 四层消费收口；本轮再补**编辑已有任务**（`PATCH /v1/me/scheduled-tasks/:id`，删掉重建会连 outcome 一起丢掉）；本轮再补广义批量 agent 指令的执行与回执（`batch_skill_exec`，并发 + 每台一份回执 + 上限 50 台）；纯服务端候选 `codex/upstream-qd-607-batch-revoke@badf990` 已独立验证，编辑动作随 #25 基线拆分回贡 |
| 本地收口 | QD-005 / QD-808 自动更新真机验证 | `Fork` / `Legacy` / `Upstream` PR #25 | Windows 已完成 4.2.8.0→4.2.9.0 自动下载、校验、静默安装、自动重启、版本和服务闭环，提交 `a982a10`；macOS 已实现 DMG 自动替换和失败回退，并通过 ad-hoc 真实 DMG E2E | 取得 Apple Developer ID 和公证凭据，执行 hardened runtime 签名、notarytool、staple、Gatekeeper 与真实升级回归；上游候选仍应在 PR #25/更新通道基线确认后重做最小增量 |
| 已完成 | QD-006 / QD-809 SQLite 轻量服务端 | `Fork` / `Upstream-Open` | fork 提交 `c77dcae`；候选 `codex/upstream-qd-006-sqlite@7758456` 已在 PR #25 基线上通过完整测试、race、vet、构建和 E2E | PR #25 合并后 rebase 到 `upstream/dev` 并创建正式 PR；明确单实例和易失 Redis 边界，不和多实例生产配置混在同一 PR |
| 已完成 | QD-810 备份恢复与存储配额 | `Fork` / `Upstream-Open` | 提交 `64ab93a`；SQLite 一致性归档、恢复校验/回滚、PostgreSQL 运维文档、录像与安装包总量配额、两个管理存储接口和升级通道页状态均完成 | PR #25 合并后作为自托管运维增量拆最小候选；备份 CLI 与配额边界应独立于控制面功能 PR |
| 上游候选 | QD-003 多标签控制与音频抢占 | `Legacy` / `Upstream` Roadmap 补强 | 候选 `codex/upstream-cross-tab-coordination` = `93eee37`，仅 WebClient 改动；真实 Chrome 三标签 E2E 已通过 | 远端写入恢复后推送并建 PR |
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
| 后续增强 | QD-610 业务工作流 | `Legacy` / `Fork` / `Upstream-Open` | QD-510 已完成录制/回放/报告；`bca8149` 补 HTTP trigger；`3f94911` 补持久化 CRUD、固定 interval worker 与运行结果回写；`e518194` 补 Idempotency-Key；`e8ce807` 补异步队列、状态查询和取消；`9f7be0f` 补 HMAC 出站回调；`2e09d5b` 补白名单控制面事件订阅、过滤和幂等 claim；`1176a92`~`a94bba1` 补持久 outbox、死信、重放和 ack；`d3dc4cf` 补 queued job 重启恢复；`3c31d9a` 补 UTC cron；`20d02cc` 补 IANA 时区和 DST；`47d5e19` 补批量 ack；`af8653f` 补 MCP 短期事件重连；`5a4ff4a` 补按用户持久化事件 replay；本轮补法定节假日跳过、可配置 replay 容量、cron 漏跑补跑、`5f24ff7` 的同目录多进程回调/job claim 协调、`cd730d1` 的回调操作者归属、`c39443c` 的服务端 replay retention 配置和 `2d211c1` 的 retry claim 崩溃恢复；数据库事务 event outbox 及 `scheduled_task.failed`、设备访问、策略变更、远端会话、录像、访问审批、共享访问码、离线告警、指标阈值告警、设备控制面、收藏设备、在线状态离线事件和账号会话撤销事件共 14 类迁移已完成；`2848be5` 补公平调度：异步工作流队列改为按提交方轮转，每提交方每轮至多一个，并按提交方和总量设上限；`97b8619` 再补 Owner 委托权限：`DelegationGrant` 按 subject 限定 workflow 与设备范围、步骤上限和禁权工具，可撤销、可过期、落盘持久化，`QUICKDESK_DELEGATION_REQUIRED` 控制是否强制，grant 管理走 trigger API；`QUICKDESK_WORKFLOW_CLAIM_DB` 补跨机器分布式队列：此前 claim 落在本机文件系统上，`create_new` 只对本地原子，另一台机器上的 worker 根本看不到那个目录——两个 worker 会各自开自己的 claim 文件、都认为自己持有这个 run、一起驱动同一台远程桌面；现在 claim 放进共享 SQLite，一条 `INSERT` 就是互斥，心跳列让崩溃 worker 的 claim 可被接管（ takeover 的 `UPDATE` 带心跳条件，两个 worker 同时看到陈旧行也不会双赢），`ClaimLease` 在 run 执行期间持续心跳、结束时释放，未配置时行为与之前完全一致 | 继续做不依赖外部凭据的业务编排 |
| 质量收口 | QD-906、QD-907、QD-908、QD-909 | `Legacy` / `Fork` | 真机矩阵、异常路径和 2h+ 长稳仍不完整；文件传输审计已完成，图片/系统级剪贴板边界仍按 QD-406/407 跟踪 | 每完成一个主功能就补对应边界，不集中到最后一次性补测 |
| 明确暂缓 | QD-318 Linux、QD-319 原生移动端 | `Upstream` Roadmap | 上游已列计划，当前 fork 不做实现承诺 | 保持接口和构建兼容，等上游方向明确后接 |
| 明确不做 | RDP、重写 WebRTC/Remoting、多租户/SSO/计费、通用插件市场 | `Fork` 决策 | 与当前“复用成熟远控内核 + 补业务控制面”的路线冲突或收益不足 | 不进入待办池，除非用户重新定义产品目标 |

### 3.2 合并后的执行队列

这张表把 fork 自己的未完成项与上游 11 个 open issue、8 个 open PR、3 个未完成
Roadmap 项放到同一条实施路径中。详细追溯仍以第 16 节为准。

| 顺序 | 队列 | 包含工作 | 为什么现在放这里 | 完成条件 |
|---|---|---|---|---|
| 1 | Fork 真机收口 | QD-005 / QD-808 macOS 签名与公证、QD-215 Windows Qt 插件、QD-214 Windows Host 用户上下文、QD-607 定时连接边界 | Windows 自动更新已闭环，macOS 自动替换和回退也已通过 ad-hoc 真实 DMG E2E；当前只缺正式签名、公证链和公证包升级回归 | 使用 Developer ID 完成 hardened runtime 签名、notarytool、staple、Gatekeeper 评估和 4.2.8→4.2.9 真实替换；Windows Host 上传和 Qt 插件问题有真实证据 |
| 2 | 可独立回贡上游 | QD-002/QD-403 Web 文件管理、QD-003、QD-114 第一阶段、QD-212、QD-213、QD-214 SPAKE2/诊断、QD-215 Windows 发布校验、QD-316 会话聊天、QD-510 MCP 工作流、QD-911 私密披露策略 | 改动边界清楚，可从 `upstream/dev` 独立审查；Web 文件管理和聊天只依赖现有 `qd-skill-bridge`，QD-510 候选不含 fork 业务和清单，QD-911 候选只含根目录 `SECURITY.md` | 每个主题一个基于 `upstream/dev` 的最小分支；Web 文件管理拆成 `file-ops` 与 WebClient 两个 PR；远端写入恢复后推送 |
| 3 | 等 PR #25 基线 | QD-204 API/客户端、QD-607 服务端、QD-108 / QD-508 / QD-706 / QD-708 管理面、QD-005 客户端 | 这些能力依赖控制面、OpenAPI、录像或管理页基线；直接塞进 #25 会变成巨型 PR | 维护者确认 #25 合并或 stacked base；再按服务端/Web/Qt/更新拆 PR |
| 4 | 跟踪上游已有 PR | PR #22 自动审批、#23 录像、#24 skill-host 运行库、#25 控制面 | 上游已有实现，重复开发只会制造分叉 | PR 合并后同步 `upstream/dev`，重跑受影响测试；删除重复本地实现 |
| 5 | 提 Remoting 需求 | QD-004 Host 权威录制、QD-007 图片剪贴板、QD-114 真实 CertVerifier、QD-214 Windows 用户上下文 | 必须改 Chromium Remoting 或宿主生命周期，维护者已明确这类改动由他处理 | 每项提交协议草案、权限边界、兼容性和验收用例；不在 fork 伪造第二套媒体链路 |
| 6 | fork 自主增强 | QD-610 业务集成、QD-511 智能回放、QD-904 / QD-907 / QD-908 / QD-909 | QD-510、HTTP trigger、持久化 interval/IANA 时区 cron trigger、法定节假日跳过、cron 漏跑补跑、Idempotency-Key、异步队列/取消、HMAC 回调、持久 outbox、死信查询/重放、单条与批量 ack、操作者归属、queued job 重启恢复、同目录多进程 claim 协调、QuickDesk 本机持久 replay、服务端 Redis replay retention 和跨 worker retry claim 恢复已落地；数据库事务 event outbox 已覆盖 14 类关键事件，公平调度、Owner 委托权限和跨机器分布式队列（`93a0d13`，`QUICKDESK_WORKFLOW_PEER_JOB_DIRS` + `QUICKDESK_WORKFLOW_CLAIM_DB`）均已落地；QD-610 至此在 fork 侧收口 | 有明确用户入口、失败路径和对应自动化或真机证据 |
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
| U3 交 Remoting | QD-004、QD-007、QD-114 真实 CertVerifier、QD-214 Windows 用户上下文 | 写协议草案、权限边界、兼容性和验收用例，先向上游提需求 | `UP-Remoting` | 不重写媒体/输入链路，不在 fork 造第二套 RDP 或伪协议；QD-111/QD-408 已走控制端事后上报，不阻塞 Remoting |
| U4 fork 产品层 | 业务系统专属 provider、委托权限策略、运营页面、品牌和部署配置；QD-610 中依赖具体业务凭据的部分 | 在 fork 主干完成产品化，通用底层抽成 U1/U2 的 PR | `Fork-only` | 不把私有业务规则伪装成上游通用能力 |
| U5 质量底座 | QD-904、QD-906、QD-907、QD-908、QD-909、QD-910 | 每完成一个主功能同步补测试、真机或长稳证据；上游通用测试随后回贡 | `UP-Ready` 或 `Fork-only` | 不集中到最后补测，不用单次成功代替稳定性 |
| U6 跟随上游 | QD-318 Linux、QD-319 原生移动端 | 保持接口和构建兼容，等待上游路线明确 | `Upstream-Open` | 不提前承诺实现，不把 Web 触摸或跨平台构建冒充原生客户端 |

“现在要做”的 fork 工作按以下顺序执行：

1. **QD-610 后续阶段**：`Idempotency-Key`、异步队列/取消、HMAC 回调、控制面事件订阅、
   UTC cron、IANA 时区/DST、法定节假日跳过、cron 漏跑补跑、批量 ack、短期事件重连、按用户持久 replay
   和服务端可配置 replay retention
   已完成；同目录多进程回调/job claim、回调操作者归属、数据库事务 event outbox
   基础设施、`scheduled_task.failed`、设备访问 granted/denied、策略变更、远端会话
   started/ended、录像 created/updated/deleted、访问审批、共享访问码、离线告警、
   指标阈值告警、设备控制面变更、收藏设备、在线状态离线事件迁移也已落地；
   `314c73b` + `a1d34c5` 进一步把账号会话撤销事件的全部 8 个生产者（改密码、短信改密、
   refresh 家族断裂、管理端改密/删用户/强制下线/批量禁用/批量删除、本人登出、按 id
   删会话、计划任务全局登出）迁入事务 outbox。撤销本身在 Redis、事件在数据库，
   两者无法共享事务，因此这些迁移的口径统一为“只保证发布不丢，不宣称原子提交”。
   `2848be5` 再补公平调度：异步工作流的 FIFO 队列换成按提交方轮转的 `FairQueue`，
   单个提交方无法把别人的 run 压在自己的积压后面，队列上限按提交方和总量分别设置；
   配置了 operator token 时各操作者独占一条 lane，未配置时共用一条（与改动前一致）。
   `97b8619` 再补 Owner 委托权限：HTTP 触发器此前只有一个共享 bearer token，任何持有者都能跑任意
   workflow、打任意设备；现在 Owner 可签发 `DelegationGrant`，按 subject 限定 workflow 与设备范围、
   步骤上限和禁权工具，可撤销可过期并落盘，`QUICKDESK_DELEGATION_REQUIRED` 默认关闭以保持既有行为，
   启用后无 grant 的调用方在碰到 workflow 之前就拿到 403；grant id 随 run 进入回调和审计。
   本轮 `93a0d13` 完成跨机器分布式队列：worker 通过
   `QUICKDESK_WORKFLOW_PEER_JOB_DIRS` 监视 peer 提交目录，认领、`get` 和重启恢复都跨目录
   解析；必须与 `QUICKDESK_WORKFLOW_CLAIM_DB` 成对设置，后者才是防重复执行的机制。
   因上游 `quickdesk-mcp` 没有 `workflow_job.rs` 等 fork 自有模块，该项无法摘成最小 PR，
   归入 `Fork-only`。通用触发链路能从当前的 MCP trigger 候选继续 stacked 回贡，
   后几项若绑定 fork 业务数据库则留在 `Fork-only`。
2. **QD-511 智能回放**：预录屏幕断言与状态不匹配暂停已完成；`screen_contains` 已由 Host
   PP-OCRv4 `text_present` 实现，OCR 关键词不是缺口；`1ad0b85` 再补人工接管：park 的 run 会在报告
   旁边留下 resume point（停在哪个步骤、原因、当时的 overrides、该 run 自己的步骤总数），
   `GET /v1/workflow-runs/resume-point` 交给 operator，`DELETE` 在接管后清除，响应会说明恢复后
   是否还有步骤可跑；本轮再纠错阻断性契约缺陷：MCP 原先发送 Host 不认识的
   `textContains` / `frameHash` 条件类型，未知类型一律 `passed=false`，带断言的步骤
   永远 park；现改为 `screen_contains` 发 Host 的 `text_present`、`screen_hash` 经
   `getUiState` 取 `ocr.frameHash` 本地比对，观测值改读 Host 真正报告的
   `results[0].actual`；剩余自动 AI 失败解释/决策，通用部分在 QD-510 候选合并后回贡。
3. **QD-907 异常路径**：按“协议错误、权限拒绝、重连耗尽、上传失败、设备离线”补
   可重复的自动化或真机证据，避免把 Windows 上传根因误判成 WebClient 协议错误。
4. **QD-904 / QD-908**：QD-904 已闭环，重连、文件管理器、多标签、屏幕墙、设备页和
   真实自托管 SignalingServer 均有浏览器 E2E；QD-908 的 200 路 7200 秒正式跑已于
   2026-10-24 跑完（headroom 从 1 小时提到 2 小时后不再被 `node:test` 掐断），
   并已加按采样打点的时间序列。劣化形状确定：FPS 在 267~681 秒**阶跃**崩塌而非渐进，
   DOM 节点数在 267 秒见顶后不再增长（不是原因），heap 增长与 FPS 崩塌是两条独立劣化
   （FPS 归零后 heap 仍线性涨，分配源不依赖主线程帧循环）。不调阈值，下一阶段在
   267~681 秒窗口内加密采样抓住阶跃时刻，并查周期性分配源。
5. **Remoting 需求单**：QD-004（已形成 `docs/dev/Host权威录制Remoting需求.md`）、QD-007、QD-114 根因和 QD-214 用户上下文
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
| QD-105 | 设备配额与用户设备隔离 | `[x]` | 设备归属和 owner-scoped API；**本轮 `b68a1b0` 补上配额强制执行**：此前 `users.device_count` 只有列、管理端可改、`/v1/me` 会回显，但全仓库没有任何一处读它做判断——`device_count = 1` 的用户可以无限绑定设备，管理端那个数字一直是装饰品。判定放在 `BindToUserWithEvents` 事务内、device 行 `FOR UPDATE` 之后、所有权翻转之前：两个并发绑定在同一份语句快照里数配额，谁都 race 不过最后一个空位；被拒则连翻转带事件一起回滚，不会有事件宣布一台不属于用户的机器。三条边界决定它是能用还是 merely 严格：**配额 0 表示未配置而非零台**（每个新账号的默认值，读成硬零会把所有新用户锁在自己的第一台机器外）、**重复绑定已拥有的设备不重复计数**（Qt 被控端每次重连都重新 bind，把幂等重试算进去会让配额满之后连重试都失败）、**接管不计入新属主配额**（该账号绑定的机器总数没变）。拒绝映射为 `403 DEVICE_QUOTA_EXCEEDED`，而不是原先落入的内部错误分支：要行动的是属主，"解绑一台"可执行，"设备不存在"不可执行。Qt 侧 `CloudDeviceManager` 解析它本来就会读的 RFC 7807 体并发出 `deviceBindQuotaExceeded`，设备页把它变成带服务端原文的 toast——被控端开机自动绑定，这是唯一一个能让机器"已登录、在线、却不在属主列表里"且屏幕上没有任何解释的服务端拒绝；WebClient 设备列表显示"已绑定 2 / 上限 5 台"并在满额时告警，配额此前在写入 localStorage 时被丢掉，客户端从来不可能显示它。`POST /v1/me/devices` 此前根本不在 OpenAPI 契约里，本轮补上并附新拒绝分支，七套生成客户端已重新生成。验证：7 个 service 测试（真实 SQLite 文件：配额 0 时首个绑定成功、满额拒绝、幂等重绑、解绑腾位、接管、管理端中途提额、outbox 回滚），三条强制拒绝的测试做过红绿双向；1 个 handler 测试断言 403 code、非空 detail 和被拒设备保持无主，摘掉映射即以用户实际拿到的 `500 INTERNAL_ERROR` 形状失败；5 个 WebClient 单测，改掉"0 表示无限制"即红；2 个 Playwright E2E 用真实 Chrome 驱动真实 `DevicesPage`，满额告警出现、无限制账号既不见计数也不见告警；干净 scratch Postgres 上 `go test ./...` 全绿、`go vet` 干净、sdk/go vet+build 干净、`npm test` 239/239、生产构建绿、七套 `check-generated-sdk.sh` 全新鲜。新增 Qt 测试已接进 CMake 但本地无 Qt 工具链未跑，首跑在 Windows 与 macOS CI job。 |
| QD-106 | 设备访问策略 | `[x]` | 开关、IP/CIDR、允许时段、例外日 |
| QD-107 | 临时访客授权 | `[x]` | 限时、限次、可撤销、IP/client 绑定 |
| QD-108 | 二次访问审批 | `[x]` | 服务端、WebClient 和 Qt 均已完成；Qt 提交 `7c23f13` 覆盖入口、决定、汇总、冲突处理和实时消费 |
| QD-109 | 来源封禁与解封 | `[~]` | 服务端、WebClient、Qt、管理端和 SDK 已完成；真实 Windows 踢人回归待补 |
| QD-110 | 审计、归档、导出、保留策略 | `[x]` | 管理端和用户侧事件已覆盖。本轮 `30e51a6` 补上四项里唯独缺的「导出」的用户侧半边：审计、归档、保留都是真的（保留 sweep 按定时器跑，带 per-actor / per-device / per-action 上限），管理端也早就导出了这些行，但 owner 只能在自己屏幕上看着自己的传输历史——想交给别人或留一份在服务器之外，就得找管理员要自己的数据。新增 `GET /v1/me/audit-logs:export`，CSV 或 JSON。做成独立方法而不是读取加个 format 参数，因为导出不分页，那边的 bug 不会躲在页大小后面：读取校验的每个过滤器这里都重新校验，actor id 也按读取完全一样的方式施加，并有一条专门测这个的用例——把 actor 作用域去掉，它就红着把别人账号的传输记录摆进文件里。上限是刻意的：忙账号的无界导出是服务端内存尖峰加一个谁也打不开的文件，管理端导出同样有上限。顺带修掉一个管理端导出一直将就着的 CORS 缺口：中间件允许跨源 API 调用却从不 expose `Content-Disposition`，跨源客户端读不到服务端精心设好的文件名，只好静默退回自己起的名字——下载是成功的所以没人发现，丢的是服务端时间戳，而那是区分相隔几分钟两次导出的唯一依据。已 expose，两个方向都有测试。验证：5 条 handler 测试 + 2 条中间件测试，Go 全量 9 包绿，WebClient 260/260 且 E2E 真点按钮并断言下载文件带服务端给的名字，七套 SDK 门禁绿，改动文件 gofmt 干净，红绿双向（作用域与 CORS 头各自验证）。`4b91c5d` 再补「归档」的用户侧半边：保留会把过期行按天追加进 JSONL 文件再删除，管理端能搜这些文件而 owner 不能——保留一跑，唯一还能读某行的人就成了管理员，与导出那个不对称是同一种，只是更深一层。新增 `GET /v1/me/audit-logs/archive`；`QueryArchive` 本就支持按 `ActorUserID` 过滤，所以作用域靠传调用方自己的 id 并**忽略请求携带的**那个来实现——忽略而不是拒绝，因为该参数是管理端查询用来收窄 actor 的，回「未知参数」等于告诉探测者它存在。方向限定的 action 一律拒绝：归档查询没有 details 过滤，照做等于用「两个方向都给你」回答「我上传了什么」，正是在线读取的方向过滤要防的那件事；某天没有归档文件是空答案而非错误（年轻部署什么都没过期是常态）。客户端只在在线读取为空时才问归档， populated 面板依旧一次请求；归档读失败不让整个加载失败（没设 `AUDIT_ARCHIVE_DIR` 是服务端配置事实，不是 owner 做错了什么）；面板为来自归档的行加标注，因为「什么都没传过」和「在线表里已经没有了」是两件事——标注放在行**上方**而不是取代行，这些行是存在的，只是离开了在线表。验证：6 条 handler 测试 + 3 条 composable 测试，Go 全量 9 包绿，WebClient 263/263 且 E2E 清空在线表后断言归档行渲染并带标注，生产构建绿，七套 SDK 门禁绿，红绿双向（去掉回退，E2E 与 composable 测试同时红） |
| QD-111 | 文件传输审计 | `[x]` | 控制端完成/失败上报；设备归属校验、按 transfer ID 幂等；Web/Qt、OpenAPI 和七套 SDK 已同步；边界是控制端可信上报而非 Host 签名证明。**本轮 `a7d0a68` 修掉幂等的半个漏洞**：说明承诺"按 transfer ID 幂等"，但 `LogActorOnce` 的键是 (actor, action, resource_type, resource_id)，而 action 里带着 direction——`file.transfer.upload` 与 `file.transfer.download` 是两个 action。于是同一个 transfer ID 用两个 direction 上报会产生两行，承诺只对重试成立、对 ID 本身不成立。这不是假想形状：控制端分别上报上传和下载，而先推后拉同一个文件的同步、或改过 direction 后的重试，在审计里读作两次传输。direction 移进 details（它描述这次传输）而非身份键（它标识这次传输），file 与 clipboard 两个 handler 同病同修。按 direction 过滤仍然可用——"我上传过什么"是真问题——direction 专有名解析为聚合 action 加该 direction，在 details 上匹配；聚合名也可作过滤，未知名仍是 400 而非空页，拼错不能结案。OpenAPI 枚举从四个值涨到六个，七套客户端已重新生成。
| QD-112 | SSO、多租户、计费 | `[-]` | 当前不做 |
| QD-113 | Trust 自动审批线程亲和性 | `[x]` | 上游 issue #17 / PR #22 的修复 `5969ae3` 已合并进 `upstream/master`，fork `3d44dfc` 已同步且 `QuickDesk/src/api/TrustHandler.cpp` 与上游零差异；本 fork 未重复实现。其回归测试现在真的跑过：Windows `Unit tests` job（run `40128304387`，head `f956d17`）报 `1/17 trust_handler_thread_affinity ... Passed 1.25 sec`、`100% tests passed, 0 tests failed out of 17`。此前它从未进入任何构建树——`test/trust/` 是随上游合并进来的独立 CMake 工程，根 `CMakeLists.txt` 没有 `add_subdirectory` 指向它，`QUICKDESK_BUILD_TESTS` 只构建 `QuickDesk/tests`；这期间还先后暴露并修掉两个更外层的阻塞：`recordinglifecycle.cpp` 只进测试 target 未进 app target（`8120ed6`），以及 multi-config 生成器下 `ctest` 缺少 `-C` 导致 16 个测试全部 `***Not Run`（`270b7d3`）。`f956d17` 补上接线，并新增 `scripts/e2e/test-quickdesk-trust-test-wiring.sh` 静态看守「在构建图里」这件事（红绿三种破坏均验证；其中两种抓出了检查器自身把注释里的选项名当成条件判断的假阴性） |
| QD-114 | 自托管 HTTPS/WSS 证书校验 | `[~]` | 上游 issue #20；Qt 子进程支持 SPKI 列表，Windows 服务安装支持 `QUICKDESK_SPKI`；需求单已形成 `docs/dev/自托管TLS真实CertVerifier需求.md`：缺陷定位到 `quickdesk_network_service.cc` 第 86-99 行的 `dangling_remote`（只为绕过 DCHECK、接收端从未绑定），推荐在 network 线程接真实 `CertVerifierService`，明确不接受 `SetCertVerifierForTesting` 关闭校验，附 8 条验收用例；仍需 Remoting 侧实现与 Qt/Windows 实机构建 |
| QD-115 | 信令服务器持有设备密钥的信任边界 | `[~]` | `3474203` 已明确当前服务器仍是信任方：device_secret 只有 Argon2id 哈希，但 access_code 明文、signal token/审批/归属由服务器控制。目标方案为 Host 身份密钥 + Owner 签名授权 + Host 本地验证；协议验收项尚未实现 |

## 5. 设备与会话

| ID | 功能 | 状态 | 说明 |
|---|---|---|---|
| QD-201 | 设备 ID、设备密钥、自动 provisioning | `[x]` | Host 自动注册 |
| QD-202 | Pair Code / 设备绑定 | `[x]` | 用户绑定与解绑 |
| QD-203 | 设备列表、收藏、最近连接 | `[x]` | WebClient 和 Qt 均有入口。本轮 `8977f28` 补上「最近连接」真正可读的那一半：连接记录一直在取、在记、在渲染，所以行数是真的，但每行只打印裸 device_id 加时间戳，而 API 在每一行都返回 `device_name`、`connect_ip` 和 `error_msg`——三个全被丢掉，其中 `error_msg` 正是用户打开这个列表要读的那个：标着 failed 却不说明原因，等于告诉 owner「出事了」而不说「出了什么事」。另外两个同样是白扔：设备名才是人认得的，九位数字不是；客户端地址则用来区分「是我的另一台机器」和「是我不认识的地址」。行派生收进 `connectionLogModel.js`，模板读命名字段而不是直接摸 API 形状，兜底也因此可测：没有设备名就退回 id 而不是渲染空白，失败但没有记录原因就完全不显示理由而不是一个空括号或「undefined」，时长为 0 不渲染而不是「0m0s」。既有 E2E 只数了行数，这正是它能存活的原因——数行数会在行内容什么都不说的时候照样绿，现在同时断言设备名、时长和记录在案的失败原因都到达页面。验证：WebClient 245/245（原 239，新增 6 条模型测试加扩写后的 E2E）、生产构建绿、`git diff --check` 干净；红绿双向：把旧的裸 id 行放回去，E2E 在三条新断言上同时红 |
| QD-204 | 地址簿与设备分组 | `[x]` | 上游 Roadmap 的 Address book & device groups；owner-scoped API/SDK、WebClient、Qt、成员整体替换、500 上限和绑定生命周期清理均已完成，提交 `f270487`、`b1a1155`、`162ac6e`、`b013a62` |
| QD-205 | 在线状态与心跳 | `[x]` | Redis presence + heartbeat |
| QD-206 | 连接历史与快速连接 | `[x]` | 上游 `dev` 与 fork 都已有 `ConnectionHistory` 和连接历史 UI；issue #18 看起来已过期，先验证再关闭，不重复实现。本轮 `8977f28` + `f56d88d` 补上它承诺的「快速连接」和记录本身的可读性：历史行此前只渲染裸 device_id 加时间戳，API 返回的 `device_name` / `connect_ip` / `error_msg` 一个都没用——`error_msg` 尤其致命，标着 failed 却不说明原因；而设备列表和收藏卡片都有连接按钮，唯独历史行没有，可历史恰恰是用户为了「回到刚才那台机器」才打开的列表。访问码不取自历史行：API 有意不存它，与 issue #18「Do NOT persist verification-code by default for security」一致；收藏把它明文存下来专供快速连接，对一个每次连接都会增长一行的列表是更差的取舍。改为按 device_id 与设备列表连接，设备列表确实返回 `access_code`。两个边界单独钉住，因为那里答错比没有按钮更糟：行点名的设备已解绑则不给按钮，设备在但没有码也不给——连接必须校验码，那个按钮只会产出「no access code」toast；连接只按 device_id 匹配，匹配不上的行不会悄悄借用邻行的码。**issue #18 四项诉求的核对结论**：①连接后自动保存 device-id/别名/时间戳——已有（服务端 `connection_histories`，WebClient 自动 `recordConnection`）；②主界面展示历史、点击填入以快速重连——本轮补齐（历史行直接给连接按钮，比「填入输入框再连」少一步，且 WebClient 本就没有设备 id 输入框）；③编辑别名、删除单条、清空——当时**部分**：别名编辑在设备行有（`PATCH` remark），历史行自身没有编辑/删除/清空入口；`489e215` 补上删除半边：`DELETE /v1/me/connections/:id` 删单条、`DELETE /v1/me/connections` 清空并回报删了几行，两者都以 owner 作用域收口——单条的 owner 写进 WHERE 而不是取出再比对（先查后判会在两条语句之间仍然删掉别人的行），别人的行回 404 而不是 403（调用方对那个 id 的视角就是「不存在」，确认它存在等于泄露 id 是真的）；客户端删除成功后本地摘行而非重取，清空先确认。④不持久化验证码——满足且必须保持（前两轮的设计正依赖它）。**至此 issue #18 四项诉求全部满足**，可回复并建议关闭 |
| QD-207 | 会话创建、运行、结束、清理 | `[x]` | `remote_sessions` 控制面已落地。前三段（创建/运行/结束）确实完整：`Open`、`Close`、孤儿 sweep（没有存活实例上报时甚至拒绝执行而不是把所有人关掉）都在；第四段「清理」此前是空的——已结束的行没有任何上界，每个曾经建立的会话都永久占着一行。审计配置里早就把这个道理写明了：没有自然终态的表只会增长，保留是唯一能约束它的东西。`9c38ce6` 按同一形状补上：`PurgeEndedSessions` + `REMOTE_SESSION_RETENTION_DAYS`（默认 0＝永久，升级不该让现存历史开始消失）+ `REMOTE_SESSION_RETENTION_SWEEP_INTERVAL_MINUTES`（默认 60）。两条不能越过的边界：`ended_at` 为 NULL 的行绝不碰（那个会话还在，删掉等于让控制面忘记里面有人）；窗口按 `ended_at` 而不是 `started_at` 计——跑了四个月、昨天结束的会话是近期历史，按开始时间算会正好删掉 owner 最可能查的那些行。第二条由最强的那条测试钉住：WHERE 写成 `started_at` 时，sweep 删长会话、留昨天的短会话，正好反了。验证：5 条服务测试，Go 全量 9 包绿，WebClient 263/263 未触及仍绿，七套 SDK 门禁绿；红绿双向：把 WHERE 换成 `started_at`，五条里三条红（含点名四个月会话那条） |
| QD-208 | 会话详情、录像、访问日志聚合 | `[x]` | WebClient 和 API 已完成。本轮 `359cbf7` 补上聚合里被跳过的半边：服务端 `detailEnvelope` 一直把完整 `recordings` 数组放在响应里，UI 却只读 `counts.recordings` 什么都不列，而访问日志有完整列表——于是「有几条录像」成了全部信息，owner 不知道是哪一条、什么时候录的、有没有东西可播。详情改由 `sessionDetailModel.js` 描述而不是模板直接摸原始结构，这也顺带定下一件事：**counts 与数组不一致时以数组为准**，因为 owner 依据的是行，而「写着 99 旁边两条」是个谎；计数仍显示，缺 counts 时回退到数组长度而非 0，免得负载不含 counts 就被渲染成空。每条录像显示时间、状态、时长，只在有媒体时可播；服务端还在写的行明说「暂无媒体」而不是给一个注定失败的按钮。播放解析 artifact URL 并先取短期签名 URL（浏览器没法给 `<video src>` 或 `window.open` 带 Bearer token），与录像页同一份实现共享而非复制，两页不会对「这条能不能播」给出不同答案。验证：10 条新模型测试，E2E 断言录像以行渲染（时间、状态、时长、不可播行说明原因），WebClient 260/260，生产构建绿，红绿双向（删掉模板里的录像列表，新 E2E 即红） |
| QD-209 | 单会话踢人、按设备批量清场 | `[~]` | 服务端、Web 已完成；Qt 侧 `revokeSession` / `revokeDeviceSessions` 一直存在但没有任何自动化检查，本轮补齐（`a0fc422`）：对记录的请求而非仅响应做断言——POST 与目标路径、成功信号带回 id、按设备清场的 `revoked`/`failed`/`truncated` 三个计数完整到达、404 与 503 以服务端原文冒泡、空 id 不发请求。**该测试当即抓到一个真实缺陷**（`874336b`）：`QUrl::toPercentEncoding` 的第二参是「保持原样的字符集」且默认为 `/`，适合编整条路径却不适合编单个路径段——会话 ID 含 `/` 时会多出一个路径段，请求打到错误端点，而服务端只会回 404，用户看到的是「会话不存在」而不是「id 在传输中被改写了」；同一 helper 也服务于会话详情 GET，故一并修复。断言本身也错过一次：`QUrl::path()` 返回**解码后**的路径，`%2F` 会变回字面 `/`，拿它比较恰好看不见要抓的缺陷，改用 `toEncoded()` 后才行。Windows CI run `40145248663`（head `409deba`）19/19 通过。`6a9324a` 随后把这条规则固化为单一实现 `common/urlencode.h` 的 `EncodeUrlPathSegment`：原有十处调用里，两份各自复制的 helper（其中一份仍带着默认 exclude 集合）加六处内联写法，现全部收敛到一处，并新增 `quickdesk_urlencode_test` 钉住「段内斜杠必须编码」等边界（Windows CI run `40146772610`，head `6a9324a`，20/20 通过）。**已核实其余九处均为 `v1/me/devices/<device_id>/...`，而信令服务生成的 device_id 是 `%09d` 九位纯数字，斜杠不可达——那九处现在正确，但都不是活缺陷；唯一活缺陷是 session id，其契约允许 64 字节任意字符。** 剩余：Qt 真机回归仍待补 |
| QD-210 | 跨实例会话撤销 | `[x]` | 已按实例寻址；命令发布前先落库（`e236629`）：此前"读库与发布之间进程死掉"会让撤销命令凭空消失，只能等 presence lease 过期后由 sweep 在约一分钟后收尾，而那是在说"实例死了"不是说"这次撤销"。新增 `session_revoke_commands` 表（与 `event_outbox` 分开：事件扇出给订阅者，命令寻址给唯一实例且只动作一次），`revokeOne` 远程路径先 `Record` 再 `AskInstanceRevoke`，发布失败即 500 而非假装成功；`SessionRevokeCommandDeliverer` 作为后台 worker 负责 claim→publish→mark，发布失败释放 claim 并记原因，下一轮重试而不是把尝试当完成；`RemoteSessionService.Close` 在同一事务里清掉命令，覆盖 socket 自己断开的路径（自然结束的会话没有可撤销的东西，留着命令会被永远重投到早已关闭的 socket 上）。记录失败映射为 500（`revokeRecordFailed`）：写不进去就发布，等于退回原来那个窗口。顺带修掉 `LiveInstances` 的枚举缺陷（`861c5b3`）：它用 SCAN 数 lease key，而 Redis 文档允许 SCAN 在游标归零前漏掉已存在的 key——漏一个活实例，sweep 就把它的会话全部盖成已结束，正是 sweep 注释里说自己要防的事故。改为读实例集合（SMEMBERS 精确），成员按各自 lease 校验并在读时清理，无需 reaper；`seed-smoke` 的 `liveInstance` 同病（它会把设备挂到 lease 已过期但 Redis 尚未回收的实例上），一并改。验证：6 个 handler 测试（命令带齐投递所需、熬过写入方退出、已投递与未尝试可区分、会话结束清命令、`Pending` 跳过已结束会话、发布从未发生仍回 500）；5 个 service 测试覆盖投递循环（按实例寻址、捡起发布方从未发布的命令、发布失败重试而非消耗、会话已结束不投递、每次投递只报一次）；2 个 `Close` 路径测试；1 个 Postgres 测试在私有 schema 里重放迁移文件里这张表的 DDL 并驱动 claim/deliver，钉住 SQL 与模型不漂移。红绿双向三处（去掉持久化写四条同红、忽略会话 join 两条同红且含"给已结束会话投递"这个危险行为、去掉 Close 清理两条同红）。`go test ./...` 八个包在干净 scratch Postgres 全绿，`go vet` 干净。真机端到端：造出"开着会话 + 未投递命令"，运行中的 deliverer 捡起并发布（日志 `delivered a recorded command for session=e2e-crash-session`，行 delivered=1 attempts=1）。 |
| QD-211 | 撤销冷却期 | `[x]` | 支持窗口和递增冷却 |
| QD-212 | 设备列表标识本机 | `[x]` | 上游 issue #19；Qt 我的设备/收藏已标记“本机”，候选 `f1ce096` 已独立验证 |
| QD-213 | 访问码不可用回归 | `[x]` | 上游 issue #19；Qt 缺失/过期访问码会按单设备回源并自动继续连接，候选 `6928d24` 已独立验证 |
| QD-214 | Windows 文件上传用户上下文 | `[!]` | 上游 issue #19；SPAKE2 已修通，macOS 上传成功；Windows 普通用户为 `type=2`（`0x522`），SYSTEM/session 0 为 `type=7 NOT_LOGGED_IN`，根因是 Host 服务未在可解析的交互用户上下文运行 |
| QD-215 | Windows 启动 Qt platform plugin 错误 | `[~]` | 上游 issue #7；发布链新增 x64、VC Runtime、`qwindows.dll` 和 Qt 直接依赖校验，Windows 正负向测试通过；上游候选已 rebase 到 `upstream/dev@5db316c` 成为 `8385856`，并创建上游 PR #32（OPEN）；PR #32 已补充 Qt 版本一致性检查：issue #7 的错误信息是「Available platform plugins are: windows」，说明插件**在**且被找到却仍初始化失败，而原有校验（文件存在、x64、直接依赖可解析）对这种包全部通过；新增按文件版本比对包内 Qt 集合，出现多个版本即拒绝并指出具体文件与版本，版本读取器可注入以便测试。测试在 PowerShell 容器内实跑通过（4 条用例，含红绿双向验证）。原始坏安装包仍未复现，因此这是对该类失败的防护而非对那个具体包的确认修复 |
| QD-216 | 主机连接与设备 ID/访问码加载失败 | `[x]` | 上游 issue #9；根因已定位，见 `docs/dev/QD-216设备ID访问码加载缺陷.md`：启动用两段固定 `singleShot(500)` 发 `hello`+`connect`，而 `connectToServer` 的 `isReady()` 只查进程/socket 状态、不查握手完成，Host 启动慢于 1 秒时 `connect` 被丢弃；此后无重试、Host 不补发，`onHostReady` 永不到达，`deviceId`/`accessCode` 永为空；且 `HostManager::errorOccurred` 在 MainController 未连接，`NOT_READY` 被静默丢弃，用户只看到两个空白字段。已有未使用的就绪信号 `helloResponseReceived`。修复已落地：门控与退避决策抽成不依赖 Qt 的 `src/core/hostreadiness.{h,cpp}`，启动路径改为消费它。**第 1、3、4 条验收用例已有自动化证据**：`QuickDesk/tests/fake_host.cpp` 是按参数延迟回答 `hello` 的假 Host，走真实 native-messaging 二进制帧；`hoststartup_integration_test.cpp` 用真实 `HostManager` 驱动它，Windows CI run `40133964841`（head `cbfcbe9`）报 18/18 通过，`quickdesk_host_startup_integration_test ... Passed 1.74 sec`。**第 2 条已补**：`af57353` 新增 `ServerStatusDetail` 枚举（与 `ServerStatus` 同置于 `common/ProcessStatus.h`），`MainController` 暴露 `hostServerStatusDetail` 属性，所有 `m_hostServerStatus` 写入收敛到 `setHostServerStatus()`，重试预算耗尽时按「有没有见过 hello」区分「Host 进程未响应握手」与「Host 未接受连接」；`MainWindow.qml` 把理由拼进状态栏并由 `.ts` 翻译（Windows CI run `40137597030`，head `d8be923`，19/19 通过，`quickdesk_server_status_detail_test` 在列）——第一版在 C++ 里拼英文句子，改成枚举正是为了让中文用户看到中文。写 `core::` 前缀与漏提交 `QuickDesk/CMakeLists.txt` 两处各废一轮 CI（`c24fbdb`、`d8be923` 修复）。**第 6 条的两个平台证据都已具备**：Windows run `40137597030`（19/19）与 macOS arm64 run `40201665718`（head `28b7b9b`，20/20，`quickdesk_host_startup_integration_test` `Passed 2.37 sec`，该 job 十个步骤全部 success）。此前只记 Windows 是因为 macOS run 整体是红的，没去看它内部——同一个 job 里 `Build and run the test suite` 其实通过了，红的是另一个步骤（见下）。**第 5 条已补**：`a9c8261` 给假 Host 加 `--host-ready-delay-ms` / `--snake-case-ids`，新增用例走**独立 hostReady** 这条正常启动主路径（此前只覆盖了 ID 随 hello 一并到达的捷径），并跑两种字段拼写各一遍。Windows run `40209985101`（head `a9c8261`）20/20 通过，`quickdesk_host_startup_integration_test` `Passed 1.83 sec`。**红绿已验证**：把 snake_case 那半边的期望值改错后推送（`cb2da01`），CI 回报且仅回报 `FAIL: case 5 (snake_case): the device ID arrives on hostReady`（95% tests passed, 1 tests failed out of 20），证明该分支真的在执行、参数真的传到假 Host、`handleHostReady` 的 snake_case 兼容真的被测到；`af7052d` 已恢复。QD-216 六条验收全部完成，两个平台（Windows run `40209985101` 与 macOS run `40211749988`，均含 `a9c8261` 的新用例，各 20/20）均有证据，且第 5 条做过红绿双向验证，符合第 15 节完成定义，转 `[x]` |

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
| QD-314 | 多人同时控制 | `[~]` | `d435c6f` 已落控制权模型与冲突策略：设备可设 `control_mode=exclusive`，信令服务器是唯一权威，持锁者收 `control_grant`、被抢占者收 `control_revoked`（含新持有者），WebClient 收到即停止注入输入；默认 `shared` 保持既有行为，`max_sessions` 仍独立管「能看」，`control_mode` 管「能控」。WebClient 策略设置入口已补：设备行的控制权对话框读真实策略、只在实际变更时启用保存，并经 `PUT /access-policy` 全量提交（该端点是整体替换，所以表单带回整份策略，否则一次单字段保存会把 IP 列表、会话上限、时间窗和审批设置一起清掉）；真机双控 E2E 已补：`control-mode.e2e.mjs` 走真实登录与 HTTP 断言其余字段存活，`control-grant.e2e.mjs` 开两个真实 client 信令 socket 观察线上抢占——第二个控制器收到 `control_grant`，第一个收到带新持有者的 `control_revoked`；shared 模式两人都发且谁都不通知；持有者断开后机器恢复可控。`fb43070` 再补多实例：此前持有者只记在单个进程的内存里，而请求控制权的控制器连的是 A 实例、已持有者的 socket 往往在 B 实例——两个进程各自回答「你可以控」，正是独占模式要防的「两人同时开一辆车」。持有者现在是设备事实而非进程事实：改存 Redis 带 TTL（与撤销冷却同一类短生命跨实例状态），任一实例上的抢占对所有实例可见，任一侧发起都能被另一侧看到，释放后机器到处都可接手，Host 掉线也一并清掉；模式与持有者同存，所以释放后的机器读作「空闲」而不是悄悄变成 shared；无 Redis 时行为与改动前完全一致。`control_revoked` 有同形缺陷：原先只投递给本进程持有的 socket，被抢占者在别的实例上就永远收不到通知、会继续往一台已不属于它的机器里打字；现在该帧按寻址通道跨实例投递（与 revoke bus 同构），路由依据是 socket→实例索引，注销时只在条目仍指向自己时才清除，避免旧 socket 的清理删掉新实例的记录。三处都有红绿证据。剩余：Qt 客户端不消费这两个帧（信令 WS 由预编译 Chromium Remoting 持有，改它需动 Remoting，与 QD-214 同边界） |
| QD-315 | 画笔标注 | `[~]` | 控制端标注画布已落地：`js/ui/annotationModel.js` 管笔画数据（**按被控桌面坐标存储**，所以不同窗口大小的观看端把同一笔画画在桌面同一位置），`annotationLayer.js` 是覆盖在视频之上的独立 canvas（画进视频自己的 canvas 会被下一帧解码清掉），`annotationChannel.js` 经 BroadcastChannel 在同一设备的观看端之间共享——与控制/音频抢占同一条通道，因为笔画是几十个数字，走控制面会给每次指针移动写一条持久 outbox 行；消息是增量的且清限只清自己的（按 authorId），一个观看端无法抹掉别人的标注；笔画上限 200 条，超了丢最旧的——最新的那条正是观看端此刻在看的那条；工具含画笔/荧光笔/箭头/矩形，荧光笔半透明；真机 E2E 在真实 canvas 上画并采样像素确认落墨，且断言空闲时 `pointer-events: none`（否则关掉标注会把远程控制一起杀死）。`onloadedmetadata` 与显示器切换都会把新的被控分辨率送进图层（笔画按桌面坐标存储，分辨率一变缩放因子就变，不同步会让所有已有笔画落到桌面错误位置）；断开时连同 canvas 一起拆除，迟到的笔画不会让页面崩。移动端入口已补：桌面浮动菜单在移动布局下不创建，标注原本在手机上完全不可达，现在 `mobileToolbar` 有独立按钮；同时补 `touch-action: none`——没有它浏览器会把触摸拖拽识别成滚动或捏合并取消指针流，手指画出来是一次跳变而不是一条线。剩余：Host 侧把标注叠加进被控画面需要 Remoting 协议扩展，与 QD-004/QD-007 同边界 |
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
| QD-406 | 双向图片剪贴板 | `[ ]` | 需求单见 `docs/dev/图片剪贴板Remoting需求.md`；关键约束是图片不能走控制通道（文本剪贴板走 `controlChannel`，一条 4K 截图就有数 MB，会挤占同通道的输入事件）；实现待 Remoting |
| QD-407 | 系统级文件剪贴板 | `[ ]` | 当前不做主链路承诺 |
| QD-408 | 文件传输审计 | `[x]` | 同 QD-111 |

## 8. 录制、回放与文件生命周期

| ID | 功能 | 状态 | 说明 |
|---|---|---|---|
| QD-501 | 录像索引、归属、分页 | `[x]` | `session_recordings` |
| QD-502 | Web 控制端手动录制 | `[x]` | MediaRecorder |
| QD-503 | Qt 控制端手动录制 | `[x]` | DeviceRecorder |
| QD-504 | Host/Relay 自动录制 | `[ ]` | 当前迭代 QD-004 |
| QD-505 | 断线自动收尾 | `[x]` | 控制端录制在 `disconnected` / `failed` / `closed` / `error` 时自动 `stopRecording()` 并上传；`connected` / `reconnecting` 不中断。Host 侧权威录制的收尾仍属 QD-004 |
| QD-506 | 录像上传、断点续传 | `[x]` | `de88f07`：Web 在线录像按 4 MiB 分块上传，409 按服务端权威 offset 续传，瞬时错误重试、进度展示和失败收口已完成；WebClient 50/50、生产构建和真实 SQLite 4 分片逐字节 E2E 通过 |
| QD-512 | 录像列表实时刷新 | `[x]` | `52cca18`：`RecordingsPage.vue` 原先只在 `onMounted` 刷新，而服务端会因**另一个控制器**的行为发出 `recording.created` / `updated` / `deleted`——别人上传完成、删掉一条，打开着的页面不会变，刷新页是唯一解法。**这个缺口不是读页面找到的**（页面看起来很完整），而是把服务端事件清单与 `userSync` 的分发 switch 做差集找到的：33 个服务端事件里 12 个没有客户端分支，其中只有这一个对应着「展示受影响行」的页面，其余是通知类（走自己的频道，不需要分发分支）。`userSync` 现将三个生命周期事件 surfaced 为 `recording-changed`，与其他类型共用同一套 rev 守卫，并丢弃没有 `recording_id` 的帧——页面说不出哪一行变了，据它刷新就是猜。页面监听后刷新，按 250ms 去抖：一次上传会产生 created 紧跟 updated，不防抖就会反复重载列表。验证：5 个新单测 + 一个新的 mocked-browser E2E（驱动真实页面走真实分发器，断言列表被重新拉取——只测分发 switch 的单测会在页面仍然无视事件的情况下通过）；E2E 在 workflow 里按文件名列出而非 glob，两小时的 wall-soak 门禁因此不会被吞掉；`npm test` 223/223、新 E2E 绿、生产构建绿、fresh 库上 SignalingServer 全包绿；红绿两轮。 |
| QD-507 | Web 录像列表、播放、下载、删除 | `[x]` | `RecordingsPage.vue` |
| QD-508 | Qt 录像回放页 | `[x]` | `74c7022` 已覆盖列表、分页、缩略图、应用内播放、外部打开、删除和实时刷新 |
| QD-509 | 缩略图、保留期、孤儿回收 | `[x]` | 服务端生命周期已完成 |
| QD-510 | 工作流录制与回放核心 | `[x]` | 远程 UI/skill-host 双通道录制回放、旧格式兼容、递归参数替换、失败策略、持久化报告和查询工具已完成；`cargo test` 19/19 通过 |
| QD-511 | 工作流状态校验与智能回放 | `[~]` | 预录屏幕断言与状态不匹配暂停已完成：`screen_hash` / `screen_contains` 可挂到任一步骤，回放前校验，不匹配即 park 并写入原因，新增 `set_step_expectation` 工具；OCR 关键词不是缺口；`1ad0b85` 补人工接管：park 的 run 留下 resume point 并可通过 HTTP 查询、接管和清除，只有 pending 步骤才产生 point，步骤总数取自 run 而非 workflow；本轮纠错一个阻断性契约缺陷——MCP 原先发送 `textContains` / `frameHash`，而 Host `VerificationService::evaluate` 只认 `text_present` / `text_absent` / `text_present_exact` / `window_title_contains` / `window_title_equals`，未知类型一律 `passed=false`，导致带断言的步骤**永远 park**（屏幕匹配也如此）；同时 `screen_hash` 在 Host 侧没有对应条件类型，改为经 `getUiState` 取 `ocr.frameHash` 本地比对；观测值也从不存在于响应的 `observed.frameHash` 改读 Host 真正报告的 `results[0].actual`，park 报告不再把「期望 Save，观测到空白」写成事实；剩余自动 AI 失败解释/决策 ；本轮 `a3ebfae` 补上失败原因的分层：此前 park 只输出事实句「expected X, observed Y」，读它的人无法区分「屏幕确实变了」「文案变了或 OCR 漏识别」「断言本身没录好」「该检查类型本 build 不支持」——四种失败的排查去处完全不同。新增 `MismatchKind` 与 `classify_mismatch()`，**从 `StepExpectation` 而非从 outcome 分类**：outcome 只带两个字符串，不支持的 kind 在那儿与真不匹配完全同形；`expectation_mismatch_reason` 增收 `&StepExpectation` 并把分类建议接在事实之后，事实记录与建议因此可分开演进。只改一个调用点，`Mismatched` 的 11 处构造点未动——这正是传 expectation 而不是加字段的目的。测试红绿做了两轮才成立：只断言渲染后字符串时，两类塌成一类仍剩三种不同字符串，照样绿；只断言 kind 互异时，四类共配同一句通用建议也照样绿。两种漏洞都已补，且都确认过破坏即红才提交。`cargo test --offline --bin quickdesk-mcp` 119/119、`cargo clippy --all-targets -- -D warnings` 零告警、`git diff --check` 通过。剩余自动 AI 失败解释/决策（需外部 AI 服务） |

## 9. 通知、任务与集成

| ID | 功能 | 状态 | 说明 |
|---|---|---|---|
| QD-601 | 用户通知通道与 Webhook | `[x]` | HMAC、outbox、重试、死信重放 |
| QD-602 | 投递确认与静默时段 | `[x]` | ack、一次性和周期静默 |
| QD-603 | 设备离线持续告警 | `[x]` | 阈值、重复提醒、维护窗口 |
| QD-604 | 指标型告警 | `[x]` | `37dd8aa`：Host 60 秒上报、单设备最新快照、三项独立 episode、持久 outbox、Web/Qt 规则入口和四套 SDK；`sustained_minutes=0` 首样本告警，真实 SQLite+Webhook E2E 通过 |
| QD-605 | WebClient 通知管理页 | `[x]` | `/notifications` |
| QD-606 | Qt 通知规则管理页 | `[x]` | `dbd828f` 已覆盖通道 CRUD/test/启停、静默时段、离线告警和投递运维 |
| QD-607 | 计划任务 | `[~]` | 服务端、WebClient、Qt 管理页和重试已完成；`revoke_all_remote_sessions` 已补齐控制面批量清场（最多 200 台）。**本轮 `151d85f` 补上「定时连接」中控制面真能交付的那一半**：新增 `connect_prompt` kind，到点向属主已打开的客户端发布 `device.connect_prompted`（带 task id，走与其他 kind 相同的 outbox 路径），并注册进 action 表使用户可创建、worker 真会跑。之所以只做这一半，理由写在上游 action 注册处自己的注释里：只有整个效果都在控制面内的 kind 才注册，因为需要客户端配合的任务「能被安排但永远不会真的运行」——连接本身就是客户端动作，控制面无法代替某人的控制器去拨一台设备。而连接周边的一切都是控制面工作，提示正是其中有用的那一半。**接线时抓到一个缺口**：事件从第一个提交起就能到 realtime 流，但它不在 `SupportedUserNotificationEvents` 里，`DispatchEvent` 直接丢弃，于是任何通知频道都订阅不到——提示会在 socket 上生效，却在属主真正配置通知的所有地方隐形，正是清单完成定义要防的那种半成品；两个测试在没有它时同时红。WebClient 的事件标签映射原先内联在 Vue 文件里，未收录的事件类型会把原始线上名渲染进表格单元格，现移到 `js/notifications/eventLabels.js` 并带明确的兜底，新类型补齐中英文案。设计取舍：`ConnectPromptAction` 依赖 publisher 接口而非 `*EventBus`，持久化路径经可注入的 enqueue 函数而非 `*gorm.DB`——裸 `&gorm.DB{}` 会在 gorm `Session` 内 panic，那会让 outbox 测试变成在测 gorm。验证：8 个新 Go 测试（含一个驱动真实 user-notification subscriber 并断言 delivery 行落地的端到端）、4 个新 WebClient 测试、带 CI DSN 的全包 `go test ./...` 绿、`npm test` 211/211、生产构建绿；红绿各做两轮。**本轮 `3ead9a3` 补上客户端消费半边**。上一轮把 `connect_prompt` 在服务端接通、并在客户端加了 i18n 标签就停了——`userSync` 的分发 switch 里没有这个分支，事件到达 socket 后无处可去：任务响了却谁都不告诉，与更早一轮通知订阅缺口是同一个半成品形状，这也是本轮从「上一轮留下了什么」开始查的原因。三处收口：①`userSync` 现在把它分发为 `connect-prompted`，与其他类型共用同一套 rev 守卫，且丢弃没有 task_id 的帧——说不出自己来自哪个任务的提示，等于让属主面对一个打不开的东西；②`ScheduledTasksPage` 监听后刷新列表，因为属主需要据以行动的 schedule、`last_run_at` 和状态本就在服务端，帧里只有一个 id，toast 说明连接已到点；③kind 标签映射移出 Vue 文件——它原先只收录最初两个 kind，第三个在表格里渲染成原始线上名；创建对话框的兜底列表是同一份映射的第二份拷贝、带着同样的缺口，现在改成同一份列表而非拷贝，否则在 `supported_kinds` 到达前打开对话框的用户根本无法安排这个功能。验证：6 个新单测，外加一个驱动真实页面、走真实分发器并断言 toast/标签/刷新的 mocked-browser E2E，且该 E2E 在 workflow 里按文件名逐个列出而非 glob；`npm test` 218/218，8 个 mocked E2E 全绿，生产构建绿；红绿两轮（移除页面监听则 E2E 红，移除 task_id 守卫则单测红）。**本轮 `78bf247` 补上 Qt 桌面端，这是同一条链的第四层**。前三层依次是：服务端发布 → 通知频道可订阅 → WebClient 的 `userSync` 分发 → 页面 toast；桌面端是唯一还没接到的地方，而它是靠「再走一遍链路」找到的，不是从清单上挑的。事件在到达页面前要过四道闸，每道都得放行：①`ScheduledTaskManager::handleRealtimeEvent` 现在与 `scheduled_task.failed` 并列接受该类型，发出 `realtimeConnectPrompted(taskId)`，并丢弃没有 task_id 的帧——页面拿不出可打开的东西；②QML 页面据此弹 toast 并刷新，`kindLabel` 补上新 kind（此前渲染成原始线上名）；③创建对话框的兜底 kind 列表补上 `connect_prompt`——它是 WebClient 那份列表的第二份拷贝、带着同样的遗漏，在 `supported_kinds` 到达前打开对话框的用户根本无法安排这个功能；④`controlplaneevent` 白名单放行，否则该事件完全无法触发 workflow，而「到点自动连上」恰是计划连接最合理的自动化用途。**一次险些扩大的误判**：grep `realtimeTaskFailed` 的消费者时，manager 和测试之外一无所获，看起来整个 Qt 侧都没接——并非如此，QML 是用 `Connections` 按名字连信号的，那种 grep 看不见，而失败事件那条路本来就是全通的。真正的缺口只是新事件，若按「Qt 侧都没接」去修，会重写本来正确的代码。**本地未验证**：本机没有 Qt 工具链也没有 Qt 构建容器，C++/QML 改动只做了审阅与 `SIGNAL()` 宏和声明签名一致性核对，没有构建证据；新增的两个测试（manager 集成用例、控制面白名单断言）首次运行在 macOS 与 Windows 的 CI job 上，与前几轮 Qt 工作取得证据的位置相同。本轮补上**编辑已有任务**：此前改一个排期只能删掉重建，而行里恰恰存着「它失败过」——`last_run_at` / `last_status` / `last_error` / `attempts` 全在删除时丢掉，而owner 想改的往往正是那个失败的任务。新增 `PATCH /v1/me/scheduled-tasks/:id`：**PATCH 而不是 PUT**，因为请求体就是创建那份，字符串字段没有「保持不动」的表示，PUT 会变成一次把调用方没送的字段全部清零的假承诺；服务端先读行再只覆盖 body 带上的字段。校验复用 Create 的全套（编辑就是创建，存一个 worker 算不出来的排期等于让它第一次 tick 就停车），两处差别都因为行已存在：outcome 四列一律不写（改排期不是替 scheduler 重新武装），以及**已跑过的一次性任务会被重新启用**——scheduler 在它跑完时把 `enabled` 置假，编辑若保持假，存进去的新排期永远不触发。`run_at` 与 `schedule` 互斥、跨用户 id 是 404 且行不被改动。客户端三层收口：WebClient 的编辑按钮把 stored row 经 `formFromScheduledTask()` 灌回同一个对话框（打开时是默认值就等于让 owner 覆盖一个他没选过的排期），Qt 的 `updateTask` + QML `requestEdit` 走同一条对话框。本轮补上**广义批量 agent 指令与回执**中控制面真能交付的那一部分：新增 `batch_skill_exec` 工具（`quickdesk-mcp/src/batch.rs`），把同一个 skill-host 工具并发发到多台设备，每台等一份回执，返回 `{total, succeeded, failed, devices[]}`。此前「批量」只有 `batch_operation` 这个 **prompt**——它只教模型自己循环，于是「哪台算成功」由模型决定，操作员拿到的是散文而不是台数。三个设计点：①**并发**，否则几十台排成一队等，而批量存在的理由就是同一条命令同时去打多台；barrier 测试证明真的在并发（串行实现会在 barrier 上死锁而不是 merely 变慢）；②**一台失败不带掉整批**，另外几十台的结果正是操作员要的；③报告按调用方的设备顺序排列，不按回执到达顺序，否则没法和请求对照。上限 50 台、device_id 先去重（重复会把一台机器算成两份回执）、单台 detail 截断到 2 KB 且不切坏多字节字符（切坏会在 slice 上 panic，而一批 panic 会连带丢掉其他所有设备的回执）。信任层按设备逐台前置检查——批量不是绕过单次确认的途径；审计与 memory 同样逐台写，复盘时必须看到每台机器各自发生了什么，而不是一行「 fleet 」。剩余：跨被控端的指令编排**已完成**（`c17c406`）：新增 `chain_skill_exec` 工具，按顺序跑多步、每步只在满足条件时运行——`always` / `on_all_ok` / `on_any_ok`（分层放量：上一步成功的那些才跑下一步）/ `on_failure`（清理场景，必须在失败时触发）。`batch_skill_exec` 表达不了这件事：它把同一个工具同时发到所有机器，回答的是另一个问题。不带 `device_ids` 的步骤跟随上一步的成功集合，于是一条链可以说"给服务器打补丁，然后重启打上的那些"而不必在两处重复罗列机器。三个判定：①整链先验证再执行——已对集群动作过之后才发现第三步写错，而这个系统无法回滚，所以未知条件名一律拒绝而不是默默当成 `always`（`when` 是唯一决定步骤是否运行的字段，拼错等于想要相反的意思）；②被跳过的步骤报"跳过"而不是"跑了但失败"，且不成为下一步的条件依据，报告列出每一步含没跑的——链停住是關於一条链最重要的事；③信任按设备按步骤在发送前逐一检查，链不是绕过单次确认的途径。步骤内设备复用批量的并发扇出，链不比它取代的批量更慢。**MCP 测试首次进入 CI**：`quickdesk-mcp` 此前只被两个 workflow `cargo build`，测试模块在 `cfg(test)` 下才编译，所以那 129 条一直只有开发机证据。本轮在 Windows 与 macOS workflow 的 build 之后各加一步 `cargo test --offline` + `cargo clippy --all-targets -- -D warnings`。**该步骤第一次跑就失败**：runner 上较新的 clippy 把 `format!("{}", &p.device_id)` 的冗余引用当错误，本机 clippy 不报——正是加它的理由。`4724af1` 修掉后 Windows run `40308590041` 报 `129 passed; 0 failed`，clippy 零告警。 |
| QD-608 | OpenAPI 与生成 SDK | `[x]` | TypeScript/Go/Python/Java/C#/PHP/Kotlin 七套客户端已提交，生成器、freshness 门禁和 CI 编译覆盖全部语言；C# `dotnet build`、PHP Composer/lint、Kotlin Gradle build 通过 |
| QD-609 | MCP / AI 自动化 | `[x]` | QuickDesk 原生能力；也是上游维护者当前最优先方向 |
| QD-610 | 业务系统集成与自定义工作流 | `[x]` | `bca8149` + `3f94911` + `e518194` + `e8ce807` + `9f7be0f` + `2e09d5b` 已提供同步/异步 HTTP trigger、持久化 trigger CRUD、60 秒~365 天 interval worker、next_run_at at-most-once、运行结果回写、Idempotency-Key 持久化重放、异步 job 状态/取消、HMAC 结果回调、白名单控制面事件订阅、事件过滤/幂等 claim、设备白名单、trust/emergency、单并发和 429；`1176a92`~`a94bba1` 补持久回调 outbox、重试、死信、人工重放和幂等 ack；`d3dc4cf` 补 queued job 重启恢复；`3c31d9a` 补 UTC cron；`20d02cc` 补 IANA 时区和 DST；`47d5e19` 补批量 ack；`af8653f` 补 MCP 事件重连；`5a4ff4a` 补按用户持久事件 replay；本轮补法定节假日跳过、年度日历扩展、默认 2048/可配置 replay 容量、cron 漏跑补跑、`5f24ff7` 的同目录多进程回调/job claim 协调、`cd730d1` 的回调操作者归属、`c39443c` 的服务端 replay retention 配置、`2d211c1` 的 retry claim 崩溃恢复、`6c71f97` 的数据库事务 event outbox 基础设施/`scheduled_task.failed` 迁移、`f2534ec` 的设备访问 granted/denied 迁移、`6bdc057` 的策略变更迁移、`3bbc175` 的远端会话生命周期迁移、`cd08a0f` 的录像生命周期迁移、`c650d3d` 的访问审批生命周期迁移、`5e2542c` 的共享访问码生命周期迁移、`b6855cc` 的离线告警迁移、`c4a28ee` 的指标阈值告警迁移、`0c621ef` 的设备控制面变更迁移、`0b746b9` 的收藏设备迁移、`cddc0db` 的在线状态离线事件迁移，以及 `314c73b` + `a1d34c5` 的账号会话撤销事件（改密码、短信改密、refresh 家族断裂、管理端改密/删用户/强制下线/批量禁用/批量删除、本人登出、按 id 删会话和计划任务全局登出共 8 个生产者）迁移，上游候选继续推进；`2848be5` 补公平调度，`97b8619` 补 Owner 委托权限，本轮再修一个会冻结 MCP 服务的自死锁：`log_operation` 持着 `db` 锁调用 `record_failure`，后者抢同一把非重入锁，任何一次失败的工具调用都会让服务端永久挂起；已改为先 `drop(db)` 再记录失败，并用带看门狗的测试把回归固定成失败而不是无限挂起（`cargo test --bin quickdesk-mcp` 113 项全过）。`93a0d13` 再补跨机器分布式队列：worker 可用 `QUICKDESK_WORKFLOW_PEER_JOB_DIRS` 监视其他机器提交 run 的目录，`list`/`get`/认领都跨目录解析，重启恢复因此也能接管 peer 那里 claim 已过期的 `Running` run；配对的 `QUICKDESK_WORKFLOW_CLAIM_DB` 才是防止两台机器同时执行同一个 run 的机制，两者必须一起设置。该能力依赖 fork 自有的 `workflow_job.rs` / `FairQueue` / `DistributedClaimStore`，上游没有同名文件，无法摘成最小 PR，留在 `Fork-only`（详见 `docs/dev/远端发布与上游PR清单.md`）。本轮同时让 `get_failure_memory` 真正返回 resolution：该列自建表起从未写入，而工具描述一直承诺它；按读解释会让读变成写，因此未变化的行跳过 UPDATE，轮询失败内存的客户端不会反复重写数据库（`cargo test --bin quickdesk-mcp` 117 项全过，含看门狗死锁回归与幂等写回归，均做过双向验证）。QD-610 至此在 fork 侧收口 |

QD-610 当前增量：`b2a0a98` 已把 admin force-unbind、单条/批量设备硬删除和 secret
rotation 的事件写入同一数据库事务，`6386a48` 又把 TURN/STUN settings 与
`turn.config.changed` 合并到同一事务，`8de8af0` 补齐管理员跨账号来源 ban/unban
的策略事件事务化；其中来源封禁依赖 fork 的 QD-109 策略模型，不进入 PR #25 的
通用 outbox 候选。上游
`codex/upstream-event-outbox@4a3566f` 保持最小候选，不携带 fork-only 的分组清理或
指标代码。

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
| QD-808 | 客户端自动更新消费 | `[~]` | 当前迭代 QD-005；Windows 用 `/SILENT /CLOSEAPPLICATIONS /LAUNCHAPP` 启动安装器，已验证 4.2.8.0→4.2.9.0 全闭环。macOS 更新助手完成 DMG 挂载、bundle/version/签名校验、暂存替换、失败回退、重开和临时包清理，ad-hoc 真实 DMG E2E 通过。`2c3ca9b` 补 Release 打包默认强制公证的门禁。`e702a19` 起 macOS 更新助手会拒绝用户 Mac 打不开的包（ad-hoc 签名或 Gatekeeper 不信任），默认严格、`QUICKDESK_ALLOW_UNNOTARIZED_UPDATE` 显式放宽；两个 macOS E2E 已进 CI。剩余：Developer ID 签名、Apple 公证凭据，以及用真实公证包做一次升级回归 |
| QD-809 | SQLite 轻量服务端 | `[x]` | QD-006 已收口；单文件 SQLite + 默认进程内 Redis，健康检查、启动和重启 E2E 通过；仅支持单进程，Redis 状态重启即失 |
| QD-810 | 正式备份、恢复和容量配额 | `[x]` | `64ab93a`：SQLite 一致性归档与安全恢复、`-force` 旧副本和失败回滚；PostgreSQL `pg_dump` + 媒体目录运维路径；录像/安装包总配额、管理接口和升级通道页状态；真实备份恢复 E2E 通过 |

## 12. 质量与验证

| ID | 功能 | 状态 | 说明 |
|---|---|---|---|
| QD-901 | Go 全包测试 | `[x]` | 已有 `go test ./...` |
| QD-902 | WebClient 单测 | `[x]` | Node test |
| QD-903 | WebClient 生产构建 | `[x]` | Vite build |
| QD-904 | Web End-to-End / Playwright | `[x]` | 已覆盖 QD-907 信令重连 7 场景、QD-002 文件管理器、QD-003 真实三标签抢占/释放/隔离、屏幕墙多瓦片、设备页完整流程，以及临时 Redis + SQLite SignalingServer 上的真实登录、Realtime snapshot、分组写库和 access-code 校验；长稳与规模转入 QD-908 |
| QD-905 | 信令、TURN 并发压测 | `[x]` | 已有负载和 soak 脚本 |
| QD-906 | Windows/macOS 真机矩阵 | `[~]` | Windows 主链已验证；输入、音频、剪贴板边界待补 |
| QD-907 | 协议异常、权限异常、重连耗尽 | `[~]` | 已补信令 auth_ok 超时、认证前 payload/send 门禁、访问策略/错误访问码/排队/签名失败分类、上传 transport/503 重试耗尽，以及 WebSocket 断线后重新签发 token 的有界 `0/1/3/7s` 重连与耗尽状态。Playwright E2E 已覆盖第 1~3 次恢复、四次耗尽、不可重试策略拒绝、上传中断与信令重连并行、旧 Session 迟到事件隔离；`39ea81c` 补弱网证据：在既有 WebSocket harness 上按固定模式丢 auth 回复，覆盖「首次回复丢失后仍重连成功」和「链路始终不应答时仍走完四次并落到耗尽状态」两端；破坏「失败不重试」这条边界时两个新测试连同原有重连测试一起失败，恢复后全绿（`node --test test/e2e/reconnect.e2e.mjs` 11 项、`npm test` 147 项）。未用 Chrome CDP 网络节流：Playwright 接管后的 WebSocket 不受其影响，已实测确认。本轮补**长时间抖动**与抖动终态：`reconnect.e2e.mjs` 新增两条——链路只晚不回（120/260/400/180ms 四连）必须恢复、落定且不再重试，不能把「慢」当「坏」；先挂后断必须花完整套重试预算再落到耗尽状态，不能因第一个慢回复就判死。红绿：把退避预算砍成一次，两条连同原有重连测试一起红。**并抓到一个真缺陷**：`_connect` 捕获异常后只返回 false，于是会话连接内部抛出的**不可重试拒绝**被压成通用的 `reconnect attempt failed`，重试循环于是花完整套预算去重新发现一个服务端已经说过的拒绝——四次尝试一次就够，审计写 `reconnect_exhausted` 而不是 `reconnect_rejected`，状态栏还告诉操作员「网络一直失败」，而服务端拒掉的是他的凭据。三处修：`_connect` 保留错误供调用方重抛（成功时清除，避免把旧失败归给新尝试）；transport 在服务端发出 auth 拒绝时记录下来并随 close 带上——被拒的凭据不是坏链路；`_attemptReconnect` 带服务端 code 重抛，状态栏据此显示「信令鉴权失败」而非泛泛的耗尽文案。本轮补**进程级断网**：`reconnect.e2e.mjs` 用 Playwright 的 `context.setOffline` 把整个浏览器进程断网——这和 socket 关闭是两种失败，socket 关掉会来一个事件，进程断网什么事件都没有，连接不关闭、只是不再工作。**该测试第一次跑就失败，缺陷因此现形**：客户端根本不听浏览器自己的 `offline`/`online`，于是状态栏在一条已经消失的链路上继续显示「已连接」，操作员是靠「打字对方收不到」发现的。修法：`offline` 时主动断开会话，让重连路径有事可 react（留着 socket 等于等一个网络恢复后才会来的 close），`online` 时启动重连，并用「有连接可恢复」把关，避免从未连上的页面启动一个 没有 config 的重连。断言两条：网络恢复后客户端真的再拨一次（第二个 socket 即证据），以及不把这次断网记成 `reconnect_exhausted`。红绿：删掉 online 处理即红。剩余：代理切换仍未验证 |
| QD-908 | 2h+ 桌面长稳和几百路观看 | `[~]` | 200 路 7200 秒正式跑已两次执行完毕并拿到带时间序列的 `WALL_SOAK_SUMMARY`：时长、118 次采样、三个采样 iframe 帧持续推进、零页面错误均通过；`mainFps` 0.71~0.77（阈值 1）与 `heapGrowthMb` 535~554 MiB（阈值 256）未过；同参数 42 秒短跑对照为 9.17 FPS 和 -118 MiB。按采样打点后退化形状确定：**FPS 在 267~681 秒之间阶跃崩塌**（前 267 秒稳定 7~8.7 FPS，681 秒已跌到 3.51，之后贴 0.4），不是渐进；**DOM 节点数在 267 秒见顶 9241 后不再增长**，稳定在 9132~9137，因此节点增长不是原因（上一轮的猜测被推翻）；**heap 增长与 FPS 崩塌是两条独立劣化**——FPS 归零后 heap 仍以约每分钟 3 MB 线性增长，说明分配源不依赖主线程帧循环。headroom 已从 1 小时提到 2 小时使跑完不被 `node:test` 掐断。**不调阈值**：调松会把 QD-908 要抓的退化藏掉。`dd79ef2` 补上加密采样能力：`js/soak/samplingPlan.js` 决定任一时刻用哪个采样间隔，密集窗口半开区间 `[from, to)`，不覆盖任何时间的窗口直接丢弃而不是半配置——否则摘要会声称有密集窗口而实际全程粗粒度，那是一份误导性记录；密集间隔若不细于粗粒度也拒绝，因为花了采样预算却买不到分辨率。规则抽成模块而不是写在 E2E 里，因为规则才是值得测的部分，而埋在两小时测试里的函数不跑两小时测不到。9 条单测覆盖，含两处边界（半开区间、零宽窗口丢弃）；红绿两次：把上界改成闭区间、接受零宽窗口，各自被对应测试抓住。20 秒实测：窗口 8~14 秒、密集 100ms，73 个样本中 59 个落在窗口内；关闭时 12 个样本，默认路径逐字节不变。正式跑需在 267~681 秒窗口内设 `QD_WALL_DENSE_FROM_MS=267000 QD_WALL_DENSE_TO_MS=681000 QD_WALL_DENSE_SAMPLE_MS=100`，并需夜间低负载窗口（白天 load 3~5.6，会污染 FPS 数据）。**本轮实证推翻两个前提**。其一，劣化不是规模无关的：40 路 90 秒全程密集采样（180 点）heap 仅 -0.4 MiB、FPS 稳在 52，完全不复现，所以「在小规模上找分配源」这条路不存在，必须用 200 路。其二，密集采样在 200 路下不可达：单次采样（`page.evaluate` + CDP `Performance.getMetrics`）实测约 6.5~7 秒，200 ms 的密集间隔被超出 14 倍，配置 240 点的窗口只产出 9 点。`denseResolution()` 因此把「配置的间隔」和「达到的间隔」分别报进摘要——一份声称密集而实际全程粗粒度的记录，比没有这个功能更糟。60 路实测可达（142 点、实际 211ms/请求 200ms，`achievable: true`），200 路正确报 `false`。**heap 增长被证明是测量噪声，不是劣化**。`QD_WALL_PEAK_TRACK_MS`（默认 1000，可关）用一个只读 `JSHeapUsedSize` 的廉价心跳在整轮内独立持有峰值，不参与完整采样所以不被其开销拖累。同样 200 路三次跑：`heapGrowthMb` 分别为 **-156.19、-11.48、+295.50**，峰值 384 MiB 时而高于终值 250 MiB、时而低于终值 13 MiB。一个在相同参数下摆动 300 MiB 的指标测的是噪声，不是泄漏。这直接推翻此前「heap 增长与 FPS 崩塌是两条独立劣化」的论断——那条结论建立在这个数字上。峰值读数为 118~175 次，证明跟踪器本身跑得起来，问题在被跟踪的指标。**门禁含义**：`heapGrowthMb` 不能再作为 QD-908 的通过条件，继续用它会把随机波动当成回归、也会把真回归藏进噪声。峰值门禁已落地并抓到真缺陷。门禁改判峰值（整轮最大值）而非首末差，两者仍都报进摘要。**200 路峰值 heap 超线性增长**：40 路 33.1 MiB、100 路 75.3 MiB、200 路 369~384 MiB——规模翻倍而峰值涨 4 倍，这不是每路恒定开销。根因是 `WallPage.vue` 的 `tiles` computed 里对每个选中设备做 `devices.find(...)`，即 O(n²)，而屏幕墙本就是几百瓦片的设计，每次设备轮询和每次 live 状态更新都会全量重算。`1f5e7f1` 改为先建索引再一次遍历（`js/wall/tileModel.js`）：200 路峰值从 369~384 MiB 降到 **117.3 MiB**，峰值增长从 129~274 MiB 降到约 0，门禁由 FAIL 转 PASS。测试计数「设备列表被搜索几次」而非计时——计时守卫试过但无效：V8 对 400 元素 `find` 优化到二次版只花 9.1ms（索引版 3.6ms），任何值得设的阈值两者都能过，那会是一个永远绿的守卫；把 `find` 放回去该测试立刻红。FPS 与 `pageErrors` 门禁始终有效，未动。剩余：200 路 7200 秒夜间正式跑仍未做。 |
| QD-909 | 安全审计：文件、剪贴板、权限 | `[~]` | 访问、会话和文件传输已有 attributable 上报；`e04451c` 补剪贴板：`POST /v1/me/clipboard-audits` 记方向、MIME 类型和字节数，**契约里没有存放文本的字段**——剪贴板内容照例是密码、恢复码或私信，「14:03 拷走 1.2 KB 文本」足以回答调查问题，而存文本会把审计链变成每个后来读库者都继承的凭据库；客户端 builder 只取文本用于测量，从构造上就没有内容进入审计行的路径；真机 E2E 直接回读 SQLite 文件，断言行存在、带字节数、且既不含文本也不含片段；本地写入失败也按 failed 上报（字节确实过了数据通道）。读侧已补：此前 owner 能写文件/剪贴板审计行，却只能经 `/v1/admin` 读回——「14:03 我的控制器拷走了什么」这个问题，唯一能据此行动的人反而问不到答案，审计链成了只写日志。新增 `GET /v1/me/audit-logs`：按调用者自己的 actor id 收敛，两个写端点本就拒绝非属主设备，所以行上的 actor 即该设备 owner，这个收敛不会变成别人传输的视图；`action` 是白名单而非自由匹配——不在本链路上的动作返回 400 而不是空页，因为「你什么都没做」不该是拼写错误能产出的结论；分页沿用通用 cursor。`details` 以 JSON 字符串返回，写端点「永不存剪贴板文本」的性质因此对客户端可见，而不是被类型化字段藏起来。真机 E2E 在同服务器上 seed 第二个账号，证明「我的行」确实只含我的；红绿在单测与 E2E 两级都做过。客户端入口本轮补齐：此前读侧只有服务端 API，按第 15 节「有用户可发现的入口」这一项本不该算完成。新增 `src/api/auditLogModel.js`（解析 `details` JSON 字符串、四类 action 白名单、标签只说「传了什么、多少字节」而不说内容）、`src/composables/useSessionAudit.js`（读账号自己的传输审计，失败与空结果是两种不同呈现）、`userApi.fetchAuditLogs`（Vue 侧与 legacy `js/api` 两侧都补），以及 `SessionsPage.vue` 的「传输审计」面板（按需加载，不重复读，中英文案）。**一个隐蔽缺陷**：composable 返回对象里的 ref 在 Vue 模板中不会自动解包，`audit.loading` 读到的是 Ref 包装器本身（恒为真），面板于是永远转圈且无任何报错；改为在 setup 里显式拆成顶层 `auditLoading` / `auditError` / `auditRows`。验证：单测 11 + 7、客户端 E2E 4、UI E2E 4（真实浏览器三态：有行/失败/空，并断言失败不渲染成空列表）；`npm test` 187/187，生产构建通过。七套 SDK 已随源 spec 重新生成，freshness 门禁七种语言全部通过；`sdk/go` vet/build、Python 导入、Java package、Kotlin build 本机实跑通过，C#/PHP 需 dotnet 与 php，交 CI。**WebClient 测试接入 CI**：`webclient-verify.yml` 分两个 job——unit（`npm test` + 7 个 mock/vite 浏览器 E2E）与 integration（7 个需真实 Redis + `go build` 的服务端 E2E）。此前 macOS/Windows workflow 把 `WebClient/**` 列为触发路径却只构建不测试，本轮多轮新增的 20+ 单测与 6 个 E2E 只有本机证据。分层按依赖划：5 个纯 mock、5 个需 vite、5 个需 redis+go；`wall-soak` 是两小时长跑，显式排除在常规流水线外。两个 job 均已在 runner 上实测通过（run `40266200476`，head `1ce6cb5`：unit 1m49s、integration 2m34s）。接入过程连修四个 runner 专有阻塞，本机全部不可见：①`WebClient/package-lock.json` 被 `.gitignore` 的无前缀 `package-lock.json` 规则匹配、从未提交，`npm ci` 与 npm cache 同时失效，改用 `npm install`；②`go build` 冷缓存下载整个模块图，输出超出 16 MiB 的 `maxBuffer`，改为 workflow 内 `go mod download` 预热 + 测试侧 64 MiB；③`//go:embed web/dist/*` 在 fresh checkout 无文件可嵌导致编译失败，照 `signaling-server-verify.yml` 既有做法补占位 `index.html`；④`control-mode.e2e.mjs` 服务的是 `dist/` 而非 vite dev server，job 不构建就无从加载，补 `npm run build`。四个都不是测试本身的问题，全部只在 fresh checkout 上显形——这正是接线它的全部理由。剩余：权限侧边界随 QD-115 收口。**`58f49f5` 修掉一个并发洞**：`LogActorOnce` 承诺同一 (actor, action, resource_type, resource_id) 只接受一次，文件传输与剪贴板上报都依赖它——控制器重试时不能让同一 transfer id 在审计链上多出一行。原实现是「先 count 再 insert」而背后什么都没有，两个并发上报同时看到 0 就同时写；新测试三次运行三次复现，失败信息精确到「两个调用方都被告知写成功了」。修法是加部分唯一索引，让第二次 insert 失败，而这个失败正是调用方所问问题的答案，不是要上报的错误。**索引的scope 改了两轮，两轮都是测试发现的而不是读代码发现的**：①只覆盖 actor 行——管理员本就会对同一资源重复同一动作，唯一约束会静默丢掉那些行，先红的是播种这种行状的保留测试；②只覆盖两个 transfer action——`LogActorDeduped` 靠时间窗对同一资源写多行，覆盖所有 actor 行的索引等于直接禁掉媒体审计，抓到它的是 dedupe 测试。两个功能在同一个元组上要相反的东西，而此前只有一件被写下来。action 列表因此提成命名常量 `models.AuditOnceActions`，并有测试在任一 action 缺失时失败，防止 scope 悄悄缩到「什么都不覆盖」。SQLite 不能建部分索引，给它的是**同一条语句**而不是变通——两个形状就是两个保证，该测的那个才该上线。ensure 调用放在 AutoMigrate 分支之外，手工管 schema 的运维同样拿得到。验证：全新库全量 9 包绿（含前两个索引形状曾经弄坏的那些测试）、WebClient 263/263 未触及仍绿、七套 SDK 门禁绿、gofmt 干净；红绿双向。**`SignalingServer verify` 从 10-16 起连败，本轮修复后首次全绿**（run `40270895099`，head `8ec5fed`，四个 job 全过）。根因不是某个功能的回归，而是 `gorm.io/driver/postgres` 的一个驱动缺陷：`Migrator.GetRows` 在 `PrepareStmt` 关闭时把 `pgx.QueryExecModeSimpleProtocol` 当作查询参数塞进 `SELECT * FROM "t" LIMIT 1`——这条语句没有任何占位符，pgx 的 sanitize 于是报 `insufficient arguments`。该路径只在**表已存在**时走到（新表直接 CREATE TABLE），所以 CI 那个被反复 migrate 的 scratch 库必炸，而开发者本机不设 `QD_TEST_DATABASE_URL`、全部测试 skip，于是从没见过：这正是它红了十天没人发现的原因。`bc47d45`/`5ea85dd`/`dba6721`/`8ec5fed` 四连修：①21 处重复的打开逻辑收敛为 `openTestPostgres` + `migrateTestSchema`（migrate 走 prepared 会话，因为整句 multi-statement 的 `001_init.sql` 会被 prepared 会话拒绝）；②`openTestPostgres` 补 `t.Cleanup` 关闭连接池——此前每个测试各开一个池且从不关闭，共享库连接数耗尽后，后跑的测试失败在 connect 而非被测逻辑上（service 包 120 → 22 个失败）；③**产品 bug**：`RemoteSessionService.Open` 声称按 session id 幂等，唯一冲突时回查已有行，但回查在同一个事务里，而失败的 INSERT 在 Postgres 上早已把事务中止，于是每次重试的握手都失败（SQLite 不会中止事务，所以本地与 SQLite 测试从未暴露）；④**同一个驱动缺陷也在生产路径上**：`OpenPostgreSQL` 用裸 `gorm.Config`，`cmd/signaling` 逐个模型 migrate 且每失败只打 warning 继续——后果是全新库能建表、看似健康，而既有安装从此再也收不到 schema 更新。它最终以 conformance smoke 的 `relation "devices" does not exist` 暴露。新增 `database.AutoMigrate`（prepared 会话），并补上 `internal/database` 本就缺失的 Postgres 测试。证据：本机 fresh 库 + CI DSN 全包 `go test ./...` 绿；server 连续启动两次（第二次即原先必败的「表已存在」路径）均 0 warning；CI run `40270895099` 四 job 全过。 |

| QD-910 | 需求、上游与实现持续对账 | `[~]` | 每次上游合并、issue/PR 变化或功能完成时，同步本清单、状态证据和上游贡献决策 |
| QD-513 | 事件消费链路四层对账 | `[x]` | 上一轮用「服务端事件清单 vs `userSync` 分发 switch」的差集在 WebClient 层找到 QD-512，本轮把同一方法推到其余三层并逐条查证，结论是**其余三层无同类缺口**，此处记录判据与边界，避免重复排查。**Qt 层**：33 个服务端域事件中 27 个在 `QuickDesk/src` 有对应处理，6 个逐字未匹配——其中 `favorite.added` / `favorite.updated` 由 `CloudDeviceManager` 的 `type.startsWith("favorite.")` 前缀分支覆盖（该分支内还有注释说明 added/updated 要 refetch），是假缺口；`device.access.session_started`、`device.access_grant.changed`、`device.access_grant.used`、`turn.config.changed` 四个的生产者全在 handler 层且都列入 `webhookEligibleTypes`，设计目标就是 webhook/通知而非客户端实时刷新，Qt 不需要消费。**SDK 层**：不涉及事件类型（那是 OpenAPI 契约层），无此形状缺口。**管理端层**：`/v1/realtime/events` 是 per-user 流，管理端用 admin token（另一套身份体系），要做实时刷新需先建 admin 事件流，属完整工程而非切片；现有 14 个页面中仅 HomePage 与 RemoteSessionsPage 用轮询，其余手动刷新，符合运维工具的使用频率。**WebClient 残余**：`userSync` 派发 12 个 CustomEvent，8 个有监听；未监听的 `connected` / `disconnected` / `error` 是连接生命周期，`session-revoked` 的处理走的是 `userApi.handleServerRevoked()` → `_sessionEnded()` → `_onSessionEndedHandler` 回调这条更完整的通路，CustomEvent 属冗余而非缺口。**方法学教训（本轮最值得记的一条）**：这一轮两次差点据假信号动手。第一次用 grep 抓 Qt 事件只得到 12 个、看起来有 21 个缺口，实际是正则没匹配 `QStringLiteral("...")` 形式；改用 Python 后仍漏，最后靠「搜带引号子串」才得到可信的 27 个。第二次逐字搜 views/ 目录判断消费方，全为 0，实际消费方监听的是 CustomEvent 名而非原始 type。两次错误的修正是换判据后重新取证，而不是在错误数据上继续推理——若信了第一个信号，会去重写 6 条本来正常的路径。**结论已固化为测试**：`WebClient/test/eventCoverage.test.mjs` 不解析任何代码，而是直接驱动 `userSync._onMessage` 并把 `serverEventTypes()`（从 `event_bus.go` 的事件常量正则提取）逐条灌进去，用包装 `dispatchEvent` 观察输出，断言 10 个「有页面在看」的事件必须派发、且派发名不得带点（点说明 wire type 漏进了 CustomEvent 命名空间）。写它时又踩了一次同一个坑：同一 `server_rev` 喂 33 帧，第一帧消费掉 rev、其余全被 `_shouldApplyFrame` 当成已应用而丢弃，于是所有事件都「没派发」——守卫本身是对的（重放不该重跑事件），测试去跟它对抗就什么都学不到；改为每帧一个递增 rev 才对。红绿已做：把 `recording-changed` 的派发条件改成 `recording_id !== 1`，测试立刻报 `recording.created reaches no page`，即它确实能抓住 QD-512 那一类回归。 |
| QD-911 | 上游私密安全披露入口 | `[~]` | fork 提交 `3021ca9` 与上游候选（已 rebase 到 `upstream/dev@5db316c` 成为 `68d4a38`）新增 `SECURITY.md`，覆盖私密邮箱兜底、公开 issue 禁令、范围、协调披露和 maintainer 检查项；已创建上游 PR #33（OPEN）。PR 说明已明确：加文件不会启用 GitHub Private Vulnerability Reporting，那是只有维护者能开的仓库设置，开启前兜底邮箱是唯一私密路径；issue #6 仍 OPEN，该设置仍 disabled |
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
   QD-114 真实 CertVerifier 均已有需求单（`docs/dev/Host权威录制Remoting需求.md`、
   `docs/dev/自托管TLS真实CertVerifier需求.md`、`docs/dev/图片剪贴板Remoting需求.md`）。
   至此第 4 项列出的 Remoting 需求单已全部成文。
   每项写清协议、宿主生命周期、权限、
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
- 上游基线（2026-10-27 复核）：`master` = `5969ae3`（PR #22 的线程亲和性修复）、
  `dev` = `88d4d36`（PR #23 会话录像与回放，2026-10-25 23:31 合并，7 个提交
  +5822 行）。两个分支已不同线，`master` 领先 `f6651f0` 一个提交。
  **PR #23 已核对无功能缺口**：那 7 个提交落在录像子系统，而 fork 在
  `5db316c` 之后自己实现了同一片区域并且是**严格超集**——
  `session_recording_service.go` 的上游符号全部在 fork 里，fork 还多
  `WithOutbox` / `AttachArtifactVerified` / `ListForSessionIDs` /
  `VerifyArtifactDigest` 等 13 个；`recording_ticket.go` 与
  `session_recording_lifecycle.go` 同样是超集；
  `QuickDesk/src/manager/DeviceRecorder.cpp` 两边逐字节相同；
  `recording_upload_ticket.go`、`session_recording_resume.go`、
  `RecordingManager.cpp`、`recordinglifecycle.cpp`、`recording-upload.js`
  只存在于 fork。所以这不是「fork 缺上游功能」而是两条线并行长出同一区域，
  整树合并只会带来 22 个冲突文件而没有新增能力；U0 闸门对该 PR 的结论
  由「等待同步」改为「已确认超集，无需回灌」。PR #23 里唯一看似 fork 缺失的
  是 `event_bus_test.go` 的 5 个测试，实为 fork 侧新增（上游 1 个，fork 6 个）。
- 上游 Roadmap：Windows、macOS、P2P/TURN、MCP、虚拟屏、隐私屏、文件传输、音频、无人值守已完成；Linux、iOS/Android、Address book & device groups 未完成
- `Multi-tab multi-connection` 在上游已勾选；QD-003 补的是同一设备多个标签的控制/音频互斥，不重复“多连接”能力
- 维护计划：issue #21，维护者明确“持续维护、暂缓更新、重点偏 MCP、欢迎从 dev 切分支贡献、Remoting 改动由维护者处理”
- 当前 11 个 open issue：#6、#7、#9、#10、#13、#18、#19、#20、#21、#29、#30；其中
  #13、#18、#19、#20、#29、#30 是用户问题或 Roadmap 入口，#6、#10、#19、#20 已经拆入
  QD-911、QD-115、QD-212/213/214、QD-114 的交付队列。#17 已不在 open 列表——维护者在
  PR #22 合并后关闭了它，QD-113 随之满足「上游已修且已回灌」。新增的 #29（信令服务器 WSS）
  与 #30（与 Local Commander 配合的说明）尚未拆解，属 `Upstream-Open`。
- 当前 8 个 open PR：#23 录像与回放、#25 自托管控制面/OpenAPI/SDK、#26 file-ops 分块与
  版本化回收、#27 WebClient 远程文件管理器、#31 MCP 存储锁自死锁、#32 Windows 发布依赖校验、
  #33 私密安全披露策略
  （均 base `dev`），#28 意大利语翻译与永久访问密码（base `master`，上游维护者自己的提交）。
  #22 和 #24 已分别合并为 `5969ae3` 和 `5db316c`。
  其中 #26、#27、#31、#32、#33 是本 fork 回贡的 QD-002/QD-403、MCP 死锁、QD-215 与 QD-911。
- 2026-10-25 00:40 复核：上游 `dev` 为 `5db316c`，11 个 open issue、8 个 open PR。
  新增 #29（WSS 何时支持，与 #20 同一诉求）与 #30（Local Commander "Works with" 互链请求，
  第三方项目，属文档决策）。远端写入已确认可用：PR #31 与 #32 均创建成功，
  此前记录"push/PR 被策略阻塞"不再成立。
- 2026-10-27 12:40 复核：上游 `dev` 仍为 `88d4d36`（自 PR #23 合并后无新提交），
  `master` 仍为 `5969ae3`，与 10-25 的记录无漂移。本 fork 的 9 个候选分支
  （PR #31/#32/#33/#34 加上尚未开 PR 的 QD-212/213/114/115/214）
  相对 `upstream/dev` 均 behind 7~8 个提交，但 `git merge-tree --write-tree` 逐个实测
  **零冲突**，与 GitHub 报的 `mergeable=MERGEABLE` / `mergeStateStatus=CLEAN` 一致。
  「behind N」在这里只是"基线旧"，不是"需要 rebase"：那 7 个提交落在录像子系统，
  而这些候选碰的文件不重叠。`codex/upstream-qd-002-file-ops` 另在隔离目录
  `cargo build --offline` + `cargo test --offline` 实跑通过（4 passed），
  确认候选本身健康而非仅"git 说能合"。
- 历史记录 2026-10-17 23:04：上游两个分支为 `f6651f0`，10 个 open issue、4 个 open PR
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
| #20 自托管 WSS 失败 | QD-114 | 双轨推进 | `UP-Ready` + `UP-Remoting` | Qt/Windows SPKI 已落地；真实 CertVerifier 需求单已成文（`docs/dev/自托管TLS真实CertVerifier需求.md`），交 Remoting 实现 |
| #29 WSS 何时支持 | QD-114 | 与 #20 合并 | `Upstream-Open` | 与 #20 是同一诉求的第二次提问，说明自托管 TLS 是真实阻塞；根因同上，不单独立项 |
| #30 "Works with" 互链请求 | QD-910 | 待维护者决定 | `Upstream-Open` | 第三方项目（Local Commander，MIT）请求 README/docs 互链；属文档决策，不涉及代码，等维护者表态 |
| #19 设备列表问题 | QD-212、QD-213、QD-214 | 拆分处理 | `UP-Ready` + `UP-Remoting` | 本机标识、访问码回源分别提；上传错误诊断可独立提，Windows 用户上下文根因交 Host/Remoting |
| #18 连接历史 | QD-206 | 不重复实现 | `Upstream-Open` | 核验现有 WebClient/Qt 入口，整理版本证据后回复 issue |
| #17 Auto Approve 失效 | QD-113 | 已合并，测试已补跑 | `UP-PR#22` | `5969ae3` 已进 `upstream/master`，fork `3d44dfc` 已回灌；`trust_handler_test` 已在 Windows CI 实跑通过（`f956d17`，17/17），QD-113 转 `[x]`；不在 fork 重写第二套线程亲和性修复 |
| #13 Linux 支持 | QD-318 | 暂缓 | `Upstream-Open` | 保留跨平台接口，不承诺当前迭代实现 |
| #10 信令服务设备密钥信任 | QD-115 | 设计完成，待协议 | `UP-Ready` + `Upstream-Open` | 安全边界候选 `1ca2543` 可直接提 PR；实现侧需要 Host 身份密钥、Owner 签名 grant 和 Host 本地验证 |
| #9 主机连接失败 | QD-216 | 先排障 | `Upstream-Open` | 收集 host/signaling/TLS 日志，确认可复现 bug 后再提单 |
| #7 Windows Qt 插件启动失败 | QD-215 | PR 已提 | `UP-Ready` | 候选 rebase 到 `upstream/dev@5db316c` 成为 `8385856`，已创建上游 PR #32；仍未取得原始坏包 |
| #6 私密安全披露入口 | QD-911 | PR 已提 | `UP-Ready` | `codex/upstream-qd-911-security@68d4a38` 只新增 `SECURITY.md`，已创建上游 PR #33；仍需维护者启用 Private Vulnerability Reporting，否则兜底邮箱是唯一私密路径 |
| PR #22 自动审批线程亲和性 | QD-113 | 已合并、已回灌、测试已跑 | `UP-PR#22` | `5969ae3` 已于 2026-10-21 合并；fork 经 `3d44dfc` 合入 `upstream/master`，与上游零差异；CTest 补跑已完成，`trust_handler_thread_affinity` 在 Windows CI 通过（17/17） |
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
| PR #22 auto-approve thread affinity | MERGED 2026-10-21 为 `5969ae3`，基于 master | QD-113 | fork `3d44dfc` 已回灌，与上游零差异 | 不要重复提交 issue #17；补跑 `test/trust` CTest |

### 16.6 上游 issue / Roadmap 映射

| 上游项 | 类型 | 涉及 QD | 判断 | 动作 |
|---|---|---|---|---|
| #21 维护计划与参与维护意向 | 维护策略 | QD-001、QD-609 | 上游优先 MCP、从 dev 贡献、Remoting 由维护者改 | 所有新功能按 dev 最小分支准备；涉及 Remoting 先提需求 |
| #20 自托管 HTTPS/WSS 失败 | Bug / 自托管 | QD-114 | 证书验证器在 Remoting 内未绑定；Qt-only SPKI 已支持子进程和服务安装参数 | 推送候选并补 Windows/Qt 回归；长期向 Remoting 提 CertVerifier 修复 |
| #18 连接历史列表 | Feature | QD-206 | WebClient 已有 `ConnectionHistory`，Qt 已有 `RemoteDeviceManager` 历史下拉、删除和重连；issue 仍开放 | 在 issue 回复现有入口和版本证据，建议关闭；不再重复实现 |
| #19 设备列表相关问题 | Bug cluster | QD-212、QD-213、QD-214 | 本机标识和访问码回源已完成；上传日志确认 Windows Host 无法取得交互用户 token，普通用户为 `0x522/type=2`，SYSTEM/session 0 为 `type=7` | 推送三个独立诊断候选；QD-214 根因按 Host 服务或 Remoting 用户上下文单独提交，不混成一个“设备列表”大修 |
| #17 自动审批被忽略 | Bug | QD-113 | 已由 PR #22 覆盖；`5969ae3` 已合并进 `upstream/master`，fork `3d44dfc` 已回灌且与上游零差异 | 不再重复实现；补跑上游 `test/trust` CTest 后可建议关闭 issue |
| #13 Linux 支持 | Roadmap | QD-318 | 上游 Roadmap 已列出，本 fork 暂缓 | 保持跨平台接口，不在当前迭代投入 Linux 被控端 |
| #10 信令服务器设备密钥信任 | Security | QD-115 | device_secret 只有哈希；但明文 access_code + server token/审批意味着被攻陷服务器仍可冒充 Controller | 推送 `signal-trust-model` 候选并在 issue 讨论协议升级；不能把哈希化 device_secret 当作已修复 |
| #9 主机连接失败 | Support / Bug | QD-216 | 需要真实环境日志，不宜直接猜实现 | 收集 host/signaling/TLS 日志后再拆 bug |
| #7 Windows 启动 Qt platform plugin 错误 | Packaging | QD-215 | Qt 插件/运行库打包问题；Windows publish/package 与缺 DLL 负向测试已通过，候选 `f449b87` 仅 3 个文件 | 推送候选并开 issue #7 PR；取得原始坏包后做直接复现 |
| #6 私密安全披露入口 | Process | QD-911 | 候选已 rebase 为 `68d4a38` 并创建上游 PR #33；上游仍无该文件且 Private Vulnerability Reporting 关闭 | 请维护者合并 PR #33 并启用 GitHub 私密报告；不在公开 issue 贴利用细节 |

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
| 17 | `feat(audit): record controller file transfers` | QD-111 / QD-408 | 候选 `codex/upstream-qd-111-file-transfer-audit@c57a2f4` 基于 PR #25 的 `8998466`，44 个文件，只含文件传输审计和四套基线 SDK；候选自身 Go/SDK/Web 构建通过，Qt 源码编译到既有 `AGL` 链接限制 | 推送后以 `feat/signaling-control-plane` 为基线创建 stacked PR；PR #25 合入后再 rebase 到 `upstream/dev` |

QD-002 / QD-403 的上游候选已经推送并开 PR：

- `codex/upstream-qd-002-file-ops` = `e106be4`：基于 `upstream/dev@f6651f0`，只含 host
  `file-ops` 的分块读写、默认目录、版本保留软删除、SKILL.md 和 Rust 测试；
  上游 PR <https://github.com/barry-ran/QuickDesk/pull/26>，OPEN / MERGEABLE / CLEAN。
- `codex/upstream-qd-403-webclient` = `796e4cb`：stacked 在 file-ops 分支上，只含浏览器
  `qd-skill-bridge` 客户端、Web 文件管理器 UI、remote-main 接线、样式、双语和 Node 测试；
  上游 PR <https://github.com/barry-ran/QuickDesk/pull/27>，当前 base 为 `dev`。
- 验证基线：候选分支上 file-ops 4/4、Clippy `-D warnings`、WebClient 12/12、生产构建、
  接线检查（完整能力集打开菜单入口）和 diff check 全部通过；fork 主线另跑 62/62 与
  桌面/移动 Chrome 烟测。
- 上游 `#22`~`#25` 的既有候选分支已全部推送到 fork，等维护者审查。

QD-003 的上游候选已经单独准备：

- 分支：`codex/upstream-cross-tab-coordination`
- 基线：`upstream/dev` = `f6651f0`
- 本地提交：`93eee37`
- 范围：仅 `TabCoordinator`、WebClient 接入、双语文案、Node 测试、真实 Chrome
  多标签 E2E 和 ESM 构建兼容
- 验证：`npm test` 6/6、`npm run test:e2e:tab-coordinator` 1/1、`npm run build`
  和 `git diff --check` 均通过
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
