#!/bin/sh
# Proves that running ctest without a configuration leaves multi-config
# generators with tests that never execute.
#
# Why this needs to exist: the CI Unit tests job ran ctest from a Visual Studio
# build tree and every test came back "***Not Run - Test not available without
# configuration. (Missing "-C <config>"?)". ctest exits non-zero, so the job
# failed, but the failure says nothing about the code under test - all 16
# targets were skipped without a single assertion running. The first CI run
# that could have validated QD-113's trust regression therefore proved nothing.
#
# The container has no Visual Studio, so this uses the Ninja Multi-Config
# generator, which is multi-config for the same reason and fails the same way.
# Single-config generators (Unix Makefiles) do not need -C, and the non-Qt
# target check already covers that path; this one is about the multi-config
# shape specifically.

set -eu

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
image="${QUICKDESK_CMAKE_TEST_IMAGE:-alpine:3.20}"
work="$(mktemp -d "${TMPDIR:-/tmp}/quickdesk-ctest-config.XXXXXX")"
cleanup() { rm -rf "$work"; }
trap cleanup EXIT

mkdir -p "$work/project/src/core" "$work/project/sub"
cp "$repo_root"/QuickDesk/src/core/hostreadiness.h \
   "$repo_root"/QuickDesk/src/core/hostreadiness.cpp \
   "$work/project/src/core/"
cp "$repo_root"/QuickDesk/tests/hostreadiness_test.cpp "$work/project/sub/"

cat >"$work/project/CMakeLists.txt" <<'EOF'
cmake_minimum_required(VERSION 3.20)
project(quickdesk_ctest_config CXX)
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
option(QUICKDESK_BUILD_TESTS "Build QuickDesk unit tests" OFF)
enable_testing()
add_subdirectory(sub)
EOF

cat >"$work/project/sub/CMakeLists.txt" <<'EOF'
if(QUICKDESK_BUILD_TESTS)
    add_executable(quickdesk_host_readiness_test
        hostreadiness_test.cpp
        ../src/core/hostreadiness.cpp
    )
    target_include_directories(quickdesk_host_readiness_test PRIVATE ../src)
    add_test(NAME quickdesk_host_readiness_test COMMAND quickdesk_host_readiness_test)
endif()
EOF

# Same reasoning as the sibling script: the container supplies the multi-config
# generator without a Qt install, and neither GitHub-hosted runner has Docker.
# Skipping with an explanation beats failing with "docker: command not found",
# which is what this did on macOS for twelve rounds while the job was assumed to
# be checking something.
if ! command -v docker >/dev/null 2>&1; then
    echo "[*] docker is not available here; this check needs a container and is skipped" >&2
    exit 0
fi

echo "[*] checking ctest configuration handling in $image"
docker run --rm -v "$work/project:/project" -w /project "$image" sh -c "
    set -e
    apk add --no-cache cmake ninja g++ >/dev/null 2>&1

    # Only Release is offered, so the value -C needs is unambiguous. With the
    # default set of configurations a bare build lands in Debug, and -C Release
    # then fails for a reason that has nothing to do with the missing -C: the
    # executable is in Debug/. Restricting the list keeps this about the one
    # thing under test.
    echo '[*] multi-config generator, ctest without -C must not run the test'
    cmake -S /project -B /tmp/multi -G 'Ninja Multi-Config' \
        -DCMAKE_CONFIGURATION_TYPES=Release \
        -DQUICKDESK_BUILD_TESTS=ON >/dev/null
    cmake --build /tmp/multi >/dev/null
    cd /tmp/multi
    if ctest --output-on-failure >ctest.log 2>&1; then
        echo 'FAIL: ctest reported success while the test never ran' >&2
        cat ctest.log >&2
        exit 1
    fi
    # Character class rather than a literal quote: this line already sits inside
    # a double-quoted shell string, so a nested quote would end it.
    grep -q 'Missing .-C' ctest.log || {
        echo 'FAIL: ctest failed for some other reason, not a missing configuration:' >&2
        cat ctest.log >&2
        exit 1
    }
    grep -q 'Not Run' ctest.log || {
        echo 'FAIL: expected the test to be skipped, got:' >&2
        cat ctest.log >&2
        exit 1
    }
    if grep -q 'Passed' ctest.log; then
        echo 'FAIL: a test was reported as passed without a configuration' >&2
        exit 1
    fi
    echo '[*] confirmed: missing -C skips every test without running it'

    echo '[*] same tree, ctest with -C must actually run the test'
    ctest -C Release --output-on-failure
    echo '[*] ctest with a configuration runs the test'
"

echo "[*] ctest configuration handling verified"
