// Copyright 2026 QuickDesk Authors

// Covers the startup handshake against a real HostManager on a real wire, which
// is the part the policy-level test cannot reach.
//
// hostreadiness_test.cpp proves the *decision*: given "hello not seen yet", the
// answer is wait. It cannot prove the *wiring*, and it cannot prove timing. The
// acceptance cases for QD-216 are almost all timing:
//
//   1. a Host that takes seconds to become ready still ends up with its device
//      ID and access code loaded;
//   3. a hello that is dropped for want of readiness is retried and eventually
//      succeeds;
//   4. when helloResponse itself carries the IDs, the UI gets them at once.
//
// All three need something that answers late, so the fake Host takes the delay
// as an argument. Nothing here stubs HostManager: the point is to exercise the
// real signal routing that used to drop the message.

#include "manager/HostManager.h"
#include "manager/NativeMessaging.h"

#include <QCoreApplication>
#include <QElapsedTimer>
#include <QEventLoop>
#include <QJsonObject>
#include <QObject>
#include <QProcess>
#include <QStringList>
#include <QSignalSpy>
#include <QThread>

#include <cstdlib>
#include <functional>
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

bool WaitUntil(const std::function<bool()>& predicate, int timeoutMs)
{
    QElapsedTimer timer;
    timer.start();
    while (!predicate() && timer.elapsed() < timeoutMs) {
        QCoreApplication::processEvents(QEventLoop::AllEvents, 50);
        QThread::msleep(10);
    }
    return predicate();
}

// Drives the real HostManager against the fake Host on the real wire format.
//
// A QObject so the NativeMessaging and QProcess it owns can be parented to it:
// NativeMessaging's constructor takes a QObject parent, and passing something
// that is not one does not compile. Parenting rather than raw ownership also
// means a test that fails midway still tears the process down.
//
// No Q_OBJECT: this class adds no signals or slots of its own, and the moc step
// for a test target that has never needed one is an extra moving part for no
// gain. QObject alone is enough to be a valid parent.
class HostFixture : public QObject {
  public:
    explicit HostFixture(const QString& program, QObject* parent = nullptr)
        : QObject(parent)
        , m_program(program)
    {
    }

    ~HostFixture() override
    {
        if (m_process) {
            m_process->disconnect();
            m_process->kill();
            m_process->waitForFinished(3000);
        }
    }

    // helloDelayMs: how long the fake Host sits on hello before answering.
    // carryIds: whether helloResponse also carries the device ID and access code.
    // hostReadyDelayMs: whether to send a standalone hostReady afterwards, and
    //     how long after. Negative means never - that is the path where the IDs
    //     ride along with helloResponse instead.
    // snakeCaseIds: which field spelling the hostReady uses.
    void Start(int helloDelayMs, bool carryIds, int hostReadyDelayMs = -1,
               bool snakeCaseIds = false)
    {
        m_process = new QProcess();
        m_process->setProgram(m_program);
        // Explicit QStringList rather than a braced list: the same brace form is
        // ambiguous for QCommandLineOption's constructors, and being explicit
        // here keeps the two files from diverging on a subtlety that only shows
        // up on one compiler.
        QStringList arguments;
        arguments << QStringLiteral("--hello-delay-ms")
                  << QString::number(helloDelayMs);
        if (carryIds) {
            arguments << QStringLiteral("--carry-ids");
        }
        if (hostReadyDelayMs >= 0) {
            arguments << QStringLiteral("--host-ready-delay-ms")
                      << QString::number(hostReadyDelayMs);
        }
        if (snakeCaseIds) {
            arguments << QStringLiteral("--snake-case-ids");
        }
        m_process->setArguments(arguments);
        m_process->start();
        Expect(WaitUntil([&]() { return m_process->state() == QProcess::Running; }, 5000),
               QStringLiteral("fake Host starts"));

        m_messaging = new quickdesk::NativeMessaging(m_process, this);
        m_hostManager.setMessaging(m_messaging);
    }

    quickdesk::HostManager* hostManager() { return &m_hostManager; }
    QProcess* process() { return m_process; }

  private:
    QString m_program;
    QProcess* m_process = nullptr;
    quickdesk::NativeMessaging* m_messaging = nullptr;
    quickdesk::HostManager m_hostManager;
};

}  // namespace

int main(int argc, char** argv)
{
    QCoreApplication app(argc, argv);
    Expect(argc >= 2, QStringLiteral("fake host path is provided"));
    const QString fakeHost = QString::fromLocal8Bit(argv[1]);

    // --- Case 4: helloResponse carries the IDs, so they arrive with the hello ---
    //
    // This is the path where the controller does not have to wait for a separate
    // hostReady. It used to work by accident; asserting it keeps a refactor of
    // handleHelloResponse from silently dropping the fields again.
    {
        HostFixture fixture(fakeHost);
        fixture.Start(/*helloDelayMs=*/0, /*carryIds=*/true);

        QSignalSpy helloSpy(fixture.hostManager(),
                            &quickdesk::HostManager::helloResponseReceived);
        fixture.hostManager()->sendHello();
        Expect(WaitUntil([&]() { return helloSpy.count() > 0; }, 5000),
               QStringLiteral("case 4: hello is answered"));

        Expect(fixture.hostManager()->deviceId() == QStringLiteral("123456789"),
               QStringLiteral("case 4: device_id from helloResponse is applied "
                              "without waiting for hostReady"));
        Expect(fixture.hostManager()->accessCode() == QStringLiteral("ABC-123"),
               QStringLiteral("case 4: access_code from helloResponse is applied "
                              "without waiting for hostReady"));
    }

    // --- Case 3: a connect sent before the Host is ready is refused, and the
    // sequence advances instead of ending silently ---
    //
    // The fake Host is running, so isReady() is true and connectToServer will
    // not emit NOT_READY. What makes this case meaningful is that the controller
    // must not treat "the process is up" as "the handshake is done": the refusal
    // has to move the sequence forward. That behaviour lives in
    // HostConnectPolicy::ConnectSequence, and hostreadiness_test.cpp pins it;
    // this case pins that the signal actually reaches it, which is what was
    // broken before - errorOccurred was never connected, so a NOT_READY went
    // nowhere and the user saw two blank fields.
    {
        HostFixture fixture(fakeHost);
        fixture.Start(/*helloDelayMs=*/0, /*carryIds=*/false);

        QSignalSpy errorSpy(fixture.hostManager(),
                            &quickdesk::HostManager::errorOccurred);
        QSignalSpy helloSpy(fixture.hostManager(),
                            &quickdesk::HostManager::helloResponseReceived);

        fixture.hostManager()->sendHello();
        Expect(WaitUntil([&]() { return helloSpy.count() > 0; }, 5000),
               QStringLiteral("case 3: hello is answered"));
        Expect(fixture.hostManager()->deviceId().isEmpty(),
               QStringLiteral("case 3: without carry-ids the IDs stay empty "
                              "until a later message provides them"));

        // A not-ready transport must surface as an error the controller can act
        // on, not as silence. Killing the process makes isReady() false, which is
        // the state the old startup path could not distinguish from ready.
        fixture.process()->kill();
        Expect(WaitUntil([&]() {
                   return fixture.process()->state() == QProcess::NotRunning;
               }, 5000),
               QStringLiteral("case 3: fake Host stops"));

        fixture.hostManager()->connectToServer("wss://example.invalid", QString());
        Expect(WaitUntil([&]() { return errorSpy.count() > 0; }, 5000),
               QStringLiteral("case 3: connecting while the Host is not ready "
                              "reports an error instead of failing silently"));

        // Look for the code among the reports rather than assuming it is the
        // first one. Killing the process makes NativeMessaging emit its own
        // "Process crashed" first, and the order is not something this test
        // should be asserting on.
        bool sawNotReady = false;
        for (const auto& args : errorSpy) {
            if (args.at(0).toString() == QStringLiteral("NOT_READY")) {
                sawNotReady = true;
                break;
            }
        }
        Expect(sawNotReady,
               QStringLiteral("case 3: the reported code includes NOT_READY, "
                              "so the controller can act on it"));
    }

    // --- Case 5: the ordinary path, where hostReady carries the IDs ---
    //
    // helloResponse only proves the process is alive. In a normal startup the
    // IDs arrive on a separate hostReady, after provisioning has finished, so
    // this is the path nearly every real session takes. Case 4 above covers the
    // shortcut where the IDs ride along with hello; without this one, a
    // regression in handleHostReady would go unnoticed while the tests stayed
    // green.
    //
    // Run twice, once per field spelling, because HostManager accepts both and a
    // change that dropped one would break whichever Host uses it while the other
    // case kept passing.
    for (int spelling = 0; spelling < 2; ++spelling) {
        const bool snakeCase = (spelling == 1);
        const QString label = snakeCase ? QStringLiteral("snake_case")
                                        : QStringLiteral("camelCase");
        HostFixture fixture(fakeHost);
        fixture.Start(/*helloDelayMs=*/0, /*carryIds=*/false,
                      /*hostReadyDelayMs=*/0, /*snakeCaseIds=*/snakeCase);

        QSignalSpy readySpy(fixture.hostManager(),
                            &quickdesk::HostManager::hostReady);
        fixture.hostManager()->sendHello();
        Expect(WaitUntil([&]() { return readySpy.count() > 0; }, 5000),
               QStringLiteral("case 5 (%1): the standalone hostReady arrives")
                   .arg(label));

        const QString expectId =
            snakeCase ? QStringLiteral("000000000") : QStringLiteral("987654321");
        Expect(fixture.hostManager()->deviceId() == expectId,
               QStringLiteral("case 5 (%1): the device ID arrives on hostReady")
                   .arg(label));
        Expect(fixture.hostManager()->accessCode() == QStringLiteral("XYZ-789"),
               QStringLiteral("case 5 (%1): the access code arrives on hostReady")
                   .arg(label));
        if (readySpy.count() > 0) {
            const QList<QVariant> args = readySpy.first();
            Expect(args.at(0).toString() == QStringLiteral("987654321"),
                   QStringLiteral("case 5 (%1): hostReady carries the device ID")
                       .arg(label));
            Expect(args.at(1).toString() == QStringLiteral("XYZ-789"),
                   QStringLiteral("case 5 (%1): hostReady carries the access code")
                       .arg(label));
        }
    }

    // --- Case 1: a Host that is slow to answer hello still yields its IDs ---
    //
    // The delay is what makes this a timing test. The controller must not give
    // up before the Host has had a chance to answer, and must not assume the
    // process being up means the handshake is finished.
    {
        HostFixture fixture(fakeHost);
        fixture.Start(/*helloDelayMs=*/1500, /*carryIds=*/true);

        QSignalSpy helloSpy(fixture.hostManager(),
                            &quickdesk::HostManager::helloResponseReceived);
        fixture.hostManager()->sendHello();

        // A fixed 500ms wait would have missed this entirely, which is the whole
        // defect: the message was sent before there was anything to receive it.
        Expect(WaitUntil([&]() { return helloSpy.count() > 0; }, 5000),
               QStringLiteral("case 1: a Host that answers late is still "
                              "waited for, not abandoned"));
        Expect(fixture.hostManager()->deviceId() == QStringLiteral("123456789"),
               QStringLiteral("case 1: device ID arrives from a slow Host"));
        Expect(fixture.hostManager()->accessCode() == QStringLiteral("ABC-123"),
               QStringLiteral("case 1: access code arrives from a slow Host"));
    }

    if (failures != 0) {
        std::cerr << failures << " host startup integration checks failed\n";
        return 1;
    }
    std::cout << "All host startup integration checks passed\n";
    return 0;
}
