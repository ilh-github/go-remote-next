// The connection log row is where an owner finds out why a connection did not
// happen, so the fields the server already returns on every row are worth
// surfacing: the device name, the client address, and above all the failure
// reason. Rendering the raw device id alone throws away the one field that
// explains the row somebody opened this list to read.

function formatDuration(seconds) {
  const total = Number(seconds)
  if (!Number.isFinite(total) || total <= 0) return ''
  const mins = Math.floor(total / 60)
  const secs = Math.floor(total % 60)
  if (mins <= 0) return `${secs}s`
  return `${mins}m${secs}s`
}

// Trimmed, and never "undefined"/"null" for a field the server left empty.
function text(value) {
  if (value === undefined || value === null) return ''
  return String(value).trim()
}

export function describeConnectionLog(row) {
  const log = row || {}
  const deviceId = text(log.device_id)
  const deviceName = text(log.device_name)
  const errorMsg = text(log.error_msg)

  return {
    // The name is what the owner recognises; the id stays in the detail line
    // because it is what matches the device list and any support conversation.
    name: deviceName || deviceId,
    detail: deviceId,
    // Kept so the row can be keyed and its timestamp rendered without the
    // template reaching back into the raw API shape.
    id: log.id,
    createdAt: text(log.created_at),
    // Only a failure has a reason to show, and only when one was recorded. A
    // failed row with nothing recorded renders no reason rather than an empty
    // bracket or the word "undefined".
    reason: log.status === 'success' ? '' : errorMsg,
    duration: formatDuration(log.duration),
    ip: text(log.connect_ip),
    failed: log.status !== 'success',
  }
}
