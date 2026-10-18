# Host / Relay 权威录制 Remoting 需求

**状态**：2026-10-18，待 `quickdesk-remoting` 提供宿主侧录制接口。

**对应清单**：QD-004 / QD-504。

**边界**：本文不要求在 fork 里新增第二套屏幕采集或另造 RDP/媒体协议。

## 1. 要解决的问题

当前录像由控制端从已解码画面重新编码：控制端在线时可用，但控制端断线、崩溃或网络长时间
中断后，录像会缺失或提前结束。需要由被控端 Host 或真正承载媒体的 Relay 产生权威录像，
并且即使控制器掉线，服务端也能拿到完整的会话文件和索引。

## 2. 推荐方案

优先在 `quickdesk-remoting` 的视频编码出口录制实际发送的 encoded frame，而不是再启动
一个桌面采集器：

1. 控制面创建 `session_recordings` 索引和短时上传票据；
2. Qt 只负责把会话 ID、录制策略和票据配置发给 Host；
3. Host 在 Remoting 编码线程之后把同一组 encoded frame 写入有界缓存和分段容器；
4. 会话结束、对端断开、Host 退出或缓冲区达到上限时原子收尾；
5. Host 用一次性票据上传 artifact，服务端校验摘要后把记录转为 `completed`；
6. 失败路径写入 `failed` 和原因，断点续传沿用现有录像 artifact PATCH 协议。

如果 Host 录制无法覆盖纯 Relay 场景，再单独评估 TURN/RTP 录制；不要假设普通信令服务器
能看到 P2P 媒体。

## 3. Native Messaging 契约草案

Host 先通过 `recordingCapabilities` 声明支持的格式、最大时长、磁盘预算和是否需要短时
上传票据。建议新增：

```text
recordingCapabilities
startSessionRecording
stopSessionRecording
recordingStateChanged
recordingArtifactReady
recordingFailed
```

`startSessionRecording` 至少包含：

- `session_id` / `connection_id`；
- `recording_id` 和一次性上传票据；
- 容器与编码偏好；
- 最大时长、最大字节数和剩余磁盘下限；
- 是否允许自动覆盖旧缓存。

Host 不得持久化用户 Bearer token。票据只能访问指定 recording 的上传/完成接口，并应有
短有效期和一次性使用约束。

## 4. 可靠性与安全要求

- 录制文件使用临时目录和原子 rename；半写文件不能进入可播放状态；
- 控制器断开不停止 Host 录制，Host 只按会话终态、显式 stop、时长或空间上限收尾；
- 音视频分段必须可恢复，进程崩溃时最多丢失最后一个有界分片；
- 上传前计算 SHA-256，服务端校验后再 attach；
- 磁盘不足、权限不足、编码失败必须显式 `failed`，不能留下永久 `recording`；
- 不同会话、不同用户之间不能复用上传票据或读取彼此临时文件；
- 隐私屏、锁屏和 RDP/安全桌面规则沿用 Remoting 现有可见内容，不在录制层绕过。

## 5. 验收用例

1. 控制器主动断开后，Host 继续收尾并上传完整录像；
2. 控制器进程崩溃、网络断开和正常结束时都能生成一条终态记录；
3. Host 在录制中重启，服务端把旧记录标为 `failed` 或可恢复的 interrupted；
4. 4K/高帧率录制达到时长或空间上限时安全分段/停止；
5. 票据过期、会话错配、recording ID 错配全部被拒绝；
6. Windows Service/session 0、普通交互用户和 macOS Host 各跑一遍权限与落盘测试；
7. 服务端可以从索引找到 artifact，逐字节回读并通过 SHA-256 校验；
8. 录制不会额外触发一次屏幕采集权限弹窗或改变当前编码分辨率。

## 6. 为什么当前停在 fork 外

仓库里的 `quickdesk_host` 是预编译 Remoting 二进制，当前 native messaging 只暴露连接、
授权、客户端列表、隐私屏和服务控制等能力，没有开始/停止 Host 录制或返回 artifact 的
消息。Qt 只能读取控制端解码后的共享帧，无法访问 Host 编码器后的权威媒体流。因此正确
的落点是向 `barry-ran/QuickDesk` / `quickdesk-remoting` 提交该协议和实现，而不是在
fork 中复制一套屏幕采集链。
