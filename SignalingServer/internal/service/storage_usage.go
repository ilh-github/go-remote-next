package service

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// DirectoryStorageUsage is the retained-file view of one storage directory.
// Staging files are included because they occupy the same filesystem and a
// quota that ignored them could still fill the disk during a resumable upload.
type DirectoryStorageUsage struct {
	Files          int64 `json:"files"`
	UsedBytes      int64 `json:"used_bytes"`
	QuotaBytes     int64 `json:"quota_bytes"`
	RemainingBytes int64 `json:"remaining_bytes"`
	OverQuota      bool  `json:"over_quota"`
}

// InspectDirectoryStorage walks a storage root and reports the bytes held by
// regular files. A missing directory is empty, which keeps optional artifact
// stores from turning a status request into an error. Symlinks are skipped so a
// quota report cannot wander outside the configured tree.
func InspectDirectoryStorage(dir string, quota int64) (DirectoryStorageUsage, error) {
	usage := DirectoryStorageUsage{QuotaBytes: quota, RemainingBytes: -1}
	root := strings.TrimSpace(dir)
	if root == "" {
		if quota > 0 {
			usage.RemainingBytes = quota
		}
		return usage, nil
	}
	if _, err := os.Lstat(root); err != nil {
		if os.IsNotExist(err) {
			if quota > 0 {
				usage.RemainingBytes = quota
			}
			return usage, nil
		}
		return usage, err
	}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		usage.Files++
		usage.UsedBytes += info.Size()
		return nil
	})
	if err != nil {
		return DirectoryStorageUsage{}, err
	}
	if quota > 0 {
		usage.RemainingBytes = quota - usage.UsedBytes
		if usage.RemainingBytes < 0 {
			usage.RemainingBytes = 0
		}
		usage.OverQuota = usage.UsedBytes > quota
	}
	return usage, nil
}
