// Copyright 2026 QuickDesk Authors
// Protocol adapter for in-session plain-text chat over qd-skill-bridge.

#ifndef QUICKDESK_MANAGER_SESSIONCHATMANAGER_H
#define QUICKDESK_MANAGER_SESSIONCHATMANAGER_H

#include <QHash>
#include <QJsonObject>
#include <QObject>
#include <QSet>
#include <QString>

#include <functional>

namespace quickdesk {

class SkillHostManager;

class SessionChatManager : public QObject {
    Q_OBJECT

public:
    using ChatSender =
        std::function<bool(const QString& deviceId, const QString& jsonData)>;

    explicit SessionChatManager(ChatSender sender,
                                SkillHostManager* skillHostManager,
                                QObject* parent = nullptr);

    Q_INVOKABLE bool isAvailable(const QString& deviceId) const;
    Q_INVOKABLE int unreadCount(const QString& deviceId) const;
    Q_INVOKABLE bool sendToHost(const QString& deviceId, const QString& text);
    Q_INVOKABLE bool sendToControllers(const QString& text);
    Q_INVOKABLE void markRead(const QString& deviceId);

    // Public seam used by MainController for ClientManager responses. Chat
    // shares the bridge with file-ops, so unrelated responses are ignored.
    void ingestSkillResponse(const QString& deviceId,
                             const QJsonObject& response);

signals:
    void availabilityChanged(const QString& deviceId, bool available);
    void messageAdded(const QString& deviceId,
                      const QString& from,
                      const QString& text,
                      const QString& sentAt,
                      bool outgoing);
    void hostMessageAdded(const QString& from,
                          const QString& text,
                          const QString& sentAt);
    void unreadCountChanged(const QString& deviceId, int count);
    void errorOccurred(const QString& deviceId, const QString& message);

private:
    QString nextRequestId();
    void setUnreadCount(const QString& deviceId, int count);
    bool validateText(const QString& deviceId, const QString& text);

    ChatSender m_sender;
    SkillHostManager* m_skillHostManager = nullptr;
    QHash<QString, QSet<QString>> m_outgoingIds;
    QHash<QString, int> m_unreadCounts;
    QHash<QString, bool> m_availability;
    int m_nextRequestId = 1;
};

}  // namespace quickdesk

#endif  // QUICKDESK_MANAGER_SESSIONCHATMANAGER_H
