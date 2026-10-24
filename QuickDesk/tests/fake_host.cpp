// Copyright 2026 QuickDesk Authors

// A stand-in for the prebuilt Host that speaks the native-messaging wire
// format, so the startup path can be exercised against real timing rather than
// against the policy type alone.
//
// The startup path used to send `connect` on a fixed timer, which assumed the
// Host was ready by then. A timer cannot express "the Host answered late", so
// the delay is a command-line argument here and the test asserts on what the
// controller does about it.
//
// Wire format matches NativeMessaging: 4-byte little-endian length, then JSON,
// in both directions. Line-based JSON would be simpler but would not be the
// format under test.

#include <QCommandLineOption>
#include <QCommandLineParser>
#include <QCoreApplication>
#include <QJsonDocument>
#include <QJsonObject>
#include <QStringList>
#include <QThread>

#include <cstdio>
#include <cstdlib>

namespace {

// Reads one native-messaging frame from stdin. Returns false on EOF.
bool ReadFrame(QJsonObject* out)
{
    char header[4];
    if (std::fread(header, 1, 4, stdin) != 4) {
        return false;
    }
    const auto length = static_cast<quint32>(static_cast<quint8>(header[0]))
                      | (static_cast<quint32>(static_cast<quint8>(header[1])) << 8)
                      | (static_cast<quint32>(static_cast<quint8>(header[2])) << 16)
                      | (static_cast<quint32>(static_cast<quint8>(header[3])) << 24);

    QByteArray buffer(static_cast<int>(length), Qt::Uninitialized);
    if (length > 0
        && std::fread(buffer.data(), 1, length, stdin) != length) {
        return false;
    }
    *out = QJsonDocument::fromJson(buffer).object();
    return true;
}

void WriteFrame(const QJsonObject& message)
{
    const QByteArray payload = QJsonDocument(message).toJson(QJsonDocument::Compact);
    const auto length = static_cast<quint32>(payload.size());
    char header[4] = {
        static_cast<char>(length & 0xFF),
        static_cast<char>((length >> 8) & 0xFF),
        static_cast<char>((length >> 16) & 0xFF),
        static_cast<char>((length >> 24) & 0xFF),
    };
    std::fwrite(header, 1, 4, stdout);
    std::fwrite(payload.constData(), 1, payload.size(), stdout);
    std::fflush(stdout);
}

}  // namespace

int main(int argc, char** argv)
{
    QCoreApplication app(argc, argv);
    QCommandLineParser parser;
    parser.setApplicationDescription("fake Host for the startup handshake test");
    QCommandLineOption helloDelayOption(
        QStringList{QStringLiteral("hello-delay-ms")},
        QStringLiteral("Delay before answering hello."),
        QStringLiteral("ms"),
        QStringLiteral("0"));
    QCommandLineOption carryIdsOption(
        QStringList{QStringLiteral("carry-ids")},
        QStringLiteral("helloResponse also carries device_id and access_code."));
    parser.addOption(helloDelayOption);
    parser.addOption(carryIdsOption);
    parser.process(app);

    const int helloDelayMs = parser.value(helloDelayOption).toInt();
    const bool carryIds = parser.isSet(carryIdsOption);

    QJsonObject message;
    while (ReadFrame(&message)) {
        const QString type = message.value("type").toString();

        // hello is answered, but only after the requested delay, which is how a
        // slow-starting Host is simulated. Nothing is answered before that, so a
        // connect sent in the meantime has nothing to be accepted by.
        if (type == "hello") {
            if (helloDelayMs > 0) {
                QThread::msleep(static_cast<unsigned long>(helloDelayMs));
            }
            QJsonObject response;
            response["type"] = "helloResponse";
            response["version"] = "fake-host";
            response["protocol_version"] = 3;
            if (carryIds) {
                response["device_id"] = "123456789";
                response["access_code"] = "ABC-123";
            }
            WriteFrame(response);
        } else if (type == "connect") {
            // The Host reports it took the connect. Whether it was ready to is
            // the controller's question, not this one's.
            QJsonObject response;
            response["type"] = "connectResponse";
            WriteFrame(response);
        }
        // Any other message type is ignored on purpose: the controller must not
        // depend on replies it never asked for.
    }
    return 0;
}
