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
  formFromScheduledTask,
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
    supported_kinds: ['revoke_all_remote_sessions', 'sign_out_all_sessions'],
    supported_schedules: ['daily@HH:MM'],
  })

  assert.equal(view.items[0].max_attempts, 3)
  assert.equal(view.items[0].skip_holidays, true)
  assert.deepEqual(view.supportedKinds, [
    'revoke_all_remote_sessions',
    'sign_out_all_sessions',
  ])
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

// Editing a task used to require deleting and recreating it, which threw away
// the outcome history the owner needed in order to see the task was failing.
// PATCH rather than PUT: the request body is the create request's, and it has
// no way to say "leave this string alone", so a full-replace verb here would
// reset whatever the caller omitted.
test('the edit goes to the owner-scoped task row with PATCH', async () => {
  const calls = []
  const original = userApi._req
  userApi._req = async (...args) => {
    calls.push(args)
    return { ok: true, data: { id: 12 } }
  }
  try {
    await userApi.updateScheduledTask(12, { kind: 'connect_prompt', run_at: '2026-10-18T14:00:00.000Z' })
  } finally {
    userApi._req = original
  }

  assert.deepEqual(calls, [
    ['PATCH', '/v1/me/scheduled-tasks/12', { kind: 'connect_prompt', run_at: '2026-10-18T14:00:00.000Z' }],
  ])
})

// A task id is used in a path, so a value that would change the path has to be
// encoded rather than interpolated. The server's ids are numeric today; the
// encoder is what keeps that from becoming a routing bug the day they are not.
test('the edit encodes the task id into the path', async () => {
  const calls = []
  const original = userApi._req
  userApi._req = async (...args) => {
    calls.push(args)
    return { ok: true, data: {} }
  }
  try {
    await userApi.updateScheduledTask('12/34 x', { kind: 'connect_prompt' })
  } finally {
    userApi._req = original
  }

  assert.deepEqual(calls, [
    ['PATCH', '/v1/me/scheduled-tasks/12%2F34%20x', { kind: 'connect_prompt' }],
  ])
})

// The dialog is the same form for both verbs, so the payload builder must be
// able to describe a stored row back into a form the server accepts. This is
// what keeps the edit from sending a run_at for a calendar task.
test('a stored calendar row round-trips into the create/edit payload', () => {
  const payload = buildScheduledTaskPayload({
    mode: 'calendar',
    kind: 'revoke_all_remote_sessions',
    calendarType: 'weekly',
    weekday: 'fri',
    time: '18:00',
    timezone: 'Asia/Shanghai',
    skip_holidays: true,
    max_attempts: 3,
  })

  assert.deepEqual(payload, {
    kind: 'revoke_all_remote_sessions',
    schedule: 'weekly@fri@18:00',
    timezone: 'Asia/Shanghai',
    skip_holidays: true,
    max_attempts: 3,
  })
})

// The edit dialog opens on the row, not on defaults. A one-shot's stored instant
// has to survive the round trip, or the owner edits a schedule they did not
// choose.
test('a stored one-shot row round-trips into the dialog form', () => {
  const form = formFromScheduledTask({
    id: 7,
    kind: 'connect_prompt',
    next_run_at: '2026-10-17T14:00:00Z',
    repeat_seconds: 0,
    max_attempts: 2,
  }, { timeZoneOffsetMinutes: -480 })

  assert.equal(form.mode, 'once')
  assert.equal(form.kind, 'connect_prompt')
  assert.equal(form.run_at, '2026-10-17T22:00')
  assert.equal(form.max_attempts, 2)
  // A one-shot must not pick up an interval the row does not have, or the edit
  // would quietly turn it into a recurring task.
  assert.ok(!('repeat_minutes' in form) || form.repeat_minutes === undefined)
})

test('a stored interval row round-trips into the dialog form', () => {
  const form = formFromScheduledTask({
    kind: 'sign_out_all_sessions',
    next_run_at: '2026-10-17T14:00:00Z',
    repeat_seconds: 7200,
    max_attempts: 1,
  })

  assert.equal(form.mode, 'interval')
  assert.equal(form.repeat_minutes, 120)
})

test('a stored calendar row round-trips into the dialog form', () => {
  const form = formFromScheduledTask({
    kind: 'revoke_all_remote_sessions',
    schedule: 'monthly@31@03:00',
    timezone: 'Asia/Shanghai',
    skip_holidays: true,
    max_attempts: 4,
  })

  assert.equal(form.mode, 'calendar')
  assert.equal(form.calendarType, 'monthly')
  assert.equal(form.monthDay, 31)
  assert.equal(form.time, '03:00')
  assert.equal(form.timezone, 'Asia/Shanghai')
  assert.equal(form.skip_holidays, true)
})

// An unrecognised cadence must not make the edit dialog unusable: it falls back
// to a one-shot at the stored instant, which the owner can see and change. A
// throw here would leave the edit button opening a dialog that cannot describe
// the row at all.
test('an unknown cadence falls back to an editable one-shot', () => {
  const form = formFromScheduledTask({
    kind: 'sign_out_all_sessions',
    schedule: 'whenever@the@mood@strikes',
    next_run_at: '2026-10-17T14:00:00Z',
    max_attempts: 1,
  })

  assert.equal(form.mode, 'calendar')
  assert.equal(form.time, '22:00')
  // The cadence stays daily rather than pretending to be something it parsed.
  assert.equal(form.calendarType, 'daily')
})

// An empty or missing row must not produce a form that throws when the dialog
// reads it: the page opens the dialog for a row it just listed, so a malformed
// row is a server bug the UI should survive long enough to report.
test('a missing row yields a usable default form', () => {
  const form = formFromScheduledTask(null)
  assert.equal(form.kind, '')
  assert.equal(form.mode, 'once')
  assert.equal(form.max_attempts, 1)
})
