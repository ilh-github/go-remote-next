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

}  // namespace quickdesk
