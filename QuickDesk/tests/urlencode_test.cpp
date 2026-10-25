// Copyright 2026 QuickDesk Authors

// Pins the one thing that is easy to get wrong and invisible when you do.
//
// QUrl::toPercentEncoding leaves '/' alone by default, which is right for a path
// and wrong for a segment. The result of getting it wrong is not a crash or a
// malformed request - it is a well-formed request to the wrong endpoint, answered
// with 404, which tells the user the id does not exist when in fact it was
// rewritten on the way out.
//
// The assertions are on the encoded bytes, never on QUrl::path(): path() returns
// the decoded form, so a %2F comes back as a literal '/' and the defect being
// tested for cannot be seen there at all. An earlier version of this check made
// exactly that mistake and passed while testing nothing.

#include "common/urlencode.h"

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
    using quickdesk::EncodeUrlPathSegment;

    // The case the whole function exists for.
    Expect(EncodeUrlPathSegment(QStringLiteral("a/b"))
               == QStringLiteral("a%2Fb"),
           QStringLiteral("a slash inside a segment must be encoded"));

    // Ordinary ids must pass through untouched, or every request changes shape.
    Expect(EncodeUrlPathSegment(QStringLiteral("123456789"))
               == QStringLiteral("123456789"),
           QStringLiteral("a plain digit id is unchanged"));
    // Asserted as "no percent-encoding happened" rather than as one exact
    // string: which characters count as unreserved is the encoder's business and
    // has differed between versions, but a plain id must not be rewritten.
    Expect(!EncodeUrlPathSegment(QStringLiteral("session-1"))
                .contains(QLatin1Char('%')),
           QStringLiteral("a plain alphanumeric id is not rewritten"));

    // Characters that would otherwise terminate or reinterpret the URL.
    Expect(EncodeUrlPathSegment(QStringLiteral("a b"))
               == QStringLiteral("a%20b"),
           QStringLiteral("a space is encoded"));
    Expect(EncodeUrlPathSegment(QStringLiteral("a?b"))
               == QStringLiteral("a%3Fb"),
           QStringLiteral("a question mark is encoded, so it cannot start a query"));
    Expect(EncodeUrlPathSegment(QStringLiteral("a#b"))
               == QStringLiteral("a%23b"),
           QStringLiteral("a hash is encoded, so it cannot start a fragment"));
    Expect(EncodeUrlPathSegment(QStringLiteral("a%2Fb"))
               == QStringLiteral("a%252Fb"),
           QStringLiteral("an existing percent sign is encoded, so a literal "
                          "%2F in an id cannot be smuggled through as a slash"));

    // A trailing slash is the other half of the same trap: it would extend the
    // path rather than being part of the id.
    Expect(EncodeUrlPathSegment(QStringLiteral("abc/"))
               == QStringLiteral("abc%2F"),
           QStringLiteral("a trailing slash is encoded, not treated as a separator"));

    // The encoded form must survive a round trip back to the original value, or
    // the id is being changed rather than represented.
    const QString tricky = QStringLiteral("a/b c?d#e");
    const QString encoded = EncodeUrlPathSegment(tricky);
    Expect(QUrl::fromPercentEncoding(encoded.toLatin1()) == tricky,
           QStringLiteral("the encoded form decodes back to the original id"));

    if (failures != 0) {
        std::cerr << failures << " url encode checks failed\n";
        return 1;
    }
    std::cout << "All url encode checks passed\n";
    return 0;
}
