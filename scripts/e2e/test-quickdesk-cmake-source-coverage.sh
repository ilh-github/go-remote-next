#!/bin/bash
# Every .cpp under QuickDesk/src must be referenced by some CMake source list.
#
# This exists because recordinglifecycle.cpp was added to the test target only,
# never to the application target. The file compiled, its own test passed, and
# the app then failed to link on symbols that existed nowhere. Nothing caught it
# for a week because the app was never built on the path that had tests on.
#
# The check is deliberately dumb: it looks for the file name in any CMakeLists
# under QuickDesk/. It cannot tell whether the reference is in the right target,
# only whether the file is orphaned - which is the failure that actually
# happened, and the one a build would otherwise surface as an unexplained link
# error minutes later.

set -eu

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
qt_dir="$repo_root/QuickDesk"

missing=0
# -print0 / read -d '' so a path with a space is not split.
while IFS= read -r -d '' source; do
    name="$(basename "$source")"
    # The application list is the one that matters. A file that appears only in
    # the test list is exactly the shape that bit us: it compiles, its own test
    # passes, and the app then fails to link on symbols that exist nowhere.
    if ! grep -qF "$name" "$qt_dir/CMakeLists.txt"; then
        if grep -qF "$name" "$qt_dir/tests/CMakeLists.txt"; then
            echo "FAIL: $source is only in the test target, not the app target" >&2
        else
            echo "FAIL: $source is not listed in any QuickDesk CMakeLists.txt" >&2
        fi
        missing=$((missing + 1))
    fi
done < <(find "$qt_dir/src" -name '*.cpp' -print0 | sort -z)

if [ "$missing" -gt 0 ]; then
    echo "FAIL: $missing source file(s) would be compiled by nothing" >&2
    exit 1
fi

echo "[*] every QuickDesk source file is referenced by a CMake target"
