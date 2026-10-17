// Copyright 2026 QuickDesk Authors
// Control-side bridge to the host's `file-ops` skill.

#include "RemoteFileManager.h"

#include "ClientManager.h"
#include "infra/log/log.h"

#include <QJsonDocument>
#include <QJsonValue>

namespace quickdesk {

namespace {

// The control machine may be Windows, macOS or Linux while the host is any of
// them, so path splitting cannot rely on the local separator.
int lastSeparator(const QString& path)
{
    const int slash = path.lastIndexOf(QLatin1Char('/'));
    const int backslash = path.lastIndexOf(QLatin1Char('\\'));
    return qMax(slash, backslash);
}

QString parentOf(const QString& path)
{
    const int index = lastSeparator(path);
    if (index < 0) {
        return QString();
    }
    if (index == 0) {
        return path.left(1);
    }
    // "C:\dir" -> "C:" (the drive root keeps its colon).
    if (index == 2 && path.at(1) == QLatin1Char(':')) {
        return path.left(2);
    }
    return path.left(index);
}

QString joinPath(const QString& directory, const QString& name)
{
    if (directory.isEmpty()) {
        return name;
    }
    const QChar last = directory.at(directory.size() - 1);
    const QChar separator =
        (last == QLatin1Char('\\') || directory.contains(QLatin1Char('\\')))
            && !directory.contains(QLatin1Char('/'))
        ? QLatin1Char('\\')
        : QLatin1Char('/');
    if (last == QLatin1Char('/') || last == QLatin1Char('\\')) {
        return directory + name;
    }
    return directory + separator + name;
}

} // namespace

RemoteFileManager::RemoteFileManager(ClientManager* clientManager, QObject* parent)
    : QObject(parent)
    , m_clientManager(clientManager) {
    if (m_clientManager) {
        connect(m_clientManager, &ClientManager::skillBridgeResponseReceived,
                this, &RemoteFileManager::onSkillResponse);
    }
}

void RemoteFileManager::listDirectory(const QString& deviceId, const QString& path) {
    Pending pending;
    pending.op = QStringLiteral("list");
    pending.deviceId = deviceId;
    pending.path = path;
    QJsonObject args;
    args[QStringLiteral("path")] = path;
    sendTool(deviceId, QStringLiteral("list_directory"), args, pending);
}

void RemoteFileManager::createDirectory(const QString& deviceId, const QString& path) {
    Pending pending;
    pending.op = QStringLiteral("mkdir");
    pending.deviceId = deviceId;
    pending.path = path;
    QJsonObject args;
    args[QStringLiteral("path")] = path;
    sendTool(deviceId, QStringLiteral("create_directory"), args, pending);
}

void RemoteFileManager::renamePath(const QString& deviceId, const QString& path,
                                   const QString& newName) {
    if (newName.isEmpty() || newName.contains(QLatin1Char('/'))
        || newName.contains(QLatin1Char('\\'))) {
        setLastError(tr("New name must not contain a path separator"));
        emit operationDone(QStringLiteral("rename"), deviceId, false, m_lastError);
        return;
    }
    Pending pending;
    pending.op = QStringLiteral("rename");
    pending.deviceId = deviceId;
    pending.path = path;
    const QString destination = joinPath(parentOf(path), newName);
    QJsonObject args;
    args[QStringLiteral("source")] = path;
    args[QStringLiteral("destination")] = destination;
    sendTool(deviceId, QStringLiteral("move_file"), args, pending);
}

void RemoteFileManager::trashPath(const QString& deviceId, const QString& path) {
    Pending pending;
    pending.op = QStringLiteral("trash");
    pending.deviceId = deviceId;
    pending.path = path;
    QJsonObject args;
    args[QStringLiteral("path")] = path;
    sendTool(deviceId, QStringLiteral("trash_path"), args, pending);
}

void RemoteFileManager::readFile(const QString& deviceId, const QString& path) {
    Pending pending;
    pending.op = QStringLiteral("read");
    pending.deviceId = deviceId;
    pending.path = path;
    QJsonObject args;
    args[QStringLiteral("path")] = path;
    sendTool(deviceId, QStringLiteral("read_file"), args, pending);
}

void RemoteFileManager::sendTool(const QString& deviceId, const QString& tool,
                                 const QJsonObject& args, const Pending& pending) {
    if (!m_clientManager || deviceId.isEmpty()) {
        setLastError(tr("No active connection to the host"));
        emit operationDone(pending.op, deviceId, false, m_lastError);
        return;
    }

    const QString id = nextRequestId();
    QJsonObject payload;
    payload[QStringLiteral("id")] = id;
    payload[QStringLiteral("type")] = QStringLiteral("toolCall");
    payload[QStringLiteral("tool")] = tool;
    payload[QStringLiteral("args")] = args;

    m_pending.insert(id, pending);
    setBusy(true);
    const QString jsonData =
        QString::fromUtf8(QJsonDocument(payload).toJson(QJsonDocument::Compact));
    LOG_INFO("RemoteFileManager: sending {} to {} (id={})", tool.toStdString(),
             deviceId.toStdString(), id.toStdString());
    m_clientManager->sendSkillCommand(deviceId, jsonData);
}

void RemoteFileManager::onSkillResponse(const QString& deviceId,
                                        const QJsonObject& response) {
    const QString id = response.value(QStringLiteral("id")).toString();
    if (id.isEmpty()) {
        return;
    }
    auto it = m_pending.find(id);
    if (it == m_pending.end()) {
        return; // Not ours (the MCP bridge uses its own ids).
    }
    Pending pending = it.value();
    m_pending.erase(it);

    const QString type = response.value(QStringLiteral("type")).toString();
    if (type == QLatin1String("toolError")) {
        finish(pending, false, response.value(QStringLiteral("error")).toString());
        return;
    }

    QJsonValue result = response.value(QStringLiteral("result"));
    // The skill host passes the tool's JSON through; a text-only skill would
    // hand back a string, so accept both shapes.
    if (result.isString()) {
        const QJsonDocument parsed =
            QJsonDocument::fromJson(result.toString().toUtf8());
        if (parsed.isObject()) {
            result = parsed.object();
        }
    }
    const QJsonObject resultObject = result.toObject();

    if (pending.op == QLatin1String("list")) {
        const QJsonArray entries = resultObject.value(QStringLiteral("entries")).toArray();
        const QString path = resultObject.value(QStringLiteral("path")).toString(pending.path);
        emit listingReady(deviceId, path, entries);
        finish(pending, true, QString());
        return;
    }

    if (pending.op == QLatin1String("read")) {
        emit fileReady(deviceId, pending.path,
                       resultObject.value(QStringLiteral("content")).toString());
        finish(pending, true, QString());
        return;
    }

    finish(pending, true, QString());
}

void RemoteFileManager::finish(const Pending& pending, bool ok, const QString& message) {
    if (!ok) {
        setLastError(message);
    }
    // busy only while something is still in flight
    setBusy(!m_pending.isEmpty());
    if (!ok) {
        LOG_WARN("RemoteFileManager: {} {} failed: {}", pending.op.toStdString(),
                 pending.path.toStdString(), message.toStdString());
    }
    emit operationDone(pending.op, pending.deviceId, ok, message);
}

void RemoteFileManager::setBusy(bool busy) {
    if (m_busy == busy) {
        return;
    }
    m_busy = busy;
    emit busyChanged();
}

void RemoteFileManager::setLastError(const QString& message) {
    if (m_lastError == message) {
        return;
    }
    m_lastError = message;
    emit lastErrorChanged();
}

QString RemoteFileManager::nextRequestId() {
    return QStringLiteral("qt-files-%1").arg(m_nextId++);
}

} // namespace quickdesk
