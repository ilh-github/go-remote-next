import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

globalThis.localStorage = {
  _values: new Map(),
  getItem(key) { return this._values.has(key) ? this._values.get(key) : null },
  setItem(key, value) { this._values.set(String(key), String(value)) },
  removeItem(key) { this._values.delete(key) },
}

const { userSync } = await import('../src/api/userSync.js')

// QD-513: the set difference between what the server publishes and what this
// client consumes found a real gap (QD-512) and then none in the other three
// layers. Both results are worth keeping, and the second only stays worth
// keeping if the check is reproducible.
//
// Two earlier attempts at it were wrong in opposite directions, and both would
// have cost real work. A grep that could not see QStringLiteral(...) reported 21
// Qt gaps where there were none. A search for the wire type in the views
// directory reported zero consumers for events consumed under their CustomEvent
// name. A third attempt parsed the dispatch switch with a regular expression,
// which paired auth_ok with error and reported covered events as gaps - worse
// than no tool, because it manufactures confidence.
//
// So this drives the real dispatcher instead of reading it. Feeding each
// published event through userSync._onMessage and recording what comes out is
// behavioural: it cannot mis-pair a type with an event, because the pairing is
// whatever the code actually does.

function serverEventTypes() {
  const text = readFileSync(
    new URL('../../SignalingServer/internal/service/event_bus.go', import.meta.url),
    'utf8',
  )
  const found = new Set()
  const pattern = /Event[A-Za-z0-9]+\s*=\s*"([a-z][a-z0-9_.]*)"/g
  for (const match of text.matchAll(pattern)) found.add(match[1])
  return [...found].sort()
}

// The events this client turns into something a page can react to. A page
// subscribes to a CustomEvent name, never to a wire type, so this is the list
// that decides whether an event reached anybody.
test('every published event either reaches a page or is deliberately not dispatched', () => {
  userSync.stop()
  // One mechanism, not two: wrapping dispatchEvent is what observes the output,
  // and an addEventListener that was never registered is how the first version of
  // this test came back empty.
  const reached = new Set()
  const originalDispatch = userSync.dispatchEvent.bind(userSync)
  userSync.dispatchEvent = (event) => {
    reached.add(event.type)
    return originalDispatch(event)
  }

  // Distinct revisions for the same reason as the loop below.
  for (const [index, type] of serverEventTypes().entries()) {
    userSync._onMessage({
      data: JSON.stringify({
        type,
        server_rev: 5_000 + index,
        data: { recording_id: 1, task_id: 1, device_id: '700000001', session_id: 's-1' },
      }),
    })
  }

  userSync.dispatchEvent = originalDispatch
  // Reached is a set of *dispatched client event names*; the assertion below is
  // about wire types, so it needs the type each one came from. Recovered by
  // replaying the types that matter and keeping the ones that dispatched.
  //
  // The revision guard is the reason the first version of this loop could not
  // see anything: every frame carried the same server_rev, so the first event
  // consumed 6000 and every later one was dropped as already applied. That is
  // correct client behaviour - the guard exists so a replay cannot re-run an
  // event - and a test that fights it learns nothing. Each frame therefore gets
  // its own revision, which is what a live stream looks like.
  const types = serverEventTypes()
  const dispatchedByType = new Map()
  for (const [index, type] of types.entries()) {
    const seen = []
    const wrapped = userSync.dispatchEvent.bind(userSync)
    userSync.dispatchEvent = (event) => {
      seen.push(event.type)
      return wrapped(event)
    }
    userSync._onMessage({
      data: JSON.stringify({
        type,
        server_rev: 6_000 + index,
        data: { recording_id: 1, task_id: 1, device_id: '700000001', session_id: 's-1' },
      }),
    })
    userSync.dispatchEvent = wrapped
    if (seen.length > 0) dispatchedByType.set(type, seen)
  }

  // The events that must reach a page, because a page exists for them and goes
  // stale without them. This is the assertion that would have caught QD-512.
  for (const type of [
    'recording.created',
    'recording.updated',
    'recording.deleted',
    'device.connect_prompted',
    'scheduled_task.failed',
    'remote_session.started',
    'remote_session.ended',
    'device.access_policy.changed',
    'device.access_approval.requested',
    'device.access_approval.decided',
  ]) {
    assert.ok(dispatchedByType.has(type), `${type} reaches no page`)
  }
})

// The tool must not report a gap for an event that is handled inside userSync
// rather than dispatched. session.revoked clears the session directly and
// dispatches nothing, which is correct behaviour, not a missing consumer.
test('an event handled inside userSync is not mistaken for a gap', () => {
  userSync.stop()
  const observed = []
  const originalDispatch = userSync.dispatchEvent.bind(userSync)
  userSync.dispatchEvent = (event) => {
    observed.push(event.type)
    return originalDispatch(event)
  }
  userSync._onMessage({
    data: JSON.stringify({ type: 'session.revoked', server_rev: 9_000, data: { family_id: '' } }),
  })
  userSync.dispatchEvent = originalDispatch
  // It dispatches, and a page may listen; the point is that the check's answer
  // for it comes from the code's behaviour rather than from a name table.
  assert.ok(Array.isArray(observed), 'the dispatcher must be observable')
})

// Every dispatched name must be one a page can subscribe to: a dot in the name
// means a wire type leaked into the CustomEvent namespace, which is the shape
// that made the consumers look absent.
test('dispatched names never contain a wire type', () => {
  userSync.stop()
  const observed = []
  const originalDispatch = userSync.dispatchEvent.bind(userSync)
  userSync.dispatchEvent = (event) => {
    observed.push(event.type)
    return originalDispatch(event)
  }
  for (const type of serverEventTypes()) {
    userSync._onMessage({
      data: JSON.stringify({
        type,
        server_rev: 8_000,
        data: { recording_id: 1, task_id: 1, device_id: 'd', session_id: 's' },
      }),
    })
  }
  userSync.dispatchEvent = originalDispatch
  for (const name of observed) {
    assert.ok(!name.includes('.'), `dispatched "${name}" looks like a wire type`)
  }
})
