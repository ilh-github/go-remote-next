// /v1/admin/audit-logs (§2.2).
import { authJson } from './auth.js'
import { downloadFile } from './system.js'
import { auditLogParams } from '../views/audit-filter.js'

const BASE = '/v1/admin/audit-logs'

// The full filter set lives in views/audit-filter.js, including the two this
// used to drop: `username` (which matches an end user's account as well as an
// administrator's, so "what did this account do" was unanswerable here) and
// `actor_user_id`.
export function getAuditLogs(params = {}) {
  const qs = auditLogParams(params).toString()
  return authJson(`${BASE}${qs ? `?${qs}` : ''}`)
}

export function exportAuditLogs(format) {
  return downloadFile(`${BASE}:export?format=${encodeURIComponent(format)}`)
}

// GET /v1/admin/audit-archive:search — the archived half of the same trail.
//
// Retention appends expired rows to a per-day JSONL file before deleting them, so
// the live list above only covers the retention window. This reads the archive
// back, which is the only way "what happened last week" survives the purge.
//
// A separate base rather than a path under BASE: gin keys routes by method and
// cannot hold two wildcard names (`:export` and `:search`) under one prefix.
const ARCHIVE_BASE = '/v1/admin/audit-archive'

export function searchAuditArchive(params = {}) {
  const q = new URLSearchParams()
  // The day to read, YYYY-MM-DD. Empty means today in UTC, which is what the
  // server does with an absent date.
  if (params.date) q.set('date', params.date)
  if (params.q) q.set('q', params.q)
  if (params.action) q.set('action', params.action)
  if (params.admin) q.set('admin', params.admin)
  if (params.limit) q.set('limit', params.limit)
  const qs = q.toString()
  return authJson(`${ARCHIVE_BASE}:search${qs ? `?${qs}` : ''}`)
}

// POST /v1/admin/audit-logs:sweep — run the audit retention pass now.
//
// Six independent bounds: age, per-actor, per-device, per-action, table-wide
// rows, and how long the daily archive files are kept. Any combination may be
// passed; the server rejects a request that supplies none of them. Rows may be
// kept forever while the JSONL they were purged into is not, which is why the
// archive bound is separate rather than implied by the age bound.
export function sweepAuditLogs(params = {}) {
  const body = {}
  if (params.olderThanDays) body.older_than_days = params.olderThanDays
  if (params.maxRows) body.max_rows = params.maxRows
  if (params.maxRowsPerActor) body.max_rows_per_actor = params.maxRowsPerActor
  if (params.maxRowsPerDevice) body.max_rows_per_device = params.maxRowsPerDevice
  if (params.maxRowsPerAction) body.max_rows_per_action = params.maxRowsPerAction
  if (params.archiveRetentionDays) body.archive_retention_days = params.archiveRetentionDays
  return authJson(`${BASE}:sweep`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}
