// QD-208: the session detail aggregates a session with its recordings and its
// access log. The server sends all three as full arrays -- `detailEnvelope` puts
// the recordings themselves in the payload -- so the client's job is to describe
// them for display, not to recount them.
//
// The counts are derived server-side from the same arrays, so when the two
// disagree the arrays win: they are what the owner acts on, and a count that
// says 99 next to two rows would be the lie.

function formatDuration(seconds) {
  const total = Number(seconds)
  if (!Number.isFinite(total) || total <= 0) return ''
  const mins = Math.floor(total / 60)
  const secs = Math.floor(total % 60)
  if (mins <= 0) return `${secs}s`
  return `${mins}m${secs}s`
}

function text(value) {
  if (value === undefined || value === null) return ''
  return String(value).trim()
}

function describeRecording(row) {
  const recording = row || {}
  // Playable means there is media to play. A row the server is still writing, or
  // one whose artifact never reached a browser-reachable URL, must not offer a
  // button that can only fail.
  const url = text(recording.artifact_url)
  return {
    id: recording.id,
    sessionId: text(recording.session_id),
    status: text(recording.status),
    createdAt: text(recording.created_at),
    duration: formatDuration(recording.duration),
    format: text(recording.format),
    playable: url !== '',
    artifactUrl: url,
  }
}

function describeAccessLog(row) {
  const log = row || {}
  return {
    id: log.id,
    createdAt: text(log.created_at),
    result: text(log.result),
    ip: text(log.ip),
  }
}

// A recording's artifact URL is relative on some deployments and absolute on
// others, so it is anchored against the HTTP form of the signaling server. The
// same resolution the recordings page needs, kept here so the session detail and
// that page cannot drift into two different answers about whether a row is
// playable.
export function resolveRecordingUrl(row, serverUrl) {
  if (!row) return null
  const url = text(row.artifact_url)
  if (!url) return null
  if (/^(https?:)?\/\//i.test(url)) return url
  let base = text(serverUrl).replace(/\/+$/, '')
  if (base.startsWith('wss://')) base = base.replace(/^wss:\/\//, 'https://')
  else if (base.startsWith('ws://')) base = base.replace(/^ws:\/\//, 'http://')
  if (!base) return url
  return `${base}${url.startsWith('/') ? '' : '/'}${url}`
}

export function describeSessionDetail(detail) {
  const payload = detail || {}
  const recordings = Array.isArray(payload.recordings) ? payload.recordings : []
  const accessLog = Array.isArray(payload.access_log) ? payload.access_log : []
  const counts = payload.counts || {}
  return {
    session: payload.session || {},
    recordings: recordings.map(describeRecording),
    accessLog: accessLog.map(describeAccessLog),
    counts: {
      // Fall back to the array length rather than to zero, so a payload that
      // omits its counts still shows the rows it actually contains.
      recordings: Number.isFinite(counts.recordings) ? counts.recordings : recordings.length,
      accessLog: Number.isFinite(counts.access_log) ? counts.access_log : accessLog.length,
    },
  }
}
