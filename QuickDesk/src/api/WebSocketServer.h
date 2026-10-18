// Copyright 2026 QuickDesk Authors
// WebSocket API server for external tool integration (MCP bridge, etc.)

#ifndef QUICKDESK_API_WEBSOCKETSERVER_H
#define QUICKDESK_API_WEBSOCKETSERVER_H

#include "SecurityManager.h"
#include "ApiHandler.h"
#include "core/controlplaneevent.h"

#include <QObject>
#include <QWebSocketServer>
#include <QWebSocket>
#include <QJsonObject>
#include <QMap>
#include <QSet>

namespace quickdesk {

class MainController;

class WebSocketApiServer : public QObject {
    Q_OBJECT

public:
    explicit WebSocketApiServer(MainController* controller,
                                QObject* parent = nullptr);
    ~WebSocketApiServer() override;

    bool start(quint16 port = 9600,
               const QHostAddress& address = QHostAddress::AnyIPv4);
    void stop();
    bool isListening() const;
    quint16 port() const;

    void setAuthToken(const QString& token);

    SecurityManager* security() const { return m_security; }
    ApiHandler* handler() const { return m_handler; }

    void broadcastEvent(const QString& event, const QJsonObject& data);

    int authenticatedClientCount() const;

signals:
    void listeningChanged(bool listening);
    void authenticatedClientCountChanged(int count);

private slots:
    void onNewConnection();
    void onTextMessageReceived(const QString& message);
    void onClientDisconnected();
    void onSessionExpired(const QString& clientId);

private:
    struct ClientInfo {
        SecurityManager::PermissionLevel permission =
            SecurityManager::FullControl;
        QString id;
    };

    bool authenticateClient(QWebSocket* client, const QJsonObject& msg);
    QString clientId(QWebSocket* client) const;
    void sendEvent(QWebSocket* client, const QString& event,
                   const QJsonObject& data);
    void reloadControlPlaneReplay();

    QWebSocketServer* m_server = nullptr;
    MainController* m_controller = nullptr;
    ApiHandler* m_handler = nullptr;
    SecurityManager* m_security = nullptr;
    QSet<QWebSocket*> m_clients;
    QSet<QWebSocket*> m_authenticatedClients;
    QMap<QWebSocket*, ClientInfo> m_clientInfo;
    QMap<QString, QWebSocket*> m_clientIdMap;
    ControlPlaneEventReplayBuffer m_controlPlaneReplay;
    QString m_authToken;
    int m_nextClientId = 1;
};

} // namespace quickdesk

#endif // QUICKDESK_API_WEBSOCKETSERVER_H
