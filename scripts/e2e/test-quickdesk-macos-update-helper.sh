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
    source_app="$2"
    staging="$(mktemp -d "$root/dmg-stage.XXXXXX")"
    cp -R "$source_app" "$staging/QuickDesk.app"
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
run_helper "$target_app" "$dmg_path" "$work_dir" "$log_file"

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

if run_helper "$wrong_target" "$wrong_dmg" \
    "$root/QuickDesk/updates/install-mismatch" \
    "$root/QuickDesk/updates/install-mismatch.log"; then
    fail "mismatched bundle identifier was accepted"
fi

[ "$(cat "$wrong_target/Contents/Resources/variant.txt")" = "old" ] ||
    fail "mismatched update changed the installed app"
[ -e "$wrong_dmg" ] || fail "failed update unexpectedly removed its DMG"
[ -f "$root/QuickDesk/updates/install-mismatch.log" ] ||
    fail "failed update did not write a diagnostic log"

echo "macOS update helper E2E passed"
