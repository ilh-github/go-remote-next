package handler

import (
	"errors"

	"quickdesk/signaling/internal/service"
)

var errStorageQuotaExceeded = errors.New("storage quota exceeded")

// storageWriteAllowance returns the largest body this request may write after
// accounting for both the per-upload cap and the retained directory quota.
// stagedBytes are already part of the directory usage and are subtracted only
// from the per-upload cap, never from the quota.
func storageWriteAllowance(dir string, quotaBytes, maxBytes, stagedBytes int64) (int64, error) {
	allowance := int64(-1)
	if quotaBytes > 0 {
		usage, err := service.InspectDirectoryStorage(dir, quotaBytes)
		if err != nil {
			return 0, err
		}
		if usage.RemainingBytes <= 0 {
			return 0, errStorageQuotaExceeded
		}
		allowance = usage.RemainingBytes
	}
	if maxBytes > 0 {
		remaining := maxBytes - stagedBytes
		if remaining < 0 {
			remaining = 0
		}
		if allowance < 0 || remaining < allowance {
			allowance = remaining
		}
	}
	return allowance, nil
}
