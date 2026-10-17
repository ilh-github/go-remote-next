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
