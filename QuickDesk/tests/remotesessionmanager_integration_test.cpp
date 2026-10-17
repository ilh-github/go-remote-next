// Copyright 2026 QuickDesk Authors

#include "manager/AuthManager.h"
#include "manager/RemoteSessionManager.h"

#include <QCoreApplication>
#include <QElapsedTimer>
#include <QEventLoop>
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

QVariantMap sessionEvent(const QString& type, const QString& deviceId,
                         const QString& sessionId) {
    return {
        {QStringLiteral("type"), type},
        {QStringLiteral("data"),
         QVariantMap{
             {QStringLiteral("device_id"), deviceId},
             {QStringLiteral("session_id"), sessionId},
         }},
    };
}

QVariantMap policyEvent(const QString& deviceId, const QStringList& denied) {
    return {
        {QStringLiteral("type"),
         QStringLiteral("device.access_policy.changed")},
        {QStringLiteral("data"),
         QVariantMap{
             {QStringLiteral("device_id"), deviceId},
             {QStringLiteral("deny_ips"), denied},
         }},
    };
}

}  // namespace

int main(int argc, char** argv) {
    QCoreApplication app(argc, argv);
    quickdesk::SetAuthTestBaseUrl(QStringLiteral("https://quickdesk.test/"));
    quickdesk::SetAuthTestLoggedIn(true);
    quickdesk::SetAuthTestRequestHandler(
        [](const QString& method, const QUrl&, const QString&,
           quickdesk::AuthManager::ResponseCallback callback) {
            Expect(method == "GET",
                   QStringLiteral("session refresh uses GET"));
            callback(200, "", R"({"items":[]})");
        });

    quickdesk::AuthManager authManager(nullptr);
    quickdesk::RemoteSessionManager manager(&authManager);
    manager.refresh(QStringLiteral("active"), QStringLiteral("device-a"));
    Expect(WaitUntil([&]() { return !manager.busy(); }, 5000),
           QStringLiteral("session refresh finishes"));

    QSignalSpy refreshSpy(
        &manager, SIGNAL(realtimeSessionChanged(QString,QString)));

    QVariantMap event =
        sessionEvent(QStringLiteral("remote_session.started"),
                     QStringLiteral("device-a"),
                     QStringLiteral("session-1"));
    Expect(QMetaObject::invokeMethod(
               &manager, "handleRealtimeEvent",
               Q_ARG(QVariantMap, event)),
           QStringLiteral("manager exposes realtime event ingestion"));
    Expect(refreshSpy.count() == 1,
           QStringLiteral("matching session event requests a refresh"));
    const QList<QVariant> refreshArgs = refreshSpy.takeFirst();
    Expect(refreshArgs.at(0).toString() == QStringLiteral("device-a"),
           QStringLiteral("refresh signal carries the event device"));
    Expect(refreshArgs.at(1).toString()
               == QStringLiteral("remote_session.started"),
           QStringLiteral("refresh signal carries the lifecycle type"));

    event = sessionEvent(QStringLiteral("remote_session.ended"),
                         QStringLiteral("device-b"),
                         QStringLiteral("session-2"));
    Expect(QMetaObject::invokeMethod(
               &manager, "handleRealtimeEvent",
               Q_ARG(QVariantMap, event)),
           QStringLiteral("manager accepts non-matching session events"));
    Expect(refreshSpy.count() == 0,
           QStringLiteral("non-matching session event is ignored"));

    event = policyEvent(
        QStringLiteral("device-a"),
        QStringList{QStringLiteral("203.0.113.9"),
                    QStringLiteral("198.51.100.0/24")});
    Expect(QMetaObject::invokeMethod(
               &manager, "handleRealtimeEvent",
               Q_ARG(QVariantMap, event)),
           QStringLiteral("manager accepts policy events"));
    Expect(manager.deniedSources().size() == 2,
           QStringLiteral("matching policy event updates blocked sources"));
    Expect(manager.deniedSources().first().toString()
               == QStringLiteral("203.0.113.9"),
           QStringLiteral("policy event preserves source order"));

    event = policyEvent(QStringLiteral("device-b"),
                        QStringList{QStringLiteral("192.0.2.1")});
    Expect(QMetaObject::invokeMethod(
               &manager, "handleRealtimeEvent",
               Q_ARG(QVariantMap, event)),
           QStringLiteral("manager accepts non-matching policy events"));
    Expect(manager.deniedSources().size() == 2,
           QStringLiteral("non-matching policy event is ignored"));

    std::cout << "remote session realtime event tests passed\n";
    return 0;
}
