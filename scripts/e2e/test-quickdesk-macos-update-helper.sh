#!/bin/bash

set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
helper="$repo_root/QuickDesk/scripts/macos_update_helper.sh"
notarize_script="$repo_root/scripts/notarize_qd_mac.sh"
root="$(mktemp -d "${TMPDIR:-/tmp}/quickdesk-macos-update.XXXXXX")"

cleanup() {
    for mount_dir in "$root"/*/mount; do
        if [ -d "$mount_dir" ]; then
            hdiutil detach "$mount_dir" -force >/dev/null 2>&1 || true
        fi
    done
    rm -rf "$root"
}
trap cleanup EXIT

fail() {
    echo "FAIL: $*" >&2
    exit 1
}

make_app() {
    app_path="$1"
    bundle_id="$2"
    version="$3"
    marker="$4"
    mkdir -p "$app_path/Contents/MacOS" "$app_path/Contents/Resources"
    cat >"$app_path/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleIdentifier</key>
    <string>$bundle_id</string>
    <key>CFBundleExecutable</key>
    <string>QuickDeskTest</string>
    <key>CFBundleShortVersionString</key>
    <string>$version</string>
    <key>CFBundleVersion</key>
    <string>$version</string>
</dict>
</plist>
EOF
    cat >"$app_path/Contents/MacOS/QuickDeskTest" <<'EOF'
#!/bin/sh
exit 0
EOF
    chmod +x "$app_path/Contents/MacOS/QuickDeskTest"
    printf '%s\n' "$marker" >"$app_path/Contents/Resources/variant.txt"
    /usr/bin/codesign --force --sign - --timestamp=none "$app_path" \
        >/dev/null 2>&1
}

make_dmg() {
    dmg_path="$1"
    # Named apart from the global `source_app`: without `local` (this is /bin/sh)
    # this assignment would overwrite it, and every later scenario would pack the
    # bundle from whichever app was passed most recently.
    dmg_source_app="$2"
    staging="$(mktemp -d "$root/dmg-stage.XXXXXX")"
    cp -R "$dmg_source_app" "$staging/QuickDesk.app"
    ln -s /Applications "$staging/Applications"
    mkdir -p "$(dirname "$dmg_path")"
    hdiutil create \
        -volname "QuickDeskUpdateTest" \
        -srcfolder "$staging" \
        -ov \
        -format UDZO \
        -quiet \
        "$dmg_path"
    rm -rf "$staging"
}

run_helper() {
    target_app="$1"
    dmg_path="$2"
    work_dir="$3"
    log_file="$4"
    /bin/sh "$helper" \
        --dmg "$dmg_path" \
        --app "$target_app" \
        --wait-pid 999999 \
        --expected-version 4.2.9.0 \
        --work-dir "$work_dir" \
        --log "$log_file" \
        --skip-launch \
        --no-elevation \
        --remove-dmg
}

# The suite signs its bundles ad-hoc, because a Developer ID is not something a
# test run can have. That is exactly what the trust check refuses, so the
# scenarios that are not about trust opt out explicitly - and the one that is
# about trust deliberately does not.
run_helper_allowing_unnotarized() {
    QUICKDESK_ALLOW_UNNOTARIZED_UPDATE=1 \
        run_helper "$@"
}

sh -n "$helper"
bash -n "$notarize_script"

dummy_dmg="$root/release-gate.dmg"
: >"$dummy_dmg"
if ! bash "$notarize_script" "$dummy_dmg" 2>&1 |
    grep -F "notarization skipped" >/dev/null; then
    fail "ad-hoc release gate did not report a notarization skip"
fi
if QUICKDESK_REQUIRE_MACOS_NOTARIZATION=1 \
    bash "$notarize_script" "$dummy_dmg" >/dev/null 2>&1; then
    fail "strict release gate accepted a missing Developer ID"
fi

source_app="$root/source/QuickDesk.app"
make_app "$source_app" "com.quickdesk.update.test" "4.2.9.0" "new"

target_app="$root/target/QuickDesk.app"
make_app "$target_app" "com.quickdesk.update.test" "4.2.8.0" "old"

dmg_path="$root/QuickDesk/updates/4.2.9.0/QuickDesk-test.dmg"
make_dmg "$dmg_path" "$source_app"

work_dir="$root/QuickDesk/updates/install-success"
log_file="$root/QuickDesk/updates/install.log"
run_helper_allowing_unnotarized "$target_app" "$dmg_path" "$work_dir" "$log_file"

[ "$(cat "$target_app/Contents/Resources/variant.txt")" = "new" ] ||
    fail "updated app was not installed"
[ "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleVersion' "$target_app/Contents/Info.plist")" = "4.2.9.0" ] ||
    fail "updated app version is wrong"
[ ! -e "$dmg_path" ] || fail "DMG was not removed after a successful update"
[ ! -d "$work_dir" ] || fail "helper work directory was not cleaned"

wrong_app="$root/wrong/QuickDesk.app"
make_app "$wrong_app" "com.quickdesk.wrong" "4.2.9.0" "wrong"
wrong_target="$root/wrong-target/QuickDesk.app"
make_app "$wrong_target" "com.quickdesk.update.test" "4.2.8.0" "old"
wrong_dmg="$root/QuickDesk/updates/4.2.9.0/QuickDesk-wrong.dmg"
make_dmg "$wrong_dmg" "$wrong_app"

# Same ad-hoc signature as the success case, so the bundle identifier is the
# only thing under test here; trust is opted out of, as above.
if run_helper_allowing_unnotarized "$wrong_target" "$wrong_dmg" \
    "$root/QuickDesk/updates/install-mismatch" \
    "$root/QuickDesk/updates/install-mismatch.log"; then
    fail "mismatched bundle identifier was accepted"
fi

[ "$(cat "$wrong_target/Contents/Resources/variant.txt")" = "old" ] ||
    fail "mismatched update changed the installed app"
[ -e "$wrong_dmg" ] || fail "failed update unexpectedly removed its DMG"
[ -f "$root/QuickDesk/updates/install-mismatch.log" ] ||
    fail "failed update did not write a diagnostic log"

# A bundle that carries a signature but no notarization is exactly what
# Gatekeeper quarantines on a user's Mac. `codesign --verify` accepts it and
# TeamIdentifier is empty on both sides, so the checks above let it through -
# which means the updater would install an app that refuses to open, and the
# operator would find out from the user rather than from the log.
# Its own target bundle, so "the installed app was not disturbed" is a real
# observation: the success case above has already moved the shared target to
# the new version, which would make the same assertion pass either way.
unnotarized_target="$root/unnotarized-target/QuickDesk.app"
make_app "$unnotarized_target" "com.quickdesk.update.test" "4.2.8.0" "old"
unnotarized_dmg="$root/QuickDesk/updates/4.2.9.0/QuickDesk-unnotarized.dmg"
make_dmg "$unnotarized_dmg" "$source_app"

if run_helper "$unnotarized_target" "$unnotarized_dmg" \
    "$root/QuickDesk/updates/install-unnotarized" \
    "$root/QuickDesk/updates/install-unnotarized.log"; then
    fail "an update that Gatekeeper would reject was installed anyway"
fi

[ "$(cat "$unnotarized_target/Contents/Resources/variant.txt")" = "old" ] ||
    fail "rejected update disturbed the installed app"
[ "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleVersion' \
    "$unnotarized_target/Contents/Info.plist")" = "4.2.8.0" ] ||
    fail "rejected update changed the installed version"
[ -e "$unnotarized_dmg" ] || fail "rejected update unexpectedly removed its DMG"
[ -f "$root/QuickDesk/updates/install-unnotarized.log" ] ||
    fail "rejected update did not write a diagnostic log"

echo "macOS update helper E2E passed"
