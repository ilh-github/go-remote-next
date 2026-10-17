// Copyright 2026 QuickDesk Authors
#ifndef QUICKDESK_MANAGER_REMOTESESSIONMANAGER_H
#define QUICKDESK_MANAGER_REMOTESESSIONMANAGER_H

#include <QObject>
#include <QString>
#include <QVariantList>
#include <QVariantMap>

#include <string>

class QJsonArray;
class QJsonObject;

namespace quickdesk {

class AuthManager;

/**
 * User-scoped control-session history and revocation.
 *
 * The list endpoint returns only the session rows; expanding a row asks the
 * detail endpoint for the recordings and access-log entries that share its
 * session id. Rows remain plain QVariantMaps so the QML page can render both
 * the list response and the joined detail response without another model.
 */
class RemoteSessionManager : public QObject {
    Q_OBJECT
    Q_PROPERTY(QVariantList sessions READ sessions NOTIFY sessionsChanged)
    Q_PROPERTY(bool busy READ busy NOTIFY busyChanged)
    Q_PROPERTY(bool hasMore READ hasMore NOTIFY hasMoreChanged)
    Q_PROPERTY(QString lastError READ lastError NOTIFY lastErrorChanged)

public:
    explicit RemoteSessionManager(AuthManager* authManager, QObject* parent = nullptr);

    QVariantList sessions() const { return m_sessions; }
    bool busy() const { return m_pendingRequests > 0; }
    bool hasMore() const { return !m_nextCursor.isEmpty(); }
    QString lastError() const { return m_lastError; }

    Q_INVOKABLE void refresh(const QString& status = QStringLiteral("active"),
                             const QString& deviceId = QString());
    Q_INVOKABLE void loadMore();
    Q_INVOKABLE void fetchDetail(const QString& sessionId);
    Q_INVOKABLE void revokeSession(const QString& sessionId);
    Q_INVOKABLE void revokeDeviceSessions(const QString& deviceId);
    Q_INVOKABLE void banSource(const QString& deviceId, const QString& source);

signals:
    void sessionsChanged();
    void busyChanged();
    void hasMoreChanged();
    void lastErrorChanged();
    void requestFailed(const QString& message);
    void detailLoaded(const QString& sessionId);
    void sessionRevoked(const QString& sessionId);
    void deviceSessionsRevoked(const QString& deviceId, int revoked, int failed,
                               bool truncated);
    void sourceBanned(const QString& deviceId, const QString& source);

private:
    void requestList(bool append, int generation);
    void finishRequest();
    void beginRequest();
    void setLastError(const QString& message);
    int rowForSession(const QString& sessionId) const;

    static QVariantMap normalizeSession(const QJsonObject& object);
    static QVariantMap normalizeDetail(const QJsonObject& object);
    static QVariantList objectArrayToVariantList(const QJsonArray& array);
    static QString problemMessage(const std::string& data,
                                  const std::string& fallback,
                                  int statusCode);

    AuthManager* m_authManager = nullptr;
    QVariantList m_sessions;
    QString m_statusFilter = QStringLiteral("active");
    QString m_deviceIdFilter;
    QString m_nextCursor;
    QString m_lastError;
    int m_pendingRequests = 0;
    int m_listGeneration = 0;
};

} // namespace quickdesk

#endif // QUICKDESK_MANAGER_REMOTESESSIONMANAGER_H
