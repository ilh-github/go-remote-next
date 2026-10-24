// Copyright 2026 QuickDesk Authors

#include "core/serverstatusdetail.h"

namespace quickdesk {

ServerStatusDetail::Detail ServerStatusDetailForExhaustedAttempts(bool hello_seen)
{
    // A Host that never answered hello is a process problem; one that answered
    // and then refused every connect is a signaling problem.
    return hello_seen ? ServerStatusDetail::HostNeverAcceptedConnect
                      : ServerStatusDetail::HostNeverAnsweredHello;
}

}  // namespace quickdesk
