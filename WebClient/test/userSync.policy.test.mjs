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

const { userSync } = await import('../src/api/userSync.js')

test('device policy events expose the updated deny list', () => {
  userSync.stop()
  let detail = null
  userSync.addEventListener(
    'access-policy-changed',
    (event) => { detail = event.detail },
    { once: true },
  )

  userSync._onMessage({
    data: JSON.stringify({
      type: 'device.access_policy.changed',
      server_rev: 1,
      data: {
        device_id: '123456789',
        deny_ips: ['203.0.113.7'],
      },
    }),
  })

  assert.deepEqual(detail, {
    device_id: '123456789',
    deny_ips: ['203.0.113.7'],
    data: {
      device_id: '123456789',
      deny_ips: ['203.0.113.7'],
    },
  })
})

test('remote session lifecycle events expose the changed session', () => {
  userSync.stop()
  const details = []
  userSync.addEventListener(
    'remote-session-changed',
    (event) => { details.push(event.detail) },
  )

  userSync._onMessage({
    data: JSON.stringify({
      type: 'remote_session.started',
      server_rev: 2,
      data: {
        session_id: 'sig_started',
        device_id: '123456789',
      },
    }),
  })
  userSync._onMessage({
    data: JSON.stringify({
      type: 'remote_session.ended',
      server_rev: 3,
      data: {
        session_id: 'sig_started',
        device_id: '123456789',
        end_reason: 'disconnected',
      },
    }),
  })

  assert.deepEqual(details, [
    {
      type: 'remote_session.started',
      session_id: 'sig_started',
      device_id: '123456789',
      data: {
        session_id: 'sig_started',
        device_id: '123456789',
      },
    },
    {
      type: 'remote_session.ended',
      session_id: 'sig_started',
      device_id: '123456789',
      data: {
        session_id: 'sig_started',
        device_id: '123456789',
        end_reason: 'disconnected',
      },
    },
  ])
})

test('scheduled task failure events trigger a task list refresh', () => {
  userSync.stop()
  let detail = null
  userSync.addEventListener(
    'scheduled-task-changed',
    (event) => { detail = event.detail },
    { once: true },
  )

  userSync._onMessage({
    data: JSON.stringify({
      type: 'scheduled_task.failed',
      server_rev: 4,
      data: {
        task_id: 12,
        kind: 'sign_out_all_sessions',
        attempts: 3,
        error: 'redis is down',
      },
    }),
  })

  assert.deepEqual(detail, {
    type: 'scheduled_task.failed',
    task_id: 12,
    kind: 'sign_out_all_sessions',
    attempts: 3,
    error: 'redis is down',
    data: {
      task_id: 12,
      kind: 'sign_out_all_sessions',
      attempts: 3,
      error: 'redis is down',
    },
  })
})
