# QuickDesk 功能总清单

> 本文件是 QuickDesk fork 后续产品开发的唯一功能清单。旧 Go-Remote 仓库不再作为需求来源、实现基线或兼容目标；这里列出的范围以当前仓库代码、测试和已验收文档为准。

## 1. 状态与规则

| 状态 | 含义 |
|---|---|
| `[x]` | 已完成，代码已落地并有测试、构建或真机验证证据 |
| `[~]` | 部分完成，底层已具备但产品入口、跨端消费或边界验证仍缺 |
| `[ ]` | 未开始 |
| `[!]` | 被外部依赖或协议改造阻塞 |
| `[>]` | 上游 Roadmap 已列出，本 fork 当前暂缓 |
| `[-]` | 明确不做 |

维护规则：

- 功能完成时同步更新状态、验收标准和证据路径。
- 不以“底层已经有接口”等同于“产品已完成”；必须把用户入口、错误处理和验证一起收口。
- 核心媒体与输入不重写，继续复用 Chromium Remoting、WebRTC、ICE、STUN、TURN。
- Windows 和 macOS 当前优先；Linux、iOS/Android 属于上游 Roadmap，本 fork 暂缓但不删除贡献接口；RDP 明确不做。
- 每项功能都要判断上游贡献方式：`UP-PR`、`UP-Ready`、`UP-Remoting`、`Fork-only` 或 `Upstream-Open`。
- 任何新功能先写失败测试，再实现，最后补构建或端到端验证。

## 2. 当前主线

- 仓库：`go-remote-next`
- 主分支：`dev`
- 上游基线：`barry-ran/QuickDesk` 的 `dev` 和 `master` 当前均为 `f6651f0`
- 上游维护口径：持续维护但暂缓更新，下一阶段重点偏 MCP；外部贡献从 `dev` 切分支，完成后合入 `dev`，维护者定期合入 `master`
- Remoting 边界：`quickdesk-remoting` 需要编译整个 Chromium，涉及 Remoting 的改动优先向上游提需求，不要求贡献者本地编译
- 本轮起点：`c4091701add83ad5aa92e36511ad5db3f4a7e813`
- 本轮合并：`2f61f8e`（QD-003 多标签控制与音频抢占）
- 本轮功能：`a48d793`（QD-114 Qt/Windows 服务 SPKI 公网自托管兼容）
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

- 上游核对：`upstream/dev` 与 `upstream/master` 均为 `f6651f0`；当前上游 open issues 10 个、open PR 4 个。
- WebClient `npm test`：25/25 通过。
- WebClient `npm run build`：通过，JS 348.48 kB，gzip 112.14 kB。
- Chrome 同源双标签实测：后开标签接管控制后，前一个标签释放控制；接管音频后，前一个标签释放音频。
- QD-114 SPKI 纯 C++ 单测：通过；覆盖标准/URL-safe Base64、无 padding、重复项、
  非法项和 Chromium 参数构造。
- QD-114 上游候选：`codex/upstream-qd-114-spki` = `13559e6`，基于 `upstream/dev`，
  17 个文件；当前机器没有 CMake/Qt 开发环境，完整 Qt 构建和 Windows 安装器编译待补。
- QD-005 功能提交：`cae77ff`。Qt 客户端已有启动后自动检查和每 6 小时复查、手动检查、
  更新提示、下载进度/取消、大小与 SHA-256 校验、可选版本推迟、强制更新、同源 API key
  下载和安装器启动；非同一 origin 的下载不会泄漏 API key。
- QD-005 测试：`quickdesk_update_support_test` 与 `quickdesk_update_manager_test` 通过；
  后者使用真实本地 HTTP 服务覆盖检查、下载、摘要不匹配拒绝、残件清理和跨源凭据隔离。
- QuickDesk 完整 Qt 6.8.3 Debug 构建通过，`QuickDesk_qmllint` 退出 0；现有旧 QML
  未解析类型和布局告警仍在，本轮新增更新对话框没有阻断告警。三个 CTest 目标全部通过。
- `git diff --check`：通过。
- 远端推送仍被当前执行策略拒绝，本地 `dev` 尚未同步到 GitHub。

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
| QD-108 | 二次访问审批 | `[~]` | 服务端和 WebClient 已完成；Qt 审批入口和实时消费待补 |
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
| QD-204 | 设备分组与批量操作 | `[~]` | 上游 Roadmap 的 Address book & device groups；管理后台已有基础，用户端入口仍可补 |
| QD-205 | 在线状态与心跳 | `[x]` | Redis presence + heartbeat |
| QD-206 | 连接历史与快速连接 | `[x]` | 上游 `dev` 与 fork 都已有 `ConnectionHistory` 和连接历史 UI；issue #18 看起来已过期，先验证再关闭，不重复实现 |
| QD-207 | 会话创建、运行、结束、清理 | `[x]` | `remote_sessions` 控制面已落地 |
| QD-208 | 会话详情、录像、访问日志聚合 | `[x]` | WebClient 和 API 已完成 |
| QD-209 | 单会话踢人、按设备批量清场 | `[~]` | 服务端和 Web 已完成；Qt 真机回归待补 |
| QD-210 | 跨实例会话撤销 | `[~]` | 已按实例寻址；Redis pub/sub 不是可靠队列 |
| QD-211 | 撤销冷却期 | `[x]` | 支持窗口和递增冷却 |
| QD-212 | 设备列表标识本机 | `[ ]` | 上游 issue #19；当前设备需要明确的 “本机” 身份 |
| QD-213 | 访问码不可用回归 | `[ ]` | 上游 issue #19；远程其他设备提示访问码不可用，需要复现宿主与权限链路 |
| QD-214 | 文件上传 Host error type=2 | `[ ]` | 上游 issue #19；需要定位 Remoting 文件传输错误码和宿主机端失败路径 |
| QD-215 | Windows 启动 Qt platform plugin 错误 | `[ ]` | 上游 issue #7；属安装包/运行时打包问题 |
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
| QD-508 | Qt 录像回放页 | `[ ]` | Qt 目前没有完整回放页 |
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
| QD-606 | Qt 通知规则管理页 | `[ ]` | Qt 端待补 |
| QD-607 | 计划任务 | `[~]` | 服务端、WebClient 和重试已完成；Qt/客户端动作待补 |
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
| QD-706 | Qt 会话页 | `[~]` | 页面已有；实时事件消费仍靠轮询 |
| QD-707 | Qt 文件管理器 | `[x]` | host `file-ops` skill bridge |
| QD-708 | Qt 通知/计划/审批页 | `[ ]` | 三个管理页未完成 |
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

## 13. 明确不做

- `[-]` RDP：会锁被控端本机会话。
- `[-]` 重写 Chromium Remoting 媒体内核。
- `[-]` 自研 WebRTC / ICE / STUN / TURN。
- `[>]` Linux 被控端：上游 Roadmap 已列出；本 fork 暂缓，接口和验证基线保留。
- `[>]` iOS/Android 原生控制端：上游 Roadmap 已列出；本 fork 暂缓。
- `[-]` 多租户、SSO、计费。
- `[-]` 通用插件市场和完整插件权限模型。

## 14. 推荐执行顺序

1. 保持上游 open PR #22 / #23 / #24 / #25 同步，不重复实现。
2. 推送 QD-114 上游候选 `codex/upstream-qd-114-spki` 并创建 issue #20 PR；补齐 Qt 构建、
   Windows 服务安装和真实 `wss://` 回归；彻底修复向 Remoting 仓库提 CertVerifier 需求。
3. 把已完成的 QD-003 多标签控制/音频抢占整理为上游独立 PR。
4. 把 QD-206 本地连接历史按上游 issue #18 拆成最小 Web/Qt PR。
5. QD-002 Web 文件操作通道与完整文件管理器；若必须改 Remoting，改为上游需求。
6. QD-004 Host/Relay 自动录制；先向 Remoting 维护者提需求，不在 fork 内自造第二套媒体链路。
7. QD-005 客户端消费已落地，补 Windows 安装包真机升级/自动重启和 macOS DMG
   签名、公证、替换回归；上游等服务端 PR #25 合并后再拆客户端消费 PR。
8. QD-006 SQLite 轻量部署、QD-007 图片剪贴板、QD-708 Qt 管理页和实时事件。
9. QD-604 指标告警、QD-908 业务级长稳，以及上游 Roadmap 的地址簿、Linux、移动端。

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

上游贡献状态：

| 标记 | 含义 |
|---|---|
| `UP-PR#n` | 已经在上游 open PR 中，先同步审查，不重复提交 |
| `UP-Ready` | 可作为独立最小 PR 提交到 `upstream/dev` |
| `UP-Remoting` | 需要修改 `quickdesk-remoting`，优先提需求给维护者 |
| `Fork-only` | 产品品牌、部署配置或本 fork 专属业务，不进入上游 |
| `Upstream-Open` | 上游 issue/Roadmap 仍开放，当前 fork 可有本地实现，但需要单独拆 PR |

### 16.2 上游 PR 映射

| 上游项 | 状态 | 涉及 QD | 当前处理 | 后续动作 |
|---|---|---|---|---|
| PR #25 self-hosted control plane | OPEN，MERGEABLE，CI unstable | QD-101~110、QD-201~211、QD-501、QD-601~608、QD-801~808 | 服务端、契约、SDK、smoke 和 Qt 更新消费已在本 fork；本地不重复造一套 | 等维护者审查；若要求拆分，按 access / alerting / recording / ops / update 拆 |
| PR #23 session recordings | OPEN，MERGEABLE，CI unstable | QD-501~507 | 控制端手动录制、索引、回放、生命周期已具备 | 先等 PR #23；Host 权威录制另走 `UP-Remoting` |
| PR #24 skill-host runtime path | OPEN | QD-402、QD-609 | 本 fork 已包含 PATH/stderr 修复，和上游补丁同向 | 上游合并后同步回 fork，避免二套实现 |
| PR #22 auto-approve thread affinity | OPEN，基于 master | QD-113 | 本 fork 尚未重复实现 | 等 PR #22 合并后 cherry-pick；不要重复提交 issue #17 |

### 16.3 上游 issue / Roadmap 映射

| 上游项 | 类型 | 涉及 QD | 判断 | 动作 |
|---|---|---|---|---|
| #21 维护计划与参与维护意向 | 维护策略 | QD-001、QD-609 | 上游优先 MCP、从 dev 贡献、Remoting 由维护者改 | 所有新功能按 dev 最小分支准备；涉及 Remoting 先提需求 |
| #20 自托管 HTTPS/WSS 失败 | Bug / 自托管 | QD-114 | 证书验证器在 Remoting 内未绑定；Qt-only SPKI 已支持子进程和服务安装参数 | 推送候选并补 Windows/Qt 回归；长期向 Remoting 提 CertVerifier 修复 |
| #18 连接历史列表 | Feature | QD-206 | 上游 `dev` 已有 `ConnectionHistory`、历史列表、删除和重连；issue 仍开放 | 先做完整功能核对，确认无缺口后向下游说明可关闭，不再重复实现 |
| #19 设备列表相关问题 | Bug cluster | QD-212、QD-213、QD-214 | 本机标识、访问码和上传错误码三个独立问题 | 分别复现、分别提 PR；不要混成一个“设备列表”大修 |
| #17 自动审批被忽略 | Bug | QD-113 | 已由 PR #22 覆盖 | 跟踪 PR #22，不重复实现 |
| #13 Linux 支持 | Roadmap | QD-318 | 上游 Roadmap 已列出，本 fork 暂缓 | 保持跨平台接口，不在当前迭代投入 Linux 被控端 |
| #10 信令服务器设备密钥信任 | Security | QD-115 | 服务端是否可冒充设备需要明确威胁模型 | 与上游安全维护者讨论；先写安全边界，不改协议 |
| #9 主机连接失败 | Support / Bug | QD-216 | 需要真实环境日志，不宜直接猜实现 | 收集 host/signaling/TLS 日志后再拆 bug |
| #7 Windows 启动 Qt platform plugin 错误 | Packaging | QD-215 | Qt 插件/运行库打包问题 | 在 Windows 实机复现并补安装包回归 |
| #6 私密安全披露入口 | Process | 安全流程 | 不是产品功能，但影响上游贡献可信度 | 推动 SECURITY.md 或私密披露渠道，不在公开 issue 贴利用细节 |

### 16.4 上游 Roadmap 映射

| 上游 Roadmap | 当前 fork | 状态 | 备注 |
|---|---|---|---|
| Address book & device groups | QD-204 | `[~]` | 管理后台已有基础，用户端地址簿/分组入口可做成上游通用能力 |
| Linux support | QD-318 | `[>]` | 上游已列计划，fork 暂缓；不能把 Linux 写成永久不做 |
| iOS / Android clients | QD-319 | `[>]` | 上游已列计划，fork 暂缓；现有 Web 触摸输入不等于原生客户端 |
| MCP / AI 自动化 | QD-609 | `[x]` | 上游当前最高优先级；我们的新增业务能力尽量不破坏 MCP 兼容 |

## 17. 上游贡献与 PR 策略

### 17.1 分支与合并

- 从 `upstream/dev` 创建最小功能分支，不把整个 fork `dev` 直接推成一个巨型 PR。
- 每个 PR 只包含一个功能或一个 bug；服务端、OpenAPI、生成 SDK、smoke 必须同 PR。
- 上游合并后，把 `upstream/dev` 合回 fork `dev`，重新跑受影响测试，不在两边维护不同实现。
- 具有本 fork 品牌、部署环境、私有业务规则的页面和文档留在 fork；通用能力优先上游。

### 17.2 可直接准备的上游 PR

| 顺序 | PR 主题 | 来源 | 前置条件 |
|---|---|---|---|
| 1 | `feat(webclient): coordinate cross-tab control and audio` | QD-003 | 当前功能已测试；补上游 WebClient 约定和截图 |
| 2 | `fix(self-host): allow pinned SPKI for self-hosted WSS` | QD-114 / issue #20 | 候选已准备；需推送、创建 PR，并补 Qt 构建和 Windows service 实机验证 |
| 3 | `feat(webclient): control-plane management pages` | QD-108/605/607/703 | 依赖 PR #25 合并；按页面拆成多个 PR |
| 4 | `feat(qt): remote session and management pages` | QD-706/708 | 依赖 PR #25 合并；先做实时事件消费，再拆页面 |
| 5 | `feat(qt): consume self-hosted update channel` | QD-005 / QD-808 | 依赖 PR #25 合并；保留检查、下载、摘要校验、凭据隔离和测试，不携带 fork 私有文档 |

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
- 未完成：Qt/CMake 构建、Inno Setup 编译、Windows 服务和真实 `wss://` 回归；
  远端推送仍被策略拒绝，尚未创建 upstream PR

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
- 把 QD-005 的 Qt 更新消费作为 PR #25 的后续增量，先保证跨平台安装行为和签名链；
- 把 Web 文件管理器、Host 自动录制、图片剪贴板整理成 Remoting 需求，由维护者评估；
- 把 PR #25 的控制面能力先等上游审查，避免重复提交；
- fork 的 Web/Qt 业务页面在 #25 合并后按页面拆成增量 PR，而不是塞进一个巨型控制面 PR。
