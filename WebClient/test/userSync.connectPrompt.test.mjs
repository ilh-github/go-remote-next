import assert from 'node:assert/strict'
import test from 'node:test'

globalThis.localStorage = {
  _values: new Map(),
  getItem(key) { return this._values.has(key) ? this._values.get(key) : null },
  setItem(key, value) { this._values.set(String(key), String(value)) },
  removeItem(key) { this._values.delete(key) },
}

const { userSync } = await import('../src/api/userSync.js')

// QD-607's connect_prompt was wired on the server and labelled in the client,
// but nothing consumed it: userSync's dispatch switch had no branch, so the
// event arrived on the socket and stopped there. The prompt is the whole feature
// - a task that fires and tells nobody is the half-wired version again, which is
// the exact failure the previous round found on the notification subscription.

// The event has to surface as a client event a page can listen for.
test('a connect prompt surfaces as a client event', () => {
  userSync.stop()
  let detail = null
  userSync.addEventListener('connect-prompted', (event) => { detail = event.detail }, { once: true })

  userSync._onMessage({
    data: JSON.stringify({
      type: 'device.connect_prompted',
      server_rev: 1,
      data: { task_id: 42 },
    }),
  })

  assert.ok(detail, 'no connect-prompted event was dispatched')
  assert.equal(detail.task_id, 42)
})

// A stale frame must be dropped, the same as every other event type: a prompt
// replayed after the owner already acted on it is noise, and the rev guard is
// what keeps a reconnect from re-delivering the whole backlog.
test('a stale prompt is dropped by the rev guard', () => {
  userSync.stop()
  let seen = 0
  userSync.addEventListener('connect-prompted', () => { seen += 1 })

  // Prime the guard with a higher rev first.
  userSync._onMessage({
    data: JSON.stringify({ type: 'device.connect_prompted', server_rev: 10, data: { task_id: 1 } }),
  })
  userSync._onMessage({
    data: JSON.stringify({ type: 'device.connect_prompted', server_rev: 3, data: { task_id: 2 } }),
  })

  assert.equal(seen, 1, `expected only the newer frame, got ${seen}`)
})

// A frame with no task id is not actionable: a prompt that cannot name the task
// it came from leaves the owner with nothing to open.
test('a prompt with no task id is not dispatched', () => {
  userSync.stop()
  let seen = 0
  userSync.addEventListener('connect-prompted', () => { seen += 1 })

  userSync._onMessage({
    data: JSON.stringify({ type: 'device.connect_prompted', server_rev: 1, data: {} }),
  })

  assert.equal(seen, 0, 'a prompt with no task id was dispatched anyway')
})
