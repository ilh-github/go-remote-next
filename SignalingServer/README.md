# QuickDesk Signaling Server

QuickDesk 信令服务器，使用 Go + Gin + GORM 开发，提供设备 provisioning、用户认证、设备绑定、WebRTC 信令转发与实时事件推送。

> 完整 API 契约见 [`docs/user-api-docs.md`](docs/user-api-docs.md)。
> 设计原理与决策记录见 [`../docs/dev/信令服务器API重构方案.md`](../docs/dev/信令服务器API重构方案.md)。

## 功能

- 设备 provisioning（首次启动自动分配 9 位 device_id + device_secret）
- 用户注册/登录（密码 + SMS）；access/refresh token + family rotation
- 我的设备 / 收藏 / 连接历史 CRUD
- WebRTC 信令中继（host ↔ client，jingle 信封）
- 实时事件推送（snapshot + 增量；Redis stream resume）
- 管理员后台（CRUD + 2FA + 审计 + webhook）
- Redis 双信号 presence（hb + wsconn 派生 online；keyspace notifications）
- PostgreSQL 持久化

## 依赖

- Go 1.21+
- PostgreSQL 15+
- Redis 7+（**必须**启用 `notify-keyspace-events Ex`）

### 轻量单进程模式

只运行一个信令进程、使用单个 SQLite 文件时，可以不安装 PostgreSQL 和 Redis：

```bash
DB_DRIVER=sqlite \
DB_PATH=./data/quickdesk.db \
REDIS_DRIVER=embedded \
SERVER_HOST=127.0.0.1 \
SERVER_PORT=8000 \
go run cmd/signaling/main.go
```

`DB_DRIVER=sqlite` 时如果 `REDIS_DRIVER` 留空，会自动选择 `embedded`。这个模式适合
单机自托管、开发和家庭网络，并有以下边界：

- 只能运行一个信令进程，不能做多实例共享；
- SQLite 保存账号、设备、设置、会话索引等业务数据，重启后仍然存在；
- 进程内 Redis 只存在于当前进程，重启会清空 refresh token、presence、限流和实时流
  续传状态，用户需要重新登录；
- 通知投递 worker 保留运行，消费同一 SQLite 中的持久 outbox；计划任务和跨实例日志等
  后台能力在轻量模式中关闭；
- STUN/TURN 仍是独立基础设施；跨 NAT 时继续配置 `STUN_URLS` / `TURN_URLS`。

数据库文件默认位于 `./data/quickdesk.db`，备份时应停止服务或使用 SQLite 的一致性
备份方式。完整 E2E 可运行 `bash ../scripts/e2e/test-quickdesk-minimal-server.sh`。

### 一致性备份与恢复

`quickdesk-backup` 对 SQLite 使用 `VACUUM INTO` 生成一致性快照，并把数据库、
`RECORDING_DIR` 和 `UPDATE_ARTIFACT_DIR` 打进一个带 SHA-256 manifest 的归档：

```bash
go run ./cmd/quickdesk-backup backup \
  -db data/quickdesk.db \
  -recordings data/recordings \
  -updates data/updates \
  -output quickdesk-$(date +%Y%m%dT%H%M%S).tar.gz
```

恢复应在信令服务停止时执行。目标已存在时默认拒绝；确认要替换后使用 `-force`，
旧目录或旧数据库会改名为 `*.pre-restore-<UTC 时间>`，不会被直接删除。数据库或任一
媒体目录安装失败时，已经替换的目标会回滚到原路径：

```bash
go run ./cmd/quickdesk-backup restore \
  -archive quickdesk-20261017T130000.tar.gz \
  -db data/quickdesk.db \
  -recordings data/recordings \
  -updates data/updates \
  -force
```

Docker 部署中的等价命令是
`docker exec <container> /opt/quickdesk/quickdesk-backup ...`；归档输出路径应位于挂载的
`/data` 卷内，或先用 `docker cp` 取出。

归档恢复会验证 manifest 版本、文件集合和每个文件的 SHA-256，并拒绝绝对路径、`..`
路径穿越、符号链接和重复文件。PostgreSQL 部署继续使用 `pg_dump` / `pg_restore`；
媒体目录使用同一份归档或独立备份盘保存。

### 存储容量配额

`RECORDING_STORAGE_QUOTA_GB` 和 `UPDATE_STORAGE_QUOTA_GB` 分别限制录像与自托管安装包
目录的保留总量，默认 `0` 表示不限制。配额统计真实的普通文件，包括可恢复上传的
`.part` 暂存文件，因此客户端不能用未完成上传绕过上限。配额在进程内、上传写入前和
写入过程中执行，多实例共享目录仍应使用文件系统配额或对象存储桶配额。

管理员可通过以下接口查看实际磁盘字节、索引字节、配额、剩余量和超限状态：

- `GET /v1/admin/recordings:storage`
- `GET /v1/admin/update-releases:storage`

## 快速开始

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 启动数据库

使用 Docker：

```bash
# PostgreSQL
docker run -d -p 5432:5432 \
  -e POSTGRES_USER=quickdesk \
  -e POSTGRES_PASSWORD=quickdesk123 \
  -e POSTGRES_DB=quickdesk \
  postgres:15

# Redis（必须开启 keyspace notifications，host 心跳 TTL 过期靠它派发 device.online.changed 事件）
docker run -d -p 6379:6379 redis:7 redis-server --appendonly yes --notify-keyspace-events Ex
```

### 3. 配置环境变量

复制 `.env.example` 到 `.env` 并修改配置：

```bash
cp .env.example .env
```

### 4. 运行服务器

```bash
go run cmd/signaling/main.go
```

服务器将在 `http://localhost:8000` 启动。

## API 端点

服务器暴露的全部路由前缀都是 `/v1/*`（旧 `/api/v1/*`、`/signal/*`、`/host/*`、`/client/*` 在 v1 重构后已下线）。

主要分组：

| 分组 | 路径前缀 | 说明 |
|---|---|---|
| 公开 | `/health`、`/v1/preset`、`/v1/settings/public`、`/v1/features`、`/v1/verification-codes` | 无 user token，但需 `X-API-Key`（若服务端配置） |
| 认证 | `/v1/auth/*` | 注册、登录、SMS 登录、token 刷新、密码重置 |
| 当前用户 | `/v1/me/*` | profile、设备、连接、收藏、session 管理（Bearer access_token） |
| 设备侧 | `/v1/devices*`、`/v1/ice-config` | provision / heartbeat / signal-tokens / access-code:verify |
| 管理后台 | `/v1/admin/*` | 完整 CRUD + 2FA + 审计 + webhook |
| 实时 WebSocket | `/v1/realtime/events`、`/v1/realtime/signal` | 首帧 auth 握手；浏览器无 X-API-Key 走 Origin 白名单 |

完整路由 / 请求体 / 响应体 / 错误码契约请见 [`docs/user-api-docs.md`](docs/user-api-docs.md)。

## 项目结构

```
server/
├── cmd/
│   └── signaling/
│       └── main.go              # 程序入口
├── internal/
│   ├── config/                  # 配置管理
│   ├── models/                  # 数据模型
│   ├── repository/              # 数据访问层
│   ├── service/                 # 业务逻辑层
│   ├── handler/                 # HTTP 和 WebSocket 处理器
│   └── middleware/              # 中间件
├── migrations/                  # 数据库迁移文件
├── go.mod
├── go.sum
├── .env.example
└── README.md
```

## 开发

```bash
# 运行测试
go test ./...

# 构建
go build -o signaling cmd/signaling/main.go

# 运行
./signaling
```

## License

Copyright 2026 QuickDesk
