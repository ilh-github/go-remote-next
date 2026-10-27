#!/usr/bin/env python3
"""Fails when the conformance smoke restores a rule without every field it can clear.

Usage: check-smoke-restore-coverage.py

`scripts/sdk/openapi-smoke.mjs` promises that the rules it touches are "put back
exactly as they were". A restore body that omits a field the PUT treats as
"omitted means clear" does not restore anything - it wipes that field, and the
smoke reports success while having changed the device's configuration.

That is not hypothetical: the offline-alert restore omitted `snoozed_until` while
the metric-alert restore two hundred lines below carried it, for a PUT whose
request struct documents an omitted snooze as clearing the snooze.

This compares, statically, the fields of each PUT/PATCH request struct against
the field names the corresponding restore body in the smoke actually mentions.
It is deliberately narrow - it only knows about restore bodies - so a rule with
no restore body is not its business.
"""
import os
import re
import sys

# This script lives in scripts/sdk/, one level deeper than the other checks in
# scripts/, so the repo root is three parents up.
ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
HANDLER_GLOB = os.path.join(ROOT, "SignalingServer", "internal", "handler")
SMOKE = os.path.join(ROOT, "scripts", "sdk", "openapi-smoke.mjs")

# Route -> the local variable holding the "before" snapshot in the smoke.
RESTORES = [
    ("/devices/{device_id}/offline-alert", "alertBefore.json"),
    ("/devices/{device_id}/metric-alert", "metricAlertBefore.json"),
]


def find_request_structs():
    """route path -> (struct name, [json field names])."""
    import glob

    structs = {}
    for path in glob.glob(os.path.join(HANDLER_GLOB, "*.go")):
        if path.endswith("_test.go"):
            continue
        text = open(path, encoding="utf-8", errors="ignore").read()
        for match in re.finditer(r"type (\w*[Rr]eq\w*) struct \{(.*?)\n\}", text, re.S):
            name, body = match.group(1), match.group(2)
            fields = re.findall(r'json:"([^",]+)', body)
            if fields:
                structs[name] = fields
    return structs


def main():
    smoke = open(SMOKE, encoding="utf-8").read()
    structs = find_request_structs()

    failures = []
    checked = 0
    for route, snapshot in RESTORES:
        # The restore body is the object literal that reads the snapshot.
        pattern = re.compile(
            r"const (\w+) = \{\s*([^}]*?)\n\s*\};", re.S
        )
        bodies = []
        for match in pattern.finditer(smoke):
            body = match.group(2)
            if snapshot in body:
                bodies.append(body)
        if not bodies:
            failures.append(f"{route}: no restore body reading {snapshot}")
            continue

        body = bodies[0]
        mentioned = set(re.findall(r"(\w+):", body))

        # Find the struct bound by this route's handler.
        route_suffix = route.split("{device_id}")[-1]
        candidate = None
        for name, fields in structs.items():
            if route_suffix.strip("/").replace("-", "") in name.lower().replace("_", ""):
                candidate = (name, fields)
                break
        if not candidate:
            # No request struct found statically; not this script's business.
            continue

        name, fields = candidate
        checked += 1
        missing = [f for f in fields if f not in mentioned]
        if missing:
            failures.append(
                f"{route}: restore body omits {missing} from {name}; "
                f"a PUT that clears an omitted field would wipe it"
            )

    if not checked:
        print("no restore bodies matched; is the smoke's naming still what this expects?")
        return 2

    if failures:
        print("restore bodies that would clear rather than restore:")
        for line in failures:
            print("  " + line)
        print("")
        print("A restore that drops a field the PUT treats as \"omitted means clear\"")
        print("changes the device's configuration while the smoke reports success.")
        return 1

    print(f"restore coverage ok: {checked} restore bodies carry every field their PUT can clear")
    return 0


if __name__ == "__main__":
    sys.exit(main())
