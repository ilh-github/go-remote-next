// Cross-user control sessions (/v1/admin/remote-sessions*).
//
// The list is already joined: every item is {session, recordings, access_log,
// counts}, so the page never needs a second request per row to show what a
// session produced. The revoke actions are the same ones the owner has, issued
// at whichever instance this admin is talking to - the server addresses the
// command at the instance that actually holds the socket.

import { authJson } from './auth.js'

const BASE = '/v1/admin'

function buildQuery(params = {}) {
  const q = new URLSearchParams()
  if (params.status && params.status !== 'all') q.set('status', params.status)
  if (params.deviceId) q.set('device_id', params.deviceId)
  if (params.userId) q.set('user_id', String(params.userId))
  if (params.cursor) q.set('cursor', params.cursor)
  if (params.limit) q.set('limit', String(params.limit))
  const s = q.toString()
  return s ? `?${s}` : ''
}

export function listRemoteSessions(params = {}) {
  return authJson(`${BASE}/remote-sessions${buildQuery(params)}`)
}

export function getRemoteSession(sessionId) {
  return authJson(`${BASE}/remote-sessions/${encodeURIComponent(sessionId)}`)
}

export function revokeRemoteSession(sessionId) {
  return authJson(`${BASE}/remote-sessions/${encodeURIComponent(sessionId)}/revoke`, {
    method: 'POST',
  })
}

// "Clear this machine": reports per session, so a partial result is expected.
export function revokeDeviceSessions(deviceId) {
  return authJson(`${BASE}/devices/${encodeURIComponent(deviceId)}/sessions:revoke`, {
    method: 'POST',
  })
}

export function banAccessSource(deviceId, source) {
  return authJson(`${BASE}/devices/${encodeURIComponent(deviceId)}/access-bans`, {
    method: 'POST',
    body: JSON.stringify({ source }),
  })
}

export function unbanAccessSource(deviceId, source) {
  return authJson(`${BASE}/devices/${encodeURIComponent(deviceId)}/access-bans:remove`, {
    method: 'POST',
    body: JSON.stringify({ source }),
  })
}

// GET /v1/admin/recordings:storage — what the recording directory costs on disk.
//
// Distinct from the per-session recording counts: those index rows, this walks
// the real directory and so also counts thumbnails and .part staging files that
// no row points at. An operator deciding whether a quota is about to be hit
// needs the walked number.
const RECORDING_BASE = '/v1/admin/recordings'

export function getRecordingStorage() {
  return authJson(`${RECORDING_BASE}:storage`)
}

// POST /v1/admin/recordings:sweep — run the stale + retention + orphan pass now.
//
// Takes no arguments on purpose: it runs the configured window, so what an
// operator sees here is exactly what the periodic runner will do, rather than a
// one-off window that only this call uses.
//
// Returns {stale_marked_failed, artifacts_purged, bytes_freed, orphans_removed,
// uploads_reaped}. uploads_reaped counts abandoned resumable-upload staging
// files, which is the count most likely to be non-zero on a busy server.
export function sweepRecordings() {
  return authJson(`${RECORDING_BASE}:sweep`, { method: 'POST' })
}
