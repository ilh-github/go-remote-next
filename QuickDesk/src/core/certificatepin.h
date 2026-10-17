// Copyright 2026 QuickDesk Authors

#ifndef QUICKDESK_CORE_CERTIFICATEPIN_H
#define QUICKDESK_CORE_CERTIFICATEPIN_H

#include <string>

namespace quickdesk {

struct CertificatePinResult {
    bool valid = true;
    std::string normalized;
};

// Parses a comma-separated list of base64 SHA-256 SubjectPublicKeyInfo hashes.
// Empty input is valid and disables pinning. Partial/invalid input is rejected
// as a whole so a typo cannot silently weaken certificate validation.
CertificatePinResult ParseCertificatePinList(const std::string& input);

// Returns the Chromium command-line switch for a valid, non-empty pin list.
// Returns an empty string for empty or invalid input.
std::string BuildCertificatePinArgument(const std::string& input);

}  // namespace quickdesk

#endif  // QUICKDESK_CORE_CERTIFICATEPIN_H
