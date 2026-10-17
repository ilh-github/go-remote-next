const DEFAULT_OFFLINE_ALERT_MINUTES = 10

function trimmed(value) {
  return typeof value === 'string' ? value.trim() : ''
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

export { parseStoredEvents, normalizeWindows }
