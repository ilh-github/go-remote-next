// Copyright 2026 QuickDesk Authors

#include "AccessApprovalManager.h"

#include "AuthManager.h"
#include "infra/log/log.h"

#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QUrl>
#include <QUrlQuery>

namespace quickdesk {

namespace {

constexpr int kApprovalLimit = 100;
constexpr int kDeniedSourceLimit = 10;
const QStringList kApprovalStatuses = {
    QStringLiteral("pending"),
    QStringLiteral("approved"),
    QStringLiteral("denied"),
    QStringLiteral("expired"),
    QStringLiteral("consumed"),
};

QString encodedPathSegment(const QString& value)
{
    return QString::fromLatin1(QUrl::toPercentEncoding(value));
}

}  // namespace

AccessApprovalManager::AccessApprovalManager(AuthManager* authManager,
                                             QObject* parent)
    : QObject(parent)
    , m_authManager(authManager)
{
    if (m_authManager) {
        connect(m_authManager, &AuthManager::loggedOut, this, [this]() {
            ++m_generation;
            m_approvals.clear();
            m_summary.clear();
            emit approvalsChanged();
            emit summaryChanged();
        });
    }
}

void AccessApprovalManager::refresh(const QString& deviceId,
                                    const QString& statusFilter)
{
    const QString nextDeviceId = deviceId.trimmed();
    const QString nextStatus = normalizeStatusFilter(statusFilter);
    if (nextDeviceId.isEmpty()) {
        setLastError(tr("Select a device to view access approvals"));
        emit requestFailed(m_lastError);
        return;
    }

    if (m_deviceId != nextDeviceId) {
        m_deviceId = nextDeviceId;
        emit deviceIdChanged();
    }
    if (m_statusFilter != nextStatus) {
        m_statusFilter = nextStatus;
        emit statusFilterChanged();
    }

    const int generation = ++m_generation;
    setLastError(QString());
    requestList(generation);
    requestSummary(generation);
}

void AccessApprovalManager::requestList(int generation)
{
    QUrl url(m_authManager->httpBaseUrl()
             + QStringLiteral("v1/me/devices/")
             + encodedPathSegment(m_deviceId)
             + QStringLiteral("/access-approvals"));
    QUrlQuery query;
    query.addQueryItem(QStringLiteral("status"), m_statusFilter);
    query.addQueryItem(QStringLiteral("limit"),
                       QString::number(kApprovalLimit));
    url.setQuery(query);

    beginRequest();
    m_authManager->request(
        QStringLiteral("GET"), url, QString(),
        [this, generation](int statusCode,
                           const std::string& errorMsg,
                           const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, generation, statusCode, errorMsg, data]() {
                    if (generation == m_generation) {
                        if (statusCode == 200 && errorMsg.empty()) {
                            const QJsonObject root =
                                QJsonDocument::fromJson(
                                    QByteArray::fromStdString(data)).object();
                            QVariantList approvals;
                            for (const QJsonValue& value :
                                 root.value(QStringLiteral("items")).toArray()) {
                                approvals.append(
                                    normalizeApproval(value.toObject()));
                            }
                            m_approvals = approvals;
                            setLastError(QString());
                            emit approvalsChanged();
                        } else {
                            const QString message =
                                problemMessage(data, errorMsg, statusCode);
                            setLastError(message);
                            LOG_WARN("[AccessApprovalManager] list failed: status={} err={}",
                                     statusCode, message.toStdString());
                            emit requestFailed(message);
                        }
                    }
                    finishRequest();
                });
        });
}

void AccessApprovalManager::requestSummary(int generation)
{
    QUrl url(m_authManager->httpBaseUrl()
             + QStringLiteral("v1/me/devices/")
             + encodedPathSegment(m_deviceId)
             + QStringLiteral("/access-approvals:summary"));
    QUrlQuery query;
    query.addQueryItem(QStringLiteral("limit"),
                       QString::number(kDeniedSourceLimit));
    url.setQuery(query);

    beginRequest();
    m_authManager->request(
        QStringLiteral("GET"), url, QString(),
        [this, generation](int statusCode,
                           const std::string& errorMsg,
                           const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, generation, statusCode, errorMsg, data]() {
                    if (generation == m_generation) {
                        if (statusCode == 200 && errorMsg.empty()) {
                            const QJsonObject root =
                                QJsonDocument::fromJson(
                                    QByteArray::fromStdString(data)).object();
                            m_summary = normalizeSummary(root);
                            setLastError(QString());
                            emit summaryChanged();
                        } else {
                            const QString message =
                                problemMessage(data, errorMsg, statusCode);
                            setLastError(message);
                            LOG_WARN("[AccessApprovalManager] summary failed: status={} err={}",
                                     statusCode, message.toStdString());
                            emit requestFailed(message);
                        }
                    }
                    finishRequest();
                });
        });
}

void AccessApprovalManager::decide(int approvalId, bool approve,
                                   const QString& reason)
{
    if (!m_authManager || !m_authManager->isLoggedIn()) {
        const QString message = tr("Please login to decide access requests");
        setLastError(message);
        emit requestFailed(message);
        return;
    }
    if (m_deviceId.isEmpty() || approvalId <= 0) {
        const QString message = tr("Select a valid access request");
        setLastError(message);
        emit requestFailed(message);
        return;
    }

    const QUrl url(m_authManager->httpBaseUrl()
                   + QStringLiteral("v1/me/devices/")
                   + encodedPathSegment(m_deviceId)
                   + QStringLiteral("/access-approvals/")
                   + QString::number(approvalId)
                   + (approve ? QStringLiteral("/approve")
                              : QStringLiteral("/deny")));
    QJsonObject body;
    const QString trimmedReason = reason.trimmed();
    if (!trimmedReason.isEmpty()) {
        body[QStringLiteral("reason")] = trimmedReason;
    }
    const QString bodyData = body.isEmpty()
        ? QString()
        : QString::fromUtf8(
              QJsonDocument(body).toJson(QJsonDocument::Compact));

    beginRequest();
    m_authManager->request(
        QStringLiteral("POST"), url, bodyData,
        [this, approvalId](int statusCode,
                           const std::string& errorMsg,
                           const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, approvalId, statusCode, errorMsg, data]() {
                    const QJsonObject root =
                        QJsonDocument::fromJson(
                            QByteArray::fromStdString(data)).object();
                    if (statusCode == 200 && errorMsg.empty()) {
                        const QString status =
                            root.value(QStringLiteral("status")).toString();
                        for (int i = 0; i < m_approvals.size(); ++i) {
                            QVariantMap row = m_approvals[i].toMap();
                            if (row.value(QStringLiteral("id")).toInt()
                                == approvalId) {
                                row[QStringLiteral("status")] = status;
                                if (root.contains(
                                        QStringLiteral("decided_at"))) {
                                    row[QStringLiteral("decided_at")] =
                                        root.value(
                                            QStringLiteral("decided_at"))
                                            .toVariant();
                                }
                                if (root.contains(QStringLiteral("reason"))) {
                                    row[QStringLiteral("reason")] =
                                        root.value(QStringLiteral("reason"))
                                            .toString();
                                }
                                m_approvals[i] = row;
                                break;
                            }
                        }
                        setLastError(QString());
                        emit approvalsChanged();
                        emit approvalDecided(approvalId, status);
                    } else {
                        const QString currentStatus =
                            root.value(QStringLiteral("status")).toString();
                        const QString message =
                            problemMessage(data, errorMsg, statusCode);
                        setLastError(message);
                        LOG_WARN("[AccessApprovalManager] decide({}) failed: status={} current={} err={}",
                                 approvalId, statusCode,
                                 currentStatus.toStdString(),
                                 message.toStdString());
                        emit decisionFailed(approvalId, statusCode,
                                            currentStatus, message);
                        emit requestFailed(message);
                    }
                    finishRequest();
                });
        });
}

void AccessApprovalManager::beginRequest()
{
    if (m_pendingRequests++ == 0) {
        emit busyChanged();
    }
}

void AccessApprovalManager::finishRequest()
{
    if (m_pendingRequests <= 0) {
        return;
    }
    if (--m_pendingRequests == 0) {
        emit busyChanged();
    }
}

void AccessApprovalManager::setLastError(const QString& message)
{
    if (m_lastError == message) {
        return;
    }
    m_lastError = message;
    emit lastErrorChanged();
}

QString AccessApprovalManager::normalizeStatusFilter(const QString& value)
{
    QStringList accepted;
    for (const QString& part : value.split(QLatin1Char(','),
                                           Qt::SkipEmptyParts)) {
        const QString status = part.trimmed().toLower();
        if (kApprovalStatuses.contains(status) && !accepted.contains(status)) {
            accepted.append(status);
        }
    }
    return accepted.isEmpty() ? QStringLiteral("pending")
                              : accepted.join(QLatin1Char(','));
}

QVariantMap AccessApprovalManager::normalizeApproval(
    const QJsonObject& object)
{
    QVariantMap row = object.toVariantMap();
    row[QStringLiteral("id")] = object.value(QStringLiteral("id")).toInt();
    row[QStringLiteral("client_id")] =
        object.value(QStringLiteral("client_id")).toString();
    row[QStringLiteral("client_ip")] =
        object.value(QStringLiteral("client_ip")).toString();
    row[QStringLiteral("source")] =
        object.value(QStringLiteral("source")).toString();
    row[QStringLiteral("status")] =
        object.value(QStringLiteral("status")).toString();
    return row;
}

QVariantMap AccessApprovalManager::normalizeSummary(
    const QJsonObject& object)
{
    QVariantMap summary = object.toVariantMap();
    const QJsonObject wireStatuses =
        object.value(QStringLiteral("by_status")).toObject();
    QVariantMap byStatus;
    for (const QString& status : kApprovalStatuses) {
        byStatus[status] = wireStatuses.value(status).toInt();
    }
    summary[QStringLiteral("by_status")] = byStatus;
    summary[QStringLiteral("denial_cooldown_multiplier")] =
        qMax(1, object.value(
                     QStringLiteral("denial_cooldown_multiplier")).toInt());
    return summary;
}

QString AccessApprovalManager::problemMessage(const std::string& data,
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
