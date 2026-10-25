// Copyright 2026 QuickDesk Authors

#include "manager/AuthManager.h"
#include "manager/RemoteSessionManager.h"

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

// Records what the revoke paths actually sent, so the assertions are about the
// request and not only about how the response was handled. A wrong method or a
// wrong URL passes every response-side assertion while doing nothing on the
// server.
struct CapturedRequest {
    QString method;
    QUrl url;
    QString body;
};

CapturedRequest g_last_request;

// What the next response should be. Defaults to the empty success the refresh
// case expects; each revoke case overrides what it needs and restores it after,
// so a later case is not silently answering with a previous case's failure.
int g_next_status = 200;
std::string g_next_error;
std::string g_next_data;

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
    // The refresh above only ever GETs. The revoke cases below need to answer
    // differently, so the handler records the request and consults a script that
    // each case sets up.
    quickdesk::SetAuthTestRequestHandler(
        [](const QString& method, const QUrl& url, const QString& body,
           quickdesk::AuthManager::ResponseCallback callback) {
            g_last_request.method = method;
            g_last_request.url = url;
            g_last_request.body = body;
            callback(g_next_status, g_next_error, g_next_data);
        });

    quickdesk::AuthManager authManager(nullptr);
    quickdesk::RemoteSessionManager manager(&authManager);

    // --- QD-209: single-session revoke and per-device bulk clear ---
    //
    // The service and the Web client were already done; what was missing was any
    // automated check that the Qt side issues the right request and reacts to the
    // right response. A wrong method or URL would pass every response-side
    // assertion while doing nothing on the server, which is why these cases
    // assert on the captured request too.

    // An empty id must not produce a request at all. Sending one would hit
    // /v1/me/remote-sessions//revoke and be rejected for a reason that has
    // nothing to do with what the user did.
    g_last_request = CapturedRequest();
    manager.revokeSession(QString());
    QCoreApplication::processEvents(QEventLoop::AllEvents, 50);
    Expect(g_last_request.method.isEmpty(),
           QStringLiteral("revokeSession with an empty id sends nothing"));

    g_last_request = CapturedRequest();
    QSignalSpy revokedSpy(&manager, SIGNAL(sessionRevoked(QString)));
    QSignalSpy failedSpy(&manager, SIGNAL(requestFailed(QString)));
    manager.revokeSession(QStringLiteral("session-1"));
    Expect(WaitUntil([&]() { return revokedSpy.count() > 0 || failedSpy.count() > 0; }, 5000),
           QStringLiteral("revokeSession completes"));
    Expect(g_last_request.method == QStringLiteral("POST"),
           QStringLiteral("revokeSession uses POST"));
    Expect(g_last_request.url.path()
               == QStringLiteral("/v1/me/remote-sessions/session-1/revoke"),
           QStringLiteral("revokeSession targets the per-session revoke path"));
    Expect(revokedSpy.count() == 1,
           QStringLiteral("a successful revoke signals sessionRevoked"));
    if (revokedSpy.count() == 1) {
        Expect(revokedSpy.first().at(0).toString()
                   == QStringLiteral("session-1"),
               QStringLiteral("sessionRevoked carries the revoked id"));
    }
    Expect(failedSpy.count() == 0,
           QStringLiteral("a successful revoke does not report failure"));
    Expect(manager.lastError().isEmpty(),
           QStringLiteral("a successful revoke clears the last error"));

    // An id that needs escaping must arrive escaped, or the path is wrong on the
    // server and the failure looks like a missing session.
    g_last_request = CapturedRequest();
    revokedSpy.clear();
    manager.revokeSession(QStringLiteral("session/with space"));
    Expect(WaitUntil([&]() { return revokedSpy.count() > 0; }, 5000),
           QStringLiteral("revoke with a special id completes"));
    // Assert on the encoded form, not on path(). QUrl::path() returns the decoded
    // path, so a %2F comes back as a literal slash and the very defect being
    // tested for becomes invisible - the first version of this check compared
    // path() and could not have caught it.
    const QString actualPath =
        QString::fromLatin1(g_last_request.url.toEncoded());
    // Count only the path part. The encoded form is the whole URL, so scheme and
    // authority contribute their own slashes; counting them made a correct
    // encoding look like six segments.
    const int scheme_end = actualPath.indexOf(QStringLiteral("://"));
    const QString path_only =
        scheme_end < 0 ? actualPath
                       : actualPath.mid(actualPath.indexOf(
                             '/', scheme_end + 3));
    const QStringList path_segments =
        path_only.split('/', Qt::SkipEmptyParts);
    // v1 / me / remote-sessions / <id> / revoke. Five, not six: the id must not
    // contribute an extra segment.
    Expect(path_segments.size() == 5,
           QStringLiteral("an id containing a slash stays one path segment "
                          "(path %1 split into %2)")
               .arg(path_only, QString::number(path_segments.size())));
    if (path_segments.size() == 5) {
        Expect(path_segments.at(0) == QStringLiteral("v1")
                   && path_segments.at(1) == QStringLiteral("me")
                   && path_segments.at(2) == QStringLiteral("remote-sessions")
                   && path_segments.at(4) == QStringLiteral("revoke"),
               QStringLiteral("the prefix and suffix are intact (path %1)")
                   .arg(path_only));
        // The id itself must still be one segment holding the encoded slash and
        // the encoded space, not the literal characters.
        Expect(path_segments.at(3)
                   == QStringLiteral("session%2Fwith%20space"),
               QStringLiteral("the id is encoded as a single segment "
                              "(got %1)").arg(path_segments.at(3)));
    }

    // A non-200 must surface as a failure with the server's own words, not as a
    // silent success. Clearing a session that is already gone is a real case.
    g_last_request = CapturedRequest();
    revokedSpy.clear();
    failedSpy.clear();
    g_next_status = 404;
    g_next_data = R"({"detail":"session not found"})";
    manager.revokeSession(QStringLiteral("gone"));
    Expect(WaitUntil([&]() { return failedSpy.count() > 0; }, 5000),
           QStringLiteral("a failed revoke reports requestFailed"));
    Expect(revokedSpy.count() == 0,
           QStringLiteral("a failed revoke does not signal success"));
    Expect(failedSpy.first().at(0).toString()
               == QStringLiteral("session not found"),
           QStringLiteral("the server's message reaches the caller"));
    Expect(!manager.lastError().isEmpty(),
           QStringLiteral("a failed revoke leaves an error to display"));
    g_next_status = 200;
    g_next_data = "";

    // Per-device bulk clear: the counts in the response are the whole point, so
    // they have to arrive intact.
    g_last_request = CapturedRequest();
    QSignalSpy clearedSpy(&manager,
                          SIGNAL(deviceSessionsRevoked(QString,int,int,bool)));
    g_next_data = R"({"revoked":3,"failed":1,"truncated":true})";
    manager.revokeDeviceSessions(QStringLiteral("device-a"));
    Expect(WaitUntil([&]() { return clearedSpy.count() > 0 || failedSpy.count() > 0; }, 5000),
           QStringLiteral("per-device clear completes"));
    Expect(g_last_request.method == QStringLiteral("POST"),
           QStringLiteral("per-device clear uses POST"));
    Expect(g_last_request.url.path()
               == QStringLiteral("/v1/me/devices/device-a/sessions:revoke"),
           QStringLiteral("per-device clear targets the bulk revoke path"));
    Expect(clearedSpy.count() == 1,
           QStringLiteral("a successful clear signals deviceSessionsRevoked"));
    if (clearedSpy.count() == 1) {
        const QList<QVariant> args = clearedSpy.first();
        Expect(args.at(0).toString() == QStringLiteral("device-a"),
               QStringLiteral("the signal carries the device id"));
        Expect(args.at(1).toInt() == 3,
               QStringLiteral("the revoked count reaches the caller"));
        Expect(args.at(2).toInt() == 1,
               QStringLiteral("the failed count reaches the caller"));
        Expect(args.at(3).toBool() == true,
               QStringLiteral("the truncated flag reaches the caller"));
    }

    // Same guard as the single-session path.
    g_last_request = CapturedRequest();
    clearedSpy.clear();
    manager.revokeDeviceSessions(QString());
    QCoreApplication::processEvents(QEventLoop::AllEvents, 50);
    Expect(g_last_request.method.isEmpty(),
           QStringLiteral("per-device clear with an empty id sends nothing"));

    // A bulk clear that fails must not be mistaken for "nothing to clear".
    g_last_request = CapturedRequest();
    clearedSpy.clear();
    failedSpy.clear();
    g_next_status = 503;
    g_next_data = R"({"detail":"signaling unavailable"})";
    manager.revokeDeviceSessions(QStringLiteral("device-b"));
    Expect(WaitUntil([&]() { return failedSpy.count() > 0; }, 5000),
           QStringLiteral("a failed clear reports requestFailed"));
    Expect(clearedSpy.count() == 0,
           QStringLiteral("a failed clear does not signal success"));
    Expect(failedSpy.first().at(0).toString()
               == QStringLiteral("signaling unavailable"),
           QStringLiteral("the server's message reaches the caller"));
    g_next_status = 200;
    g_next_data = "";
    manager.refresh(QStringLiteral("active"), QStringLiteral("device-a"));
    Expect(WaitUntil([&]() { return !manager.busy(); }, 5000),
           QStringLiteral("session refresh finishes"));
    // Kept from before the revoke cases existed: the refresh is still a GET, and
    // dropping that assertion while generalising the handler would have quietly
    // removed coverage.
    Expect(g_last_request.method == "GET",
           QStringLiteral("session refresh uses GET"));
    Expect(g_last_request.url.path().contains(
               QStringLiteral("/v1/me/remote-sessions")),
           QStringLiteral("session refresh targets the remote-sessions path"));

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
