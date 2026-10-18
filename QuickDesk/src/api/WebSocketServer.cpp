// Copyright 2026 QuickDesk Authors

#include "WebSocketServer.h"
#include "ApiHandler.h"
#include "controller/MainController.h"
#include "infra/log/log.h"
#include "manager/AuthManager.h"
#include <QDir>
#include <QJsonDocument>
#include <QJsonObject>
#include <QPointer>
#include <QThreadPool>
#include "infra/env/applicationcontext.h"

namespace quickdesk {

namespace {
constexpr int kMaxControlPlaneReplayCapacity = 10000;
}

WebSocketApiServer::WebSocketApiServer(MainController* controller,
                                       QObject* parent)
    : QObject(parent)
    , m_controller(controller)
    , m_handler(new ApiHandler(controller, this))
    , m_security(new SecurityManager(this)) {
    connect(m_security, &SecurityManager::sessionExpired,
            this, &WebSocketApiServer::onSessionExpired);

    auto logDir = infra::ApplicationContext::instance().logPath();
    m_security->initAuditLog(logDir);

    int replayCapacity = kDefaultControlPlaneReplayCapacity;
    const QByteArray configuredCapacity =
        qgetenv("QUICKDESK_CONTROL_PLANE_REPLAY_CAPACITY");
    if (!configuredCapacity.isEmpty()) {
        bool ok = false;
        const int requested = configuredCapacity.toInt(&ok);
        if (ok) {
            replayCapacity =
                qBound(1, requested, kMaxControlPlaneReplayCapacity);
        } else {
            LOG_WARN("Ignoring invalid QUICKDESK_CONTROL_PLANE_REPLAY_CAPACITY={}",
                     configuredCapacity.toStdString());
        }
    }
    if (!m_controlPlaneReplay.configureCapacity(replayCapacity)) {
        LOG_WARN("Failed to persist configured control-plane replay capacity");
    }

    if (m_controller && m_controller->authManager()) {
        connect(m_controller->authManager(), &AuthManager::loginStateChanged,
                this, &WebSocketApiServer::reloadControlPlaneReplay);
    }
    reloadControlPlaneReplay();
}

WebSocketApiServer::~WebSocketApiServer() {
    stop();
}

bool WebSocketApiServer::start(quint16 port, const QHostAddress& address) {
    reloadControlPlaneReplay();
    if (m_server) {
        return m_server->isListening();
    }

    m_server = new QWebSocketServer(
        QStringLiteral("QuickDesk API"),
        QWebSocketServer::NonSecureMode, this);

    if (!m_server->listen(address, port)) {
        LOG_ERROR("WebSocket API server failed to listen on port {}: {}",
                  port, m_server->errorString().toStdString());
        delete m_server;
        m_server = nullptr;
        return false;
    }

    connect(m_server, &QWebSocketServer::newConnection,
            this, &WebSocketApiServer::onNewConnection);

    LOG_INFO("WebSocket API server listening on {}:{}",
             address.toString().toStdString(), m_server->serverPort());
    emit listeningChanged(true);
    return true;
}

void WebSocketApiServer::stop() {
    if (!m_server) {
        return;
    }

    for (auto* client : m_clients) {
        client->close();
    }
    m_clients.clear();
    m_authenticatedClients.clear();
    m_clientInfo.clear();
    m_clientIdMap.clear();
    m_controlPlaneReplay.clear();

    m_server->close();
    delete m_server;
    m_server = nullptr;
    LOG_INFO("WebSocket API server stopped");
    emit listeningChanged(false);
}

bool WebSocketApiServer::isListening() const {
    return m_server && m_server->isListening();
}

quint16 WebSocketApiServer::port() const {
    return m_server ? m_server->serverPort() : 0;
}

void WebSocketApiServer::setAuthToken(const QString& token) {
    m_authToken = token;
    m_security->setFullAccessToken(token);
}

int WebSocketApiServer::authenticatedClientCount() const {
    return m_authenticatedClients.size();
}

void WebSocketApiServer::broadcastEvent(const QString& event,
                                        const QJsonObject& data) {
    if (event == QLatin1String("controlPlaneEvent")) {
        if (!m_controlPlaneReplay.append(data)) {
            LOG_WARN("Failed to persist control-plane event replay buffer");
        }
    }
    auto& targets = m_authToken.isEmpty() ? m_clients : m_authenticatedClients;
    for (auto* client : targets) {
        sendEvent(client, event, data);
    }
}

void WebSocketApiServer::sendEvent(QWebSocket* client, const QString& event,
                                   const QJsonObject& data) {
    QJsonObject msg;
    msg["event"] = event;
    msg["data"] = data;

    auto text = QString::fromUtf8(
        QJsonDocument(msg).toJson(QJsonDocument::Compact));

    client->sendTextMessage(text);
}

void WebSocketApiServer::reloadControlPlaneReplay() {
    const qint64 userId =
        m_controller && m_controller->authManager()
            ? static_cast<qint64>(m_controller->authManager()->userId())
            : 0;
    const QString dataPath =
        infra::ApplicationContext::instance().localDataPath();
    const QString storagePath =
        userId > 0 && !dataPath.isEmpty()
            ? QDir(dataPath).filePath(
                  QStringLiteral("control-plane-event-replay.json"))
            : QString();
    if (!m_controlPlaneReplay.configureStorage(storagePath, userId)) {
        LOG_WARN("Failed to load control-plane event replay buffer from {}",
                 storagePath.toStdString());
    }
}

QString WebSocketApiServer::clientId(QWebSocket* client) const {
    auto it = m_clientInfo.constFind(client);
    if (it != m_clientInfo.constEnd()) {
        return it->id;
    }
    return client->peerAddress().toString();
}

void WebSocketApiServer::onNewConnection() {
    while (m_server->hasPendingConnections()) {
        auto* client = m_server->nextPendingConnection();
        if (!client) {
            continue;
        }

        connect(client, &QWebSocket::textMessageReceived,
                this, &WebSocketApiServer::onTextMessageReceived);
        connect(client, &QWebSocket::disconnected,
                this, &WebSocketApiServer::onClientDisconnected);

        m_clients.insert(client);

        ClientInfo info;
        info.id = QString("client_%1").arg(m_nextClientId++);
        info.permission = SecurityManager::FullControl;
        m_clientInfo[client] = info;
        m_clientIdMap[info.id] = client;

        if (m_authToken.isEmpty()) {
            m_authenticatedClients.insert(client);
            emit authenticatedClientCountChanged(m_authenticatedClients.size());
            for (const QJsonObject& event : m_controlPlaneReplay.eventsAfter(0)) {
                sendEvent(client, QStringLiteral("controlPlaneEvent"), event);
            }
        }

        m_security->recordActivity(info.id);

        LOG_INFO("WebSocket API client connected: {} from {}",
                 info.id.toStdString(),
                 client->peerAddress().toString().toStdString());
    }
}

void WebSocketApiServer::onTextMessageReceived(const QString& message) {
    auto* client = qobject_cast<QWebSocket*>(sender());
    if (!client) {
        return;
    }

    auto cid = clientId(client);
    m_security->recordActivity(cid);

    auto doc = QJsonDocument::fromJson(message.toUtf8());
    if (!doc.isObject()) {
        QJsonObject err;
        err["error"] = QJsonObject{
            {"code", -32700},
            {"message", "Parse error"}
        };
        client->sendTextMessage(
            QString::fromUtf8(QJsonDocument(err).toJson(QJsonDocument::Compact)));
        return;
    }

    auto request = doc.object();
    auto method = request["method"].toString();
    auto params = request["params"].toObject();

    if (!m_authToken.isEmpty() && !m_authenticatedClients.contains(client)) {
        if (!authenticateClient(client, request)) {
            return;
        }
        if (method == "auth") {
            return;
        }
    }

    auto permission = m_clientInfo.value(client).permission;

    if (!m_security->checkRateLimit(cid)) {
        m_security->logAudit(cid, method, params, false, "rate_limit");
        QJsonObject err;
        err["error"] = QJsonObject{
            {"code", 429},
            {"message", "Rate limit exceeded. Please slow down."}
        };
        if (request.contains("id")) {
            err["id"] = request["id"];
        }
        client->sendTextMessage(
            QString::fromUtf8(QJsonDocument(err).toJson(QJsonDocument::Compact)));
        return;
    }

    if (!m_security->isMethodAllowedForPermission(method, permission)) {
        m_security->logAudit(cid, method, params, false, "permission_denied");
        QJsonObject err;
        err["error"] = QJsonObject{
            {"code", 403},
            {"message",
             QString("Permission denied. Method '%1' requires full control access.")
                 .arg(method)}
        };
        if (request.contains("id")) {
            err["id"] = request["id"];
        }
        client->sendTextMessage(
            QString::fromUtf8(QJsonDocument(err).toJson(QJsonDocument::Compact)));
        return;
    }

    if (method == "connectToHost") {
        auto deviceId = params["deviceId"].toString();
        if (!m_security->isDeviceAllowed(deviceId)) {
            m_security->logAudit(cid, method, params, false, "device_not_allowed");
            QJsonObject err;
            err["error"] = QJsonObject{
                {"code", 403},
                {"message",
                 QString("Device '%1' is not in the allowed device list.")
                     .arg(deviceId)}
            };
            if (request.contains("id")) {
                err["id"] = request["id"];
            }
            client->sendTextMessage(
                QString::fromUtf8(
                    QJsonDocument(err).toJson(QJsonDocument::Compact)));
            return;
        }
    }

    if (m_security->isDangerousOperation(method, params)) {
        m_security->logAudit(cid, method, params, false, "dangerous_operation");
        emit m_security->dangerousOperationBlocked(cid, method, params);
        QJsonObject err;
        err["error"] = QJsonObject{
            {"code", 403},
            {"message",
             "This operation has been blocked for safety. Dangerous operations "
             "(shutdown, format, rm -rf, Alt+F4, Ctrl+Alt+Delete, disconnect all) "
             "are not allowed via the API."}
        };
        if (request.contains("id")) {
            err["id"] = request["id"];
        }
        client->sendTextMessage(
            QString::fromUtf8(QJsonDocument(err).toJson(QJsonDocument::Compact)));
        return;
    }

    m_security->logAudit(cid, method, params, true);

    // OCR-heavy methods block for 100–300 ms (single recognize) or up to 5 s
    // (polling loops in waitForText / verifyActionResult).  Running them on the
    // UI thread would stall the event loop and delay all other WebSocket
    // messages and screenChanged broadcasts.  Dispatch to the global thread
    // pool; the underlying services are thread-safe:
    //   - OcrEngine::recognize()        — mutex-protected singleton
    //   - OcrCache::get / put           — mutex-protected
    //   - SharedMemoryManager::readVideoFrame — QSharedMemory::lock() guard
    //   - getActiveWindowTitle()        — Win32 API, callable from any thread
    static const QSet<QString> kAsyncMethods{
        "getUiState", "getScreenText", "findElement", "clickText",
        "assertTextPresent", "assertScreenState", "screenDiffSummary",
        "waitForText", "verifyActionResult",
        "skillExec", "skillListTools",
        "requestConfirmation"
    };

    if (kAsyncMethods.contains(method)) {
        QPointer<WebSocketApiServer> self(this);
        ApiHandler* handlerPtr = m_handler;   // owned by this, stable for app lifetime
        QWebSocket* clientPtr  = client;
        QJsonObject reqCopy    = request;

        QThreadPool::globalInstance()->start([self, handlerPtr, clientPtr, reqCopy]() mutable {
            auto response = handlerPtr->handleRequest(reqCopy);
            // self.data() check: QPointer's null test is lock-free and safe to call
            // from non-UI threads.  If the server was destroyed, skip posting.
            // If it's alive now, invokeMethod queues the lambda; the inner check
            // (on the UI thread) handles any intervening destruction.
            auto* srv = self.data();
            if (!srv) return;
            QMetaObject::invokeMethod(srv, [self, clientPtr, reqCopy,
                                            response = std::move(response)]() mutable {
                if (!self) return;
                // Verify client is still connected before sending
                if (!self->m_clients.contains(clientPtr) &&
                    !self->m_authenticatedClients.contains(clientPtr)) {
                    return;
                }
                if (reqCopy.contains("id")) response["id"] = reqCopy["id"];
                clientPtr->sendTextMessage(
                    QString::fromUtf8(QJsonDocument(response).toJson(QJsonDocument::Compact)));
            }, Qt::QueuedConnection);
        });
        return;
    }

    auto response = m_handler->handleRequest(request);
    if (request.contains("id")) {
        response["id"] = request["id"];
    }

    client->sendTextMessage(
        QString::fromUtf8(QJsonDocument(response).toJson(QJsonDocument::Compact)));
}

void WebSocketApiServer::onClientDisconnected() {
    auto* client = qobject_cast<QWebSocket*>(sender());
    if (!client) {
        return;
    }

    auto cid = clientId(client);
    m_security->removeClient(cid);
    m_clients.remove(client);
    bool wasAuthenticated = m_authenticatedClients.remove(client);
    m_clientIdMap.remove(cid);
    m_clientInfo.remove(client);
    client->deleteLater();
    LOG_INFO("WebSocket API client disconnected: {}", cid.toStdString());
    if (wasAuthenticated) {
        emit authenticatedClientCountChanged(m_authenticatedClients.size());
    }
}

void WebSocketApiServer::onSessionExpired(const QString& expiredClientId) {
    auto it = m_clientIdMap.find(expiredClientId);
    if (it == m_clientIdMap.end()) {
        return;
    }

    auto* client = it.value();
    m_security->logAudit(expiredClientId, "SESSION_TIMEOUT", {}, false,
                         "session_timeout");

    QJsonObject msg;
    msg["event"] = "sessionTimeout";
    msg["data"] = QJsonObject{
        {"message", "Session timed out due to inactivity."}
    };
    client->sendTextMessage(
        QString::fromUtf8(QJsonDocument(msg).toJson(QJsonDocument::Compact)));
    client->close();
}

bool WebSocketApiServer::authenticateClient(QWebSocket* client,
                                            const QJsonObject& msg) {
    auto method = msg["method"].toString();
    if (method != "auth") {
        QJsonObject err;
        err["error"] = QJsonObject{
            {"code", 401},
            {"message",
             "Authentication required. Send "
             "{\"method\":\"auth\",\"params\":{\"token\":\"...\"}} first."}
        };
        if (msg.contains("id")) {
            err["id"] = msg["id"];
        }
        client->sendTextMessage(
            QString::fromUtf8(QJsonDocument(err).toJson(QJsonDocument::Compact)));
        return false;
    }

    auto token = msg["params"].toObject()["token"].toString();
    auto permission = m_security->authenticateToken(token);

    if (permission == SecurityManager::NoAccess) {
        auto cid = clientId(client);
        m_security->logAudit(cid, "auth", {}, false, "invalid_token");
        emit m_security->anomalyDetected(cid, "Failed authentication attempt");

        QJsonObject err;
        err["error"] = QJsonObject{
            {"code", 401},
            {"message", "Invalid token"}
        };
        if (msg.contains("id")) {
            err["id"] = msg["id"];
        }
        client->sendTextMessage(
            QString::fromUtf8(QJsonDocument(err).toJson(QJsonDocument::Compact)));
        client->close();
        return false;
    }

    m_authenticatedClients.insert(client);
    m_clientInfo[client].permission = permission;
    emit authenticatedClientCountChanged(m_authenticatedClients.size());

    auto cid = clientId(client);
    m_security->logAudit(cid, "auth", {}, true);

    QJsonObject resp;
    resp["result"] = QJsonObject{
        {"authenticated", true},
        {"permission", permission == SecurityManager::FullControl
                           ? "full_control"
                           : "read_only"},
    };
    if (msg.contains("id")) {
        resp["id"] = msg["id"];
    }
    client->sendTextMessage(
        QString::fromUtf8(QJsonDocument(resp).toJson(QJsonDocument::Compact)));

    const qint64 sinceServerRev =
        controlPlaneReplaySinceRev(msg.value(QStringLiteral("params")).toObject());
    for (const QJsonObject& event :
         m_controlPlaneReplay.eventsAfter(sinceServerRev)) {
        sendEvent(client, QStringLiteral("controlPlaneEvent"), event);
    }
    return true;
}

} // namespace quickdesk
