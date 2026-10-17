// Copyright 2026 QuickDesk Authors
// SkillHostManager implementation

#include "SkillHostManager.h"
#include "HostManager.h"
#include "infra/log/log.h"

#include <QJsonDocument>
#include <QProcess>
#include <QRegularExpression>
#include <QStringList>
#include <QTimer>
#include <QUuid>

namespace quickdesk {

namespace {

constexpr int kInternalRequestTimeoutMs = 10000;

QJsonObject toolResultObject(const QJsonValue& value)
{
    if (value.isObject()) {
        return value.toObject();
    }
    if (!value.isString()) {
        return {};
    }
    const QJsonDocument document =
        QJsonDocument::fromJson(value.toString().toUtf8());
    return document.isObject() ? document.object() : QJsonObject{};
}

}  // namespace

SkillHostManager::SkillHostManager(QObject* parent)
    : QObject(parent)
{}

SkillHostManager::~SkillHostManager()
{
    failPendingInternalRequests(
        QStringLiteral("Skill host stopped before the request completed"));
    if (m_skillHostProcess) {
        if (m_skillHostProcess->state() != QProcess::NotRunning) {
            m_skillHostProcess->kill();
            m_skillHostProcess->waitForFinished(1000);
        }
        delete m_skillHostProcess;
        m_skillHostProcess = nullptr;
    }
}

void SkillHostManager::setHostManager(HostManager* hostManager)
{
    m_hostManager = hostManager;

    // Wire: client → skill-host
    connect(m_hostManager, &HostManager::skillMessage,
            this, &SkillHostManager::onMessageFromClient);

    // When a new client connects, push cached skill-host capabilities
    connect(m_hostManager, &HostManager::clientConnected,
            this, [this](const QString& /*connectionId*/, const QJsonObject& /*info*/) {
        if (m_cachedTools.isEmpty()) return;

        QJsonObject msg;
        msg["type"] = QStringLiteral("capabilitiesReady");
        msg["tools"] = m_cachedTools;

        QByteArray bytes = QJsonDocument(msg).toJson(QJsonDocument::Compact);
        QString jsonData = QString::fromUtf8(bytes);
        m_hostManager->sendSkillBridgeSend(jsonData);

        LOG_INFO("SkillHostManager: pushed cached capabilities ({} tools) to new client",
                 m_cachedTools.size());
    });
}

void SkillHostManager::startSkillHost(const QString& skillHostPath, const QStringList& skillsDirs)
{
    if (m_skillHostProcess && m_skillHostProcess->state() != QProcess::NotRunning) {
        LOG_WARN("SkillHostManager: skill-host already running");
        return;
    }

    delete m_skillHostProcess;
    m_skillHostProcess = new QProcess(this);
    m_skillHostProcess->setProgram(skillHostPath);

    QStringList args;
    for (const auto& dir : skillsDirs) {
        if (!dir.isEmpty()) {
            args << "--skills-dir" << dir;
        }
    }
    m_skillHostProcess->setArguments(args);

    connect(m_skillHostProcess, &QProcess::readyReadStandardOutput,
            this, &SkillHostManager::onSkillHostStdout);
    // The skill host writes skill load failures (and everything else tracing
    // produces) to stderr. Without this the app only ever saw "0 tool(s)" and
    // had no way to say why, so keep those lines in the app log.
    connect(m_skillHostProcess, &QProcess::readyReadStandardError, this, [this]() {
        if (!m_skillHostProcess) {
            return;
        }
        QString text = QString::fromUtf8(m_skillHostProcess->readAllStandardError());
        // tracing_subscriber colours its output; the log file should not carry
        // the escape sequences.
        static const QRegularExpression ansi(QStringLiteral("\u001b\\[[0-9;]*m"));
        text.remove(ansi);
        const QStringList lines = text.split(QLatin1Char('\n'), Qt::SkipEmptyParts);
        for (const QString& line : lines) {
            LOG_WARN("SkillHost: {}", line.trimmed().toStdString());
        }
    });
    connect(m_skillHostProcess,
            QOverload<int, QProcess::ExitStatus>::of(&QProcess::finished),
            this, &SkillHostManager::onSkillHostFinished);
    connect(m_skillHostProcess, &QProcess::started, this, [this]() {
        LOG_INFO("SkillHostManager: skill-host started (pid={})", m_skillHostProcess->processId());
        emit skillHostStarted();
    });
    connect(m_skillHostProcess, &QProcess::errorOccurred, this, [skillHostPath](QProcess::ProcessError err) {
        if (err == QProcess::FailedToStart) {
            LOG_ERROR("SkillHostManager: failed to start skill-host at {}", skillHostPath.toStdString());
        }
    });

    m_skillHostProcess->start();
}

void SkillHostManager::stopSkillHost()
{
    if (!m_skillHostProcess) return;
    failPendingInternalRequests(
        QStringLiteral("Skill host stopped before the request completed"));

    if (m_skillHostProcess->state() == QProcess::NotRunning) {
        delete m_skillHostProcess;
        m_skillHostProcess = nullptr;
        m_readBuffer.clear();
        m_cachedTools = QJsonArray();
        return;
    }

    QProcess* proc = m_skillHostProcess;
    m_skillHostProcess = nullptr;
    m_readBuffer.clear();
    m_cachedTools = QJsonArray();

    proc->terminate();
    connect(proc, QOverload<int, QProcess::ExitStatus>::of(&QProcess::finished),
            proc, &QObject::deleteLater);

    QTimer::singleShot(3000, proc, [proc]() {
        if (proc->state() == QProcess::NotRunning) return;
        LOG_WARN("SkillHostManager: skill-host did not terminate, killing...");
        proc->kill();
    });
}

bool SkillHostManager::isRunning() const
{
    return m_skillHostProcess && m_skillHostProcess->state() == QProcess::Running;
}

void SkillHostManager::requestSystemInfo(
    std::function<void(const QJsonObject&, const QString&)> callback)
{
    if (!isRunning()) {
        if (callback) {
            callback(
                {},
                QStringLiteral("Skill host is not running"));
        }
        return;
    }

    const QString requestId =
        QStringLiteral("quickdesk-internal-%1")
            .arg(QUuid::createUuid().toString(QUuid::WithoutBraces));
    m_pendingInternalRequests.insert(requestId, std::move(callback));
    QTimer::singleShot(kInternalRequestTimeoutMs, this, [this, requestId]() {
        auto pending = m_pendingInternalRequests.take(requestId);
        if (pending) {
            pending({}, QStringLiteral("Internal skill request timed out"));
        }
    });

    QJsonObject request;
    request[QStringLiteral("id")] = requestId;
    request[QStringLiteral("type")] = QStringLiteral("toolCall");
    request[QStringLiteral("tool")] = QStringLiteral("get_system_info");
    request[QStringLiteral("args")] = QJsonObject{};
    if (!sendToSkillHost(request)) {
        auto pending = m_pendingInternalRequests.take(requestId);
        if (pending) {
            pending(
                {},
                QStringLiteral("Failed to send internal request to skill host"));
        }
    }
}

// ---- Private slots ----

void SkillHostManager::onSkillHostStdout()
{
    if (!m_skillHostProcess) return;

    m_readBuffer += m_skillHostProcess->readAllStandardOutput();

    // Process complete JSON lines (delimited by '\n')
    while (true) {
        int newlinePos = m_readBuffer.indexOf('\n');
        if (newlinePos < 0) break;

        QByteArray line = m_readBuffer.left(newlinePos).trimmed();
        m_readBuffer.remove(0, newlinePos + 1);

        if (line.isEmpty()) continue;

        QJsonParseError err;
        QJsonDocument doc = QJsonDocument::fromJson(line, &err);
        if (err.error != QJsonParseError::NoError || !doc.isObject()) {
            LOG_WARN("SkillHostManager: invalid JSON from skill-host: {}",
                     line.toStdString());
            continue;
        }

        handleSkillHostMessage(doc.object());
    }
}

void SkillHostManager::onSkillHostFinished(int exitCode, QProcess::ExitStatus exitStatus)
{
    Q_UNUSED(exitStatus);
    failPendingInternalRequests(
        QStringLiteral("Skill host exited before the request completed"));
    LOG_INFO("SkillHostManager: skill-host exited with code {}", exitCode);
}

void SkillHostManager::onMessageFromClient(const QString& jsonData)
{
    QJsonParseError err;
    QJsonDocument doc = QJsonDocument::fromJson(jsonData.toUtf8(), &err);
    if (err.error != QJsonParseError::NoError || !doc.isObject()) {
        LOG_WARN("SkillHostManager: invalid JSON from client: {}",
                 jsonData.toStdString());
        return;
    }

    if (!sendToSkillHost(doc.object())) {
        LOG_WARN("SkillHostManager: failed to forward client message to skill-host");
    }
}

// ---- Private helpers ----

bool SkillHostManager::sendToSkillHost(const QJsonObject& message)
{
    if (!m_skillHostProcess || m_skillHostProcess->state() != QProcess::Running) {
        LOG_WARN("SkillHostManager: skill-host not running, dropping message");
        return false;
    }

    QByteArray line = QJsonDocument(message).toJson(QJsonDocument::Compact);
    line += '\n';
    const qint64 written = m_skillHostProcess->write(line);
    if (written != line.size()) {
        LOG_WARN("SkillHostManager: short write to skill-host ({} of {})",
                 written,
                 line.size());
        return false;
    }
    return true;
}

void SkillHostManager::handleSkillHostMessage(const QJsonObject& message)
{
    if (handleInternalResponse(message)) {
        return;
    }

    QString type = message["type"].toString();

    if (type == "capabilitiesReady" || type == "capabilitiesChanged") {
        QJsonArray tools = message["tools"].toArray();
        m_cachedTools = tools;
        LOG_INFO("SkillHostManager: capabilities {}, {} tool(s)",
                 type.toStdString(), tools.size());
        emit capabilitiesChanged(tools);
        // Fall through to forward to connected clients
    }

    // Forward all messages (toolResult, capabilitiesReady, error, etc.) to the client.
    if (!m_hostManager) return;

    QByteArray bytes = QJsonDocument(message).toJson(QJsonDocument::Compact);
    QString jsonData = QString::fromUtf8(bytes);

    emit skillHostResponseReady(message);
    m_hostManager->sendSkillBridgeSend(jsonData);
}

bool SkillHostManager::handleInternalResponse(const QJsonObject& message)
{
    const QString requestId =
        message.value(QStringLiteral("id")).toString();
    auto pending = m_pendingInternalRequests.find(requestId);
    if (pending == m_pendingInternalRequests.end()) {
        return false;
    }

    auto callback = m_pendingInternalRequests.take(requestId);
    const QString type = message.value(QStringLiteral("type")).toString();
    if (type == QStringLiteral("toolError")) {
        const QString error =
            message.value(QStringLiteral("error")).toString(
                QStringLiteral("Internal system-info request failed"));
        if (callback) {
            callback({}, error);
        }
        return true;
    }

    const QJsonObject result =
        toolResultObject(message.value(QStringLiteral("result")));
    if (result.isEmpty()) {
        if (callback) {
            callback(
                {},
                QStringLiteral("Skill host returned an invalid system-info result"));
        }
        return true;
    }
    if (callback) {
        callback(result, QString());
    }
    return true;
}

void SkillHostManager::failPendingInternalRequests(const QString& error)
{
    const auto pending = m_pendingInternalRequests;
    m_pendingInternalRequests.clear();
    for (const auto& callback : pending) {
        if (callback) {
            callback({}, error);
        }
    }
}

} // namespace quickdesk
