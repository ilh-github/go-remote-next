// Copyright 2026 QuickDesk Authors
#ifndef QUICKDESK_MANAGER_RECORDINGMANAGER_H
#define QUICKDESK_MANAGER_RECORDINGMANAGER_H

#include <QObject>
#include <QString>
#include <QVariantList>
#include <QVariantMap>

#include <string>

class QJsonObject;

namespace quickdesk {

class AuthManager;

class RecordingManager : public QObject {
    Q_OBJECT
    Q_PROPERTY(QVariantList recordings READ recordings NOTIFY recordingsChanged)
    Q_PROPERTY(QString nextCursor READ nextCursor NOTIFY nextCursorChanged)
    Q_PROPERTY(bool hasMore READ hasMore NOTIFY nextCursorChanged)
    Q_PROPERTY(bool busy READ busy NOTIFY busyChanged)
    Q_PROPERTY(QString lastError READ lastError NOTIFY lastErrorChanged)

public:
    explicit RecordingManager(AuthManager* authManager,
                              QObject* parent = nullptr);

    QVariantList recordings() const { return m_recordings; }
    QString nextCursor() const { return m_nextCursor; }
    bool hasMore() const { return !m_nextCursor.isEmpty(); }
    bool busy() const { return m_pendingRequests > 0; }
    QString lastError() const { return m_lastError; }

    Q_INVOKABLE void refresh();
    Q_INVOKABLE void loadMore();
    Q_INVOKABLE void deleteRecording(int recordingId);
    Q_INVOKABLE void requestArtifactUrl(int recordingId);
    Q_INVOKABLE void requestThumbnailUrl(int recordingId);
    Q_INVOKABLE void handleRealtimeEvent(const QVariantMap& message);

signals:
    void recordingsChanged();
    void nextCursorChanged();
    void busyChanged();
    void lastErrorChanged();
    void requestFailed(const QString& message);
    void recordingDeleted(int recordingId);
    void artifactUrlReady(int recordingId, const QString& url);
    void thumbnailUrlReady(int recordingId, const QString& url);
    void realtimeRefreshRequested();

private:
    void requestPage(const QString& cursor, bool append, int generation);
    void requestTicket(int recordingId, const QString& suffix,
                       const char* signalName);
    void beginRequest();
    void finishRequest();
    void setLastError(const QString& message);
    QString resolveUrl(const QString& raw) const;
    static QVariantMap normalizeRecording(const QJsonObject& object);
    static QString problemMessage(const std::string& data,
                                  const std::string& fallback,
                                  int statusCode);

    AuthManager* m_authManager = nullptr;
    QVariantList m_recordings;
    QString m_nextCursor;
    QString m_lastError;
    int m_pendingRequests = 0;
    int m_generation = 0;
};

}  // namespace quickdesk

#endif  // QUICKDESK_MANAGER_RECORDINGMANAGER_H
