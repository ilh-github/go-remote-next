// Copyright 2026 QuickDesk Authors
// Control-side session recorder implementation.

#include "DeviceRecorder.h"

#include "AuthManager.h"
#include "ServerManager.h"
#include "SharedMemoryManager.h"
#include "infra/log/log.h"

#include <QDateTime>
#include <QDir>
#include <QFile>
#include <QFileInfo>
#include <QHttpMultiPart>
#include <QHttpPart>
#include <QJsonDocument>
#include <QJsonObject>
#include <QMediaFormat>
#include <QNetworkReply>
#include <QNetworkRequest>
#include <QStandardPaths>
#include <QUrl>
#include <QVideoFrame>

namespace quickdesk {

namespace {

// Sampling rate for pushing decoded frames into the encoder. The remote stream
// may arrive at a different rate; the encoder is fed at a steady cadence and
// timestamps are generated locally so the resulting file has correct timing.
constexpr int kFrameIntervalMs = 1000 / 15;
// QVideoFrame presentation timestamps are in microseconds, not milliseconds.
constexpr qint64 kFrameDurationUs = 1000000 / 15;

QString isoNow() {
    return QDateTime::currentDateTimeUtc().toString(Qt::ISODate);
}

} // namespace

DeviceRecorder::DeviceRecorder(SharedMemoryManager* sharedMemory,
                               AuthManager* auth,
                               ServerManager* server,
                               QObject* parent)
    : QObject(parent)
    , m_sharedMemory(sharedMemory)
    , m_auth(auth)
    , m_server(server) {
    m_frameTimer.setInterval(kFrameIntervalMs);
    m_frameTimer.setTimerType(Qt::PreciseTimer);
    connect(&m_frameTimer, &QTimer::timeout, this, &DeviceRecorder::onFrameTick);
    connect(&m_recorder, &QMediaRecorder::recorderStateChanged,
            this, &DeviceRecorder::onRecorderStateChanged);
    connect(&m_recorder, &QMediaRecorder::errorOccurred,
            this, &DeviceRecorder::onRecorderError);
}

DeviceRecorder::~DeviceRecorder() {
    if (m_recording) {
        m_frameTimer.stop();
        m_recorder.stop();
    }
}

bool DeviceRecorder::startRecording(const QString& deviceId) {
    if (m_recording) {
        return true;
    }
    if (deviceId.isEmpty()) {
        fail(QStringLiteral("no active device to record"));
        return false;
    }
    if (!m_sharedMemory || !m_auth || !m_server) {
        fail(QStringLiteral("recorder is not wired to the client managers"));
        return false;
    }

    const QString dir = QStandardPaths::writableLocation(QStandardPaths::TempLocation);
    QDir().mkpath(dir);
    m_outputPath = QDir(dir).filePath(
        QStringLiteral("quickdesk-rec-%1.mp4").arg(QDateTime::currentMSecsSinceEpoch()));
    QFile::remove(m_outputPath);

    QMediaFormat format;
    format.setFileFormat(QMediaFormat::MPEG4);
    format.setVideoCodec(QMediaFormat::VideoCodec::H264);
    m_recorder.setMediaFormat(format);
    m_recorder.setQuality(QMediaRecorder::HighQuality);
    m_recorder.setVideoFrameRate(15);
    m_recorder.setOutputLocation(QUrl::fromLocalFile(m_outputPath));

    // Video-only: the decoded remote stream is fed as frames, so there is no
    // capture source and no audio input to attach.
    m_session.setVideoFrameInput(&m_frameInput);
    m_session.setRecorder(&m_recorder);

    m_deviceId = deviceId;
    m_startedAtMs = QDateTime::currentMSecsSinceEpoch();
    m_lastError.clear();
    emit lastErrorChanged();

    m_recorder.record();
    m_recording = true;
    emit recordingChanged();
    m_frameTimer.start();
    LOG_INFO("DeviceRecorder: recording started device={} path={}",
             deviceId.toStdString(), m_outputPath.toStdString());
    return true;
}

void DeviceRecorder::stopRecording() {
    if (!m_recording) {
        return;
    }
    m_frameTimer.stop();
    m_recording = false;
    emit recordingChanged();
    // The upload is kicked off from the StoppedState transition so the file is
    // already finalised on disk.
    m_recorder.stop();
}

void DeviceRecorder::onFrameTick() {
    if (!m_recording || m_deviceId.isEmpty() || !m_sharedMemory) {
        return;
    }
    QVideoFrame frame = m_sharedMemory->readVideoFrame(m_deviceId);
    if (!frame.isValid()) {
        return;
    }
    // readVideoFrame() copies the planes into an owned buffer, so the frame can
    // be handed to the encoder without racing the producer.
    // Qt expects microsecond presentation timestamps here; feeding milliseconds
    // collapses a multi-second recording into a few milliseconds of container
    // time (caught by probing the produced MP4, not by the build).
    const qint64 nowUs = QDateTime::currentMSecsSinceEpoch() * 1000;
    const qint64 elapsedUs = nowUs - (m_startedAtMs * 1000);
    frame.setStartTime(elapsedUs);
    frame.setEndTime(elapsedUs + kFrameDurationUs);
    m_frameInput.sendVideoFrame(frame);
}

void DeviceRecorder::onRecorderStateChanged(QMediaRecorder::RecorderState state) {
    if (state != QMediaRecorder::StoppedState || m_outputPath.isEmpty()) {
        return;
    }
    const QString finishedPath = m_outputPath;
    m_outputPath.clear();
    uploadRecording(finishedPath);
}

void DeviceRecorder::onRecorderError(QMediaRecorder::Error error, const QString& errorString) {
    Q_UNUSED(error);
    fail(errorString.isEmpty() ? QStringLiteral("recorder error") : errorString);
}

void DeviceRecorder::fail(const QString& message) {
    m_lastError = message;
    emit lastErrorChanged();
    LOG_WARN("DeviceRecorder: {}", message.toStdString());
}

QString DeviceRecorder::httpBaseUrl() const {
    QString url = m_server ? m_server->serverUrl() : QString();
    url.replace(QStringLiteral("ws://"), QStringLiteral("http://"));
    url.replace(QStringLiteral("wss://"), QStringLiteral("https://"));
    if (!url.endsWith(QLatin1Char('/'))) {
        url += QLatin1Char('/');
    }
    return url;
}

void DeviceRecorder::uploadRecording(const QString& filePath) {
    const QFileInfo info(filePath);
    if (!info.exists() || info.size() <= 0) {
        fail(QStringLiteral("recording produced no data"));
        QFile::remove(filePath);
        return;
    }
    createIndexRowThenUpload(filePath, info.size());
}

void DeviceRecorder::createIndexRowThenUpload(const QString& filePath, qint64 sizeBytes) {
    Q_UNUSED(sizeBytes);
    const qint64 durationSeconds =
        qMax<qint64>(1, (QDateTime::currentMSecsSinceEpoch() - m_startedAtMs) / 1000);

    QJsonObject payload;
    payload.insert(QStringLiteral("device_id"), m_deviceId);
    payload.insert(QStringLiteral("device_name"), QString());
    payload.insert(QStringLiteral("session_id"),
                   QStringLiteral("qt-%1").arg(QDateTime::currentMSecsSinceEpoch()));
    payload.insert(QStringLiteral("status"), QStringLiteral("completed"));
    payload.insert(QStringLiteral("format"), QStringLiteral("mp4"));
    payload.insert(QStringLiteral("duration_seconds"), durationSeconds);
    payload.insert(QStringLiteral("started_at"),
                   QDateTime::fromMSecsSinceEpoch(m_startedAtMs).toUTC().toString(Qt::ISODate));
    payload.insert(QStringLiteral("ended_at"), isoNow());

    const QString body = QString::fromUtf8(QJsonDocument(payload).toJson(QJsonDocument::Compact));
    const QUrl url(httpBaseUrl() + QStringLiteral("v1/me/recordings"));

    m_auth->request(QStringLiteral("POST"), url, body,
                    [this, filePath](int statusCode, const std::string& errorMsg,
                                     const std::string& data) {
                        const QString message = QString::fromStdString(errorMsg);
                        const QString payloadText = QString::fromStdString(data);
                        QMetaObject::invokeMethod(this, [this, statusCode, message, payloadText, filePath]() {
                            if (statusCode != 200 && statusCode != 201) {
                                fail(QStringLiteral("could not index recording: HTTP %1 %2")
                                         .arg(statusCode)
                                         .arg(message));
                                return;
                            }
                            const QJsonObject row =
                                QJsonDocument::fromJson(payloadText.toUtf8()).object();
                            const QJsonValue idValue = row.value(QStringLiteral("id"));
                            if (idValue.isUndefined() || idValue.isNull()) {
                                fail(QStringLiteral("index response carried no recording id"));
                                return;
                            }
                            uploadBytes(filePath, QString::number(idValue.toInt()));
                        });
                    });
}

void DeviceRecorder::uploadBytes(const QString& filePath, const QString& recordingId) {
    auto* file = new QFile(filePath);
    if (!file->open(QIODevice::ReadOnly)) {
        delete file;
        fail(QStringLiteral("cannot reopen the recorded file for upload"));
        return;
    }

    auto* multipart = new QHttpMultiPart(QHttpMultiPart::FormDataType);
    QHttpPart filePart;
    filePart.setHeader(QNetworkRequest::ContentTypeHeader, QStringLiteral("video/mp4"));
    filePart.setHeader(QNetworkRequest::ContentDispositionHeader,
                       QStringLiteral("form-data; name=\"file\"; filename=\"recording.mp4\""));
    filePart.setBodyDevice(file);
    file->setParent(multipart);
    multipart->append(filePart);

    QNetworkRequest request(
        QUrl(httpBaseUrl() + QStringLiteral("v1/me/recordings/%1/artifact").arg(recordingId)));
    for (const auto& header : m_auth->userAuthHeaders(false)) {
        request.setRawHeader(header.first.toUtf8(), header.second.toUtf8());
    }

    QNetworkReply* reply = m_network.post(request, multipart);
    multipart->setParent(reply);
    connect(reply, &QNetworkReply::finished, this, [this, reply, file, filePath, recordingId]() {
        const int status = reply->attribute(QNetworkRequest::HttpStatusCodeAttribute).toInt();
        const QByteArray responseBody = reply->readAll();
        const QNetworkReply::NetworkError networkError = reply->error();
        reply->deleteLater();

        // Close the source file before removing it: Windows refuses to delete an
        // open handle.
        if (file && file->isOpen()) {
            file->close();
        }

        if (networkError != QNetworkReply::NoError || (status != 200 && status != 201)) {
            fail(QStringLiteral("recording upload failed: HTTP %1 %2")
                     .arg(status)
                     .arg(QString::fromUtf8(responseBody.left(200))));
            return;
        }
        m_lastRecordingId = recordingId;
        emit uploadFinished(m_lastRecordingId);
        LOG_INFO("DeviceRecorder: uploaded recording id={} file={}",
                 recordingId.toStdString(), filePath.toStdString());
        // The temp file only exists to carry the bytes to the server.
        QFile::remove(filePath);
    });
}

} // namespace quickdesk
