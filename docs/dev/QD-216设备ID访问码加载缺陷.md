# QD-216：设备 ID 与访问码加载不出来（issue #9）

**状态**：2026-10-25，根因已定位，修复已落地；纯逻辑层本机编译并双向验证，Qt 侧接线待 CMake/Qt6 构建。

**对应清单**：QD-216；上游 issue #9「主机一直连接不上」，正文只有一句
「设备ID和访问码加载不出来」。

**本节只记录可从当前代码证明的事实。**

## 1. 数据是怎么到达 UI 的

`HostManager` 用两个 QML 属性把值暴露给界面（`QuickDesk/src/manager/HostManager.h:37-38`）：

```cpp
Q_PROPERTY(QString deviceId READ deviceId NOTIFY deviceIdChanged)
Q_PROPERTY(QString accessCode READ accessCode NOTIFY accessCodeChanged)
```

两者初始为空（`HostManager.cpp:42-43` 在构造路径上 `clear()`），
并且只在收到 Host 的 native message 时被赋值（`HostManager.cpp:333-346`）：
从消息里取 `device_id`/`deviceId` 与 `access_code`/`accessCode`，非空且变化才写入并发信号。

最终来源是 `MainController::onHostReady(deviceId, accessCode)`
（`MainController.cpp:1100-1105`），它同时把 `m_hostReady` 置真。

**也就是说：UI 上的设备 ID 和访问码，完全依赖 Host 回传一次。**

## 2. 触发回传的那条消息，可能根本发不出去

启动时的自动连接是两段固定延迟（`MainController.cpp:901-935`）：

```cpp
QTimer::singleShot(500, this, [this]() {
    m_hostManager->sendHello();
    QTimer::singleShot(500, this, [this]() {
        m_hostServerStatus = ServerStatus::Connecting;
        ...
        m_hostManager->connectToServer(serverUrl, savedAccessCode);
    });
});
```

而 `connectToServer` 的第一件事是就绪检查（`HostManager.cpp:68-71`）：

```cpp
if (!m_messaging || !m_messaging->isReady()) {
    emit errorOccurred("NOT_READY", "Host process is not ready");
    return;
}
```

`NativeMessaging::isReady()`（`NativeMessaging.cpp:71-79`）只判断
**进程在跑、两个 socket 已连接**，不判断握手是否完成、能否处理 `connect`。

于是：Host 进程启动慢于 1000ms（Windows 上磁盘、杀软扫描、服务启动都容易造成），
`connect` 就在 Host 还没准备好时被丢弃。此后：

- 没有任何重试——`HostManager` 里的 `signalingRetryCount` 是 Host 侧信令重试，
  与这条 `connect` 消息能否送达无关；
- Host 也不会主动补发，协议里没有这个方向；
- `onHostReady` 因此永不到达，`deviceId`/`accessCode` 永远是空字符串。

## 3. 失败还被静音了

`HostManager::errorOccurred` 在 `MainController` 里**没有连接**
（全文只有 `ClientManager::errorOccurred` 被转发，`MainController.cpp:589`）。
所以上面那个 `NOT_READY` 发出来没有人接，用户既看不到重试，也看不到原因——
界面就是两个空白字段，与 issue #9 描述完全一致。

## 4. 已有的就绪信号没有被用上

`HostManager` 已经有正确的信号，只是自动连接没用：

- `helloResponseReceived(QString version)`（`HostManager.cpp:370` emit，`:125` 声明）——
  Host 对 `hello` 的回应，才是"可以发 connect 了"的真实时刻；
- `helloResponse` 消息本身**可能直接携带** `device_id` / `device_secret` /
  `access_code`（`HostManager.cpp:328` 注释明确写了这一点）。

## 5. 建议修复

按代价从低到高，前两条建议一起做：

1. **用就绪信号代替固定延迟**：把 `connectToServer` 从 `singleShot(500)` 里挪到
   `helloResponseReceived` 之后发起；`hello` 本身保留一个短超时重发，
   因为 `hello` 也可能撞上同样的未就绪窗口。
2. **有限重试 + 可见失败**：`connectToServer` 在 `NOT_READY` 时不要只 emit 就返回，
   按有界退避重试若干次；仍失败则把状态推到 `Failed` 并把错误接到 UI，
   让"加载不出来"至少有一个原因可读。
3. **接住 `errorOccurred`**：`MainController` 连接 `HostManager::errorOccurred`，
   至少记日志并反映到连接状态，不要静默丢弃。

## 6. 验收用例

1. Host 进程延迟 3 秒才就绪，设备 ID 与访问码仍能加载并显示；
2. Host 始终不就绪时，界面显示可读的失败原因，而不是两个空白字段；
3. `hello` 首次因未就绪丢弃后，重发最终成功；
4. `helloResponse` 直接携带 `device_id`/`access_code` 时，UI 立即得到值；
5. 正常启动路径不受影响，现有 HostManager 相关测试保持通过；
6. Windows 与 macOS 各验证一遍（issue #9 的报告环境是 Windows）。

## 7. 修复已如何落地

门控与退避的**决策**被抽成不依赖 Qt 的纯逻辑 `QuickDesk/src/core/hostreadiness.{h,cpp}`
（`HostConnectPolicy`：给定「hello 是否见过、第几次尝试、上次结果」，回答「等/发/重试/放弃/完成」）。
抽得出来是因为它本来就不依赖 Qt——它是「根据已观测到的状态，现在该做什么」。
模式与同目录的 `updatesupport` / `certificatepin` 一致：纯 cpp + 独立测试 target，
不链 Qt。

`QuickDesk/src/controller/MainController.cpp` 的启动路径改为消费该策略：

- `connect` 从固定 `singleShot(500)` 里挪到 `scheduleHostConnectAttempt()`，
  由策略决定下一刻做什么；
- `HostManager::helloResponseReceived` 接进 MainController（此前从未被连接），
  置 `m_hostHelloSeen` 并从 attempt 0 重启序列——Host 答得晚不应消耗重试预算；
- `HostManager::errorOccurred` 也接进来（此前同样从未被连接），
  收到 `NOT_READY` 时把结果记为 `NotReady` 并按预算推进下一次；
- 预算用尽时把状态推到 `Failed` 并打 WARN，而不是静默留在两个空白字段；
- `onHostReady` 记 `Accepted`，Host 进程重启时重置 `m_hostHelloSeen` 与尝试计数，
  不让上一个进程的握手替下一个进程作保。

### 验证边界（如实说明）

- **纯逻辑层已在本机验证**：`g++ -std=c++17 -Wall -Wextra` 编译零警告，
  测试通过；并做了四项双向验证——`!hello_seen` 改成放行后
  「before the Host answers hello...」失败、`NotReady` 改成直接放弃后
  「a not-ready refusal must schedule another attempt」失败、
  去掉 hello 去重后「a second hello is a no-op」失败、
  去掉 acceptance 保护后「a refusal after acceptance is ignored」失败；恢复后全绿。
- **序号簿记也收进了纯逻辑**：`MainController` 原先手写三个字段
  （`m_hostHelloSeen` / `m_hostConnectAttempt` / `m_lastHostConnectOutcome`），
  五处各自更新，正是容易写错的地方。现改为单个
  `HostConnectPolicy::ConnectSequence`，把去重、+1 推进、接受即终态、
  重启清零四条规则变成可测的；`MainController` 退化成薄翻译层。
  上限用 `constexpr` + `static_assert` 与策略绑定，防止两处数字漂移。
- **Qt 侧接线已编译并链接通过**（2026-10-25 更正）。此前这里写的是「未编译，
  需在有工具链的一侧跑」——那是在本机没有 Qt6/CMake 的前提下能确认的极限，
  但结论留了个错误印象，好像这件事还悬着。CI 的 `Unit tests` job 现在会构建
  整个 Qt 工程：`MainController.cpp` 与 `hostreadiness.cpp` 都在其中编译并链接
  成功（run `40133964841`，head `cbfcbe9`）。
- **第 6 节验收用例的覆盖情况**（同一 run，18/18 通过）：
  - 第 1、3、4 条已有自动化证据：`QuickDesk/tests/fake_host.cpp` 是按参数延迟
    回答 `hello` 的假 Host，走真实 native-messaging 二进制帧；
    `hoststartup_integration_test.cpp` 用真实 `HostManager` 驱动它，
    `quickdesk_host_startup_integration_test` 已通过（1.74 秒）。
  - 第 2 条只到状态层：预算用尽推 `ServerStatus::Failed`，状态栏显示
    「Connection Failed」，但还没有说明原因的具体文案。
  - 第 5、6 条未覆盖：现有 `HostManager` 相关测试未单独跑一遍回归，
    macOS 侧也未实跑。
  - 因此该项保持 `[~]`，不因「编译过了」就写成完成。
