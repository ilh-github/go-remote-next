// Copyright 2026 QuickDesk Authors

// Pins the drive-relative case, which is the one that fails silently.
//
// "C:docs" has no separator at all, so a lastIndexOf-based parent lookup finds
// nothing. The natural empty return then flows into a join that produces the
// bare name, and a rename of "C:docs" is delivered to the host as "renamed.txt"
// - a relative path resolved against whatever the host process considers its
// working directory. The file lands somewhere nobody looked, and the operator is
// told the rename succeeded.
//
// The WebClient fixed this exact shape earlier; the Qt side had the same helper
// copied inline with the same hole. Assertions are on the parent and on the
// joined destination, never on the intermediate alone, because the destination
// is what the host actually receives.

#include "common/remotepath.h"

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
    using quickdesk::RemoteBaseName;
    using quickdesk::RemoteJoinPath;
    using quickdesk::RemoteParentPath;

    // The case the whole header exists for.
    Expect(RemoteParentPath(QStringLiteral("C:docs")) == QStringLiteral("C:\\"),
           QStringLiteral("a drive-relative path's parent is the drive root"));
    // And the destination that parent produces, which is what the host sees.
    Expect(RemoteJoinPath(RemoteParentPath(QStringLiteral("C:docs")),
                          QStringLiteral("renamed.txt"))
               == QStringLiteral("C:\\renamed.txt"),
           QStringLiteral("renaming a drive-relative path stays inside the drive"));
    Expect(RemoteBaseName(QStringLiteral("C:docs")) == QStringLiteral("docs"),
           QStringLiteral("a drive-relative path's name is the part after the colon"));

    // Ordinary POSIX paths must be unaffected.
    Expect(RemoteParentPath(QStringLiteral("/home/user/a.txt"))
               == QStringLiteral("/home/user"),
           QStringLiteral("a posix parent is the directory"));
    Expect(RemoteParentPath(QStringLiteral("/")) == QStringLiteral("/"),
           QStringLiteral("the root's parent is the root"));
    Expect(RemoteBaseName(QStringLiteral("/home/user/a.txt")) == QStringLiteral("a.txt"),
           QStringLiteral("a posix name is the last component"));

    // Ordinary Windows paths must be unaffected, including the drive root.
    Expect(RemoteParentPath(QStringLiteral("C:\\Users\\tim\\a.txt"))
               == QStringLiteral("C:\\Users\\tim"),
           QStringLiteral("a windows parent is the directory"));
    Expect(RemoteParentPath(QStringLiteral("C:\\Users")) == QStringLiteral("C:"),
           QStringLiteral("the drive root keeps its colon"));
    Expect(RemoteBaseName(QStringLiteral("C:\\Users\\a.txt")) == QStringLiteral("a.txt"),
           QStringLiteral("a windows name is the last component"));

    // Mixed separators: the host may be reached from any control machine, and a
    // path may legitimately contain both.
    Expect(RemoteParentPath(QStringLiteral("C:/Users/a.txt"))
               == QStringLiteral("C:/Users"),
           QStringLiteral("a forward-slash windows path is handled"));
    Expect(RemoteJoinPath(QStringLiteral("C:/Users"), QStringLiteral("b.txt"))
               == QStringLiteral("C:/Users/b.txt"),
           QStringLiteral("a join keeps the separator the directory already uses"));

    // A trailing separator must not produce an empty last component, or a rename
    // would target the directory itself.
    Expect(RemoteBaseName(QStringLiteral("/home/user/")) == QStringLiteral("user"),
           QStringLiteral("a trailing separator does not empty the name"));

    if (failures != 0) {
        std::cerr << failures << " remote path checks failed\n";
        return 1;
    }
    std::cout << "All remote path checks passed\n";
    return 0;
}
