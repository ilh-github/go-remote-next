// Copyright 2026 QuickDesk Authors

#include "controlplaneevent.h"

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

void ControlPlaneEventReplayBuffer::append(const QJsonObject& event) {
    if (!isWorkflowTriggerEvent(
            event.value(QStringLiteral("type")).toString())) {
        return;
    }
    const QString id = event.value(QStringLiteral("id")).toString();
    const qint64 serverRev =
        event.value(QStringLiteral("server_rev")).toVariant().toLongLong();
    if (id.isEmpty() && serverRev <= 0) {
        return;
    }
    if (contains(event)) {
        return;
    }
    m_events.append(event);
    while (m_events.size() > m_capacity) {
        m_events.removeFirst();
    }
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

void ControlPlaneEventReplayBuffer::clear() {
    m_events.clear();
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
