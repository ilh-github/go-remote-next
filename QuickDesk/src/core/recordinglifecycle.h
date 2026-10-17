// Copyright 2026 QuickDesk Authors
// Decides when a control-side recording must be finalised.

#ifndef QUICKDESK_CORE_RECORDINGLIFECYCLE_H
#define QUICKDESK_CORE_RECORDINGLIFECYCLE_H

#include <QHash>
#include <QObject>
#include <QString>

namespace quickdesk {

/// Terminal signalling states: a recording must be finalised when the
/// connection reaches one of these, or the file grows forever.
bool isTerminalSignalingState(const QString& state);

/**
 * @brief Tracks which device each active recording belongs to.
 *
 * A control-side recording is only useful if it is finalised and uploaded. The
 * recorder itself has no idea a session ended: the signalling layer knows. This
 * class answers one question - "does this signalling event end a recording?" -
 * so the controller can call stopRecording() exactly once instead of leaving a
 * file growing forever.
 */
class RecordingLifecycle : public QObject {
    Q_OBJECT
    Q_PROPERTY(bool recording READ isRecording NOTIFY recordingChanged)

public:
    explicit RecordingLifecycle(QObject* parent = nullptr);

    bool isRecording() const { return !m_activeDevice.isEmpty(); }
    QString activeDevice() const { return m_activeDevice; }

    Q_INVOKABLE void started(const QString& deviceId);
    /// @return the device whose recording must be finalised, or an empty string.
    Q_INVOKABLE QString shouldStopFor(const QString& deviceId, const QString& state);

signals:
    void recordingChanged();

private:
    QString m_activeDevice;
};

} // namespace quickdesk

#endif // QUICKDESK_CORE_RECORDINGLIFECYCLE_H
