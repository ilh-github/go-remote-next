// Copyright 2026 QuickDesk Authors

#include "core/hostreadiness.h"

namespace quickdesk {
namespace {

// First retry is quick, because the common case is a Host that is a fraction of
// a second behind; later ones back off so a genuinely absent Host is not
// hammered.
constexpr int kFirstDelayMs = 250;
constexpr int kDelayGrowthFactor = 2;

// The sequence is a nested type and cannot read the policy's bound, so the
// number lives here too. The static_assert below is what keeps the two from
// drifting apart, which is otherwise invisible until the budget is wrong.
constexpr int kMaxAttempts = 8;
static_assert(HostConnectPolicy().max_attempts() == kMaxAttempts,
              "the sequence bound and the policy bound must agree");

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

HostConnectPolicy::ConnectSequence::HelloResult
HostConnectPolicy::ConnectSequence::note_hello() {
    if (hello_seen_) {
        return HelloResult::AlreadySeen;
    }
    hello_seen_ = true;
    return HelloResult::Started;
}

std::optional<int> HostConnectPolicy::ConnectSequence::note_refusal() {
    // Once the Host has taken the connect there is nothing left to retry, and a
    // late stray refusal must not resurrect a finished sequence.
    if (accepted_) {
        return std::nullopt;
    }
    attempts_used_ += 1;
    if (attempts_used_ >= kMaxAttempts) {
        return std::nullopt;
    }
    return attempts_used_;
}

void HostConnectPolicy::ConnectSequence::note_accepted() {
    accepted_ = true;
}

void HostConnectPolicy::ConnectSequence::reset() {
    hello_seen_ = false;
    attempts_used_ = 0;
    accepted_ = false;
}

std::optional<int> HostConnectPolicy::ConnectSequence::next_attempt() const {
    // Only "is there work left" belongs here. Whether the Host is ready for it
    // is the policy's question, not this one's - mixing them made the initial
    // state read as "finished" before hello had even arrived.
    if (accepted_ || attempts_used_ >= kMaxAttempts) {
        return std::nullopt;
    }
    return attempts_used_;
}

}  // namespace quickdesk
