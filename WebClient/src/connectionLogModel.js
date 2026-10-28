/**
 * Describes the toast a clear-history call earns from the server's answer.
 *
 * The endpoint answers `{removed: N}`, and the page's comment says the
 * confirmation uses that number "rather than a generic 'cleared'". The count is
 * the only thing that distinguishes clearing one row from clearing a year of
 * them, and an operator who just wiped 137 entries is owed the number.
 *
 * Three cases, kept separate on purpose:
 *  - a usable count above zero names it;
 *  - a usable count of zero says nothing was there, rather than claiming a
 *    generic success over an empty list;
 *  - a response without a usable count falls back to the generic message,
 *    because inventing "cleared 0" would be a specific claim about an answer
 *    that was not actually given.
 *
 * @param {{removed?: unknown}} response the server body
 * @returns {{key: string, params: object}}
 */
export function describeClearResult(response) {
  const raw = response && typeof response === 'object' ? response.removed : undefined
  const removed = Number(raw)
  // `Number(null)` is 0 and `Number('')` is 0, so the raw value is checked for
  // nullish and non-numeric shapes before the count is trusted.
  const usable = raw !== null && raw !== undefined && raw !== '' && Number.isFinite(removed)
  if (!usable) {
    return { key: 'devices.logsCleared', params: {} }
  }
  if (removed <= 0) {
    return { key: 'devices.logsClearedNone', params: {} }
  }
  return { key: 'devices.logsClearedCount', params: { count: removed } }
}

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

// QD-206 promises quick connect as well as the history itself, and the history
// is the list an owner opens to find a machine they were on earlier. The access
// code deliberately does not live on the history row -- the API does not store
// it, which matches upstream issue #18's "do NOT persist verification-code" --
// so it is joined from the device list, which does return it.
export function describeConnectionLog(row, devices) {
  const log = row || {}
  const deviceId = text(log.device_id)
  const deviceName = text(log.device_name)
  const errorMsg = text(log.error_msg)

  // Matched on device_id alone. A row whose id matches nothing stays
  // unconnectable rather than borrowing a neighbour's code, and a device present
  // but without a code is also unconnectable because connecting has to verify
  // one -- offering the button there would only produce a "no access code" toast.
  const bound = Array.isArray(devices)
    ? devices.find((device) => text(device?.device_id) === deviceId && deviceId !== '')
    : undefined
  const accessCode = bound ? text(bound.access_code) : ''

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
    canConnect: accessCode !== '',
    accessCode,
  }
}
