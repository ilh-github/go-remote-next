// Copyright 2026 QuickDesk Authors

#pragma once

#include <QList>
#include <QJsonObject>
#include <QString>

namespace quickdesk {

bool isWorkflowTriggerEvent(const QString& type);
QJsonObject workflowTriggerEventData(const QJsonObject& message);
qint64 controlPlaneReplaySinceRev(const QJsonObject& authParams);

class ControlPlaneEventReplayBuffer {
public:
    explicit ControlPlaneEventReplayBuffer(int capacity = 256);

    void append(const QJsonObject& event);
    QList<QJsonObject> eventsAfter(qint64 sinceServerRev) const;
    int size() const;
    void clear();

private:
    bool contains(const QJsonObject& event) const;

    int m_capacity;
    QList<QJsonObject> m_events;
};

}  // namespace quickdesk
