// Copyright 2026 QuickDesk Authors

#ifndef QUICKDESK_CORE_HOSTREADINESS_H
#define QUICKDESK_CORE_HOSTREADINESS_H

#include <optional>

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
    // unbounded retry loop. constexpr so the sequence's copy of the bound can
    // be checked against it at compile time.
    static constexpr int max_attempts() { return 8; }

    // Ceiling on any single wait, so a wedged Host is reported rather than
    // waited on indefinitely.
    int max_delay_ms() const { return 10000; }

    int delay_before_attempt_ms(int attempt) const;

    NextAction evaluate(bool hello_seen,
                        int attempt,
                        AttemptOutcome last_outcome = AttemptOutcome::None) const;

    // Which attempt should be scheduled next, or nothing when the sequence is
    // finished. `nullopt` means "there is no work", not "wait": a caller that
    // treats it as a wait will spin.
    //
    // This exists because the bookkeeping - dedupe a repeated hello, advance by
    // exactly one on a refusal, stop after acceptance, clear everything on a
    // Host restart - lives in MainController, which needs Qt to compile and so
    // cannot be exercised directly here. Stating the rules against a type makes
    // them testable, and MainController becomes a thin translation layer.
    class ConnectSequence {
      public:
        enum class HelloResult {
            // First hello seen; the sequence may begin.
            Started,
            // Already seen; nothing changed.
            AlreadySeen,
        };

        HelloResult note_hello();

        // Records a refusal and returns the attempt to schedule next.
        std::optional<int> note_refusal();

        // Records that the Host took the connect.
        void note_accepted();

        // Clears the handshake flag and the attempt count, for a Host restart.
        void reset();

        bool hello_seen() const { return hello_seen_; }
        int attempts_used() const { return attempts_used_; }
        bool finished() const { return accepted_; }

        // The attempt to schedule next, or nothing when there is no work left.
        // `nullopt` means finished, not "wait": a caller that treats it as a
        // wait will spin.
        std::optional<int> next_attempt() const;

      private:
        bool hello_seen_ = false;
        int attempts_used_ = 0;
        bool accepted_ = false;
    };
};

}  // namespace quickdesk

#endif  // QUICKDESK_CORE_HOSTREADINESS_H
