# 远端发布与上游 PR 清单

**更新时间**: 2026-10-17

这份清单只记录本地验证完成后仍需远端写入的动作。当前机器上的 `git push`
会被执行策略拒绝，因此以下步骤应在允许远端写入的环境中执行。

## 当前本地基线

- fork 主干：`dev` 已包含 `cae77ff`（QD-005 Qt 客户端自动更新消费）
- 功能分支：`codex/product-backlog` = `dea02f9`
- 上游 QD-003 候选：`codex/upstream-cross-tab-coordination` = `6047615`（基于 `upstream/dev`）
- 上游 QD-114 候选：`codex/upstream-qd-114-spki` = `13559e6`（基于 `upstream/dev`）
- 上游最小候选：`codex/upstream-access-bans` = `b89c24c`
- 上游基线：`barry-ran/QuickDesk` 的 `dev` / `master` = `f6651f0`
- fork PR #1：<https://github.com/ilh-github/go-remote-next/pull/1>
- upstream PR #25：<https://github.com/barry-ran/QuickDesk/pull/25>
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
- `codex/product-backlog` 必须包含 `dea02f9`；
- `codex/upstream-access-bans` 必须包含 `b89c24c`；
- `codex/upstream-qd-114-spki` 必须包含 `13559e6`；
- 如果远端出现新提交，先合并或 rebase，再重跑受影响测试，不能强推覆盖他人工作。

## 2. 推送 fork 分支

```bash
git push origin \
  dev \
  codex/product-backlog \
  codex/upstream-qd-114-spki \
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
- 控制面管理页：等 PR #25 合并后按页面拆成独立 PR，不再使用“所有 fork 页面都留在 fork”
  的旧口径。
- 上游 issue #18 连接历史：上游 `dev` 已存在 `ConnectionHistory` 和历史列表 UI，
  当前不应重复实现；先核对完整功能，再决定是否在 issue 中说明可关闭。

## 5. 上游暂不合并时

不阻塞 fork 主线：

- 所有功能继续进入 `go-remote-next/dev`；
- fork PR 保持可独立发布；
- 上游候选分支保留 rebase 能力，定期对 `barry-ran/QuickDesk:dev` 重新应用；
- 通用 WebClient、Qt 和管理后台页面在控制面 #25 合并后按页面单独上游；
- 只有品牌、部署配置、本 fork 专属业务和旧 Go 迁移文档留在 fork。
