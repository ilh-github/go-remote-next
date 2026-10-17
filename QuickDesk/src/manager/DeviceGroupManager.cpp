// Copyright 2026 QuickDesk Authors
#include "DeviceGroupManager.h"

#include "AuthManager.h"
#include "infra/log/log.h"

#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QSet>
#include <QUrl>
#include <QVariantMap>

namespace quickdesk {

DeviceGroupManager::DeviceGroupManager(AuthManager* authManager,
                                       QObject* parent)
    : QObject(parent)
    , m_authManager(authManager)
{
}

void DeviceGroupManager::refreshGroups()
{
    if (!m_authManager || !m_authManager->isLoggedIn()) return;

    const int generation = ++m_refreshGeneration;
    beginRequest();
    const QUrl url(m_authManager->httpBaseUrl()
                   + QStringLiteral("v1/me/device-groups"));
    m_authManager->request(
        QStringLiteral("GET"), url, QString(),
        [this, generation](int statusCode,
                           const std::string& errorMsg,
                           const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, generation, statusCode, errorMsg, data]() {
                    finishRequest();
                    if (statusCode != 200 || !errorMsg.empty()) {
                        const QString message =
                            problemMessage(data, errorMsg, statusCode);
                        setLastError(message);
                        emit requestFailed(message);
                        return;
                    }
                    if (generation != m_refreshGeneration) return;

                    const QJsonDocument document =
                        QJsonDocument::fromJson(
                            QByteArray::fromStdString(data));
                    if (!document.isObject()
                        || !document.object()
                                .value(QStringLiteral("items"))
                                .isArray()) {
                        const QString message =
                            tr("Invalid device group response");
                        setLastError(message);
                        emit requestFailed(message);
                        return;
                    }

                    QVariantList nextGroups;
                    const QJsonArray items =
                        document.object()
                            .value(QStringLiteral("items"))
                            .toArray();
                    for (const auto& value : items) {
                        if (value.isObject()) {
                            nextGroups.append(
                                normalizeGroup(value.toObject()));
                        }
                    }
                    m_groups = nextGroups;
                    setLastError(QString());
                    LOG_INFO("[DeviceGroupManager] Fetched {} groups",
                             m_groups.size());
                    emit groupsChanged();
                });
        });
}

void DeviceGroupManager::createGroup(const QString& name,
                                     const QString& description,
                                     const QString& color)
{
    if (!m_authManager || !m_authManager->isLoggedIn()) return;

    beginRequest();
    const QUrl url(m_authManager->httpBaseUrl()
                   + QStringLiteral("v1/me/device-groups"));
    m_authManager->request(
        QStringLiteral("POST"), url, groupBody(name, description, color),
        [this](int statusCode,
               const std::string& errorMsg,
               const std::string& data) {
            QMetaObject::invokeMethod(
                this, [this, statusCode, errorMsg, data]() {
                    finishRequest();
                    if (statusCode != 201 || !errorMsg.empty()) {
                        const QString message =
                            problemMessage(data, errorMsg, statusCode);
                        setLastError(message);
                        emit requestFailed(message);
                        return;
                    }
                    const QJsonDocument document =
                        QJsonDocument::fromJson(
                            QByteArray::fromStdString(data));
                    if (!document.isObject()) {
                        const QString message =
                            tr("Invalid device group response");
                        setLastError(message);
                        emit requestFailed(message);
                        return;
                    }
                    const QVariantMap group =
                        normalizeGroup(document.object());
                    const int groupId =
                        group.value(QStringLiteral("id")).toInt();
                    m_groups.append(group);
                    setLastError(QString());
                    emit groupsChanged();
                    emit groupCreated(groupId);
                });
        });
}

void DeviceGroupManager::updateGroup(int groupId,
                                     const QString& name,
                                     const QString& description,
                                     const QString& color)
{
    if (!m_authManager || !m_authManager->isLoggedIn()
        || groupId <= 0) {
        return;
    }

    beginRequest();
    const QUrl url(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/device-groups/%1").arg(groupId));
    m_authManager->request(
        QStringLiteral("PATCH"), url, groupBody(name, description, color),
        [this, groupId](int statusCode,
                        const std::string& errorMsg,
                        const std::string& data) {
            QMetaObject::invokeMethod(
                this, [this, groupId, statusCode, errorMsg, data]() {
                    finishRequest();
                    if (statusCode != 200 || !errorMsg.empty()) {
                        const QString message =
                            problemMessage(data, errorMsg, statusCode);
                        setLastError(message);
                        emit requestFailed(message);
                        return;
                    }
                    const QJsonDocument document =
                        QJsonDocument::fromJson(
                            QByteArray::fromStdString(data));
                    if (!document.isObject()) {
                        const QString message =
                            tr("Invalid device group response");
                        setLastError(message);
                        emit requestFailed(message);
                        return;
                    }
                    const QVariantMap group =
                        normalizeGroup(document.object());
                    const int index = groupListIndex(groupId);
                    if (index >= 0) {
                        m_groups[index] = group;
                    } else {
                        m_groups.append(group);
                    }
                    setLastError(QString());
                    emit groupsChanged();
                    emit groupUpdated(groupId);
                });
        });
}

void DeviceGroupManager::deleteGroup(int groupId)
{
    if (!m_authManager || !m_authManager->isLoggedIn()
        || groupId <= 0) {
        return;
    }

    beginRequest();
    const QUrl url(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/device-groups/%1").arg(groupId));
    m_authManager->request(
        QStringLiteral("DELETE"), url, QString(),
        [this, groupId](int statusCode,
                          const std::string& errorMsg,
                          const std::string& data) {
            QMetaObject::invokeMethod(
                this, [this, groupId, statusCode, errorMsg, data]() {
                    finishRequest();
                    if (statusCode != 200 || !errorMsg.empty()) {
                        const QString message =
                            problemMessage(data, errorMsg, statusCode);
                        setLastError(message);
                        emit requestFailed(message);
                        return;
                    }
                    const int index = groupListIndex(groupId);
                    if (index >= 0) {
                        m_groups.removeAt(index);
                    }
                    setLastError(QString());
                    emit groupsChanged();
                    emit groupDeleted(groupId);
                });
        });
}

void DeviceGroupManager::replaceGroupDevices(
    int groupId, const QStringList& deviceIds)
{
    if (!m_authManager || !m_authManager->isLoggedIn()
        || groupId <= 0) {
        return;
    }

    beginRequest();
    const QUrl url(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/device-groups/%1/devices").arg(groupId));
    m_authManager->request(
        QStringLiteral("PUT"), url, devicesBody(deviceIds),
        [this, groupId](int statusCode,
                          const std::string& errorMsg,
                          const std::string& data) {
            QMetaObject::invokeMethod(
                this, [this, groupId, statusCode, errorMsg, data]() {
                    finishRequest();
                    if (statusCode != 200 || !errorMsg.empty()) {
                        const QString message =
                            problemMessage(data, errorMsg, statusCode);
                        setLastError(message);
                        emit requestFailed(message);
                        return;
                    }
                    const QJsonDocument document =
                        QJsonDocument::fromJson(
                            QByteArray::fromStdString(data));
                    if (!document.isObject()) {
                        const QString message =
                            tr("Invalid device group response");
                        setLastError(message);
                        emit requestFailed(message);
                        return;
                    }
                    const QVariantMap group =
                        normalizeGroup(document.object());
                    const int index = groupListIndex(groupId);
                    if (index >= 0) {
                        m_groups[index] = group;
                    } else {
                        m_groups.append(group);
                    }
                    setLastError(QString());
                    emit groupsChanged();
                    emit groupUpdated(groupId);
                });
        });
}

void DeviceGroupManager::clear()
{
    ++m_refreshGeneration;
    const bool hadGroups = !m_groups.isEmpty();
    m_groups.clear();
    setLastError(QString());
    if (hadGroups) {
        emit groupsChanged();
    }
}

void DeviceGroupManager::beginRequest()
{
    const bool wasBusy = busy();
    ++m_pendingRequests;
    if (!wasBusy) {
        emit busyChanged();
    }
}

void DeviceGroupManager::finishRequest()
{
    if (m_pendingRequests <= 0) return;
    --m_pendingRequests;
    if (!busy()) {
        emit busyChanged();
    }
}

void DeviceGroupManager::setLastError(const QString& message)
{
    if (m_lastError == message) return;
    m_lastError = message;
    emit lastErrorChanged();
}

int DeviceGroupManager::groupListIndex(int groupId) const
{
    for (int index = 0; index < m_groups.size(); ++index) {
        if (m_groups[index]
                .toMap()
                .value(QStringLiteral("id"))
                .toInt()
            == groupId) {
            return index;
        }
    }
    return -1;
}

QVariantMap DeviceGroupManager::normalizeGroup(const QJsonObject& object)
{
    QVariantMap group = object.toVariantMap();
    group[QStringLiteral("id")] =
        object.value(QStringLiteral("id")).toInt();
    group[QStringLiteral("user_id")] =
        object.value(QStringLiteral("user_id")).toInt();
    group[QStringLiteral("name")] =
        object.value(QStringLiteral("name")).toString().trimmed();
    group[QStringLiteral("description")] =
        object.value(QStringLiteral("description")).toString().trimmed();
    group[QStringLiteral("color")] =
        object.value(QStringLiteral("color")).toString().trimmed();
    const QStringList deviceIds =
        normalizeDeviceIds(
            object.value(QStringLiteral("device_ids")).toArray());
    group[QStringLiteral("device_ids")] = deviceIds;
    group[QStringLiteral("device_count")] = deviceIds.size();
    return group;
}

QStringList DeviceGroupManager::normalizeDeviceIds(const QJsonArray& array)
{
    QStringList deviceIds;
    QSet<QString> seen;
    for (const auto& value : array) {
        const QString deviceId = value.toString().trimmed();
        if (deviceId.isEmpty() || seen.contains(deviceId)) continue;
        seen.insert(deviceId);
        deviceIds.append(deviceId);
    }
    return deviceIds;
}

QStringList DeviceGroupManager::normalizeDeviceIds(
    const QStringList& deviceIds)
{
    QStringList normalized;
    QSet<QString> seen;
    for (const QString& value : deviceIds) {
        const QString deviceId = value.trimmed();
        if (deviceId.isEmpty() || seen.contains(deviceId)) continue;
        seen.insert(deviceId);
        normalized.append(deviceId);
    }
    return normalized;
}

QString DeviceGroupManager::groupBody(const QString& name,
                                      const QString& description,
                                      const QString& color)
{
    QJsonObject body;
    body[QStringLiteral("name")] = name.trimmed();
    body[QStringLiteral("description")] = description.trimmed();
    body[QStringLiteral("color")] = color.trimmed();
    return QString::fromUtf8(
        QJsonDocument(body).toJson(QJsonDocument::Compact));
}

QString DeviceGroupManager::devicesBody(const QStringList& deviceIds)
{
    QJsonArray members;
    for (const QString& deviceId : normalizeDeviceIds(deviceIds)) {
        members.append(deviceId);
    }
    QJsonObject body;
    body[QStringLiteral("device_ids")] = members;
    return QString::fromUtf8(
        QJsonDocument(body).toJson(QJsonDocument::Compact));
}

QString DeviceGroupManager::problemMessage(const std::string& data,
                                           const std::string& fallback,
                                           int statusCode)
{
    const QJsonDocument document =
        QJsonDocument::fromJson(QByteArray::fromStdString(data));
    if (document.isObject()) {
        const QJsonObject problem = document.object();
        const QString detail =
            problem.value(QStringLiteral("detail")).toString();
        if (!detail.isEmpty()) return detail;
        const QString title =
            problem.value(QStringLiteral("title")).toString();
        if (!title.isEmpty()) return title;
    }
    if (!fallback.empty()) {
        return QString::fromStdString(fallback);
    }
    return statusCode > 0
        ? tr("Request failed (HTTP %1)").arg(statusCode)
        : tr("Request failed");
}

}  // namespace quickdesk
