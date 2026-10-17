// Copyright 2026 QuickDesk Authors

#include "UpdateManager.h"

#include "AuthManager.h"
#include "ServerManager.h"
#include "core/localconfigcenter.h"
#include "core/updatesupport.h"
#include "infra/log/log.h"

#include <QCryptographicHash>
#include <QCoreApplication>
#include <QDateTime>
#include <QDir>
#include <QFile>
#include <QFileInfo>
#include <QJsonDocument>
#include <QJsonObject>
#include <QNetworkReply>
#include <QNetworkRequest>
#include <QProcess>
#include <QStandardPaths>
#include <QUrl>
#include <QUrlQuery>
#include <QtGlobal>

#include <algorithm>

namespace quickdesk {
namespace {

constexpr int kInitialCheckDelayMs = 20 * 1000;
constexpr int kCheckIntervalMs = 6 * 60 * 60 * 1000;
constexpr int kCheckTimeoutMs = 15 * 1000;
constexpr int kDownloadTimeoutMs = 10 * 60 * 1000;
constexpr qint64 kHashChunkSize = 1024 * 1024;

QString ProblemText(const QByteArray& body, const QString& fallback) {
    const QJsonDocument document = QJsonDocument::fromJson(body);
    if (document.isObject()) {
        const QJsonObject problem = document.object();
        const QString detail = problem.value(QStringLiteral("detail")).toString();
        if (!detail.isEmpty()) {
            return detail;
        }
        const QString title = problem.value(QStringLiteral("title")).toString();
        if (!title.isEmpty()) {
            return title;
        }
    }
    return fallback;
}

bool IsValidSha256(const QString& value) {
    if (value.size() != 64) {
        return false;
    }
    for (const QChar character : value) {
        if (!((character >= QLatin1Char('0') && character <= QLatin1Char('9')) ||
              (character >= QLatin1Char('a') && character <= QLatin1Char('f')))) {
            return false;
        }
    }
    return true;
}

}  // namespace

UpdateManager::UpdateManager(ServerManager* serverManager,
                             AuthManager* authManager,
                             QObject* parent)
    : QObject(parent)
    , m_serverManager(serverManager)
    , m_authManager(authManager)
{
    m_network.setRedirectPolicy(QNetworkRequest::NoLessSafeRedirectPolicy);

    m_initialTimer.setSingleShot(true);
    m_initialTimer.setInterval(kInitialCheckDelayMs);
    connect(&m_initialTimer, &QTimer::timeout, this,
            [this]() { checkForUpdates(false); });

    m_pollTimer.setInterval(kCheckIntervalMs);
    connect(&m_pollTimer, &QTimer::timeout, this,
            [this]() { checkForUpdates(false); });

    connect(m_serverManager, &ServerManager::serverUrlChanged, this, [this]() {
        resetUpdateInfo();
        if (m_started) {
            m_initialTimer.start(1000);
        }
    });
}

UpdateManager::~UpdateManager()
{
    stop();
}

QString UpdateManager::currentVersion() const
{
    const QString version = QCoreApplication::applicationVersion();
    return version.isEmpty() ? QStringLiteral(APP_VERSION_STR) : version;
}

QString UpdateManager::installerFileName() const
{
    return m_installerFileName;
}

void UpdateManager::start()
{
    if (m_started) {
        return;
    }
    m_started = true;
    m_initialTimer.start();
    m_pollTimer.start();
    LOG_INFO("[UpdateManager] started, first check in {}ms", kInitialCheckDelayMs);
}

void UpdateManager::stop()
{
    m_started = false;
    m_initialTimer.stop();
    m_pollTimer.stop();

    if (m_checkReply) {
        m_checkReply->abort();
        m_checkReply->deleteLater();
        m_checkReply = nullptr;
    }
    if (m_downloadReply) {
        m_downloadReply->abort();
        m_downloadReply->deleteLater();
        m_downloadReply = nullptr;
    }
    cleanupDownload();
}

void UpdateManager::checkForUpdates(bool userInitiated)
{
    if (!m_serverManager || !m_authManager) {
        failCheck(QStringLiteral("Update manager is not configured"));
        return;
    }
    if (m_state == QStringLiteral("downloading")) {
        emit checkFinished(userInitiated, m_updateAvailable,
                           QStringLiteral("An update is already downloading"));
        return;
    }
    if (m_checkReply) {
        return;
    }

    setError(QString());
    setState(QStringLiteral("checking"));
    const QUrl url(updateCheckUrl());
    QNetworkRequest request(url);
    request.setTransferTimeout(kCheckTimeoutMs);
    request.setRawHeader("Accept", "application/json");
    request.setRawHeader(
        "User-Agent",
        QStringLiteral("QuickDesk/%1").arg(currentVersion()).toUtf8());
    for (const auto& header : m_authManager->publicHeaders()) {
        request.setRawHeader(header.first.toUtf8(), header.second.toUtf8());
    }

    m_checkReply = m_network.get(request);
    QNetworkReply* reply = m_checkReply;
    connect(reply, &QNetworkReply::finished, this,
            [this, reply, userInitiated]() {
                onCheckFinished(reply, userInitiated);
            });
}

void UpdateManager::onCheckFinished(QNetworkReply* reply, bool userInitiated)
{
    if (m_checkReply != reply) {
        reply->deleteLater();
        return;
    }
    const QByteArray body = reply->readAll();
    const int status =
        reply->attribute(QNetworkRequest::HttpStatusCodeAttribute).toInt();
    const QNetworkReply::NetworkError networkError = reply->error();
    reply->deleteLater();
    if (m_checkReply == reply) {
        m_checkReply = nullptr;
    }

    if (networkError != QNetworkReply::NoError || status != 200) {
        const QString fallback = networkError != QNetworkReply::NoError
                                     ? reply->errorString()
                                     : QStringLiteral("HTTP %1").arg(status);
        failCheck(ProblemText(body, fallback));
        emit checkFinished(userInitiated, false, m_errorMessage);
        return;
    }

    QJsonParseError parseError;
    const QJsonDocument document = QJsonDocument::fromJson(body, &parseError);
    if (parseError.error != QJsonParseError::NoError || !document.isObject()) {
        failCheck(QStringLiteral("Invalid update response"));
        emit checkFinished(userInitiated, false, m_errorMessage);
        return;
    }

    const QJsonObject root = document.object();
    const bool available = root.value(QStringLiteral("update_available")).toBool();
    const QJsonValue latestValue = root.value(QStringLiteral("latest"));
    if (!available || !latestValue.isObject()) {
        resetUpdateInfo();
        setState(QStringLiteral("up-to-date"));
        emit checkFinished(userInitiated, false, QString());
        scheduleNextCheck();
        return;
    }

    const QJsonObject latest = latestValue.toObject();
    const QString version = latest.value(QStringLiteral("version")).toString();
    const QString downloadUrl =
        latest.value(QStringLiteral("download_url")).toString();
    const QString sha256 =
        latest.value(QStringLiteral("sha256")).toString().toLower();
    const qint64 sizeBytes =
        latest.value(QStringLiteral("size_bytes")).toVariant().toLongLong();
    const QUrl parsedDownloadUrl(downloadUrl);
    if (!quickdesk::IsUpdateNewer(version.toStdString(), currentVersion().toStdString()) ||
        !parsedDownloadUrl.isValid() ||
        (parsedDownloadUrl.scheme() != QStringLiteral("http") &&
         parsedDownloadUrl.scheme() != QStringLiteral("https")) ||
        !IsValidSha256(sha256) || sizeBytes < 0) {
        failCheck(QStringLiteral("The update metadata is invalid"));
        emit checkFinished(userInitiated, false, m_errorMessage);
        return;
    }

    m_updateAvailable = true;
    m_mandatory = root.value(QStringLiteral("mandatory")).toBool() ||
                  latest.value(QStringLiteral("mandatory")).toBool();
    m_latestVersion = version;
    m_releaseNotes = latest.value(QStringLiteral("notes")).toString();
    m_downloadUrl = downloadUrl;
    m_sha256 = sha256;
    m_expectedSize = sizeBytes;
    m_errorMessage.clear();
    setState(QStringLiteral("available"));
    emit updateInfoChanged();
    emit errorMessageChanged();
    emit checkFinished(userInitiated, true, QString());

    const bool skipped =
        effectiveSkippedVersion() == m_latestVersion && !userInitiated;
    if (!skipped || m_mandatory) {
        emit updatePromptRequested(m_mandatory);
    }
    scheduleNextCheck();
}

void UpdateManager::downloadUpdate()
{
    if (!m_updateAvailable || m_downloadUrl.isEmpty()) {
        emit downloadFinished(false, QStringLiteral("No update is available"));
        return;
    }

    const QUrl url(m_downloadUrl);
    const QString extension = QString::fromStdString(
        quickdesk::InstallerExtensionForPlatform(currentPlatform().toStdString()));
    if (extension.isEmpty()) {
        const QString error =
            QStringLiteral("Automatic updates are not supported on this platform");
        setError(error);
        setState(QStringLiteral("failed"));
        emit downloadFinished(false, error);
        return;
    }

    QString proposedName = url.fileName(QUrl::FullyDecoded);
    const QString fallbackName =
        QStringLiteral("QuickDesk-%1%2").arg(m_latestVersion, extension);
    proposedName = QString::fromStdString(quickdesk::SanitizeInstallerFileName(
        proposedName.toStdString(), fallbackName.toStdString()));
    if (!extension.isEmpty() &&
        !proposedName.endsWith(extension, Qt::CaseInsensitive)) {
        proposedName += extension;
    }
    m_installerFileName = proposedName;

    const QString directory = downloadDirectory();
    if (!QDir().mkpath(directory)) {
        const QString error = QStringLiteral("Unable to create update directory");
        setError(error);
        setState(QStringLiteral("failed"));
        emit downloadFinished(false, error);
        return;
    }
    m_installerPath = QDir(directory).filePath(proposedName);
    const QFileInfo finalInfo(m_installerPath);
    if (finalInfo.exists() && verifyInstaller(m_installerPath)) {
        setState(QStringLiteral("ready"));
        m_downloadProgress = 1.0;
        emit progressChanged();
        emit downloadFinished(true, QString());
        return;
    }
    QFile::remove(m_installerPath + QStringLiteral(".part"));

    QNetworkRequest request(url);
    request.setTransferTimeout(kDownloadTimeoutMs);
    request.setRawHeader(
        "User-Agent",
        QStringLiteral("QuickDesk/%1").arg(currentVersion()).toUtf8());
    if (quickdesk::UrlsHaveSameOrigin(
            m_authManager->httpBaseUrl().toStdString(), m_downloadUrl.toStdString())) {
        for (const auto& header : m_authManager->publicHeaders()) {
            request.setRawHeader(header.first.toUtf8(), header.second.toUtf8());
        }
    }

    m_cancelRequested = false;
    m_downloadedBytes = 0;
    m_totalBytes = m_expectedSize;
    m_downloadProgress = 0.0;
    emit progressChanged();
    emit updateInfoChanged();
    setState(QStringLiteral("downloading"));
    if (!startDownload()) {
        return;
    }

    m_downloadReply = m_network.get(request);
    QNetworkReply* reply = m_downloadReply;
    connect(reply, &QNetworkReply::readyRead, this,
            [this, reply]() { onDownloadReadyRead(reply); });
    connect(reply, &QNetworkReply::downloadProgress, this,
            [this](qint64, qint64 total) {
                m_totalBytes = total > 0 ? qMax(total, m_expectedSize)
                                         : m_expectedSize;
                m_downloadProgress =
                    m_totalBytes > 0
                        ? std::clamp(static_cast<qreal>(m_downloadedBytes) /
                                         static_cast<qreal>(m_totalBytes),
                                     0.0, 1.0)
                        : 0.0;
                emit progressChanged();
            });
    connect(reply, &QNetworkReply::finished, this,
            [this, reply]() { onDownloadFinished(reply); });
}

bool UpdateManager::startDownload()
{
    const QString partPath = m_installerPath + QStringLiteral(".part");
    m_downloadFile = new QFile(partPath, this);
    if (!m_downloadFile->open(QIODevice::WriteOnly | QIODevice::Truncate)) {
        const QString error =
            QStringLiteral("Unable to create installer file: %1")
                .arg(m_downloadFile->errorString());
        cleanupDownload();
        setError(error);
        setState(QStringLiteral("failed"));
        emit downloadFinished(false, error);
        return false;
    }
    return true;
}

void UpdateManager::onDownloadReadyRead(QNetworkReply* reply)
{
    if (!m_downloadFile || !m_downloadFile->isOpen()) {
        return;
    }
    const QByteArray chunk = reply->readAll();
    if (chunk.isEmpty()) {
        return;
    }
    if (m_downloadFile->write(chunk) != chunk.size()) {
        reply->abort();
        return;
    }
    m_downloadedBytes += chunk.size();
    if (m_totalBytes > 0) {
        m_downloadProgress =
            std::clamp(static_cast<qreal>(m_downloadedBytes) /
                           static_cast<qreal>(m_totalBytes),
                       0.0, 1.0);
    }
    emit progressChanged();
}

void UpdateManager::onDownloadFinished(QNetworkReply* reply)
{
    if (m_downloadReply != reply) {
        reply->deleteLater();
        return;
    }
    if (m_downloadFile && m_downloadFile->isOpen()) {
        const QByteArray remaining = reply->readAll();
        if (!remaining.isEmpty()) {
            m_downloadFile->write(remaining);
            m_downloadedBytes += remaining.size();
        }
        m_downloadFile->flush();
        m_downloadFile->close();
    }

    const int status =
        reply->attribute(QNetworkRequest::HttpStatusCodeAttribute).toInt();
    const QNetworkReply::NetworkError networkError = reply->error();
    const QString networkErrorText = reply->errorString();
    reply->deleteLater();
    if (m_downloadReply == reply) {
        m_downloadReply = nullptr;
    }
    cleanupDownload();

    if (m_cancelRequested) {
        m_cancelRequested = false;
        m_downloadedBytes = 0;
        m_downloadProgress = 0.0;
        emit progressChanged();
        setState(QStringLiteral("available"));
        emit downloadFinished(false, QStringLiteral("Download cancelled"));
        return;
    }

    if (networkError != QNetworkReply::NoError ||
        (status != 200 && status != 206)) {
        const QString error =
            networkError != QNetworkReply::NoError
                ? networkErrorText
                : QStringLiteral("HTTP %1").arg(status);
        QFile::remove(m_installerPath + QStringLiteral(".part"));
        setError(error);
        setState(QStringLiteral("failed"));
        emit downloadFinished(false, error);
        return;
    }

    const QString partPath = m_installerPath + QStringLiteral(".part");
    QFile::remove(m_installerPath);
    if (!QFile::rename(partPath, m_installerPath)) {
        const QString error =
            QStringLiteral("Unable to finalize the downloaded installer");
        setError(error);
        setState(QStringLiteral("failed"));
        emit downloadFinished(false, error);
        return;
    }
    if (!verifyInstaller(m_installerPath)) {
        QFile::remove(m_installerPath);
        const QString error =
            QStringLiteral("Downloaded installer failed size or SHA-256 verification");
        setError(error);
        setState(QStringLiteral("failed"));
        emit downloadFinished(false, error);
        return;
    }

    m_downloadProgress = 1.0;
    m_downloadedBytes = m_expectedSize > 0 ? m_expectedSize : m_downloadedBytes;
    m_totalBytes = m_downloadedBytes;
    emit progressChanged();
    setState(QStringLiteral("ready"));
    emit downloadFinished(true, QString());
}

void UpdateManager::cancelDownload()
{
    if (!m_downloadReply) {
        return;
    }
    m_cancelRequested = true;
    m_downloadReply->abort();
}

void UpdateManager::installUpdate()
{
    if (m_installerPath.isEmpty() || !verifyInstaller(m_installerPath)) {
        const QString error = QStringLiteral("The update installer is unavailable");
        setError(error);
        setState(QStringLiteral("failed"));
        return;
    }
    if (!launchInstaller()) {
        const QString error = QStringLiteral("Unable to launch the update installer");
        setError(error);
        setState(QStringLiteral("failed"));
        return;
    }
    setState(QStringLiteral("installing"));
    emit installerLaunched();
}

void UpdateManager::dismissUpdate()
{
    if (!m_updateAvailable || m_mandatory || m_latestVersion.isEmpty()) {
        return;
    }
    core::LocalConfigCenter::instance().setSkippedUpdateVersion(m_latestVersion);
    LOG_INFO("[UpdateManager] optional update {} postponed",
             m_latestVersion.toStdString());
}

void UpdateManager::scheduleNextCheck()
{
    if (m_started) {
        m_pollTimer.start();
    }
}

void UpdateManager::resetUpdateInfo()
{
    m_updateAvailable = false;
    m_mandatory = false;
    m_latestVersion.clear();
    m_releaseNotes.clear();
    m_downloadUrl.clear();
    m_sha256.clear();
    m_expectedSize = 0;
    m_downloadProgress = 0.0;
    m_downloadedBytes = 0;
    m_totalBytes = 0;
    m_installerPath.clear();
    m_installerFileName.clear();
    emit updateInfoChanged();
    emit progressChanged();
}

void UpdateManager::setState(const QString& state)
{
    if (m_state == state) {
        return;
    }
    m_state = state;
    emit stateChanged();
}

void UpdateManager::setError(const QString& error)
{
    if (m_errorMessage == error) {
        return;
    }
    m_errorMessage = error;
    emit errorMessageChanged();
}

void UpdateManager::failCheck(const QString& error)
{
    setError(error);
    setState(QStringLiteral("failed"));
    LOG_WARN("[UpdateManager] update check failed: {}", error.toStdString());
}

void UpdateManager::cleanupDownload()
{
    if (m_downloadFile) {
        if (m_downloadFile->isOpen()) {
            m_downloadFile->close();
        }
        m_downloadFile->deleteLater();
        m_downloadFile = nullptr;
    }
}

bool UpdateManager::verifyInstaller(const QString& path) const
{
    QFile file(path);
    if (!file.open(QIODevice::ReadOnly)) {
        return false;
    }
    if (m_expectedSize > 0 && file.size() != m_expectedSize) {
        return false;
    }

    QCryptographicHash hash(QCryptographicHash::Sha256);
    while (!file.atEnd()) {
        const QByteArray chunk = file.read(kHashChunkSize);
        if (chunk.isEmpty() && file.error() != QFileDevice::NoError) {
            return false;
        }
        hash.addData(chunk);
    }
    return QString::fromLatin1(hash.result().toHex()).compare(
               m_sha256, Qt::CaseInsensitive) == 0;
}

bool UpdateManager::launchInstaller()
{
#ifdef Q_OS_WIN
    return QProcess::startDetached(
        m_installerPath,
        {QStringLiteral("/SILENT"), QStringLiteral("/CLOSEAPPLICATIONS"),
         QStringLiteral("/RESTARTAPPLICATIONS")});
#elif defined(Q_OS_MACOS)
    return QProcess::startDetached(QStringLiteral("open"),
                                   {m_installerPath});
#else
    return false;
#endif
}

QString UpdateManager::updateCheckUrl() const
{
    QUrl url(m_authManager->httpBaseUrl() + QStringLiteral("v1/update:check"));
    QUrlQuery query;
    query.addQueryItem(QStringLiteral("platform"), currentPlatform());
    query.addQueryItem(QStringLiteral("arch"), currentArch());
    query.addQueryItem(QStringLiteral("channel"), QStringLiteral("stable"));
    query.addQueryItem(QStringLiteral("current_version"), currentVersion());
    url.setQuery(query);
    return url.toString();
}

QString UpdateManager::downloadDirectory() const
{
    return QDir(QStandardPaths::writableLocation(QStandardPaths::TempLocation))
        .filePath(QStringLiteral("QuickDesk/updates/%1").arg(m_latestVersion));
}

QString UpdateManager::currentPlatform() const
{
#ifdef Q_OS_WIN
    return QStringLiteral("windows");
#elif defined(Q_OS_MACOS)
    return QStringLiteral("macos");
#else
    return QStringLiteral("linux");
#endif
}

QString UpdateManager::currentArch() const
{
#if defined(Q_PROCESSOR_ARM_64)
    return QStringLiteral("arm64");
#elif defined(Q_PROCESSOR_X86_64)
    return QStringLiteral("x64");
#else
    return QStringLiteral("universal");
#endif
}

QString UpdateManager::effectiveSkippedVersion() const
{
    return core::LocalConfigCenter::instance().skippedUpdateVersion();
}

}  // namespace quickdesk
