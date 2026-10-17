// Copyright 2026 QuickDesk Authors

#include "NotificationManager.h"

#include "AuthManager.h"
#include "infra/log/log.h"

#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QUrl>
#include <QUrlQuery>

#include <algorithm>

namespace quickdesk {

NotificationManager::NotificationManager(AuthManager* authManager,
                                         QObject* parent)
    : QObject(parent)
    , m_authManager(authManager)
{
    if (m_authManager) {
        connect(m_authManager, &AuthManager::loggedOut, this, [this]() {
            ++m_channelsGeneration;
            ++m_quietGeneration;
            ++m_offlineGeneration;
            ++m_metricAlertGeneration;
            ++m_deliveryGeneration;
            m_channels.clear();
            m_supportedEvents.clear();
            m_signatureHeader.clear();
            m_signatureScheme.clear();
            m_quietHours.clear();
            m_offlineAlert.clear();
            m_offlineDeviceId.clear();
            m_metricAlert.clear();
            m_metricAlertDeviceId.clear();
            m_metricCurrentValue.clear();
            m_deliveries.clear();
            m_unackedDeliveries = 0;
            m_deliveryChannelId = 0;
            m_lastRequeuedCount = 0;
            emit channelsChanged();
            emit supportedEventsChanged();
            emit signatureInfoChanged();
            emit quietHoursChanged();
            emit offlineAlertChanged();
            emit offlineDeviceIdChanged();
            emit metricAlertChanged();
            emit metricAlertDeviceIdChanged();
            emit metricCurrentValueChanged();
            emit deliveriesChanged();
            emit deliveryChannelIdChanged();
            emit lastRequeuedCountChanged();
        });
    }
}

void NotificationManager::refreshChannels()
{
    if (!m_authManager || !m_authManager->isLoggedIn()) {
        const QString message =
            tr("Please login to manage notification channels");
        setLastError(message);
        emit requestFailed(message);
        return;
    }

    const int generation = ++m_channelsGeneration;
    const QUrl url(m_authManager->httpBaseUrl()
                   + QStringLiteral("v1/me/notification-channels"));
    beginRequest();
    m_authManager->request(
        QStringLiteral("GET"), url, QString(),
        [this, generation](int statusCode, const std::string& errorMsg,
                           const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, generation, statusCode, errorMsg, data]() {
                    if (generation == m_channelsGeneration) {
                        if (statusCode == 200 && errorMsg.empty()) {
                            const QJsonObject root =
                                QJsonDocument::fromJson(
                                    QByteArray::fromStdString(data)).object();
                            QVariantList rows;
                            for (const QJsonValue& value :
                                 root.value(QStringLiteral("items")).toArray()) {
                                rows.append(
                                    normalizeChannel(value.toObject()));
                            }
                            m_channels = rows;
                            m_supportedEvents = stringList(
                                root.value(
                                    QStringLiteral("supported_events")).toArray());
                            m_signatureHeader =
                                root.value(
                                    QStringLiteral("signature_header")).toString();
                            m_signatureScheme =
                                root.value(
                                    QStringLiteral("signature_scheme")).toString();
                            setLastError(QString());
                            emit channelsChanged();
                            emit supportedEventsChanged();
                            emit signatureInfoChanged();
                        } else {
                            const QString message =
                                problemMessage(data, errorMsg, statusCode);
                            setLastError(message);
                            LOG_WARN("[NotificationManager] channel list failed: status={} err={}",
                                     statusCode, message.toStdString());
                            emit requestFailed(message);
                        }
                    }
                    finishRequest();
                });
        });
}

void NotificationManager::createChannel(const QString& name,
                                        const QString& url,
                                        const QString& secret,
                                        const QStringList& events,
                                        bool enabled)
{
    if (!m_authManager || !m_authManager->isLoggedIn()) {
        const QString message =
            tr("Please login to create notification channels");
        setLastError(message);
        emit requestFailed(message);
        return;
    }
    const QStringList selectedEvents = normalizedEvents(events);
    if (url.trimmed().isEmpty() || selectedEvents.isEmpty()) {
        const QString message =
            tr("Enter a webhook URL and select at least one event");
        setLastError(message);
        emit requestFailed(message);
        return;
    }

    const QUrl endpoint(m_authManager->httpBaseUrl()
                        + QStringLiteral("v1/me/notification-channels"));
    beginRequest();
    m_authManager->request(
        QStringLiteral("POST"), endpoint,
        channelBody(name, url, secret, selectedEvents, enabled),
        [this](int statusCode, const std::string& errorMsg,
               const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, statusCode, errorMsg, data]() {
                    if (statusCode == 201 && errorMsg.empty()) {
                        const QVariantMap row = normalizeChannel(
                            QJsonDocument::fromJson(
                                QByteArray::fromStdString(data)).object());
                        m_channels.append(row);
                        setLastError(QString());
                        emit channelsChanged();
                        emit channelCreated(
                            row.value(QStringLiteral("id")).toInt());
                    } else {
                        const QString message =
                            problemMessage(data, errorMsg, statusCode);
                        setLastError(message);
                        LOG_WARN("[NotificationManager] channel create failed: status={} err={}",
                                 statusCode, message.toStdString());
                        emit requestFailed(message);
                    }
                    finishRequest();
                });
        });
}

void NotificationManager::updateChannel(int channelId,
                                        const QString& name,
                                        const QString& url,
                                        const QString& secret,
                                        const QStringList& events,
                                        bool enabled)
{
    if (!m_authManager || !m_authManager->isLoggedIn() || channelId <= 0) {
        return;
    }
    const QStringList selectedEvents = normalizedEvents(events);
    if (url.trimmed().isEmpty() || selectedEvents.isEmpty()) {
        const QString message =
            tr("Enter a webhook URL and select at least one event");
        setLastError(message);
        emit requestFailed(message);
        return;
    }

    const QUrl endpoint(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/notification-channels/")
        + QString::number(channelId));
    beginRequest();
    m_authManager->request(
        QStringLiteral("PATCH"), endpoint,
        channelBody(name, url, secret, selectedEvents, enabled),
        [this, channelId](int statusCode, const std::string& errorMsg,
                          const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, channelId, statusCode, errorMsg, data]() {
                    if (statusCode == 200 && errorMsg.empty()) {
                        const QVariantMap row = normalizeChannel(
                            QJsonDocument::fromJson(
                                QByteArray::fromStdString(data)).object());
                        bool replaced = false;
                        for (int i = 0; i < m_channels.size(); ++i) {
                            if (m_channels.at(i).toMap()
                                    .value(QStringLiteral("id")).toInt()
                                == channelId) {
                                m_channels[i] = row;
                                replaced = true;
                                break;
                            }
                        }
                        if (!replaced) {
                            m_channels.append(row);
                        }
                        setLastError(QString());
                        emit channelsChanged();
                        emit channelUpdated(channelId);
                    } else {
                        const QString message =
                            problemMessage(data, errorMsg, statusCode);
                        setLastError(message);
                        LOG_WARN("[NotificationManager] channel update({}) failed: status={} err={}",
                                 channelId, statusCode, message.toStdString());
                        emit requestFailed(message);
                    }
                    finishRequest();
                });
        });
}

void NotificationManager::deleteChannel(int channelId)
{
    if (!m_authManager || !m_authManager->isLoggedIn() || channelId <= 0) {
        return;
    }
    const QUrl endpoint(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/notification-channels/")
        + QString::number(channelId));
    beginRequest();
    m_authManager->request(
        QStringLiteral("DELETE"), endpoint, QString(),
        [this, channelId](int statusCode, const std::string& errorMsg,
                          const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, channelId, statusCode, errorMsg, data]() {
                    if (statusCode == 200 && errorMsg.empty()) {
                        for (int i = 0; i < m_channels.size(); ++i) {
                            if (m_channels.at(i).toMap()
                                    .value(QStringLiteral("id")).toInt()
                                == channelId) {
                                m_channels.removeAt(i);
                                break;
                            }
                        }
                        setLastError(QString());
                        emit channelsChanged();
                        emit channelDeleted(channelId);
                    } else {
                        const QString message =
                            problemMessage(data, errorMsg, statusCode);
                        setLastError(message);
                        LOG_WARN("[NotificationManager] channel delete({}) failed: status={} err={}",
                                 channelId, statusCode, message.toStdString());
                        emit requestFailed(message);
                    }
                    finishRequest();
                });
        });
}

void NotificationManager::testChannel(int channelId)
{
    if (!m_authManager || !m_authManager->isLoggedIn() || channelId <= 0) {
        return;
    }
    const QUrl endpoint(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/notification-channels/")
        + QString::number(channelId)
        + QStringLiteral("/test"));
    beginRequest();
    m_authManager->request(
        QStringLiteral("POST"), endpoint, QString(),
        [this, channelId](int statusCode, const std::string& errorMsg,
                          const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, channelId, statusCode, errorMsg, data]() {
                    if (statusCode == 202 && errorMsg.empty()) {
                        setLastError(QString());
                        emit channelTestQueued(channelId);
                    } else {
                        const QString message =
                            problemMessage(data, errorMsg, statusCode);
                        setLastError(message);
                        LOG_WARN("[NotificationManager] channel test({}) failed: status={} err={}",
                                 channelId, statusCode, message.toStdString());
                        emit requestFailed(message);
                    }
                    finishRequest();
                });
        });
}

void NotificationManager::loadQuietHours()
{
    if (!m_authManager || !m_authManager->isLoggedIn()) {
        const QString message =
            tr("Please login to manage notification quiet hours");
        setLastError(message);
        emit requestFailed(message);
        return;
    }

    const int generation = ++m_quietGeneration;
    const QUrl url(m_authManager->httpBaseUrl()
                   + QStringLiteral("v1/me/notification-quiet-hours"));
    beginRequest();
    m_authManager->request(
        QStringLiteral("GET"), url, QString(),
        [this, generation](int statusCode, const std::string& errorMsg,
                           const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, generation, statusCode, errorMsg, data]() {
                    if (generation == m_quietGeneration) {
                        if (statusCode == 200 && errorMsg.empty()) {
                            const QJsonObject root =
                                QJsonDocument::fromJson(
                                    QByteArray::fromStdString(data)).object();
                            m_quietHours = normalizeQuietHours(root);
                            setLastError(QString());
                            emit quietHoursChanged();
                        } else {
                            const QString message =
                                problemMessage(data, errorMsg, statusCode);
                            setLastError(message);
                            LOG_WARN("[NotificationManager] quiet-hours load failed: status={} err={}",
                                     statusCode, message.toStdString());
                            emit requestFailed(message);
                        }
                    }
                    finishRequest();
                });
        });
}

void NotificationManager::saveQuietHours(const QString& snoozedUntil,
                                         const QVariantList& windows,
                                         const QString& timezone)
{
    if (!m_authManager || !m_authManager->isLoggedIn()) {
        const QString message =
            tr("Please login to manage notification quiet hours");
        setLastError(message);
        emit requestFailed(message);
        return;
    }

    const int generation = ++m_quietGeneration;
    const QUrl url(m_authManager->httpBaseUrl()
                   + QStringLiteral("v1/me/notification-quiet-hours"));
    beginRequest();
    m_authManager->request(
        QStringLiteral("PUT"), url,
        quietHoursBody(snoozedUntil, windows, timezone),
        [this, generation](int statusCode, const std::string& errorMsg,
                           const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, generation, statusCode, errorMsg, data]() {
                    if (generation == m_quietGeneration) {
                        if (statusCode == 200 && errorMsg.empty()) {
                            const QJsonObject root =
                                QJsonDocument::fromJson(
                                    QByteArray::fromStdString(data)).object();
                            m_quietHours = normalizeQuietHours(root);
                            setLastError(QString());
                            emit quietHoursChanged();
                        } else {
                            const QString message =
                                problemMessage(data, errorMsg, statusCode);
                            setLastError(message);
                            LOG_WARN("[NotificationManager] quiet-hours save failed: status={} err={}",
                                     statusCode, message.toStdString());
                            emit requestFailed(message);
                        }
                    }
                    finishRequest();
                });
        });
}

void NotificationManager::loadOfflineAlert(const QString& deviceId)
{
    const QString normalizedDeviceId = deviceId.trimmed();
    if (m_offlineDeviceId != normalizedDeviceId) {
        m_offlineDeviceId = normalizedDeviceId;
        emit offlineDeviceIdChanged();
    }
    if (!m_authManager || !m_authManager->isLoggedIn()
        || normalizedDeviceId.isEmpty()) {
        const QString message =
            normalizedDeviceId.isEmpty()
                ? tr("Select a device to configure offline alerts")
                : tr("Please login to manage offline alerts");
        setLastError(message);
        emit requestFailed(message);
        return;
    }

    const int generation = ++m_offlineGeneration;
    const QUrl url(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/devices/")
        + QString::fromLatin1(
            QUrl::toPercentEncoding(normalizedDeviceId))
        + QStringLiteral("/offline-alert"));
    beginRequest();
    m_authManager->request(
        QStringLiteral("GET"), url, QString(),
        [this, generation](int statusCode, const std::string& errorMsg,
                           const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, generation, statusCode, errorMsg, data]() {
                    if (generation == m_offlineGeneration) {
                        if (statusCode == 200 && errorMsg.empty()) {
                            const QJsonObject root =
                                QJsonDocument::fromJson(
                                    QByteArray::fromStdString(data)).object();
                            m_offlineAlert = normalizeOfflineAlert(root);
                            setLastError(QString());
                            emit offlineAlertChanged();
                        } else {
                            const QString message =
                                problemMessage(data, errorMsg, statusCode);
                            setLastError(message);
                            LOG_WARN("[NotificationManager] offline-alert load failed: status={} err={}",
                                     statusCode, message.toStdString());
                            emit requestFailed(message);
                        }
                    }
                    finishRequest();
                });
        });
}

void NotificationManager::saveOfflineAlert(
    bool enabled, int afterMinutes, int repeatMinutes,
    const QString& snoozedUntil, const QVariantList& maintenanceWindows,
    const QString& maintenanceTimezone)
{
    if (!m_authManager || !m_authManager->isLoggedIn()
        || m_offlineDeviceId.isEmpty()) {
        const QString message = tr("Select a device first");
        setLastError(message);
        emit requestFailed(message);
        return;
    }
    if (afterMinutes < 1 || afterMinutes > 10080
        || (repeatMinutes != 0
            && (repeatMinutes < 1 || repeatMinutes > 10080))) {
        const QString message =
            tr("Offline alert thresholds must be between 1 and 10080 minutes");
        setLastError(message);
        emit requestFailed(message);
        return;
    }

    const int generation = ++m_offlineGeneration;
    const QUrl url(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/devices/")
        + QString::fromLatin1(
            QUrl::toPercentEncoding(m_offlineDeviceId))
        + QStringLiteral("/offline-alert"));
    beginRequest();
    m_authManager->request(
        QStringLiteral("PUT"), url,
        offlineAlertBody(enabled, afterMinutes, repeatMinutes,
                         snoozedUntil, maintenanceWindows,
                         maintenanceTimezone),
        [this, generation](int statusCode, const std::string& errorMsg,
                           const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, generation, statusCode, errorMsg, data]() {
                    if (generation == m_offlineGeneration) {
                        if (statusCode == 200 && errorMsg.empty()) {
                            const QJsonObject root =
                                QJsonDocument::fromJson(
                                    QByteArray::fromStdString(data)).object();
                            m_offlineAlert = normalizeOfflineAlert(root);
                            setLastError(QString());
                            emit offlineAlertChanged();
                        } else {
                            const QString message =
                                problemMessage(data, errorMsg, statusCode);
                            setLastError(message);
                            LOG_WARN("[NotificationManager] offline-alert save failed: status={} err={}",
                                     statusCode, message.toStdString());
                            emit requestFailed(message);
                        }
                    }
                    finishRequest();
                });
        });
}

void NotificationManager::removeOfflineAlert()
{
    if (!m_authManager || !m_authManager->isLoggedIn()
        || m_offlineDeviceId.isEmpty()) {
        return;
    }
    const int generation = ++m_offlineGeneration;
    const QUrl url(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/devices/")
        + QString::fromLatin1(
            QUrl::toPercentEncoding(m_offlineDeviceId))
        + QStringLiteral("/offline-alert"));
    beginRequest();
    m_authManager->request(
        QStringLiteral("DELETE"), url, QString(),
        [this, generation](int statusCode, const std::string& errorMsg,
                           const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, generation, statusCode, errorMsg, data]() {
                    if (generation == m_offlineGeneration) {
                        if (statusCode == 204 && errorMsg.empty()) {
                            m_offlineAlert = QVariantMap{
                                {QStringLiteral("device_id"),
                                 m_offlineDeviceId},
                                {QStringLiteral("enabled"), false},
                                {QStringLiteral("after_minutes"), 10},
                                {QStringLiteral("repeat_minutes"), 0},
                                {QStringLiteral("maintenance_windows"),
                                 QVariantList{}},
                                {QStringLiteral("maintenance_timezone"),
                                 QString()},
                            };
                            setLastError(QString());
                            emit offlineAlertChanged();
                        } else {
                            const QString message =
                                problemMessage(data, errorMsg, statusCode);
                            setLastError(message);
                            LOG_WARN("[NotificationManager] offline-alert delete failed: status={} err={}",
                                     statusCode, message.toStdString());
                            emit requestFailed(message);
                        }
                    }
                    finishRequest();
                });
        });
}

void NotificationManager::loadMetricAlert(const QString& deviceId)
{
    const QString normalizedDeviceId = deviceId.trimmed();
    if (m_metricAlertDeviceId != normalizedDeviceId) {
        m_metricAlertDeviceId = normalizedDeviceId;
        m_metricCurrentValue.clear();
        emit metricAlertDeviceIdChanged();
        emit metricCurrentValueChanged();
    }
    if (!m_authManager || !m_authManager->isLoggedIn()
        || normalizedDeviceId.isEmpty()) {
        const QString message =
            normalizedDeviceId.isEmpty()
                ? tr("Select a device to configure metric alerts")
                : tr("Please login to manage metric alerts");
        setLastError(message);
        emit requestFailed(message);
        return;
    }

    const int generation = ++m_metricAlertGeneration;
    const QUrl url(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/devices/")
        + QString::fromLatin1(QUrl::toPercentEncoding(normalizedDeviceId))
        + QStringLiteral("/metric-alert"));
    beginRequest();
    m_authManager->request(
        QStringLiteral("GET"), url, QString(),
        [this, generation](int statusCode, const std::string& errorMsg,
                           const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, generation, statusCode, errorMsg, data]() {
                    if (generation == m_metricAlertGeneration) {
                        if (statusCode == 200 && errorMsg.empty()) {
                            const QJsonObject root =
                                QJsonDocument::fromJson(
                                    QByteArray::fromStdString(data)).object();
                            m_metricAlert = normalizeMetricAlert(root);
                            if (root.value(QStringLiteral("current")).isObject()) {
                                m_metricCurrentValue =
                                    root.value(QStringLiteral("current"))
                                        .toObject()
                                        .toVariantMap();
                            } else {
                                m_metricCurrentValue.clear();
                            }
                            emit metricCurrentValueChanged();
                            setLastError(QString());
                            emit metricAlertChanged();
                        } else {
                            const QString message =
                                problemMessage(data, errorMsg, statusCode);
                            setLastError(message);
                            LOG_WARN("[NotificationManager] metric-alert load failed: status={} err={}",
                                     statusCode, message.toStdString());
                            emit requestFailed(message);
                        }
                    }
                    finishRequest();
                });
        });
}

void NotificationManager::saveMetricAlert(
    bool enabled, int cpuPercent, int memoryPercent, int diskPercent,
    int sustainedMinutes, int repeatMinutes, const QString& snoozedUntil)
{
    if (!m_authManager || !m_authManager->isLoggedIn()
        || m_metricAlertDeviceId.isEmpty()) {
        const QString message = tr("Select a device first");
        setLastError(message);
        emit requestFailed(message);
        return;
    }
    if (cpuPercent < 0 || cpuPercent > 100
        || memoryPercent < 0 || memoryPercent > 100
        || diskPercent < 0 || diskPercent > 100) {
        const QString message =
            tr("Metric thresholds must be between 0 and 100 percent");
        setLastError(message);
        emit requestFailed(message);
        return;
    }
    if (enabled && cpuPercent == 0 && memoryPercent == 0 && diskPercent == 0) {
        const QString message =
            tr("Enable at least one metric threshold");
        setLastError(message);
        emit requestFailed(message);
        return;
    }
    if (sustainedMinutes < 0 || sustainedMinutes > 10080
        || (repeatMinutes != 0
            && (repeatMinutes < 1 || repeatMinutes > 10080))) {
        const QString message =
            tr("Metric alert intervals must be between 0 and 10080 minutes");
        setLastError(message);
        emit requestFailed(message);
        return;
    }

    const int generation = ++m_metricAlertGeneration;
    const QUrl url(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/devices/")
        + QString::fromLatin1(
            QUrl::toPercentEncoding(m_metricAlertDeviceId))
        + QStringLiteral("/metric-alert"));
    beginRequest();
    m_authManager->request(
        QStringLiteral("PUT"), url,
        metricAlertBody(enabled, cpuPercent, memoryPercent, diskPercent,
                        sustainedMinutes, repeatMinutes, snoozedUntil),
        [this, generation](int statusCode, const std::string& errorMsg,
                           const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, generation, statusCode, errorMsg, data]() {
                    if (generation == m_metricAlertGeneration) {
                        if (statusCode == 200 && errorMsg.empty()) {
                            const QJsonObject root =
                                QJsonDocument::fromJson(
                                    QByteArray::fromStdString(data)).object();
                            m_metricAlert = normalizeMetricAlert(root);
                            const QJsonObject current =
                                root.value(QStringLiteral("current")).toObject();
                            if (!current.isEmpty()) {
                                m_metricCurrentValue = current.toVariantMap();
                            } else {
                                m_metricCurrentValue.clear();
                            }
                            emit metricCurrentValueChanged();
                            setLastError(QString());
                            emit metricAlertChanged();
                        } else {
                            const QString message =
                                problemMessage(data, errorMsg, statusCode);
                            setLastError(message);
                            LOG_WARN("[NotificationManager] metric-alert save failed: status={} err={}",
                                     statusCode, message.toStdString());
                            emit requestFailed(message);
                        }
                    }
                    finishRequest();
                });
        });
}

void NotificationManager::removeMetricAlert()
{
    if (!m_authManager || !m_authManager->isLoggedIn()
        || m_metricAlertDeviceId.isEmpty()) {
        return;
    }
    const int generation = ++m_metricAlertGeneration;
    const QUrl url(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/devices/")
        + QString::fromLatin1(
            QUrl::toPercentEncoding(m_metricAlertDeviceId))
        + QStringLiteral("/metric-alert"));
    beginRequest();
    m_authManager->request(
        QStringLiteral("DELETE"), url, QString(),
        [this, generation](int statusCode, const std::string& errorMsg,
                           const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, generation, statusCode, errorMsg, data]() {
                    if (generation == m_metricAlertGeneration) {
                        if (statusCode == 204 && errorMsg.empty()) {
                            m_metricAlert = QVariantMap{
                                {QStringLiteral("device_id"),
                                 m_metricAlertDeviceId},
                                {QStringLiteral("enabled"), false},
                                {QStringLiteral("cpu_percent"), 0},
                                {QStringLiteral("memory_percent"), 0},
                                {QStringLiteral("disk_percent"), 0},
                                {QStringLiteral("sustained_minutes"), 5},
                                {QStringLiteral("repeat_minutes"), 0},
                            };
                            setLastError(QString());
                            emit metricAlertChanged();
                        } else {
                            const QString message =
                                problemMessage(data, errorMsg, statusCode);
                            setLastError(message);
                            LOG_WARN("[NotificationManager] metric-alert delete failed: status={} err={}",
                                     statusCode, message.toStdString());
                            emit requestFailed(message);
                        }
                    }
                    finishRequest();
                });
        });
}

void NotificationManager::loadDeliveries(int channelId)
{
    if (!m_authManager || !m_authManager->isLoggedIn() || channelId <= 0) {
        return;
    }
    if (m_deliveryChannelId != channelId) {
        m_deliveryChannelId = channelId;
        emit deliveryChannelIdChanged();
    }
    const int generation = ++m_deliveryGeneration;
    QUrl url(m_authManager->httpBaseUrl()
             + QStringLiteral("v1/me/notification-channels/")
             + QString::number(channelId)
             + QStringLiteral("/deliveries"));
    QUrlQuery query;
    query.addQueryItem(QStringLiteral("limit"), QStringLiteral("100"));
    url.setQuery(query);

    beginRequest();
    m_authManager->request(
        QStringLiteral("GET"), url, QString(),
        [this, generation](int statusCode, const std::string& errorMsg,
                           const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, generation, statusCode, errorMsg, data]() {
                    if (generation == m_deliveryGeneration) {
                        if (statusCode == 200 && errorMsg.empty()) {
                            const QJsonObject root =
                                QJsonDocument::fromJson(
                                    QByteArray::fromStdString(data)).object();
                            QVariantList rows;
                            for (const QJsonValue& value :
                                 root.value(QStringLiteral("items")).toArray()) {
                                rows.append(value.toObject().toVariantMap());
                            }
                            m_deliveries = rows;
                            m_unackedDeliveries =
                                qMax(0, root.value(
                                            QStringLiteral("unacked"))
                                            .toInt());
                            setLastError(QString());
                            emit deliveriesChanged();
                        } else {
                            const QString message =
                                problemMessage(data, errorMsg, statusCode);
                            setLastError(message);
                            LOG_WARN("[NotificationManager] delivery list failed: status={} err={}",
                                     statusCode, message.toStdString());
                            emit requestFailed(message);
                        }
                    }
                    finishRequest();
                });
        });
}

void NotificationManager::retryDelivery(int channelId, int deliveryId)
{
    if (!m_authManager || !m_authManager->isLoggedIn()
        || channelId <= 0 || deliveryId <= 0) {
        return;
    }
    const QUrl url(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/notification-channels/")
        + QString::number(channelId)
        + QStringLiteral("/deliveries/")
        + QString::number(deliveryId)
        + QStringLiteral("/retry"));
    beginRequest();
    m_authManager->request(
        QStringLiteral("POST"), url, QString(),
        [this, deliveryId](int statusCode, const std::string& errorMsg,
                           const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, deliveryId, statusCode, errorMsg, data]() {
                    if (statusCode == 200 && errorMsg.empty()) {
                        const QVariantMap row =
                            QJsonDocument::fromJson(
                                QByteArray::fromStdString(data))
                                .object()
                                .toVariantMap();
                        for (int i = 0; i < m_deliveries.size(); ++i) {
                            if (m_deliveries.at(i).toMap()
                                    .value(QStringLiteral("id")).toInt()
                                == deliveryId) {
                                m_deliveries[i] = row;
                                break;
                            }
                        }
                        setLastError(QString());
                        emit deliveriesChanged();
                    } else {
                        const QString message =
                            problemMessage(data, errorMsg, statusCode);
                        setLastError(message);
                        emit requestFailed(message);
                    }
                    finishRequest();
                });
        });
}

void NotificationManager::ackDelivery(int channelId, int deliveryId)
{
    if (!m_authManager || !m_authManager->isLoggedIn()
        || channelId <= 0 || deliveryId <= 0) {
        return;
    }
    const QUrl url(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/notification-channels/")
        + QString::number(channelId)
        + QStringLiteral("/deliveries/")
        + QString::number(deliveryId)
        + QStringLiteral("/ack"));
    beginRequest();
    m_authManager->request(
        QStringLiteral("POST"), url, QString(),
        [this, deliveryId](int statusCode, const std::string& errorMsg,
                           const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, deliveryId, statusCode, errorMsg, data]() {
                    if (statusCode == 200 && errorMsg.empty()) {
                        const QVariantMap row =
                            QJsonDocument::fromJson(
                                QByteArray::fromStdString(data))
                                .object()
                                .toVariantMap();
                        for (int i = 0; i < m_deliveries.size(); ++i) {
                            QVariantMap current = m_deliveries[i].toMap();
                            if (current.value(QStringLiteral("id")).toInt()
                                == deliveryId) {
                                const bool wasUnacked =
                                    current.value(
                                        QStringLiteral("status")).toString()
                                        == QStringLiteral("delivered")
                                    && current.value(
                                           QStringLiteral("acked_at"))
                                           .isNull();
                                m_deliveries[i] = row;
                                if (wasUnacked
                                    && !row.value(
                                            QStringLiteral("acked_at"))
                                            .isNull()) {
                                    m_unackedDeliveries =
                                        qMax(0, m_unackedDeliveries - 1);
                                }
                                break;
                            }
                        }
                        setLastError(QString());
                        emit deliveriesChanged();
                    } else {
                        const QString message =
                            problemMessage(data, errorMsg, statusCode);
                        setLastError(message);
                        emit requestFailed(message);
                    }
                    finishRequest();
                });
        });
}

void NotificationManager::retryFailedDeliveries(int channelId)
{
    if (!m_authManager || !m_authManager->isLoggedIn() || channelId <= 0) {
        return;
    }
    const QUrl url(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/notification-channels/")
        + QString::number(channelId)
        + QStringLiteral("/deliveries:retry-failed"));
    beginRequest();
    m_authManager->request(
        QStringLiteral("POST"), url, QString(),
        [this](int statusCode, const std::string& errorMsg,
               const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, statusCode, errorMsg, data]() {
                    if (statusCode == 200 && errorMsg.empty()) {
                        const QJsonObject root =
                            QJsonDocument::fromJson(
                                QByteArray::fromStdString(data)).object();
                        m_lastRequeuedCount =
                            qMax(0, root.value(
                                        QStringLiteral("requeued")).toInt());
                        setLastError(QString());
                        emit lastRequeuedCountChanged();
                    } else {
                        const QString message =
                            problemMessage(data, errorMsg, statusCode);
                        setLastError(message);
                        emit requestFailed(message);
                    }
                    finishRequest();
                });
        });
}

void NotificationManager::retryFailedDeliveriesForUser()
{
    if (!m_authManager || !m_authManager->isLoggedIn()) {
        return;
    }
    const QUrl url(
        m_authManager->httpBaseUrl()
        + QStringLiteral("v1/me/notification-channels:retry-failed"));
    beginRequest();
    m_authManager->request(
        QStringLiteral("POST"), url, QString(),
        [this](int statusCode, const std::string& errorMsg,
               const std::string& data) {
            QMetaObject::invokeMethod(
                this,
                [this, statusCode, errorMsg, data]() {
                    if (statusCode == 200 && errorMsg.empty()) {
                        const QJsonObject root =
                            QJsonDocument::fromJson(
                                QByteArray::fromStdString(data)).object();
                        m_lastRequeuedCount =
                            qMax(0, root.value(
                                        QStringLiteral("requeued")).toInt());
                        setLastError(QString());
                        emit lastRequeuedCountChanged();
                    } else {
                        const QString message =
                            problemMessage(data, errorMsg, statusCode);
                        setLastError(message);
                        emit requestFailed(message);
                    }
                    finishRequest();
                });
        });
}

void NotificationManager::handleRealtimeEvent(const QVariantMap& message)
{
    const QString type = message.value(QStringLiteral("type")).toString();
    if (type.isEmpty()) {
        return;
    }
    const QVariantMap data = message.value(QStringLiteral("data")).toMap();
    emit realtimeEventReceived(
        type, data.value(QStringLiteral("device_id")).toString());
    if (type == QStringLiteral("device.metrics.updated")) {
        const QString eventDeviceId =
            data.value(QStringLiteral("device_id")).toString();
        if (m_metricAlertDeviceId.isEmpty()
            || eventDeviceId == m_metricAlertDeviceId) {
            m_metricCurrentValue = data;
            emit metricCurrentValueChanged();
        }
    }
    if (type.startsWith(QStringLiteral("device."))
        || type == QStringLiteral("scheduled_task.failed")) {
        emit realtimeRefreshRequested();
    }
}

void NotificationManager::beginRequest()
{
    if (m_pendingRequests++ == 0) {
        emit busyChanged();
    }
}

void NotificationManager::finishRequest()
{
    if (m_pendingRequests <= 0) {
        return;
    }
    if (--m_pendingRequests == 0) {
        emit busyChanged();
    }
}

void NotificationManager::setLastError(const QString& message)
{
    if (m_lastError == message) {
        return;
    }
    m_lastError = message;
    emit lastErrorChanged();
}

QVariantMap NotificationManager::normalizeChannel(const QJsonObject& object)
{
    QVariantMap row = object.toVariantMap();
    row[QStringLiteral("id")] = object.value(QStringLiteral("id")).toInt();
    row[QStringLiteral("name")] =
        object.value(QStringLiteral("name")).toString();
    row[QStringLiteral("url")] =
        object.value(QStringLiteral("url")).toString();
    row[QStringLiteral("events")] =
        parseEvents(object.value(QStringLiteral("events")));
    row[QStringLiteral("enabled")] =
        object.value(QStringLiteral("enabled")).toBool();
    row[QStringLiteral("last_status")] =
        object.value(QStringLiteral("last_status")).toInt();
    return row;
}

QVariantMap NotificationManager::normalizeQuietHours(
    const QJsonObject& object)
{
    QVariantMap view = object.toVariantMap();
    view[QStringLiteral("quiet_hours")] =
        normalizeWindows(object.value(QStringLiteral("quiet_hours")).toArray());
    view[QStringLiteral("timezone")] =
        object.value(QStringLiteral("timezone")).toString();
    view[QStringLiteral("active")] =
        object.value(QStringLiteral("active")).toBool();
    return view;
}

QVariantMap NotificationManager::normalizeOfflineAlert(
    const QJsonObject& object)
{
    QVariantMap view = object.toVariantMap();
    view[QStringLiteral("device_id")] =
        object.value(QStringLiteral("device_id")).toString();
    view[QStringLiteral("enabled")] =
        object.value(QStringLiteral("enabled")).toBool();
    view[QStringLiteral("after_minutes")] =
        qMax(1, object.value(QStringLiteral("after_minutes")).toInt(10));
    view[QStringLiteral("repeat_minutes")] =
        qMax(0, object.value(QStringLiteral("repeat_minutes")).toInt());
    view[QStringLiteral("maintenance_windows")] =
        normalizeWindows(
            object.value(
                QStringLiteral("maintenance_windows")).toArray());
    view[QStringLiteral("maintenance_timezone")] =
        object.value(
            QStringLiteral("maintenance_timezone")).toString();
    view[QStringLiteral("in_maintenance_window")] =
        object.value(
            QStringLiteral("in_maintenance_window")).toBool();
    return view;
}

QVariantMap NotificationManager::normalizeMetricAlert(
    const QJsonObject& object)
{
    QVariantMap view = object.toVariantMap();
    view[QStringLiteral("device_id")] =
        object.value(QStringLiteral("device_id")).toString();
    view[QStringLiteral("enabled")] =
        object.value(QStringLiteral("enabled")).toBool();
    view[QStringLiteral("cpu_percent")] =
        qBound(0,
               object.value(QStringLiteral("cpu_percent")).toInt(),
               100);
    view[QStringLiteral("memory_percent")] =
        qBound(0,
               object.value(QStringLiteral("memory_percent")).toInt(),
               100);
    view[QStringLiteral("disk_percent")] =
        qBound(0,
               object.value(QStringLiteral("disk_percent")).toInt(),
               100);
    view[QStringLiteral("sustained_minutes")] =
        qBound(0,
               object.value(QStringLiteral("sustained_minutes")).toInt(5),
               10080);
    view[QStringLiteral("repeat_minutes")] =
        qBound(0,
               object.value(QStringLiteral("repeat_minutes")).toInt(),
               10080);
    return view;
}

QVariantList NotificationManager::normalizeWindows(const QJsonArray& array)
{
    QVariantList windows;
    for (const QJsonValue& value : array) {
        const QJsonObject source = value.toObject();
        QVariantList days;
        QList<int> sortedDays;
        for (const QJsonValue& day :
             source.value(QStringLiteral("days")).toArray()) {
            const int number = day.toInt(-1);
            if (number >= 0 && number <= 6
                && !sortedDays.contains(number)) {
                sortedDays.append(number);
            }
        }
        std::sort(sortedDays.begin(), sortedDays.end());
        for (int day : sortedDays) {
            days.append(day);
        }
        windows.append(QVariantMap{
            {QStringLiteral("days"), days},
            {QStringLiteral("start"),
             source.value(QStringLiteral("start")).toString()},
            {QStringLiteral("end"),
             source.value(QStringLiteral("end")).toString()},
        });
    }
    return windows;
}

QJsonArray NotificationManager::windowsJson(const QVariantList& windows)
{
    QJsonArray result;
    for (const QVariant& value : windows) {
        const QVariantMap source = value.toMap();
        QJsonArray days;
        QList<int> sortedDays;
        for (const QVariant& day :
             source.value(QStringLiteral("days")).toList()) {
            const int number = day.toInt();
            if (number >= 0 && number <= 6
                && !sortedDays.contains(number)) {
                sortedDays.append(number);
            }
        }
        std::sort(sortedDays.begin(), sortedDays.end());
        for (int day : sortedDays) {
            days.append(day);
        }
        result.append(QJsonObject{
            {QStringLiteral("days"), days},
            {QStringLiteral("start"),
             source.value(QStringLiteral("start")).toString().trimmed()},
            {QStringLiteral("end"),
             source.value(QStringLiteral("end")).toString().trimmed()},
        });
    }
    return result;
}

QJsonValue NotificationManager::optionalInstant(const QString& value)
{
    const QString trimmed = value.trimmed();
    return trimmed.isEmpty() ? QJsonValue(QJsonValue::Null)
                             : QJsonValue(trimmed);
}

QStringList NotificationManager::stringList(const QJsonArray& array)
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

QStringList NotificationManager::parseEvents(const QJsonValue& value)
{
    if (value.isArray()) {
        return stringList(value.toArray());
    }
    if (!value.isString()) {
        return {};
    }
    const QJsonDocument document =
        QJsonDocument::fromJson(value.toString().toUtf8());
    return document.isArray() ? stringList(document.array()) : QStringList{};
}

QStringList NotificationManager::normalizedEvents(const QStringList& events)
{
    QStringList result;
    for (const QString& value : events) {
        const QString event = value.trimmed();
        if (!event.isEmpty() && !result.contains(event)) {
            result.append(event);
        }
    }
    return result;
}

QString NotificationManager::channelBody(const QString& name,
                                         const QString& url,
                                         const QString& secret,
                                         const QStringList& events,
                                         bool enabled)
{
    QJsonObject body;
    body[QStringLiteral("name")] = name.trimmed();
    body[QStringLiteral("url")] = url.trimmed();
    body[QStringLiteral("events")] = QJsonArray::fromStringList(events);
    body[QStringLiteral("enabled")] = enabled;
    const QString trimmedSecret = secret.trimmed();
    if (!trimmedSecret.isEmpty()) {
        body[QStringLiteral("secret")] = trimmedSecret;
    }
    return QString::fromUtf8(
        QJsonDocument(body).toJson(QJsonDocument::Compact));
}

QString NotificationManager::quietHoursBody(const QString& snoozedUntil,
                                            const QVariantList& windows,
                                            const QString& timezone)
{
    QJsonObject body;
    body[QStringLiteral("snoozed_until")] =
        optionalInstant(snoozedUntil);
    body[QStringLiteral("quiet_hours")] = windowsJson(windows);
    body[QStringLiteral("timezone")] = timezone.trimmed();
    return QString::fromUtf8(
        QJsonDocument(body).toJson(QJsonDocument::Compact));
}

QString NotificationManager::offlineAlertBody(
    bool enabled, int afterMinutes, int repeatMinutes,
    const QString& snoozedUntil, const QVariantList& maintenanceWindows,
    const QString& maintenanceTimezone)
{
    QJsonObject body;
    body[QStringLiteral("enabled")] = enabled;
    body[QStringLiteral("after_minutes")] = afterMinutes;
    body[QStringLiteral("repeat_minutes")] = repeatMinutes;
    body[QStringLiteral("snoozed_until")] =
        optionalInstant(snoozedUntil);
    body[QStringLiteral("maintenance_windows")] =
        windowsJson(maintenanceWindows);
    body[QStringLiteral("maintenance_timezone")] =
        maintenanceTimezone.trimmed();
    return QString::fromUtf8(
        QJsonDocument(body).toJson(QJsonDocument::Compact));
}

QString NotificationManager::metricAlertBody(
    bool enabled, int cpuPercent, int memoryPercent, int diskPercent,
    int sustainedMinutes, int repeatMinutes, const QString& snoozedUntil)
{
    QJsonObject body;
    body[QStringLiteral("enabled")] = enabled;
    body[QStringLiteral("cpu_percent")] = cpuPercent;
    body[QStringLiteral("memory_percent")] = memoryPercent;
    body[QStringLiteral("disk_percent")] = diskPercent;
    body[QStringLiteral("sustained_minutes")] = sustainedMinutes;
    body[QStringLiteral("repeat_minutes")] = repeatMinutes;
    body[QStringLiteral("snoozed_until")] =
        optionalInstant(snoozedUntil);
    return QString::fromUtf8(
        QJsonDocument(body).toJson(QJsonDocument::Compact));
}

QString NotificationManager::problemMessage(const std::string& data,
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
