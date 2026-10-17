// Copyright 2026 QuickDesk Authors

#include "core/certificatepin.h"

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
    using quickdesk::BuildCertificatePinArgument;
    using quickdesk::ParseCertificatePinList;

    Expect(ParseCertificatePinList("").valid, "empty input is valid");
    Expect(ParseCertificatePinList("  ").normalized.empty(),
           "whitespace-only input is empty");

    const std::string pin = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";
    const auto single = ParseCertificatePinList("  " + pin + "  ");
    Expect(single.valid, "single pin is valid");
    Expect(single.normalized == pin, "single pin is normalized");

    const auto unpadded =
        ParseCertificatePinList("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA");
    Expect(unpadded.valid, "unpadded pin is valid");
    Expect(unpadded.normalized == pin, "unpadded pin gets canonical padding");

    const auto url_safe = ParseCertificatePinList(
        "__________________________________________8=");
    Expect(url_safe.valid, "URL-safe base64 pin is valid");
    Expect(url_safe.normalized ==
               "//////////////////////////////////////////8=",
           "URL-safe base64 is normalized to standard base64");

    const auto duplicates =
        ParseCertificatePinList(pin + ", " + pin);
    Expect(duplicates.valid, "duplicate pin list is valid");
    Expect(duplicates.normalized == pin, "duplicate pins are removed");

    Expect(!ParseCertificatePinList("not-base64").valid,
           "invalid characters are rejected");
    Expect(!ParseCertificatePinList("AAAA").valid,
           "wrong decoded length is rejected");
    Expect(!ParseCertificatePinList(pin + ",broken").valid,
           "partial invalid list is rejected");
    Expect(BuildCertificatePinArgument("").empty(),
           "empty pin list does not add a switch");
    Expect(BuildCertificatePinArgument(pin) ==
               "--ignore-certificate-errors-spki-list=" + pin,
           "pin list builds the Chromium switch");
    Expect(BuildCertificatePinArgument("broken").empty(),
           "invalid pin list does not build a switch");

    std::cout << "certificate pin tests passed\n";
    return 0;
}
