import assert from 'node:assert/strict'
import test from 'node:test'

import { DataChannelHandler } from '../js/protocol/datachannel-handler.js'

// The actions channel is what carries releaseAllInput, and it is closed when the
// session is torn down. These tests exercise the real handler rather than a
// stand-in, because the question is whether the product's own close path leaves
// the release sendable.

function makeHandler() {
  const sent = []
  const handler = new DataChannelHandler()
  handler._actionsReady = true
  handler._nextActionRequestId = 1
  handler.actionsChannel = {
    send: (buffer) => sent.push(new Uint8Array(buffer)),
    // The real close path also clears the ready flag, which is what makes a later
    // send a no-op rather than a write to a dead channel.
    close: () => {
      handler.actionsChannel = null
      handler._actionsReady = false
    },
  }
  handler._encodeVarint = DataChannelHandler.prototype._encodeVarint
  return { handler, sent }
}

test('a release can be sent on a live actions channel', () => {
  const { handler, sent } = makeHandler()

  handler.sendAction('releaseAllInput')

  assert.equal(sent.length, 1, 'nothing reached the actions channel')
  // Field 1 (action) = 5, field 2 (request id) = 1.
  assert.deepEqual([...sent[0]], [0x08, 0x05, 0x10, 0x01])
})

// The hazard this pins is a send that reaches a channel object which has already
// been closed underneath it. `_actionsReady` alone does not cover that: the flag
// is cleared by destroy(), but a channel can also be closed by the peer or by the
// browser without destroy() running, and a write to a closed DataChannel throws
// rather than being dropped. Callers of sendAction wrap it in try/catch today,
// which means the failure is invisible - the release is simply lost and the host
// keeps the held keys.
test('a release on a channel whose ready flag is stale does not throw', () => {
  const { handler, sent } = makeHandler()

  // The channel is gone but the flag still says ready: the shape a peer-initiated
  // close leaves behind.
  handler.actionsChannel = null

  assert.doesNotThrow(() => handler.sendAction('releaseAllInput'),
    'a release on a missing channel threw instead of being refused')
  assert.equal(sent.length, 0, 'a release was written to a channel that is not there')
})

// Once the channels are closed the release cannot be sent at all, which is the
// reason the caller has to send it *before* tearing the session down rather than
// after. A send that is silently dropped looks identical to a successful one from
// the caller's side, so this is the boundary worth pinning.
test('a release cannot be sent once the channels are closed', () => {
  const { handler, sent } = makeHandler()

  handler.destroy()
  handler.sendAction('releaseAllInput')

  assert.equal(sent.length, 0, 'a release was written to a closed channel')
})
