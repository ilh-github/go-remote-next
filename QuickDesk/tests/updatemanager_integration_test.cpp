// Copyright 2026 QuickDesk Authors

#include "manager/AuthManager.h"
#include "manager/ServerManager.h"
#include "manager/UpdateManager.h"

#include <QCoreApplication>
#include <QCryptographicHash>
#include <QDir>
#include <QElapsedTimer>
#include <QEventLoop>
#include <QFile>
#include <QJsonDocument>
#include <QJsonObject>
#include <QStandardPaths>
#include <QTcpServer>
#include <QTcpSocket>
#include <QThread>

#include <cstdlib>
#include <functional>
#include <iostream>

namespace quickdesk {
void SetAuthTestBaseUrl(const QString& baseUrl);
}

namespace {

const QByteArray kArtifact = "verified update artifact";
const QString kVersion = QStringLiteral("4.2.1.0");

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

class UpdateHttpServer : public QTcpServer {
public:
    explicit UpdateHttpServer(QObject* parent = nullptr)
        : QTcpServer(parent) {
        connect(this, &QTcpServer::newConnection, this, [this]() {
            while (hasPendingConnections()) {
                handle(nextPendingConnection());
            }
        });
    }

    bool sawApiKey = false;
    int requestCount = 0;
    QString declaredSha256;
    QString artifactBaseUrl;

private:
    void handle(QTcpSocket* socket) {
        auto* buffer = new QByteArray();
        connect(socket, &QTcpSocket::readyRead, socket,
                [this, socket, buffer]() {
                    buffer->append(socket->readAll());
                    if (!buffer->contains("\r\n\r\n")) {
                        return;
                    }

                    ++requestCount;
                    const QByteArray request = *buffer;
                    sawApiKey = sawApiKey ||
                        request.toLower().contains("x-api-key: update-test-key");
                    const QByteArray requestLine =
                        request.left(request.indexOf("\r\n"));
                    const QByteArray path = requestLine.split(' ').value(1);

                    QByteArray body;
                    QByteArray contentType = "application/json";
                    int status = 200;
                    if (path.startsWith("/v1/update:check")) {
                        body = manifest();
                    } else if (path == "/QuickDesk-test.dmg") {
                        body = kArtifact;
                        contentType = "application/octet-stream";
                    } else {
                        status = 404;
                        body = "{\"title\":\"not found\"}";
                    }

                    socket->write("HTTP/1.1 " + QByteArray::number(status) +
                                  (status == 200 ? " OK" : " Not Found") +
                                  "\r\n");
                    socket->write("Content-Type: " + contentType + "\r\n");
                    socket->write("Content-Length: " +
                                  QByteArray::number(body.size()) + "\r\n");
                    socket->write("Connection: close\r\n\r\n");
                    socket->write(body);
                    socket->flush();
                    socket->disconnectFromHost();
                    delete buffer;
                });
        connect(socket, &QTcpSocket::disconnected, socket,
                &QTcpSocket::deleteLater);
    }

    QByteArray manifest() const {
        QJsonObject latest;
        latest["id"] = 7;
        latest["channel"] = "stable";
        latest["platform"] = "macos";
        latest["arch"] = "arm64";
        latest["version"] = kVersion;
        const QString baseUrl =
            artifactBaseUrl.isEmpty()
                ? QStringLiteral("http://127.0.0.1:%1/").arg(serverPort())
                : artifactBaseUrl;
        latest["download_url"] = baseUrl + "QuickDesk-test.dmg";
        latest["sha256"] =
            declaredSha256.isEmpty()
                ? QString::fromLatin1(
                      QCryptographicHash::hash(kArtifact,
                                               QCryptographicHash::Sha256)
                          .toHex())
                : declaredSha256;
        latest["size_bytes"] = kArtifact.size();
        latest["notes"] = "Integration update";
        latest["mandatory"] = false;
        latest["published_at"] = "2026-10-17T00:00:00Z";
        latest["hosted_here"] = true;

        QJsonObject root;
        root["update_available"] = true;
        root["mandatory"] = false;
        root["channel"] = "stable";
        root["platform"] = "macos";
        root["arch"] = "arm64";
        root["current_version"] = "4.2.0.0";
        root["latest"] = latest;
        return QJsonDocument(root).toJson(QJsonDocument::Compact);
    }
};

}  // namespace

int main(int argc, char** argv) {
    QCoreApplication app(argc, argv);
    QCoreApplication::setApplicationVersion("4.2.0.0");

    UpdateHttpServer server;
    Expect(server.listen(QHostAddress::LocalHost, 0),
           QStringLiteral("mock update server listens"));
    const QString baseUrl =
        QStringLiteral("http://127.0.0.1:%1/").arg(server.serverPort());
    quickdesk::SetAuthTestBaseUrl(baseUrl);

    quickdesk::ServerManager serverManager;
    quickdesk::AuthManager authManager(&serverManager);

    QString downloadedPath;
    {
        quickdesk::UpdateManager updateManager(&serverManager, &authManager);

        bool checkFinished = false;
        bool checkAvailable = false;
        QString checkError;
        QObject::connect(
            &updateManager, &quickdesk::UpdateManager::checkFinished,
            [&](bool userInitiated, bool available, const QString& error) {
                if (!userInitiated) {
                    return;
                }
                checkFinished = true;
                checkAvailable = available;
                checkError = error;
            });

        updateManager.checkForUpdates(true);
        Expect(WaitUntil([&]() { return checkFinished; }, 5000),
               QStringLiteral("update check finishes"));
        Expect(checkError.isEmpty(),
               QStringLiteral("update check has no error"));
        Expect(checkAvailable,
               QStringLiteral("update check reports a release"));
        Expect(updateManager.latestVersion() == kVersion,
               QStringLiteral("latest version is parsed"));
        Expect(updateManager.state() == "available",
               QStringLiteral("manager enters available state"));

        bool downloadFinished = false;
        bool downloadSucceeded = false;
        QString downloadError;
        QObject::connect(
            &updateManager, &quickdesk::UpdateManager::downloadFinished,
            [&](bool success, const QString& error) {
                downloadFinished = true;
                downloadSucceeded = success;
                downloadError = error;
            });

        updateManager.downloadUpdate();
        Expect(WaitUntil([&]() { return downloadFinished; }, 5000),
               QStringLiteral("update download finishes"));
        Expect(downloadError.isEmpty(),
               QStringLiteral("download has no error"));
        Expect(downloadSucceeded, QStringLiteral("download succeeds"));
        Expect(updateManager.state() == "ready",
               QStringLiteral("manager enters ready state"));
        Expect(server.sawApiKey,
               QStringLiteral(
                   "same-origin artifact request carries the API key"));

        downloadedPath =
            QDir(QStandardPaths::writableLocation(
                     QStandardPaths::TempLocation))
                .filePath(QStringLiteral("QuickDesk/updates/%1/%2")
                              .arg(kVersion,
                                   updateManager.installerFileName()));
        QFile downloaded(downloadedPath);
        Expect(downloaded.open(QIODevice::ReadOnly),
               QStringLiteral("downloaded installer can be read"));
        Expect(downloaded.readAll() == kArtifact,
               QStringLiteral("downloaded bytes match the artifact"));
        downloaded.close();
        QFile::remove(downloadedPath);
    }

    UpdateHttpServer artifactServer;
    Expect(artifactServer.listen(QHostAddress::LocalHost, 0),
           QStringLiteral("cross-origin artifact server listens"));
    server.artifactBaseUrl =
        QStringLiteral("http://127.0.0.1:%1/").arg(artifactServer.serverPort());
    {
        quickdesk::UpdateManager updateManager(&serverManager, &authManager);
        bool checkFinished = false;
        QObject::connect(
            &updateManager, &quickdesk::UpdateManager::checkFinished,
            [&](bool userInitiated, bool, const QString&) {
                if (userInitiated) {
                    checkFinished = true;
                }
            });
        updateManager.checkForUpdates(true);
        Expect(WaitUntil([&]() { return checkFinished; }, 5000),
               QStringLiteral("cross-origin update check finishes"));

        bool downloadFinished = false;
        bool downloadSucceeded = false;
        QString downloadError;
        QObject::connect(
            &updateManager, &quickdesk::UpdateManager::downloadFinished,
            [&](bool success, const QString& error) {
                downloadFinished = true;
                downloadSucceeded = success;
                downloadError = error;
            });
        updateManager.downloadUpdate();
        Expect(WaitUntil([&]() { return downloadFinished; }, 5000),
               QStringLiteral("cross-origin update download finishes"));
        Expect(downloadError.isEmpty(),
               QStringLiteral("cross-origin download has no error"));
        Expect(downloadSucceeded,
               QStringLiteral("cross-origin artifact download succeeds"));
        Expect(artifactServer.requestCount > 0,
               QStringLiteral("cross-origin artifact server is used"));
        Expect(!artifactServer.sawApiKey,
               QStringLiteral(
                   "cross-origin artifact request omits the API key"));
        downloadedPath =
            QDir(QStandardPaths::writableLocation(
                     QStandardPaths::TempLocation))
                .filePath(QStringLiteral("QuickDesk/updates/%1/%2")
                              .arg(kVersion,
                                   updateManager.installerFileName()));
        QFile::remove(downloadedPath);
    }

    server.artifactBaseUrl.clear();
    server.declaredSha256 = QString(64, QLatin1Char('0'));
    {
        quickdesk::UpdateManager updateManager(&serverManager, &authManager);
        bool checkFinished = false;
        QObject::connect(
            &updateManager, &quickdesk::UpdateManager::checkFinished,
            [&](bool userInitiated, bool, const QString&) {
                if (userInitiated) {
                    checkFinished = true;
                }
            });
        updateManager.checkForUpdates(true);
        Expect(WaitUntil([&]() { return checkFinished; }, 5000),
               QStringLiteral("mismatched update check finishes"));

        bool downloadFinished = false;
        bool downloadSucceeded = true;
        QString downloadError;
        QObject::connect(
            &updateManager, &quickdesk::UpdateManager::downloadFinished,
            [&](bool success, const QString& error) {
                downloadFinished = true;
                downloadSucceeded = success;
                downloadError = error;
            });
        updateManager.downloadUpdate();
        Expect(WaitUntil([&]() { return downloadFinished; }, 5000),
               QStringLiteral("mismatched update download finishes"));
        Expect(!downloadSucceeded,
               QStringLiteral("mismatched installer is rejected"));
        Expect(downloadError.contains(QStringLiteral("verification")),
               QStringLiteral("hash mismatch reports verification failure"));
        Expect(updateManager.state() == "failed",
               QStringLiteral("hash mismatch enters failed state"));
        downloadedPath =
            QDir(QStandardPaths::writableLocation(
                     QStandardPaths::TempLocation))
                .filePath(QStringLiteral("QuickDesk/updates/%1/%2")
                              .arg(kVersion,
                                   updateManager.installerFileName()));
        Expect(!QFile::exists(downloadedPath + QStringLiteral(".part")),
               QStringLiteral("failed download leaves no partial file"));
        Expect(!QFile::exists(downloadedPath),
               QStringLiteral("failed download leaves no installer"));
    }

    std::cout << "update manager integration tests passed\n";
    return 0;
}
