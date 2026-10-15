// Copyright 2026 QuickDesk Authors
// Control-side bridge to the host's `file-ops` skill.
//
// The host already ships a file-ops skill (list/read/write/mkdir/move/info) and
// the skill bridge already carries tool calls between the two Qt apps. This
// class is the UI-facing, asynchronous wrapper around that bridge: it never
// blocks the GUI thread (the MCP-facing SkillHandler does its own blocking
// wait on worker threads) and turns tool results into signals a Drawer can bind
// to.

#ifndef QUICKDESK_MANAGER_REMOTEFILEMANAGER_H
#define QUICKDESK_MANAGER_REMOTEFILEMANAGER_H

#include <QJsonArray>
#include <QJsonObject>
#include <QObject>
#include <QString>

#include <QHash>

namespace quickdesk {

class ClientManager;

class RemoteFileManager : public QObject {
    Q_OBJECT
    Q_PROPERTY(bool busy READ isBusy NOTIFY busyChanged)
    Q_PROPERTY(QString lastError READ lastError NOTIFY lastErrorChanged)

public:
    explicit RemoteFileManager(ClientManager* clientManager, QObject* parent = nullptr);

    bool isBusy() const { return m_busy; }
    QString lastError() const { return m_lastError; }

    /// List a directory. Emits listingReady() with the entries.
    Q_INVOKABLE void listDirectory(const QString& deviceId, const QString& path);
    /// Create a directory (and missing parents).
    Q_INVOKABLE void createDirectory(const QString& deviceId, const QString& path);
    /// Rename/move within the same directory.
    Q_INVOKABLE void renamePath(const QString& deviceId, const QString& path,
                                const QString& newName);
    /// Soft delete: create `<parent>/.quickdesk-trash` and move the entry there.
    /// file-ops has no delete tool on purpose; a reversible move is the safe
    /// equivalent and works with the tools every host already has.
    Q_INVOKABLE void trashPath(const QString& deviceId, const QString& path);
    /// Read a text file (for the preview pane).
    Q_INVOKABLE void readFile(const QString& deviceId, const QString& path);

signals:
    void listingReady(const QString& deviceId, const QString& path, const QJsonArray& entries);
    void fileReady(const QString& deviceId, const QString& path, const QString& content);
    /// op is one of: list, mkdir, rename, trash, read.
    void operationDone(const QString& op, const QString& deviceId, bool ok,
                       const QString& message);
    void busyChanged();
    void lastErrorChanged();

private slots:
    void onSkillResponse(const QString& deviceId, const QJsonObject& response);

private:
    struct Pending {
        QString op;
        QString deviceId;
        QString path;
        QString destination;
        QString newName;
        // trashPath is two calls: mkdir the trash directory, then move into it.
        bool trashSecondStage = false;
    };

    void sendTool(const QString& deviceId, const QString& tool, const QJsonObject& args,
                  const Pending& pending);
    void finish(const Pending& pending, bool ok, const QString& message);
    void setBusy(bool busy);
    void setLastError(const QString& message);
    QString nextRequestId();

    ClientManager* m_clientManager = nullptr;
    QHash<QString, Pending> m_pending;
    int m_nextId = 1;
    bool m_busy = false;
    QString m_lastError;
};

} // namespace quickdesk

#endif // QUICKDESK_MANAGER_REMOTEFILEMANAGER_H
