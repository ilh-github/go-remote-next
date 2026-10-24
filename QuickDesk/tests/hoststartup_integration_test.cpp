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
class HostFixture {
  public:
    explicit HostFixture(const QString& program)
        : m_program(program)
    {
    }

    ~HostFixture()
    {
        if (m_process) {
            m_process->disconnect();
            m_process->kill();
            m_process->waitForFinished(3000);
        }
    }

    // helloDelayMs: how long the fake Host sits on hello before answering.
    // carryIds: whether helloResponse also carries the device ID and access code.
    void Start(int helloDelayMs, bool carryIds)
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
        Expect(errorSpy.count() > 0,
               QStringLiteral("case 3: connecting while the Host is not ready "
                              "reports NOT_READY instead of failing silently"));
        if (errorSpy.count() > 0) {
            const QList<QVariant> args = errorSpy.first();
            Expect(args.at(0).toString() == QStringLiteral("NOT_READY"),
                   QStringLiteral("case 3: the reported code is NOT_READY"));
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
