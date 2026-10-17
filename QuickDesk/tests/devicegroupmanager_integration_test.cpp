// Copyright 2026 QuickDesk Authors

#include "manager/AuthManager.h"
#include "manager/DeviceGroupManager.h"

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

QJsonObject ParseBody(const QString& body) {
    return QJsonDocument::fromJson(body.toUtf8()).object();
}

QStringList JsonStrings(const QJsonArray& array) {
    QStringList values;
    for (const auto& value : array) {
        values.append(value.toString());
    }
    return values;
}

}  // namespace

int main(int argc, char** argv) {
    QCoreApplication app(argc, argv);
    quickdesk::SetAuthTestBaseUrl(QStringLiteral("https://quickdesk.test/"));
    quickdesk::SetAuthTestLoggedIn(true);

    QStringList requestedRoutes;
    QList<QJsonObject> requestBodies;
    bool rejectMembership = false;
    quickdesk::SetAuthTestRequestHandler(
        [&](const QString& method, const QUrl& url, const QString& body,
            quickdesk::AuthManager::ResponseCallback callback) {
            requestedRoutes.append(method + QStringLiteral(" ") + url.path());

            if (method == QLatin1String("GET")
                && url.path() == QLatin1String("/v1/me/device-groups")) {
                callback(
                    200, "",
                    R"({"items":[{"id":7,"user_id":2,"name":" Office ","description":" Primary devices ","color":" #1f6feb ","device_ids":[" device-a ","","device-b","device-a"],"device_count":99,"created_at":"2026-10-17T04:00:00Z","updated_at":"2026-10-17T04:01:00Z"}]})");
                return;
            }

            if (method == QLatin1String("POST")
                && url.path() == QLatin1String("/v1/me/device-groups")) {
                requestBodies.append(ParseBody(body));
                callback(
                    201, "",
                    R"({"id":8,"user_id":2,"name":"Team","description":"Lab","color":"#123456","device_ids":[],"device_count":0,"created_at":"2026-10-17T04:02:00Z","updated_at":"2026-10-17T04:02:00Z"})");
                return;
            }

            if (method == QLatin1String("PATCH")
                && url.path()
                    == QLatin1String("/v1/me/device-groups/7")) {
                requestBodies.append(ParseBody(body));
                callback(
                    200, "",
                    R"({"id":7,"user_id":2,"name":"Renamed","description":"Updated","color":"#abcdef","device_ids":["device-b","device-a"],"device_count":2,"created_at":"2026-10-17T04:00:00Z","updated_at":"2026-10-17T04:03:00Z"})");
                return;
            }

            if (method == QLatin1String("PUT")
                && url.path()
                    == QLatin1String("/v1/me/device-groups/7/devices")) {
                const QJsonObject payload = ParseBody(body);
                requestBodies.append(payload);
                if (rejectMembership) {
                    callback(
                        403,
                        "",
                        R"({"type":"about:blank","title":"Forbidden","status":403,"code":"DEVICE_NOT_OWNED","detail":"One or more devices are not owned by this user"})");
                    return;
                }
                callback(
                    200, "",
                    R"({"id":7,"user_id":2,"name":"Renamed","description":"Updated","color":"#abcdef","device_ids":["device-a","device-b"],"device_count":2,"created_at":"2026-10-17T04:00:00Z","updated_at":"2026-10-17T04:04:00Z"})");
                return;
            }

            if (method == QLatin1String("DELETE")
                && url.path()
                    == QLatin1String("/v1/me/device-groups/7")) {
                callback(200, "", R"({"status":"ok"})");
                return;
            }

            callback(404, "unexpected path", "");
        });

    quickdesk::AuthManager authManager(nullptr);
    quickdesk::DeviceGroupManager manager(&authManager);

    manager.refreshGroups();
    Expect(WaitUntil([&]() { return !manager.busy(); }, 5000),
           QStringLiteral("group refresh finishes"));
    Expect(manager.groups().size() == 1,
           QStringLiteral("group list is populated"));
    const QVariantMap firstGroup = manager.groups().first().toMap();
    Expect(firstGroup.value(QStringLiteral("name")).toString()
               == QStringLiteral("Office"),
           QStringLiteral("group text fields are normalized"));
    Expect(firstGroup.value(QStringLiteral("device_ids")).toStringList()
               == QStringList{QStringLiteral("device-a"),
                              QStringLiteral("device-b")},
           QStringLiteral("member ids are trimmed and deduplicated"));
    Expect(firstGroup.value(QStringLiteral("device_count")).toInt() == 2,
           QStringLiteral("device count is derived from normalized members"));

    QSignalSpy createdSpy(&manager, SIGNAL(groupCreated(int)));
    manager.createGroup(QStringLiteral(" Team "),
                        QStringLiteral(" Lab "),
                        QStringLiteral(" #123456 "));
    Expect(WaitUntil([&]() { return createdSpy.count() == 1; }, 5000),
           QStringLiteral("group creation completes"));
    const QJsonObject createBody = requestBodies.last();
    Expect(createBody.value(QStringLiteral("name")).toString()
               == QStringLiteral("Team"),
           QStringLiteral("create payload trims the name"));
    Expect(createBody.value(QStringLiteral("description")).toString()
               == QStringLiteral("Lab"),
           QStringLiteral("create payload trims the description"));
    Expect(createBody.value(QStringLiteral("color")).toString()
               == QStringLiteral("#123456"),
           QStringLiteral("create payload trims the color"));
    Expect(manager.groups().size() == 2,
           QStringLiteral("created group is appended"));

    QSignalSpy updatedSpy(&manager, SIGNAL(groupUpdated(int)));
    manager.updateGroup(7,
                        QStringLiteral(" Renamed "),
                        QStringLiteral(" Updated "),
                        QStringLiteral(" #abcdef "));
    Expect(WaitUntil([&]() { return updatedSpy.count() == 1; }, 5000),
           QStringLiteral("group update completes"));
    const QJsonObject updateBody = requestBodies.last();
    Expect(updateBody.value(QStringLiteral("name")).toString()
               == QStringLiteral("Renamed"),
           QStringLiteral("update payload trims the name"));
    Expect(updateBody.value(QStringLiteral("description")).toString()
               == QStringLiteral("Updated"),
           QStringLiteral("update payload trims the description"));
    Expect(updateBody.value(QStringLiteral("color")).toString()
               == QStringLiteral("#abcdef"),
           QStringLiteral("update payload trims the color"));

    QSignalSpy membersSpy(&manager, SIGNAL(groupUpdated(int)));
    manager.replaceGroupDevices(
        7, QStringList{QStringLiteral(" device-b "), QString(),
                       QStringLiteral("device-a"),
                       QStringLiteral("device-b")});
    Expect(WaitUntil([&]() { return membersSpy.count() == 1; }, 5000),
           QStringLiteral("member replacement completes"));
    const QJsonObject membersBody = requestBodies.last();
    Expect(JsonStrings(membersBody.value(QStringLiteral("device_ids")).toArray())
               == QStringList{QStringLiteral("device-b"),
                              QStringLiteral("device-a")},
           QStringLiteral("member payload is trimmed and deduplicated"));

    const QVariantList beforeFailure = manager.groups();
    rejectMembership = true;
    QSignalSpy failedSpy(&manager, SIGNAL(requestFailed(QString)));
    manager.replaceGroupDevices(
        7, QStringList{QStringLiteral("device-not-owned")});
    Expect(WaitUntil([&]() { return failedSpy.count() == 1; }, 5000),
           QStringLiteral("membership failure is reported"));
    Expect(manager.groups() == beforeFailure,
           QStringLiteral("failed membership save preserves the group list"));
    Expect(manager.lastError().contains(
               QStringLiteral("not owned by this user")),
           QStringLiteral("membership failure exposes the server detail"));
    rejectMembership = false;

    QSignalSpy deletedSpy(&manager, SIGNAL(groupDeleted(int)));
    manager.deleteGroup(7);
    Expect(WaitUntil([&]() { return deletedSpy.count() == 1; }, 5000),
           QStringLiteral("group deletion completes"));
    Expect(manager.groups().size() == 1,
           QStringLiteral("deleted group is removed"));

    for (const auto& route : requestedRoutes) {
        Expect(!route.contains(QStringLiteral("/v1/admin/groups")),
               QStringLiteral("manager never calls the admin group routes"));
    }

    std::cout << "device group manager integration tests passed\n";
    return 0;
}
