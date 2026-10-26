// Copyright 2026 QuickDesk Authors

#include "controlplaneevent.h"

#include <QDir>
#include <QFile>
#include <QFileInfo>
#include <QJsonArray>
#include <QJsonDocument>
#include <QSaveFile>
#include <QSet>

namespace quickdesk {

bool isWorkflowTriggerEvent(const QString& type) {
    static const QSet<QString> allowed{
        QStringLiteral("device.online.changed"),
        QStringLiteral("device.offline.sustained"),
        QStringLiteral("device.metric.alert"),
        QStringLiteral("device.access.granted"),
        QStringLiteral("device.access.denied"),
        QStringLiteral("device.access_approval.requested"),
        QStringLiteral("device.access_approval.decided"),
        QStringLiteral("remote_session.started"),
        QStringLiteral("remote_session.ended"),
        QStringLiteral("recording.created"),
        QStringLiteral("recording.updated"),
        QStringLiteral("recording.deleted"),
        QStringLiteral("scheduled_task.failed"),
        // QD-607: a scheduled connection prompt. Without it the event cannot
        // trigger a workflow, which is the one automated thing an owner might
        // reasonably want a scheduled connection to do.
        QStringLiteral("device.connect_prompted"),
    };
    return allowed.contains(type);
}

QJsonObject workflowTriggerEventData(const QJsonObject& message) {
    const QString type = message.value(QStringLiteral("type")).toString();
    if (!isWorkflowTriggerEvent(type)) {
        return {};
    }

    static const QStringList envelopeFields{
        QStringLiteral("id"),
        QStringLiteral("type"),
        QStringLiteral("device_id"),
        QStringLiteral("data"),
        QStringLiteral("ts"),
        QStringLiteral("server_rev"),
    };
    QJsonObject event;
    for (const QString& field : envelopeFields) {
        if (message.contains(field)) {
            event.insert(field, message.value(field));
        }
    }
    if (!event.contains(QStringLiteral("device_id"))) {
        const QString deviceId =
            event.value(QStringLiteral("data"))
                .toObject()
                .value(QStringLiteral("device_id"))
                .toString();
        if (!deviceId.isEmpty()) {
            event.insert(QStringLiteral("device_id"), deviceId);
        }
    }
    return event;
}

qint64 controlPlaneReplaySinceRev(const QJsonObject& authParams) {
    return qMax<qint64>(
        0,
        authParams.value(QStringLiteral("since_server_rev"))
            .toVariant()
            .toLongLong());
}

ControlPlaneEventReplayBuffer::ControlPlaneEventReplayBuffer(int capacity)
    : m_capacity(qMax(1, capacity)) {}

bool ControlPlaneEventReplayBuffer::append(const QJsonObject& event) {
    if (!appendInMemory(event)) {
        return true;
    }
    return persist();
}

bool ControlPlaneEventReplayBuffer::appendInMemory(
    const QJsonObject& event) {
    if (!isWorkflowTriggerEvent(
            event.value(QStringLiteral("type")).toString())) {
        return false;
    }
    const QString id = event.value(QStringLiteral("id")).toString();
    const qint64 serverRev =
        event.value(QStringLiteral("server_rev")).toVariant().toLongLong();
    if (id.isEmpty() && serverRev <= 0) {
        return false;
    }
    if (contains(event)) {
        return false;
    }
    m_events.append(event);
    while (m_events.size() > m_capacity) {
        m_events.removeFirst();
    }
    return true;
}

QList<QJsonObject>
ControlPlaneEventReplayBuffer::eventsAfter(qint64 sinceServerRev) const {
    const qint64 cursor = qMax<qint64>(0, sinceServerRev);
    QList<QJsonObject> events;
    for (const QJsonObject& event : m_events) {
        const qint64 serverRev =
            event.value(QStringLiteral("server_rev")).toVariant().toLongLong();
        if (serverRev <= 0 || serverRev > cursor) {
            events.append(event);
        }
    }
    return events;
}

int ControlPlaneEventReplayBuffer::size() const {
    return m_events.size();
}

bool ControlPlaneEventReplayBuffer::configureCapacity(int capacity) {
    const int normalized = qMax(1, capacity);
    if (normalized == m_capacity) {
        return true;
    }
    m_capacity = normalized;
    while (m_events.size() > m_capacity) {
        m_events.removeFirst();
    }
    return persist();
}

void ControlPlaneEventReplayBuffer::clear() {
    m_events.clear();
}

bool ControlPlaneEventReplayBuffer::loadFromFile(const QString& path,
                                                  qint64 userId) {
    clear();
    if (path.isEmpty() || userId <= 0) {
        return true;
    }
    QFile file(path);
    if (!file.exists()) {
        return true;
    }
    if (!file.open(QIODevice::ReadOnly)) {
        return false;
    }
    const QJsonDocument doc = QJsonDocument::fromJson(file.readAll());
    if (!doc.isObject()) {
        return false;
    }
    const QJsonObject root = doc.object();
    if (root.value(QStringLiteral("version")).toInt() != 1 ||
        root.value(QStringLiteral("user_id")).toVariant().toLongLong() !=
            userId) {
        return true;
    }
    const QJsonValue eventsValue = root.value(QStringLiteral("events"));
    if (!eventsValue.isArray()) {
        return false;
    }
    for (const QJsonValue& value : eventsValue.toArray()) {
        if (value.isObject()) {
            appendInMemory(value.toObject());
        }
    }
    return true;
}

bool ControlPlaneEventReplayBuffer::saveToFile(const QString& path,
                                                qint64 userId) const {
    if (path.isEmpty() || userId <= 0) {
        return false;
    }
    const QFileInfo fileInfo(path);
    if (!QDir().mkpath(fileInfo.absolutePath())) {
        return false;
    }
    QJsonArray events;
    for (const QJsonObject& event : m_events) {
        events.append(event);
    }
    const QJsonObject root{
        {QStringLiteral("version"), 1},
        {QStringLiteral("user_id"), userId},
        {QStringLiteral("events"), events},
    };
    QSaveFile file(path);
    if (!file.open(QIODevice::WriteOnly)) {
        return false;
    }
    file.write(QJsonDocument(root).toJson(QJsonDocument::Compact));
    return file.commit();
}

bool ControlPlaneEventReplayBuffer::configureStorage(const QString& path,
                                                      qint64 userId) {
    m_storagePath = path;
    m_userId = userId;
    if (m_storagePath.isEmpty() || m_userId <= 0) {
        clear();
        return true;
    }
    return loadFromFile(m_storagePath, m_userId);
}

bool ControlPlaneEventReplayBuffer::persist() const {
    if (m_storagePath.isEmpty() || m_userId <= 0) {
        return true;
    }
    return saveToFile(m_storagePath, m_userId);
}

bool ControlPlaneEventReplayBuffer::contains(
    const QJsonObject& event) const {
    const QString id = event.value(QStringLiteral("id")).toString();
    const qint64 serverRev =
        event.value(QStringLiteral("server_rev")).toVariant().toLongLong();
    for (const QJsonObject& existing : m_events) {
        if (!id.isEmpty() &&
            existing.value(QStringLiteral("id")).toString() == id) {
            return true;
        }
        if (id.isEmpty() && serverRev > 0 &&
            existing.value(QStringLiteral("server_rev"))
                    .toVariant()
                    .toLongLong() == serverRev) {
            return true;
        }
    }
    return false;
}

}  // namespace quickdesk
