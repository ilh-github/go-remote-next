# 远端发布与上游 PR 清单

**更新时间**: 2026-10-17

这份清单只记录本地验证完成后仍需远端写入的动作。当前机器上的 `git push`
会被执行策略拒绝，因此以下步骤应在允许远端写入的环境中执行。

## 当前本地基线

- fork 主干：`dev` = `0cc0300`
- 来源解封功能分支：`codex/source-unban` = `6230d16`
- 上游最小候选：`codex/upstream-access-bans` = `7021ad1`
- fork PR #1：<https://github.com/ilh-github/go-remote-next/pull/1>
- upstream PR #25：<https://github.com/barry-ran/QuickDesk/pull/25>

本地已经完成：

- 来源解封服务、房主/管理员接口、WebClient、Qt、管理后台；
- OpenAPI、Go/TypeScript/Python/Java SDK 和 contract smoke；
- 真 PostgreSQL 全量测试、两轮 live smoke、两个前端构建和四套 SDK 编译；
- `dev` 合并以及只含服务端/契约的上游候选提交；
- WebClient 实时消费 `device.access_policy.changed` 与 `remote_session.started/ended`，
  封禁列表和会话列表不再依赖手动刷新，10 秒轮询只作兜底。
- realtime smoke 增加真实策略事件和客户端会话生命周期覆盖，当前为 57 条断言。

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
- `dev` 必须包含 `0cc0300`；
- `codex/upstream-access-bans` 必须包含 `7021ad1`；
- 如果远端出现新提交，先合并或 rebase，再重跑受影响测试，不能强推覆盖他人工作。

## 2. 推送 fork 分支

```bash
git push origin \
  dev \
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

## 4. 上游 PR #25 与来源解封候选

先确认 PR #25 仍开放且基线未变化：

```bash
gh pr view 25 \
  --repo barry-ran/QuickDesk \
  --json number,title,state,baseRefName,headRefName,url,mergeable,commits
```

推荐先让 PR #25 合并或由维护者确认堆叠基线，再为来源解封开独立 PR。若 GitHub 不允许
跨仓库把 PR 直接堆叠到 fork 的 `feat/signaling-control-plane` 分支，可以临时以下面的方式
提交到上游 `dev`，并在 PR 描述中明确“depends on #25”：

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

## 5. 上游暂不合并时

不阻塞 fork 主线：

- 所有功能继续进入 `go-remote-next/dev`；
- fork PR 保持可独立发布；
- 上游候选分支保留 rebase 能力，定期对 `barry-ran/QuickDesk:dev` 重新应用；
- 不把 fork 专属的 WebClient、Qt 和管理后台页面混进上游控制面补丁。
