# 远端发布与上游 PR 清单

**更新时间**: 2026-10-26 16:20 CST

## 本轮推送（2026-10-26 16:20 CST）

- `b0cf3f4`（feat(scheduled): let an owner correct a task instead of recreating it）
  已推送：`git push origin dev` 成功，`origin/dev` 从 `87a4be7` 前进到 `b0cf3f4`，
  `git rev-list --left-right --count origin/dev...dev` = `0 0`；本地 `master`
  已同步到同一提交。工作树只剩三个约定不提交的未跟踪路径。
- 该提交含 C++/QML 改动，本机无 Qt 工具链，构建与测试证据来自 CI
  （`QuickDesk Windows` / `QuickDesk macOS` workflow，触发路径含 `QuickDesk/**`）。
  **这两条 run 的结果需在上传后核对**，通过前不把 Qt 侧标记为已验证。
- 上游 PR 状态本轮复核：#34 `CLEAN`/`MERGEABLE` 且 `build` SUCCESS，#33/#32/#31/#26 均
  `CLEAN`，#25 仍 `DIRTY`/`CONFLICTING`（base `dev`，与上游 `dev` 已分叉），未动作。
- 上游 `dev` 仍为 `88d4d36`。
- 计划任务的编辑动作属于 QD-607，与 `codex/upstream-qd-607-batch-revoke@badf990`
  同一主题；该候选已独立验证，等 #25 基线确认后 rebase 并 stacked 回贡，
  不在本轮另开 PR。

**更新时间**: 2026-10-26 23:10 CST

## 本轮推送（2026-10-26 23:10 CST）

- `e4e58db`（test(webclient): pin the event coverage check as behaviour, not text）
  已推送：`git push origin dev` 成功，`origin/dev` 从 `00deb1d` 前进到 `e4e58db`，
  `git rev-list --left-right --count origin/dev...dev` = `0 0`；本地 `master`
  已同步到同一提交。该提交只含一个测试文件与清单/联调记录的三处文档，无产品行为变更，
  因此不建上游 PR。
- 上游 PR 状态本轮复核：#34 `CLEAN`/`MERGEABLE` 且唯一检查 `build` 为 SUCCESS；
  #33 `CLEAN`；#32 `CLEAN`；#31 `CLEAN`；#26 `CLEAN`；#25 仍 `DIRTY`/`CONFLICTING`
  （base `dev`，与上游 `dev` 已分叉），未动作。
- 上游 `dev` 仍为 `88d4d36`（PR #24 的 skill-host 修复），与 fork 基线一致。

**更新时间**: 2026-10-26 12:50 CST

## 本轮推送（2026-10-26 12:50 CST）

- `bc47d45`、`5ea85dd`、`dba6721`、`8ec5fed`、`998aae2` 五连提交已推送：
  `git push origin dev master` 成功，两个分支均从 `caf4ae5` 前进到 `998aae2`，
  `git rev-list --left-right --count origin/dev...dev` = `0 0`。
- **`SignalingServer verify` 首次全绿**（run `40270895099`，head `8ec5fed`，
  四个 job 全过）。该 workflow 自 10-16 加入 `QD_TEST_DATABASE_URL` 起连败十天，
  根因是 `gorm.io/driver/postgres` 的驱动缺陷，详情见清单 QD-909 行。
- 其中两个是产品缺陷，不只是测试问题：`RemoteSessionService.Open` 的幂等重试
  在 Postgres 上必然失败（回查在已被失败 INSERT 中止的同一事务里），以及
  `cmd/signaling` 的 AutoMigrate 对已存在的表全部静默跳过。
- **上游候选已建**：`codex/upstream-signaling-postgres-migrate` = `3db3ad9`，
  基于 `upstream/dev@88d4d36`，只含 `database.AutoMigrate` 辅助（30 行）与
  `cmd/signaling` 一处调用改动，未掺入任何 fork 专有内容。
  **PR #34**（`barry-ran/QuickDesk`，base `dev`）已创建：OPEN / MERGEABLE。
  上游 `SignalingServer` 全文没有任何 `PrepareStmt`，同一个缺陷在上游同样存在。
- 上游 `internal/database` 没有测试文件、CI 也没有 `QD_TEST_DATABASE_URL`，
  所以候选只带一个不依赖数据库的单测（`AutoMigrate` 的 nil 防护）。
  需要真实 Postgres 的那条迁移回归测试留在 fork 侧，见
  `SignalingServer/internal/database/postgres_test.go` 的历史版本。

这份清单记录本地验证完成后的远端写入与 PR 状态。远端状态必须以实际
`origin/*` ref、push 结果和 GitHub PR 为准，不能把本地候选写成已共享。

## 本轮推送（2026-10-25 03:50 CST）

- `270b7d3`（ci: give ctest the configuration it needs to run anything）已推送：
  `git push origin dev master` 成功，`origin/dev` 与 `origin/master` 均从
  `17db717` 前进到 `270b7d3`，`git rev-list --left-right --count origin/dev...dev`
  = `0 0`。该提交含两个 workflow 的 `ctest -C Release` 与新检查脚本
  `scripts/e2e/test-quickdesk-cmake-ctest-config.sh`。
- 该修复属于 fork 自有 CI，不构成上游改动，不建 PR。
- 上游候选分支本轮无变化；`codex/upstream-qd-215-win-runtime-check` 的
  `7892a7b` 与 `8385856` 仍对应 PR #32，两端 workflow 均 success。

## 本轮推送（2026-10-25 04:20 CST）

- `f956d17`（build: put the trust-handler CTest in a build tree at all）已推送：
  `git push origin dev master` 成功，`origin/dev` 与 `origin/master` 均前进到
  `f956d17`，`git rev-list --left-right --count origin/dev...dev` = `0 0`。
  该提交含根 `CMakeLists.txt` 的 `add_subdirectory(test/trust)` 接线与
  `scripts/e2e/test-quickdesk-trust-test-wiring.sh`。
- CI 实跑结果（run `40128304387`，head `f956d17`，Windows `Unit tests` job）：
  17/17 通过，其中 `1/17 trust_handler_thread_affinity ... Passed 1.25 sec`。
  这是该测试首次真正参与构建并执行，QD-113 据此转 `[x]`。
- 同轮前一提交 `270b7d3`（ctest `-C`）对应 run `40126751594`，同为 success，
  16/16。两次合起来才是完整的「测试真的在跑」证据。
- 本轮改动属 fork 自有 CI/构建描述，不构成上游产品行为变更，不建 PR。
  上游 `barry-ran/QuickDesk` 的 `test/trust/CMakeLists.txt` 与 fork 一致
  （同源自 `5969ae3`），若上游愿意收「把该工程接进 `QUICKDESK_BUILD_TESTS`」
  这一行，可作为独立最小候选，本轮未创建。

## 本轮推送（2026-10-25 05:00 CST）

- QD-216 时序验收的测试已落地并推送：`af814a6`（假 Host 与集成测试）、
  `306f5c1`（修 MSVC 上 `QCommandLineOption` 歧义）、`cc37e00`（fixture 改为
  QObject）、`2be3f03`（补 config 源文件）、`cbfcbe9`（修自己写错的断言）。
  `git push origin dev master` 成功，`origin/dev` 与 `origin/master` 均到达
  `cbfcbe9`，`git rev-list --left-right --count origin/dev...dev` = `0 0`。
- CI 实跑（run `40133964841`，head `cbfcbe9`，Windows `Unit tests` job）：
  18/18 通过，`quickdesk_host_startup_integration_test` `Passed 1.74 sec`。
  这是 QD-216 第 1、3、4 条验收用例的第一份自动化证据。
- QD-216 保持 `[~]`：第 2 条只到状态层、第 5 条未单独回归、第 6 条需 macOS
  实跑。不因编译通过或部分用例转绿就写成完成。
- 本轮改动属 fork 自有测试，不构成上游产品行为变更，不建 PR。

## 本轮推送（2026-10-25 06:20 CST）

- QD-216 第 2 条（可读失败原因）已落地并推送：`af57353`（ServerStatusDetail
  枚举 + 属性 + QML 翻译）、`c24fbdb`（去掉多余的 `core::` 前缀）、
  `d8be923`（把 `serverstatusdetail.cpp` 补进 app target——改动做了却没 stage，
  白费一轮 CI）。`git push origin dev master` 成功，远端均到达 `d8be923`，
  `git rev-list --left-right --count origin/dev...dev` = `0 0`。
- QD-216 状态：第 1、3、4 条有 CI 证据（run `40133964841`，18/18），
  第 2 条已实现且有 CI 证据（run `40137597030`，19/19，含新增的
  `quickdesk_server_status_detail_test`）；第 5、6 条未验证，保持 `[~]`。
- 本轮改动属 fork 自有 UI/状态语义，不构成上游产品行为变更，不建 PR。

## 本轮推送（2026-10-25 07:40 CST）

- QD-209 的 Qt 撤销路径测试与真实缺陷修复已推送：`a0fc422`（测试）、
  `cfd09c9`（补上被漏掉的 struct 定义）、`874336b`（修 `/` 未编码）、
  `1b78bdd`（第二参类型）、`b9d72fa`（改用 `toEncoded`）、`409deba`（按路径段断言）。
  `git push origin dev master` 成功，远端均到达 `409deba`，
  `git rev-list --left-right --count origin/dev...dev` = `0 0`。
- CI 实跑（run `40145248663`，head `409deba`，Windows `Unit tests` job）：
  19/19 通过，`quickdesk_remote_session_manager_test` 在列。
- QD-209 的 Qt 侧请求/响应契约已有自动化证据；真机踢人与批量清场效果仍待补，
  保持 `[~]`。
- `874336b` 是产品缺陷修复，值得考虑单独拆上游候选：`encodedPathSegment`
  的写法在 `QuickDesk/src` 下多个 manager 重复，上游同样受益。
  本轮未创建，留待下一轮按最小分支切出。

## 本轮推送（2026-10-25 08:50 CST）

- `6a9324a`（refactor: one percent-encoder for a path segment, not three copies）
  已推送，`origin/dev` 与 `origin/master` 均到达该提交，
  `git rev-list --left-right --count origin/dev...dev` = `0 0`。
- CI 实跑（run `40146772610`，head `6a9324a`，Windows `Unit tests` job）：
  20/20 通过，`quickdesk_urlencode_test` 在列。
- 上游候选更新：`874336b` 的 `/` 编码修复现已扩展为 `6a9324a` 的单一实现重构。
  切上游最小分支时应从 `upstream/dev` 重新取，只带 `common/urlencode.h`、
  `urlencode_test.cpp` 与被改的 CMake 片段，不带 fork 的会话撤销业务代码。

## 本轮推送（2026-10-25 21:10 CST）

- macOS run 长期失败的根因已处理：`test-quickdesk-cmake-test-targets.sh` 与
  `test-quickdesk-cmake-ctest-config.sh` 都需要「有 cmake/编译器但没有 Qt」的环境，
  只有容器能给，而 macOS 与 Windows 两个 hosted runner 都没有 Docker。
  两脚本已加 `command -v docker` 守卫（缺失时说明原因并退出 0），
  并从 macOS job 摘掉，真实情况写入 workflow 注释。
- QD-216 第 6 条的两个平台证据均已具备：Windows run `40137597030`（19/19）、
  macOS arm64 run `40146772765`（20/20，host startup 集成测试在列）。
  此前只记 Windows 是因为整个 macOS run 是红的，没往下看一层。
- `28b7b9b` 另外把 macOS `build` strategy 改成 `fail-fast: false`：
  x64 的 `Package DMG` 因缺签名凭据失败是长期已知的外部阻塞，
  默认的 fail-fast 会连带取消 arm64 build，把已经拿到的证据一起扔掉。
- 实跑确认：macOS run `40201665718`（head `28b7b9b`）`Unit tests (arm64)`
  十个步骤全部 success，20/20；Windows run `40196394078`（head `550afca`）
  亦 success。两个平台均恢复健康。
- macOS `Build (x64)` / `Build (arm64)` 仍因缺 `MACOS_SIGN_IDENTITY` 失败，
  属 QD-005 已知外部阻塞；`fail-fast: false` 后不再污染测试证据。
- **覆盖缺口（未修复）**：这两个 cmake 容器检查现在 CI 里没有、只在本地跑。
  不是被修好的问题，是没有假装它还在。补上需要一个带容器的 job，
  或换一种不依赖"无 Qt 环境"的验证思路。

## 本轮推送（2026-10-26 02:30 CST）

- QD-216 已收口并转 `[x]`：`a9c8261`（hostReady 主路径用例 + 两种字段拼写）、
  `cb2da01`（红绿验证用的故意破坏，已恢复）、`af7052d`（恢复）。
  `git push origin dev master` 成功，远端均到达 `af7052d`，
  `git rev-list --left-right --count origin/dev...dev` = `0 0`。
- 两个平台、且都是含新用例的提交：Windows run `40209985101`（head `a9c8261`）20/20；
  macOS arm64 run `40211749988`（head `af7052d`）20/20、十个步骤全部 success。
  早先的 macOS 证据 `28b7b9b` 早于 `a9c8261`，不覆盖新用例，已重新取。
- 清单统计：88 `[x]`、20 `[~]`、5 `[ ]`、1 `[!]`、2 `[>]`、1 `[-]`（共 117）。
- 上游候选：QD-216 属上游 issue #9 的修复，`hostreadiness` 纯逻辑层可独立切出；
  Qt 接线依赖 fork 的 MainController 结构，留 fork。

## 本轮推送（2026-10-26 09:40 CST）

- QD-511 的「可读失败原因」层已落地：`a3ebfae`（feat: tell the operator which of
  four failures they have）。`git push origin dev master` 成功，远端均到达
  `a3ebfae`，`git rev-list --left-right --count origin/dev...dev` = `0 0`。
- 验证：`cargo test --offline --bin quickdesk-mcp` 119/119（新增 2 个用例）、
  `cargo clippy --all-targets -- -D warnings` 零告警、`git diff --check` 通过。
- 红绿做了三种破坏：分类塌陷、guidance 统一文案，以及第一版测试自身漏测。
  前两种确认会红；第三种是测试的缺陷，加固后才抓住。已在联调记录第 171 节记全过程。
- QD-511 保持 `[~]`：剩余「自动 AI 失败解释/决策」需外部 AI 服务。
- 上游候选：`MismatchKind` / `classify_mismatch` / `mismatch_guidance` 是纯函数，
  不依赖 fork 业务，可单独切成上游最小 PR；本轮未创建。

## 当前本地基线

- fork 本地主线：`dev` / `master` 已同步；QD-907 功能提交为 `33e9851`、`fd92cdd`，
  上传重试耗尽测试为 `f8d0a34`；QD-610 数据库事务 outbox 提交为 `6c71f97`，设备访问
  事件迁移为 `f2534ec`，策略变更迁移为 `6bdc057`，远端会话生命周期迁移为 `3bbc175`，
  录像生命周期迁移为 `cd08a0f`，访问审批生命周期迁移为 `c650d3d`。文档收口后
  共享访问码生命周期迁移为 `5e2542c`，离线告警迁移为 `b6855cc`。文档收口后
  指标阈值告警迁移为 `c4a28ee`，设备控制面变更迁移为 `0c621ef`。文档收口后
  收藏设备生命周期迁移为 `0b746b9`，管理端设备事件迁移为 `b2a0a98`，TURN 配置事件迁移为 `6386a48`，管理员来源封禁事务化为 `8de8af0`，WebClient 有界重连为 `565f643`。QD-907
  Playwright E2E 为 `830a469`、`2ab09d5`；领先数量和当前文档提交以
  `git rev-list --count origin/dev..dev`、`git rev-parse HEAD` 的实时值为准；
  QD-004 僵死录制显式终态补 `ended_at` 的提交为 `f19db63`
- fork 远端：`origin/dev` 与 `origin/master` 当前均为 `23af445`
- 本地 `dev` / `master` 已同步，均领先各自远端；每次发布前重新核对实时数量
- 2026-10-23 16:10 `git push origin dev master` 成功：远端 `dev` 与 `master` 均到达
  `1ad0b85`（QD-511 park 的 run 可被人工接管），本地 `dev` / `master` 已同步
- 2026-10-24 14:00 `git push origin dev master` 成功：远端 `dev` 与 `master` 均到达
  `82ca8b2`（QD-315 移动端入口与触摸笔迹），本地 `dev` / `master` 已同步
- 2026-10-24 11:00 `git push origin dev master` 成功：远端 `dev` 与 `master` 均到达
  `6960fd9`（QD-315 收尾：分辨率同步、断线清理、迟到笔画），本地 `dev` / `master`
  已同步（`git rev-list --left-right --count origin/dev...dev` = `0 0`）
- 2026-10-24 08:20 `git push origin dev master` 成功：远端 `dev` 与 `master` 均到达
  `fce1bf5`（QD-315 控制端画笔标注与观看端共享），本地 `dev` / `master` 已同步
- 2026-10-24 05:10 `git push origin dev master` 成功：远端 `dev` 与 `master` 均到达
  `5b739a9`（QD-610 跨机器分布式执行队列），本地 `dev` / `master`
  已同步（`git rev-list --left-right --count origin/dev...dev` = `0 0`）
- 2026-10-24 02:40 `git push origin dev master` 成功：远端 `dev` 与 `master` 均到达
  `e04451c`（QD-909 剪贴板审计，契约不存放文本），本地 `dev` / `master`
  已同步（`git rev-list --left-right --count origin/dev...dev` = `0 0`）
- 2026-10-24 00:20 `git push origin dev master` 成功：远端 `dev` 与 `master` 均到达
  `3858f11`（QD-314 控制权设置入口与真机双控 E2E），本地 `dev` / `master`
  已同步（`git rev-list --left-right --count origin/dev...dev` = `0 0`）
- 2026-10-23 20:05 `git push origin dev master` 成功：远端 `dev` 与 `master` 均到达
  `d435c6f`（QD-314 多人同时控制的控制权模型与冲突策略），本地 `dev` / `master`
  已同步（`git rev-list --left-right --count origin/dev...dev` = `0 0`）
- 2026-10-23 18:58 `git push origin dev master` 成功：远端 `dev` 与 `master` 均到达
  `f19db63`（QD-004 僵死录制进 `failed` 时同时写入 `ended_at`），本地 `dev` / `master`
  已同步（`git rev-list --left-right --count origin/dev...dev` = `0 0`）
- 2026-10-23 00:40 `git push origin dev master` 成功：远端 `dev` 与 `master` 均到达
  `97b8619`（QD-610 Owner 委托权限），本地 `dev` / `master` 已同步
- 2026-10-22 00:30 `git push origin dev master` 成功：远端 `dev` 与 `master` 均到达
  `2848be5`（QD-610 异步工作流公平调度），本地 `dev` / `master` 已同步
- 2026-10-21 01:05 `git push origin dev` 成功：远端 `dev` 到达 `a1d34c5`
  （QD-610 账号会话撤销事件全部迁入事务 outbox，含上一片 `314c73b` 的改密码路径），
  本地 `dev` 与 `master` 均已同步到同一提交
- 2026-10-20 00:10 执行策略放行：`git push origin dev master` 成功，远端
  `dev` 与 `master` 均到达 `cddc0db`，与本地一致（`git rev-list --left-right
  --count origin/dev...dev` = `0 0`）。此前 111 个积压提交全部推送完成，
  包含 QD-908 soak 门禁修正与 `cddc0db` 的在线状态 outbox 迁移
- 推送 blocker（历史，已于 2026-10-20 解除）：2026-10-18 执行 `git push origin dev master`
  和候选分支 push 均被执行策略拒绝，错误为
  `approval required by policy, but AskForApproval is set to Never`；未绕过审批策略。
  该阻塞在 2026-10-20 00:10 解除，主分支已同步；候选分支见下方“待推送候选”
- 功能分支：`codex/product-backlog` = `dea02f9`
- 上游 QD-003 候选：`codex/upstream-cross-tab-coordination` = `93eee37`（基于 `upstream/dev`）
- 上游 QD-114 候选：`codex/upstream-qd-114-spki` = `13559e6`（基于 `upstream/dev`）
- 上游 QD-212 候选：`codex/upstream-qd-212-local-device` = `f1ce096`（基于 `upstream/dev`）
- 上游 QD-213 候选：`codex/upstream-qd-213-access-code-refresh` = `6928d24`（基于 `upstream/dev`）
- 上游 QD-214 SPAKE2 候选：`codex/upstream-qd-214-spake2-webclient` = `ed5437b`
  （基于 `upstream/dev`）
- 上游 QD-214 错误码候选：`codex/upstream-qd-214-file-transfer-errors` = `60837af`
  （基于 `upstream/dev`）
- 上游 QD-215 候选：`codex/upstream-qd-215-win-runtime-check` = `f449b87`
  （基于 `upstream/dev`，只含 Windows 发布依赖校验和打包失败关闭）
- 上游 QD-002 file-ops 候选：`codex/upstream-qd-002-file-ops` = `e106be4`（基于
  `upstream/dev@f6651f0`），只含分块读写、默认目录、版本保留软删除、SKILL.md 和
  Rust 测试；已推送并创建上游 PR #26（OPEN / MERGEABLE / CLEAN）
- 上游 QD-403 WebClient 候选：`codex/upstream-qd-403-webclient` = `796e4cb`（stacked 在
  file-ops 候选上），只含 `SkillBridgeClient`、Web 文件管理器、remote-main 接线、样式、
  双语、Node 测试和 Chrome 文件管理器 E2E；上游 PR #27 已存在（当前 base 为 `dev`），
  但远端分支仍是旧 `9ff00af`，`796e4cb` 尚未推送
- 上游 QD-204 API 候选：`codex/upstream-qd-204-api` = `d2321b9`（基于 PR #25 的
  `feat/signaling-control-plane` = `8998466`）
- 上游 QD-204 客户端候选：`codex/upstream-qd-204-client` = `b259d03`（同一 PR #25
  基线，只含 Web/Qt，不含服务端/API 提交）
- 上游 QD-607 服务端候选：`codex/upstream-qd-607-batch-revoke` = `badf990`（基于 PR
  #25 的 `feat/signaling-control-plane` = `8998466`，只含批量远控清场服务端、handler、
  测试和用户 API 文档）
- 上游 QD-111 / QD-408 候选：`codex/upstream-qd-111-file-transfer-audit` = `c57a2f4`，
  基于 PR #25 的 `feat/signaling-control-plane@8998466`；44 个文件，只含文件传输审计、
  contract smoke 和该基线已有的四套 SDK，不含 fork 清单或 C#/PHP/Kotlin SDK；本地验证
  完成但尚未推送到 `origin`
- QD-907 上游候选已准备：`codex/upstream-websocket-auth-timeout` = `729d9fe`，基于
  PR #25 的 `feat/signaling-control-plane@8998466`；只改 WebClient 信令 transport、
  有界重连、访问码错误分类、remote 入口、Node test 和 Playwright E2E。候选自身
  单测 10/10、浏览器 E2E 6/6 + fork-only 上传用例跳过 1 项、生产构建和 diff check
  通过；修复 auth_ok 超时、认证前 payload/send 门禁、
  访问策略/错误访问码/排队/签名失败分类，以及断线后重新签发 token 的 0/1/3/7 秒
  有界重连与耗尽状态。fork 已增加真实 Chrome + Playwright 的 QD-907 E2E，
  `2ab09d5` 的 7/7 测试覆盖第 1~3 次恢复、四次耗尽、策略拒绝、上传并发和旧 Session 隔离；
  `729d9fe` 已把同一测试追加到上游候选；候选基线没有 fork-only 录像上传模块，
  因此该用例显式跳过，其余 6 项通过。分支仍未推送到 `origin`
- 上游 QD-006 候选：`codex/upstream-qd-006-sqlite` = `7758456`，基于 PR #25 的
  `feat/signaling-control-plane@8998466`；候选自身完整 Go 测试、database race、
  `go vet`、Web 生产构建和最小服务 E2E 通过。PR #25 合并后需 rebase 到
  `upstream/dev` 再创建正式 PR，不能直接拿 fork `dev` 的 `c77dcae`
- 上游 QD-604 服务端候选：`codex/upstream-qd-604-metrics` = `0ea9e7c`，基于
  `codex/upstream-qd-006-sqlite@7758456`，形成 `PR #25 -> QD-006 -> QD-604` 的 stacked
  链；只含 SignalingServer、OpenAPI、四套 SDK、contract smoke 和 metric E2E，不含
  Web/Qt/fork 清单。Web/Qt 候选等 PR #25 客户端基线确认后另拆
- 上游 QD-911 候选：`codex/upstream-qd-911-security` = `9862717`，直接基于
  `upstream/dev@f6651f0`，只新增根目录 `SECURITY.md`；上游 issue #6 仍 OPEN，
  GitHub Private Vulnerability Reporting 当前 `enabled=false`
- 上游 QD-115 候选：`codex/upstream-qd-115-trust-model` = `1ca2543`，直接基于
  `upstream/dev@f6651f0`，只新增 `docs/security/signaling-trust-model.md`；对应 issue #10
- 上游 QD-608 候选：`codex/upstream-qd-608-extra-sdks` = `cca9e44`，基于 PR #25 的
  `feat/signaling-control-plane@8998466`；在 PR #25 基线上重新生成 TypeScript/Go/Python/
  Java/C#/PHP/Kotlin 七套客户端，不携带 fork 后续业务代码
- 上游 QD-810 候选：尚未创建；fork 功能提交为 `64ab93a`。它依赖 PR #25 和 QD-006
  SQLite 候选，应在两者落地后再拆 `quickdesk-backup`、配额与两个存储视图的最小增量
- 上游最小候选：`codex/upstream-access-bans` = `b89c24c`
- 上游 QD-002 file-ops 候选：`codex/upstream-qd-002-file-ops` = `e106be4`（基于
  `upstream/dev@f6651f0`），已推送并创建上游 PR #26（OPEN / MERGEABLE / CLEAN）
- 上游 QD-403 WebClient 候选：`codex/upstream-qd-403-webclient` = `796e4cb`（stacked 在
  file-ops 候选上）；PR #27 当前仍指向远端旧 head `9ff00af`
- 上游基线：`barry-ran/QuickDesk` 的 `dev` = `5db316c`，`master` = `f6651f0`
- 上游维护口径：issue #21 已确认“持续维护、暂缓更新、重点偏 MCP”；贡献从
  `upstream/dev` 切分支，完成后 PR 合入 `dev`；Remoting 改动由维护者处理
- 上游待办快照：10 个 open issue、5 个 open PR（#22、#23、#25、#26、#27）；
  #24 已合并为 `5db316c`；#26 为 `CLEAN`，#22、#23、#25、#27 为 `UNSTABLE`
- 上游待办融合口径：fork 自有缺口、旧 Go-Remote 差距、上游 open issue/PR/Roadmap
  已统一落入 `docs/dev/QuickDesk功能总清单.md`；开发和上游 PR 顺序以其 3.1、3.2、
  3.3 和第 16、17 节为准
- 本轮合并结论：上游没有新增必须立即重复开发的 issue；`QD-608`
  （C#/PHP/Kotlin 生成 SDK）和 `QD-510` MCP 工作流核心均已落地，下一步推进
  `QD-610` 业务集成；QD-510 可拆成独立上游候选，业务触发与回调留在 fork
- 上游 QD-510 候选：`codex/upstream-qd-510-workflow` = `0a9ecd5`，基于
  `upstream/dev@f6651f0`；8 个文件，只含 MCP 双通道工作流、报告工具和通用文档，
  19/19 测试、Clippy 零告警及 diff check 通过
- QD-506 Web 录像分块上传：fork 提交 `de88f07`、smoke 自举 `eba7ae2`；WebClient
  50/50、生产构建和真实 SQLite 4 分片逐字节 E2E 通过。候选
  `codex/upstream-qd-506-recording-upload` = `de45538`，基于 PR #23 的 `00a7cb8`；
  候选自身 3/3 单测、生产构建和 diff check 通过，真实 SQLite E2E 因旧基线无 SQLite
  驱动而留在 fork 主线验证
- QD-610 本机业务 HTTP trigger：fork 提交 `bca8149`，MCP 23/23、Clippy 和 release
  构建通过；通用 MCP 部分已准备 stacked 候选
  `codex/upstream-qd-610-business-trigger` = `3262cc9`，基于 QD-510 候选 `0a9ecd5`；
  `3f94911`、`e518194`、`e8ce807` 和 `9f7be0f` 已分别作为持久化 trigger、
  Idempotency-Key、异步任务队列/取消和 HMAC 回调增量叠加，候选代码 `55b2d8d`；
  `1d44842` 再叠加 Qt 白名单事件桥和 MCP event trigger，候选自身 39/39、Clippy、
  release 构建和 diff check 通过。`1176a92` 再叠加 fork 主线的持久回调 outbox、同一
  delivery id 重试、重启恢复和死信状态；`9f75331` 再补受保护的死信查询和人工重放；
  `a94bba1` 补 delivered/dead 的幂等 ack；`d3dc4cf` 补 queued job 单进程重启恢复和
  running fail-closed；`7940248` 补 UTC cron、`interval_seconds`/`cron_expression`
  互斥校验和错过槽位不补跑的 at-most-once 行为；`1be154c` 补 IANA `timezone` 和真实
  DST 偏移；`86e994f` 补最多 200 个 id 的批量回调 ack；`7f938f5` 补 QuickDesk
  本机事件 buffer 和 MCP `since_server_rev` 重连补发；`fae947e` 按用户 ID
  原子持久化 replay buffer，覆盖 QuickDesk 重启；`b0ef926` 补 cron 法定节假日跳过，
  `340b463` 把 2026 日历作为 MCP 自带资源，修复上游候选缺少服务端 holiday 目录时的
  编译失败；`908696f` 补默认 2048、可配置 1~10000 的控制面 replay 容量；`0b4bbb9`
  补 cron 漏跑槽位逐条补跑；`a507aa6` 补同目录多进程回调 delivery/job claim、异常
  claim 回收和 running 有效所有权保护，`b7cffa9` 同步共享 worker 契约；`f90df36`
  增加命名 operator token、ack 操作者/确认方式持久化和首次操作者保护，`bdcf7a8`
  同步通用契约，`699e1ef` 补服务端 replay retention 配置说明，`5fe849a` 补 retry
  claim 崩溃恢复契约。候选自身现为 59/59、Clippy、QuickDesk C++
  编译到既有 `AGL` 链接点、release 构建和 diff check 通过；fork 侧 Qt CTest 15/15，
  QuickDesk 完整源码编译到链接阶段后仍受预编译 Remoting 缺失 `AGL` framework 限制。
  分支 push 仍被当前策略拒绝。数据库事务 event outbox、跨机器分布式
  队列和公平调度仍留在后续阶段
- QD-610 服务端 event reliability：fork 提交 `c39443c`，把每用户 Redis stream 从
  固定 `MAXLEN~1000` / 1 小时改成 `EVENT_REPLAY_MAX_LEN` 与
  `EVENT_REPLAY_TTL_SECONDS`，正常 publish、retry 补写和 replay 读取共用同一配置；
  `2d211c1` 再把 retry list 改为 `LMOVE` 原子 claim、per-worker processing 和 heartbeat
  崩溃恢复。独立上游候选 `codex/upstream-event-replay-retention` = `b5da22f`，直接
  基于 PR #25 的 `feat/signaling-control-plane@8998466`，9 个文件；候选 Go 全包测试、
  `go vet`、EventBus race 和直接基线 diff check 通过。分支 push 仍被当前策略拒绝
- QD-610 数据库事务 outbox 上游候选：`codex/upstream-event-outbox` = `4a3566f`，
  stacked 在 `codex/upstream-event-replay-retention@b5da22f`；只新增 `event_outbox`
  表/模型、条件 claim worker、EventBus durable
  publish 路径、`scheduled_task.failed`、设备访问、设备控制面、收藏、共享访问码、
  离线告警、远端会话、录像和访问审批生命周期原子迁移及对应测试。候选 chain 为
  `8998466 -> b5da22f -> f2d7cf7 -> 215f9ed -> f730557 -> 6ecf4e6 -> d9e231a -> 16c00d4 -> 36af1ca -> dda240d -> b14601b -> fa18755 -> 4a3566f`；
  `8de8af0` 的管理员跨账号来源 ban/unban 依赖 fork 的 QD-109 策略模型，未追加到候选。
  Go 全包测试、`go vet`、race、生产构建和单提交 diff check 通过；数据库 outbox
  行为测试在 fork 的 SQLite 套件中执行，候选本身按现有上游测试约定需要
  `QD_TEST_DATABASE_URL` 才运行 Postgres 用例。分支尚未推送
- 策略变更 outbox 增量 `6bdc057` 暂留 fork 主干：PR #25 基线还没有当前 deny/approval
  策略模型，直接 cherry-pick 会把 QD-109 等不相关功能拖进 outbox PR。等 QD-109 策略
  候选进入 stacked 基线后再追加到上游候选
- 收藏、设备控制面、远端会话、录像、访问审批、共享访问码和离线告警 outbox 增量已追加
  到上游候选，当前 `codex/upstream-event-outbox@4a3566f`；候选全包测试、vet、race、构建和单提交 diff
  check 通过，数据库行为测试仍需 `QD_TEST_DATABASE_URL`
- 指标阈值告警 outbox 增量 `c4a28ee` 暂留 fork 主干：PR #25 的 stacked 基线还没有
  QD-604 的 DeviceMetricAlertService，直接 cherry-pick 会拖入整套指标功能。等 QD-604
  服务端候选进入同一基线后，再追加最小 outbox 迁移
- QD-111 / QD-408 文件传输审计：fork 提交 `b9f414c`，控制端在 DataChannel 完成/失败后
  上报，服务端校验设备归属并按 `(actor, direction, transfer_id)` 幂等写审计。
  WebClient 66/66、生产构建、Qt CTest 15/15、Go 测试/vet 和真实 SQLite E2E 通过。
  上游候选 `c57a2f4` 自身 Go 测试/vet、四套 SDK freshness/编译、WebClient 和管理端
  生产构建通过；QuickDesk 源码编译经过新增 C++，最终链接仍受既有 `AGL` 缺失限制
- fork PR #1：<https://github.com/ilh-github/go-remote-next/pull/1>，
  OPEN / MERGEABLE / CLEAN，7 项检查全部成功
- upstream PR #25：<https://github.com/barry-ran/QuickDesk/pull/25>
- 上游 issue、PR、Roadmap 与本 fork 的融合矩阵：`docs/dev/QuickDesk功能总清单.md`
  第 16.2 节
- 两套清单合并后的实施顺序：`docs/dev/QuickDesk功能总清单.md` 第 3.2 节
- 上游映射与 PR 拆分规则：`docs/dev/QuickDesk功能总清单.md` 第 16、17 节

本地已经完成：

- 来源解封服务、房主/管理员接口、WebClient、Qt、管理后台；
- OpenAPI、TypeScript/Go/Python/Java/C#/PHP/Kotlin 七套 SDK 和 contract smoke；
- QD-111 / QD-408 文件传输审计：服务端、WebClient、Qt、七套 SDK、真实 SQLite E2E
  与上游四套 SDK 最小候选；
- 真 PostgreSQL 全量测试、两轮 live smoke、两个前端构建和四套 SDK 编译；
- QD-608 七套生成 SDK freshness、C# build、PHP Composer/lint、Kotlin Gradle build；
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
- WebClient 新增“远程文件”侧栏：经预编译 Remoting 的 `qd-skill-bridge` 调用 host
  `file-ops`，支持目录、新建、重命名、版本保留软删除、文本预览和 64 KiB 分块上传下载；
  file-ops 4/4、WebClient 62/62、生产构建和桌面/移动 Chrome 烟测通过；
  已推送 `origin/dev` / `origin/master`，并拆成上游 PR #26（file-ops）与 #27（WebClient）。
- 远端状态：PR #26 与 #27 已创建；QD-506 之后新增的 QD-510、QD-604、QD-608、
  QD-610 和 QD-111 候选不能沿用旧推送记录，必须逐分支核对 `origin`。当前环境若仍被
  `approval required by policy` 拒绝，就保持本地候选并明确记录 blocker。
- Qt 设置页支持自托管 `wss://` 信令服务器的 SPKI 固定值，Host/Client 子进程会自动
  附加 Chromium `--ignore-certificate-errors-spki-list`；Windows Host 服务安装器支持
  通过 `QUICKDESK_SPKI` 把同一固定值写入服务注册参数。
- Qt 客户端已消费自有更新通道：自动/手动检查、提示、下载进度与取消、大小和
  SHA-256 校验、可选版本推迟、强制更新、Windows 静默安装，以及 macOS DMG
  挂载、bundle/version/签名校验、暂存替换、失败回退、重开和临时包清理。
- macOS 更新助手已用真实 UDZO DMG 完成 ad-hoc E2E：正常替换成功，bundle identifier
  不匹配时拒绝替换；无写权限时会走 macOS 管理员授权。`publish_qd_mac.sh` 支持
  Developer ID hardened runtime 签名，`notarize_qd_mac.sh` 支持 notarytool、staple 和
  Gatekeeper 验证；`QUICKDESK_REQUIRE_MACOS_NOTARIZATION=1` 时缺少证书或公证凭据即失败。
- 更新下载只在同一 origin 携带 API key；跨源安装包请求已验证不会泄漏凭据。
- QD-005 真实本地 HTTP 集成测试覆盖成功下载、摘要不匹配拒绝、残件清理和跨源隔离。
- Qt 我的设备和收藏列表已用当前 Host 设备 ID 标记“本机”，不影响备注/设备名展示；
  QD-212 上游候选已在干净基线上构建并通过 QML 测试。
- Qt 点击设备时如果云快照没有访问码，会按单设备 GET 回源，成功后自动继续连接；
  并发点击合并成一次请求，QD-213 上游候选已在干净基线上构建并通过集成测试。
- WebClient 文件传输错误已按 Chromium `FileTransfer.Error.Type` 覆盖 0~7：未指定、取消、
  保存失败、协议错误、权限拒绝、磁盘不足、IO 错误和未登录；上传取消继续发送协议值 1。
- QD-214 根因已确认：macOS Host 上传 37 字节 probe 成功且落盘字节一致；Windows
  普通用户 Host 在 `EnsureUserContext()` 取得 token 时返回 `0x522`，映射为
  `type=2`；SYSTEM/session 0 没有交互桌面用户，映射为 `type=7 NOT_LOGGED_IN`。
  修复点属于 Host 服务/宿主生命周期或 Remoting，不是 WebClient 保存逻辑。
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
- Windows 自动更新已在 Parallels Windows 11 中完成全链路：带 `a982a10` 修复的
  4.2.8.0 基线包为 90,095,272 字节、SHA-256
  `0249d1d3573e26b4bba9064e0cc8879bf94301c29c50c71c11ed426a1c1830a3`；
  4.2.9.0 升级包为 90,084,629 字节、SHA-256
  `8e5e32cb038f5f6757bef58e2ed0907c5918e131e0d118d61de41f3090379383`。
  客户端真实发现并下载更新，下载文件摘要完全一致；点击“安装并重启”后，旧进程
  PID `12460` 退出，新进程 PID `6412` 自动启动。关于页和卸载注册表均为
  `4.2.9.0`，QuickDeskHost 服务保持 RUNNING，升级临时目录已清理。
- QD-005 根因修复：旧实现依赖 Inno `/RESTARTAPPLICATIONS`，但 Qt 客户端在启动安装器后
  会主动退出，安装器没有可恢复的 Restart Manager 应用，所以升级后不会自动启动。
  `a982a10` 改为 `/LAUNCHAPP`，由安装器完成后显式启动新版本；新版本启动时清理不高于
  当前版本的更新目录，同时保留尚未安装的更高版本下载。
- QD-005 macOS 当前边界：自动替换链路已经本地验收，但本机
  `security find-identity -v -p codesigning` 为 0 个有效身份，也没有 Apple 公证凭据；
  Developer ID 签名、notarytool、staple、Gatekeeper 和公证包真实升级不能宣称完成。
- QD-005 上游候选尚未创建：实测把自托管更新提交 `cae77ff` 直接 cherry-pick 到
  `upstream/dev@f6651f0` 会在 `QuickDesk/CMakeLists.txt`、`MainController`、
  `localconfigcenter`、中英文翻译和测试 CMake 基线发生冲突，`tests/CMakeLists.txt`
  在上游仍不存在。不能为了开 PR 把整个 fork 控制面带进去；等 PR #25/更新通道基线
  确定后，再从该基线提取更新管理器、安装器和测试的最小增量。
- QD-215 Windows 发布依赖校验已落地：`verify_qd_win_release.ps1` 检查 x64 PE、
  `platforms/qwindows.dll`、VC Runtime 和 Qt 直接依赖，`publish_qd_win.bat` 与
  `package_qd_win.bat` 都会在缺依赖时中止。Windows 11 上 publish、Inno Setup 打包
  和缺 Runtime/平台插件/Qt 依赖的负向测试均通过；Windows 自动更新复测包落在
  90,084,629 字节，QD-005 修复后的 4.2.9.0 安装包已完成真实升级。
- QD-215 上游候选 `f449b87` 已从 `upstream/dev` 单独切出，只含 3 个文件。候选校验器
  已在同一 Windows 发布目录上通过“正常包接受、缺 Runtime/Qt 插件/Qt 依赖均拒绝”的
  回归；`git push` 仍被当前执行策略拒绝，尚未创建 issue #7 PR。
- `version=4.2.9.0` 已随 `a982a10` 提交；该版本号对应完成 Windows 真机升级的构建，
  不再只是未提交的临时验证值。
- QD-214 的前置 SPAKE2 阻断已经解决：WebClient 现在复刻 BoringSSL
  password-scalar 低三位修正，固定向量、真实 macOS Host 和 Windows Host 均通过验证
  哈希并进入 DataChannel。
- QD-214 当前状态为 `[!]`，阻塞点是 Windows Host 用户上下文，不是上传协议。下一步
  需要把 QuickDesk Host worker 启动到活动交互 session，或向 Remoting 提正式需求；
  在此之前不再扩大 WebClient 的错误提示补丁。
- QD-006 / QD-809 已在 fork 完成并提交 `c77dcae`：SQLite 单文件、`DB_DRIVER=sqlite`
  自动选择进程内 Redis、健康检查识别 sqlite，空目录启动/重启 E2E、全量 Go 测试、
  database race 和 `go vet` 通过。该模式只支持一个信令进程，进程内 Redis 重启即失；
  上游候选必须保留这两个边界，并基于 PR #25 的 SignalingServer 基线重做最小增量。
- QD-006 上游候选 `7758456` 已从 PR #25 基线独立切出，只含 SignalingServer 配置、
  SQLite/embedded Redis、健康检查、测试和 E2E，不带 fork 清单或 QD-607 调度动作。
  候选测试与 fork 等价通过，当前等待 PR #25 合并后再 rebase 到 `upstream/dev`。
- QD-604 / 指标型告警已在 fork 完成并提交 `37dd8aa`：Host 通过内部 sys-info 每 60 秒
  上报 CPU、内存和最高使用率磁盘；服务端只保留单设备最新快照，CPU/内存/磁盘独立
  episode，超限事件进入持久通知 outbox；Web/Qt 均有规则和当前值入口。真实 SQLite
  服务 E2E 已把 `device.metric.alert` 投递到 Webhook。
- QD-604 验证：SignalingServer 全量 Go 测试、相关包 race、`go vet`、真实 SQLite+Webhook
  E2E、WebClient 47/47 和生产构建、四套 SDK freshness、QuickDesk Qt 6.8.3 Debug
  完整构建、QML lint、离屏启动和 12/12 CTest 通过。
- QD-604 上游边界：它依赖 PR #25 的控制面、OpenAPI 和通知基线。PR #25 合并后先拆
  SignalingServer + 契约 + SDK + E2E，再拆 Web/Qt 客户端；fork-only 清单和部署文档
  不进入上游。
- QD-604 上游服务端候选已验证：`codex/upstream-qd-604-metrics@0ea9e7c` 基于
  `7758456`，自身 `go test -p 1 -count=1 ./...`、`go vet ./...`、真实 SQLite+Webhook
  E2E、四套 SDK freshness 和 Go/TS/Python/Java 编译全部通过。远端 push 仍被策略拒绝，
  因此尚未创建 stacked PR。
- QD-911 已在 fork 完成 `SECURITY.md` 并提交 `3021ca9`：优先 GitHub Private
  Vulnerability Reporting，入口不可用时使用维护者 GitHub 公开资料中的邮箱；明确禁止
  在公开 issue 披露细节，并给出支持版本、范围内外、协调披露和 maintainer 检查项。
- QD-911 上游候选 `9862717` 已验证只含 1 个文件，`git diff --check upstream/dev...HEAD`
  通过。由于 issue #6 仍开放，且 `gh api
  repos/barry-ran/QuickDesk/private-vulnerability-reporting` 返回 `enabled=false`，QD-911
  只能保持 `[~]`，不能把本地文档说成上游私密入口已生效。
- QD-115 已在 fork 完成安全边界文档并提交 `3474203`：明确 device_secret 只有 Argon2id
  哈希，但 access_code 明文、signal token/审批/设备归属由服务器控制，因此当前信令
  服务器仍是设备访问信任方；目标方案是 Host 身份密钥 + Owner 签名 grant + Host
  本地验证，服务器降级为路由。
- QD-115 上游候选 `1ca2543` 已验证只含 1 个文档，`git diff --check upstream/dev...HEAD`
  通过。该候选是设计输入，不代表风险已修复；QD-115 的 8 条协议验收项完成前保持 `[~]`。

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
- `dev` 必须包含 `37dd8aa`；
- `dev` 必须包含 `3021ca9`；
- `dev` 必须包含 `3474203`；
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
- `codex/upstream-qd-111-file-transfer-audit` 必须指向 `c57a2f4`，且父提交必须是
  `8998466`；
- `codex/upstream-qd-006-sqlite` 必须指向 `7758456`；
- `codex/upstream-qd-604-metrics` 必须指向 `0ea9e7c`，且父提交必须是 `7758456`；
- `codex/upstream-qd-911-security` 必须指向 `9862717`，且父提交必须是 `f6651f0`；
- `codex/upstream-qd-115-trust-model` 必须指向 `1ca2543`，且父提交必须是 `f6651f0`；
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
  codex/upstream-qd-111-file-transfer-audit \
  codex/upstream-qd-006-sqlite \
  codex/upstream-qd-604-metrics \
  codex/upstream-qd-911-security \
  codex/upstream-qd-115-trust-model \
  codex/source-unban \
  codex/upstream-access-bans
```

推送后确认：

```bash
git rev-parse dev
git rev-parse origin/dev
git rev-parse codex/upstream-access-bans
git rev-parse origin/codex/upstream-access-bans
git rev-parse codex/upstream-qd-006-sqlite
git rev-parse origin/codex/upstream-qd-006-sqlite
git rev-parse codex/upstream-qd-604-metrics
git rev-parse origin/codex/upstream-qd-604-metrics
git rev-parse codex/upstream-qd-911-security
git rev-parse origin/codex/upstream-qd-911-security
git rev-parse codex/upstream-qd-115-trust-model
git rev-parse origin/codex/upstream-qd-115-trust-model
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
  无 Remoting 改动；候选分支 `codex/upstream-cross-tab-coordination` = `93eee37`，
  仅 9 个文件，已通过 `npm test` 6/6、真实 Chrome 三标签 E2E 1/1、
  `npm run build` 和 `git diff --check`。
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
- `feat(audit): record controller file transfers`：QD-111 / QD-408 的控制端文件传输
  事件；候选分支 `codex/upstream-qd-111-file-transfer-audit` = `c57a2f4`，基于 PR #25
  的 `feat/signaling-control-plane@8998466`。候选 44 个文件，只含审计 API、OpenAPI、
  Web/Qt 接线、smoke 和 PR #25 基线已有的四套 SDK。候选 Go 测试/vet、四套 SDK
  freshness、Go/Python/Java/TypeScript 编译、WebClient 和管理端生产构建通过；Qt
  源码编译经过新增 C++，最终链接仅受既有 `AGL` 缺失影响。
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
- `feat(chat): add in-session text chat`：QD-316 会话内聊天，候选分支
  `codex/upstream-qd-316-session-chat-v2` = `96b36d0`，基于 `upstream/dev@5db316c`。
  只包含内建 `session_chat` bridge、Qt/Web 聊天 UI、双语文案和 Web 测试，不含 fork
  清单、文件管理器或业务控制面。候选自身 Web 测试 4/4、生产构建、Qt 源码/QML 编译
  通过，最终 App 链接仍受既有 macOS `AGL` 缺失影响；`git push origin` 被当前策略
  拒绝，因此尚未创建 upstream PR。
- 上游 issue #18 连接历史：上游 `dev` 已存在 `ConnectionHistory` 和历史列表 UI，
  当前不应重复实现；先核对完整功能，再决定是否在 issue 中说明可关闭。

## 5. 上游暂不合并时

不阻塞 fork 主线：

- 所有功能继续进入 `go-remote-next/dev`；
- fork PR 保持可独立发布；
- 上游候选分支保留 rebase 能力，定期对 `barry-ran/QuickDesk:dev` 重新应用；
- 通用 WebClient、Qt 和管理后台页面在控制面 #25 合并后按页面单独上游；
- 只有品牌、部署配置、本 fork 专属业务和旧 Go 迁移文档留在 fork。

## QD-908 soak 正式跑（2026-10-24）

- 200 路 7200 秒正式跑已执行完毕，`headroom` 从 1 小时提高到 2 小时后跑完，不再被
  `node:test` 掐断；
- 结果：`mainFps` 0.77、`heapGrowthMb` 554.46 MiB 未过门禁；时长、118 次采样、
  三个采样 iframe 帧持续推进、零页面错误均通过；
- 同参数 42 秒短跑对照：`mainFps` 9.17、heap 增长 -118.76 MiB，证明是随运行时间累积的
  劣化，不是 200 路固有代价；
- 处理：不调阈值，作为待定位的稳定性缺陷跟踪，详见联调记录第 224 节；
- 该跑为本地门禁，不涉及上游 PR。

## 跨机器队列候选评估（2026-10-25）

- 结论：**不可作为最小 PR 回贡**。上游 `barry-ran/QuickDesk:dev` 的
  `quickdesk-mcp/src/` 下没有 `workflow_job.rs`（只有 config、event_bus、
  http_transport、main、memory、server、trust、workflow、ws_client）。
  `workflow_job.rs`、`fair_queue.rs`、`distributed_claim.rs`、`callback.rs`、
  `idempotency.rs`、`trigger.rs`、`workflow_trigger.rs`、`delegation.rs`、
  `recovery.rs` 整个是 fork 自有模块。
- 因此 `93a0d13` 的 peer 目录发现能力依赖 fork 独有的 job 存储格式、
  `FairQueue` 和 `DistributedClaimStore`，单独摘出来上游无法编译。
- 处理：该功能留在 fork 主干（`Fork-only`），与 QD-610 其余部分一致。
  已在 `QuickDesk功能总清单.md` 的 QD-610 条目中记录本轮内容。
- 教训：判断能否回贡的第一步是 `git ls-tree --name-only upstream/dev <路径>`，
  确认上游存在同一文件；不要先写代码再找基线。

## MCP 自死锁修复候选（2026-10-25）

- 缺陷：`quickdesk-mcp` 的 `MemoryStore::log_operation` 持着连接互斥锁调用
  `record_failure`，后者再抢同一把非重入锁。任何一次 `success=false` 的工具调用都会
  把 MCP 服务端永久挂死。上游 `barry-ran/QuickDesk:dev@5db316c` 的
  `src/memory.rs` 第 290 行与第 310 字字相同，缺陷同样存在。
- 候选分支：`codex/upstream-mcp-failure-lock` = `c0202f1`，基于
  `upstream/dev@5db316c`，只含 1 个提交、1 个文件（+116 行：修复 + 3 个回归测试 +
  测试用 `open_at` 入口）。
- 验证：修复前 3 个测试全部以看门狗超时失败（`timed out: the store is deadlocked,
  not slow`），修复后全部通过。双向验证过，不是只跑绿。
- 已推送到 fork 并创建上游 PR #31：
  https://github.com/barry-ran/QuickDesk/pull/31
- PR 说明中已声明上游 `dev` 的 `cargo test` 当前编不过（`src/event_bus.rs:253` 与
  `:274` 调用 `wait_for` 缺第 4 个参数 `check_history`），与本次改动无关；是在本地
  临时打补丁跑通测试后还原的，未包含在候选内。
- fork 侧同一修复提交为 `33c6fcb`，`dev` 与 `master` 已同步并推送。

## QD-215 Windows 发布依赖校验候选（2026-10-25）

- 候选分支：`codex/upstream-qd-215-win-runtime-check`，已从 `upstream/dev@5db316c`
  rebase 成为 `8385856`（原 `f449b87` 基于 `f6651f0`，落后一个提交）。
  只含 3 个文件：`scripts/verify_qd_win_release.ps1`（新增）、
  `scripts/publish_qd_win.bat` 与 `scripts/package_qd_win.bat`（接入校验并失败即中断）。
- 已推送到 fork 并创建上游 PR #32：
  https://github.com/barry-ran/QuickDesk/pull/32
- PR 说明中已声明：本机（macOS）没有 `pwsh`，该校验器未从当前 checkout 执行过；
  Windows 上的 publish/package 通过与负向用例证据来自当时的 Windows 运行。
- 仍未取得 issue #7 的原始坏安装包，因此这是对该类失败的防护，不是对那个具体包的确认修复。
- 此前记录「push/PR 被策略阻塞」已不成立：PR #31 与本次 PR #32 均成功创建，
  远端写入可用。后续候选直接推送并开 PR，不必再按阻塞处理。

## QD-314 独占控制跨实例（2026-10-26）

- fork 提交：`fb43070`，`dev` 与 `master` 已同步。
- 上游候选：**未准备**。`ControlGrantStore` 依赖 fork 已有的
  `RevokeCooldown` 同类 Redis 基建与 `remote_sessions` 控制面基线，且上游
  `quickdesk-mcp` 没有对应模块；`control_grant` / `control_revoked` 两个帧
  本身是上游可独立审查的纯函数，可作为后续最小候选从 `upstream/dev` 切出。
- 远端写入当前可用（PR #31、#32 均已成功创建），本条不是阻塞记录。

## QD-909 审计链读侧（2026-10-26）

- fork 提交：见本轮 `feat(signaling)`，`dev` 与 `master` 同步。
- 上游候选：**未准备**。依赖 fork 已有的 `file-transfer-audits` /
  `clipboard-audits` 写端点与控制面基线（PR #25 系列），因此与
  `codex/upstream-qd-111-file-transfer-audit` 同一基线约束，等 #25 合并后
  作为 stacked 增量拆出。
- SDK 已重新生成并提交：见第 208 节，7 套 freshness 门禁全部通过。
