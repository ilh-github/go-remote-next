// Copyright 2026 QuickDesk Authors

#include "core/updatesupport.h"

#include <algorithm>
#include <cctype>
#include <limits>
#include <optional>
#include <utility>

namespace quickdesk {
namespace {

constexpr std::size_t kMaxVersionParts = 8;
constexpr std::size_t kMaxInstallerNameLength = 200;

std::string TrimAsciiWhitespace(const std::string& input) {
    std::size_t begin = 0;
    while (begin < input.size() &&
           std::isspace(static_cast<unsigned char>(input[begin]))) {
        ++begin;
    }

    std::size_t end = input.size();
    while (end > begin &&
           std::isspace(static_cast<unsigned char>(input[end - 1]))) {
        --end;
    }
    return input.substr(begin, end - begin);
}

std::optional<std::uint64_t> ParseVersionPart(const std::string& part) {
    if (part.empty()) {
        return std::nullopt;
    }

    std::uint64_t value = 0;
    for (char character : part) {
        if (character < '0' || character > '9') {
            return std::nullopt;
        }
        const std::uint64_t digit =
            static_cast<std::uint64_t>(character - '0');
        if (value >
            (std::numeric_limits<std::uint64_t>::max() - digit) / 10) {
            return std::nullopt;
        }
        value = value * 10 + digit;
    }
    return value;
}

std::string SanitizeOneFileName(const std::string& proposed) {
    std::string name = TrimAsciiWhitespace(proposed);
    const std::size_t separator = name.find_last_of("/\\");
    if (separator != std::string::npos) {
        name = name.substr(separator + 1);
    }

    const std::string forbidden = "<>:\"|?*";
    for (char& character : name) {
        const unsigned char byte = static_cast<unsigned char>(character);
        if (std::iscntrl(byte) || forbidden.find(character) != std::string::npos) {
            character = '_';
        }
    }

    name = TrimAsciiWhitespace(name);
    if (name.size() > kMaxInstallerNameLength) {
        name.resize(kMaxInstallerNameLength);
    }
    return name;
}

bool IsUsableFileName(const std::string& name) {
    return !name.empty() && name != "." && name != "..";
}

std::string ToLowerAscii(std::string value) {
    std::transform(value.begin(), value.end(), value.begin(),
                   [](unsigned char character) {
                       return static_cast<char>(std::tolower(character));
                   });
    return value;
}

struct UrlOrigin {
    bool valid = false;
    std::string scheme;
    std::string host;
    std::uint64_t port = 0;
};

UrlOrigin ParseUrlOrigin(const std::string& url) {
    UrlOrigin origin;
    const std::string trimmed = TrimAsciiWhitespace(url);
    const std::size_t scheme_end = trimmed.find("://");
    if (scheme_end == std::string::npos || scheme_end == 0) {
        return origin;
    }

    origin.scheme = ToLowerAscii(trimmed.substr(0, scheme_end));
    if (origin.scheme != "http" && origin.scheme != "https") {
        return origin;
    }

    const std::size_t authority_begin = scheme_end + 3;
    const std::size_t authority_end =
        trimmed.find_first_of("/?#", authority_begin);
    const std::string authority = trimmed.substr(
        authority_begin,
        authority_end == std::string::npos
            ? std::string::npos
            : authority_end - authority_begin);
    if (authority.empty() || authority.find('@') != std::string::npos) {
        return origin;
    }

    std::string host;
    std::string port_text;
    if (authority.front() == '[') {
        const std::size_t bracket = authority.find(']');
        if (bracket == std::string::npos) {
            return origin;
        }
        host = authority.substr(0, bracket + 1);
        if (bracket + 1 < authority.size()) {
            if (authority[bracket + 1] != ':') {
                return origin;
            }
            port_text = authority.substr(bracket + 2);
        }
    } else {
        const std::size_t colon = authority.rfind(':');
        if (colon != std::string::npos) {
            host = authority.substr(0, colon);
            port_text = authority.substr(colon + 1);
        } else {
            host = authority;
        }
    }
    if (host.empty()) {
        return origin;
    }

    std::uint64_t port = 0;
    if (port_text.empty()) {
        port = origin.scheme == "https" ? 443 : 80;
    } else {
        for (char character : port_text) {
            if (character < '0' || character > '9') {
                return origin;
            }
            port = port * 10 + static_cast<std::uint64_t>(character - '0');
            if (port > 65535) {
                return origin;
            }
        }
    }

    origin.host = ToLowerAscii(host);
    origin.port = port;
    origin.valid = true;
    return origin;
}

}  // namespace

std::optional<std::vector<std::uint64_t>> ParseUpdateVersion(
    const std::string& version) {
    std::string raw = TrimAsciiWhitespace(version);
    if (!raw.empty() && (raw.front() == 'v' || raw.front() == 'V')) {
        raw.erase(raw.begin());
    }
    if (raw.empty()) {
        return std::nullopt;
    }

    std::vector<std::uint64_t> parts;
    std::size_t begin = 0;
    while (begin <= raw.size()) {
        if (parts.size() >= kMaxVersionParts) {
            return std::nullopt;
        }
        const std::size_t dot = raw.find('.', begin);
        const std::size_t end = dot == std::string::npos ? raw.size() : dot;
        const auto part = ParseVersionPart(raw.substr(begin, end - begin));
        if (!part.has_value()) {
            return std::nullopt;
        }
        parts.push_back(*part);
        if (dot == std::string::npos) {
            break;
        }
        begin = dot + 1;
    }
    return parts;
}

int CompareUpdateVersions(const std::string& left, const std::string& right) {
    const auto left_parts = ParseUpdateVersion(left);
    const auto right_parts = ParseUpdateVersion(right);
    if (!left_parts.has_value() || !right_parts.has_value()) {
        return 0;
    }

    const std::size_t width = std::max(left_parts->size(), right_parts->size());
    for (std::size_t i = 0; i < width; ++i) {
        const std::uint64_t lhs =
            i < left_parts->size() ? (*left_parts)[i] : 0;
        const std::uint64_t rhs =
            i < right_parts->size() ? (*right_parts)[i] : 0;
        if (lhs < rhs) {
            return -1;
        }
        if (lhs > rhs) {
            return 1;
        }
    }
    return 0;
}

bool IsUpdateNewer(const std::string& latest, const std::string& current) {
    if (!ParseUpdateVersion(latest).has_value() ||
        !ParseUpdateVersion(current).has_value()) {
        return false;
    }
    return CompareUpdateVersions(latest, current) > 0;
}

std::string SanitizeInstallerFileName(const std::string& proposed,
                                      const std::string& fallback) {
    std::string name = SanitizeOneFileName(proposed);
    if (IsUsableFileName(name)) {
        return name;
    }

    name = SanitizeOneFileName(fallback);
    return IsUsableFileName(name) ? name : std::string("QuickDesk-update.bin");
}

std::string InstallerExtensionForPlatform(const std::string& platform) {
    if (platform == "windows") {
        return ".exe";
    }
    if (platform == "macos") {
        return ".dmg";
    }
    return "";
}

bool UrlsHaveSameOrigin(const std::string& base_url,
                        const std::string& target_url) {
    const UrlOrigin base = ParseUrlOrigin(base_url);
    const UrlOrigin target = ParseUrlOrigin(target_url);
    return base.valid && target.valid && base.scheme == target.scheme &&
           base.host == target.host && base.port == target.port;
}

}  // namespace quickdesk
