# QuickDesk 业务工作流 HTTP 触发器

> 状态（2026-10-18）：QD-610 第十四阶段。业务系统可以保存 interval、IANA 时区 cron 或控制面事件
> trigger，通过独立 Bearer token 立即运行，并由 HTTP transport 的后台 worker 领取到期
> 触发器。`Idempotency-Key`、异步 job、HMAC 出站回调、持久回调 outbox 和受控控制面
> 事件订阅及本机短期重连补发已支持；法定节假日、QuickDesk 重启后的持久事件补发、
> 委托权限细分和跨进程 worker 仍未完成。

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
QUICKDESK_CALLBACK_SECRET='a-callback-signing-secret' \
quickdesk-mcp
```

`QUICKDESK_TRIGGER_TOKEN` 为空时两个业务入口都返回 404。它应与
`QUICKDESK_TOKEN`、`QUICKDESK_READONLY_TOKEN` 分开配置，业务系统不应持有 QuickDesk
的完整控制 token。`QUICKDESK_CALLBACK_SECRET` 只在配置 `callback_url` 时需要，
用于签名出站回调，不能与 trigger token 复用。

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
  "stop_on_failure": true,
  "callback_url": "http://127.0.0.1:8090/quickdesk/result"
}
```

字段：

| 字段 | 必填 | 说明 |
|---|---|---|
| `device_id` | 是 | 要执行工作流的已连接设备；仍受 `QUICKDESK_ALLOWED_DEVICES` 限制 |
| `overrides` | 否 | 传给 `replay_workflow` 的参数覆盖 |
| `stop_on_failure` | 否 | 默认 `true`；安全失败不受该字段影响，永远终止 |
| `callback_url` | 否 | 运行结束后接收签名回调的 HTTP URL；使用它时必须配置 `QUICKDESK_CALLBACK_SECRET` |

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

## 4. 异步运行与取消

异步入口不会等待工作流结束：

```http
POST /v1/workflows/{workflow_id}/run-async
Authorization: Bearer <QUICKDESK_TRIGGER_TOKEN>
Content-Type: application/json
```

请求体与同步入口相同。成功返回 `202 Accepted` 和一个 `workflow job`：

```json
{
  "run_id": "c7ca0a2f-0d4c-4f0c-9dbf-4a92e8a67d10",
  "workflow_id": "b7ca0a2f-0d4c-4f0c-9dbf-4a92e8a67d10",
  "device_id": "device-1",
  "status": "queued",
  "created_at": "2026-10-17T15:20:00Z",
  "started_at": null,
  "completed_at": null,
  "report": null,
  "error": ""
}
```

查询状态：

```http
GET /v1/workflow-runs/{run_id}/status
Authorization: Bearer <QUICKDESK_TRIGGER_TOKEN>
```

状态为 `queued`、`running`、`cancelling`、`completed`、`failed` 或 `cancelled`。
`completed` 和 `failed` 会带完整的 `report`；执行错误写入 `error`。

取消：

```http
POST /v1/workflow-runs/{run_id}/cancel
Authorization: Bearer <QUICKDESK_TRIGGER_TOKEN>
```

排队中的任务会立即标为 `cancelled`；运行中的任务先变成 `cancelling`，worker 在步骤
边界或当前 WebSocket 请求被取消后写入带 `cancelled=true` 的报告。已完成、已失败或
已取消的任务返回 `409`。当前队列容量为 32，worker 与同步触发、定时触发共用同一个
单执行槽。

执行参数保存在 `~/.quickdesk/workflow-jobs/{run_id}.request.json`，状态和报告仍保存在
`{run_id}.json`。进程重启后：

- `queued` job 会重新进入进程内队列并继续执行；
- `running` job 不会自动重放，因为退出前是否已经产生副作用无法证明；状态转为 `failed`，
  错误明确写为 process restart 中断；
- `cancelling` job 转为 `cancelled`，不再执行后续步骤。

恢复仍只支持同一数据目录上的单进程；多个 MCP 进程同时消费同一目录没有分布式锁。

## 5. HMAC 回调

同步、异步和持久化 trigger 都可在运行结束后向 `callback_url` POST JSON。回调头：

```text
X-QuickDesk-Event: workflow.run.completed | workflow.run.failed
X-QuickDesk-Delivery-Id: <uuid>
X-QuickDesk-Timestamp: <unix seconds>
X-QuickDesk-Signature: hmac-sha1=<hex>
```

签名原文是 `timestamp + "." + raw_json_body`，算法为 HMAC-SHA1，密钥来自
`QUICKDESK_CALLBACK_SECRET`。接收端应先校验时间窗口和签名，再处理事件；成功事件包含
完整 `report`，失败事件包含 `error`。当前只允许 HTTP URL，跨机器部署必须把回调入口
放在 TLS 反向代理后。

回调先持久化到 `~/.quickdesk/callback-outbox/{delivery_id}.json`，再由后台 worker
异步投递。同一个 `X-QuickDesk-Delivery-Id` 在多次重试间保持不变，接收端可以用它做
幂等。单次请求 10 秒超时；失败后按 5 秒起步指数退避，最长 1 小时，最多 8 次，随后
进入 `dead`。进程重启会重新扫描 `pending` / `retry` 记录继续投递；投递成功后状态转为
`delivered`。异步 job JSON 同时记录 `callback_delivery_id`、`callback_status` 和
`callback_error`，便于把 job 与 outbox 记录对应起来。

### 5.1 死信查询与人工重放

以下接口复用 `QUICKDESK_TRIGGER_TOKEN`：

```http
GET  /v1/callback-deliveries?status=dead&limit=100
POST /v1/callback-deliveries/{delivery_id}/retry
POST /v1/callback-deliveries/{delivery_id}/ack
POST /v1/callback-deliveries/ack
Authorization: Bearer <QUICKDESK_TRIGGER_TOKEN>
```

列表返回 `{"deliveries": [...]}`，可按 `pending`、`retry`、`delivered`、`dead` 过滤；
`limit` 默认 100，范围 1~500。人工重放只接受 `dead` 记录，会清空失败状态和尝试次数，
把记录改回 `pending` 并立即唤醒 worker；原 `X-QuickDesk-Delivery-Id` 不变，因此接收端
仍可按同一键幂等。不存在返回 404，非 `dead` 记录返回 409。

`ack` 只接受 `delivered` 或 `dead` 记录，首次调用写入 `acknowledged_at`，重复调用幂等
返回同一时间。它表示“有人处理过这条投递”，不改变 `status`，也不替代接收端对 HTTP 2xx
的处理。pending/retry 记录返回 409。

批量确认：

```json
{
  "delivery_ids": [
    "f13f5f2c-7f97-4b1d-99d9-6ff6fef2ad08",
    "8a5b2eb6-c418-4f7c-931a-ce66cf2e96da"
  ]
}
```

`POST /v1/callback-deliveries/ack` 接受去重后的 1~200 个 id，返回
`acknowledged`、`skipped` 和 `not_found` 三组结果。已 delivered/dead 的记录幂等确认，
pending/retry 放入 `skipped`，不存在或非法 UUID 放入 `not_found`；空数组返回 `400`。
批量确认不会因为个别非终态记录而整体失败。

## 6. 查询运行报告

```http
GET /v1/workflow-runs/{run_id}
Authorization: Bearer <QUICKDESK_TRIGGER_TOKEN>
```

返回持久化的 `WorkflowRunReport`；不存在时返回 404。同步触发已经返回同一份报告，
该接口用于业务系统保存 `run_id` 后补查或人工复盘。

## 7. 持久化触发器

触发器定义写入 `~/.quickdesk/workflow-triggers/{trigger_id}.json`，保存：

- 名称、工作流 ID、设备 ID；
- 默认参数覆盖和 `stop_on_failure`；
- 可选 `interval_seconds` 或 `cron_expression`，两者互斥；
- cron 可选 IANA `timezone`，省略时默认 `UTC`；
- 可选 `event_type`、`event_device_id` 和 `event_filter`；
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
  "interval_seconds": 3600,
  "callback_url": "http://127.0.0.1:8090/quickdesk/result"
}
```

`interval_seconds` 范围是 60 秒到 365 天。`interval_seconds`、`cron_expression` 和
`event_type` 至少要提供一个；时间规则可以和事件规则同时使用，但 `interval_seconds`
与 `cron_expression` 互斥，三者都省略时返回 `400`。只创建时间触发器后仍可随时通过
`POST /v1/triggers/{trigger_id}/run` 手动执行。到期领取会先推进 `next_run_at` 再执行，
采用 at-most-once 语义；worker 一次只领取一条，并与手动触发共用同一个执行槽。

cron trigger：

```json
{
  "name": "工作日早上九点生成报表",
  "workflow_id": "b7ca0a2f-0d4c-4f0c-9dbf-4a92e8a67d10",
  "device_id": "device-1",
  "cron_expression": "0 0 9 * * Mon-Fri",
  "timezone": "Asia/Shanghai"
}
```

`cron_expression` 使用标准六段或七段格式：

```text
秒 分 时 日 月 周 [年]
```

也接受 `@hourly`、`@daily`、`@weekly` 等简写。`timezone` 接受 IANA 名称，例如
`Asia/Shanghai`、`America/New_York`；省略时为 `UTC`。`timezone` 只能和
`cron_expression` 一起使用，非法名称返回 `400`。

IANA 时区遵循真实 DST 偏移：春季不存在的本地时间会跳到下一有效槽位，秋季重复的本地
时间只执行第一次，不会因回拨执行两次。当前不提供法定节假日和复杂工作日日历。进程停机
期间错过的 cron 槽位不会补跑；恢复后从当前时刻计算下一次未来槽位，继续保证
at-most-once，而不是一次性回放多个历史槽位。

`POST /v1/triggers/{trigger_id}/run` 返回本次 `WorkflowRunReport`，同时把 run_id、
completed/failed 和错误写回 trigger 定义。

### 7.1 控制面事件触发器

事件触发器把已经登录的 QuickDesk 控制面事件流映射到本机工作流：

```text
SignalingServer user event stream
        |
        v
Qt CloudDeviceManager (snapshot / ordered replay / server_rev checks)
        |
        v
本机 WebSocket API: controlPlaneEvent
        |
        v
quickdesk-mcp event worker -> workflow replay
```

创建示例：

```json
{
  "name": "访问被拒绝时收集诊断",
  "workflow_id": "b7ca0a2f-0d4c-4f0c-9dbf-4a92e8a67d10",
  "device_id": "device-1",
  "event_type": "device.access.denied",
  "event_device_id": "device-1",
  "event_filter": {
    "reason": "policy"
  },
  "overrides": {
    "target": "C:\\diagnostics\\access-denied.txt"
  },
  "stop_on_failure": true
}
```

字段：

| 字段 | 必填 | 说明 |
|---|---|---|
| `event_type` | 是 | 必须来自下方白名单 |
| `event_device_id` | 否 | 产生事件的设备；设备类事件省略时默认等于执行目标 `device_id`，避免跨设备误触发。`scheduled_task.failed` 等用户级事件默认不按设备过滤 |
| `event_filter` | 否 | 对事件 `data` 做顶层精确匹配的 JSON 对象；不能匹配 `access_code`、password、secret 或 token 字段 |

当前允许的事件类型：

- `device.online.changed`
- `device.offline.sustained`
- `device.metric.alert`
- `device.access.granted`
- `device.access.denied`
- `device.access_approval.requested`
- `device.access_approval.decided`
- `remote_session.started`
- `remote_session.ended`
- `recording.created`
- `recording.updated`
- `recording.deleted`
- `scheduled_task.failed`

Qt 和 MCP 各保留一份白名单，并使用同一套双层过滤：

1. 先按 `event_type`、`event_device_id` 和 `event_filter` 匹配；
2. 再持久化 claim，成功 claim 后才执行工作流。

SignalingServer 的实时信封不保证有顶层 `device_id`。Qt 事件桥和 MCP worker 都会从
`data.device_id` 归一化设备 ID；用户级 `scheduled_task.failed` 没有设备 ID，因此使用
`event_filter`（例如按 `kind`）限定范围。

事件 ID 是跨重启的幂等键：同一个 `id` 不会执行第二次。没有 `id` 的旧事件才使用
`server_rev` 去重。SQLite 轻量模式的进程内 Redis 重启后会让 `server_rev` 重新计数，
因此不能把 revision 单独作为长期唯一键，否则重启后触发器会永久停在旧 revision。
claim 在执行前写入；如果工作流随后失败，也不会回滚 claim，避免重放网络或桌面操作。
Qt 的本机 WebSocket API 在内存中保留最近 256 条白名单 `controlPlaneEvent`，并按事件
`id` 或 `server_rev` 去重。`quickdesk-mcp` 重连时会在 auth 参数中发送最后见到的
`since_server_rev`；认证成功后，API 会补发 revision 更大的事件。API token 为空时，
新连接直接接收当前缓冲。trigger 仍以事件 `id` 做幂等 claim，所以重复补发不会重复执行。

该补发窗口覆盖 `quickdesk-mcp` 短暂断开、重启，或 MCP 尚未启动但 QuickDesk 仍在运行的
场景。它不是持久事件库：缓冲只保存在 QuickDesk 进程内存中，最多 256 条；QuickDesk
自身重启、服务端 replay retention 不足或事件量超过窗口时仍可能丢失。

## 8. 安全

- 默认绑定 `127.0.0.1`。需要跨机器调用时，应放在 TLS 反向代理后面，不能直接暴露
  MCP HTTP 端口。
- trigger token 使用常量时间比较，且与 QuickDesk WS token 分离。
- 设备白名单在进入工作流前检查。
-  replay 仍经过 trust policy、人工确认和 emergency stop。
- 工作流 ID 和 run ID 都经过 UUID 与路径逃逸校验。
- `Idempotency-Key` 只做请求去重，不是认证凭证；仍需保护 trigger token。
- 事件桥只广播白名单类型，并只保留 `id`、`type`、`device_id`、`data`、`ts` 和
  `server_rev` 这些信封字段；不会把访问码或设备密钥转给 MCP。
- 同步请求会占用一个执行槽；业务系统应设置足够超时并处理 429。
- 异步任务状态写入 `~/.quickdesk/workflow-jobs/{run_id}.json`，执行参数独立写入
  `{run_id}.request.json`；重启会恢复 queued、失败关闭 running、收口 cancelling。
- 回调只允许 HTTP，不允许直接在公网暴露；签名密钥必须由接收端独立保管。
- 回调 outbox 会在重启后继续投递 `pending` / `retry` 记录，但 `dead` 记录当前只能
  从本地 JSON 查看，或通过受 Bearer token 保护的列表、人工重放和 ack 接口处理。

## 9. 尚未完成

- 法定节假日和复杂工作日日历；IANA 时区与 DST 已支持，但没有节假日跳过规则；
- 多进程共享 worker、分布式锁和公平调度；当前只支持单进程队列和单进程重启恢复；
- QuickDesk 进程重启后的持久事件补发、跨实例事件 outbox 和超过 256 条的积压；
  当前只有 QuickDesk 进程内的短期 replay buffer；
- 回调确认操作者归属；批量确认已支持，但 trigger token 无法证明具体操作者身份；
- Owner 委托 grant、最小权限范围和审计回传控制面。

因此 QD-610 仍保持部分完成，不能把本接口描述成完整的业务流程引擎。
