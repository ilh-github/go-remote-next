// Copyright 2026 QuickDesk Authors

#ifndef QUICKDESK_COMMON_REMOTEPATH_H
#define QUICKDESK_COMMON_REMOTEPATH_H

#include <QChar>
#include <QString>

namespace quickdesk {

// Path helpers for paths that live on the *remote* machine.
//
// The control machine may be Windows, macOS or Linux while the host is any of
// them, so nothing here may assume the local separator, and nothing may assume a
// POSIX root.

// True when the path is rooted at a drive letter, e.g. "C:\" or "C:/".
inline bool IsDriveRooted(const QString& path)
{
    return path.size() >= 2 && path.at(1) == QLatin1Char(':')
        && ((path.at(0) >= QLatin1Char('A') && path.at(0) <= QLatin1Char('Z'))
            || (path.at(0) >= QLatin1Char('a') && path.at(0) <= QLatin1Char('z')));
}

// The parent directory of a remote path.
//
// A drive-relative path such as "C:docs" has no separator at all, so the index
// lookup finds nothing. Returning an empty string there is the bug this guards:
// callers read "no parent" as "no directory part" and then build a destination
// from the bare name, so a rename of "C:docs" lands on a relative path rather
// than inside the drive. Its parent is the drive root.
inline QString RemoteParentPath(const QString& path)
{
    const int slash = path.lastIndexOf(QLatin1Char('/'));
    const int backslash = path.lastIndexOf(QLatin1Char('\\'));
    const int index = qMax(slash, backslash);

    // "C:docs" - drive-relative, no separator. Parent is the drive root.
    if (index < 0) {
        if (IsDriveRooted(path)) {
            return path.left(2) + QLatin1Char('\\');
        }
        return QString();
    }
    if (index == 0) {
        return path.left(1);
    }
    // "C:\dir" -> "C:" keeps the colon so the drive root is not mistaken for a
    // directory named "C:".
    if (index == 2 && path.at(1) == QLatin1Char(':')) {
        return path.left(2);
    }
    return path.left(index);
}

// The last component of a remote path.
//
// The mirror of the drive-relative case above: "C:docs" has no directory part,
// so its name is "docs" rather than the whole string.
// The last component of a remote path.
//
// The mirror of the drive-relative case above: "C:docs" has no directory part,
// so its name is "docs" rather than the whole string. A trailing separator is
// stripped first, so "/home/user/" names "user" rather than nothing - a rename
// built from an empty name would target the directory itself.
inline QString RemoteBaseName(const QString& path)
{
    // Strip trailing separators, but never the one that makes a root a root:
    // "/home/user/" names "user", while "/" stays "/".
    QString trimmed = path;
    while (trimmed.size() > 1) {
        const QChar last = trimmed.at(trimmed.size() - 1);
        if (last != QLatin1Char('/') && last != QLatin1Char('\\')) {
            break;
        }
        trimmed = trimmed.left(trimmed.size() - 1);
    }

    const int slash = trimmed.lastIndexOf(QLatin1Char('/'));
    const int backslash = trimmed.lastIndexOf(QLatin1Char('\\'));
    const int index = qMax(slash, backslash);
    if (index < 0) {
        if (IsDriveRooted(trimmed)) {
            return trimmed.mid(2);
        }
        return trimmed;
    }
    if (index == 0) {
        return trimmed.mid(1);
    }
    return trimmed.mid(index + 1);
}

// Joins a directory and a name using the separator the directory already uses.
inline QString RemoteJoinPath(const QString& directory, const QString& name)
{
    if (directory.isEmpty()) {
        return name;
    }
    const QChar last = directory.at(directory.size() - 1);
    const QChar separator =
        (last == QLatin1Char('\\') || directory.contains(QLatin1Char('\\')))
                && !directory.contains(QLatin1Char('/'))
            ? QLatin1Char('\\')
            : QLatin1Char('/');
    if (last == QLatin1Char('/') || last == QLatin1Char('\\')) {
        return directory + name;
    }
    return directory + separator + name;
}

}  // namespace quickdesk

#endif  // QUICKDESK_COMMON_REMOTEPATH_H
