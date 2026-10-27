// Copyright 2026 QuickDesk Authors

#include "manager/AuthManager.h"
#include "manager/CloudDeviceManager.h"
#include "manager/ServerManager.h"

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

// QD-105, Qt half. The server now refuses a bind that would put the account over
// its device limit, and the host is the client that binds itself automatically
// on startup. A refusal that only reaches the log leaves the machine sitting
// there logged in, online, and missing from the owner's list with no explanation
// anywhere on screen - which is the exact failure this signal exists to prevent.
int main(int argc, char** argv) {
    QCoreApplication app(argc, argv);
    quickdesk::SetAuthTestBaseUrl(QStringLiteral("https://quickdesk.test/"));
    quickdesk::SetAuthTestLoggedIn(true);

    QString quotaDetail;
    int quotaEmissions = 0;
    quickdesk::SetAuthTestRequestHandler(
        [&](const QString& method, const QUrl& url, const QString& body,
            quickdesk::AuthManager::ResponseCallback callback) {
            Q_UNUSED(body);
            if (method == QLatin1String("POST")
                && url.path() == QLatin1String("/v1/me/devices")) {
                // The exact RFC 7807 body the signaling server answers with.
                callback(
                    403, "",
                    R"({"type":"https://quickdesk.io/problems/device-quota-exceeded","title":"Forbidden","status":403,"code":"DEVICE_QUOTA_EXCEEDED","detail":"This account has reached its device limit; unbind a device or ask an administrator to raise the limit"})");
                return;
            }
            callback(404, "unexpected path", "");
        });

    quickdesk::AuthManager authManager(nullptr);
    quickdesk::CloudDeviceManager manager(
        nullptr, &authManager, []() { return QString(); });

    QObject::connect(
        &manager, &quickdesk::CloudDeviceManager::deviceBindQuotaExceeded,
        [&](const QString& deviceId, const QString& detail) {
            Q_UNUSED(deviceId);
            quotaDetail = detail;
            ++quotaEmissions;
        });

    manager.autoBindDevice(QStringLiteral("123456789"));
    Expect(WaitUntil([&]() { return quotaEmissions == 1; }, 5000),
           QStringLiteral("quota refusal is reported to the client"));

    // The detail is what the user reads. An empty one would put a blank toast on
    // screen, which tells the owner nothing about what to do next.
    Expect(!quotaDetail.isEmpty(),
           QStringLiteral("quota detail reaches the signal"));
    Expect(quotaDetail.contains(QStringLiteral("device limit")),
           QStringLiteral("quota detail explains the limit"));

    // A second bind of the same machine must not double-report: the host retries
    // this call on reconnect, and a toast per retry would bury the owner.
    manager.autoBindDevice(QStringLiteral("123456789"));
    Expect(WaitUntil([&]() { return quotaEmissions >= 2; }, 5000),
           QStringLiteral("retry reports again"));
    Expect(quotaEmissions == 2,
           QStringLiteral("each bind attempt reports exactly once"));

    std::cout << "device bind quota signal OK" << std::endl;
    return 0;
}
