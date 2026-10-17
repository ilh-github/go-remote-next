import assert from 'node:assert/strict'
import test from 'node:test'

globalThis.localStorage = {
  _values: new Map(),
  getItem(key) {
    return this._values.has(key) ? this._values.get(key) : null
  },
  setItem(key, value) {
    this._values.set(key, String(value))
  },
  removeItem(key) {
    this._values.delete(key)
  },
}

const {
  buildCalendarSchedule,
  buildScheduledTaskPayload,
  normalizeScheduledTaskList,
  toScheduledTaskLocalValue,
} = await import('../src/api/scheduledTaskModels.js')
const { userApi } = await import('../src/api/userApi.js')

test('scheduled task list keeps server-supported kinds and schedules', () => {
  const view = normalizeScheduledTaskList({
    items: [{
      id: 12,
      kind: 'sign_out_all_sessions',
      next_run_at: '2026-10-17T14:00:00Z',
      repeat_seconds: 0,
      enabled: false,
      schedule: 'daily@22:00',
      timezone: 'Asia/Shanghai',
      skip_holidays: true,
      max_attempts: 3,
      attempts: 0,
      last_status: 'ok',
    }],
    supported_kinds: ['sign_out_all_sessions'],
    supported_schedules: ['daily@HH:MM'],
  })

  assert.equal(view.items[0].max_attempts, 3)
  assert.equal(view.items[0].skip_holidays, true)
  assert.deepEqual(view.supportedKinds, ['sign_out_all_sessions'])
  assert.deepEqual(view.supportedSchedules, ['daily@HH:MM'])
})

test('one-shot and interval payloads send UTC run_at and the right repeat', () => {
  assert.deepEqual(buildScheduledTaskPayload({
    mode: 'once',
    kind: 'sign_out_all_sessions',
    run_at: '2026-10-17T22:00',
    repeat_seconds: 3600,
    max_attempts: 2,
  }, { timeZoneOffsetMinutes: -480 }), {
    kind: 'sign_out_all_sessions',
    run_at: '2026-10-17T14:00:00.000Z',
    repeat_seconds: 0,
    max_attempts: 2,
  })

  assert.deepEqual(buildScheduledTaskPayload({
    mode: 'interval',
    kind: 'sign_out_all_sessions',
    run_at: '2026-10-17T22:00',
    repeat_seconds: 3600,
    max_attempts: 1,
  }, { timeZoneOffsetMinutes: -480 }), {
    kind: 'sign_out_all_sessions',
    run_at: '2026-10-17T14:00:00.000Z',
    repeat_seconds: 3600,
    max_attempts: 1,
  })
})

test('calendar payloads never send run_at or repeat_seconds', () => {
  assert.deepEqual(buildScheduledTaskPayload({
    mode: 'calendar',
    kind: 'sign_out_all_sessions',
    run_at: '2026-10-17T22:00',
    repeat_seconds: 3600,
    max_attempts: 4,
    calendarType: 'weekly',
    weekday: 'mon',
    time: '22:30',
    timezone: ' Asia/Shanghai ',
    skip_holidays: true,
  }), {
    kind: 'sign_out_all_sessions',
    schedule: 'weekly@mon@22:30',
    timezone: 'Asia/Shanghai',
    skip_holidays: true,
    max_attempts: 4,
  })
})

test('calendar schedule builder covers every supported cadence', () => {
  assert.equal(buildCalendarSchedule({ calendarType: 'daily', time: '09:05' }), 'daily@09:05')
  assert.equal(buildCalendarSchedule({ calendarType: 'weekdays', time: '09:05' }), 'weekdays@09:05')
  assert.equal(buildCalendarSchedule({ calendarType: 'weekly', weekday: 'fri', time: '18:00' }), 'weekly@fri@18:00')
  assert.equal(buildCalendarSchedule({ calendarType: 'monthly', monthDay: 31, time: '03:00' }), 'monthly@31@03:00')
  assert.equal(buildCalendarSchedule({ calendarType: 'every', everyDays: 3, time: '07:30' }), 'every@3@days@07:30')
})

test('server run instants round-trip into datetime-local controls', () => {
  assert.equal(
    toScheduledTaskLocalValue('2026-10-17T14:00:00Z', { timeZoneOffsetMinutes: -480 }),
    '2026-10-17T22:00',
  )
})

test('scheduled task API uses owner-scoped v1 paths', async () => {
  const calls = []
  const original = userApi._req
  userApi._req = async (...args) => {
    calls.push(args)
    return { ok: true, data: {} }
  }
  try {
    await userApi.fetchScheduledTasks()
    await userApi.createScheduledTask({ kind: 'sign_out_all_sessions' })
    await userApi.deleteScheduledTask(12)
  } finally {
    userApi._req = original
  }

  assert.deepEqual(calls, [
    ['GET', '/v1/me/scheduled-tasks'],
    ['POST', '/v1/me/scheduled-tasks', { kind: 'sign_out_all_sessions' }],
    ['DELETE', '/v1/me/scheduled-tasks/12'],
  ])
})
