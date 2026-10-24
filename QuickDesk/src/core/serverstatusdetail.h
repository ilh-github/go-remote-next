// Copyright 2026 QuickDesk Authors

#ifndef QUICKDESK_CORE_SERVERSTATUSDETAIL_H
#define QUICKDESK_CORE_SERVERSTATUSDETAIL_H

#include "../common/ProcessStatus.h"

namespace quickdesk {

// The detail to report for a startup that ran out of attempts.
//
// "Never answered hello" and "answered but never accepted" are separated
// because they point at different things: the first means the Host process is
// not healthy at all, the second means it is alive but its signaling path is
// not working. Reporting either as the other sends the reader the wrong way.
//
// This returns an enum rather than a sentence on purpose. The words live in QML,
// where the translation files can reach them; a string produced here would be
// English in every locale.
ServerStatusDetail::Detail ServerStatusDetailForExhaustedAttempts(bool hello_seen);

}  // namespace quickdesk

#endif  // QUICKDESK_CORE_SERVERSTATUSDETAIL_H
