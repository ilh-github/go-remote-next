// Copyright 2026 QuickDesk Authors

#ifndef QUICKDESK_CORE_HOSTREADINESS_H
#define QUICKDESK_CORE_HOSTREADINESS_H

namespace quickdesk {

// What the Host did with the last connect attempt.
enum class AttemptOutcome {
    // No attempt has been made yet.
    None,
    // The Host took the message.
    Accepted,
    // The Host process or its sockets were not ready to take it.
    NotReady,
};

// What the startup path should do next.
enum class NextAction {
    // The Host has not answered hello yet, so a connect would be premature.
    Wait,
    // Send the connect now.
    SendConnect,
    // The Host refused; wait out the backoff and try again.
    RetryLater,
    // The retry budget is spent. Surface a failure instead of sitting silent.
    GiveUp,
    // The Host accepted the connect. Nothing further to do.
    Done,
};

// Decides when the auto-connect may fire and how a refusal is handled.
//
// The startup path used to send connect on a fixed 1000ms timer. That assumes
// the Host is ready by then, but the readiness check in front of it only proves
// the process is running and both sockets are attached - not that the Host can
// take a connect. When the Host is slower than the timer, which on Windows is
// ordinary, the message is dropped and nothing retries it, so the device ID and
// access code never arrive and the UI shows two empty fields.
//
// This is deliberately free of Qt and of the messaging layer: the decision is
// "given what has been observed, what now", which is testable without a live
// Host process. The caller supplies the observations.
class HostConnectPolicy {
  public:
    HostConnectPolicy() = default;

    // Enough attempts to cover a Host that is slow to start without being an
    // unbounded retry loop.
    int max_attempts() const { return 8; }

    // Ceiling on any single wait, so a wedged Host is reported rather than
    // waited on indefinitely.
    int max_delay_ms() const { return 10000; }

    int delay_before_attempt_ms(int attempt) const;

    NextAction evaluate(bool hello_seen,
                        int attempt,
                        AttemptOutcome last_outcome = AttemptOutcome::None) const;
};

}  // namespace quickdesk

#endif  // QUICKDESK_CORE_HOSTREADINESS_H
