// Copyright 2026 QuickDesk Authors

#include "core/updatesupport.h"

#include <cstdlib>
#include <iostream>
#include <string>

namespace {

void Expect(bool condition, const std::string& message) {
    if (!condition) {
        std::cerr << "FAIL: " << message << '\n';
        std::exit(1);
    }
}

}  // namespace

int main() {
    using quickdesk::CompareUpdateVersions;
    using quickdesk::InstallerExtensionForPlatform;
    using quickdesk::IsUpdateNewer;
    using quickdesk::ParseUpdateVersion;
    using quickdesk::SanitizeInstallerFileName;
    using quickdesk::UrlsHaveSameOrigin;

    const auto parsed = ParseUpdateVersion("v4.2.10.0");
    Expect(parsed.has_value(), "leading v and numeric version parse");
    Expect(parsed->size() == 4 && (*parsed)[2] == 10,
           "version parts retain numeric ordering");

    Expect(!ParseUpdateVersion("4.2-beta").has_value(),
           "non-numeric version is rejected");
    Expect(!ParseUpdateVersion("4..1").has_value(),
           "empty version part is rejected");

    Expect(CompareUpdateVersions("4.2", "4.2.0.0") == 0,
           "missing trailing parts count as zero");
    Expect(CompareUpdateVersions("4.10.0", "4.9.9") > 0,
           "versions compare numerically rather than lexically");
    Expect(IsUpdateNewer("4.2.1.0", "4.2.0.0"),
           "newer release is detected");
    Expect(!IsUpdateNewer("4.2.0", "4.2.0.0"),
           "equal release is not an update");
    Expect(!IsUpdateNewer("bad", "4.2.0.0"),
           "invalid latest version is not treated as newer");

    Expect(SanitizeInstallerFileName("QuickDesk-4.3.0.exe",
                                     "fallback.exe") ==
               "QuickDesk-4.3.0.exe",
           "plain installer name is preserved");
    Expect(SanitizeInstallerFileName("../../QuickDesk-4.3.0.exe",
                                     "fallback.exe") ==
               "QuickDesk-4.3.0.exe",
           "directory traversal is reduced to a basename");
    Expect(SanitizeInstallerFileName("C:\\Temp\\QuickDesk-4.3.0.exe",
                                     "fallback.exe") ==
               "QuickDesk-4.3.0.exe",
           "Windows path components are removed");
    Expect(SanitizeInstallerFileName("..", "fallback.exe") ==
               "fallback.exe",
           "dot-only names fall back");

    Expect(InstallerExtensionForPlatform("windows") == ".exe",
           "Windows installers are executables");
    Expect(InstallerExtensionForPlatform("macos") == ".dmg",
           "macOS installers use DMG");
    Expect(InstallerExtensionForPlatform("linux").empty(),
           "unsupported platform has no installer extension");
    Expect(UrlsHaveSameOrigin("https://updates.example.com/",
                              "https://updates.example.com/v1/file"),
           "same HTTPS origin is accepted");
    Expect(!UrlsHaveSameOrigin("https://updates.example.com/",
                               "https://evil.example.com/file"),
           "different host is rejected");
    Expect(!UrlsHaveSameOrigin("https://updates.example.com/",
                               "http://updates.example.com/file"),
           "scheme downgrade is rejected");
    Expect(UrlsHaveSameOrigin("https://updates.example.com:443/",
                              "https://updates.example.com/file"),
           "default HTTPS port is normalized");
    Expect(!UrlsHaveSameOrigin("https://updates.example.com/",
                               "https://updates.example.com:8443/file"),
           "non-default port is rejected");

    std::cout << "update support tests passed\n";
    return 0;
}
