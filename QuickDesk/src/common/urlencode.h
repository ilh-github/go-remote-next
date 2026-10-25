// Copyright 2026 QuickDesk Authors

#ifndef QUICKDESK_COMMON_URLENCODE_H
#define QUICKDESK_COMMON_URLENCODE_H

#include <QByteArray>
#include <QUrl>

namespace quickdesk {

// Percent-encodes one path segment.
//
// This exists because the obvious call is wrong. QUrl::toPercentEncoding's second
// argument is the set of characters to *leave alone*, and it defaults to "/".
// That default suits encoding a whole path; it does not suit encoding a single
// segment of one. An id containing a slash would come out with the slash intact,
// silently adding a path segment, so the request would reach a different endpoint
// and the server would answer 404 - which reads as "no such id" rather than "the
// id was mangled in transit".
//
// The same mistake was made in three places at once, which is the argument for
// one shared function rather than three copies of the correct spelling.
inline QString EncodeUrlPathSegment(const QString& value)
{
    // An empty exclude set, so '/' is encoded along with everything else that
    // needs it.
    return QString::fromLatin1(
        QUrl::toPercentEncoding(value, QByteArray()));
}

}  // namespace quickdesk

#endif  // QUICKDESK_COMMON_URLENCODE_H
