// Copyright 2026 QuickDesk Authors

// The mapping from "what happened" to "which detail" is the part worth pinning:
// it is a single branch, and getting it backwards sends the reader after the
// Host process when the signaling path is what broke, or the other way round.
//
// The words are deliberately not tested here. They live in QML so the
// translation files can reach them, and a test asserting on English strings
// would have to be rewritten for every locale while proving nothing about
// behaviour.

#include "core/serverstatusdetail.h"

#include <QCoreApplication>

#include <iostream>

namespace {

int failures = 0;

void Expect(bool condition, const QString& message)
{
    if (!condition) {
        std::cerr << "FAIL: " << message.toStdString() << '\n';
        ++failures;
    }
}

}  // namespace

int main(int argc, char** argv)
{
    QCoreApplication app(argc, argv);
    using quickdesk::ServerStatusDetail;
    using quickdesk::ServerStatusDetailForExhaustedAttempts;

    // No hello at all means the Host process is the suspect. The Host may not
    // have started, may have crashed, or may be wedged - none of which is a
    // signaling problem, and pointing the user at the server would waste their
    // time.
    Expect(ServerStatusDetailForExhaustedAttempts(/*hello_seen=*/false)
               == ServerStatusDetail::HostNeverAnsweredHello,
           QStringLiteral("no hello means the Host process is the suspect"));

    // A hello proves the process is alive and talking. Refusing every connect
    // after that is a signaling-path problem, so the detail must point there.
    Expect(ServerStatusDetailForExhaustedAttempts(/*hello_seen=*/true)
               == ServerStatusDetail::HostNeverAcceptedConnect,
           QStringLiteral("a hello but no acceptance means the signaling path "
                          "is the suspect"));

    // The two outcomes must be distinguishable, or the mapping is pointless.
    Expect(ServerStatusDetail::HostNeverAnsweredHello
               != ServerStatusDetail::HostNeverAcceptedConnect,
           QStringLiteral("the two details must be different values"));

    if (failures != 0) {
        std::cerr << failures << " server status detail checks failed\n";
        return 1;
    }
    std::cout << "All server status detail checks passed\n";
    return 0;
}
