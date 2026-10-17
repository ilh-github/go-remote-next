# 远端发布与上游 PR 清单

**更新时间**: 2026-10-17 18:20 CST

这份清单只记录本地验证完成后仍需远端写入的动作。当前机器上的 `git push`
会被执行策略拒绝，因此以下步骤应在允许远端写入的环境中执行。

## 当前本地基线

- fork 主干：本地 `dev`、`master` 指向同一最新提交；代码基线为 `83d8caf`，
  已包含 QD-607 批量远控清场、QD-204 地址簿/设备分组、QD-508 Qt 录像回放页，
  QD-215 Windows 发布依赖校验及本清单维护提交
- fork 远端：`origin/dev`、`origin/master` 仍为 `f6651f0`；本地 `dev` 领先数量以
  `git rev-list --count origin/dev..dev` 为准，远端写入尚未完成
- 功能分支：`codex/product-backlog` = `dea02f9`
- 上游 QD-003 候选：`codex/upstream-cross-tab-coordination` = `6047615`（基于 `upstream/dev`）
- 上游 QD-114 候选：`codex/upstream-qd-114-spki` = `13559e6`（基于 `upstream/dev`）
- 上游 QD-212 候选：`codex/upstream-qd-212-local-device` = `f1ce096`（基于 `upstream/dev`）
- 上游 QD-213 候选：`codex/upstream-qd-213-access-code-refresh` = `6928d24`（基于 `upstream/dev`）
- 上游 QD-214 候选：`codex/upstream-qd-214-file-transfer-errors` = `94d3ec3`（基于 `upstream/dev`）
- 上游 QD-215 候选：`codex/upstream-qd-215-win-runtime-check` = `f449b87`
  （基于 `upstream/dev`，只含 Windows 发布依赖校验和打包失败关闭）
- 上游 QD-204 API 候选：`codex/upstream-qd-204-api` = `d2321b9`（基于 PR #25 的
  `feat/signaling-control-plane` = `8998466`）
- 上游 QD-204 客户端候选：`codex/upstream-qd-204-client` = `b259d03`（同一 PR #25
  基线，只含 Web/Qt，不含服务端/API 提交）
- 上游 QD-607 服务端候选：`codex/upstream-qd-607-batch-revoke` = `badf990`（基于 PR
  #25 的 `feat/signaling-control-plane` = `8998466`，只含批量远控清场服务端、handler、
  测试和用户 API 文档）
- 上游最小候选：`codex/upstream-access-bans` = `b89c24c`
- 上游基线：`barry-ran/QuickDesk` 的 `dev` / `master` = `f6651f0`
- 上游维护口径：issue #21 已确认“持续维护、暂缓更新、重点偏 MCP”；贡献从
  `upstream/dev` 切分支，完成后 PR 合入 `dev`；Remoting 改动由维护者处理
- 上游待办快照：10 个 open issue、4 个 open PR（#22、#23、#24、#25）；
  四个 PR 均为 MERGEABLE / UNSTABLE，且没有 reported checks
- 上游待办融合口径：fork 自有缺口、旧 Go-Remote 差距、上游 open issue/PR/Roadmap
  已统一落入 `docs/dev/QuickDesk功能总清单.md`；开发和上游 PR 顺序以其 3.1、3.2、
  第 16、17 节为准
- fork PR #1：<https://github.com/ilh-github/go-remote-next/pull/1>，
  OPEN / MERGEABLE / CLEAN，7 项检查全部成功
- upstream PR #25：<https://github.com/barry-ran/QuickDesk/pull/25>
- 上游 issue、PR、Roadmap 与本 fork 的融合矩阵：`docs/dev/QuickDesk功能总清单.md`
  第 16.2 节
- 两套清单合并后的实施顺序：`docs/dev/QuickDesk功能总清单.md` 第 3.2 节
- 上游映射与 PR 拆分规则：`docs/dev/QuickDesk功能总清单.md` 第 16、17 节

本地已经完成：

- 来源解封服务、房主/管理员接口、WebClient、Qt、管理后台；
- OpenAPI、Go/TypeScript/Python/Java SDK 和 contract smoke；
- 真 PostgreSQL 全量测试、两轮 live smoke、两个前端构建和四套 SDK 编译；
- `dev` 合并以及只含服务端/契约的上游候选提交；
- WebClient 实时消费 `device.access_policy.changed` 与 `remote_session.started/ended`，
  封禁列表和会话列表不再依赖手动刷新，10 秒轮询只作兜底。
- realtime smoke 增加真实策略事件和客户端会话生命周期覆盖，当前为 57 条断言。
- 上游候选包含来源解封、真实策略事件和会话生命周期 smoke，不含 fork 专属 UI。
- WebClient 新增 `/notifications` 通知与告警管理页，覆盖通道 CRUD/测试/启停、投递记录、
  失败重投、投递确认、静默时段与设备离线告警。
- WebClient 新增 `/scheduled-tasks` 计划任务页，覆盖一次性/固定间隔/日历排期、
  重试预算、执行状态、最后错误和失败实时刷新。
- WebClient 新增 `/access-approvals` 访问审批页，覆盖按设备审批、五状态汇总、
  重复拒绝来源、批准/拒绝原因和实时刷新。
- WebClient 新增同源多标签控制/音频抢占，BroadcastChannel 不可用时降级到
  localStorage；已完成真实 Chrome 双标签验证。
- Qt 设置页支持自托管 `wss://` 信令服务器的 SPKI 固定值，Host/Client 子进程会自动
  附加 Chromium `--ignore-certificate-errors-spki-list`；Windows Host 服务安装器支持
  通过 `QUICKDESK_SPKI` 把同一固定值写入服务注册参数。
- Qt 客户端已消费自有更新通道：自动/手动检查、提示、下载进度与取消、大小和
  SHA-256 校验、可选版本推迟、强制更新、Windows 静默安装、macOS 打开 DMG。
- 更新下载只在同一 origin 携带 API key；跨源安装包请求已验证不会泄漏凭据。
- QD-005 真实本地 HTTP 集成测试覆盖成功下载、摘要不匹配拒绝、残件清理和跨源隔离。
- Qt 我的设备和收藏列表已用当前 Host 设备 ID 标记“本机”，不影响备注/设备名展示；
  QD-212 上游候选已在干净基线上构建并通过 QML 测试。
- Qt 点击设备时如果云快照没有访问码，会按单设备 GET 回源，成功后自动继续连接；
  并发点击合并成一次请求，QD-213 上游候选已在干净基线上构建并通过集成测试。
- WebClient 会把宿主 `type=2` 文件传输错误解释为保存失败，并提示权限、目标目录和磁盘空间；
  上传取消发送协议取消值 1。QD-214 候选改善诊断，但宿主侧失败根因仍需日志确认。
- Qt 新增“访问审批”页，支持按设备审批、五状态汇总、重复拒绝来源、批准/拒绝原因、
  409 已决定冲突提示和 `device.access_approval.requested/decided` 实时刷新；提交
  `7c23f13` 已通过完整 Qt 构建、6/6 CTest、QML lint 和离屏启动。
- Qt 会话页实时消费 `remote_session.started/ended` 与
  `device.access_policy.changed`；会话事件按设备过滤并 150ms 防抖刷新，封禁来源直接
  使用事件中的 `deny_ips` 更新。WebSocket 未 bootstrap 完成时保留 10 秒轮询，断线后
  自动恢复；提交 `ea5a65f` 已通过完整构建和 7/7 CTest。
- Qt 新增计划任务页：一次、固定间隔和五种日历排期，服务端 action/排期能力直接驱动
  UI，展示下一次执行、尝试次数、最后错误并支持删除；`scheduled_task.failed` 会即时
  提示并刷新。提交 `42b6c61` 已通过完整构建、8/8 CTest、零新增页面 lint 和离屏启动。
- Qt 新增通知规则页：Webhook 通道 CRUD/test/启停、事件订阅、静默时段、设备离线阈值/
  重复提醒/维护窗口、投递历史/重试/确认、按通道和跨通道失败重投，消费设备实时事件并
  防抖刷新。提交 `dbd828f` 已通过完整构建、9/9 CTest、双语 701/701、页面 lint 和
  离屏启动。
- Qt 新增录像回放页：服务端录像游标分页、缩略图票据、应用内播放/暂停/停止、外部打开、
  删除和 `recording.*` 实时刷新。提交 `74c7022` 已通过完整构建、10/10 CTest、
  双语 736/736、页面 lint 和离屏启动。
- QD-204 新增 owner-scoped 地址簿：`user_device_groups`、成员表、五个
  `/v1/me/device-groups` API、OpenAPI 和四套 SDK；WebClient 与 Qt 都有分组筛选、CRUD、
  颜色/描述、成员整体替换和失败保留服务端状态。提交 `f270487`、`b1a1155`、
  `162ac6e`、`b013a62`。
- QD-204 已通过全新 PostgreSQL 从零迁移、全量 Go 测试、四套 SDK freshness/编译、
  真实服务 `388` 条 contract smoke、WebClient 34/34 + 生产构建，以及 Qt 完整构建、
  11/11 CTest、新弹窗 lint、双语 764/764 和离屏启动。
- QD-607 新增 `revoke_all_remote_sessions`：执行时按 owner 当前拥有的设备关闭活动远控
  会话，复用按设备的跨实例撤销路径，一次最多 200 台；不注销账号登录会话。空测试库全量
  Go 测试、`go vet`、`go build`、服务启动和 fork 三端测试均通过。
- QD-607 上游候选只含服务端、handler、测试和用户 API 文档 7 个文件；在 PR #25 基线上
  重新执行全量 Go 测试、`go vet`、`go build`、空库 `001_init.sql` 迁移和服务启动均通过。
  候选基线还没有 Web/Qt 计划任务页，所以客户端 kind 文案留在 fork，不强行塞进服务端 PR。
- Windows 自动更新已在 Parallels Windows 11 中实测到安装前一步：4.2.8.0 基线已安装，
  QuickDeskHost 服务 RUNNING；4.2.9.0 发布后客户端真实发现更新、下载完成并显示
  “下载已通过校验”，下载器的 SHA-256 与服务端安装包一致。尚未点击“安装并重启”，
  所以升级落盘、进程自动重启和服务重建仍不能标记完成。
- QD-215 Windows 发布依赖校验已落地：`verify_qd_win_release.ps1` 检查 x64 PE、
  `platforms/qwindows.dll`、VC Runtime 和 Qt 直接依赖，`publish_qd_win.bat` 与
  `package_qd_win.bat` 都会在缺依赖时中止。Windows 11 上 publish、Inno Setup 打包
  和缺 Runtime/平台插件/Qt 依赖的负向测试均通过，安装包仍为 89,676,841 字节，
  SHA-256 为 `67dbc4e5f56dec06d20b0feeef7d70545f087af4b8673025adba987cfcf9eeb6`。
- QD-215 上游候选 `f449b87` 已从 `upstream/dev` 单独切出，只含 3 个文件。候选校验器
  已在同一 Windows 发布目录上通过“正常包接受、缺 Runtime/Qt 插件/Qt 依赖均拒绝”的
  回归；`git push` 仍被当前执行策略拒绝，尚未创建 issue #7 PR。
- 本轮已加入 Windows 打包/发布校验脚本和负向回归；`version=4.2.9.0` 仍保留为
  未提交的验证版本。在 Windows 升级闭环确认前，不要把 `version` 当作正式发布版本
  盲目推送。
- QD-214 已开始用 Windows 服务模式复现上传失败；当前在线 Host 为
  `256983850`，下一步必须取得真实 `fileTransferError`、宿主日志和目录/磁盘/权限证据，
  根因确认前不修改文件传输实现。
- QD-214 前置联调实测（2026-10-17 18:23 CST）：WebClient 以生产构建连接
  `256983850`，信令 WebSocket 已返回 `auth_ok`，但 Host 随后发送
  `session-terminate`，原因为 `decline Verification hash mismatched.`；当前尚未进入
  DataChannel 和文件上传阶段。必须先解释这条 SPAKE2 认证差异，不能用错误提示掩盖。

## 1. 推送前检查

```bash
cd /Users/timbl/timbl_work/go-remote-next
git status --short --branch
git fetch origin --prune
git fetch upstream --prune
git log --oneline --left-right origin/dev...dev
git log --oneline --left-right origin/feat/signaling-control-plane...codex/upstream-access-bans
```

要求：

- 除机器本地的 `.idea/` 外，工作树必须干净；
- `dev` 必须包含 `f5d3283`；
- `dev` 必须包含 `2deb8c8`；
- `dev` 必须包含 `a48d793`；
- `dev` 必须包含 `cae77ff`；
- `dev` 必须包含 `4e8ea5a`；
- `dev` 必须包含 `9a2608b`；
- `dev` 必须包含 `cea23ce`；
- `dev` 必须包含 `7c23f13`；
- `dev` 必须包含 `ea5a65f`；
- `dev` 必须包含 `42b6c61`；
- `dev` 必须包含 `dbd828f`；
- `dev` 必须包含 `74c7022`；
- `dev` 必须包含 `f270487`；
- `dev` 必须包含 `b1a1155`；
- `dev` 必须包含 `162ac6e`；
- `dev` 必须包含 `b013a62`；
- `codex/upstream-qd-204-api` 必须指向 `d2321b9`；
- `codex/upstream-qd-204-client` 必须指向 `b259d03`；
- `codex/upstream-qd-607-batch-revoke` 必须指向 `badf990`；
- `codex/product-backlog` 必须包含 `dea02f9`；
- `codex/upstream-access-bans` 必须包含 `b89c24c`；
- `codex/upstream-qd-114-spki` 必须包含 `13559e6`；
- `codex/upstream-qd-212-local-device` 必须包含 `f1ce096`；
- `codex/upstream-qd-213-access-code-refresh` 必须包含 `6928d24`；
- `codex/upstream-qd-214-file-transfer-errors` 必须包含 `94d3ec3`；
- `codex/upstream-qd-215-win-runtime-check` 必须指向 `f449b87`；
- 如果远端出现新提交，先合并或 rebase，再重跑受影响测试，不能强推覆盖他人工作。

## 2. 推送 fork 分支

```bash
git push origin \
  dev \
  codex/product-backlog \
  codex/upstream-qd-607-batch-revoke \
  codex/upstream-qd-114-spki \
  codex/upstream-qd-212-local-device \
  codex/upstream-qd-213-access-code-refresh \
  codex/upstream-qd-214-file-transfer-errors \
  codex/upstream-qd-215-win-runtime-check \
  codex/source-unban \
  codex/upstream-access-bans
```

推送后确认：

```bash
git rev-parse dev
git rev-parse origin/dev
git rev-parse codex/upstream-access-bans
git rev-parse origin/codex/upstream-access-bans
```

fork 的 `dev` 是自主可控的持续开发主线。即使上游不合并，也不阻塞 fork 发布和后续开发。

## 3. 更新 fork PR #1

PR #1 的记录：

```bash
gh pr view 1 \
  --repo ilh-github/go-remote-next \
  --json number,title,state,baseRefName,headRefName,url,mergeable
```

PR #1 的 head 是 `codex/go-remote-foundation`，不是 `dev`。如果本次只需要把新功能留在
fork 主干，那么推送 `dev` 已经完成目标，不要为了更新 PR 把整个 `dev` 强行塞进一个旧的
最小功能分支。

如果该 PR 仍承担“控制面总集”的职责，应先把其 head 分支更新到当前目标基线，再重新运行
PR 检查并更新说明；否则保留它作为原始控制面审查记录，新功能以 `dev` 和独立 PR 交付。

## 4. 上游 PR 同步与候选

先确认上游 open PR 仍开放且基线未变化：

```bash
gh pr view 22 --repo barry-ran/QuickDesk \
  --json number,title,state,baseRefName,headRefName,url,mergeable,commits
gh pr view 23 --repo barry-ran/QuickDesk \
  --json number,title,state,baseRefName,headRefName,url,mergeable,commits
gh pr view 24 --repo barry-ran/QuickDesk \
  --json number,title,state,baseRefName,headRefName,url,mergeable,commits
gh pr view 25 \
  --repo barry-ran/QuickDesk \
  --json number,title,state,baseRefName,headRefName,url,mergeable,commits
```

不要重复提交 #22、#23、#24、#25 已有内容。上游合并后先把 `upstream/dev` 合回 fork，
再处理下面的增量候选。

推荐先让 PR #25 合并或由维护者确认堆叠基线，再为来源解封和 Web/Qt 业务页面开独立 PR。
若 GitHub 不允许跨仓库把 PR 直接堆叠到 fork 的 `feat/signaling-control-plane` 分支，
可以临时以下面的方式提交到上游 `dev`，并在 PR 描述中明确“depends on #25”：

```bash
gh pr create \
  --repo barry-ran/QuickDesk \
  --base dev \
  --head ilh-github:codex/upstream-access-bans \
  --title "feat(access): remove permanent source bans" \
  --body-file /tmp/source-unban-pr.md
```

### 4.1 QD-204 两个上游候选

两个候选都从 `feat/signaling-control-plane@8998466` 切出，不包含 fork 的文档、品牌页或
其他管理面：

- `codex/upstream-qd-204-api`：owner-scoped 表、service、handler、迁移、OpenAPI、
  TypeScript/Go/Python/Java SDK 和 contract smoke。已验证全量 Go 测试、`go vet`、
  四套 SDK freshness，以及真实服务 `PASS: 380 documented assertions`。
- `codex/upstream-qd-204-client`：WebClient 分组筛选/管理弹窗和 Qt `DeviceGroupManager`
  对话框，包括最小测试入口。已验证 Web 5/5、生产构建、Qt 完整 Debug 构建、
  CTest 1/1、弹窗 lint 和离屏启动。

远端写入恢复后：

```bash
git push origin \
  codex/upstream-qd-204-api \
  codex/upstream-qd-204-client
```

如果 PR #25 仍开放，两个 PR 的 base 先选 `feat/signaling-control-plane`；#25 合并后
GitHub 会自动把已合并功能的后续 PR 重定向到 `dev`。如果 GitHub 不接受以该分支为
base，先合并 #25/API PR，再把客户端候选 rebase 到新的 `upstream/dev`。当前机器
`git push` 仍被策略拒绝，因此尚未创建实际 PR。

PR 描述至少写明：

- 依赖 `#25`；
- 只包含来源解封的服务端、OpenAPI、SDK 和 smoke；
- `UnbanIP` 是幂等操作，已不存在来源成功 no-op；
- 非法来源即使设备没有策略也必须返回 `400`；
- 验证包括真 PostgreSQL、`go test ./...`、`go vet ./...`、四种 SDK 新鲜度与编译、
  以及 `PASS: 366 documented assertions` 的 live smoke。

如果维护者要求把改动并入 #25，而不是新建重复提交的 PR，则把 `7021ad1` 追加到
`feat/signaling-control-plane` 后更新 #25，并在评论中说明新增 scope。

### 上游可直接审查的增量

- `feat(webclient): coordinate cross-tab control and audio`：QD-003，纯 WebClient，
  无 Remoting 改动；候选分支 `codex/upstream-cross-tab-coordination` = `6047615`，
  仅 6 个文件，已通过 `npm test` 6/6、`npm run build` 和真实 Chrome 双标签验证。
- `fix(self-host): pinned SPKI for self-hosted WSS`：对应上游 issue #20；候选分支
  `codex/upstream-qd-114-spki` = `13559e6`，基于 `upstream/dev`，17 个文件。已通过
  SPKI 纯 C++ 单测、翻译 `.ts` XML 校验和 `git diff --check`；Qt/CMake 构建、
  Inno Setup 编译、Windows 服务安装和真实 `wss://` 回归仍待补充。该 PR 只解决
  `quickdesk-remoting` 临时兼容，不替代真实 `CertVerifierService` 修复。
- `feat(qt): mark the local device in device lists`：对应上游 issue #19 的本机标识；
  候选分支 `codex/upstream-qd-212-local-device` = `f1ce096`，基于 `upstream/dev`，
  10 个文件。已在干净上游基线上完整构建、运行 QML 测试和 `QuickDesk_qmllint`；
  远端推送仍被策略拒绝。
- `fix(qt): refresh missing access codes before connecting`：对应上游 issue #19 的
  “访问码不可用”；候选分支 `codex/upstream-qd-213-access-code-refresh` = `6928d24`，
  基于 `upstream/dev`，9 个文件。已验证单设备回源、并发合并、失败路径和自动续连；
  远端推送仍被策略拒绝。
- `fix(webclient): explain host file transfer failures`：对应上游 issue #19 的
  `Host error:type=2`；候选分支 `codex/upstream-qd-214-file-transfer-errors` = `94d3ec3`，
  基于 `upstream/dev`，4 个文件。WebClient 测试 4/4 和生产构建通过；这是诊断改进，
  不是宿主上传根因修复。远端推送仍被策略拒绝。
- `fix(win): reject incomplete Qt runtime releases`：对应上游 issue #7；候选分支
  `codex/upstream-qd-215-win-runtime-check` = `f449b87`，基于 `upstream/dev`，仅含
  3 个文件。校验 `QuickDesk.exe`/`qwindows.dll` 的 x64 架构、VC Runtime 和 Qt 直接依赖，
  并让 publish/package 在缺依赖时失败关闭；Windows 11 正常包和缺 Runtime/平台插件/Qt
  依赖的负向用例均通过。远端推送仍被策略拒绝。
- `feat(scheduled): add batch remote-session revoke`：QD-607 的控制面批量清场；候选分支
  `codex/upstream-qd-607-batch-revoke` = `badf990`，基于
  `feat/signaling-control-plane@8998466`，7 个服务端/测试/API 文档文件。候选独立执行
  全量 Go 测试、`go vet`、`go build`、空库迁移和服务启动通过；web/dist 是新 worktree
  的构建前置，不属于提交内容。远端推送仍被策略拒绝。
- 控制面管理页：等 PR #25 合并后按页面拆成独立 PR，不再使用“所有 fork 页面都留在 fork”
  的旧口径。Qt 访问审批页 `7c23f13`、会话实时消费 `ea5a65f`、计划任务页 `42b6c61`、
  通知规则页 `dbd828f` 和录像回放页 `74c7022` 作为 Qt 管理面候选，不携带 fork
  专属总清单。
- 上游 issue #18 连接历史：上游 `dev` 已存在 `ConnectionHistory` 和历史列表 UI，
  当前不应重复实现；先核对完整功能，再决定是否在 issue 中说明可关闭。

## 5. 上游暂不合并时

不阻塞 fork 主线：

- 所有功能继续进入 `go-remote-next/dev`；
- fork PR 保持可独立发布；
- 上游候选分支保留 rebase 能力，定期对 `barry-ran/QuickDesk:dev` 重新应用；
- 通用 WebClient、Qt 和管理后台页面在控制面 #25 合并后按页面单独上游；
- 只有品牌、部署配置、本 fork 专属业务和旧 Go 迁移文档留在 fork。
