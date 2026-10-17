// Copyright 2026 QuickDesk Authors

#include "manager/AuthManager.h"
#include "manager/CloudDeviceManager.h"
#include "manager/ServerManager.h"

#include <QCoreApplication>
#include <QElapsedTimer>
#include <QEventLoop>
#include <QTcpServer>
#include <QTcpSocket>
#include <QWebSocket>
#include <QWebSocketServer>
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

QByteArray HttpRequestBytes(QTcpSocket* socket, QByteArray& buffer) {
    buffer += socket->readAll();
    const qsizetype headerEnd = buffer.indexOf("\r\n\r\n");
    if (headerEnd < 0) return {};

    qsizetype contentLength = 0;
    for (const QByteArray& line : buffer.left(headerEnd).split('\n')) {
        const QByteArray trimmed = line.trimmed();
        if (trimmed.toLower().startsWith("content-length:")) {
            contentLength = trimmed.mid(trimmed.indexOf(':') + 1).trimmed().toLongLong();
        }
    }
    const qsizetype requestSize = headerEnd + 4 + contentLength;
    if (buffer.size() < requestSize) return {};

    const QByteArray request = buffer.left(requestSize);
    buffer.remove(0, requestSize);
    return request;
}

}  // namespace

int main(int argc, char** argv) {
    QCoreApplication app(argc, argv);
    quickdesk::SetAuthTestBaseUrl(QStringLiteral("https://quickdesk.test/"));
    quickdesk::SetAuthTestLoggedIn(true);

    quickdesk::ServerManager serverManager;
    quickdesk::AuthManager authManager(&serverManager);
    QString testDeviceSecret = QStringLiteral("test-device-secret");
    quickdesk::CloudDeviceManager manager(
        &serverManager, &authManager,
        [&]() { return testDeviceSecret; });

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

    QTcpServer accessCodeServer;
    Expect(accessCodeServer.listen(QHostAddress::LocalHost, 0),
           QStringLiteral("access code retry server listens"));
    int accessCodeRequestCount = 0;
    QList<QByteArray> accessCodeRequests;
    QObject::connect(
        &accessCodeServer, &QTcpServer::newConnection,
        [&]() {
            while (QTcpSocket* socket = accessCodeServer.nextPendingConnection()) {
                auto buffer = std::make_shared<QByteArray>();
                QObject::connect(
                    socket, &QTcpSocket::readyRead,
                    [&, socket, buffer]() {
                        const QByteArray request = HttpRequestBytes(socket, *buffer);
                        if (request.isEmpty()) return;

                        ++accessCodeRequestCount;
                        accessCodeRequests.append(request);
                        const bool firstAttempt = accessCodeRequestCount == 1;
                        const QByteArray status = firstAttempt
                            ? QByteArrayLiteral("503 Service Unavailable")
                            : QByteArrayLiteral("200 OK");
                        const QByteArray body = firstAttempt
                            ? QByteArrayLiteral(R"({"code":"TEMPORARY"})")
                            : QByteArrayLiteral(R"({"status":"ok"})");
                        socket->write(
                            "HTTP/1.1 " + status + "\r\n"
                            "Content-Type: application/json\r\n"
                            "Content-Length: " + QByteArray::number(body.size()) + "\r\n"
                            "Connection: close\r\n\r\n" + body);
                        socket->disconnectFromHost();
                    });
                QObject::connect(socket, &QTcpSocket::disconnected,
                                 socket, &QObject::deleteLater);
            }
        });

    serverManager.setServerUrl(
        QStringLiteral("ws://127.0.0.1:%1").arg(accessCodeServer.serverPort()));
    manager.syncAccessCode(QStringLiteral("256983850"), QStringLiteral("079898"));
    Expect(WaitUntil([&]() { return accessCodeRequestCount >= 2; }, 5000),
           QStringLiteral("failed access code sync is retried"));
    Expect(accessCodeRequests.size() == 2,
           QStringLiteral("successful retry stops further requests"));
    for (const QByteArray& request : accessCodeRequests) {
        Expect(request.contains("\"access_code\":\"079898\""),
               QStringLiteral("every retry uploads the current access code"));
        Expect(request.toLower().contains(
                   "authorization: bearer test-device-secret"),
               QStringLiteral("retry keeps device-secret authentication"));
    }
    manager.syncAccessCode(QStringLiteral("256983850"), QStringLiteral("079898"));
    WaitUntil([&]() { return accessCodeRequestCount > 2; }, 150);
    Expect(accessCodeRequestCount == 2,
           QStringLiteral("successful sync remains de-duplicated"));
    accessCodeServer.close();

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

    QWebSocketServer realtimeServer(QStringLiteral("QuickDesk test realtime"),
                                    QWebSocketServer::NonSecureMode);
    Expect(realtimeServer.listen(QHostAddress::LocalHost, 0),
           QStringLiteral("realtime test server listens"));
    serverManager.setServerUrl(
        QStringLiteral("ws://127.0.0.1:%1").arg(realtimeServer.serverPort()));

    QWebSocket* realtimeSocket = nullptr;
    QObject::connect(
        &realtimeServer, &QWebSocketServer::newConnection,
        [&]() {
            realtimeSocket = realtimeServer.nextPendingConnection();
            Expect(realtimeSocket != nullptr,
                   QStringLiteral("realtime client connects"));
            QObject::connect(
                realtimeSocket, &QWebSocket::textMessageReceived,
                [&, realtimeSocket](const QString&) {
                    realtimeSocket->sendTextMessage(
                        R"({"type":"auth_ok","server_rev":1,"data":{"family_id":"family-test"}})");
                    realtimeSocket->sendTextMessage(
                        R"({"type":"snapshot","server_rev":2,"data":{"devices":[],"favorites":[]}})");
                    realtimeSocket->sendTextMessage(
                        R"({"type":"bootstrap_done","server_rev":2})");
                });
        });

    manager.startSync();
    Expect(WaitUntil(
               [&]() {
                   return manager.property("realtimeConnected").toBool();
               },
               5000),
           QStringLiteral("manager exposes connected realtime state"));

    manager.stopSync();
    Expect(!manager.property("realtimeConnected").toBool(),
           QStringLiteral("stopping sync clears connected realtime state"));
    realtimeServer.close();

    std::cout << "cloud device manager integration tests passed\n";
    return 0;
}
