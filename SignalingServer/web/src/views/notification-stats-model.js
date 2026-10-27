// What the notification delivery panel shows, from
// GET /v1/admin/notification-deliveries:stats.
//
// The number that matters is not `total`. A channel that answers 2xx and drops
// the event is indistinguishable from one that works unless you look at
// `unacked` - rows that reached an endpoint and were never confirmed by a
// human. That is the whole reason the ack loop exists, so it is shown next to
// the state counts rather than buried.
import { retentionLabel } from './storage-size.js'

const STATES = ['pending', 'delivering', 'delivered', 'failed']

export function statusRows(s) {
  const by = s?.by_status || {}
  // The server always sends all four. Rendering only what arrived would make
  // "zero failures" look like "never asked", so missing states show as 0.
  return STATES.map((state) => [state, Number(by[state]) || 0])
}

export function summary(s) {
  const rows = statusRows(s)
  const pending = rows.find(([state]) => state === 'pending')?.[1] || 0
  const delivering = rows.find(([state]) => state === 'delivering')?.[1] || 0
  const delivered = rows.find(([state]) => state === 'delivered')?.[1] || 0
  const failed = rows.find(([state]) => state === 'failed')?.[1] || 0
  const unacked = Number(s?.unacked) || 0
  const acked = Number(s?.acked) || 0

  // Two shapes of trouble, and they are not the same: rows nobody confirmed
  // (the channel lies) and rows that never left (the worker is stuck).
  const stuck = (pending + delivering) > 0 && delivered === 0
  const needsAttention = unacked > 0 || stuck

  return {
    total: Number(s?.total) || 0,
    rows,
    pending,
    delivering,
    delivered,
    failed,
    acked,
    unacked,
    needsAttention,
    retention: retentionLabel(s?.retention_days),
    archiveRetention: retentionLabel(s?.archive_retention_days),
    archiveDir: s?.archive_dir ? s.archive_dir : 'not set',
  }
}
