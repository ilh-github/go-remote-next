// Copyright 2026 QuickDesk Authors

#include "manager/HostManager.h"
#include "manager/SkillHostManager.h"

#include <QCoreApplication>
#include <QElapsedTimer>
#include <QEventLoop>
#include <QJsonObject>
#include <QSignalSpy>
#include <QThread>

#include <cstdlib>
#include <functional>
#include <iostream>

namespace {

void Expect(bool condition, const QString& message)
{
    if (!condition) {
        std::cerr << "FAIL: " << message.toStdString() << '\n';
        std::exit(1);
    }
}

bool WaitUntil(const std::function<bool()>& predicate, int timeoutMs)
{
    QElapsedTimer timer;
    timer.start();
    while (!predicate() && timer.elapsed() < timeoutMs) {
        QCoreApplication::processEvents(QEventLoop::AllEvents, 50);
        QThread::msleep(10);
    }
    return predicate();
}

}  // namespace

int main(int argc, char** argv)
{
    QCoreApplication app(argc, argv);
    Expect(argc >= 2, QStringLiteral("fake skill-host path is provided"));

    quickdesk::HostManager hostManager;
    quickdesk::SkillHostManager manager;
    manager.setHostManager(&hostManager);
    QSignalSpy forwardedSpy(
        &manager, &quickdesk::SkillHostManager::skillHostResponseReady);

    manager.startSkillHost(QString::fromLocal8Bit(argv[1]), {});
    Expect(WaitUntil([&]() { return manager.isRunning(); }, 5000),
           QStringLiteral("skill-host starts"));

    bool completed = false;
    QString error;
    QJsonObject systemInfo;
    manager.requestSystemInfo(
        [&](const QJsonObject& result, const QString& requestError) {
            completed = true;
            systemInfo = result;
            error = requestError;
        });
    Expect(WaitUntil([&]() { return completed; }, 5000),
           QStringLiteral("internal system-info request completes"));
    Expect(error.isEmpty(),
           QStringLiteral("internal system-info request succeeds"));
    Expect(systemInfo.value(QStringLiteral("hostname")).toString()
               == QStringLiteral("fake-host"),
           QStringLiteral("internal callback receives the parsed tool result"));
    Expect(forwardedSpy.count() == 0,
           QStringLiteral("internal system-info response is not forwarded"));
    Expect(manager.supportsCapability(QStringLiteral("session_chat")),
           QStringLiteral("session chat is advertised without a skill tool"));

    QSignalSpy chatReceivedSpy(
        &manager, &quickdesk::SkillHostManager::chatMessageReceived);
    hostManager.skillMessage(QStringLiteral(
        R"({"id":"chat-in-1","type":"chatSend","text":"hello from controller"})"));
    Expect(WaitUntil([&]() { return chatReceivedSpy.count() == 1; }, 1000),
           QStringLiteral("controller chat reaches the host UI"));
    const QList<QVariant> inboundArgs = chatReceivedSpy.takeFirst();
    Expect(inboundArgs.at(0).toString() == QStringLiteral("controller"),
           QStringLiteral("controller chat is labelled by sender"));
    Expect(inboundArgs.at(1).toString() == QStringLiteral("chat-in-1"),
           QStringLiteral("controller chat keeps the request id"));
    Expect(inboundArgs.at(2).toString()
               == QStringLiteral("hello from controller"),
           QStringLiteral("controller chat keeps the text"));
    Expect(!inboundArgs.at(3).toString().isEmpty(),
           QStringLiteral("host assigns a receive timestamp"));

    bool sawAcceptedAck = false;
    bool sawBroadcast = false;
    for (const QList<QVariant>& arguments : forwardedSpy) {
        const QJsonObject response = arguments.first().toJsonObject();
        if (response.value(QStringLiteral("type")).toString()
                == QStringLiteral("chatAck")
            && response.value(QStringLiteral("id")).toString()
                == QStringLiteral("chat-in-1")) {
            sawAcceptedAck =
                response.value(QStringLiteral("accepted")).toBool();
        }
        if (response.value(QStringLiteral("type")).toString()
                == QStringLiteral("chatMessage")
            && response.value(QStringLiteral("id")).toString()
                == QStringLiteral("chat-in-1")) {
            sawBroadcast = true;
        }
    }
    Expect(sawAcceptedAck,
           QStringLiteral("accepted controller chat gets a positive ack"));
    Expect(sawBroadcast,
           QStringLiteral("accepted controller chat is broadcast to controllers"));

    const int responsesBeforeHostReply = forwardedSpy.count();
    manager.sendChatMessage(QStringLiteral("hello from host"));
    Expect(WaitUntil(
               [&]() {
                   for (int i = responsesBeforeHostReply;
                        i < forwardedSpy.count(); ++i) {
                       const QJsonObject response =
                           forwardedSpy.at(i).first().toJsonObject();
                       if (response.value(QStringLiteral("type")).toString()
                               == QStringLiteral("chatMessage")
                           && response.value(QStringLiteral("from")).toString()
                               == QStringLiteral("host")) {
                           return true;
                       }
                   }
                   return false;
               },
               1000),
           QStringLiteral("host chat is broadcast to controllers"));
    Expect(chatReceivedSpy.count() == 1,
           QStringLiteral("host chat is also visible to the host UI"));
    const QList<QVariant> outboundArgs = chatReceivedSpy.takeFirst();
    Expect(outboundArgs.at(0).toString() == QStringLiteral("host"),
           QStringLiteral("host chat is labelled by sender"));
    Expect(outboundArgs.at(2).toString() == QStringLiteral("hello from host"),
           QStringLiteral("host chat keeps the text"));

    const QString clientRequest =
        QStringLiteral(
            R"({"id":"client-1","type":"toolCall","tool":"list_processes","args":{}})");
    hostManager.skillMessage(clientRequest);
    Expect(WaitUntil(
               [&]() {
                   for (const QList<QVariant>& arguments : forwardedSpy) {
                       if (!arguments.isEmpty()
                           && arguments.first().toJsonObject()
                                  .value(QStringLiteral("id")).toString()
                               == QStringLiteral("client-1")) {
                           return true;
                       }
                   }
                   return false;
               },
               5000),
           QStringLiteral("client-originated responses are still forwarded"));

    manager.stopSkillHost();
    bool stoppedError = false;
    manager.requestSystemInfo(
        [&](const QJsonObject&, const QString& requestError) {
            stoppedError = !requestError.isEmpty();
        });
    Expect(stoppedError,
           QStringLiteral("stopped skill-host reports an internal request error"));

    std::cout << "skill host manager internal request tests passed\n";
    return 0;
}
