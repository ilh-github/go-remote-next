#!/bin/sh

set -u

DMG_PATH=""
TARGET_APP=""
WAIT_PID="0"
EXPECTED_VERSION=""
WORK_DIR=""
LOG_FILE=""
SKIP_LAUNCH=0
ALLOW_ELEVATION=1
REMOVE_DMG=0

MOUNT_DIR=""
STAGED_APP=""
ROLLBACK_APP=""
TARGET_BACKUP=""
OPENED=0
KEEP_WORK=0

usage() {
    printf '%s\n' \
        "usage: $0 --dmg PATH --app PATH --wait-pid PID --expected-version VERSION" \
        "          --work-dir PATH --log PATH [--skip-launch] [--no-elevation]" \
        "          [--remove-dmg]" >&2
}

log() {
    if [ -n "$LOG_FILE" ]; then
        mkdir -p "$(dirname "$LOG_FILE")" >/dev/null 2>&1 || true
        printf '%s [%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$$" "$*" \
            >>"$LOG_FILE" 2>/dev/null || true
    fi
}

remove_owned_path() {
    path="$1"
    case "$path" in
        "$WORK_DIR"/*|"$WORK_DIR")
            rm -rf -- "$path"
            ;;
        *)
            log "refusing to remove path outside update work dir: $path"
            return 1
            ;;
    esac
}

remove_target_backup() {
    path="$1"
    case "$path" in
        "$TARGET_APP".quickdesk-backup.*)
            rm -rf -- "$path"
            ;;
        *)
            log "refusing to remove unexpected target backup: $path"
            return 1
            ;;
    esac
}

cleanup() {
    status=$?
    trap - EXIT

    if [ -n "$MOUNT_DIR" ] && [ -d "$MOUNT_DIR" ]; then
        hdiutil detach "$MOUNT_DIR" -force >>"$LOG_FILE" 2>&1 || true
    fi
    if [ -n "$STAGED_APP" ] && [ -e "$STAGED_APP" ]; then
        remove_owned_path "$STAGED_APP" >/dev/null 2>&1 || true
    fi
    if [ -n "$ROLLBACK_APP" ] && [ -e "$ROLLBACK_APP" ]; then
        remove_owned_path "$ROLLBACK_APP" >/dev/null 2>&1 || true
    fi

    if [ "$status" -ne 0 ] && [ "$OPENED" -eq 0 ] &&
       [ "$SKIP_LAUNCH" -eq 0 ] && [ -d "$TARGET_APP" ]; then
        /usr/bin/open -n "$TARGET_APP" >>"$LOG_FILE" 2>&1 || true
    fi

    if [ "$KEEP_WORK" -eq 0 ] && [ -n "$WORK_DIR" ] && [ -d "$WORK_DIR" ]; then
        remove_owned_path "$WORK_DIR" >/dev/null 2>&1 || true
    fi
    exit "$status"
}

fail() {
    log "ERROR: $*"
    exit 1
}

plist_value() {
    app_path="$1"
    key="$2"
    /usr/libexec/PlistBuddy -c "Print :$key" \
        "$app_path/Contents/Info.plist" 2>/dev/null
}

team_identifier() {
    app_path="$1"
    identifier="$(
        /usr/bin/codesign -dv --verbose=4 "$app_path" 2>&1 |
        awk -F= '/^TeamIdentifier=/{print $2; exit}'
    )"
    if [ "$identifier" = "not set" ]; then
        identifier=""
    fi
    printf '%s\n' "$identifier"
}

validate_bundle() {
    app_path="$1"
    expected_id="$2"
    expected_executable="$3"

    [ -d "$app_path" ] || return 1
    [ -f "$app_path/Contents/Info.plist" ] || return 1

    bundle_id="$(plist_value "$app_path" CFBundleIdentifier)"
    executable="$(plist_value "$app_path" CFBundleExecutable)"
    [ "$bundle_id" = "$expected_id" ] || return 1
    [ "$executable" = "$expected_executable" ] || return 1
    [ -x "$app_path/Contents/MacOS/$executable" ] || return 1
    /usr/bin/codesign --verify --deep --strict "$app_path" \
        >>"$LOG_FILE" 2>&1
}

shell_quote() {
    printf "'"
    printf '%s' "$1" | sed "s/'/'\\\\''/g"
    printf "'"
}

write_root_replace_script() {
    script_path="$1"
    {
        printf '%s\n' '#!/bin/sh' 'set -u'
        printf 'TARGET_APP=%s\n' "$(shell_quote "$TARGET_APP")"
        printf 'STAGED_APP=%s\n' "$(shell_quote "$STAGED_APP")"
        printf 'TARGET_BACKUP=%s\n' "$(shell_quote "$TARGET_BACKUP")"
        cat <<'EOF'
if ! mv "$TARGET_APP" "$TARGET_BACKUP"; then
    exit 1
fi
if ! /usr/bin/ditto "$STAGED_APP" "$TARGET_APP"; then
    rm -rf -- "$TARGET_APP"
    mv "$TARGET_BACKUP" "$TARGET_APP"
    exit 1
fi
rm -rf -- "$TARGET_BACKUP"
EOF
    } >"$script_path" || return 1
    chmod 700 "$script_path"
}

write_root_restore_script() {
    script_path="$1"
    {
        printf '%s\n' '#!/bin/sh' 'set -u'
        printf 'TARGET_APP=%s\n' "$(shell_quote "$TARGET_APP")"
        printf 'ROLLBACK_APP=%s\n' "$(shell_quote "$ROLLBACK_APP")"
        cat <<'EOF'
rm -rf -- "$TARGET_APP"
/usr/bin/ditto "$ROLLBACK_APP" "$TARGET_APP"
EOF
    } >"$script_path" || return 1
    chmod 700 "$script_path"
}

run_as_admin() {
    script_path="$1"
    /usr/bin/osascript \
        -e 'on run argv' \
        -e 'do shell script "/bin/sh " & quoted form of item 1 of argv with administrator privileges' \
        -e 'end run' \
        "$script_path" >>"$LOG_FILE" 2>&1
}

restore_after_failed_launch() {
    log "new application failed to open; restoring previous bundle"
    if [ -w "$(dirname "$TARGET_APP")" ]; then
        rm -rf -- "$TARGET_APP"
        /usr/bin/ditto "$ROLLBACK_APP" "$TARGET_APP" >>"$LOG_FILE" 2>&1 || return 1
    else
        restore_script="$WORK_DIR/root-restore.sh"
        write_root_restore_script "$restore_script" || return 1
        run_as_admin "$restore_script" || return 1
    fi
    /usr/bin/open -n "$TARGET_APP" >>"$LOG_FILE" 2>&1 || true
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --dmg)
            [ "$#" -ge 2 ] || { usage; exit 64; }
            DMG_PATH="$2"
            shift 2
            ;;
        --app)
            [ "$#" -ge 2 ] || { usage; exit 64; }
            TARGET_APP="$2"
            shift 2
            ;;
        --wait-pid)
            [ "$#" -ge 2 ] || { usage; exit 64; }
            WAIT_PID="$2"
            shift 2
            ;;
        --expected-version)
            [ "$#" -ge 2 ] || { usage; exit 64; }
            EXPECTED_VERSION="$2"
            shift 2
            ;;
        --work-dir)
            [ "$#" -ge 2 ] || { usage; exit 64; }
            WORK_DIR="$2"
            shift 2
            ;;
        --log)
            [ "$#" -ge 2 ] || { usage; exit 64; }
            LOG_FILE="$2"
            shift 2
            ;;
        --skip-launch)
            SKIP_LAUNCH=1
            shift
            ;;
        --no-elevation)
            ALLOW_ELEVATION=0
            shift
            ;;
        --remove-dmg)
            REMOVE_DMG=1
            shift
            ;;
        *)
            usage
            exit 64
            ;;
    esac
done

[ -n "$DMG_PATH" ] || { usage; exit 64; }
[ -n "$TARGET_APP" ] || { usage; exit 64; }
[ -n "$WORK_DIR" ] || { usage; exit 64; }
[ -n "$LOG_FILE" ] || { usage; exit 64; }

mkdir -p "$(dirname "$LOG_FILE")" "$WORK_DIR" || exit 1
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

log "starting macOS update"
log "dmg=$DMG_PATH"
log "target=$TARGET_APP"
log "expected_version=$EXPECTED_VERSION"

case "$DMG_PATH" in
    /*) ;;
    *) fail "DMG path must be absolute" ;;
esac
case "$TARGET_APP" in
    /*.app) ;;
    *) fail "target app path must be an absolute .app bundle" ;;
esac
case "$WORK_DIR" in
    */QuickDesk/updates/install-*) ;;
    *) fail "work directory is outside the expected QuickDesk update area" ;;
esac
case "$TARGET_APP" in
    /Volumes/*) fail "cannot replace an application running from a mounted volume" ;;
esac

[ -f "$DMG_PATH" ] || fail "update DMG does not exist"
[ -d "$TARGET_APP" ] || fail "current application bundle does not exist"

case "$WAIT_PID" in
    ''|*[!0-9]*) fail "wait PID is not numeric" ;;
esac

if [ "$WAIT_PID" -gt 1 ]; then
    log "waiting for process $WAIT_PID to exit"
    attempts=0
    while kill -0 "$WAIT_PID" 2>/dev/null; do
        attempts=$((attempts + 1))
        if [ "$attempts" -ge 300 ]; then
            fail "timed out waiting for QuickDesk to exit"
        fi
        sleep 0.2
    done
fi

TARGET_ID="$(plist_value "$TARGET_APP" CFBundleIdentifier)"
TARGET_EXECUTABLE="$(plist_value "$TARGET_APP" CFBundleExecutable)"
[ -n "$TARGET_ID" ] || fail "current bundle identifier is missing"
[ -n "$TARGET_EXECUTABLE" ] || fail "current bundle executable is missing"
validate_bundle "$TARGET_APP" "$TARGET_ID" "$TARGET_EXECUTABLE" ||
    fail "current application signature is invalid"

MOUNT_DIR="$WORK_DIR/mount"
mkdir -p "$MOUNT_DIR" || fail "unable to create mount directory"
log "mounting update DMG"
hdiutil attach -nobrowse -readonly -mountpoint "$MOUNT_DIR" "$DMG_PATH" \
    >>"$LOG_FILE" 2>&1 || fail "unable to mount update DMG"

if [ -d "$MOUNT_DIR/QuickDesk.app" ]; then
    SOURCE_APP="$MOUNT_DIR/QuickDesk.app"
else
    SOURCE_APP=""
    for candidate in "$MOUNT_DIR"/*.app; do
        if [ -d "$candidate" ]; then
            SOURCE_APP="$candidate"
            break
        fi
    done
fi
[ -n "$SOURCE_APP" ] && [ -d "$SOURCE_APP" ] ||
    fail "update DMG does not contain an application bundle"

SOURCE_EXECUTABLE="$(plist_value "$SOURCE_APP" CFBundleExecutable)"
SOURCE_VERSION="$(plist_value "$SOURCE_APP" CFBundleVersion)"
validate_bundle "$SOURCE_APP" "$TARGET_ID" "$SOURCE_EXECUTABLE" ||
    fail "update application bundle is invalid or has a different identifier"
[ "$SOURCE_EXECUTABLE" = "$TARGET_EXECUTABLE" ] ||
    fail "update application executable name changed"
TARGET_TEAM_ID="$(team_identifier "$TARGET_APP")"
SOURCE_TEAM_ID="$(team_identifier "$SOURCE_APP")"
if [ -n "$TARGET_TEAM_ID" ] && [ "$SOURCE_TEAM_ID" != "$TARGET_TEAM_ID" ]; then
    fail "update application TeamIdentifier does not match the installed app"
fi
if [ -n "$EXPECTED_VERSION" ] && [ "$SOURCE_VERSION" != "$EXPECTED_VERSION" ]; then
    fail "update bundle version $SOURCE_VERSION does not match $EXPECTED_VERSION"
fi

STAGED_APP="$WORK_DIR/staged/$(basename "$TARGET_APP")"
ROLLBACK_APP="$WORK_DIR/rollback/$(basename "$TARGET_APP")"
mkdir -p "$(dirname "$STAGED_APP")" "$(dirname "$ROLLBACK_APP")" ||
    fail "unable to create staging directories"
log "copying verified application to staging"
/usr/bin/ditto "$SOURCE_APP" "$STAGED_APP" >>"$LOG_FILE" 2>&1 ||
    fail "unable to stage update application"
validate_bundle "$STAGED_APP" "$TARGET_ID" "$TARGET_EXECUTABLE" ||
    fail "staged application signature is invalid"

/usr/bin/ditto "$TARGET_APP" "$ROLLBACK_APP" >>"$LOG_FILE" 2>&1 ||
    fail "unable to create rollback bundle"

TARGET_PARENT="$(dirname "$TARGET_APP")"
if [ -w "$TARGET_PARENT" ]; then
    TARGET_BACKUP="$TARGET_APP.quickdesk-backup.$$"
    log "replacing application without elevation"
    mv "$TARGET_APP" "$TARGET_BACKUP" ||
        fail "unable to move current application to backup"
    if ! /usr/bin/ditto "$STAGED_APP" "$TARGET_APP" >>"$LOG_FILE" 2>&1; then
        rm -rf -- "$TARGET_APP"
        mv "$TARGET_BACKUP" "$TARGET_APP"
        fail "unable to install staged application"
    fi
    remove_target_backup "$TARGET_BACKUP" >/dev/null 2>&1 || true
else
    [ "$ALLOW_ELEVATION" -eq 1 ] ||
        fail "application directory requires administrator privileges"
    TARGET_BACKUP="$TARGET_APP.quickdesk-backup.$$"
    root_script="$WORK_DIR/root-replace.sh"
    write_root_replace_script "$root_script" ||
        fail "unable to prepare privileged replacement"
    log "requesting administrator privileges for application replacement"
    run_as_admin "$root_script" ||
        fail "privileged application replacement failed"
fi

if [ "$SKIP_LAUNCH" -eq 0 ]; then
    log "opening updated application"
    if ! /usr/bin/open -n "$TARGET_APP" >>"$LOG_FILE" 2>&1; then
        restore_after_failed_launch || fail "updated application failed to open and rollback failed"
        fail "updated application failed to open; previous version restored"
    fi
    OPENED=1
fi

if [ "$REMOVE_DMG" -eq 1 ]; then
    case "$DMG_PATH" in
        */QuickDesk/updates/*)
            rm -f -- "$DMG_PATH"
            ;;
        *)
            log "leaving DMG in place because it is outside the update directory"
            ;;
    esac
fi

log "macOS update completed"
exit 0
