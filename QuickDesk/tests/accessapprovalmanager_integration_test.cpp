// Copyright 2026 QuickDesk Authors

#include "manager/AccessApprovalManager.h"
#include "manager/AuthManager.h"

#include <QCoreApplication>
#include <QElapsedTimer>
#include <QEventLoop>
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

    QString listStatus;
    QString listLimit;
    QString summaryLimit;
    QString decisionPath;
    QString decisionBody;
    bool conflictDecision = false;
    int requestCount = 0;
    quickdesk::SetAuthTestRequestHandler(
        [&](const QString& method, const QUrl& url, const QString& body,
            quickdesk::AuthManager::ResponseCallback callback) {
            ++requestCount;
            if (method == "POST") {
                decisionPath = url.path();
                decisionBody = body;
                if (conflictDecision) {
                    callback(
                        409, "already decided",
                        R"({"status":"expired","message":"This request was already decided"})");
                    return;
                }
                callback(
                    200, "",
                    R"({"id":42,"device_id":"123456789","client_id":"client-a","client_ip":"203.0.113.9","source":"device_code","status":"approved","created_at":"2026-10-17T05:00:00Z","expires_at":"2026-10-17T05:02:00Z","decided_at":"2026-10-17T05:00:10Z"})");
                return;
            }
            Expect(method == "GET",
                   QStringLiteral("approval refresh uses GET"));
            const QUrlQuery query(url);
            if (url.path().endsWith(QStringLiteral("access-approvals"))) {
                listStatus = query.queryItemValue(QStringLiteral("status"));
                listLimit = query.queryItemValue(QStringLiteral("limit"));
                callback(
                    200, "",
                    R"({"items":[{"id":42,"device_id":"123456789","client_id":"client-a","client_ip":"203.0.113.9","source":"device_code","status":"pending","created_at":"2026-10-17T05:00:00Z","expires_at":"2026-10-17T05:02:00Z"}]})");
                return;
            }
            if (url.path().endsWith(
                    QStringLiteral("access-approvals:summary"))) {
                summaryLimit = query.queryItemValue(QStringLiteral("limit"));
                callback(
                    200, "",
                    R"({"since":"2026-10-16T06:00:00Z","total":2,"by_status":{"pending":2,"approved":0,"denied":0,"expired":0,"consumed":0},"denied_sources":[],"denial_cooldown_seconds":300,"denial_cooldown_multiplier":2,"denial_cooldown_max_seconds":3600})");
                return;
            }
            callback(404, "unexpected path", "");
        });

    quickdesk::AuthManager authManager(nullptr);
    quickdesk::AccessApprovalManager manager(&authManager);

    manager.refresh(QStringLiteral("123456789"), QStringLiteral("pending"));
    Expect(WaitUntil([&]() { return !manager.busy(); }, 5000),
           QStringLiteral("approval refresh finishes"));
    Expect(requestCount == 2,
           QStringLiteral("refresh requests list and summary"));
    Expect(listStatus == "pending",
           QStringLiteral("list request carries the status filter"));
    Expect(listLimit == "100",
           QStringLiteral("list request uses the page limit"));
    Expect(summaryLimit == "10",
           QStringLiteral("summary request limits denied sources"));
    Expect(manager.lastError().isEmpty(),
           QStringLiteral("successful refresh clears the error"));
    Expect(manager.approvals().size() == 1,
           QStringLiteral("approval list is populated"));

    const QVariantMap approval = manager.approvals().first().toMap();
    Expect(approval.value(QStringLiteral("id")).toInt() == 42,
           QStringLiteral("approval id is numeric"));
    Expect(approval.value(QStringLiteral("client_ip")).toString()
               == QStringLiteral("203.0.113.9"),
           QStringLiteral("approval source IP is preserved"));
    Expect(approval.value(QStringLiteral("status")).toString()
               == QStringLiteral("pending"),
           QStringLiteral("approval status is preserved"));

    const QVariantMap summary = manager.summary();
    Expect(summary.value(QStringLiteral("total")).toInt() == 2,
           QStringLiteral("summary total is preserved"));
    const QVariantMap byStatus =
        summary.value(QStringLiteral("by_status")).toMap();
    Expect(byStatus.value(QStringLiteral("pending")).toInt() == 2,
           QStringLiteral("summary pending count is preserved"));
    Expect(byStatus.contains(QStringLiteral("consumed")),
           QStringLiteral("summary exposes every lifecycle status"));
    Expect(summary.value(QStringLiteral("denial_cooldown_multiplier")).toInt()
               == 2,
           QStringLiteral("summary cooldown multiplier is preserved"));

    bool decided = false;
    int decidedId = 0;
    QString decidedStatus;
    QObject::connect(
        &manager, &quickdesk::AccessApprovalManager::approvalDecided,
        [&](int approvalId, const QString& status) {
            decided = true;
            decidedId = approvalId;
            decidedStatus = status;
        });

    manager.decide(42, true, QStringLiteral("Verified by the owner"));
    Expect(WaitUntil([&]() { return decided; }, 5000),
           QStringLiteral("approval decision completes"));
    Expect(decisionPath
               == QStringLiteral(
                   "/v1/me/devices/123456789/access-approvals/42/approve"),
           QStringLiteral("approve uses the owner-scoped decision route"));
    Expect(decidedId == 42,
           QStringLiteral("decision signal carries the approval id"));
    Expect(decidedStatus == "approved",
           QStringLiteral("decision signal carries the returned status"));
    Expect(decisionBody.contains(QStringLiteral("Verified by the owner")),
           QStringLiteral("decision request carries the optional reason"));

    conflictDecision = true;
    bool decisionFailed = false;
    int failedApprovalId = 0;
    int failedHttpStatus = 0;
    QString failedCurrentStatus;
    QObject::connect(
        &manager, &quickdesk::AccessApprovalManager::decisionFailed,
        [&](int approvalId, int httpStatus, const QString& currentStatus,
            const QString&) {
            decisionFailed = true;
            failedApprovalId = approvalId;
            failedHttpStatus = httpStatus;
            failedCurrentStatus = currentStatus;
        });

    manager.decide(43, false, QString());
    Expect(WaitUntil([&]() { return decisionFailed; }, 5000),
           QStringLiteral("conflicting decision completes"));
    Expect(failedApprovalId == 43,
           QStringLiteral("failure carries the approval id"));
    Expect(failedHttpStatus == 409,
           QStringLiteral("failure carries HTTP conflict"));
    Expect(failedCurrentStatus == "expired",
           QStringLiteral("failure carries the server's current status"));

    std::cout << "access approval manager refresh tests passed\n";
    return 0;
}
