#!/usr/bin/env bash
# Regenerates the committed TypeScript client from the signaling OpenAPI spec.
#
# The generator is pinned by version and run from its jar rather than through
# `npx @openapitools/openapi-generator-cli`: that wrapper resolves the latest
# 7.x generator no matter what OPENAPI_GENERATOR_VERSION says, so a freshness
# check built on it would fail for reasons that have nothing to do with the
# spec. Override OPENAPI_GENERATOR_JAR to use a jar that is already on the host.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
spec="${OPENAPI_SPEC:-$root/SignalingServer/api/openapi.yaml}"
out="${1:-$root/sdk/ts}"

generator_version="7.14.0"
jar="${OPENAPI_GENERATOR_JAR:-${HOME}/.cache/quickdesk/openapi-generator-cli-${generator_version}.jar}"
url="https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/${generator_version}/openapi-generator-cli-${generator_version}.jar"

if [ ! -f "$jar" ]; then
  echo "downloading openapi-generator ${generator_version}..."
  mkdir -p "$(dirname "$jar")"
  curl -fsSL -o "${jar}.part" "$url"
  mv "${jar}.part" "$jar"
fi

if ! command -v java >/dev/null 2>&1; then
  echo "java is required to run openapi-generator (install a JDK 11+)" >&2
  exit 1
fi

# The property list is the one documented in the spec's description, so a
# hand-run of that example and this script produce the same client.
java -jar "$jar" generate \
  -i "$spec" \
  -g typescript-fetch \
  -o "$out" \
  --additional-properties=supportsES6=true,npmName=@quickdesk/signaling-client,npmVersion=1.0.0
