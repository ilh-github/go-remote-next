// Copyright 2026 QuickDesk Authors

#include "core/certificatepin.h"

#include <array>
#include <cctype>
#include <set>
#include <vector>

namespace quickdesk {
namespace {

constexpr size_t kSha256Size = 32;
constexpr char kBase64Alphabet[] =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

int Base64Value(char value) {
    if (value >= 'A' && value <= 'Z') {
        return value - 'A';
    }
    if (value >= 'a' && value <= 'z') {
        return value - 'a' + 26;
    }
    if (value >= '0' && value <= '9') {
        return value - '0' + 52;
    }
    if (value == '+' || value == '-') {
        return 62;
    }
    if (value == '/' || value == '_') {
        return 63;
    }
    return -1;
}

std::string TrimAsciiWhitespace(const std::string& input) {
    size_t begin = 0;
    while (begin < input.size() &&
           std::isspace(static_cast<unsigned char>(input[begin]))) {
        ++begin;
    }

    size_t end = input.size();
    while (end > begin &&
           std::isspace(static_cast<unsigned char>(input[end - 1]))) {
        --end;
    }
    return input.substr(begin, end - begin);
}

bool DecodePin(const std::string& pin,
               std::array<unsigned char, kSha256Size>* out) {
    if (pin.empty()) {
        return false;
    }

    size_t padding = 0;
    while (padding < pin.size() && pin[pin.size() - padding - 1] == '=') {
        ++padding;
    }
    if (padding > 2) {
        return false;
    }
    if (padding > 0 && pin.size() % 4 != 0) {
        return false;
    }

    const size_t encoded_size = pin.size() - padding;
    if (encoded_size == 0 || encoded_size % 4 == 1) {
        return false;
    }

    size_t output_size = 0;
    unsigned int bit_buffer = 0;
    int bit_count = 0;
    for (size_t i = 0; i < encoded_size; ++i) {
        const int value = Base64Value(pin[i]);
        if (value < 0) {
            return false;
        }

        bit_buffer = (bit_buffer << 6) | static_cast<unsigned int>(value);
        bit_count += 6;
        while (bit_count >= 8) {
            bit_count -= 8;
            const unsigned char byte =
                static_cast<unsigned char>((bit_buffer >> bit_count) & 0xff);
            if (output_size < out->size()) {
                (*out)[output_size] = byte;
            }
            ++output_size;
        }
    }

    return output_size == kSha256Size;
}

std::string EncodePin(const std::array<unsigned char, kSha256Size>& bytes) {
    std::string encoded;
    encoded.reserve(44);

    size_t index = 0;
    while (index + 2 < bytes.size()) {
        const unsigned int value =
            (static_cast<unsigned int>(bytes[index]) << 16) |
            (static_cast<unsigned int>(bytes[index + 1]) << 8) |
            static_cast<unsigned int>(bytes[index + 2]);
        encoded.push_back(kBase64Alphabet[(value >> 18) & 0x3f]);
        encoded.push_back(kBase64Alphabet[(value >> 12) & 0x3f]);
        encoded.push_back(kBase64Alphabet[(value >> 6) & 0x3f]);
        encoded.push_back(kBase64Alphabet[value & 0x3f]);
        index += 3;
    }

    const size_t remaining = bytes.size() - index;
    if (remaining == 1) {
        const unsigned int value =
            static_cast<unsigned int>(bytes[index]) << 16;
        encoded.push_back(kBase64Alphabet[(value >> 18) & 0x3f]);
        encoded.push_back(kBase64Alphabet[(value >> 12) & 0x3f]);
        encoded.append("==");
    } else if (remaining == 2) {
        const unsigned int value =
            (static_cast<unsigned int>(bytes[index]) << 16) |
            (static_cast<unsigned int>(bytes[index + 1]) << 8);
        encoded.push_back(kBase64Alphabet[(value >> 18) & 0x3f]);
        encoded.push_back(kBase64Alphabet[(value >> 12) & 0x3f]);
        encoded.push_back(kBase64Alphabet[(value >> 6) & 0x3f]);
        encoded.push_back('=');
    }

    return encoded;
}

}  // namespace

CertificatePinResult ParseCertificatePinList(const std::string& input) {
    CertificatePinResult result;
    const std::string trimmed = TrimAsciiWhitespace(input);
    if (trimmed.empty()) {
        return result;
    }

    std::set<std::string> unique_pins;
    std::vector<std::string> ordered_pins;
    size_t begin = 0;
    while (begin <= trimmed.size()) {
        const size_t comma = trimmed.find(',', begin);
        const size_t end = comma == std::string::npos ? trimmed.size() : comma;
        const std::string pin =
            TrimAsciiWhitespace(trimmed.substr(begin, end - begin));

        std::array<unsigned char, kSha256Size> decoded{};
        if (!DecodePin(pin, &decoded)) {
            CertificatePinResult invalid;
            invalid.valid = false;
            return invalid;
        }

        const std::string normalized_pin = EncodePin(decoded);
        if (unique_pins.insert(normalized_pin).second) {
            ordered_pins.push_back(normalized_pin);
        }

        if (comma == std::string::npos) {
            break;
        }
        begin = comma + 1;
    }

    for (size_t i = 0; i < ordered_pins.size(); ++i) {
        if (i > 0) {
            result.normalized.push_back(',');
        }
        result.normalized += ordered_pins[i];
    }
    return result;
}

std::string BuildCertificatePinArgument(const std::string& input) {
    const CertificatePinResult parsed = ParseCertificatePinList(input);
    if (!parsed.valid || parsed.normalized.empty()) {
        return "";
    }
    return "--ignore-certificate-errors-spki-list=" + parsed.normalized;
}

}  // namespace quickdesk
