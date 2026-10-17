# Signaling Server Trust Boundary for Device Control

> Status: design conclusion for upstream issue #10. The fork tracks the remaining
> implementation work as QD-115.
>
> 结论先行：当前架构要求信任信令服务器。服务器不能从数据库恢复
> `device_secret`，但会保存明文 `access_code`，并负责签发和消费 signal token、配对
> Host/Controller、转发 WebRTC 信令。因此恶意或被攻陷的服务器可以把自己当作控制端，
> 完成当前协议下的认证并尝试接管设备。要消除这一能力，必须把最终授权从服务器移到
> Host 可独立验证的签名声明；只换数据库字段、增加管理员权限或继续加 server-side
> approval 都不够。

## 1. 当前实现事实

| 对象 | 当前存储/使用方式 | 被服务器攻陷后的能力 |
|---|---|---|
| `device_secret` | 只保存 Argon2id 哈希；明文只在 provision/rotate 响应中出现一次 | 不能从数据库反推出原密钥，也不能直接冒充 Host 调 device API；可在实时进程中拦截新 provision，或修改代码/令牌存储 |
| `access_code` | `devices.access_code` 明文保存；普通用户 REST、实时 snapshot 和管理端都能读取 | 可直接知道当前接入码，并在 `access-code:verify` 中通过校验 |
| `signal_token` | 随机 token 存 Redis，带 `role/device_id/client_id`，由服务器签发和消费 | 可伪造 Host 或 Controller token，不依赖用户或设备同意 |
| 设备所有权 | `devices.user_id`、`user_devices` 由服务器维护 | 可改绑、解绑或伪装所有权变化 |
| 访问审批 | 待审批记录和批准决定均经过服务器 | 可伪造批准、拒绝、冷却期和摘要 |
| Host 信任确认 | QuickDesk 内有 MCP/tool 级 `TrustHandler`，可设 `auto_approve` | 不是远端控制会话的密码学准入闸门；不能抵御恶意服务器 |
| WebRTC 媒体 | P2P/TURN，媒体内容依赖 Chromium Remoting | 服务器可通过信令位置插入或拒绝会话；TURN 运维方还可观察中继流量元数据 |
| SPAKE2/access code | Host 和 Controller 用接入码参与 Remoting 认证 | 服务器知道接入码时，具备同时扮演两端的条件 |

代码位置：

- `SignalingServer/internal/models/device.go`
- `SignalingServer/internal/service/device_secret_service.go`
- `SignalingServer/internal/service/device_service.go`
- `SignalingServer/internal/service/token_service.go`
- `SignalingServer/internal/handler/host_handler.go`
- `SignalingServer/internal/handler/realtime_handler.go`
- `QuickDesk/src/api/TrustHandler.cpp`

## 2. 威胁模型

### 2.1 保护资产

- 被控端屏幕、输入、文件、剪贴板和麦克风/音频能力；
- `device_secret` 和 Host 身份；
- `access_code` 与临时授权码；
- Controller 身份、授权范围和有效期；
- 审计、审批和撤销记录。

### 2.2 攻击者能力

考虑以下攻击者：

1. 取得数据库只读备份；
2. 取得 Redis/服务器进程控制权；
3. 控制信令服务器全部代码和响应；
4. 读取管理端数据，但不能直接读被控端文件；
5. 能拒绝、延迟或重放信令消息；
6. 不能破解 TLS、Argon2id 或 QuickDesk/Chromium 的密码学原语。

被控端本机已有管理员权限、操作系统已被植入、Host 二进制被替换、物理接触和
用户主动把接入码交给攻击者，不属于本模型能够消除的风险。

### 2.3 安全声明

| 声明 | 当前是否成立 | 证据/说明 |
|---|---|---|
| 拿到数据库备份不能恢复 `device_secret` | 是 | 只有 Argon2id hash；设备密钥是 48 字节随机值的 hex |
| 拿到数据库备份不能控制设备 | 否 | `access_code` 是明文；攻击者仍可使用当前接入流程 |
| 恶意服务器必须取得用户批准才能控制设备 | 否 | 审批由服务器实现，服务器可伪造批准或直接改设备状态 |
| Host 能独立验证 Controller 的所有者签名 | 否 | 当前没有 Host 侧签名授权验证 |
| 服务器只能拒绝服务，不能伪造授权 | 否 | 当前 signal token、access code、approval 都依赖服务器 |
| 撤销后恶意服务器立即失去能力 | 否 | 实时恶意服务器可读取下一次接入码或改绑/改审批状态 |
| `device_secret` 泄露后可以单设备吊销 | 是 | 管理端 `secret:rotate` 会替换哈希，旧 secret 失效 |

## 3. 风险分级

| 场景 | 风险 | 原因 |
|---|---|---|
| 使用自己控制、只有自己可登录的自托管服务器 | 中 | 服务器仍保存接入码和会话能力，但运维者与设备 owner 是同一信任主体 |
| 使用第三方/公共信令服务器 | 高 | 第三方具备控制端能力，用户无法从协议上证明其没有使用 |
| 服务器数据库或备份泄露 | 高 | 当前接入码明文泄露，依赖攻击者是否及时使用 |
| Redis 或实时进程被攻陷 | 高 | 可伪造 token、审批和实时事件 |
| 只泄露 `device_secret_hash` | 低 | 高熵 secret 不可逆，不能直接得到明文 |
| Host 开启 MCP `auto_approve` | 风险放大 | 工具级确认不覆盖远端控制，且不提供密码学保证 |

因此上游 release 不能仅因为“device_secret 已哈希”就声称信令服务器被攻陷不会导致
远程控制。更准确的表述是：

> 数据库泄露不能恢复 device_secret；但当前信令服务器仍然是设备访问的信任方，
> 其管理员或攻陷者仍可能通过明文 access_code 和服务器签发流程控制设备。

## 4. 为什么服务器端修补不够

### 4.1 只把 `access_code` 哈希化

能降低静态数据库泄露风险，但服务器仍会收到 Host 上报的明文接入码，也会执行
`verify`。实时被攻陷的服务器仍可在输入时截获接入码。

### 4.2 只收紧管理员权限

恶意服务器或 host root 可以绕过应用权限，直接读数据库、Redis 或修改代码。权限模型
不能给用户提供独立于服务器运维者的安全保证。

### 4.3 只把审批逻辑放到“另一个服务”

如果另一个服务仍由同一信任方控制，或者最终 token 仍由它签发，攻击者只是增加了
跳板，Host 仍没有可验证的授权根。

### 4.4 只依赖 TLS/证书固定

证书固定能防止网络中间人，不能防止合法持有服务器私钥、进程或数据库的运营者。

## 5. 推荐目标架构

最终授权应由 Host 在本地独立验证，信令服务器只负责发现、排队和转发。

### 5.1 设备身份密钥

- Host 首次启动生成 Ed25519 设备身份密钥；
- `device_secret` 只用于向信令服务器证明“我是这台 Host”，不能作为设备所有权根；
- 设备公钥由 owner 绑定，服务器只保存公钥和签名声明；
- 设备身份私钥只存在 Host 侧，服务器永远不获取。

### 5.2 Controller 公钥与 Owner 授权

Controller 生成临时或持久公钥。Owner 给授权声明签名，至少包含：

```text
device_id
controller_public_key
permissions
issued_at
expires_at
nonce
```

服务器可以存这份声明并做索引，但不能伪造 Owner 签名。Host 在建立控制会话前：

1. 验证声明签名属于已绑定的 Owner 公钥；
2. 验证 device_id、权限、有效期和 nonce；
3. 要求 Controller 对会话 challenge 签名；
4. 只有本地验证通过才接受 WebRTC/Jingle 会话。

这样服务器可以拒绝或隐藏请求，但不能凭空生成一个 Host 会接受的控制权。

### 5.3 Access code 的新位置

- 默认不再把明文 `access_code` 持久化到服务器；
- Owner 需要的当前接入码通过 Host 到 Owner 的端到端通道展示，或仅保存在本地 Qt/Web
  的短期内存；
- 如果保留服务器验证码，服务器只保存 PAKE/OPAQUE verifier，不能将其用于冒充 Host；
- 服务器上的 verification endpoint 不能单独产生足以控制设备的授权。

### 5.4 Signal token 降级为路由凭证

`signal_token` 仍可用于 WS 接入、限流和防重放，但 Host 不应因为“服务器说 token 有效”
就相信 Controller 已获授权。token 只是路由层凭证，最终权限由 Host 的签名声明和
challenge 响应决定。

### 5.5 审计

- 记录 Owner 签名、Controller 公钥摘要、设备公钥摘要和权限；
- 记录 Host 本地接受/拒绝原因；
- 服务器侧的批准只能作为 UX 提示，不能替代 Host 验证；
- 支持 owner 本地撤销和服务器不可用时的本地 deny list。

## 6. 分阶段落地

| 阶段 | 改动 | 能解决的问题 | 不能解决的问题 |
|---|---|---|---|
| A. 运维加固 | 自托管、最小权限、TLS、禁用 `auto_approve`、定期轮换接入码 | 降低外部滥用和误配置 | 不能抵御服务器 owner/root |
| B. 数据最小化 | 不再向普通 API/管理端返回明文接入码；数据库只保存 verifier | 数据库泄露和后台误用 | 实时服务器仍可截获明文上传 |
| C. 签名授权 | Host 设备密钥、Controller 公钥、Owner 签名 grant、Host 本地验证 | 恶意服务器不能伪造控制权 | 仍可拒绝服务、观察元数据 |
| D. 可验证发布 | Host/Controller 二进制签名、版本透明日志、设备固件/安装包校验 | 降低替换客户端/升级链攻击 | 不能解决用户主动交出权限 |

建议先把 A+B 做成可发布的加固，再以 C 作为协议升级目标。不要在没有 C 的情况下把
“私有化部署”宣传成对服务器管理员也安全。

## 7. Completion Criteria for Issue #10

Issue #10 不能只以“有一份说明文档”关闭。最终至少需要：

- [ ] 新设备默认生成 Host 身份密钥，服务器不能读取私钥；
- [ ] Owner 授权声明可在离线或另一设备上验证；
- [ ] Host 在建立媒体/数据通道前验证 Owner 签名和 Controller challenge；
- [ ] 服务器伪造 token、批准记录或数据库行时，Host 仍拒绝控制；
- [ ] 普通 API、管理 API 和实时事件不再泄漏明文 access code；
- [ ] 撤销、过期、权限缩小和多设备隔离有自动化测试；
- [ ] 模拟恶意服务器的 E2E：服务器签发假 Controller、伪造 approval、重放 token，
      Host 必须拒绝；
- [ ] 旧设备/旧客户端有明确迁移和兼容策略。

在这些条件完成前，issue #10 应保持开放；当前文档只把边界和攻击面说清楚，不把风险
伪装成已修复。

## 8. Communication Guidance

可以回复：

> `device_secret` 本身只保存 Argon2id 哈希，服务端不能从数据库恢复它。但当前控制面
> 仍保存明文 `access_code`，并负责签发 signal token 和审批，因此被攻陷的信令服务器
> 仍是设备访问的信任方，不能仅靠“私有化部署”或“secret 已哈希”宣称不会 RAT。
> 建议把最终控制授权改为 Owner 签名、Host 本地验证；服务器降级为路由和可用性组件。
> 具体边界和分阶段方案见
> `docs/security/signaling-trust-model.md`。

不要在公开 issue 中粘贴可直接利用的服务器占位、真实 token、用户设备 ID、接入码或
生产日志；这些应通过私密渠道提交。
