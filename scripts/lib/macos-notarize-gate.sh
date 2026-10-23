#!/bin/bash

# Sourced by `package_qd_mac.sh` and by the notarization gate E2E test.
#
# The decision lives in its own file so the test can load the real logic without
# executing the packaging script around it - a copy inside the test would be free
# to drift from what actually ships.

# notarization_gate decides how strict the macOS notarization step should be and
# exports the flag `notarize_qd_mac.sh` reads.
#
# A Release DMG is the artifact that reaches users, and a signed-but-unnotarized
# package is exactly what Gatekeeper quarantines on their machine - so Release
# is strict by default. A Debug package is a local build, and a developer
# machine usually has no Developer ID at all, so that path stays permissive.
# An explicit setting always wins.
notarization_gate() {
    local build_mode="$1"
    if [ -z "${QUICKDESK_REQUIRE_MACOS_NOTARIZATION:-}" ]; then
        if [ "$build_mode" = "Release" ]; then
            QUICKDESK_REQUIRE_MACOS_NOTARIZATION=1
        else
            QUICKDESK_REQUIRE_MACOS_NOTARIZATION=0
        fi
    fi
    export QUICKDESK_REQUIRE_MACOS_NOTARIZATION
    echo "[*] macOS notarization gate: $QUICKDESK_REQUIRE_MACOS_NOTARIZATION (1 = strict)"
}
