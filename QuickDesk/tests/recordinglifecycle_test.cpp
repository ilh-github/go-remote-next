// Copyright 2026 QuickDesk Authors

#include "core/recordinglifecycle.h"

#include <QCoreApplication>
#include <QSignalSpy>

#include <iostream>

namespace {

void Expect(bool condition, const char* message) {
    if (!condition) {
        std::cerr << "FAIL: " << message << '\n';
        std::exit(1);
    }
}

} // namespace

int main(int argc, char** argv) {
    QCoreApplication app(argc, argv);

    {
        quickdesk::RecordingLifecycle lifecycle;
        QSignalSpy changes(&lifecycle, &quickdesk::RecordingLifecycle::recordingChanged);
        Expect(!lifecycle.isRecording(), "no recording is active by default");

        lifecycle.started(QStringLiteral("device-a"));
        Expect(lifecycle.isRecording(), "starting marks the lifecycle as recording");
        Expect(lifecycle.activeDevice() == QStringLiteral("device-a"),
               "the active device is remembered");

        // Non-terminal states must never finalise a recording: a reconnecting
        // session resumes and the operator expects the file to keep growing.
        Expect(lifecycle.shouldStopFor(QStringLiteral("device-a"),
                                       QStringLiteral("connected")).isEmpty(),
               "connected does not stop the recording");
        Expect(lifecycle.shouldStopFor(QStringLiteral("device-a"),
                                       QStringLiteral("reconnecting")).isEmpty(),
               "reconnecting does not stop the recording");
        Expect(lifecycle.isRecording(), "the recording is still active");

        Expect(lifecycle.shouldStopFor(QStringLiteral("other-device"),
                                       QStringLiteral("disconnected")).isEmpty(),
               "another device's disconnect does not stop the recording");
        Expect(lifecycle.isRecording(), "the recording survives an unrelated disconnect");

        Expect(lifecycle.shouldStopFor(QStringLiteral("device-a"),
                                       QStringLiteral("disconnected"))
                   == QStringLiteral("device-a"),
               "disconnected finalises the active recording");
        Expect(!lifecycle.isRecording(), "the lifecycle is cleared after finalising");
        Expect(changes.count() == 2, "exactly one start and one stop signal are emitted");

        Expect(lifecycle.shouldStopFor(QStringLiteral("device-a"),
                                       QStringLiteral("failed")).isEmpty(),
               "a repeated terminal event is a no-op");
    }

    {
        quickdesk::RecordingLifecycle lifecycle;
        lifecycle.started(QStringLiteral("device-b"));
        Expect(lifecycle.shouldStopFor(QStringLiteral("device-b"),
                                       QStringLiteral("failed"))
                   == QStringLiteral("device-b"),
               "failed finalises the active recording");

        lifecycle.started(QStringLiteral("device-b"));
        Expect(lifecycle.shouldStopFor(QStringLiteral("device-b"),
                                       QStringLiteral("closed"))
                   == QStringLiteral("device-b"),
               "closed finalises the next recording on the same device");
    }

    {
        // The controller calls the same predicate the lifecycle uses, so a
        // non-terminal state must never reach stopRecording().
        Expect(quickdesk::isTerminalSignalingState(QStringLiteral("disconnected")),
               "disconnected is terminal");
        Expect(quickdesk::isTerminalSignalingState(QStringLiteral("failed")),
               "failed is terminal");
        Expect(quickdesk::isTerminalSignalingState(QStringLiteral("closed")),
               "closed is terminal");
        Expect(quickdesk::isTerminalSignalingState(QStringLiteral("error")),
               "error is terminal");
        Expect(!quickdesk::isTerminalSignalingState(QStringLiteral("connected")),
               "connected is not terminal");
        Expect(!quickdesk::isTerminalSignalingState(QStringLiteral("reconnecting")),
               "reconnecting is not terminal");
        Expect(!quickdesk::isTerminalSignalingState(QStringLiteral("connecting")),
               "connecting is not terminal");
    }

    std::cout << "recording lifecycle tests passed\n";
    return 0;
}
