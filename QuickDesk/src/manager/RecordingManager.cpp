// Copyright 2026 QuickDesk Authors

#include "RecordingManager.h"

#include "AuthManager.h"
#include "infra/log/log.h"

#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QMetaMethod>
#include <QUrl>
#include <QUrlQuery>

#include <algorithm>

namespace quickdesk {

RecordingManager::RecordingManager(AuthManager* authManager,
                                   QObject* parent)
    : QObject(parent)
    , m_authManager(authManager)
{
    if (m_authManager) {
        connect(m_authManager, &AuthManager::loggedOut, this, [this]() {
            ++m_generation;
            m_recordings.clear();
            m_nextCursor.clear();
            emit recordingsChanged();
            emit nextCursorChanged();
        });
    }
}

void RecordingManager::refresh()
{
    if (!m_authManager || !m_authManager->isLoggedIn()) {
        const QString message =
            tr("Please login to view recordings");
        setLastError(message);
        emit requestFailed(message);
        return;
    }
    const int generation = ++m_generation;
    requestPage(QString(), false, generation);
}

void RecordingManager::loadMore()
{
    if (!m_authManager || !m_authManager->isLoggedIn()
        || m_nextCursor.isEmpty()) {
        return;
    }
    requestPage(m_nextCursor, true, m_generation);
}

void RecordingManager::requestPage(const QString& cursor, bool append,
                                   int generation)
{
    QUrl url(m_authManager->httpBaseUrl()
             + QStringLiteral("v1/me/recordings"));
    if (!cursor.isEmpty()) {
        QUrlQuery query;
        query.addQueryItem(QStringLiteral("cursor"), cursor);
        url.setQuery(query);
    }

    beginRequest();
    m_authManager->request(
        QStringLiteral("GET"), url, QString(),
        [this, append, generation](int statusCode,
                                   const std::string& errorMsg,
                                   const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, append, generation, statusCode, errorMsg, data]() {
                    if (generation == m_generation) {
                        if (statusCode == 200 && errorMsg.empty()) {
                            const QJsonObject root =
                                QJsonDocument::fromJson(
                                    QByteArray::fromStdString(data)).object();
                            QVariantList rows;
                            for (const QJsonValue& value :
                                 root.value(QStringLiteral("items")).toArray()) {
                                rows.append(
                                    normalizeRecording(value.toObject()));
                            }
                            if (append) {
                                m_recordings.append(rows);
                            } else {
                                m_recordings = rows;
                            }
                            m_nextCursor =
                                root.value(
                                    QStringLiteral("next_cursor")).toString();
                            setLastError(QString());
                            emit recordingsChanged();
                            emit nextCursorChanged();
                        } else {
                            const QString message =
                                problemMessage(data, errorMsg, statusCode);
                            setLastError(message);
                            LOG_WARN("[RecordingManager] list failed: status={} err={}",
                                     statusCode, message.toStdString());
                            emit requestFailed(message);
                        }
                    }
                    finishRequest();
                });
        });
}

void RecordingManager::deleteRecording(int recordingId)
{
    if (!m_authManager || !m_authManager->isLoggedIn()
        || recordingId <= 0) {
        return;
    }
    const QUrl url(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/recordings/")
        + QString::number(recordingId));
    beginRequest();
    m_authManager->request(
        QStringLiteral("DELETE"), url, QString(),
        [this, recordingId](int statusCode,
                            const std::string& errorMsg,
                            const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, recordingId, statusCode, errorMsg, data]() {
                    if (statusCode == 204 && errorMsg.empty()) {
                        for (int i = 0; i < m_recordings.size(); ++i) {
                            if (m_recordings.at(i).toMap()
                                    .value(QStringLiteral("id")).toInt()
                                == recordingId) {
                                m_recordings.removeAt(i);
                                break;
                            }
                        }
                        setLastError(QString());
                        emit recordingsChanged();
                        emit recordingDeleted(recordingId);
                    } else {
                        const QString message =
                            problemMessage(data, errorMsg, statusCode);
                        setLastError(message);
                        LOG_WARN("[RecordingManager] delete({}) failed: status={} err={}",
                                 recordingId, statusCode, message.toStdString());
                        emit requestFailed(message);
                    }
                    finishRequest();
                });
        });
}

void RecordingManager::requestArtifactUrl(int recordingId)
{
    requestTicket(recordingId, QStringLiteral("artifact-ticket"),
                  "artifactUrlReady");
}

void RecordingManager::requestThumbnailUrl(int recordingId)
{
    requestTicket(recordingId, QStringLiteral("thumbnail-ticket"),
                  "thumbnailUrlReady");
}

void RecordingManager::requestTicket(int recordingId, const QString& suffix,
                                     const char* signalName)
{
    if (!m_authManager || !m_authManager->isLoggedIn()
        || recordingId <= 0) {
        return;
    }
    const QUrl url(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/recordings/")
        + QString::number(recordingId)
        + QLatin1Char('/')
        + suffix);
    beginRequest();
    m_authManager->request(
        QStringLiteral("POST"), url, QString(),
        [this, recordingId, signalName](int statusCode,
                                        const std::string& errorMsg,
                                        const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, recordingId, signalName, statusCode, errorMsg, data]() {
                    if (statusCode == 200 && errorMsg.empty()) {
                        const QJsonObject root =
                            QJsonDocument::fromJson(
                                QByteArray::fromStdString(data)).object();
                        const QString raw =
                            root.value(QStringLiteral("url")).toString();
                        if (raw.isEmpty()) {
                            const QString message =
                                tr("Recording media URL is unavailable");
                            setLastError(message);
                            emit requestFailed(message);
                        } else {
                            setLastError(QString());
                            const int signalIndex =
                                metaObject()->indexOfSignal(
                                    QByteArray(signalName)
                                    + "(int,QString)");
                            if (signalIndex >= 0) {
                                QMetaMethod signal =
                                    metaObject()->method(signalIndex);
                                signal.invoke(this,
                                              Q_ARG(int, recordingId),
                                              Q_ARG(QString,
                                                    resolveUrl(raw)));
                            }
                        }
                    } else {
                        const QString message =
                            problemMessage(data, errorMsg, statusCode);
                        setLastError(message);
                        LOG_WARN("[RecordingManager] ticket({}) failed: status={} err={}",
                                 recordingId, statusCode,
                                 message.toStdString());
                        emit requestFailed(message);
                    }
                    finishRequest();
                });
        });
}

void RecordingManager::handleRealtimeEvent(const QVariantMap& message)
{
    const QString type = message.value(QStringLiteral("type")).toString();
    if (type.startsWith(QStringLiteral("recording."))) {
        emit realtimeRefreshRequested();
    }
}

void RecordingManager::beginRequest()
{
    if (m_pendingRequests++ == 0) {
        emit busyChanged();
    }
}

void RecordingManager::finishRequest()
{
    if (m_pendingRequests <= 0) {
        return;
    }
    if (--m_pendingRequests == 0) {
        emit busyChanged();
    }
}

void RecordingManager::setLastError(const QString& message)
{
    if (m_lastError == message) {
        return;
    }
    m_lastError = message;
    emit lastErrorChanged();
}

QString RecordingManager::resolveUrl(const QString& raw) const
{
    const QString value = raw.trimmed();
    if (value.isEmpty()) {
        return {};
    }
    const QUrl parsed(value);
    if (parsed.isValid() && !parsed.scheme().isEmpty()) {
        return parsed.toString();
    }
    QString base = m_authManager->httpBaseUrl();
    if (!base.endsWith(QLatin1Char('/'))) {
        base += QLatin1Char('/');
    }
    return QUrl(base
                + (value.startsWith(QLatin1Char('/'))
                       ? value.mid(1) : value)).toString();
}

QVariantMap RecordingManager::normalizeRecording(
    const QJsonObject& object)
{
    QVariantMap row = object.toVariantMap();
    row[QStringLiteral("id")] =
        object.value(QStringLiteral("id")).toInt();
    row[QStringLiteral("duration_seconds")] =
        qMax(0, object.value(
                    QStringLiteral("duration_seconds")).toInt());
    row[QStringLiteral("size_bytes")] =
        qMax<qint64>(0, object.value(
                            QStringLiteral("size_bytes")).toInteger());
    row[QStringLiteral("status")] =
        object.value(QStringLiteral("status")).toString();
    return row;
}

QString RecordingManager::problemMessage(const std::string& data,
                                          const std::string& fallback,
                                          int statusCode)
{
    const QJsonDocument document =
        QJsonDocument::fromJson(QByteArray::fromStdString(data));
    if (document.isObject()) {
        const QJsonObject problem = document.object();
        const QString detail =
            problem.value(QStringLiteral("detail")).toString();
        if (!detail.isEmpty()) {
            return detail;
        }
        const QString title =
            problem.value(QStringLiteral("title")).toString();
        if (!title.isEmpty()) {
            return title;
        }
    }
    if (!fallback.empty()) {
        return QString::fromStdString(fallback);
    }
    return statusCode > 0
        ? tr("Request failed (HTTP %1)").arg(statusCode)
        : tr("Request failed");
}

}  // namespace quickdesk
