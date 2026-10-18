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

    bool append(const QJsonObject& event);
    QList<QJsonObject> eventsAfter(qint64 sinceServerRev) const;
    int size() const;
    void clear();
    bool loadFromFile(const QString& path, qint64 userId);
    bool saveToFile(const QString& path, qint64 userId) const;
    bool configureStorage(const QString& path, qint64 userId);

private:
    bool contains(const QJsonObject& event) const;
    bool appendInMemory(const QJsonObject& event);
    bool persist() const;

    int m_capacity;
    QList<QJsonObject> m_events;
    QString m_storagePath;
    qint64 m_userId = 0;
};

}  // namespace quickdesk
