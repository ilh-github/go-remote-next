import assert from 'node:assert/strict'
import test from 'node:test'

import { DataChannelHandler } from '../js/protocol/datachannel-handler.js'

// The actions channel encodes ActionRequest by hand: field 1 is the action
// enum, field 2 is a request id. Reading the bytes back is the only way to
// assert which action was actually requested.
function decodeActionRequest(buffer) {
  const bytes = new Uint8Array(buffer)
  let i = 0
  const fields = {}
  while (i < bytes.length) {
    let key = 0
    let shift = 0
    while (true) {
      const b = bytes[i++]
      key |= (b & 0x7f) << shift
      shift += 7
      if (!(b & 0x80)) break
    }
    const fieldNumber = key >> 3
    const wireType = key & 0x7
    if (wireType !== 0) throw new Error(`unexpected wire type ${wireType}`)
    let value = 0
    shift = 0
    while (true) {
      const b = bytes[i++]
      value |= (b & 0x7f) << shift
      shift += 7
      if (!(b & 0x80)) break
    }
    fields[fieldNumber] = value
  }
  return fields
}

// The host keeps its own authoritative record of which keys and buttons are
// down. A controller that goes away mid-hold leaves that record wrong: the
// next person to use the machine gets every keystroke modified. The Qt client
// guards this with releaseAllInput on focus loss and teardown; the web client
// has the same exposure and the same available action.
test('the client can ask the host to drop every key and button it is holding', () => {
  const sent = []
  const fake = {
    _actionsReady: true,
    _nextActionRequestId: 7,
    actionsChannel: { send: (buf) => sent.push(buf) },
    // Bind the real varint encoder rather than reimplementing it: a copy would
    // test the copy instead of the code that ships.
    _encodeVarint: DataChannelHandler.prototype._encodeVarint,
  }

  DataChannelHandler.prototype.sendAction.call(fake, 'releaseAllInput')

  assert.equal(sent.length, 1, 'nothing was sent on the actions channel')
  const decoded = decodeActionRequest(sent[0])
  // Field 1 is the action enum; 5 is releaseAllInput in the host descriptor.
  assert.equal(decoded[1], 5, 'the wrong action was requested')
  assert.equal(decoded[2], 7, 'the request id was not carried through')
})

// A host that supports only the release action still needs an actions channel,
// because that is the one action that stops stuck input from following the
// next person to the machine. Gating channel creation on the other three
// capabilities would make release unsendable on exactly the host that needs it.
test('a host advertising only releaseAllInput still gets an actions channel', () => {
  const created = []
  const fake = {
    _pc: {
      createDataChannel: (label) => {
        const channel = {
          label,
          binaryType: 'arraybuffer',
          onopen: null,
          onclose: null,
          onmessage: null,
        }
        created.push(channel)
        return channel
      },
    },
    dispatchEvent() {},
  }

  DataChannelHandler.prototype._onHostCapabilities.call(
    fake, 'releaseAllInputAction fileTransfer',
  )

  assert.equal(created.length, 1, 'no actions channel was created')
  assert.equal(created[0].label, 'actions')
  assert.equal(fake.supportsReleaseAllInput, true)
})

// The mirror case: a host that advertises nothing relevant must not get a
// channel, or every sendAction would fail on a closed one instead of being
// refused up front.
test('a host advertising no actions gets no actions channel', () => {
  const created = []
  const fake = {
    _pc: {
      createDataChannel: (label) => {
        created.push(label)
        return { label, onopen: null, onclose: null, onmessage: null }
      },
    },
    dispatchEvent() {},
  }

  DataChannelHandler.prototype._onHostCapabilities.call(fake, 'fileTransfer')

  assert.equal(created.length, 0, 'an actions channel was created for a host with no actions')
  assert.equal(fake.supportsReleaseAllInput, false)
})

// An unknown action must be refused rather than silently encoded as some
// other action: the wire format has no room for "unknown", and guessing a
// number would send a real instruction the operator never asked for.
test('an action outside the known set is refused, not encoded', () => {
  const sent = []
  const fake = {
    _actionsReady: true,
    _nextActionRequestId: 1,
    actionsChannel: { send: (buf) => sent.push(buf) },
    _encodeVarint: DataChannelHandler.prototype._encodeVarint,
  }

  DataChannelHandler.prototype.sendAction.call(fake, 'rebootThePlanet')

  assert.equal(sent.length, 0, 'an unknown action was written to the wire')
})
