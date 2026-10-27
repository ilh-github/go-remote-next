// User-notification delivery outbox (/v1/admin/notification-deliveries).
//
// This is the operator's view of a channel that answers 2xx and still drops the
// event: `unacked` counts rows that reached an endpoint and were never
// confirmed, which is the only place that failure is visible.
import { authJson } from './auth.js'

const BASE = '/v1/admin/notification-deliveries'

// GET /v1/admin/notification-deliveries:stats
//
// {total, by_status, acked, unacked, retention_days, archive_retention_days,
//  archive_dir}. by_status always carries all four states, zeroes included.
export function getNotificationDeliveryStats() {
  return authJson(`${BASE}:stats`)
}

// POST /v1/admin/notification-deliveries:sweep — run the retention pass now.
//
// Both bounds are optional and independent: rows may be kept forever while the
// JSONL files they were purged into are not. Sending an empty object means "use
// the configured windows", which the server distinguishes from no body at all
// only by Content-Length - both land on the same defaults here, so this always
// sends a body rather than omitting it.
export function sweepNotificationDeliveries(params = {}) {
  const body = {}
  if (params.olderThanDays) body.older_than_days = params.olderThanDays
  if (params.archiveRetentionDays) body.archive_retention_days = params.archiveRetentionDays
  return authJson(`${BASE}:sweep`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}
