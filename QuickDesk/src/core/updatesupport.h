// Copyright 2026 QuickDesk Authors

#ifndef QUICKDESK_CORE_UPDATESUPPORT_H
#define QUICKDESK_CORE_UPDATESUPPORT_H

#include <cstdint>
#include <optional>
#include <string>
#include <vector>

namespace quickdesk {

std::optional<std::vector<std::uint64_t>> ParseUpdateVersion(
    const std::string& version);

int CompareUpdateVersions(const std::string& left, const std::string& right);

bool IsUpdateNewer(const std::string& latest, const std::string& current);

std::string SanitizeInstallerFileName(const std::string& proposed,
                                      const std::string& fallback);

std::string InstallerExtensionForPlatform(const std::string& platform);

bool UrlsHaveSameOrigin(const std::string& base_url,
                        const std::string& target_url);

}  // namespace quickdesk

#endif  // QUICKDESK_CORE_UPDATESUPPORT_H
