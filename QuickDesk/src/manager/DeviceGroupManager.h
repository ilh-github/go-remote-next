// Copyright 2026 QuickDesk Authors
#ifndef QUICKDESK_MANAGER_DEVICEGROUPMANAGER_H
#define QUICKDESK_MANAGER_DEVICEGROUPMANAGER_H

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

/**
 * DeviceGroupManager owns the current user's private address-book groups.
 *
 * It deliberately talks only to /v1/me/device-groups. The administrator
 * inventory under /v1/admin/groups is a different, global model and must not
 * leak into the user-facing device list.
 */
class DeviceGroupManager : public QObject {
    Q_OBJECT
    Q_PROPERTY(QVariantList groups READ groups NOTIFY groupsChanged)
    Q_PROPERTY(bool busy READ busy NOTIFY busyChanged)
    Q_PROPERTY(QString lastError READ lastError NOTIFY lastErrorChanged)

public:
    explicit DeviceGroupManager(AuthManager* authManager,
                                QObject* parent = nullptr);

    QVariantList groups() const { return m_groups; }
    bool busy() const { return m_pendingRequests > 0; }
    QString lastError() const { return m_lastError; }

    Q_INVOKABLE void refreshGroups();
    Q_INVOKABLE void createGroup(const QString& name,
                                 const QString& description = QString(),
                                 const QString& color = QString());
    Q_INVOKABLE void updateGroup(int groupId,
                                 const QString& name,
                                 const QString& description = QString(),
                                 const QString& color = QString());
    Q_INVOKABLE void deleteGroup(int groupId);
    Q_INVOKABLE void replaceGroupDevices(int groupId,
                                         const QStringList& deviceIds);
    void clear();

signals:
    void groupsChanged();
    void busyChanged();
    void lastErrorChanged();
    void requestFailed(const QString& message);
    void groupCreated(int groupId);
    void groupUpdated(int groupId);
    void groupDeleted(int groupId);

private:
    void beginRequest();
    void finishRequest();
    void setLastError(const QString& message);
    int groupListIndex(int groupId) const;
    static QVariantMap normalizeGroup(const QJsonObject& object);
    static QStringList normalizeDeviceIds(const QJsonArray& array);
    static QStringList normalizeDeviceIds(const QStringList& deviceIds);
    static QString groupBody(const QString& name,
                             const QString& description,
                             const QString& color);
    static QString devicesBody(const QStringList& deviceIds);
    static QString problemMessage(const std::string& data,
                                  const std::string& fallback,
                                  int statusCode);

    AuthManager* m_authManager = nullptr;
    QVariantList m_groups;
    QString m_lastError;
    int m_pendingRequests = 0;
    int m_refreshGeneration = 0;
};

}  // namespace quickdesk

#endif  // QUICKDESK_MANAGER_DEVICEGROUPMANAGER_H
