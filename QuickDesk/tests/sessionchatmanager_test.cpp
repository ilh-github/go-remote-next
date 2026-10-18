// Copyright 2026 QuickDesk Authors

#include "manager/HostManager.h"
#include "manager/SessionChatManager.h"
#include "manager/SkillHostManager.h"

#include <QCoreApplication>
#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QSignalSpy>

#include <cstdlib>
#include <iostream>

namespace {

void Expect(bool condition, const char* message)
{
    if (!condition) {
        std::cerr << "FAIL: " << message << '\n';
        std::exit(1);
    }
}

QJsonObject parseSent(const QString& json)
{
    return QJsonDocument::fromJson(json.toUtf8()).object();
}

}  // namespace

int main(int argc, char** argv)
{
    QCoreApplication app(argc, argv);

    quickdesk::HostManager hostManager;
    quickdesk::SkillHostManager skillHost;
    skillHost.setHostManager(&hostManager);

    QString sentDeviceId;
    QJsonObject sentPayload;
    quickdesk::SessionChatManager manager(
        [&](const QString& deviceId, const QString& json) {
            sentDeviceId = deviceId;
            sentPayload = parseSent(json);
            return true;
        },
        &skillHost);

    QSignalSpy addedSpy(&manager, &quickdesk::SessionChatManager::messageAdded);
    QSignalSpy hostSpy(
        &manager, &quickdesk::SessionChatManager::hostMessageAdded);

    Expect(manager.sendToHost(QStringLiteral("device-1"),
                              QStringLiteral("  hello  ")),
           "controller chat is accepted");
    Expect(sentDeviceId == QStringLiteral("device-1"),
           "controller chat targets the active device");
    Expect(sentPayload.value(QStringLiteral("type")).toString()
               == QStringLiteral("chatSend"),
           "controller chat uses the built-in chatSend type");
    Expect(sentPayload.value(QStringLiteral("text")).toString()
               == QStringLiteral("hello"),
           "controller chat is trimmed");
    const QString requestId =
        sentPayload.value(QStringLiteral("id")).toString();
    Expect(!requestId.isEmpty(), "controller chat has a request id");

    manager.ingestSkillResponse(
        QStringLiteral("device-1"),
        QJsonObject{
            {QStringLiteral("id"), requestId},
            {QStringLiteral("type"), QStringLiteral("chatMessage")},
            {QStringLiteral("from"), QStringLiteral("controller")},
            {QStringLiteral("text"), QStringLiteral("hello")},
            {QStringLiteral("sentAt"),
             QStringLiteral("2026-10-18T00:00:00.000Z")},
        });
    Expect(addedSpy.count() == 1,
           "the echoed controller chat reaches the UI");
    const QList<QVariant> outgoing = addedSpy.takeFirst();
    Expect(outgoing.at(0).toString() == QStringLiteral("device-1"),
           "controller chat is scoped to its device");
    Expect(outgoing.at(1).toString() == QStringLiteral("controller"),
           "controller chat keeps its sender");
    Expect(outgoing.at(4).toBool(),
           "the echoed controller chat is marked outgoing");

    manager.sendToControllers(QStringLiteral("hello from host"));
    Expect(hostSpy.count() == 1,
           "host chat reaches the controlled machine UI");
    const QList<QVariant> hostMessage = hostSpy.takeFirst();
    Expect(hostMessage.at(0).toString() == QStringLiteral("host"),
           "host chat keeps its sender");
    Expect(hostMessage.at(1).toString() == QStringLiteral("hello from host"),
           "host chat keeps its text");

    manager.ingestSkillResponse(
        QStringLiteral("device-1"),
        QJsonObject{
            {QStringLiteral("type"), QStringLiteral("capabilitiesReady")},
            {QStringLiteral("tools"),
             QJsonArray{
                 QJsonObject{
                     {QStringLiteral("name"),
                      QStringLiteral("session_chat")},
                 },
             }},
        });
    Expect(manager.isAvailable(QStringLiteral("device-1")),
           "session_chat capability enables the chat UI");
    manager.ingestSkillResponse(
        QStringLiteral("device-1"),
        QJsonObject{
            {QStringLiteral("id"), QStringLiteral("host-chat-1")},
            {QStringLiteral("type"), QStringLiteral("chatMessage")},
            {QStringLiteral("from"), QStringLiteral("host")},
            {QStringLiteral("text"), QStringLiteral("hi")},
            {QStringLiteral("sentAt"),
             QStringLiteral("2026-10-18T00:00:01.000Z")},
        });
    Expect(manager.unreadCount(QStringLiteral("device-1")) == 1,
           "incoming host chat increments unread");
    manager.markRead(QStringLiteral("device-1"));
    Expect(manager.unreadCount(QStringLiteral("device-1")) == 0,
           "opening the chat clears unread");

    std::cout << "session chat manager tests passed\n";
    return 0;
}
