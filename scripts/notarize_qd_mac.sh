#!/bin/bash

set -euo pipefail

dmg_path="${1:-}"
if [ -z "$dmg_path" ] || [ ! -f "$dmg_path" ]; then
    echo "[!] usage: $0 /absolute/path/to/QuickDesk.dmg" >&2
    exit 1
fi

sign_identity="${MACOS_SIGN_IDENTITY:-}"
notary_profile="${MACOS_NOTARY_KEYCHAIN_PROFILE:-}"
require_notarization="${QUICKDESK_REQUIRE_MACOS_NOTARIZATION:-0}"

if [ -z "$sign_identity" ]; then
    if [ "$require_notarization" = "1" ]; then
        echo "[!] MACOS_SIGN_IDENTITY is required for a notarized release" >&2
        exit 1
    fi
    echo "[*] Developer ID signing identity not configured; notarization skipped"
    exit 0
fi

if ! security find-identity -v -p codesigning |
    grep -F "$sign_identity" >/dev/null; then
    echo "[!] codesigning identity not found: $sign_identity" >&2
    exit 1
fi

echo "[*] signing DMG with: $sign_identity"
codesign \
    --force \
    --sign "$sign_identity" \
    --timestamp \
    "$dmg_path"

notary_args=()
if [ -n "$notary_profile" ]; then
    notary_args=(--keychain-profile "$notary_profile")
else
    apple_id="${APPLE_ID:-}"
    team_id="${APPLE_TEAM_ID:-}"
    app_password="${APPLE_APP_PASSWORD:-}"
    if [ -z "$apple_id" ] || [ -z "$team_id" ] || [ -z "$app_password" ]; then
        if [ "$require_notarization" = "1" ]; then
            echo "[!] notarization credentials are missing" >&2
            echo "[!] set MACOS_NOTARY_KEYCHAIN_PROFILE or APPLE_ID + APPLE_TEAM_ID + APPLE_APP_PASSWORD" >&2
            exit 1
        fi
        echo "[*] notarization credentials not configured; signed DMG left unnotarized"
        exit 0
    fi
    notary_args=(
        --apple-id "$apple_id"
        --team-id "$team_id"
        --password "$app_password"
    )
fi

echo "[*] submitting DMG for Apple notarization"
xcrun notarytool submit "$dmg_path" "${notary_args[@]}" --wait

echo "[*] stapling notarization ticket"
xcrun stapler staple "$dmg_path"
xcrun stapler validate "$dmg_path"

echo "[*] verifying Gatekeeper assessment"
spctl \
    --assess \
    --type open \
    --context context:primary-signature \
    --verbose=2 \
    "$dmg_path"

echo "[*] macOS DMG signing and notarization finished"
