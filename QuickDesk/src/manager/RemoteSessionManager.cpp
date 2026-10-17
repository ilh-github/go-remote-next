// Copyright 2026 QuickDesk Authors
#include "RemoteSessionManager.h"

#include "AuthManager.h"
#include "infra/log/log.h"

#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QUrl>
#include <QUrlQuery>

namespace quickdesk {

namespace {
constexpr int kListLimit = 50;

QString encodedPathSegment(const QString& value)
{
    return QString::fromLatin1(QUrl::toPercentEncoding(value));
}
}

RemoteSessionManager::RemoteSessionManager(AuthManager* authManager, QObject* parent)
    : QObject(parent)
    , m_authManager(authManager)
{
    if (m_authManager) {
        connect(m_authManager, &AuthManager::loggedOut, this, [this]() {
            ++m_listGeneration;
            m_sessions.clear();
            m_nextCursor.clear();
            emit sessionsChanged();
            emit hasMoreChanged();
        });
    }
}

void RemoteSessionManager::refresh(const QString& status, const QString& deviceId)
{
    const QString nextStatus = (status == QStringLiteral("active")
                                || status == QStringLiteral("ended")
                                || status == QStringLiteral("all"))
        ? status
        : QStringLiteral("all");
    const QString nextDeviceId = deviceId.trimmed();
    const bool filtersChanged = nextStatus != m_statusFilter
        || nextDeviceId != m_deviceIdFilter;

    m_statusFilter = nextStatus;
    m_deviceIdFilter = nextDeviceId;
    ++m_listGeneration;
    m_nextCursor.clear();
    if (filtersChanged) {
        m_sessions.clear();
        emit sessionsChanged();
    }
    emit hasMoreChanged();
    requestList(false, m_listGeneration);
}

void RemoteSessionManager::loadMore()
{
    if (m_nextCursor.isEmpty()) {
        return;
    }
    requestList(true, m_listGeneration);
}

void RemoteSessionManager::requestList(bool append, int generation)
{
    if (!m_authManager || !m_authManager->isLoggedIn()) {
        setLastError(tr("Please login to view remote sessions"));
        emit requestFailed(m_lastError);
        return;
    }

    QUrl url(m_authManager->httpBaseUrl() + QStringLiteral("v1/me/remote-sessions"));
    QUrlQuery query;
    query.addQueryItem(QStringLiteral("status"), m_statusFilter);
    if (!m_deviceIdFilter.isEmpty()) {
        query.addQueryItem(QStringLiteral("device_id"), m_deviceIdFilter);
    }
    if (append && !m_nextCursor.isEmpty()) {
        query.addQueryItem(QStringLiteral("cursor"), m_nextCursor);
    }
    query.addQueryItem(QStringLiteral("limit"), QString::number(kListLimit));
    url.setQuery(query);

    beginRequest();
    m_authManager->request(
        QStringLiteral("GET"), url, QString(),
        [this, append, generation](int statusCode,
                                   const std::string& errorMsg,
                                   const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, append, generation, statusCode, errorMsg, data]() {
                    if (generation == m_listGeneration) {
                        if (statusCode == 200 && errorMsg.empty()) {
                            const QJsonObject root =
                                QJsonDocument::fromJson(
                                    QByteArray::fromStdString(data)).object();
                            const QJsonArray items = root.value(QStringLiteral("items")).toArray();
                            QVariantList nextRows;
                            if (append) {
                                nextRows = m_sessions;
                            }
                            for (const QJsonValue& value : items) {
                                nextRows.append(normalizeSession(value.toObject()));
                            }
                            m_sessions = nextRows;
                            m_nextCursor =
                                root.value(QStringLiteral("next_cursor")).toString();
                            setLastError(QString());
                            emit sessionsChanged();
                            emit hasMoreChanged();
                        } else {
                            const QString message =
                                problemMessage(data, errorMsg, statusCode);
                            setLastError(message);
                            LOG_WARN("[RemoteSessionManager] list failed: status={} err={}",
                                     statusCode, message.toStdString());
                            emit requestFailed(message);
                        }
                    }
                    finishRequest();
                });
        });
}

void RemoteSessionManager::fetchDetail(const QString& sessionId)
{
    if (!m_authManager || !m_authManager->isLoggedIn() || sessionId.isEmpty()) {
        return;
    }
    const int row = rowForSession(sessionId);
    if (row < 0) {
        return;
    }
    QVariantMap current = m_sessions.at(row).toMap();
    if (current.value(QStringLiteral("detail_loaded")).toBool()
        || current.value(QStringLiteral("detail_loading")).toBool()) {
        return;
    }
    current[QStringLiteral("detail_loading")] = true;
    m_sessions[row] = current;
    emit sessionsChanged();

    const QUrl url(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/remote-sessions/")
        + encodedPathSegment(sessionId));
    beginRequest();
    m_authManager->request(
        QStringLiteral("GET"), url, QString(),
        [this, sessionId](int statusCode,
                          const std::string& errorMsg,
                          const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, sessionId, statusCode, errorMsg, data]() {
                    const int index = rowForSession(sessionId);
                    if (statusCode == 200 && errorMsg.empty()) {
                        const QJsonObject root =
                            QJsonDocument::fromJson(
                                QByteArray::fromStdString(data)).object();
                        if (index >= 0) {
                            QVariantMap row = normalizeDetail(root);
                            row[QStringLiteral("detail_loading")] = false;
                            row[QStringLiteral("detail_loaded")] = true;
                            m_sessions[index] = row;
                            setLastError(QString());
                            emit sessionsChanged();
                            emit detailLoaded(sessionId);
                        }
                    } else {
                        const QString message =
                            problemMessage(data, errorMsg, statusCode);
                        if (index >= 0) {
                            QVariantMap row = m_sessions.at(index).toMap();
                            row[QStringLiteral("detail_loading")] = false;
                            m_sessions[index] = row;
                            emit sessionsChanged();
                        }
                        setLastError(message);
                        LOG_WARN("[RemoteSessionManager] detail({}) failed: status={} err={}",
                                 sessionId.toStdString(), statusCode,
                                 message.toStdString());
                        emit requestFailed(message);
                    }
                    finishRequest();
                });
        });
}

void RemoteSessionManager::revokeSession(const QString& sessionId)
{
    if (!m_authManager || !m_authManager->isLoggedIn() || sessionId.isEmpty()) {
        return;
    }
    const QUrl url(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/remote-sessions/")
        + encodedPathSegment(sessionId)
        + QStringLiteral("/revoke"));
    beginRequest();
    m_authManager->request(
        QStringLiteral("POST"), url, QString(),
        [this, sessionId](int statusCode,
                          const std::string& errorMsg,
                          const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, sessionId, statusCode, errorMsg, data]() {
                    if (statusCode == 200 && errorMsg.empty()) {
                        setLastError(QString());
                        emit sessionRevoked(sessionId);
                    } else {
                        const QString message =
                            problemMessage(data, errorMsg, statusCode);
                        setLastError(message);
                        LOG_WARN("[RemoteSessionManager] revoke({}) failed: status={} err={}",
                                 sessionId.toStdString(), statusCode,
                                 message.toStdString());
                        emit requestFailed(message);
                    }
                    finishRequest();
                });
        });
}

void RemoteSessionManager::revokeDeviceSessions(const QString& deviceId)
{
    if (!m_authManager || !m_authManager->isLoggedIn() || deviceId.isEmpty()) {
        return;
    }
    const QUrl url(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/devices/")
        + encodedPathSegment(deviceId)
        + QStringLiteral("/sessions:revoke"));
    beginRequest();
    m_authManager->request(
        QStringLiteral("POST"), url, QString(),
        [this, deviceId](int statusCode,
                         const std::string& errorMsg,
                         const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, deviceId, statusCode, errorMsg, data]() {
                    if (statusCode == 200 && errorMsg.empty()) {
                        const QJsonObject root =
                            QJsonDocument::fromJson(
                                QByteArray::fromStdString(data)).object();
                        setLastError(QString());
                        emit deviceSessionsRevoked(
                            deviceId,
                            root.value(QStringLiteral("revoked")).toInt(),
                            root.value(QStringLiteral("failed")).toInt(),
                            root.value(QStringLiteral("truncated")).toBool());
                    } else {
                        const QString message =
                            problemMessage(data, errorMsg, statusCode);
                        setLastError(message);
                        LOG_WARN("[RemoteSessionManager] clear({}) failed: status={} err={}",
                                 deviceId.toStdString(), statusCode,
                                 message.toStdString());
                        emit requestFailed(message);
                    }
                    finishRequest();
                });
        });
}

void RemoteSessionManager::banSource(const QString& deviceId, const QString& source)
{
    if (!m_authManager || !m_authManager->isLoggedIn()
        || deviceId.trimmed().isEmpty() || source.trimmed().isEmpty()) {
        return;
    }
    const QString normalizedDeviceId = deviceId.trimmed();
    const QString normalizedSource = source.trimmed();
    const QUrl url(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/devices/")
        + encodedPathSegment(normalizedDeviceId)
        + QStringLiteral("/access-bans"));
    const QJsonObject body{
        {QStringLiteral("source"), normalizedSource},
    };
    beginRequest();
    m_authManager->request(
        QStringLiteral("POST"), url,
        QString::fromUtf8(QJsonDocument(body).toJson(QJsonDocument::Compact)),
        [this, normalizedDeviceId, normalizedSource](
            int statusCode, const std::string& errorMsg, const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, normalizedDeviceId, normalizedSource,
                 statusCode, errorMsg, data]() {
                    if (statusCode == 200 && errorMsg.empty()) {
                        setLastError(QString());
                        emit sourceBanned(normalizedDeviceId, normalizedSource);
                    } else {
                        const QString message =
                            problemMessage(data, errorMsg, statusCode);
                        setLastError(message);
                        LOG_WARN("[RemoteSessionManager] ban({} -> {}) failed: status={} err={}",
                                 normalizedDeviceId.toStdString(),
                                 normalizedSource.toStdString(),
                                 statusCode, message.toStdString());
                        emit requestFailed(message);
                    }
                    finishRequest();
                });
        });
}

void RemoteSessionManager::beginRequest()
{
    const bool wasBusy = busy();
    ++m_pendingRequests;
    if (!wasBusy) {
        emit busyChanged();
    }
}

void RemoteSessionManager::finishRequest()
{
    if (m_pendingRequests <= 0) {
        return;
    }
    --m_pendingRequests;
    if (m_pendingRequests == 0) {
        emit busyChanged();
    }
}

void RemoteSessionManager::setLastError(const QString& message)
{
    if (m_lastError == message) {
        return;
    }
    m_lastError = message;
    emit lastErrorChanged();
}

int RemoteSessionManager::rowForSession(const QString& sessionId) const
{
    for (int i = 0; i < m_sessions.size(); ++i) {
        if (m_sessions.at(i).toMap().value(QStringLiteral("session_id")).toString()
            == sessionId) {
            return i;
        }
    }
    return -1;
}

QVariantMap RemoteSessionManager::normalizeSession(const QJsonObject& object)
{
    QVariantMap row = object.toVariantMap();
    row[QStringLiteral("detail_loaded")] = false;
    row[QStringLiteral("detail_loading")] = false;
    row[QStringLiteral("recordings")] = QVariantList();
    row[QStringLiteral("access_log")] = QVariantList();
    row[QStringLiteral("recording_count")] = 0;
    row[QStringLiteral("access_log_count")] = 0;
    return row;
}

QVariantMap RemoteSessionManager::normalizeDetail(const QJsonObject& object)
{
    QVariantMap row =
        object.value(QStringLiteral("session")).toObject().toVariantMap();
    const QJsonObject counts =
        object.value(QStringLiteral("counts")).toObject();
    row[QStringLiteral("recordings")] =
        objectArrayToVariantList(
            object.value(QStringLiteral("recordings")).toArray());
    row[QStringLiteral("access_log")] =
        objectArrayToVariantList(
            object.value(QStringLiteral("access_log")).toArray());
    row[QStringLiteral("recording_count")] =
        counts.value(QStringLiteral("recordings")).toInt();
    row[QStringLiteral("access_log_count")] =
        counts.value(QStringLiteral("access_log")).toInt();
    return row;
}

QVariantList RemoteSessionManager::objectArrayToVariantList(const QJsonArray& array)
{
    QVariantList result;
    result.reserve(array.size());
    for (const QJsonValue& value : array) {
        result.append(value.toObject().toVariantMap());
    }
    return result;
}

QString RemoteSessionManager::problemMessage(const std::string& data,
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
        message = object.value(QStringLiteral("error")).toString();
    }
    if (message.isEmpty()) {
        message = object.value(QStringLiteral("message")).toString();
    }
    if (message.isEmpty()) {
        message = QString::fromStdString(fallback);
    }
    if (message.isEmpty()) {
        message = QStringLiteral("HTTP %1").arg(statusCode);
    }
    return message;
}

} // namespace quickdesk
