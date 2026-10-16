#!/usr/bin/env bash
# Fails when the committed client no longer matches what the spec generates.
#
# This is the gate that keeps `SignalingServer/api/openapi.yaml` and `sdk/ts`
# from drifting: a spec edit without a regenerated client fails here, and so does
# a hand-edit of the generated tree. CI runs it; so can anyone locally.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
committed="$root/sdk/ts"
fresh="$(mktemp -d)"
trap 'rm -rf "$fresh"' EXIT

echo "generating a fresh client to compare against sdk/ts..."
"$root/scripts/sdk/generate-client.sh" "$fresh/ts" >/dev/null

# Excluded from the comparison, and why:
#   FILES             the generator's own manifest of what *that run* wrote. It
#                     lists `.openapi-generator-ignore` only when the file did
#                     not already exist, so regenerating in place differs from a
#                     fresh generation for reasons that say nothing about the
#                     client.
#   node_modules      local npm state; the committed client carries no deps.
#   dist              local build output.
#   package-lock.json produced by `npm install` here, not by the generator.
if ! diff -r -x FILES -x node_modules -x dist -x package-lock.json "$committed" "$fresh/ts"; then
  echo
  echo "sdk/ts is stale: run scripts/sdk/generate-client.sh and commit the result." >&2
  exit 1
fi

echo "sdk/ts matches a fresh generation from SignalingServer/api/openapi.yaml"
