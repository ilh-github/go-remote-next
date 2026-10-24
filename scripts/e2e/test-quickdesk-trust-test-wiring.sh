#!/bin/sh
# Proves the trust-handler CTest is reachable from the build that turns tests
# on, so "the tests ran" cannot be claimed while it silently did not.
#
# Why this needs to exist: test/trust/ is a standalone CMake project that came
# in with the upstream thread-affinity fix. Nothing added it to any build tree -
# QUICKDESK_BUILD_TESTS builds QuickDesk/tests, which knows nothing about this
# directory - so its CTest could not have run on any path. The CI job that
# finally built the unit tests reported 16/16 passing, and that trust regression
# was not among them. A green count that omits a suite is worse than a red one,
# because it reads as coverage.
#
# This checks the wiring statically rather than by configuring: configuring
# needs Qt and spdlog, and the point here is only whether the directory is
# reachable at all, which is a property of the build description.

set -eu

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
trust_dir="$repo_root/test/trust"

[ -f "$trust_dir/CMakeLists.txt" ] || {
    echo "FAIL: $trust_dir/CMakeLists.txt is gone; this check has nothing to guard" >&2
    exit 1
}

# The root project is the only place that can reach it. QuickDesk/tests is a
# sibling that would have to know about ../.. paths to include it, and does not.
if ! grep -qE 'add_subdirectory\(\s*test/trust' "$repo_root/CMakeLists.txt"; then
    echo "FAIL: root CMakeLists.txt does not add_subdirectory(test/trust)," >&2
    echo "      so the trust-handler CTest is in no build tree" >&2
    exit 1
fi

# The release build must not gain a test target, so the wiring has to sit inside
# a QUICKDESK_BUILD_TESTS guard. Only the enclosing block is examined, not
# everything above it: the explanatory comment names that option too, and
# searching the whole prefix made an unconditional wiring look conditional -
# which is exactly the false green this check exists to prevent.
wiring_line="$(grep -nE 'add_subdirectory\(\s*test/trust' "$repo_root/CMakeLists.txt" | cut -d: -f1)"
# The nearest preceding if( is the enclosing one. Taking the last if in the file
# instead, as an earlier version did, walks past it whenever the file has any
# later conditional.
guard_start="$(awk -v target="$wiring_line" '
    NR > target { exit }
    /^[[:space:]]*if\(/ { line = NR }
    END { print line }
' "$repo_root/CMakeLists.txt")"
if [ -z "$guard_start" ] || [ "$guard_start" -gt "$wiring_line" ]; then
    echo "FAIL: no enclosing if() around the test/trust wiring" >&2
    exit 1
fi
# Comment lines are dropped first. An earlier version matched the option name
# inside a comment that described the guard, so a deliberately unconditional
# wiring still looked conditional.
if ! sed -n "${guard_start},${wiring_line}p" "$repo_root/CMakeLists.txt" \
        | sed '/^[[:space:]]*#/d' | grep -q 'QUICKDESK_BUILD_TESTS'; then
    echo "FAIL: the enclosing if() does not test QUICKDESK_BUILD_TESTS;" >&2
    echo "      the release build would gain a test target" >&2
    exit 1
fi

# The suite must register a CTest with a name, otherwise being in the build tree
# still proves nothing about running.
if ! grep -qE '^\s*add_test\(' "$trust_dir/CMakeLists.txt"; then
    echo "FAIL: test/trust/CMakeLists.txt registers no CTest" >&2
    exit 1
fi

echo "[*] the trust-handler CTest is reachable from QUICKDESK_BUILD_TESTS"
