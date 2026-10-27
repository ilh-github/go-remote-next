// Admin-side access to a recording's bytes.
//
// The admin list shows a row's size and status; the media is a separate route,
// and a browser cannot put a Bearer token on a plain link or an <img src>. So
// the UI fetches through authFetch and hands the browser a blob URL, the same
// pattern the rest of the admin app already uses for downloads.
//
// Two guards, because both wrong answers are worse than no button: offering a
// download for a row the server is still writing produces a failure with no
// explanation, and a broken image in a table reads as a bug rather than as
// "no thumbnail was captured".
const ARTIFACT_ROUTE = '/v1/admin/recordings'

export function recordingId(id) {
  const n = Number(id)
  return Number.isInteger(n) && n > 0 ? n : null
}

export function artifactUrl(id) {
  const n = recordingId(id)
  return n ? `${ARTIFACT_ROUTE}/${n}/artifact` : ''
}

export function thumbnailUrl(id) {
  const n = recordingId(id)
  return n ? `${ARTIFACT_ROUTE}/${n}/thumbnail` : ''
}

export function canDownload(row) {
  return Boolean(row?.status === 'completed' && Number(row.size_bytes) > 0)
}

export function canThumbnail(row) {
  return Boolean(row?.thumbnail_path)
}

export function artifactName(id) {
  const n = recordingId(id)
  return n ? `recording-${n}.h264` : ''
}
