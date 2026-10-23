#!/bin/bash

set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
notarize_script="$repo_root/scripts/notarize_qd_mac.sh"
root="$(mktemp -d "${TMPDIR:-/tmp}/quickdesk-notarize-gate.XXXXXX")"

fail() {
    echo "FAIL: $*" >&2
    exit 1
}

bash -n "$notarize_script"

dummy_dmg="$root/release.dmg"
: >"$dummy_dmg"

# Fake toolchain so the script's decision paths can be driven on a machine that
# has neither a real signing identity nor Apple credentials. The identity is
# "found" and codesign "succeeds", so the run reaches the notarization step.
fake_bin="$root/bin"
mkdir -p "$fake_bin"
cat >"$fake_bin/security" <<'EOF'
#!/bin/bash
echo '  1) ABCDEF "Developer ID Application: Test"'
EOF
cat >"$fake_bin/codesign" <<'EOF'
#!/bin/bash
exit 0
EOF
chmod +x "$fake_bin/security" "$fake_bin/codesign"
export QUICKDESK_SECURITY_BIN="$fake_bin/security"
export QUICKDESK_CODESIGN_BIN="$fake_bin/codesign"

# A missing signing identity is a hard failure in strict mode: a release that
# was never signed must never pass the gate.
if MACOS_SIGN_IDENTITY="" QUICKDESK_REQUIRE_MACOS_NOTARIZATION=1 \
    bash "$notarize_script" "$dummy_dmg" >/dev/null 2>&1; then
    fail "strict gate accepted a DMG with no signing identity"
fi

# The gap this test exists for: an identity is present, signing succeeds, but
# the notarization credentials are missing. The script used to exit 0 here, so
# `package_qd_mac.sh` reported success for a DMG that was signed but never
# notarized - exactly the artifact Gatekeeper quarantines on a user's machine.
if MACOS_SIGN_IDENTITY="Developer ID Application: Test" \
    QUICKDESK_REQUIRE_MACOS_NOTARIZATION=1 \
    MACOS_NOTARY_KEYCHAIN_PROFILE="" \
    APPLE_ID="" APPLE_TEAM_ID="" APPLE_APP_PASSWORD="" \
    bash "$notarize_script" "$dummy_dmg" >/dev/null 2>&1; then
    fail "strict gate accepted a signed but unnotarized DMG"
fi

# The same situation in the default (non-strict) mode still succeeds, because a
# developer building locally without credentials should not be blocked.
if ! MACOS_SIGN_IDENTITY="Developer ID Application: Test" \
    QUICKDESK_REQUIRE_MACOS_NOTARIZATION=0 \
    MACOS_NOTARY_KEYCHAIN_PROFILE="" \
    APPLE_ID="" APPLE_TEAM_ID="" APPLE_APP_PASSWORD="" \
    bash "$notarize_script" "$dummy_dmg" >/dev/null 2>&1; then
    fail "default gate refused a local build without notarization credentials"
fi

# The release packager must not depend on the operator remembering to export the
# strict flag: a Release DMG reaches users, so it has to have been notarized. A
# Debug package stays permissive, because a developer machine usually has no
# Developer ID at all and blocking it would stop local builds for no gain.
package_script="$repo_root/scripts/package_qd_mac.sh"
gate_lib="$repo_root/scripts/lib/macos-notarize-gate.sh"
if ! grep -F 'notarization_gate "$build_mode"' "$package_script" >/dev/null; then
    fail "package_qd_mac.sh does not route the gate through notarization_gate"
fi
if ! grep -F 'lib/macos-notarize-gate.sh' "$package_script" >/dev/null; then
    fail "package_qd_mac.sh does not source the shared gate logic"
fi

# The strict default must be overridable, so an operator with credentials stored
# elsewhere can still point the gate at them.
if ! grep -F 'QUICKDESK_REQUIRE_MACOS_NOTARIZATION:-' "$gate_lib" >/dev/null; then
    fail "the shared gate logic does not honour an explicit override"
fi

# Grepping the script proves the words are there, not that the gate actually
# flips. The real decision is exercised by sourcing the script's own function,
# so a copy cannot drift from what ships.
gate_for() {
    local mode="$1"
    bash -c '
        . "'"$gate_lib"'"
        notarization_gate "'"$mode"'" >/dev/null
        echo "$QUICKDESK_REQUIRE_MACOS_NOTARIZATION"
    '
}
if [ "$(gate_for Release)" != "1" ]; then
    fail "a Release build did not select the strict notarization gate"
fi
if [ "$(gate_for Debug)" != "0" ]; then
    fail "a Debug build did not select the permissive notarization gate"
fi
if [ "$(QUICKDESK_REQUIRE_MACOS_NOTARIZATION=0 gate_for Release)" != "0" ]; then
    fail "an explicit gate override was ignored for a Release build"
fi

echo "PASS: macOS notarization gate refuses signed-but-unnotarized releases"
rm -rf "$root"
