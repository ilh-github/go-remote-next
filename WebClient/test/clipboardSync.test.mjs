import assert from 'node:assert/strict'
import test from 'node:test'

// The clipboard poller is the upload half of the sync, and it has a loop to
// close: the handler records the last text it saw so that a text it just wrote
// locally is not sent straight back out. These tests pin that bookkeeping, which
// is where the loop either closes or runs away.

globalThis.localStorage = {
  _values: new Map(),
  getItem(key) { return this._values.has(key) ? this._values.get(key) : null },
  setItem(key, value) { this._values.set(key, String(value)) },
  removeItem(key) { this._values.delete(key) },
}

globalThis.document = { hasFocus: () => true }

const { ClipboardHandler } = await import('../js/input/clipboard-handler.js')

class FakeDataChannel {
  constructor() {
    this.sent = []
    this.listeners = {}
  }
  addEventListener(type, fn) { (this.listeners[type] ||= []).push(fn) }
  sendClipboardEvent(event) { this.sent.push(event) }
  emit(type, detail) { for (const fn of this.listeners[type] || []) fn({ detail }) }
}

function makeHandler(overrides = {}) {
  const reported = []
  return {
    reported,
    handler: new ClipboardHandler(new FakeDataChannel(), {
      deviceId: '123456789',
      now: () => 1700000000000,
      reportAudit: (body) => { reported.push(body); return Promise.resolve({}) },
      ...overrides,
    }),
  }
}

function setClipboard({ readText, writeText }) {
  Object.defineProperty(globalThis, 'navigator', {
    value: { clipboard: { readText, writeText } },
    configurable: true,
    writable: true,
  })
}

// The poller must not send a text straight back to the machine it came from.
// Without the loop guard, a host that echoes its own clipboard would trade the
// same text with the controller forever, once a second, for as long as the
// session is open.
test('a text received from the host is not sent straight back', async () => {
  setClipboard({ readText: async () => 'from host', writeText: async () => {} })
  const { handler } = makeHandler()

  await handler._onRemoteClipboard({
    detail: { mimeType: 'text/plain', data: new TextEncoder().encode('from host') },
  })

  await handler._pollClipboard()

  assert.equal(handler.dcHandler.sent.length, 0, 'the host\'s own text was echoed back to it')
})

// The mirror case: a text the controller copied locally must go out exactly
// once, not once per poll.
test('a locally copied text is sent once, not once per poll', async () => {
  setClipboard({ readText: async () => 'copied here', writeText: async () => {} })
  const { handler } = makeHandler()

  await handler._pollClipboard()
  await handler._pollClipboard()
  await handler._pollClipboard()

  assert.equal(handler.dcHandler.sent.length, 1, 'the same text was sent on every poll')
})

// When the local write fails the clipboard still holds the old text, so the
// next poll sees a difference and sends it again. Marking the text as seen has
// to happen whether or not the write landed, or one rejected write turns into
// an upload every second for the rest of the session.
test('a rejected local write does not restart the upload loop', async () => {
  setClipboard({
    readText: async () => 'from host',
    writeText: async () => { throw new Error('denied') },
  })
  const { handler } = makeHandler()

  await handler._onRemoteClipboard({
    detail: { mimeType: 'text/plain', data: new TextEncoder().encode('from host') },
  })

  await handler._pollClipboard()
  await handler._pollClipboard()

  assert.equal(handler.dcHandler.sent.length, 0, 'a failed write left the loop open')
})

// A background tab must not read the clipboard: the browser refuses it, and
// retrying every second on a hidden tab is noise the operator cannot see.
test('a poll while the document is unfocused does not read the clipboard', async () => {
  let reads = 0
  setClipboard({ readText: async () => { reads += 1; return 'text' }, writeText: async () => {} })
  const { handler } = makeHandler()
  globalThis.document = { hasFocus: () => false }

  await handler._pollClipboard()

  assert.equal(reads, 0, 'the clipboard was read from an unfocused document')
})
