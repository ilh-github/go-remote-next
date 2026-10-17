# QuickDesk 业务工作流 HTTP 触发器

> 状态（2026-10-17）：QD-610 第三阶段。业务系统可以保存 manual/interval trigger，
> 通过独立 Bearer token 立即运行，并由 HTTP transport 的后台 worker 每 5 秒领取一次
> 到期触发器。`Idempotency-Key` 已支持持久化重放；控制面事件订阅、出站回调、
> 委托权限细分、cron/日历和异步任务队列仍未完成。

## 1. 边界

QuickDesk 的核心媒体与输入链路仍然由 Chromium Remoting / WebRTC 负责。该触发器只
解决“业务系统怎样启动一条已经录好的桌面工作流”，不复制远程桌面协议，也不经过
SignalingServer 转发桌面媒体。

触发器运行在 `quickdesk-mcp` 的 HTTP transport 中：

```text
业务系统 / n8n / Zapier / 自研后台
        |
        | HTTP + Bearer trigger token
        v
quickdesk-mcp HTTP
        |
        | 已有 MCP workflow replay + trust layer
        v
QuickDesk WebSocket API -> 远端设备
```

stdio transport 不监听网络端口，因此不启用该接口。

## 2. 启动

```bash
quickdesk-mcp \
  --transport http \
  --ws-url ws://127.0.0.1:9600 \
  --host 127.0.0.1 \
  --port 8080 \
  --trigger-token "$QUICKDESK_TRIGGER_TOKEN"
```

也可以用环境变量：

```bash
QUICKDESK_TRANSPORT=http \
QUICKDESK_WS_URL=ws://127.0.0.1:9600 \
QUICKDESK_TRIGGER_TOKEN='a-separate-business-secret' \
quickdesk-mcp
```

`QUICKDESK_TRIGGER_TOKEN` 为空时两个业务入口都返回 404。它应与
`QUICKDESK_TOKEN`、`QUICKDESK_READONLY_TOKEN` 分开配置，业务系统不应持有 QuickDesk
的完整控制 token。

## 3. 触发工作流

```http
POST /v1/workflows/{workflow_id}/run
Authorization: Bearer <QUICKDESK_TRIGGER_TOKEN>
Idempotency-Key: <unique-request-id>
Content-Type: application/json
```

```json
{
  "device_id": "deviceId-from-list_connections",
  "overrides": {
    "target": "C:\\reports\\daily.txt"
  },
  "stop_on_failure": true
}
```

字段：

| 字段 | 必填 | 说明 |
|---|---|---|
| `device_id` | 是 | 要执行工作流的已连接设备；仍受 `QUICKDESK_ALLOWED_DEVICES` 限制 |
| `overrides` | 否 | 传给 `replay_workflow` 的参数覆盖 |
| `stop_on_failure` | 否 | 默认 `true`；安全失败不受该字段影响，永远终止 |

成功时返回 `200` 和完整的 `WorkflowRunReport`，报告同时写入
`~/.quickdesk/workflow-runs/{run_id}.json`。

`Idempotency-Key` 可选，长度为 1 到 128 个字符。相同 key、工作流、设备和请求参数再次
到达时，直接返回第一次持久化的 `WorkflowRunReport`，不会再次执行工作流；同一 key 被
不同请求复用时返回 `409`，第一次请求仍在运行也返回 `409`。映射写入
`~/.quickdesk/workflow-idempotency.json`，进程重启后仍可重放。第一次执行失败时，
后续相同 key 返回相同失败状态，不会把失败伪装成成功。

错误：

| HTTP | 含义 |
|---|---|
| `400` | `device_id` 为空或其他请求体错误 |
| `401` | Bearer token 缺失或错误 |
| `403` | 设备不在 `QUICKDESK_ALLOWED_DEVICES` |
| `404` | 触发器未启用、工作流不存在 |
| `409` | `Idempotency-Key` 正在执行或被不同请求复用 |
| `429` | 已有另一条业务工作流正在运行 |
| `500` | QuickDesk 执行失败或报告持久化失败 |

当前每个 `quickdesk-mcp` 进程只允许一个业务触发工作流并发运行。远程桌面工作流通常
依赖当前窗口、输入焦点和屏幕状态，并发执行会让结果不可预测。

## 4. 查询运行报告

```http
GET /v1/workflow-runs/{run_id}
Authorization: Bearer <QUICKDESK_TRIGGER_TOKEN>
```

返回持久化的 `WorkflowRunReport`；不存在时返回 404。同步触发已经返回同一份报告，
该接口用于业务系统保存 `run_id` 后补查或人工复盘。

## 5. 持久化触发器

触发器定义写入 `~/.quickdesk/workflow-triggers/{trigger_id}.json`，保存：

- 名称、工作流 ID、设备 ID；
- 默认参数覆盖和 `stop_on_failure`；
- 可选 `interval_seconds`；
- `next_run_at`、上次运行 ID/状态/错误。

管理接口：

```http
GET    /v1/triggers
POST   /v1/triggers
DELETE /v1/triggers/{trigger_id}
POST   /v1/triggers/{trigger_id}/run
```

创建 interval trigger：

```json
{
  "name": "每小时生成报表",
  "workflow_id": "b7ca0a2f-0d4c-4f0c-9dbf-4a92e8a67d10",
  "device_id": "device-1",
  "overrides": {
    "target": "C:\\reports\\hourly.txt"
  },
  "stop_on_failure": true,
  "interval_seconds": 3600
}
```

`interval_seconds` 范围是 60 秒到 365 天；省略时创建 manual trigger，只通过
`POST /v1/triggers/{trigger_id}/run` 执行。到期领取会先推进 `next_run_at` 再执行，
采用 at-most-once 语义；worker 一次只领取一条，并与手动触发共用同一个执行槽。

`POST /v1/triggers/{trigger_id}/run` 返回本次 `WorkflowRunReport`，同时把 run_id、
completed/failed 和错误写回 trigger 定义。

## 6. 安全

- 默认绑定 `127.0.0.1`。需要跨机器调用时，应放在 TLS 反向代理后面，不能直接暴露
  MCP HTTP 端口。
- trigger token 使用常量时间比较，且与 QuickDesk WS token 分离。
- 设备白名单在进入工作流前检查。
-  replay 仍经过 trust policy、人工确认和 emergency stop。
- 工作流 ID 和 run ID 都经过 UUID 与路径逃逸校验。
- `Idempotency-Key` 只做请求去重，不是认证凭证；仍需保护 trigger token。
- 同步请求会占用一个执行槽；业务系统应设置足够超时并处理 429。

## 7. 尚未完成

- 控制面事件订阅和 event-to-workflow 映射；
- cron/日历定时规则；当前只有固定 interval；
- 异步队列、取消和跨进程 worker；当前只支持单进程执行槽；
- 运行完成后的 HMAC 出站回调；
- Owner 委托 grant、最小权限范围和审计回传控制面。

因此 QD-610 仍保持部分完成，不能把本接口描述成完整的业务流程引擎。
