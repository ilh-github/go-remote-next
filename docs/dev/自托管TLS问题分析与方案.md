# 自托管 TLS 问题分析与方案

> 关联 Issue：`barry-ran/QuickDesk#20`（自托管服务客户端连接失败，OPEN）
> 调查对象：QuickDesk `v4.2.0.0` + quickdesk-remoting `v1.0.3`
> 调查性质：只读。未修改 quickdesk-remoting 任何源码
> 记录日期：2026-10-15

## 0. 结论摘要

- 这是 **Remoting 内核（quickdesk-remoting）的真实缺陷**，不是 Nginx 配置、证书链或限流问题。
- `quickdesk_network_service.cc` 在构造 `NetworkContext` 时，传入的是一个**未绑定的证书校验服务管道（dangling remote）**，源码注释自认为"仅适合 localhost 开发"。任何真实 TLS 握手都会因为无法完成证书校验而中断，Chromium 侧返回 `net_error = -3 (ERR_ABORTED)`。
- 该文件自 2026-01-16 起**未再被任何提交修改**，Remoting 默认分支 `140.0.7339.249` 与 `v1.0.3` 内容一致，**上游没有修复**，QuickDesk 与 Remoting 两个仓库里也没有相关的修复 PR 或后续讨论。
- 最省成本的可行修复：**不改 Remoting 源码**，在 QuickDesk 产品层给 host/client 子进程追加 `--ignore-certificate-errors-spki-list=<服务器证书 SPKI>`。它只会放行你自己服务器公钥对应的证书，**不需要编译 Chromium**，只需重编 QuickDesk 的 Qt 应用。
- 彻底修复：接通真实的 `CertVerifierService`，这需要改动 Remoting 并**编译 Chromium/Remoting**。
- 该缺陷只影响 `https/wss`。`http/ws` 能跑通，这正是"本地开发一直是好的、公网自托管会挂"的原因。

## 1. 问题链路核实

### 1.1 Issue #20 事实

| 项 | 值 |
|---|---|
| 编号 | barry-ran/QuickDesk#20 |
| 标题 | 自托管服务客户端连接失败 |
| 状态 | OPEN，0 评论，2026-09-12 创建 |
| 环境 | 服务端为 Docker 镜像自托管；客户端为预编译 v4.2.0.0 |
| 现象 | `signal-token ok` 后，`wss://<域名>:4160/v1/realtime/signal` 在 TLS 阶段失败：`handshake failed ... net_error -3`、`WebSocket opening handshake was canceled`、`response_code=-1` |

`response_code=-1` 说明客户端在收到任何 HTTP 响应之前就中止了握手，因此不是 Nginx 返回的限流错误；紧随其后的 `429` 属于客户端重试命中了应用层的一秒最小请求间隔。

### 1.2 源码证据

缺陷位于 `remoting/quickdesk/signaling/quickdesk_network_service.cc`（v1.0.3，第 86-99 行，文件 SHA `ecdcb2a0…`）：

```cpp
// Create a minimal CertVerifierServiceRemoteParams with dangling remote
// This satisfies the DCHECK but doesn't actually verify certs
// Acceptable for localhost development
context_params->cert_verifier_params =
    network::mojom::CertVerifierServiceRemoteParams::New();

// Bind to a dangling remote (will never be used for localhost WebSocket)
mojo::PendingRemote<cert_verifier::mojom::CertVerifierService> dangling_remote;
auto receiver = dangling_remote.InitWithNewPipeAndPassReceiver();
// Let the receiver drop (no actual service bound)
context_params->cert_verifier_params->cert_verifier_service =
    std::move(dangling_remote);
```

这里的 `dangling_remote` 只创建了一对管道后就把接收端丢弃，**接收端从未绑定任何实现**。它唯一的作用是绕过 `NetworkContext` 内部的 `DCHECK(params_->cert_verifier_params)`。

### 1.3 失败机制（为什么会是 -3）

TLS 握手时，`NetworkContext` 会用这个 `cert_verifier_params` 构造证书校验器（`services/network/network_context.cc`，`NetworkContext::MakeURLRequestContext`，第 2643-2656 行）：

```cpp
DCHECK(params_->cert_verifier_params);
cert_verifier = std::make_unique<cert_verifier::MojoCertVerifier>(
    std::move(params_->cert_verifier_params->cert_verifier_service),
    std::move(params_->cert_verifier_params
                  ->cert_verifier_service_client_receiver),
    ...);
```

`MojoCertVerifier` 把校验请求发向那个**已经断开**的管道，请求无法返回结果，TLS 握手因此中止。对 `ws://`/`http://`（明文本地开发）不触发证书校验，所以本地一直是好的。

### 1.4 影响版本

- quickdesk-remoting 自 `v0.0.1`（2026-02-25）起就在这条自研信令路径上；`quickdesk_network_service.cc` 只有一个提交命中（2026-01-16 的 `feat: quickdesk host使用websocket信令`），结论是 **从引入该文件起至 v1.0.3 全部受影响**。
- QuickDesk 每个 release 都会同步 Remoting 版本（`feat: update remoting vX`），`v4.2.0.0` 的 `cmake/quickdesk-remoting.cmake` 固定的是 `v1.0.3`。因此 **预编译 v4.2.0.0 一定受影响**。
- 只有把信令放到 `https/wss` 时才暴露；纯 `ws` 或不走 TLS 的部署不受影响。

### 1.5 是否已有修复

| 检查项 | 结果 |
|---|---|
| Remoting 默认分支 `140.0.7339.249` 上该文件 | 与 v1.0.3 同一 SHA `ecdcb2a0…`，未修改 |
| Remoting 最新 release | `v1.0.3`（2026-08-16），无更新 |
| Remoting 仓库相关关键字（cert/TLS/wss） | 无相关 Issue |
| QuickDesk 仓库相关关键字（自托管/TLS/证书/wss） | 无相关 Issue/PR |

结论：**上游未修复，Issue #20 仍处于开放且无回应状态。**

## 2. 三类方案

### 方案 A：不改 Remoting 源码

#### A1（推荐）：产品层钉扎服务器证书 SPKI

原理：`NetworkContext` 构造证书校验器时，会先包一层 `IgnoreErrorsCertVerifier`（`services/network/ignore_errors_cert_verifier.cc`）。只要进程命令行带 `--ignore-certificate-errors-spki-list=<hash>`，且服务器证书链里的某张证书公钥哈希命中该列表，`Verify()` 会**同步直接返回 `net::OK`**（该文件第 88-98 行），**完全不触碰那个已断开的证书校验管道**：

```cpp
if (ignore_errors) {
  verify_result->Reset();
  verify_result->verified_cert = params.certificate();
  ...
  return net::OK;   // 不调用底层 dead verifier_
}
```

开关名定义在 `services/network/public/cpp/network_switches.cc`：

```cpp
const char kIgnoreCertificateErrorsSPKIList[] =
    "ignore-certificate-errors-spki-list";
```

改法（QuickDesk 产品层，仅需重编 Qt 应用，**不编译 Chromium**）：在 `QuickDesk/src/manager/ProcessManager.cpp` 的 `startProcess()` 里，给 host/client 子进程命令行追加该开关：

```cpp
// 仅对 host/client 子进程注入证书钉扎；GUI 进程不需要
QString spkiPins = m_tlsSpkiPins;   // 例如 "AbCd...=,EfGh...="
if (!spkiPins.isEmpty()) {
    arguments << QString("--ignore-certificate-errors-spki-list=%1").arg(spkiPins);
}
```

服务器证书 SPKI 的 SHA-256 + Base64 计算：

```bash
openssl s_client -connect <域名>:<端口> -servername <域名> </dev/null 2>/dev/null \
  | openssl x509 -pubkey -noout \
  | openssl pkey -pubin -outform der \
  | openssl dgst -sha256 -binary \
  | openssl base64
```

注意事项：

- 这是"只钉扎你自己的公钥"，比无脑忽略全部证书安全，但仍绕过标准链验证。
- 若用 Let's Encrypt 自动续期且每次换私钥，SPKI 会变，钉扎失效，需要固定私钥或把新 SPKI 重新下发。
- Windows 上如果 host 是以 **Windows 服务**方式启动（`ProcessManager` 走命名管道路径），则不会经过 `startProcess()`，开关需在服务注册/命令行处设置。

#### A2：信令降级为明文 `ws://` + 外部加密通道

- 完全不改任何代码和二进制：把信令暴露为 `ws://`，再用 VPN / SSH 端口转发 / WireGuard 等在传输层加密。
- 优点：零编译。
- 缺点：公网直连不安全，运维复杂；WebClient 走浏览器时还会遇到混合内容限制，因此只适合内网或隧道场景，不作为推荐。

> 补充：如果不想重编 QuickDesk，也可以在 Windows 上把真实 `quickdesk_host.exe`/`quickdesk_client.exe` 改名，用一个补参数的 shim 顶替。可作应急，但脆弱，不作为方案。

### 方案 B：最小改 Remoting（长期正确方案）

把那个 dangling remote 换成真实的 `CertVerifierService`：

1. 在 `QuickDeskNetworkService::InitializeOnNetworkThread()` 中，于 network 线程创建证书校验服务工厂（`services/cert_verifier/cert_verifier_service_factory.{h,cc}`）。
2. 通过 `CertVerifierServiceFactory::GetNewCertVerifier(...)` 拿到已绑定的 `pending_remote<CertVerifierService>`，并同时提供 `CertVerifierServiceClient` 接收端。
3. 把它写入 `context_params->cert_verifier_params->cert_verifier_service`，替换掉当前的 `dangling_remote`。
4. `NetworkContext` 侧原本就会为 `MojoCertVerifier` 注入 `cert_net_fetcher` 用的 URLLoaderFactory，无需额外处理。

- 优点：符合 Chromium 正常设计，可支持标准 CA、Chrome Root Store、证书更新。
- 代价：**必须编译 Chromium/Remoting**，并随 Remoting 版本维护该补丁。

### 方案 C：完全替换证书校验

- 用 `NetworkContext::SetCertVerifierForTesting()`（`services/network/network_context.{h,cc}`，第 196 / 975 行）注入一个"接受一切"的 `net::CertVerifier`，让所有证书都通过。
- 或自行实现 `net::CertVerifier` 并注入做自定义校验。
- 优点：实现简单、能立刻连通。
- 缺点：**完全丧失证书校验，安全性最低**；`SetCertVerifierForTesting` 名为测试用，被用作生产路径属于滥用；同样**需要编译 Chromium**。
- 仅在你完全自用、只在受控网络里跑时才可接受。

### 方案对比

| 方案 | 改 Remoting 源码 | 需编译 Chromium | 安全性 | 维护成本 | 适用 |
|---|---|---|---|---|---|
| A1 SPKI 钉扎 | 否 | 否（只编 QuickDesk） | 中（钉扎单公钥） | 低，随证书轮换更新 | **推荐落地** |
| A2 明文 + 隧道 | 否 | 否 | 中（依赖外部加密） | 中，运维复杂 | 内网/应急 |
| B 接通真实校验 | 是 | 是 | 高 | 高，需跟随上游 | 长期正确解 |
| C 关闭校验 | 是 | 是 | 低 | 中 | 自用应急 |

## 3. 推荐方案

1. **短期（立刻可用）**：采用 **A1**，在 QuickDesk 产品层给 host/client 注入 `--ignore-certificate-errors-spki-list`，把自托管服务器的证书公钥钉死。不需要编译 Chromium，改动量小、可控，且只放行你自己服务器的公钥。
2. **中期（正确解）**：把 **B** 的补丁提交给 Remoting（或维护自己的 Remoting fork），让自托管 TLS 走标准证书校验，并考虑向上游提 PR / 提 Issue。
3. **不采用 C** 作为常规方案，除非明确接受"关闭证书校验"的安全代价。
4. 无论选哪个，都需要在 **PD Windows 虚拟机**上按第 4 节做端到端验证，并留存日志证据。

## 4. PD Windows 虚拟机复现与验证步骤

前提：

- 一台 Windows 10/11 虚拟机（PD 中运行）。
- 信令服务器已部署（Docker 镜像或源码），并已配置 `https/wss`（Certbot 签发的正式证书或自签证书均可）。
- 客户端使用 QuickDesk 预编译 `v4.2.0.0`。

步骤：

1. 服务器侧自检：`curl -fsS https://<域名>/health` 应返回正常；`wss` 地址形如 `wss://<域名>/v1/realtime/signal`（端口按实际部署，例如 `:4160`）。
2. 记录服务器证书 SPKI（用于 A1）：执行第 2 节 A1 的 openssl 命令，保存输出。
3. 在虚拟机中安装 QuickDesk `v4.2.0.0`，登录并在服务器设置里填入自托管 `wss://` 地址（按当前版本实际字段填写）。
4. 触发一次连接（发起或加入会话），然后查看 QuickDesk 传递给 host/client 的日志目录（`ProcessManager` 会传 `--log-dir`），预期复现：
   - `handshake failed ... net_error -3`
   - `WebSocket opening handshake was canceled`
   - `response_code=-1`
5. 验证 A1：将带 `--ignore-certificate-errors-spki-list=<SPKI>` 的 QuickDesk 构建（或 shim）替换到虚拟机，重复第 3-4 步，预期握手成功、会话建立；再故意把 SPKI 改错，预期重新失败（证明钉扎生效，而非"关闭校验"）。
6. 验证 B（如已构建）：替换带真实证书校验的 host/client，重复第 3-4 步预期成功；再换成一张不受信证书，预期失败（证明标准校验生效）。
7. 记录每次的 `net_error` 与最终结果，作为回归基线。

## 5. 证据链接

- Issue #20：https://github.com/barry-ran/QuickDesk/issues/20
- 缺陷源码（v1.0.3，第 86-99 行）：https://github.com/barry-ran/quickdesk-remoting/blob/v1.0.3/remoting/quickdesk/signaling/quickdesk_network_service.cc#L86-L99
- 同文件默认分支（SHA 一致）：https://github.com/barry-ran/quickdesk-remoting/blob/140.0.7339.249/remoting/quickdesk/signaling/quickdesk_network_service.cc#L86-L99
- 证书校验器构造位置：https://github.com/barry-ran/quickdesk-remoting/blob/v1.0.3/services/network/network_context.cc#L2643-L2656
- `CreateForTesting` / `SetCertVerifierForTesting`：https://github.com/barry-ran/quickdesk-remoting/blob/v1.0.3/services/network/network_context.h#L189-L196
- `IgnoreErrorsCertVerifier` 放行逻辑：https://github.com/barry-ran/quickdesk-remoting/blob/v1.0.3/services/network/ignore_errors_cert_verifier.cc#L88-L99
- 开关名 `ignore-certificate-errors-spki-list`：https://github.com/barry-ran/quickdesk-remoting/blob/v1.0.3/services/network/public/cpp/network_switches.cc#L31-L32
- WebSocket 经 NetworkContext 建立：https://github.com/barry-ran/quickdesk-remoting/blob/v1.0.3/remoting/quickdesk/signaling/quickdesk_signal_strategy.cc
- Remoting 版本固定：https://github.com/barry-ran/QuickDesk/blob/v4.2.0.0/cmake/quickdesk-remoting.cmake
- 子进程启动参数位置（本 fork）：`QuickDesk/src/manager/ProcessManager.cpp`（`ProcessManager::startProcess`）

## 6. 结论

- 自托管 `https/wss` 连不上，根因在 Remoting 内核对证书校验服务的实现，而不是服务端配置。
- 上游截至 v1.0.3 未修复。
- 最合理的落点组合是：**立刻用 A1（产品层 SPKI 钉扎）打通，随后用 B（接通真实证书校验）作为长期方案**；A2 仅作应急，C 不推荐。
