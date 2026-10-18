// Copyright 2026 QuickDesk Authors

#include "core/controlplaneevent.h"

#include <QJsonObject>

#include <cstdlib>
#include <iostream>

namespace {

void Expect(bool condition, const char* message) {
    if (!condition) {
        std::cerr << "FAIL: " << message << '\n';
        std::exit(1);
    }
}

}  // namespace

int main() {
    Expect(quickdesk::isWorkflowTriggerEvent(
               QStringLiteral("device.access.denied")),
           "device access denial is an automation event");
    Expect(quickdesk::isWorkflowTriggerEvent(
               QStringLiteral("remote_session.started")),
           "remote session start is an automation event");
    Expect(!quickdesk::isWorkflowTriggerEvent(
               QStringLiteral("device.access_code.changed")),
           "access code changes must not be broadcast as automation events");
    Expect(!quickdesk::isWorkflowTriggerEvent(
               QStringLiteral("favorite.updated")),
           "unrelated domain events must not be broadcast");

    const QJsonObject message{
        {QStringLiteral("id"), QStringLiteral("evt-1")},
        {QStringLiteral("type"), QStringLiteral("device.online.changed")},
        {QStringLiteral("device_id"), QStringLiteral("device-1")},
        {QStringLiteral("server_rev"), 42},
        {QStringLiteral("data"), QJsonObject{{QStringLiteral("online"), true}}},
        {QStringLiteral("unexpected_secret"), QStringLiteral("do-not-forward")},
    };
    const QJsonObject event = quickdesk::workflowTriggerEventData(message);
    Expect(event.value(QStringLiteral("type")).toString() ==
               QStringLiteral("device.online.changed"),
           "event type is preserved");
    Expect(event.value(QStringLiteral("server_rev")).toInt() == 42,
           "server revision is preserved");
    Expect(event.value(QStringLiteral("data")).toObject().value(
               QStringLiteral("online")).toBool(),
           "event payload is preserved");
    Expect(!event.contains(QStringLiteral("unexpected_secret")),
           "unknown envelope fields are stripped");

    const QJsonObject nestedDevice = quickdesk::workflowTriggerEventData(QJsonObject{
        {QStringLiteral("type"), QStringLiteral("device.metric.alert")},
        {QStringLiteral("data"),
         QJsonObject{{QStringLiteral("device_id"), QStringLiteral("device-2")}}},
    });
    Expect(nestedDevice.value(QStringLiteral("device_id")).toString() ==
               QStringLiteral("device-2"),
           "device id is normalized from the event payload");

    const QJsonObject ignored = quickdesk::workflowTriggerEventData(QJsonObject{
        {QStringLiteral("type"), QStringLiteral("device.access_code.changed")},
    });
    Expect(ignored.isEmpty(), "non-whitelisted events produce no forwarded data");

    quickdesk::ControlPlaneEventReplayBuffer replayBuffer(2);
    replayBuffer.append(QJsonObject{
        {QStringLiteral("id"), QStringLiteral("evt-1")},
        {QStringLiteral("type"), QStringLiteral("device.online.changed")},
        {QStringLiteral("server_rev"), 1},
    });
    replayBuffer.append(QJsonObject{
        {QStringLiteral("id"), QStringLiteral("evt-2")},
        {QStringLiteral("type"), QStringLiteral("device.online.changed")},
        {QStringLiteral("server_rev"), 2},
    });
    replayBuffer.append(QJsonObject{
        {QStringLiteral("id"), QStringLiteral("evt-3")},
        {QStringLiteral("type"), QStringLiteral("device.online.changed")},
        {QStringLiteral("server_rev"), 3},
    });
    replayBuffer.append(QJsonObject{
        {QStringLiteral("id"), QStringLiteral("evt-3")},
        {QStringLiteral("type"), QStringLiteral("device.online.changed")},
        {QStringLiteral("server_rev"), 3},
    });
    const QList<QJsonObject> replayed = replayBuffer.eventsAfter(1);
    Expect(replayed.size() == 2, "replay buffer keeps only the configured capacity");
    Expect(replayed.first().value(QStringLiteral("server_rev")).toInt() == 2,
           "replay buffer returns events newer than the cursor");
    Expect(replayed.last().value(QStringLiteral("server_rev")).toInt() == 3,
           "replay buffer preserves event order");
    Expect(replayBuffer.size() == 2, "duplicate event ids are not buffered twice");
    Expect(quickdesk::controlPlaneReplaySinceRev(QJsonObject{
               {QStringLiteral("since_server_rev"), -4}})
               == 0,
           "negative replay cursors are normalized to zero");
    Expect(quickdesk::controlPlaneReplaySinceRev(QJsonObject{
               {QStringLiteral("since_server_rev"), 42}})
               == 42,
           "positive replay cursors are preserved");

    std::cout << "control plane event tests passed\n";
    return 0;
}
