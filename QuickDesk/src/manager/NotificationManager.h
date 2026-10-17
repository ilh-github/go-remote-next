// Copyright 2026 QuickDesk Authors
#ifndef QUICKDESK_MANAGER_NOTIFICATIONMANAGER_H
#define QUICKDESK_MANAGER_NOTIFICATIONMANAGER_H

#include <QObject>
#include <QString>
#include <QStringList>
#include <QVariantList>
#include <QVariantMap>

#include <string>

class QJsonArray;
class QJsonObject;
class QJsonValue;

namespace quickdesk {

class AuthManager;

class NotificationManager : public QObject {
    Q_OBJECT
    Q_PROPERTY(QVariantList channels READ channels NOTIFY channelsChanged)
    Q_PROPERTY(QStringList supportedEvents READ supportedEvents
               NOTIFY supportedEventsChanged)
    Q_PROPERTY(QString signatureHeader READ signatureHeader
               NOTIFY signatureInfoChanged)
    Q_PROPERTY(QString signatureScheme READ signatureScheme
               NOTIFY signatureInfoChanged)
    Q_PROPERTY(QVariantMap quietHours READ quietHours NOTIFY quietHoursChanged)
    Q_PROPERTY(QVariantMap offlineAlert READ offlineAlert
               NOTIFY offlineAlertChanged)
    Q_PROPERTY(QString offlineDeviceId READ offlineDeviceId
               NOTIFY offlineDeviceIdChanged)
    Q_PROPERTY(QVariantList deliveries READ deliveries
               NOTIFY deliveriesChanged)
    Q_PROPERTY(int unackedDeliveries READ unackedDeliveries
               NOTIFY deliveriesChanged)
    Q_PROPERTY(int deliveryChannelId READ deliveryChannelId
               NOTIFY deliveryChannelIdChanged)
    Q_PROPERTY(int lastRequeuedCount READ lastRequeuedCount
               NOTIFY lastRequeuedCountChanged)
    Q_PROPERTY(bool busy READ busy NOTIFY busyChanged)
    Q_PROPERTY(QString lastError READ lastError NOTIFY lastErrorChanged)

public:
    explicit NotificationManager(AuthManager* authManager,
                                 QObject* parent = nullptr);

    QVariantList channels() const { return m_channels; }
    QStringList supportedEvents() const { return m_supportedEvents; }
    QString signatureHeader() const { return m_signatureHeader; }
    QString signatureScheme() const { return m_signatureScheme; }
    QVariantMap quietHours() const { return m_quietHours; }
    QVariantMap offlineAlert() const { return m_offlineAlert; }
    QString offlineDeviceId() const { return m_offlineDeviceId; }
    QVariantList deliveries() const { return m_deliveries; }
    int unackedDeliveries() const { return m_unackedDeliveries; }
    int deliveryChannelId() const { return m_deliveryChannelId; }
    int lastRequeuedCount() const { return m_lastRequeuedCount; }
    bool busy() const { return m_pendingRequests > 0; }
    QString lastError() const { return m_lastError; }

    Q_INVOKABLE void refreshChannels();
    Q_INVOKABLE void createChannel(const QString& name,
                                   const QString& url,
                                   const QString& secret,
                                   const QStringList& events,
                                   bool enabled);
    Q_INVOKABLE void updateChannel(int channelId,
                                   const QString& name,
                                   const QString& url,
                                   const QString& secret,
                                   const QStringList& events,
                                   bool enabled);
    Q_INVOKABLE void deleteChannel(int channelId);
    Q_INVOKABLE void testChannel(int channelId);
    Q_INVOKABLE void loadQuietHours();
    Q_INVOKABLE void saveQuietHours(const QString& snoozedUntil,
                                    const QVariantList& windows,
                                    const QString& timezone);
    Q_INVOKABLE void loadOfflineAlert(const QString& deviceId);
    Q_INVOKABLE void saveOfflineAlert(bool enabled,
                                      int afterMinutes,
                                      int repeatMinutes,
                                      const QString& snoozedUntil,
                                      const QVariantList& maintenanceWindows,
                                      const QString& maintenanceTimezone);
    Q_INVOKABLE void removeOfflineAlert();
    Q_INVOKABLE void loadDeliveries(int channelId);
    Q_INVOKABLE void retryDelivery(int channelId, int deliveryId);
    Q_INVOKABLE void ackDelivery(int channelId, int deliveryId);
    Q_INVOKABLE void retryFailedDeliveries(int channelId);
    Q_INVOKABLE void retryFailedDeliveriesForUser();
    Q_INVOKABLE void handleRealtimeEvent(const QVariantMap& message);

signals:
    void channelsChanged();
    void supportedEventsChanged();
    void signatureInfoChanged();
    void quietHoursChanged();
    void offlineAlertChanged();
    void offlineDeviceIdChanged();
    void deliveriesChanged();
    void deliveryChannelIdChanged();
    void lastRequeuedCountChanged();
    void busyChanged();
    void lastErrorChanged();
    void requestFailed(const QString& message);
    void channelCreated(int channelId);
    void channelUpdated(int channelId);
    void channelDeleted(int channelId);
    void channelTestQueued(int channelId);
    void realtimeEventReceived(const QString& type, const QString& deviceId);
    void realtimeRefreshRequested();

private:
    void beginRequest();
    void finishRequest();
    void setLastError(const QString& message);
    static QVariantMap normalizeChannel(const QJsonObject& object);
    static QVariantMap normalizeQuietHours(const QJsonObject& object);
    static QVariantMap normalizeOfflineAlert(const QJsonObject& object);
    static QVariantList normalizeWindows(const QJsonArray& array);
    static QJsonArray windowsJson(const QVariantList& windows);
    static QJsonValue optionalInstant(const QString& value);
    static QStringList stringList(const QJsonArray& array);
    static QStringList parseEvents(const QJsonValue& value);
    static QStringList normalizedEvents(const QStringList& events);
    static QString channelBody(const QString& name,
                               const QString& url,
                               const QString& secret,
                               const QStringList& events,
                               bool enabled);
    static QString quietHoursBody(const QString& snoozedUntil,
                                  const QVariantList& windows,
                                  const QString& timezone);
    static QString offlineAlertBody(bool enabled,
                                    int afterMinutes,
                                    int repeatMinutes,
                                    const QString& snoozedUntil,
                                    const QVariantList& maintenanceWindows,
                                    const QString& maintenanceTimezone);
    static QString problemMessage(const std::string& data,
                                  const std::string& fallback,
                                  int statusCode);

    AuthManager* m_authManager = nullptr;
    QVariantList m_channels;
    QStringList m_supportedEvents;
    QString m_signatureHeader;
    QString m_signatureScheme;
    QVariantMap m_quietHours;
    QVariantMap m_offlineAlert;
    QString m_offlineDeviceId;
    QVariantList m_deliveries;
    int m_unackedDeliveries = 0;
    int m_deliveryChannelId = 0;
    int m_lastRequeuedCount = 0;
    QString m_lastError;
    int m_pendingRequests = 0;
    int m_channelsGeneration = 0;
    int m_quietGeneration = 0;
    int m_offlineGeneration = 0;
    int m_deliveryGeneration = 0;
};

}  // namespace quickdesk

#endif  // QUICKDESK_MANAGER_NOTIFICATIONMANAGER_H
