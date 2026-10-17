// Copyright 2026 QuickDesk Authors
// RecordingLifecycle implementation.

#include "recordinglifecycle.h"

namespace quickdesk {

// Terminal signalling states for a device connection. A recording that is still
// running when the connection reaches one of these will never receive frames
// again, so it must be finalised now or it is lost. Exported so the controller
// and its tests share one definition of "the session is over".
bool isTerminalSignalingState(const QString& state) {
    return state == QLatin1String("disconnected")
        || state == QLatin1String("failed")
        || state == QLatin1String("closed")
        || state == QLatin1String("error");
}

RecordingLifecycle::RecordingLifecycle(QObject* parent)
    : QObject(parent) {}

void RecordingLifecycle::started(const QString& deviceId) {
    const QString next = deviceId.trimmed();
    if (m_activeDevice == next) {
        return;
    }
    m_activeDevice = next;
    emit recordingChanged();
}

QString RecordingLifecycle::shouldStopFor(const QString& deviceId, const QString& state) {
    if (m_activeDevice.isEmpty() || deviceId != m_activeDevice) {
        return QString();
    }
    if (!isTerminalSignalingState(state)) {
        return QString();
    }
    const QString finished = m_activeDevice;
    m_activeDevice.clear();
    emit recordingChanged();
    return finished;
}

} // namespace quickdesk
