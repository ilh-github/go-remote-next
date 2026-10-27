// What the recording storage panel shows, derived from
// GET /v1/admin/recordings:storage.
//
// The server reports two byte totals on purpose: indexed_bytes is the database
// index sum, total_bytes walks the real recording directory and therefore
// includes thumbnails and .part staging files that no row points at. An operator
// deciding whether a quota is about to be hit needs the second number, so both
// are shown and the difference is named rather than silently dropped.
import { formatSize, quotaLabel, retentionLabel, staleLabel } from './storage-size.js'

export function storageRows(s) {
  const disk = Number(s?.total_bytes) || 0
  const indexed = Number(s?.indexed_bytes) || 0
  const quota = Number(s?.quota_bytes) || 0
  const remaining = Number(s?.remaining_bytes)
  return {
    indexed: formatSize(indexed),
    disk: formatSize(disk),
    quota: quotaLabel(quota),
    hasQuota: quota > 0,
    // -1 is the server's "no quota configured" marker. It must not render as
    // "0 B remaining", which reads as a full disk on an uncapped store.
    remaining: quota > 0 ? formatSize(remaining) : 'not set',
    over: s?.over_quota === true,
    artifacts: Number(s?.artifact_count) || 0,
    files: Number(s?.file_count) || 0,
    retention: retentionLabel(s?.retention_days),
    stale: staleLabel(s?.stale_after_minutes),
  }
}

// Bytes on disk that no index row accounts for: thumbnails and staged uploads.
export function storageGap(s) {
  const disk = Number(s?.total_bytes) || 0
  const indexed = Number(s?.indexed_bytes) || 0
  return {
    hidden: disk > indexed ? disk - indexed : 0,
    hasGap: disk > indexed,
  }
}
