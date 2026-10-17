// Copyright 2026 QuickDesk Authors
#ifndef QUICKDESK_MANAGER_ACCESSAPPROVALMANAGER_H
#define QUICKDESK_MANAGER_ACCESSAPPROVALMANAGER_H

#include <QObject>
#include <QString>
#include <QVariantList>
#include <QVariantMap>

#include <string>

class QJsonArray;
class QJsonObject;

namespace quickdesk {

class AuthManager;

class AccessApprovalManager : public QObject {
    Q_OBJECT
    Q_PROPERTY(QVariantList approvals READ approvals NOTIFY approvalsChanged)
    Q_PROPERTY(QVariantMap summary READ summary NOTIFY summaryChanged)
    Q_PROPERTY(bool busy READ busy NOTIFY busyChanged)
    Q_PROPERTY(QString lastError READ lastError NOTIFY lastErrorChanged)
    Q_PROPERTY(QString deviceId READ deviceId NOTIFY deviceIdChanged)
    Q_PROPERTY(QString statusFilter READ statusFilter NOTIFY statusFilterChanged)

public:
    explicit AccessApprovalManager(AuthManager* authManager,
                                   QObject* parent = nullptr);

    QVariantList approvals() const { return m_approvals; }
    QVariantMap summary() const { return m_summary; }
    bool busy() const { return m_pendingRequests > 0; }
    QString lastError() const { return m_lastError; }
    QString deviceId() const { return m_deviceId; }
    QString statusFilter() const { return m_statusFilter; }

    Q_INVOKABLE void refresh(const QString& deviceId,
                             const QString& statusFilter =
                                 QStringLiteral("pending"));
    Q_INVOKABLE void decide(int approvalId, bool approve,
                            const QString& reason = QString());

signals:
    void approvalsChanged();
    void summaryChanged();
    void busyChanged();
    void lastErrorChanged();
    void deviceIdChanged();
    void statusFilterChanged();
    void requestFailed(const QString& message);
    void approvalDecided(int approvalId, const QString& status);
    void decisionFailed(int approvalId, int httpStatus,
                        const QString& currentStatus,
                        const QString& message);

private:
    void requestList(int generation);
    void requestSummary(int generation);
    void beginRequest();
    void finishRequest();
    void setLastError(const QString& message);
    static QString normalizeStatusFilter(const QString& value);
    static QVariantMap normalizeApproval(const QJsonObject& object);
    static QVariantMap normalizeSummary(const QJsonObject& object);
    static QString problemMessage(const std::string& data,
                                  const std::string& fallback,
                                  int statusCode);

    AuthManager* m_authManager = nullptr;
    QVariantList m_approvals;
    QVariantMap m_summary;
    QString m_lastError;
    QString m_deviceId;
    QString m_statusFilter = QStringLiteral("pending");
    int m_pendingRequests = 0;
    int m_generation = 0;
};

}  // namespace quickdesk

#endif  // QUICKDESK_MANAGER_ACCESSAPPROVALMANAGER_H
