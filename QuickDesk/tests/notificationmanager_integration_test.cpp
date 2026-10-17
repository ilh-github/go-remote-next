// Copyright 2026 QuickDesk Authors

#include "manager/AuthManager.h"
#include "manager/NotificationManager.h"

#include <QCoreApplication>
#include <QElapsedTimer>
#include <QEventLoop>
#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QSignalSpy>
#include <QThread>
#include <QUrl>

#include <cstdlib>
#include <functional>
#include <iostream>

namespace quickdesk {
void SetAuthTestBaseUrl(const QString& baseUrl);
void SetAuthTestLoggedIn(bool loggedIn);
void SetAuthTestRequestHandler(
    std::function<void(const QString&,
                       const QUrl&,
                       const QString&,
                       AuthManager::ResponseCallback)> handler);
}

namespace {

void Expect(bool condition, const QString& message) {
    if (!condition) {
        std::cerr << "FAIL: " << message.toStdString() << '\n';
        std::exit(1);
    }
}

bool WaitUntil(const std::function<bool()>& predicate, int timeoutMs) {
    QElapsedTimer timer;
    timer.start();
    while (!predicate() && timer.elapsed() < timeoutMs) {
        QCoreApplication::processEvents(QEventLoop::AllEvents, 50);
        QThread::msleep(10);
    }
    return predicate();
}

}  // namespace

int main(int argc, char** argv) {
    QCoreApplication app(argc, argv);
    quickdesk::SetAuthTestBaseUrl(QStringLiteral("https://quickdesk.test/"));
    quickdesk::SetAuthTestLoggedIn(true);

    int channelsRequests = 0;
    int quietRequests = 0;
    int nextChannelId = 20;
    QList<QJsonObject> channelBodies;
    QString deletedChannelPath;
    QString testedChannelPath;
    QJsonObject quietPutBody;
    QJsonObject offlinePutBody;
    QString offlineGetPath;
    QString offlineDeletePath;
    QJsonObject metricAlertPutBody;
    QString metricAlertGetPath;
    QString metricAlertDeletePath;
    QString deliveriesPath;
    QString retryDeliveryPath;
    QString ackDeliveryPath;
    QString retryChannelPath;
    QString retryUserPath;
    quickdesk::SetAuthTestRequestHandler(
        [&](const QString& method, const QUrl& url, const QString& body,
            quickdesk::AuthManager::ResponseCallback callback) {
            if (method == "GET"
                && url.path() == QStringLiteral("/v1/me/notification-channels")) {
                ++channelsRequests;
                callback(
                    200, "",
                    R"({"items":[{"id":7,"name":"Operations","url":"https://hooks.example.test/quickdesk","events":"[\"device.offline.sustained\",\"scheduled_task.failed\"]","enabled":true,"last_status":204,"last_triggered":"2026-10-17T05:00:00Z","created_at":"2026-10-16T05:00:00Z"}],"supported_events":["device.offline.sustained","scheduled_task.failed"],"signature_header":"X-Webhook-Signature","signature_scheme":"sha256=<hmac-sha256 of the raw body>"})");
                return;
            }
            if (method == "GET"
                && url.path()
                    == QStringLiteral("/v1/me/notification-quiet-hours")) {
                ++quietRequests;
                callback(
                    200, "",
                    R"({"snoozed_until":"2026-10-17T08:00:00Z","quiet_hours":[{"days":[1,2,3,4,5],"start":"22:00","end":"07:00"}],"timezone":"Asia/Shanghai","active":true,"active_until":"2026-10-17T08:00:00Z"})");
                return;
            }
            if (method == "POST"
                && url.path() == QStringLiteral("/v1/me/notification-channels")) {
                channelBodies.append(
                    QJsonDocument::fromJson(
                        body.toUtf8()).object());
                const int id = ++nextChannelId;
                callback(
                    201, "",
                    QStringLiteral(
                        R"({"id":%1,"name":"Primary","url":"https://hooks.example.test/new","events":"[\"device.access.denied\",\"device.offline.sustained\"]","enabled":false,"last_status":0})")
                        .arg(id)
                        .toStdString());
                return;
            }
            if (method == "PATCH"
                && url.path()
                    == QStringLiteral("/v1/me/notification-channels/21")) {
                channelBodies.append(
                    QJsonDocument::fromJson(
                        body.toUtf8()).object());
                callback(
                    200, "",
                    R"({"id":21,"name":"Renamed","url":"https://hooks.example.test/updated","events":"[\"device.online.changed\"]","enabled":true,"last_status":200})");
                return;
            }
            if (method == "DELETE"
                && url.path()
                    == QStringLiteral("/v1/me/notification-channels/21")) {
                deletedChannelPath = url.path();
                callback(200, "", R"({"success":true})");
                return;
            }
            if (method == "POST"
                && url.path()
                    == QStringLiteral("/v1/me/notification-channels/21/test")) {
                testedChannelPath = url.path();
                callback(202, "", R"({"success":true})");
                return;
            }
            if (method == "PUT"
                && url.path()
                    == QStringLiteral("/v1/me/notification-quiet-hours")) {
                quietPutBody = QJsonDocument::fromJson(
                    body.toUtf8()).object();
                callback(
                    200, "",
                    R"({"snoozed_until":null,"quiet_hours":[{"days":[0,6],"start":"21:30","end":"06:30"}],"timezone":"Europe/Berlin","active":false,"active_until":null})");
                return;
            }
            if (method == "GET"
                && url.path()
                    == QStringLiteral(
                        "/v1/me/devices/device-1/offline-alert")) {
                offlineGetPath = url.path();
                callback(
                    200, "",
                    R"({"device_id":"device-1","enabled":true,"after_minutes":15,"repeat_minutes":60,"snoozed_until":null,"maintenance_windows":[{"days":[6,0],"start":"02:00","end":"03:00"}],"maintenance_timezone":"Asia/Shanghai","in_maintenance_window":false,"offline_since":"2026-10-17T04:00:00Z","updated_at":"2026-10-17T04:10:00Z"})");
                return;
            }
            if (method == "PUT"
                && url.path()
                    == QStringLiteral(
                        "/v1/me/devices/device-1/offline-alert")) {
                offlinePutBody = QJsonDocument::fromJson(
                    body.toUtf8()).object();
                callback(
                    200, "",
                    R"({"device_id":"device-1","enabled":true,"after_minutes":20,"repeat_minutes":120,"snoozed_until":"2026-10-18T02:00:00Z","maintenance_windows":[{"days":[1],"start":"01:00","end":"02:30"}],"maintenance_timezone":"Asia/Shanghai","in_maintenance_window":false})");
                return;
            }
            if (method == "DELETE"
                && url.path()
                    == QStringLiteral(
                        "/v1/me/devices/device-1/offline-alert")) {
                offlineDeletePath = url.path();
                callback(204, "", "");
                return;
            }
            if (method == "GET"
                && url.path()
                    == QStringLiteral(
                        "/v1/me/devices/device-1/metric-alert")) {
                metricAlertGetPath = url.path();
                callback(
                    200, "",
                    R"({"device_id":"device-1","enabled":true,"cpu_percent":75.0,"memory_percent":80.0,"disk_percent":90.0,"sustained_minutes":15,"repeat_minutes":60,"snoozed_until":null,"current":{"captured_at":"2026-10-17T04:10:00Z","cpu_percent":12.5,"memory_percent":42.0,"disk_percent":88.0,"disk_mount":"/"}})");
                return;
            }
            if (method == "PUT"
                && url.path()
                    == QStringLiteral(
                        "/v1/me/devices/device-1/metric-alert")) {
                metricAlertPutBody = QJsonDocument::fromJson(
                    body.toUtf8()).object();
                callback(
                    200, "",
                    R"({"device_id":"device-1","enabled":true,"cpu_percent":81.0,"memory_percent":82.0,"disk_percent":91.0,"sustained_minutes":20,"repeat_minutes":120,"snoozed_until":"2026-10-18T02:00:00Z"})");
                return;
            }
            if (method == "DELETE"
                && url.path()
                    == QStringLiteral(
                        "/v1/me/devices/device-1/metric-alert")) {
                metricAlertDeletePath = url.path();
                callback(204, "", "");
                return;
            }
            if (method == "GET"
                && url.path()
                    == QStringLiteral(
                        "/v1/me/notification-channels/21/deliveries")) {
                deliveriesPath = url.toString();
                callback(
                    200, "",
                    R"({"items":[{"id":31,"channel_id":21,"event_type":"device.offline.sustained","device_id":"device-1","status":"failed","attempts":4,"last_status":500,"last_error":"upstream unavailable","created_at":"2026-10-17T04:00:00Z","acked_at":null},{"id":32,"channel_id":21,"event_type":"scheduled_task.failed","status":"delivered","attempts":1,"last_status":204,"created_at":"2026-10-17T04:01:00Z","acked_at":null}],"unacked":1})");
                return;
            }
            if (method == "POST"
                && url.path()
                    == QStringLiteral(
                        "/v1/me/notification-channels/21/deliveries/31/retry")) {
                retryDeliveryPath = url.path();
                callback(
                    200, "",
                    R"({"id":31,"channel_id":21,"event_type":"device.offline.sustained","status":"pending","attempts":4})");
                return;
            }
            if (method == "POST"
                && url.path()
                    == QStringLiteral(
                        "/v1/me/notification-channels/21/deliveries/32/ack")) {
                ackDeliveryPath = url.path();
                callback(
                    200, "",
                    R"({"id":32,"channel_id":21,"event_type":"scheduled_task.failed","status":"delivered","attempts":1,"last_status":204,"acked_at":"2026-10-17T04:02:00Z"})");
                return;
            }
            if (method == "POST"
                && url.path()
                    == QStringLiteral(
                        "/v1/me/notification-channels/21/deliveries:retry-failed")) {
                retryChannelPath = url.path();
                callback(
                    200, "",
                    R"({"requeued":3,"matched":3,"limit":0,"dry_run":false})");
                return;
            }
            if (method == "POST"
                && url.path()
                    == QStringLiteral(
                        "/v1/me/notification-channels:retry-failed")) {
                retryUserPath = url.path();
                callback(
                    200, "",
                    R"({"requeued":2,"matched":2,"limit":0,"dry_run":false})");
                return;
            }
            callback(404, "unexpected path", "");
        });

    quickdesk::AuthManager authManager(nullptr);
    quickdesk::NotificationManager manager(&authManager);

    manager.refreshChannels();
    Expect(WaitUntil([&]() { return !manager.busy(); }, 5000),
           QStringLiteral("channel refresh finishes"));
    Expect(channelsRequests == 1,
           QStringLiteral("channel refresh uses the owner-scoped route"));
    Expect(manager.channels().size() == 1,
           QStringLiteral("channel list is populated"));
    const QVariantMap channel = manager.channels().first().toMap();
    Expect(channel.value(QStringLiteral("id")).toInt() == 7,
           QStringLiteral("channel id is numeric"));
    Expect(channel.value(QStringLiteral("name")).toString()
               == QStringLiteral("Operations"),
           QStringLiteral("channel name is preserved"));
    Expect(channel.value(QStringLiteral("events")).toStringList()
               == QStringList{QStringLiteral("device.offline.sustained"),
                              QStringLiteral("scheduled_task.failed")},
           QStringLiteral("stored event JSON becomes a string list"));
    Expect(manager.supportedEvents()
               == QStringList{QStringLiteral("device.offline.sustained"),
                              QStringLiteral("scheduled_task.failed")},
           QStringLiteral("supported event vocabulary is preserved"));
    Expect(manager.signatureHeader()
               == QStringLiteral("X-Webhook-Signature"),
           QStringLiteral("signature header is preserved"));

    manager.loadQuietHours();
    Expect(WaitUntil([&]() { return !manager.busy(); }, 5000),
           QStringLiteral("quiet-hours refresh finishes"));
    Expect(quietRequests == 1,
           QStringLiteral("quiet-hours refresh uses the owner-scoped route"));
    Expect(manager.quietHours().value(QStringLiteral("active")).toBool(),
           QStringLiteral("quiet-hours active state is preserved"));
    Expect(manager.quietHours().value(QStringLiteral("timezone")).toString()
               == QStringLiteral("Asia/Shanghai"),
           QStringLiteral("quiet-hours timezone is preserved"));
    const QVariantList windows =
        manager.quietHours().value(QStringLiteral("quiet_hours")).toList();
    Expect(windows.size() == 1,
           QStringLiteral("quiet-hours windows are normalized"));
    Expect(windows.first().toMap()
               .value(QStringLiteral("start")).toString()
               == QStringLiteral("22:00"),
           QStringLiteral("quiet-hours window start is preserved"));

    QSignalSpy createdSpy(
        &manager, SIGNAL(channelCreated(int)));
    manager.createChannel(
        QStringLiteral("Primary"),
        QStringLiteral("https://hooks.example.test/new"),
        QStringLiteral("signing-secret"),
        QStringList{QStringLiteral("device.access.denied"),
                    QStringLiteral("device.offline.sustained")},
        false);
    Expect(WaitUntil([&]() { return createdSpy.count() == 1; }, 5000),
           QStringLiteral("channel creation completes"));
    Expect(channelBodies.size() == 1,
           QStringLiteral("channel creation sends one body"));
    const QJsonObject createBody = channelBodies.first();
    Expect(createBody.value(QStringLiteral("url")).toString()
               == QStringLiteral("https://hooks.example.test/new"),
           QStringLiteral("create payload carries the webhook URL"));
    Expect(createBody.value(QStringLiteral("enabled")).toBool() == false,
           QStringLiteral("create payload carries disabled state"));
    Expect(createBody.value(QStringLiteral("events")).toArray().size() == 2,
           QStringLiteral("create payload carries selected events"));
    Expect(createBody.value(QStringLiteral("secret")).toString()
               == QStringLiteral("signing-secret"),
           QStringLiteral("create payload carries the signing secret"));
    Expect(manager.channels().size() == 2,
           QStringLiteral("created channel is appended to the list"));

    QSignalSpy updatedSpy(
        &manager, SIGNAL(channelUpdated(int)));
    manager.updateChannel(
        21, QStringLiteral("Renamed"),
        QStringLiteral("https://hooks.example.test/updated"),
        QString(),
        QStringList{QStringLiteral("device.online.changed")}, true);
    Expect(WaitUntil([&]() { return updatedSpy.count() == 1; }, 5000),
           QStringLiteral("channel update completes"));
    const QJsonObject updateBody = channelBodies.at(1);
    Expect(!updateBody.contains(QStringLiteral("secret")),
           QStringLiteral("editing without a secret preserves the stored one"));
    Expect(updateBody.value(QStringLiteral("name")).toString()
               == QStringLiteral("Renamed"),
           QStringLiteral("update payload carries the new name"));
    Expect(manager.channels().size() == 2,
           QStringLiteral("updated channel replaces its list row"));
    Expect(manager.channels().at(1).toMap()
               .value(QStringLiteral("name")).toString()
               == QStringLiteral("Renamed"),
           QStringLiteral("updated list row carries the server response"));

    QSignalSpy testSpy(
        &manager, SIGNAL(channelTestQueued(int)));
    manager.testChannel(21);
    Expect(WaitUntil([&]() { return testSpy.count() == 1; }, 5000),
           QStringLiteral("channel test completes"));
    Expect(testedChannelPath
               == QStringLiteral("/v1/me/notification-channels/21/test"),
           QStringLiteral("channel test uses the owner-scoped route"));

    QSignalSpy deletedSpy(
        &manager, SIGNAL(channelDeleted(int)));
    manager.deleteChannel(21);
    Expect(WaitUntil([&]() { return deletedSpy.count() == 1; }, 5000),
           QStringLiteral("channel deletion completes"));
    Expect(deletedChannelPath
               == QStringLiteral("/v1/me/notification-channels/21"),
           QStringLiteral("channel deletion uses the owner-scoped route"));
    Expect(manager.channels().size() == 1,
           QStringLiteral("deleted channel is removed from the list"));

    manager.saveQuietHours(
        QStringLiteral("2026-10-17T09:30:00Z"),
        QVariantList{
            QVariantMap{
                {QStringLiteral("days"), QVariantList{0, 6}},
                {QStringLiteral("start"), QStringLiteral("21:30")},
                {QStringLiteral("end"), QStringLiteral("06:30")},
            },
        },
        QStringLiteral("Europe/Berlin"));
    Expect(WaitUntil([&]() { return !manager.busy(); }, 5000),
           QStringLiteral("quiet-hours save finishes"));
    Expect(quietPutBody.value(QStringLiteral("timezone")).toString()
               == QStringLiteral("Europe/Berlin"),
           QStringLiteral("quiet-hours save carries the timezone"));
    Expect(quietPutBody.value(QStringLiteral("quiet_hours")).toArray().size()
               == 1,
           QStringLiteral("quiet-hours save carries one window"));
    Expect(manager.quietHours().value(QStringLiteral("timezone")).toString()
               == QStringLiteral("Europe/Berlin"),
           QStringLiteral("saved quiet hours replace the manager state"));

    manager.loadOfflineAlert(QStringLiteral("device-1"));
    Expect(WaitUntil([&]() { return !manager.busy(); }, 5000),
           QStringLiteral("offline alert load finishes"));
    Expect(offlineGetPath
               == QStringLiteral("/v1/me/devices/device-1/offline-alert"),
           QStringLiteral("offline alert uses the owner-scoped device route"));
    Expect(manager.offlineAlert()
               .value(QStringLiteral("after_minutes")).toInt() == 15,
           QStringLiteral("offline alert threshold is preserved"));
    Expect(manager.offlineAlert()
               .value(QStringLiteral("maintenance_windows")).toList().size()
               == 1,
           QStringLiteral("offline maintenance windows are normalized"));

    manager.saveOfflineAlert(
        true, 20, 120, QStringLiteral("2026-10-18T02:00:00Z"),
        QVariantList{
            QVariantMap{
                {QStringLiteral("days"), QVariantList{1}},
                {QStringLiteral("start"), QStringLiteral("01:00")},
                {QStringLiteral("end"), QStringLiteral("02:30")},
            },
        },
        QStringLiteral("Asia/Shanghai"));
    Expect(WaitUntil([&]() { return !manager.busy(); }, 5000),
           QStringLiteral("offline alert save finishes"));
    Expect(offlinePutBody.value(QStringLiteral("after_minutes")).toInt() == 20,
           QStringLiteral("offline alert save carries the threshold"));
    Expect(offlinePutBody.value(QStringLiteral("repeat_minutes")).toInt()
               == 120,
           QStringLiteral("offline alert save carries the repeat interval"));
    Expect(offlinePutBody.value(QStringLiteral("maintenance_timezone")).toString()
               == QStringLiteral("Asia/Shanghai"),
           QStringLiteral("offline alert save carries its timezone"));

    manager.removeOfflineAlert();
    Expect(WaitUntil([&]() { return !manager.busy(); }, 5000),
           QStringLiteral("offline alert delete finishes"));
    Expect(offlineDeletePath
               == QStringLiteral("/v1/me/devices/device-1/offline-alert"),
           QStringLiteral("offline alert delete uses the owner-scoped route"));
    Expect(!manager.offlineAlert().value(QStringLiteral("enabled")).toBool(),
           QStringLiteral("offline alert delete resets the manager state"));

    manager.loadMetricAlert(QStringLiteral("device-1"));
    Expect(WaitUntil([&]() { return !manager.busy(); }, 5000),
           QStringLiteral("metric alert load finishes"));
    Expect(metricAlertGetPath
               == QStringLiteral("/v1/me/devices/device-1/metric-alert"),
           QStringLiteral("metric alert uses the owner-scoped device route"));
    Expect(manager.metricAlert()
               .value(QStringLiteral("cpu_percent")).toInt() == 75,
           QStringLiteral("metric alert threshold is preserved"));
    Expect(manager.metricCurrentValue()
               .value(QStringLiteral("cpu_percent")).toDouble() == 12.5,
           QStringLiteral("metric alert current values are exposed"));

    manager.saveMetricAlert(
        true, 81, 82, 91, 20, 120,
        QStringLiteral("2026-10-18T02:00:00Z"));
    Expect(WaitUntil([&]() { return !manager.busy(); }, 5000),
           QStringLiteral("metric alert save finishes"));
    Expect(metricAlertPutBody.value(QStringLiteral("enabled")).toBool(),
           QStringLiteral("metric alert save carries enabled state"));
    Expect(metricAlertPutBody.value(QStringLiteral("cpu_percent")).toInt()
               == 81,
           QStringLiteral("metric alert save carries the CPU threshold"));
    Expect(metricAlertPutBody.value(QStringLiteral("memory_percent")).toInt()
               == 82,
           QStringLiteral("metric alert save carries the memory threshold"));
    Expect(metricAlertPutBody.value(QStringLiteral("disk_percent")).toInt()
               == 91,
           QStringLiteral("metric alert save carries the disk threshold"));
    Expect(metricAlertPutBody.value(QStringLiteral("sustained_minutes")).toInt()
               == 20,
           QStringLiteral("metric alert save carries the sustained window"));
    Expect(metricAlertPutBody.value(QStringLiteral("repeat_minutes")).toInt()
               == 120,
           QStringLiteral("metric alert save carries the repeat interval"));
    Expect(metricAlertPutBody.value(QStringLiteral("snoozed_until")).toString()
               == QStringLiteral("2026-10-18T02:00:00Z"),
           QStringLiteral("metric alert save carries the snooze deadline"));

    manager.handleRealtimeEvent(
        QVariantMap{
            {QStringLiteral("type"),
             QStringLiteral("device.metrics.updated")},
            {QStringLiteral("data"),
             QVariantMap{
                 {QStringLiteral("device_id"), QStringLiteral("device-1")},
                 {QStringLiteral("captured_at"),
                  QStringLiteral("2026-10-17T04:15:00Z")},
                 {QStringLiteral("cpu_percent"), 44.5},
                 {QStringLiteral("memory_percent"), 55.5},
                 {QStringLiteral("disk_percent"), 92.0},
                 {QStringLiteral("disk_mount"), QStringLiteral("/")},
             }},
        });
    Expect(manager.metricCurrentValue()
               .value(QStringLiteral("cpu_percent")).toDouble() == 44.5,
           QStringLiteral("metric realtime event updates current values"));

    manager.removeMetricAlert();
    Expect(WaitUntil([&]() { return !manager.busy(); }, 5000),
           QStringLiteral("metric alert delete finishes"));
    Expect(metricAlertDeletePath
               == QStringLiteral("/v1/me/devices/device-1/metric-alert"),
           QStringLiteral("metric alert delete uses the owner-scoped route"));
    Expect(!manager.metricAlert().value(QStringLiteral("enabled")).toBool(),
           QStringLiteral("metric alert delete resets the manager state"));

    manager.loadDeliveries(21);
    Expect(WaitUntil([&]() { return !manager.busy(); }, 5000),
           QStringLiteral("delivery list finishes"));
    Expect(deliveriesPath.contains(QStringLiteral("limit=100")),
           QStringLiteral("delivery list asks for the operator page limit"));
    Expect(manager.deliveries().size() == 2,
           QStringLiteral("delivery list is populated"));
    Expect(manager.unackedDeliveries() == 1,
           QStringLiteral("unacked delivery count is preserved"));

    manager.retryDelivery(21, 31);
    Expect(WaitUntil([&]() { return !manager.busy(); }, 5000),
           QStringLiteral("delivery retry finishes"));
    Expect(retryDeliveryPath
               == QStringLiteral(
                   "/v1/me/notification-channels/21/deliveries/31/retry"),
           QStringLiteral("delivery retry uses the channel-scoped route"));

    manager.ackDelivery(21, 32);
    Expect(WaitUntil([&]() { return !manager.busy(); }, 5000),
           QStringLiteral("delivery acknowledgement finishes"));
    Expect(ackDeliveryPath
               == QStringLiteral(
                   "/v1/me/notification-channels/21/deliveries/32/ack"),
           QStringLiteral("delivery acknowledgement uses the channel route"));

    manager.retryFailedDeliveries(21);
    Expect(WaitUntil([&]() { return !manager.busy(); }, 5000),
           QStringLiteral("channel failed-delivery replay finishes"));
    Expect(retryChannelPath
               == QStringLiteral(
                   "/v1/me/notification-channels/21/deliveries:retry-failed"),
           QStringLiteral("channel replay uses the channel bulk route"));
    Expect(manager.lastRequeuedCount() == 3,
           QStringLiteral("channel replay reports requeued rows"));

    manager.retryFailedDeliveriesForUser();
    Expect(WaitUntil([&]() { return !manager.busy(); }, 5000),
           QStringLiteral("cross-channel failed-delivery replay finishes"));
    Expect(retryUserPath
               == QStringLiteral("/v1/me/notification-channels:retry-failed"),
           QStringLiteral("cross-channel replay uses the user bulk route"));
    Expect(manager.lastRequeuedCount() == 2,
           QStringLiteral("cross-channel replay reports requeued rows"));

    QSignalSpy realtimeSpy(
        &manager, SIGNAL(realtimeRefreshRequested()));
    manager.handleRealtimeEvent(
        QVariantMap{
            {QStringLiteral("type"),
             QStringLiteral("device.offline.sustained")},
            {QStringLiteral("data"),
             QVariantMap{{QStringLiteral("device_id"),
                          QStringLiteral("device-1")}}},
        });
    Expect(realtimeSpy.count() == 1,
           QStringLiteral("offline realtime event requests a refresh"));

    std::cout << "notification manager refresh tests passed\n";
    return 0;
}
