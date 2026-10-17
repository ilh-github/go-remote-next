#!/usr/bin/env bash
# Regenerates a committed client from the signaling OpenAPI spec.
#
# Usage: generate-client.sh [output-dir] [language]
#   language defaults to typescript-fetch. Supported values are go, python,
#   java, csharp, php and kotlin.
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
lang="${2:-typescript-fetch}"

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

case "$lang" in
  typescript-fetch)
    # The property list is the one documented in the spec's description, so a
    # hand-run of that example and this script produce the same client.
    java -jar "$jar" generate \
      -i "$spec" \
      -g typescript-fetch \
      -o "$out" \
      --additional-properties=supportsES6=true,npmName=@quickdesk/signaling-client,npmVersion=1.0.0
    ;;
  go)
    java -jar "$jar" generate \
      -i "$spec" \
      -g go \
      -o "$out" \
      --git-user-id=ilh-github \
      --git-repo-id=go-remote-next \
      --additional-properties=packageName=signaling,isGoSubmodule=true,withGoMod=true

    # Two normalisations, both deliberate and both applied here rather than by
    # hand so `check-generated-sdk.sh` reproduces them byte for byte.
    #
    # 1. The module path. The generator builds it from the git ids plus the
    #    package name, which can only ever produce
    #    `github.com/ilh-github/go-remote-next/signaling` - a path that does not
    #    exist in the repository, since the client lives in `sdk/go`. A nested Go
    #    module has to name its own directory, so it is pinned to the canonical
    #    location; that also keeps the path identical no matter which directory
    #    the generator wrote into (the freshness check uses a temporary one).
    perl -pi -e 's{^module\s+\S+$}{module github.com/ilh-github/go-remote-next/sdk/go}' "$out/go.mod"

    # 2. The generator's own release helper and its test tree. `git_push.sh` is
    #    for pushing the generated project as its own repository, which is not
    #    what this client is. The `test/` package is a set of HTTP contract tests
    #    that need testify (absent from the generated go.mod), so keeping it would
    #    make `go vet ./...` fail on a client that otherwise builds; the server's
    #    own smoke is what covers those paths.
    if [ -f "$out/git_push.sh" ]; then
      rm "$out/git_push.sh"
    fi
    if [ -d "$out/test" ]; then
      rm -r "$out/test"
    fi
    ;;
  python)
    java -jar "$jar" generate \
      -i "$spec" \
      -g python \
      -o "$out" \
      --git-user-id=ilh-github \
      --git-repo-id=go-remote-next \
      --additional-properties=packageName=quickdesk_signaling,projectName=quickdesk-signaling-client,packageVersion=1.0.0

    # The template writes `license = "NoLicense"`, which is not an SPDX
    # identifier, and a modern setuptools refuses to build a project whose
    # `project.license` it cannot parse - so `pip install .` fails before the
    # client is even imported. The repository is MIT (see LICENSE), so pin the
    # real value here rather than by hand, which keeps the freshness check
    # reproducible.
    perl -pi -e 's{^license\s*=\s*"NoLicense"$}{license = "MIT"}' "$out/pyproject.toml"

    # Same normalisation as the Go client, for the same reason. The generated
    # `test/` tree is a contract suite that needs a runner this repository does
    # not wire up, and every file that exists only to run it (`tox.ini`, the
    # test requirements, the vendor CI configs, `git_push.sh`) would then be a
    # broken pointer. The server's own smoke is what covers those paths; the
    # hand-written `examples/` script is what proves the client can talk to a
    # live server.
    rm -rf "$out/test" \
      "$out/test-requirements.txt" \
      "$out/tox.ini" \
      "$out/git_push.sh" \
      "$out/.travis.yml" \
      "$out/.gitlab-ci.yml" \
      "$out/.github"
    ;;
  java)
    java -jar "$jar" generate \
      -i "$spec" \
      -g java \
      -o "$out" \
      --git-user-id=ilh-github \
      --git-repo-id=go-remote-next \
      --additional-properties=groupId=io.quickdesk,artifactId=quickdesk-signaling-client,artifactVersion=1.0.0,licenseName=MIT,licenseUrl=https://opensource.org/licenses/MIT,library=okhttp-gson,dateLibrary=java8,hideGenerationTimestamp=true

    # The Java template writes a Maven pom *and* a Gradle build *and* an SBT
    # build, plus the CI configs and release helper for shipping the client as
    # its own project. CI can only verify one of them, and an unverified build
    # file is a promise this repository does not keep, so Maven stays and the
    # rest goes. The generated `test/` tree goes for the same reason it does in
    # the Go and Python clients: nothing here runs it. (Java keeps its tests
    # under `src/test`, which is why that path is removed rather than `test/`.)
    rm -rf "$out/test" \
      "$out/src/test" \
      "$out/build.gradle" \
      "$out/build.sbt" \
      "$out/settings.gradle" \
      "$out/gradlew" \
      "$out/gradlew.bat" \
      "$out/gradle" \
      "$out/gradle.properties" \
      "$out/git_push.sh" \
      "$out/.travis.yml" \
      "$out/.gitlab-ci.yml" \
      "$out/.github"

    # The Java template ignores --git-user-id/--git-repo-id and hard-codes the
    # upstream project into the pom's url and scm block, which would point users
    # at OpenAPI Generator's repository instead of this one.
    perl -pi -e 's{openapitools/openapi-generator}{ilh-github/go-remote-next}g' "$out/pom.xml"
    ;;
  csharp)
    java -jar "$jar" generate \
      -i "$spec" \
      -g csharp \
      -o "$out" \
      --git-user-id=ilh-github \
      --git-repo-id=go-remote-next \
      --additional-properties=packageName=QuickDesk.Signaling.Client,packageVersion=1.0.0,library=httpclient,netCoreProjectFile=true,targetFramework=net8.0,hideGenerationTimestamp=true,projectGuid=CD9206BF-2CB2-44B6-BAA7-B1295BC6E646

    # The generated test project contains only TODO placeholders and, by design,
    # OpenAPI Generator never overwrites test files. Drop it so a spec change
    # cannot leave the committed test tree showing stale contract comments.
    rm -rf "$out/src/QuickDesk.Signaling.Client.Test" \
      "$out/git_push.sh" \
      "$out/appveyor.yml"

    # Remove the deleted test project from the solution and pin the remaining
    # project GUID so the freshness check compares content rather than random
    # metadata.
    perl -0pi -e '
      my ($main) = /QuickDesk\.Signaling\.Client\.csproj", "\{([0-9A-F-]+)\}"/;
      my ($test) = /QuickDesk\.Signaling\.Client\.Test\.csproj", "\{([0-9A-F-]+)\}"/;
      s/\nProject\("\{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC\}"\) = "QuickDesk\.Signaling\.Client\.Test".*?EndProject//s;
      s/\n\t\t\{\Q{$test}\E\}\.[^\n]+//g if defined $test;
      s/\Q{$main}\E/{CD9206BF-2CB2-44B6-BAA7-B1295BC6E646}/g if defined $main;
    ' "$out/QuickDesk.Signaling.Client.sln"
    perl -ni -e '
      print unless /\{19F1DEBC-DE5E-4517-8062-F000CD499087\}\./;
    ' "$out/QuickDesk.Signaling.Client.sln"
    ;;
  php)
    java -jar "$jar" generate \
      -i "$spec" \
      -g php \
      -o "$out" \
      --git-user-id=ilh-github \
      --git-repo-id=go-remote-next \
      --additional-properties=hideGenerationTimestamp=true

    # The PHP template names the Composer package after the git repository and
    # emits a test tree whose cases are all incomplete. Keep the client
    # installable and lintable, but remove the test and release scaffolding.
    rm -rf "$out/test" \
      "$out/phpunit.xml.dist" \
      "$out/.php-cs-fixer.dist.php" \
      "$out/.travis.yml" \
      "$out/git_push.sh"
    perl -0pi -e '
      s/"name":\s*"[^"]+"/"name": "quickdesk\/signaling-client"/;
      s/"description":\s*"[^"]*"/"description": "QuickDesk Signaling Server generated PHP client"/s;
      s/"license":\s*"[^"]+"/"license": "MIT"/;
      s/,\s*"require-dev":\s*\{.*?\}\s*,\s*"autoload"/,"autoload"/s;
      s/,\s*"autoload-dev":\s*\{.*?\}\s*(?=\n})//s;
    ' "$out/composer.json"
    perl -0pi -e 's/^# composer\.lock$/composer.lock/m' "$out/.gitignore"
    ;;
  kotlin)
    java -jar "$jar" generate \
      -i "$spec" \
      -g kotlin \
      -o "$out" \
      --git-user-id=ilh-github \
      --git-repo-id=go-remote-next \
      --additional-properties=packageName=io.quickdesk.signaling,groupId=io.quickdesk,artifactId=quickdesk-signaling-client,artifactVersion=1.0.0,library=jvm-okhttp4,dateLibrary=java8,hideGenerationTimestamp=true

    # The generated tests use a discontinued kotlintest runner and are not part
    # of this client's contract gate. Gradle still compiles the production
    # source and publishes the JAR.
    rm -rf "$out/src/test" "$out/git_push.sh"
    printf '%s\n' '.gradle/' '.kotlin/' 'build/' > "$out/.gitignore"
    ;;
  *)
    echo "unknown language '$lang': expected typescript-fetch, go, python, java, csharp, php or kotlin" >&2
    exit 2
    ;;
esac
