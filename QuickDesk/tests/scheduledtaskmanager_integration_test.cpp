// Copyright 2026 QuickDesk Authors

#include "manager/AuthManager.h"
#include "manager/ScheduledTaskManager.h"

#include <QCoreApplication>
#include <QElapsedTimer>
#include <QEventLoop>
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

    QList<QJsonObject> createBodies;
    QString deletedPath;
    int nextTaskId = 40;
    quickdesk::SetAuthTestRequestHandler(
        [&](const QString& method, const QUrl& url, const QString& body,
            quickdesk::AuthManager::ResponseCallback callback) {
            if (method == "GET") {
                callback(
                    200, "",
                    R"({"items":[{"id":12,"kind":"sign_out_all_sessions","next_run_at":"2026-10-17T14:00:00Z","repeat_seconds":0,"enabled":false,"schedule":"daily@22:00","timezone":"Asia/Shanghai","skip_holidays":true,"max_attempts":3,"attempts":0,"last_status":"ok"}],"supported_kinds":["revoke_all_remote_sessions","sign_out_all_sessions"],"supported_schedules":["daily@HH:MM","weekly@<dow>@HH:MM"]})");
                return;
            }
            if (method == "POST") {
                createBodies.append(
                    QJsonDocument::fromJson(body.toUtf8()).object());
                const int id = ++nextTaskId;
                callback(
                    201, "",
                    QStringLiteral(
                        R"({"id":%1,"kind":"sign_out_all_sessions","next_run_at":"2026-10-18T14:00:00Z","repeat_seconds":0,"enabled":true,"schedule":"","timezone":"","skip_holidays":false,"max_attempts":1,"attempts":0})")
                        .arg(id)
                        .toStdString());
                return;
            }
            if (method == "DELETE") {
                deletedPath = url.path();
                callback(204, "", "");
                return;
            }
            callback(405, "unexpected method", "");
        });

    quickdesk::AuthManager authManager(nullptr);
    quickdesk::ScheduledTaskManager manager(&authManager);

    manager.refresh();
    Expect(WaitUntil([&]() { return !manager.busy(); }, 5000),
           QStringLiteral("scheduled task refresh finishes"));
    Expect(manager.tasks().size() == 1,
           QStringLiteral("task list is populated"));
    Expect(manager.tasks().first().toMap()
               .value(QStringLiteral("max_attempts")).toInt() == 3,
           QStringLiteral("task attempt budget is normalized"));
    Expect(manager.supportedKinds()
               == QStringList{QStringLiteral("revoke_all_remote_sessions"),
                              QStringLiteral("sign_out_all_sessions")},
           QStringLiteral("server-supported kinds are preserved"));
    Expect(manager.supportedSchedules().size() == 2,
           QStringLiteral("server-supported schedules are preserved"));

    QSignalSpy createdSpy(&manager, SIGNAL(taskCreated(int)));
    manager.createTask(QStringLiteral("sign_out_all_sessions"),
                       QStringLiteral("2026-10-18T22:00:00.000Z"),
                       0, 2, QString(), QString(), false);
    Expect(WaitUntil([&]() { return createdSpy.count() == 1; }, 5000),
           QStringLiteral("one-shot task creation completes"));
    Expect(createBodies.size() == 1,
           QStringLiteral("one-shot task posts one payload"));
    const QJsonObject oneShot = createBodies.first();
    Expect(oneShot.value(QStringLiteral("kind")).toString()
               == QStringLiteral("sign_out_all_sessions"),
           QStringLiteral("one-shot payload carries the kind"));
    Expect(oneShot.value(QStringLiteral("run_at")).toString()
               == QStringLiteral("2026-10-18T22:00:00.000Z"),
           QStringLiteral("one-shot payload carries RFC3339 run_at"));
    Expect(oneShot.value(QStringLiteral("repeat_seconds")).toInt() == 0,
           QStringLiteral("one-shot payload repeats zero times"));
    Expect(oneShot.value(QStringLiteral("max_attempts")).toInt() == 2,
           QStringLiteral("one-shot payload carries retry budget"));
    Expect(!oneShot.contains(QStringLiteral("schedule")),
           QStringLiteral("one-shot payload omits calendar schedule"));

    manager.createTask(QStringLiteral("sign_out_all_sessions"),
                       QString(), 0, 4,
                       QStringLiteral("weekly@mon@22:30"),
                       QStringLiteral("Asia/Shanghai"), true);
    Expect(WaitUntil([&]() { return createdSpy.count() == 2; }, 5000),
           QStringLiteral("calendar task creation completes"));
    const QJsonObject calendar = createBodies.at(1);
    Expect(calendar.value(QStringLiteral("schedule")).toString()
               == QStringLiteral("weekly@mon@22:30"),
           QStringLiteral("calendar payload carries the schedule"));
    Expect(calendar.value(QStringLiteral("timezone")).toString()
               == QStringLiteral("Asia/Shanghai"),
           QStringLiteral("calendar payload carries the timezone"));
    Expect(calendar.value(QStringLiteral("skip_holidays")).toBool(),
           QStringLiteral("calendar payload carries holiday skipping"));
    Expect(!calendar.contains(QStringLiteral("run_at")),
           QStringLiteral("calendar payload omits run_at"));
    Expect(!calendar.contains(QStringLiteral("repeat_seconds")),
           QStringLiteral("calendar payload omits repeat_seconds"));

    QSignalSpy deletedSpy(&manager, SIGNAL(taskDeleted(int)));
    manager.deleteTask(12);
    Expect(WaitUntil([&]() { return deletedSpy.count() == 1; }, 5000),
           QStringLiteral("task deletion completes"));
    Expect(deletedPath == QStringLiteral("/v1/me/scheduled-tasks/12"),
           QStringLiteral("task deletion uses the owner-scoped route"));

    QSignalSpy failedSpy(
        &manager, SIGNAL(realtimeTaskFailed(QString,int,QString)));
    QSignalSpy refreshSpy(&manager, SIGNAL(realtimeRefreshRequested()));
    const QVariantMap event{
        {QStringLiteral("type"), QStringLiteral("scheduled_task.failed")},
        {QStringLiteral("data"),
         QVariantMap{
             {QStringLiteral("task_id"), 52},
             {QStringLiteral("kind"),
              QStringLiteral("sign_out_all_sessions")},
             {QStringLiteral("error"), QStringLiteral("worker failed")},
         }},
    };
    manager.handleRealtimeEvent(event);
    Expect(failedSpy.count() == 1,
           QStringLiteral("failed event is surfaced to the page"));
    Expect(refreshSpy.count() == 1,
           QStringLiteral("failed event requests a task refresh"));
    const QList<QVariant> failedArgs = failedSpy.takeFirst();
    Expect(failedArgs.at(0).toString()
               == QStringLiteral("sign_out_all_sessions"),
           QStringLiteral("failed event carries the kind"));
    Expect(failedArgs.at(1).toInt() == 52,
           QStringLiteral("failed event carries the task id"));
    Expect(failedArgs.at(2).toString() == QStringLiteral("worker failed"),
           QStringLiteral("failed event carries the last error"));

    std::cout << "scheduled task manager integration tests passed\n";
    return 0;
}
