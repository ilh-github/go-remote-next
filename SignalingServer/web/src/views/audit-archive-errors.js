// How the archive panel reads a failed search.
//
// The server answers "cannot archive at all" and "nothing archived that day"
// with different statuses, but both messages contain the word "archive", so
// matching on the text reports the wrong cause. The distinction matters: one
// means an env var is missing, the other means the archive is healthy and the
// answer is genuinely empty.
export function archiveErrorKind(error) {
  const code = error && error.code
  const status = error && error.status
  if (status === 404 || code === 'NOT_FOUND') return 'no_file'
  if (status === 400 || code === 'INVALID_REQUEST') {
    // Only the 400 that is specifically about configuration. A bad date or a
    // bad limit is also a 400, and must stay an error.
    const detail = String((error && error.detail) || (error && error.message) || '')
    if (/not configured/i.test(detail)) return 'not_configured'
  }
  return 'error'
}
