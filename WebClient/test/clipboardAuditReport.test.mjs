import assert from 'node:assert/strict'
import test from 'node:test'

// QD-909, handler half: the ClipboardHandler must report clipboard syncs to the
// control plane, and the report must never contain the text. The model test
// pins the builder; this pins that the handler actually calls it on both
// directions and swallows a failing report instead of breaking the sync.

const reported = []
globalThis.localStorage = {
    _values: new Map(),
    getItem(key) { return this._values.has(key) ? this._values.get(key) : null },
    setItem(key, value) { this._values.set(key, String(value)) },
    removeItem(key) { this._values.delete(key) },
}
const { ClipboardHandler } = await import('../js/input/clipboard-handler.js')

// `navigator` is a getter-only global in Node, so the clipboard is swapped the
// only way that works rather than by assignment.
function setClipboard(writeText) {
    Object.defineProperty(globalThis, 'navigator', {
        value: { clipboard: { writeText } },
        configurable: true,
        writable: true,
    })
}

// A minimal DataChannelHandler stand-in: the handler under test only dispatches
// clipboard events and reads the payload it sent.
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
    reported.length = 0
    return new ClipboardHandler(new FakeDataChannel(), {
        deviceId: '123456789',
        now: () => 1700000000000,
        reportAudit: (body) => { reported.push(body); return Promise.resolve({ ok: true }) },
        ...overrides,
    })
}

test('sending to the remote reports an upload without the text', () => {
    const handler = makeHandler()
    const secret = 'xoxb-SUPER-SECRET-SLACK-TOKEN'
    handler._sendToRemote(secret)

    assert.equal(reported.length, 1)
    const report = reported[0]
    assert.equal(report.direction, 'upload')
    assert.equal(report.status, 'completed')
    assert.equal(report.deviceId, '123456789')
    assert.equal(report.mimeType, 'text/plain')
    assert.equal(report.bytes, new TextEncoder().encode(secret).length)
    assert.equal(JSON.stringify(report).includes(secret), false)
})

test('receiving from the remote reports a download', async () => {
    const handler = makeHandler()
    const text = 'password-reset-code-4471'
    setClipboard(async () => {})

    await handler._onRemoteClipboard({
        detail: { mimeType: 'text/plain', data: new TextEncoder().encode(text) },
    })

    assert.equal(reported.length, 1)
    assert.equal(reported[0].direction, 'download')
    assert.equal(reported[0].status, 'completed')
    assert.equal(JSON.stringify(reported[0]).includes(text), false)
})

test('a failed local write is still reported, as failed', async () => {
    const handler = makeHandler()
    setClipboard(async () => { throw new Error('denied') })

    await handler._onRemoteClipboard({
        detail: { mimeType: 'text/plain', data: new TextEncoder().encode('anything') },
    })

    assert.equal(reported.length, 1)
    assert.equal(reported[0].status, 'failed')
    assert.equal(reported[0].direction, 'download')
})

// Without a device id there is nothing to attribute the sync to, so the handler
// stays quiet rather than sending a report the server would reject.
test('no device id means no report', () => {
    const handler = makeHandler({ deviceId: '' })
    handler._sendToRemote('some text')
    assert.equal(reported.length, 0)
})

// A burst of syncs must be reported as the distinct transfers they are; the
// server keys idempotency on the transfer id, so reusing one would collapse them.
test('each sync gets its own transfer id', () => {
    const handler = makeHandler()
    handler._sendToRemote('first')
    handler._sendToRemote('second')
    handler._sendToRemote('third')

    assert.equal(reported.length, 3)
    const ids = new Set(reported.map((r) => r.transferId))
    assert.equal(ids.size, 3, 'transfer ids must be distinct')
})

// The audit report must never break the sync the user is actually performing.
test('a failing report does not break the sync', () => {
    const handler = makeHandler({
        reportAudit: () => Promise.reject(new Error('offline')),
    })
    assert.doesNotThrow(() => handler._sendToRemote('text that must still be sent'))
    assert.equal(handler.dcHandler.sent.length, 1, 'the clipboard event must still be sent')
})
