// Copyright 2026 QuickDesk Authors
// Self-hosted update check, download and installer launch.

#ifndef QUICKDESK_MANAGER_UPDATEMANAGER_H
#define QUICKDESK_MANAGER_UPDATEMANAGER_H

#include <QNetworkAccessManager>
#include <QObject>
#include <QPointer>
#include <QString>
#include <QTimer>

class QFile;
class QNetworkReply;

namespace quickdesk {

class AuthManager;
class ServerManager;

class UpdateManager : public QObject {
    Q_OBJECT
    Q_PROPERTY(QString state READ state NOTIFY stateChanged)
    Q_PROPERTY(QString currentVersion READ currentVersion CONSTANT)
    Q_PROPERTY(QString latestVersion READ latestVersion NOTIFY updateInfoChanged)
    Q_PROPERTY(QString releaseNotes READ releaseNotes NOTIFY updateInfoChanged)
    Q_PROPERTY(QString errorMessage READ errorMessage NOTIFY errorMessageChanged)
    Q_PROPERTY(bool updateAvailable READ updateAvailable NOTIFY updateInfoChanged)
    Q_PROPERTY(bool mandatory READ mandatory NOTIFY updateInfoChanged)
    Q_PROPERTY(bool downloading READ downloading NOTIFY stateChanged)
    Q_PROPERTY(bool readyToInstall READ readyToInstall NOTIFY stateChanged)
    Q_PROPERTY(qreal downloadProgress READ downloadProgress NOTIFY progressChanged)
    Q_PROPERTY(qint64 downloadedBytes READ downloadedBytes NOTIFY progressChanged)
    Q_PROPERTY(qint64 totalBytes READ totalBytes NOTIFY progressChanged)
    Q_PROPERTY(QString installerFileName READ installerFileName NOTIFY updateInfoChanged)

public:
    UpdateManager(ServerManager* serverManager,
                  AuthManager* authManager,
                  QObject* parent = nullptr);
    ~UpdateManager() override;

    void start();
    void stop();

    QString state() const { return m_state; }
    QString currentVersion() const;
    QString latestVersion() const { return m_latestVersion; }
    QString releaseNotes() const { return m_releaseNotes; }
    QString errorMessage() const { return m_errorMessage; }
    bool updateAvailable() const { return m_updateAvailable; }
    bool mandatory() const { return m_mandatory; }
    bool downloading() const { return m_state == QStringLiteral("downloading"); }
    bool readyToInstall() const { return m_state == QStringLiteral("ready"); }
    qreal downloadProgress() const { return m_downloadProgress; }
    qint64 downloadedBytes() const { return m_downloadedBytes; }
    qint64 totalBytes() const { return m_totalBytes; }
    QString installerFileName() const;

    Q_INVOKABLE void checkForUpdates(bool userInitiated = false);
    Q_INVOKABLE void downloadUpdate();
    Q_INVOKABLE void cancelDownload();
    Q_INVOKABLE void installUpdate();
    Q_INVOKABLE void dismissUpdate();

signals:
    void stateChanged();
    void updateInfoChanged();
    void errorMessageChanged();
    void progressChanged();
    void updatePromptRequested(bool mandatory);
    void checkFinished(bool userInitiated, bool updateAvailable,
                       const QString& error);
    void downloadFinished(bool success, const QString& error);
    void installerLaunched();

private:
    void scheduleNextCheck();
    void resetUpdateInfo();
    void setState(const QString& state);
    void setError(const QString& error);
    void failCheck(const QString& error);
    void onCheckFinished(QNetworkReply* reply, bool userInitiated);
    bool startDownload();
    void onDownloadReadyRead(QNetworkReply* reply);
    void onDownloadFinished(QNetworkReply* reply);
    void cleanupDownload();
    bool verifyInstaller(const QString& path) const;
    bool launchInstaller();
    QString updateCheckUrl() const;
    QString downloadDirectory() const;
    QString currentPlatform() const;
    QString currentArch() const;
    QString effectiveSkippedVersion() const;

    ServerManager* m_serverManager = nullptr;
    AuthManager* m_authManager = nullptr;
    QNetworkAccessManager m_network;
    QPointer<QNetworkReply> m_checkReply;
    QPointer<QNetworkReply> m_downloadReply;
    QFile* m_downloadFile = nullptr;
    QTimer m_initialTimer;
    QTimer m_pollTimer;

    QString m_state = QStringLiteral("idle");
    QString m_latestVersion;
    QString m_releaseNotes;
    QString m_errorMessage;
    QString m_downloadUrl;
    QString m_sha256;
    QString m_installerPath;
    QString m_installerFileName;
    bool m_updateAvailable = false;
    bool m_mandatory = false;
    bool m_started = false;
    bool m_cancelRequested = false;
    qint64 m_expectedSize = 0;
    qint64 m_downloadedBytes = 0;
    qint64 m_totalBytes = 0;
    qreal m_downloadProgress = 0.0;
};

}  // namespace quickdesk

#endif  // QUICKDESK_MANAGER_UPDATEMANAGER_H
