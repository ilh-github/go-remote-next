const DEFAULT_OFFLINE_ALERT_MINUTES = 10
const DEFAULT_METRIC_SUSTAINED_MINUTES = 5

function trimmed(value) {
  return typeof value === 'string' ? value.trim() : ''
}

function finiteNumber(value, fallback = 0) {
  const number = Number(value)
  return Number.isFinite(number) ? number : fallback
}

function nullableNumber(value) {
  if (value === null || value === undefined || value === '') return null
  const number = Number(value)
  return Number.isFinite(number) ? number : null
}

function uniqueStrings(values) {
  if (!Array.isArray(values)) return []
  return [...new Set(values.map(trimmed).filter(Boolean))]
}

function parseStoredEvents(raw) {
  if (Array.isArray(raw)) return uniqueStrings(raw)
  if (typeof raw !== 'string' || raw.trim() === '') return []
  try {
    return uniqueStrings(JSON.parse(raw))
  } catch {
    return []
  }
}

function normalizeWindow(raw) {
  const days = Array.isArray(raw?.days)
    ? [...new Set(raw.days.map(Number).filter((day) => Number.isInteger(day) && day >= 0 && day <= 6))]
      .sort((a, b) => a - b)
    : []
  return {
    days,
    start: trimmed(raw?.start),
    end: trimmed(raw?.end),
  }
}

function normalizeWindows(raw) {
  return Array.isArray(raw) ? raw.map(normalizeWindow) : []
}

function wallClockToRFC3339(raw, offsetMinutes) {
  const value = trimmed(raw)
  if (!value) return null
  if (!Number.isFinite(offsetMinutes)) {
    const parsed = new Date(value)
    return Number.isNaN(parsed.getTime()) ? null : parsed.toISOString()
  }
  const match = value.match(/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})(?::(\d{2}))?$/)
  if (!match) return null
  const wall = Date.UTC(
    Number(match[1]),
    Number(match[2]) - 1,
    Number(match[3]),
    Number(match[4]),
    Number(match[5]),
    Number(match[6] || 0),
  )
  return new Date(wall + offsetMinutes * 60_000).toISOString()
}

function pad2(value) {
  return String(value).padStart(2, '0')
}

export function toDatetimeLocalValue(raw, options = {}) {
  const value = trimmed(raw)
  if (!value) return ''
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return ''
  const offsetMinutes = options.timeZoneOffsetMinutes
  if (Number.isFinite(offsetMinutes)) {
    const local = new Date(parsed.getTime() - offsetMinutes * 60_000)
    return [
      local.getUTCFullYear(),
      '-',
      pad2(local.getUTCMonth() + 1),
      '-',
      pad2(local.getUTCDate()),
      'T',
      pad2(local.getUTCHours()),
      ':',
      pad2(local.getUTCMinutes()),
    ].join('')
  }
  return [
    parsed.getFullYear(),
    '-',
    pad2(parsed.getMonth() + 1),
    '-',
    pad2(parsed.getDate()),
    'T',
    pad2(parsed.getHours()),
    ':',
    pad2(parsed.getMinutes()),
  ].join('')
}

export function normalizeChannelList(data) {
  return {
    items: Array.isArray(data?.items)
      ? data.items.map((row) => ({ ...row, events: parseStoredEvents(row?.events) }))
      : [],
    supportedEvents: uniqueStrings(data?.supported_events),
    signatureHeader: trimmed(data?.signature_header),
    signatureScheme: trimmed(data?.signature_scheme),
  }
}

export function buildChannelPayload(form, { partial = false } = {}) {
  const payload = {
    name: trimmed(form?.name),
    url: trimmed(form?.url),
    events: uniqueStrings(form?.events),
    enabled: !!form?.enabled,
  }
  const secret = trimmed(form?.secret)
  if (!partial || secret) payload.secret = secret
  return payload
}

export function normalizeQuietHours(data) {
  return {
    snoozed_until: data?.snoozed_until || null,
    quiet_hours: normalizeWindows(data?.quiet_hours),
    timezone: trimmed(data?.timezone),
    active: !!data?.active,
    active_until: data?.active_until || null,
  }
}

export function buildQuietHoursPayload(form, options = {}) {
  return {
    snoozed_until: wallClockToRFC3339(form?.snoozed_until, options.timeZoneOffsetMinutes),
    quiet_hours: normalizeWindows(form?.quiet_hours),
    timezone: trimmed(form?.timezone),
  }
}

export function normalizeOfflineAlert(data) {
  return {
    device_id: trimmed(data?.device_id),
    enabled: !!data?.enabled,
    after_minutes: Number(data?.after_minutes) || DEFAULT_OFFLINE_ALERT_MINUTES,
    repeat_minutes: Number(data?.repeat_minutes) || 0,
    snoozed_until: data?.snoozed_until || null,
    maintenance_windows: normalizeWindows(data?.maintenance_windows),
    maintenance_timezone: trimmed(data?.maintenance_timezone),
    in_maintenance_window: !!data?.in_maintenance_window,
    offline_since: data?.offline_since || null,
    last_alerted_at: data?.last_alerted_at || null,
    updated_at: data?.updated_at || null,
  }
}

export function buildOfflineAlertPayload(form) {
  return {
    enabled: !!form?.enabled,
    after_minutes: Number(form?.after_minutes) || DEFAULT_OFFLINE_ALERT_MINUTES,
    repeat_minutes: Number(form?.repeat_minutes) || 0,
    snoozed_until: wallClockToRFC3339(form?.snoozed_until),
    maintenance_windows: normalizeWindows(form?.maintenance_windows),
    maintenance_timezone: trimmed(form?.maintenance_timezone),
  }
}

function normalizeMetricState(raw) {
  return {
    exceeded_since: raw?.exceeded_since || null,
    last_alerted_at: raw?.last_alerted_at || null,
  }
}

export function normalizeMetricAlert(data) {
  return {
    device_id: trimmed(data?.device_id),
    enabled: !!data?.enabled,
    cpu_percent: finiteNumber(data?.cpu_percent),
    memory_percent: finiteNumber(data?.memory_percent),
    disk_percent: finiteNumber(data?.disk_percent),
    sustained_minutes: finiteNumber(data?.sustained_minutes, DEFAULT_METRIC_SUSTAINED_MINUTES),
    repeat_minutes: finiteNumber(data?.repeat_minutes),
    snoozed_until: data?.snoozed_until || null,
    current: {
      captured_at: data?.current?.captured_at || null,
      cpu_percent: nullableNumber(data?.current?.cpu_percent),
      memory_percent: nullableNumber(data?.current?.memory_percent),
      disk_percent: nullableNumber(data?.current?.disk_percent),
      disk_mount: trimmed(data?.current?.disk_mount),
    },
    states: {
      cpu: normalizeMetricState(data?.states?.cpu),
      memory: normalizeMetricState(data?.states?.memory),
      disk: normalizeMetricState(data?.states?.disk),
    },
  }
}

export function buildMetricAlertPayload(form, options = {}) {
  return {
    enabled: !!form?.enabled,
    cpu_percent: finiteNumber(form?.cpu_percent),
    memory_percent: finiteNumber(form?.memory_percent),
    disk_percent: finiteNumber(form?.disk_percent),
    sustained_minutes: finiteNumber(form?.sustained_minutes, DEFAULT_METRIC_SUSTAINED_MINUTES),
    repeat_minutes: finiteNumber(form?.repeat_minutes),
    snoozed_until: wallClockToRFC3339(form?.snoozed_until, options.timeZoneOffsetMinutes),
  }
}

// Serializes the retry-failed replay.
//
// The server takes seven parameters; the UI used to send an empty body, so all
// seven were unreachable. Two of them are safety features rather than
// conveniences and are the reason this exists:
//
//   dry_run            report how many rows a replay would touch, before
//                      touching them. An operator re-queuing a backlog after an
//                      outage is guessing otherwise.
//   spread_over_minutes pace the replay instead of making it all due at once,
//                      which matters precisely because the outage that parked
//                      the rows is also why the endpoint is fragile right now.
//
// A blank field is omitted rather than sent as zero. The server reads 0 as "no
// cap" / "no pacing", so a value the user never entered would look like a choice
// they made - and the difference between "left blank" and "explicitly none" is
// exactly what an operator needs to see in the request they are about to send.
export function buildRetryFailedPayload(form = {}) {
  const payload = {}

  if (typeof form?.dryRun === 'boolean') payload.dry_run = form.dryRun

  const spread = Number(form?.spreadOverMinutes)
  if (Number.isFinite(spread) && spread > 0) payload.spread_over_minutes = Math.floor(spread)

  const limit = Number(form?.limit)
  if (Number.isFinite(limit) && limit > 0) payload.limit = Math.floor(limit)

  const eventType = String(form?.eventType ?? '').trim()
  if (eventType) payload.event_type = eventType

  const deviceId = String(form?.deviceId ?? '').trim()
  if (deviceId) payload.device_id = deviceId

  const after = parseReplayWindowEdge(form?.createdAfter, 'created_after')
  const before = parseReplayWindowEdge(form?.createdBefore, 'created_before')
  if (after && before && after > before) {
    // The server answers an empty window with "matched: 0", which sends the
    // operator hunting for rows that cannot exist.
    throw new Error('created_after must not be later than created_before')
  }
  if (after) payload.created_after = after.toISOString()
  if (before) payload.created_before = before.toISOString()

  if (Array.isArray(form?.channelIds) && form.channelIds.length > 0) {
    payload.channel_ids = form.channelIds.map((id) => {
      const parsed = Number(id)
      if (!Number.isFinite(parsed) || parsed <= 0) {
        throw new Error('channel_ids must all be positive')
      }
      return Math.floor(parsed)
    })
  }

  return payload
}

function parseReplayWindowEdge(value, field) {
  const raw = String(value ?? '').trim()
  if (!raw) return null
  const parsed = new Date(raw)
  if (Number.isNaN(parsed.getTime())) {
    throw new Error(`${field} must be an RFC3339 timestamp`)
  }
  return parsed
}

export { parseStoredEvents, normalizeWindows }
