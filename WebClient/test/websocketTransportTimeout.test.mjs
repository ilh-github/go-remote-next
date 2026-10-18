import assert from 'node:assert/strict'
import test from 'node:test'

import { WebSocketTransport } from '../js/signaling/websocket-transport.js'

class FakeWebSocket {
  static OPEN = 1

  constructor(url) {
    this.url = url
    this.readyState = 0
    this.sent = []
    queueMicrotask(() => {
      this.readyState = FakeWebSocket.OPEN
      this.onopen?.()
    })
  }

  send(data) {
    this.sent.push(String(data))
  }

  close() {
    this.readyState = 3
    this.onclose?.({ code: 1006, reason: 'closed by test' })
  }
}

test('auth_ok timeout is configurable and rejects stale pre-auth connections', async () => {
  const originalWebSocket = globalThis.WebSocket
  globalThis.WebSocket = FakeWebSocket
  try {
    const transport = new WebSocketTransport({
      signalingUrl: 'ws://127.0.0.1:1',
      authTimeoutMs: 5,
    })

    await assert.rejects(
      Promise.race([
        transport.connect({
          deviceId: 'device-1',
          signalToken: 'signal-token',
          clientId: 'client-1',
        }),
        new Promise((_, reject) => {
          setTimeout(() => reject(new Error('test timed out before auth failure')), 100)
        }),
      ]),
      /Auth timeout/,
    )
    assert.equal(transport.ws, null)
  } finally {
    globalThis.WebSocket = originalWebSocket
  }
})

test('pre-auth payloads and sends stay blocked until auth_ok', async () => {
  const originalWebSocket = globalThis.WebSocket
  globalThis.WebSocket = FakeWebSocket
  try {
    const received = []
    const transport = new WebSocketTransport({
      signalingUrl: 'ws://127.0.0.1:1',
      authTimeoutMs: 1000,
      onMessage: (message) => received.push(message),
    })
    const pending = transport.connect({
      deviceId: 'device-1',
      signalToken: 'signal-token',
      clientId: 'client-1',
    })
    await new Promise((resolve) => setTimeout(resolve, 0))

    transport.ws.onmessage({ data: '<iq><jingle/></iq>' })
    transport.ws.onmessage({ data: '{"payload":"<iq><jingle/></iq>"}' })
    assert.equal(transport.send('<iq><jingle/></iq>'), false)
    assert.deepEqual(received, [])
    assert.equal(transport.ws.sent.length, 1)

    transport.ws.onmessage({ data: '{"type":"auth_ok"}' })
    await pending
    transport.ws.onmessage({ data: '<iq><jingle/></iq>' })
    assert.deepEqual(received, ['<iq><jingle/></iq>'])
    assert.equal(transport.send('<iq><jingle/></iq>'), true)
    assert.equal(JSON.parse(transport.ws.sent.at(-1)).payload, '<iq><jingle/></iq>')

    transport.disconnect()
  } finally {
    globalThis.WebSocket = originalWebSocket
  }
})
