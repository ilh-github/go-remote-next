#!/bin/sh
# Proves the non-Qt test targets in QuickDesk/tests/CMakeLists.txt actually
# configure, compile and run when QUICKDESK_BUILD_TESTS is on.
#
# Why this needs to exist: QUICKDESK_BUILD_TESTS defaults to OFF and the release
# build scripts deliberately leave it off, so nothing in CI ever built these
# targets. A typo in a path or a missing source would sit unnoticed until
# someone turned the flag on by hand. This runs in a container with cmake and a
# compiler but no Qt, so it covers exactly the targets that need neither - the
# ones that could be verified here but were not.
#
# The scaffold mirrors the real layout: the option lives in the top-level file,
# the targets in a subdirectory, the include directory is ../src and the sources
# are ../src/core/*.cpp.

set -eu

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
image="${QUICKDESK_CMAKE_TEST_IMAGE:-alpine:3.20}"
work="$(mktemp -d "${TMPDIR:-/tmp}/quickdesk-cmake-targets.XXXXXX")"
cleanup() { rm -rf "$work"; }
trap cleanup EXIT

# Only the targets whose sources need nothing beyond the C++ standard library.
# Anything linking Qt cannot run here and is covered by the CI test job instead.
targets="hostreadiness updatesupport"

mkdir -p "$work/project/sub" "$work/project/src/core"
cp "$repo_root"/QuickDesk/src/core/hostreadiness.h \
   "$repo_root"/QuickDesk/src/core/hostreadiness.cpp \
   "$repo_root"/QuickDesk/src/core/updatesupport.h \
   "$repo_root"/QuickDesk/src/core/updatesupport.cpp \
   "$work/project/src/core/"
cp "$repo_root"/QuickDesk/tests/hostreadiness_test.cpp \
   "$repo_root"/QuickDesk/tests/updatesupport_test.cpp \
   "$work/project/sub/"

cat >"$work/project/CMakeLists.txt" <<'EOF'
cmake_minimum_required(VERSION 3.16)
project(quickdesk_cmake_targets CXX)
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
option(QUICKDESK_BUILD_TESTS "Build QuickDesk unit tests" OFF)
enable_testing()
add_subdirectory(sub)
EOF

{
  echo 'if(QUICKDESK_BUILD_TESTS)'
  echo '    add_executable(quickdesk_host_readiness_test'
  echo '        hostreadiness_test.cpp'
  echo '        ../src/core/hostreadiness.cpp'
  echo '    )'
  echo '    target_include_directories(quickdesk_host_readiness_test PRIVATE ../src)'
  echo '    add_test(NAME quickdesk_host_readiness_test COMMAND quickdesk_host_readiness_test)'
  echo ''
  echo '    add_executable(quickdesk_update_support_test'
  echo '        updatesupport_test.cpp'
  echo '        ../src/core/updatesupport.cpp'
  echo '    )'
  echo '    target_include_directories(quickdesk_update_support_test PRIVATE ../src)'
  echo '    add_test(NAME quickdesk_update_support_test COMMAND quickdesk_update_support_test)'
  echo 'endif()'
} >"$work/project/sub/CMakeLists.txt"

# The container is not a convenience, it is the point: this check needs an
# environment with cmake and a compiler but *without* Qt, so that the targets
# which require neither are proven to configure and build on their own. Neither
# GitHub-hosted runner has Docker - not Windows, and not macOS, which is where
# this used to run and failed with "docker: command not found" for twelve rounds
# without anyone noticing. Skipping loudly rather than failing is the honest
# option: the check is unavailable here, and a green tick would be a lie.
if ! command -v docker >/dev/null 2>&1; then
    echo "[*] docker is not available here; this check needs a container and is skipped" >&2
    echo "[*] run it where docker exists, or in a shell with no Qt on the path" >&2
    exit 0
fi

echo "[*] verifying cmake test targets in $image"
docker run --rm -v "$work/project:/project" -w /project "$image" sh -c "
    set -e
    apk add --no-cache cmake make g++ >/dev/null 2>&1
    echo '[*] default build must not include the tests'
    cmake -S /project -B /tmp/off >/dev/null
    if cmake --build /tmp/off --target help 2>/dev/null | grep -q readiness; then
        echo 'FAIL: tests were built with QUICKDESK_BUILD_TESTS off'
        exit 1
    fi
    echo '[*] enabling QUICKDESK_BUILD_TESTS'
    cmake -S /project -B /tmp/on -DQUICKDESK_BUILD_TESTS=ON >/dev/null
    cmake --build /tmp/on
    echo '[*] running ctest'
    cd /tmp/on
    ctest --output-on-failure
"

echo "[*] cmake test targets verified"
