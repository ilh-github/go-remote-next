// Copyright 2026 QuickDesk Authors

#include "manager/AuthManager.h"
#include "manager/RecordingManager.h"

#include <QCoreApplication>
#include <QElapsedTimer>
#include <QEventLoop>
#include <QSignalSpy>
#include <QThread>
#include <QUrl>
#include <QUrlQuery>

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

    int listRequests = 0;
    QString secondCursor;
    QString artifactTicketPath;
    QString thumbnailTicketPath;
    QString deletedPath;
    quickdesk::SetAuthTestRequestHandler(
        [&](const QString& method, const QUrl& url, const QString&,
            quickdesk::AuthManager::ResponseCallback callback) {
            if (method == "GET"
                && url.path() == QStringLiteral("/v1/me/recordings")) {
                ++listRequests;
                const QString cursor =
                    QUrlQuery(url).queryItemValue(QStringLiteral("cursor"));
                if (cursor.isEmpty()) {
                    callback(
                        200, "",
                        R"({"items":[{"id":11,"user_id":2,"device_id":"device-a","device_name":"Studio Mac","session_id":"session-1","status":"completed","format":"mp4","duration_seconds":95,"size_bytes":1048576,"started_at":"2026-10-17T04:00:00Z","ended_at":"2026-10-17T04:01:35Z","artifact_url":"/v1/recordings/11/artifact","thumbnail_url":"/v1/recordings/11/thumbnail","created_at":"2026-10-17T04:00:00Z","updated_at":"2026-10-17T04:01:35Z"}],"next_cursor":"next-page"})");
                } else {
                    secondCursor = cursor;
                    callback(
                        200, "",
                        R"({"items":[{"id":10,"user_id":2,"device_id":"device-b","device_name":"Office PC","status":"failed","format":"mp4","duration_seconds":0,"size_bytes":0,"started_at":"2026-10-16T04:00:00Z","created_at":"2026-10-16T04:00:00Z","updated_at":"2026-10-16T04:01:00Z"}],"next_cursor":""})");
                }
                return;
            }
            if (method == "POST"
                && url.path()
                    == QStringLiteral(
                        "/v1/me/recordings/11/artifact-ticket")) {
                artifactTicketPath = url.path();
                callback(
                    200, "",
                    R"({"url":"/v1/recordings/11/artifact?t=play-token","expires_at":"2026-10-17T05:00:00Z"})");
                return;
            }
            if (method == "POST"
                && url.path()
                    == QStringLiteral(
                        "/v1/me/recordings/11/thumbnail-ticket")) {
                thumbnailTicketPath = url.path();
                callback(
                    200, "",
                    R"({"url":"/v1/recordings/11/thumbnail?t=thumb-token","expires_at":"2026-10-17T05:00:00Z"})");
                return;
            }
            if (method == "DELETE"
                && url.path() == QStringLiteral("/v1/me/recordings/11")) {
                deletedPath = url.path();
                callback(204, "", "");
                return;
            }
            callback(404, "unexpected path", "");
        });

    quickdesk::AuthManager authManager(nullptr);
    quickdesk::RecordingManager manager(&authManager);

    manager.refresh();
    Expect(WaitUntil([&]() { return !manager.busy(); }, 5000),
           QStringLiteral("recording refresh finishes"));
    Expect(listRequests == 1,
           QStringLiteral("recording refresh uses the owner-scoped list"));
    Expect(manager.recordings().size() == 1,
           QStringLiteral("recording list is populated"));
    Expect(manager.nextCursor() == QStringLiteral("next-page"),
           QStringLiteral("recording cursor is preserved"));
    const QVariantMap first = manager.recordings().first().toMap();
    Expect(first.value(QStringLiteral("id")).toInt() == 11,
           QStringLiteral("recording id is numeric"));
    Expect(first.value(QStringLiteral("device_name")).toString()
               == QStringLiteral("Studio Mac"),
           QStringLiteral("recording device name is preserved"));
    Expect(first.value(QStringLiteral("duration_seconds")).toInt() == 95,
           QStringLiteral("recording duration is preserved"));
    Expect(first.value(QStringLiteral("status")).toString()
               == QStringLiteral("completed"),
           QStringLiteral("recording status is preserved"));

    QSignalSpy artifactSpy(
        &manager, SIGNAL(artifactUrlReady(int,QString)));
    manager.requestArtifactUrl(11);
    Expect(WaitUntil([&]() { return artifactSpy.count() == 1; }, 5000),
           QStringLiteral("artifact ticket request completes"));
    Expect(artifactTicketPath
               == QStringLiteral(
                   "/v1/me/recordings/11/artifact-ticket"),
           QStringLiteral("artifact ticket uses the owner-scoped route"));
    const QList<QVariant> artifactArgs = artifactSpy.takeFirst();
    Expect(artifactArgs.at(0).toInt() == 11,
           QStringLiteral("artifact signal carries the recording id"));
    Expect(artifactArgs.at(1).toString()
               == QStringLiteral(
                   "https://quickdesk.test/v1/recordings/11/artifact?t=play-token"),
           QStringLiteral("relative artifact ticket is resolved against the API"));

    QSignalSpy thumbnailSpy(
        &manager, SIGNAL(thumbnailUrlReady(int,QString)));
    manager.requestThumbnailUrl(11);
    Expect(WaitUntil([&]() { return thumbnailSpy.count() == 1; }, 5000),
           QStringLiteral("thumbnail ticket request completes"));
    Expect(thumbnailTicketPath
               == QStringLiteral(
                   "/v1/me/recordings/11/thumbnail-ticket"),
           QStringLiteral("thumbnail ticket uses the owner-scoped route"));
    const QList<QVariant> thumbnailArgs = thumbnailSpy.takeFirst();
    Expect(thumbnailArgs.at(1).toString()
               == QStringLiteral(
                   "https://quickdesk.test/v1/recordings/11/thumbnail?t=thumb-token"),
           QStringLiteral("relative thumbnail ticket is resolved"));

    manager.loadMore();
    Expect(WaitUntil([&]() { return !manager.busy(); }, 5000),
           QStringLiteral("recording load more finishes"));
    Expect(secondCursor == QStringLiteral("next-page"),
           QStringLiteral("load more carries the cursor"));
    Expect(manager.recordings().size() == 2,
           QStringLiteral("load more appends the next page"));
    Expect(manager.nextCursor().isEmpty(),
           QStringLiteral("empty next cursor ends pagination"));

    QSignalSpy deletedSpy(
        &manager, SIGNAL(recordingDeleted(int)));
    manager.deleteRecording(11);
    Expect(WaitUntil([&]() { return deletedSpy.count() == 1; }, 5000),
           QStringLiteral("recording deletion completes"));
    Expect(deletedPath == QStringLiteral("/v1/me/recordings/11"),
           QStringLiteral("recording deletion uses the owner-scoped route"));
    Expect(manager.recordings().size() == 1,
           QStringLiteral("deleted recording is removed from the list"));

    QSignalSpy realtimeSpy(
        &manager, SIGNAL(realtimeRefreshRequested()));
    manager.handleRealtimeEvent(
        QVariantMap{
            {QStringLiteral("type"),
             QStringLiteral("recording.updated")},
            {QStringLiteral("data"),
             QVariantMap{{QStringLiteral("recording_id"), 11}}},
        });
    Expect(realtimeSpy.count() == 1,
           QStringLiteral("recording realtime event requests a refresh"));

    std::cout << "recording manager integration tests passed\n";
    return 0;
}
