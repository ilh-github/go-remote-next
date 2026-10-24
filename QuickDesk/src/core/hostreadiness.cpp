// Copyright 2026 QuickDesk Authors

#include "core/hostreadiness.h"

namespace quickdesk {
namespace {

// First retry is quick, because the common case is a Host that is a fraction of
// a second behind; later ones back off so a genuinely absent Host is not
// hammered.
constexpr int kFirstDelayMs = 250;
constexpr int kDelayGrowthFactor = 2;

}  // namespace

int HostConnectPolicy::delay_before_attempt_ms(int attempt) const {
    if (attempt <= 0) {
        return 0;
    }
    int delay = kFirstDelayMs;
    for (int i = 1; i < attempt; ++i) {
        delay *= kDelayGrowthFactor;
        if (delay >= max_delay_ms()) {
            return max_delay_ms();
        }
    }
    return delay < max_delay_ms() ? delay : max_delay_ms();
}

NextAction HostConnectPolicy::evaluate(bool hello_seen,
                                       int attempt,
                                       AttemptOutcome last_outcome) const {
    if (last_outcome == AttemptOutcome::Accepted) {
        return NextAction::Done;
    }
    if (attempt >= max_attempts()) {
        return NextAction::GiveUp;
    }
    // hello is what proves the Host can take a connect. Without it, waiting is
    // the only correct answer; sending anyway is what dropped the message.
    if (!hello_seen) {
        return NextAction::Wait;
    }
    if (last_outcome == AttemptOutcome::NotReady) {
        return NextAction::RetryLater;
    }
    return NextAction::SendConnect;
}

}  // namespace quickdesk
