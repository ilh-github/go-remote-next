// Copyright 2026 QuickDesk Authors

#include "manager/AuthManager.h"
#include "manager/CloudDeviceManager.h"
#include "manager/ServerManager.h"

#include <QCoreApplication>
#include <QElapsedTimer>
#include <QEventLoop>
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

    quickdesk::ServerManager serverManager;
    quickdesk::AuthManager authManager(&serverManager);
    quickdesk::CloudDeviceManager manager(&serverManager, &authManager, nullptr);

    int requestCount = 0;
    QString requestedPath;
    quickdesk::SetAuthTestRequestHandler(
        [&](const QString& method, const QUrl& url, const QString&,
            quickdesk::AuthManager::ResponseCallback callback) {
            ++requestCount;
            requestedPath = url.path();
            Expect(method == "GET", QStringLiteral("access code uses GET"));
            callback(
                200, "",
                R"({"device_id":"123456789","access_code":"654321","online":true,"logged_in":true})");
        });

    bool ready = false;
    QString readyDeviceId;
    QString readyAccessCode;
    QObject::connect(
        &manager, &quickdesk::CloudDeviceManager::deviceAccessCodeReady,
        [&](const QString& deviceId, const QString& accessCode) {
            ready = true;
            readyDeviceId = deviceId;
            readyAccessCode = accessCode;
        });

    manager.requestDeviceAccessCode(QStringLiteral("123456789"));
    manager.requestDeviceAccessCode(QStringLiteral("123456789"));
    Expect(WaitUntil([&]() { return ready; }, 5000),
           QStringLiteral("access code request completes"));
    Expect(requestCount == 1,
           QStringLiteral("concurrent requests are coalesced"));
    Expect(requestedPath == "/v1/me/devices/123456789",
           QStringLiteral("single-device endpoint is requested"));
    Expect(readyDeviceId == "123456789",
           QStringLiteral("ready signal carries the device"));
    Expect(readyAccessCode == "654321",
           QStringLiteral("ready signal carries the access code"));
    Expect(manager.getDeviceAccessCode(QStringLiteral("123456789")) ==
               "654321",
           QStringLiteral("manager caches the refreshed access code"));

    bool failed = false;
    QString failedDeviceId;
    QString failedError;
    QObject::connect(
        &manager, &quickdesk::CloudDeviceManager::deviceAccessCodeFailed,
        [&](const QString& deviceId, const QString& error) {
            failed = true;
            failedDeviceId = deviceId;
            failedError = error;
        });
    quickdesk::SetAuthTestRequestHandler(
        [&](const QString&, const QUrl&, const QString&,
            quickdesk::AuthManager::ResponseCallback callback) {
            callback(500, "server unavailable", "");
        });
    manager.requestDeviceAccessCode(QStringLiteral("987654321"));
    Expect(WaitUntil([&]() { return failed; }, 5000),
           QStringLiteral("access code failure completes"));
    Expect(failedDeviceId == "987654321",
           QStringLiteral("failure signal carries the device"));
    Expect(failedError.contains(QStringLiteral("server unavailable")),
           QStringLiteral("failure signal carries the server error"));

    std::cout << "cloud device manager integration tests passed\n";
    return 0;
}
