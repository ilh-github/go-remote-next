// Copyright 2026 QuickDesk Authors

#include "core/hostreadiness.h"

#include <cstdlib>
#include <iostream>
#include <string>
#include <vector>

namespace {

void Expect(bool condition, const std::string& message) {
    if (!condition) {
        std::cerr << "FAIL: " << message << '\n';
        std::exit(1);
    }
}

}  // namespace

int main() {
    using quickdesk::AttemptOutcome;
    using quickdesk::HostConnectPolicy;
    using quickdesk::NextAction;

    // The startup path used to fire connect on a fixed 1000ms timer and treat a
    // not-ready Host as a dead end. A slow Host is ordinary on Windows, so the
    // gate has to be answerable from observed state rather than from elapsed
    // time, and a refusal has to leave the door open.
    HostConnectPolicy policy;
    Expect(policy.max_attempts() > 1,
           "a single attempt cannot survive a Host that starts slowly");

    Expect(policy.evaluate(/*hello_seen=*/false, /*attempt=*/0) == NextAction::Wait,
           "before the Host answers hello there is nothing to connect to yet");
    Expect(policy.evaluate(/*hello_seen=*/true, /*attempt=*/0) == NextAction::SendConnect,
           "once the Host has answered hello the connect is due");

    // A refusal is retryable, because isReady() only proves the process is up
    // and the sockets are attached - not that the Host can take a connect.
    Expect(policy.evaluate(/*hello_seen=*/true, /*attempt=*/1,
                          AttemptOutcome::NotReady) == NextAction::RetryLater,
           "a not-ready refusal must schedule another attempt, not give up");

    // The last attempt inside the budget still retries; only running past the
    // budget gives up, so an absent Host ends in a visible failure rather than
    // an unbounded loop.
    Expect(policy.evaluate(/*hello_seen=*/true, /*attempt=*/policy.max_attempts() - 1,
                          AttemptOutcome::NotReady) == NextAction::RetryLater,
           "the last attempt inside the budget is still a retry");
    Expect(policy.evaluate(/*hello_seen=*/true, /*attempt=*/policy.max_attempts(),
                          AttemptOutcome::NotReady) == NextAction::GiveUp,
           "running past the budget ends in a visible failure");

    // Backoff has to grow, otherwise a Host that is slow to start gets hammered
    // by a burst of connects, and it must not grow without bound.
    std::vector<int> delays;
    for (int attempt = 0; attempt < policy.max_attempts(); ++attempt) {
        delays.push_back(policy.delay_before_attempt_ms(attempt));
    }
    Expect(delays.front() >= 0, "the first attempt is not delayed by backoff");
    for (std::size_t i = 1; i < delays.size(); ++i) {
        Expect(delays[i] > delays[i - 1],
               "backoff grows with each attempt");
    }
    Expect(delays.back() <= policy.max_delay_ms(),
           "backoff is capped so a slow Host is not waited on forever");

    // A successful connect is final: no further attempts, and the caller can
    // tell "done" from "gave up" so it does not report a failure on success.
    Expect(policy.evaluate(/*hello_seen=*/true, /*attempt=*/0,
                          AttemptOutcome::Accepted) == NextAction::Done,
           "an accepted connect finishes the sequence");

    // And it stays finished: once the Host has taken the connect, a later turn
    // of the sequence must not schedule another one, even though hello is still
    // marked seen and attempts remain in the budget.
    Expect(policy.evaluate(/*hello_seen=*/true, /*attempt=*/1,
                          AttemptOutcome::Accepted) == NextAction::Done,
           "an accepted connect is not followed by another attempt");

    std::cout << "host readiness tests passed\n";
    return 0;
}
