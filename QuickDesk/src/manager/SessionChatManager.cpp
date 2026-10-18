// Copyright 2026 QuickDesk Authors

#include "SessionChatManager.h"

#include "SkillHostManager.h"
#include "infra/log/log.h"

#include <QDateTime>
#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonValue>

namespace quickdesk {

namespace {

constexpr int kMaxChatTextLength = 4000;

QString utcTimestamp()
{
    return QDateTime::currentDateTimeUtc().toString(Qt::ISODateWithMs);
}

QString capabilityName(const QJsonValue& value)
{
    if (value.isString()) {
        return value.toString();
    }
    if (!value.isObject()) {
        return {};
    }
    const QJsonObject object = value.toObject();
    return object.value(QStringLiteral("name"))
        .toString(object.value(QStringLiteral("tool")).toString());
}

}  // namespace

SessionChatManager::SessionChatManager(ChatSender sender,
                                       SkillHostManager* skillHostManager,
                                       QObject* parent)
    : QObject(parent)
    , m_sender(std::move(sender))
    , m_skillHostManager(skillHostManager)
{
    if (m_skillHostManager) {
        connect(m_skillHostManager,
                &SkillHostManager::chatMessageReceived,
                this,
                [this](const QString& from,
                       const QString& /*id*/,
                       const QString& text,
                       const QString& sentAt) {
                    emit hostMessageAdded(from, text, sentAt);
                });
    }
}

bool SessionChatManager::isAvailable(const QString& deviceId) const
{
    return m_availability.value(deviceId, false);
}

int SessionChatManager::unreadCount(const QString& deviceId) const
{
    return m_unreadCounts.value(deviceId, 0);
}

bool SessionChatManager::sendToHost(const QString& deviceId,
                                    const QString& text)
{
    if (!validateText(deviceId, text)) {
        return false;
    }
    if (!m_sender) {
        emit errorOccurred(deviceId, tr("No active connection to the host"));
        return false;
    }

    QJsonObject payload;
    const QString id = nextRequestId();
    payload[QStringLiteral("id")] = id;
    payload[QStringLiteral("type")] = QStringLiteral("chatSend");
    payload[QStringLiteral("text")] = text.trimmed();
    payload[QStringLiteral("sentAt")] = utcTimestamp();

    m_outgoingIds[deviceId].insert(id);
    const QString json =
        QString::fromUtf8(QJsonDocument(payload).toJson(QJsonDocument::Compact));
    if (!m_sender(deviceId, json)) {
        m_outgoingIds[deviceId].remove(id);
        emit errorOccurred(deviceId,
                           tr("Failed to send the chat message"));
        return false;
    }
    return true;
}

bool SessionChatManager::sendToControllers(const QString& text)
{
    if (!validateText(QString(), text)) {
        return false;
    }
    if (!m_skillHostManager) {
        emit errorOccurred(QString(),
                           tr("Session chat is not available on this host"));
        return false;
    }
    return m_skillHostManager->sendChatMessage(text.trimmed());
}

void SessionChatManager::markRead(const QString& deviceId)
{
    setUnreadCount(deviceId, 0);
}

void SessionChatManager::ingestSkillResponse(const QString& deviceId,
                                             const QJsonObject& response)
{
    const QString type = response.value(QStringLiteral("type")).toString();
    if (type == QStringLiteral("capabilitiesReady")
        || type == QStringLiteral("capabilitiesChanged")) {
        bool available = false;
        const QJsonArray tools =
            response.value(QStringLiteral("tools")).toArray();
        for (const QJsonValue& tool : tools) {
            if (capabilityName(tool) == QStringLiteral("session_chat")) {
                available = true;
                break;
            }
        }
        if (m_availability.value(deviceId, false) == available) {
            return;
        }
        m_availability[deviceId] = available;
        emit availabilityChanged(deviceId, available);
        return;
    }

    if (type == QStringLiteral("chatAck")) {
        if (!response.value(QStringLiteral("accepted")).toBool()) {
            const QString id = response.value(QStringLiteral("id")).toString();
            m_outgoingIds[deviceId].remove(id);
            emit errorOccurred(
                deviceId,
                response.value(QStringLiteral("error"))
                    .toString(tr("Host rejected the chat message")));
        }
        return;
    }

    if (type != QStringLiteral("chatMessage")) {
        return;
    }

    const QString id = response.value(QStringLiteral("id")).toString();
    const QString from = response.value(QStringLiteral("from")).toString();
    const QString text = response.value(QStringLiteral("text")).toString();
    if (text.isEmpty()) {
        return;
    }

    const bool outgoing =
        from == QStringLiteral("controller")
        && m_outgoingIds[deviceId].remove(id);
    emit messageAdded(deviceId,
                      from,
                      text,
                      response.value(QStringLiteral("sentAt")).toString(),
                      outgoing);
    if (!outgoing) {
        setUnreadCount(deviceId, unreadCount(deviceId) + 1);
    }
}

QString SessionChatManager::nextRequestId()
{
    return QStringLiteral("qt-chat-%1").arg(m_nextRequestId++);
}

void SessionChatManager::setUnreadCount(const QString& deviceId, int count)
{
    if (unreadCount(deviceId) == count) {
        return;
    }
    m_unreadCounts[deviceId] = count;
    emit unreadCountChanged(deviceId, count);
}

bool SessionChatManager::validateText(const QString& deviceId,
                                      const QString& text)
{
    if (text.trimmed().isEmpty()) {
        emit errorOccurred(deviceId, tr("Chat message must not be empty"));
        return false;
    }
    if (text.size() > kMaxChatTextLength) {
        emit errorOccurred(
            deviceId,
            tr("Chat message exceeds %1 characters").arg(kMaxChatTextLength));
        return false;
    }
    return true;
}

}  // namespace quickdesk
