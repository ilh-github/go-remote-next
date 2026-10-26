// Copyright 2026 QuickDesk Authors

#include "ScheduledTaskManager.h"

#include "AuthManager.h"
#include "infra/log/log.h"

#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QUrl>

namespace quickdesk {

ScheduledTaskManager::ScheduledTaskManager(AuthManager* authManager,
                                           QObject* parent)
    : QObject(parent)
    , m_authManager(authManager)
{
    if (m_authManager) {
        connect(m_authManager, &AuthManager::loggedOut, this, [this]() {
            ++m_generation;
            m_tasks.clear();
            m_supportedKinds.clear();
            m_supportedSchedules.clear();
            emit tasksChanged();
            emit supportedKindsChanged();
            emit supportedSchedulesChanged();
        });
    }
}

void ScheduledTaskManager::refresh()
{
    if (!m_authManager || !m_authManager->isLoggedIn()) {
        const QString message =
            tr("Please login to manage scheduled tasks");
        setLastError(message);
        emit requestFailed(message);
        return;
    }

    const int generation = ++m_generation;
    const QUrl url(m_authManager->httpBaseUrl()
                   + QStringLiteral("v1/me/scheduled-tasks"));
    beginRequest();
    m_authManager->request(
        QStringLiteral("GET"), url, QString(),
        [this, generation](int statusCode, const std::string& errorMsg,
                           const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, generation, statusCode, errorMsg, data]() {
                    if (generation == m_generation) {
                        if (statusCode == 200 && errorMsg.empty()) {
                            const QJsonObject root =
                                QJsonDocument::fromJson(
                                    QByteArray::fromStdString(data)).object();
                            QVariantList rows;
                            for (const QJsonValue& value :
                                 root.value(QStringLiteral("items")).toArray()) {
                                rows.append(normalizeTask(value.toObject()));
                            }
                            m_tasks = rows;
                            m_supportedKinds = stringList(
                                root.value(
                                    QStringLiteral("supported_kinds")).toArray());
                            m_supportedSchedules = stringList(
                                root.value(
                                    QStringLiteral("supported_schedules")).toArray());
                            setLastError(QString());
                            emit tasksChanged();
                            emit supportedKindsChanged();
                            emit supportedSchedulesChanged();
                        } else {
                            const QString message =
                                problemMessage(data, errorMsg, statusCode);
                            setLastError(message);
                            LOG_WARN("[ScheduledTaskManager] list failed: status={} err={}",
                                     statusCode, message.toStdString());
                            emit requestFailed(message);
                        }
                    }
                    finishRequest();
                });
        });
}

void ScheduledTaskManager::createTask(
    const QString& kind, const QString& runAt, int repeatSeconds,
    int maxAttempts, const QString& schedule, const QString& timezone,
    bool skipHolidays)
{
    if (!m_authManager || !m_authManager->isLoggedIn()) {
        const QString message =
            tr("Please login to create scheduled tasks");
        setLastError(message);
        emit requestFailed(message);
        return;
    }

    const QString normalizedKind = kind.trimmed();
    const QString normalizedRunAt = runAt.trimmed();
    const QString normalizedSchedule = schedule.trimmed();
    if (normalizedKind.isEmpty()
        || (normalizedRunAt.isEmpty() && normalizedSchedule.isEmpty())) {
        const QString message = tr("Choose an action and a valid schedule");
        setLastError(message);
        emit requestFailed(message);
        return;
    }
    if (maxAttempts < 1 || maxAttempts > 10) {
        const QString message =
            tr("Retry attempts must be between 1 and 10");
        setLastError(message);
        emit requestFailed(message);
        return;
    }

    QJsonObject body;
    body[QStringLiteral("kind")] = normalizedKind;
    body[QStringLiteral("max_attempts")] = maxAttempts;
    if (normalizedSchedule.isEmpty()) {
        body[QStringLiteral("run_at")] = normalizedRunAt;
        body[QStringLiteral("repeat_seconds")] = qMax(0, repeatSeconds);
    } else {
        body[QStringLiteral("schedule")] = normalizedSchedule;
        body[QStringLiteral("timezone")] = timezone.trimmed();
        body[QStringLiteral("skip_holidays")] = skipHolidays;
    }

    const QUrl url(m_authManager->httpBaseUrl()
                   + QStringLiteral("v1/me/scheduled-tasks"));
    beginRequest();
    m_authManager->request(
        QStringLiteral("POST"), url,
        QString::fromUtf8(
            QJsonDocument(body).toJson(QJsonDocument::Compact)),
        [this](int statusCode, const std::string& errorMsg,
               const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, statusCode, errorMsg, data]() {
                    if (statusCode == 201 && errorMsg.empty()) {
                        const QVariantMap row = normalizeTask(
                            QJsonDocument::fromJson(
                                QByteArray::fromStdString(data)).object());
                        m_tasks.append(row);
                        setLastError(QString());
                        emit tasksChanged();
                        emit taskCreated(
                            row.value(QStringLiteral("id")).toInt());
                    } else {
                        const QString message =
                            problemMessage(data, errorMsg, statusCode);
                        setLastError(message);
                        LOG_WARN("[ScheduledTaskManager] create failed: status={} err={}",
                                 statusCode, message.toStdString());
                        emit requestFailed(message);
                    }
                    finishRequest();
                });
        });
}

void ScheduledTaskManager::deleteTask(int taskId)
{
    if (!m_authManager || !m_authManager->isLoggedIn() || taskId <= 0) {
        return;
    }
    const QUrl url(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/scheduled-tasks/")
        + QString::number(taskId));
    beginRequest();
    m_authManager->request(
        QStringLiteral("DELETE"), url, QString(),
        [this, taskId](int statusCode, const std::string& errorMsg,
                       const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, taskId, statusCode, errorMsg, data]() {
                    if (statusCode == 204 && errorMsg.empty()) {
                        for (int i = 0; i < m_tasks.size(); ++i) {
                            if (m_tasks.at(i).toMap()
                                    .value(QStringLiteral("id")).toInt()
                                == taskId) {
                                m_tasks.removeAt(i);
                                break;
                            }
                        }
                        setLastError(QString());
                        emit tasksChanged();
                        emit taskDeleted(taskId);
                    } else {
                        const QString message =
                            problemMessage(data, errorMsg, statusCode);
                        setLastError(message);
                        LOG_WARN("[ScheduledTaskManager] delete({}) failed: status={} err={}",
                                 taskId, statusCode, message.toStdString());
                        emit requestFailed(message);
                    }
                    finishRequest();
                });
        });
}

void ScheduledTaskManager::handleRealtimeEvent(const QVariantMap& message)
{
    const QString type = message.value(QStringLiteral("type")).toString();
    if (type != QStringLiteral("scheduled_task.failed")
        && type != QStringLiteral("device.connect_prompted")) {
        return;
    }
    const QVariantMap data =
        message.value(QStringLiteral("data")).toMap();
    // QD-607: a scheduled connection came due. The server publishes it and the
    // web client consumes it; without this the desktop client is the one place
    // the owner is not told, which is the same shape as the notification
    // subscription gap and the missing userSync branch before it.
    //
    // A prompt with no task id is dropped rather than surfaced: the page would
    // have nothing to open, so an empty row is worse than no row.
    if (type == QStringLiteral("device.connect_prompted")) {
        const int taskId = data.value(QStringLiteral("task_id")).toInt();
        if (taskId <= 0) {
            return;
        }
        emit realtimeConnectPrompted(taskId);
        emit realtimeRefreshRequested();
        return;
    }
    emit realtimeTaskFailed(
        data.value(QStringLiteral("kind")).toString(),
        data.value(QStringLiteral("task_id")).toInt(),
        data.value(QStringLiteral("error")).toString());
    emit realtimeRefreshRequested();
}

void ScheduledTaskManager::beginRequest()
{
    if (m_pendingRequests++ == 0) {
        emit busyChanged();
    }
}

void ScheduledTaskManager::finishRequest()
{
    if (m_pendingRequests <= 0) {
        return;
    }
    if (--m_pendingRequests == 0) {
        emit busyChanged();
    }
}

void ScheduledTaskManager::setLastError(const QString& message)
{
    if (m_lastError == message) {
        return;
    }
    m_lastError = message;
    emit lastErrorChanged();
}

QVariantMap ScheduledTaskManager::normalizeTask(const QJsonObject& object)
{
    QVariantMap row = object.toVariantMap();
    row[QStringLiteral("id")] = object.value(QStringLiteral("id")).toInt();
    row[QStringLiteral("max_attempts")] =
        qMax(1, object.value(QStringLiteral("max_attempts")).toInt(1));
    row[QStringLiteral("attempts")] =
        qMax(0, object.value(QStringLiteral("attempts")).toInt());
    row[QStringLiteral("repeat_seconds")] =
        qMax(0, object.value(QStringLiteral("repeat_seconds")).toInt());
    row[QStringLiteral("enabled")] =
        object.value(QStringLiteral("enabled")).toBool();
    row[QStringLiteral("skip_holidays")] =
        object.value(QStringLiteral("skip_holidays")).toBool();
    return row;
}

QStringList ScheduledTaskManager::stringList(const QJsonArray& array)
{
    QStringList result;
    for (const QJsonValue& value : array) {
        const QString item = value.toString().trimmed();
        if (!item.isEmpty() && !result.contains(item)) {
            result.append(item);
        }
    }
    return result;
}

QString ScheduledTaskManager::problemMessage(const std::string& data,
                                              const std::string& fallback,
                                              int statusCode)
{
    const QJsonObject object =
        QJsonDocument::fromJson(QByteArray::fromStdString(data)).object();
    QString message = object.value(QStringLiteral("detail")).toString();
    if (message.isEmpty()) {
        message = object.value(QStringLiteral("title")).toString();
    }
    if (message.isEmpty()) {
        message = QString::fromStdString(fallback);
    }
    return message.isEmpty()
        ? tr("Request failed (HTTP %1)").arg(statusCode)
        : message;
}

}  // namespace quickdesk
