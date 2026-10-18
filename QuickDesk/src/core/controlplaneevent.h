// Copyright 2026 QuickDesk Authors

#pragma once

#include <QList>
#include <QJsonObject>
#include <QString>

namespace quickdesk {

inline constexpr int kDefaultControlPlaneReplayCapacity = 2048;

bool isWorkflowTriggerEvent(const QString& type);
QJsonObject workflowTriggerEventData(const QJsonObject& message);
qint64 controlPlaneReplaySinceRev(const QJsonObject& authParams);

class ControlPlaneEventReplayBuffer {
public:
    explicit ControlPlaneEventReplayBuffer(
        int capacity = kDefaultControlPlaneReplayCapacity);

    bool append(const QJsonObject& event);
    QList<QJsonObject> eventsAfter(qint64 sinceServerRev) const;
    int size() const;
    int capacity() const { return m_capacity; }
    bool configureCapacity(int capacity);
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
