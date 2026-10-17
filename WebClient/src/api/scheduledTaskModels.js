function trimmed(value) {
  return typeof value === 'string' ? value.trim() : ''
}

function uniqueStrings(values) {
  if (!Array.isArray(values)) return []
  return [...new Set(values.map(trimmed).filter(Boolean))]
}

function pad2(value) {
  return String(value).padStart(2, '0')
}

function localWallClockToRFC3339(raw, offsetMinutes) {
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

export function toScheduledTaskLocalValue(raw, options = {}) {
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

export function normalizeScheduledTaskList(data) {
  return {
    items: Array.isArray(data?.items)
      ? data.items.map((row) => ({
        ...row,
        max_attempts: Number(row?.max_attempts) || 1,
        attempts: Number(row?.attempts) || 0,
        repeat_seconds: Number(row?.repeat_seconds) || 0,
        enabled: !!row?.enabled,
        skip_holidays: !!row?.skip_holidays,
      }))
      : [],
    supportedKinds: uniqueStrings(data?.supported_kinds),
    supportedSchedules: uniqueStrings(data?.supported_schedules),
  }
}

export function buildCalendarSchedule(form) {
  const type = trimmed(form?.calendarType)
  const time = trimmed(form?.time)
  if (type === 'daily') return `daily@${time}`
  if (type === 'weekdays') return `weekdays@${time}`
  if (type === 'weekly') return `weekly@${trimmed(form?.weekday)}@${time}`
  if (type === 'monthly') return `monthly@${Number(form?.monthDay) || 1}@${time}`
  if (type === 'every') return `every@${Number(form?.everyDays) || 1}@days@${time}`
  return ''
}

export function buildScheduledTaskPayload(form, options = {}) {
  const payload = {
    kind: trimmed(form?.kind),
    max_attempts: Number(form?.max_attempts) || 1,
  }
  if (form?.mode === 'calendar') {
    return {
      ...payload,
      schedule: buildCalendarSchedule(form),
      timezone: trimmed(form?.timezone),
      skip_holidays: !!form?.skip_holidays,
    }
  }
  return {
    ...payload,
    run_at: localWallClockToRFC3339(form?.run_at, options.timeZoneOffsetMinutes),
    repeat_seconds: form?.mode === 'interval' ? Number(form?.repeat_seconds) || 0 : 0,
  }
}
