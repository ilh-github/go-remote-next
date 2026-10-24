# 自托管 TLS 真实 CertVerifier Remoting 需求

**状态**：2026-10-25，待 `quickdesk-remoting` 接通真实证书校验服务。

**对应清单**：QD-114（上游 issue #20）。

**前置调查**：`docs/dev/自托管TLS问题分析与方案.md`（含源码行号、文件 SHA 与影响版本）。

**边界**：本文不要求在 fork 里替换 Remoting 的网络栈，也不接受"关闭证书校验"作为常规方案。

## 1. 要解决的问题

`quickdesk-remoting` v1.0.3 的 `remoting/quickdesk/signaling/quickdesk_network_service.cc`
第 86-99 行为绕过 `NetworkContext` 内部的 `DCHECK(params_->cert_verifier_params)`，
构造了一个 `dangling_remote`：只建管道、随即使接收端析构，**从未绑定任何实现**。

`NetworkContext` 随后用它构造 `MojoCertVerifier`（`services/network/network_context.cc`
`MakeURLRequestContext`，第 2643-2656 行），校验请求发向一个已断开的管道，永不返回，
TLS 握手因此中止。`ws://` / `http://` 明文不触发证书校验，所以本地开发一直是好的——
这正是它能存活到 v1.0.3 的原因。

影响面：自该文件引入起至 v1.0.3 全部版本；`v4.2.0.0` 的 `cmake/quickdesk-remoting.cmake`
固定 `v1.0.3`，因此预编译 release 一定受影响。

fork 侧已有 A1 缓解：Qt 子进程支持 SPKI 列表，Windows 服务安装支持 `QUICKDESK_SPKI`
（见 `docs/信令服务器部署.md`）。但**钉扎公钥是临时信任机制，不能替代 CA 校验**，
且它覆盖不到 Remoting 自己发起的连接路径。本需求单针对的是根治。

## 2. 推荐方案

把那个 dangling remote 换成真实的 `CertVerifierService`：

1. 在 `QuickDeskNetworkService::InitializeOnNetworkThread()` 中，于 network 线程创建
   证书校验服务工厂（`services/cert_verifier/cert_verifier_service_factory.{h,cc}`）。
2. 通过 `CertVerifierServiceFactory::GetNewCertVerifier(...)` 取得已绑定的
   `pending_remote<CertVerifierService>`，同时提供 `CertVerifierServiceClient` 接收端。
3. 写入 `context_params->cert_verifier_params->cert_verifier_service`，替换 `dangling_remote`。
4. `NetworkContext` 侧原本就会为 `MojoCertVerifier` 注入 `cert_net_fetcher` 使用的
   URLLoaderFactory，无需额外处理。

不采用的两个方向，理由写在这里以免复审时重新讨论：

- **不接受 `SetCertVerifierForTesting()` 注入"接受一切"的 verifier**。它名为测试用，
  用于生产路径是滥用，且完全丧失证书校验。
- **不自行实现 `net::CertVerifier` 做自定义校验**。除非有明确的产品理由，标准 CA 与
  Chrome Root Store 才是自托管默认该有的行为。

## 3. 兼容性与权限要求

- 明文 `ws://` 本地开发路径必须保持可用：该校验只在 TLS 握手路径上被触发，
  不应给本地开发新增失败模式。
- 自签证书的自托管部署在校验接通后需要被信任或被固定。SPKI 固定作为**过渡**继续保留，
  但契约上要说明它是临时机制，并要求证书轮换时同步轮换固定值。
- 校验失败必须可诊断：错误要能区分"证书不受信任"、"主机名不匹配"和"握手被中断"，
  而不是现在这样一个无法归因的 `-3`。
- 不引入新的网络权限面；`cert_net_fetcher` 仍只用于 AIA/OCSP 等标准校验链。

## 4. 验收用例

1. 连接受公开 CA 签发的自托管信令服务器，握手成功且无证书警告；
2. 连接受自签证书但已被系统信任的服务器，握手成功；
3. 连接受自签证书且未被信任的服务器，握手失败并给出可归因的错误，
   而不是无法定位的握手中断；
4. 主机名与证书 SAN 不匹配时失败，且错误指明主机名不匹配；
5. 证书过期时失败并指明过期；
6. 明文 `ws://` 本地开发路径不受影响，仍能连通；
7. 以上用例在 Windows 与 macOS 各跑一遍；
8. 补齐 `test/` 下针对该校验路径的自动化用例，避免再次出现"只有明文被测试过"。

## 5. 为什么当前停在 fork 外

仓库里的 `quickdesk_host` 是预编译 Remoting 二进制。接通真实 `CertVerifierService`
必须修改并编译 `quickdesk-remoting`，这不是 fork 能在二进制外围完成的改动；
fork 侧能做的只是 SPKI 钉扎这一层缓解，已经落地。

因此正确路径是向 `barry-ran/quickdesk-remoting` 提交该协议与实现，或维护自己的
Remoting fork 并随上游版本跟进。本文把缺陷定位、推荐方案和不采用的方向写成
可提交的需求，供上游维护者评估。
