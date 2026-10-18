// Copyright 2026 QuickDesk Authors
// SkillHostManager — manages the quickdesk-skill-host subprocess on the host machine.
//
// Communication path:
//   Qt SkillHostManager ←→ quickdesk-skill-host (stdin/stdout JSON Lines)
//   Qt SkillHostManager ←→ HostManager ←→ Chromium ←→ WebRTC ←→ Client

#ifndef QUICKDESK_MANAGER_SKILLHOSTMANAGER_H
#define QUICKDESK_MANAGER_SKILLHOSTMANAGER_H

#include <QByteArray>
#include <QJsonArray>
#include <QJsonObject>
#include <QObject>
#include <QPointer>
#include <QProcess>
#include <QString>
#include <QHash>

#include <functional>

namespace quickdesk {

class HostManager;

/**
 * @brief Manages the quickdesk-skill-host subprocess and bridges messages between
 *        the skill host and the remote client (via HostManager → Chromium → WebRTC).
 *
 * Protocol (skill-host ↔ SkillHostManager, JSON Lines on stdin/stdout):
 *   Client → Host → SkillHostManager → skill-host stdin:
 *     {"id":"req-1","type":"toolCall","tool":"run_shell","args":{"cmd":"..."}}
 *   skill-host stdout → SkillHostManager → Host → Client:
 *     {"id":"req-1","type":"toolResult","result":"..."}
 *   skill-host stdout → SkillHostManager (capability report):
 *     {"type":"capabilitiesChanged","tools":[...]}
 */
class SkillHostManager : public QObject {
    Q_OBJECT

public:
    explicit SkillHostManager(QObject* parent = nullptr);
    ~SkillHostManager() override;

    // Wire up to the HostManager (must be called before startSkillHost).
    void setHostManager(HostManager* hostManager);

    // Start the skill-host subprocess.  skillHostPath is the path to the
    // quickdesk-skill-host binary; skillsDirs are the directories containing
    // skill SKILL.md files.
    void startSkillHost(const QString& skillHostPath, const QStringList& skillsDirs);

    // Stop the skill-host subprocess gracefully.
    void stopSkillHost();

    bool isRunning() const;
    bool supportsCapability(const QString& name) const;

    // Send a plain-text chat message to every connected controller. The
    // message is not persisted and never reaches the skill-host subprocess.
    bool sendChatMessage(const QString& text);

    // Query sys-info through the host-side skill bridge without exposing the
    // response to connected remote clients.
    void requestSystemInfo(
        std::function<void(const QJsonObject& result, const QString& error)> callback);

signals:
    void skillHostStarted();

    // Emitted when the skill host reports its available tools (on start or hot-reload).
    void capabilitiesChanged(const QJsonArray& tools);

    // Emitted when the skill host sends a response back to the client.
    // SkillHostManager automatically forwards it via HostManager::sendSkillBridgeSend.
    void skillHostResponseReady(const QJsonObject& response);

    // Emitted for both controller-originated and host-originated chat so the
    // controlled machine's UI can show and answer the conversation.
    void chatMessageReceived(const QString& from,
                             const QString& id,
                             const QString& text,
                             const QString& sentAt);

private slots:
    // Called when the skill-host process writes to stdout.
    void onSkillHostStdout();

    // Called when the skill-host process exits.
    void onSkillHostFinished(int exitCode, QProcess::ExitStatus exitStatus);

    // Called when HostManager receives a skillMessage from the remote client.
    void onMessageFromClient(const QString& jsonData);

private:
    // Send a JSON object to the skill host's stdin as a single JSON Line.
    bool sendToSkillHost(const QJsonObject& message);

    // Handle a complete JSON object received from the skill host.
    void handleSkillHostMessage(const QJsonObject& message);
    bool handleBuiltInClientMessage(const QJsonObject& message);
    bool handleInternalResponse(const QJsonObject& message);
    void failPendingInternalRequests(const QString& error);
    void sendBridgeMessage(const QJsonObject& message);
    QJsonArray advertisedCapabilities(const QJsonArray& tools) const;

    QPointer<HostManager> m_hostManager;
    QPointer<QProcess>    m_skillHostProcess;
    QByteArray            m_readBuffer;  // accumulates partial lines from stdout
    QJsonArray            m_cachedTools; // tools from the last capabilitiesReady
    QHash<QString,
          std::function<void(const QJsonObject&, const QString&)>>
        m_pendingInternalRequests;
};

} // namespace quickdesk

#endif // QUICKDESK_MANAGER_SKILLHOSTMANAGER_H
