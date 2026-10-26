// Copyright 2026 QuickDesk Authors
#ifndef QUICKDESK_MANAGER_SCHEDULEDTASKMANAGER_H
#define QUICKDESK_MANAGER_SCHEDULEDTASKMANAGER_H

#include <QObject>
#include <QString>
#include <QStringList>
#include <QVariantList>
#include <QVariantMap>

#include <string>

class QJsonArray;
class QJsonObject;

namespace quickdesk {

class AuthManager;

class ScheduledTaskManager : public QObject {
    Q_OBJECT
    Q_PROPERTY(QVariantList tasks READ tasks NOTIFY tasksChanged)
    Q_PROPERTY(QStringList supportedKinds READ supportedKinds
               NOTIFY supportedKindsChanged)
    Q_PROPERTY(QStringList supportedSchedules READ supportedSchedules
               NOTIFY supportedSchedulesChanged)
    Q_PROPERTY(bool busy READ busy NOTIFY busyChanged)
    Q_PROPERTY(QString lastError READ lastError NOTIFY lastErrorChanged)

public:
    explicit ScheduledTaskManager(AuthManager* authManager,
                                  QObject* parent = nullptr);

    QVariantList tasks() const { return m_tasks; }
    QStringList supportedKinds() const { return m_supportedKinds; }
    QStringList supportedSchedules() const { return m_supportedSchedules; }
    bool busy() const { return m_pendingRequests > 0; }
    QString lastError() const { return m_lastError; }

    Q_INVOKABLE void refresh();
    Q_INVOKABLE void createTask(const QString& kind,
                                const QString& runAt,
                                int repeatSeconds,
                                int maxAttempts,
                                const QString& schedule = QString(),
                                const QString& timezone = QString(),
                                bool skipHolidays = false);
    Q_INVOKABLE void deleteTask(int taskId);
    Q_INVOKABLE void handleRealtimeEvent(const QVariantMap& message);

signals:
    void tasksChanged();
    void supportedKindsChanged();
    void supportedSchedulesChanged();
    void busyChanged();
    void lastErrorChanged();
    void requestFailed(const QString& message);
    void taskCreated(int taskId);
    void taskDeleted(int taskId);
    void realtimeTaskFailed(const QString& kind, int taskId,
                            const QString& error);
    // QD-607: a scheduled connection came due. Carries only the task id, because
    // the page reads the schedule and status from the list it refreshes.
    void realtimeConnectPrompted(int taskId);
    void realtimeRefreshRequested();

private:
    void beginRequest();
    void finishRequest();
    void setLastError(const QString& message);
    static QVariantMap normalizeTask(const QJsonObject& object);
    static QStringList stringList(const QJsonArray& array);
    static QString problemMessage(const std::string& data,
                                  const std::string& fallback,
                                  int statusCode);

    AuthManager* m_authManager = nullptr;
    QVariantList m_tasks;
    QStringList m_supportedKinds;
    QStringList m_supportedSchedules;
    QString m_lastError;
    int m_pendingRequests = 0;
    int m_generation = 0;
};

}  // namespace quickdesk

#endif  // QUICKDESK_MANAGER_SCHEDULEDTASKMANAGER_H
