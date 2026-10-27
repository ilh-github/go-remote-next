// /v1/admin/audit-logs (§2.2).
import { authJson } from './auth.js'
import { downloadFile } from './system.js'

const BASE = '/v1/admin/audit-logs'

export function getAuditLogs(params = {}) {
  const q = new URLSearchParams()
  if (params.cursor) q.set('cursor', params.cursor)
  if (params.limit)  q.set('limit',  params.limit)
  if (params.size && !params.limit) q.set('limit', params.size)
  if (params.action)   q.set('action', params.action)
  if (params.admin)    q.set('admin',  params.admin)
  // Server reads snake_case date_from / date_to.
  if (params.dateFrom) q.set('date_from', params.dateFrom)
  if (params.dateTo)   q.set('date_to',   params.dateTo)
  const qs = q.toString()
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
