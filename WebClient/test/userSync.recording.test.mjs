import assert from 'node:assert/strict'
import test from 'node:test'

globalThis.localStorage = {
  _values: new Map(),
  getItem(key) { return this._values.has(key) ? this._values.get(key) : null },
  setItem(key, value) { this._values.set(String(key), String(value)) },
  removeItem(key) { this._values.delete(key) },
}

const { userSync } = await import('../src/api/userSync.js')

// The server publishes recording.created / updated / deleted, and the WebClient
// has a RecordingsPage - which refreshes only in onMounted. So a recording that
// another controller uploads, finishes or deletes while the page is open never
// appears until the operator reloads. The events reach the socket and stop
// there, which is the same shape as the connect-prompt gap three rounds ago.
//
// Found by diffing the server's event list against userSync's dispatch switch
// rather than by reading the page, which is why it survived this long: the page
// looks complete until somebody else touches a recording.

// All three lifecycle events have to surface, because the page has to react to
// each of them differently: a new row appears, a row's status changes, a row
// disappears.
for (const type of ['recording.created', 'recording.updated', 'recording.deleted']) {
  test(`${type} surfaces as a client event`, () => {
    userSync.stop()
    let detail = null
    userSync.addEventListener('recording-changed', (event) => { detail = event.detail }, { once: true })

    userSync._onMessage({
      data: JSON.stringify({
        type,
        server_rev: 1,
        data: { recording_id: 4242, device_id: '700000001' },
      }),
    })

    assert.ok(detail, `no recording-changed event for ${type}`)
    assert.equal(detail.type, type)
    assert.equal(detail.recording_id, 4242)
  })
}

// A stale frame must be dropped like every other type: a reconnect replays the
// backlog, and re-applying it would re-toast a deletion that already happened.
test('a stale recording frame is dropped by the rev guard', () => {
  userSync.stop()
  let seen = 0
  userSync.addEventListener('recording-changed', () => { seen += 1 })

  userSync._onMessage({
    data: JSON.stringify({ type: 'recording.deleted', server_rev: 10, data: { recording_id: 1 } }),
  })
  userSync._onMessage({
    data: JSON.stringify({ type: 'recording.deleted', server_rev: 4, data: { recording_id: 2 } }),
  })

  assert.equal(seen, 1, `expected only the newer frame, got ${seen}`)
})

// A frame with no recording id is not actionable: the page could not tell which
// row changed, so refreshing on it would be a guess.
test('a recording frame with no id is not dispatched', () => {
  userSync.stop()
  let seen = 0
  userSync.addEventListener('recording-changed', () => { seen += 1 })

  userSync._onMessage({
    data: JSON.stringify({ type: 'recording.updated', server_rev: 1, data: {} }),
  })

  assert.equal(seen, 0, 'a frame with no recording id was dispatched anyway')
})
