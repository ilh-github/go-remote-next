#!/bin/sh
# Every .cpp under QuickDesk/tests must be referenced by some CMake target there.
#
# The existing coverage check only sweeps QuickDesk/src, because that is where
# the recordinglifecycle.cpp defect lived. The tests directory has the same
# failure mode with an extra twist: a test .cpp that no target compiles is
# invisible, since nothing else in the tree would try to build it. A test that
# exists in the working tree but not in any build tree looks like coverage while
# asserting nothing, which is the same trap as the trust-handler CTest that was
# never wired in.
#
# Also checks the reverse direction, which the src check cannot: a source listed
# in tests/CMakeLists.txt that does not exist on disk. That is a stale entry
# left behind by a renamed or deleted file, and it fails the configure step with
# a message about a missing file rather than about the thing that changed.

set -eu

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
qt_dir="$repo_root/QuickDesk"
cmake_file="$qt_dir/tests/CMakeLists.txt"

[ -f "$cmake_file" ] || {
    echo "FAIL: $cmake_file is missing" >&2
    exit 1
}

missing=0
while IFS= read -r -d '' source; do
    name="$(basename "$source")"
    if ! grep -qF "$name" "$cmake_file"; then
        echo "FAIL: $source is not listed in tests/CMakeLists.txt" >&2
        missing=$((missing + 1))
    fi
done < <(find "$qt_dir/tests" -maxdepth 1 -name '*.cpp' -print0 | sort -z)

stale=0
while IFS= read -r referenced; do
    if [ ! -f "$qt_dir/tests/$referenced" ]; then
        echo "FAIL: tests/CMakeLists.txt lists $referenced, which does not exist" >&2
        stale=$((stale + 1))
    fi
done < <(sed -n 's/^[[:space:]]*\([A-Za-z0-9_.-]*\.cpp\)[[:space:]]*$/\1/p' "$cmake_file" | sort -u)

if [ "$missing" -gt 0 ] || [ "$stale" -gt 0 ]; then
    echo "FAIL: $missing test source(s) built by nothing, $stale stale entr(y/ies)" >&2
    exit 1
fi

echo "[*] every QuickDesk test source file is listed, and every listing exists"
