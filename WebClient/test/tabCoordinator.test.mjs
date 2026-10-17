import assert from 'node:assert/strict'
import test from 'node:test'

import { TabCoordinator } from '../js/ui/tab-coordinator.js'

class FakeBroadcastChannel {
  static channels = new Map()

  constructor(name) {
    this.name = name
    this.onmessage = null
    this.closed = false
    const peers = FakeBroadcastChannel.channels.get(name) || new Set()
    peers.add(this)
    FakeBroadcastChannel.channels.set(name, peers)
  }

  postMessage(data) {
    const peers = FakeBroadcastChannel.channels.get(this.name) || new Set()
    for (const peer of peers) {
      if (peer !== this && !peer.closed) {
        queueMicrotask(() => peer.onmessage?.({ data }))
      }
    }
  }

  close() {
    this.closed = true
    FakeBroadcastChannel.channels.get(this.name)?.delete(this)
  }
}

function resetChannels() {
  FakeBroadcastChannel.channels.clear()
}

function createCoordinator(tabId, now = () => 1000) {
  return new TabCoordinator({
    deviceId: 'device-a',
    tabId,
    now,
    BroadcastChannelImpl: FakeBroadcastChannel,
  })
}

test('a newer control claim revokes control in the previous tab', async () => {
  resetChannels()
  const first = createCoordinator('tab-a')
  const second = createCoordinator('tab-b')
  first.start()
  second.start()
  first.claim('control')

  const revoked = new Promise((resolve) => {
    first.addEventListener('revoked', (event) => resolve(event.detail), { once: true })
  })
  second.claim('control')

  assert.deepEqual(await revoked, {
    resource: 'control',
    ownerTabId: 'tab-b',
  })
  assert.equal(first.hasClaim('control'), false)
  assert.equal(second.hasClaim('control'), true)

  first.destroy()
  second.destroy()
})

test('an older claim cannot take control from the current tab', async () => {
  resetChannels()
  let now = 2000
  const first = createCoordinator('tab-a', () => now)
  const second = createCoordinator('tab-b', () => now)
  first.start()
  second.start()
  first.claim('control')
  now = 3000
  second.claim('control')

  await new Promise((resolve) => setTimeout(resolve, 0))

  assert.equal(first.hasClaim('control'), false)
  assert.equal(second.hasClaim('control'), true)

  first.destroy()
  second.destroy()
})

test('audio claims only revoke audio, not control', async () => {
  resetChannels()
  const first = createCoordinator('tab-a')
  const second = createCoordinator('tab-b')
  first.start()
  second.start()
  first.claim('control')
  first.claim('audio')

  const revoked = new Promise((resolve) => {
    first.addEventListener('revoked', (event) => resolve(event.detail), { once: true })
  })
  second.claim('audio')

  assert.deepEqual(await revoked, {
    resource: 'audio',
    ownerTabId: 'tab-b',
  })
  assert.equal(first.hasClaim('control'), true)
  assert.equal(first.hasClaim('audio'), false)

  first.destroy()
  second.destroy()
})

test('a released resource can be claimed without revoking the old owner', async () => {
  resetChannels()
  const first = createCoordinator('tab-a')
  const second = createCoordinator('tab-b')
  first.start()
  second.start()
  first.claim('control')
  first.release('control')

  let revoked = false
  first.addEventListener('revoked', () => { revoked = true }, { once: true })
  second.claim('control')

  await new Promise((resolve) => setTimeout(resolve, 0))

  assert.equal(revoked, false)
  assert.equal(second.hasClaim('control'), true)

  first.destroy()
  second.destroy()
})

test('storage events act as a BroadcastChannel fallback', () => {
  const listeners = new Map()
  const target = {
    addEventListener(type, listener) {
      listeners.set(type, listener)
    },
    removeEventListener(type) {
      listeners.delete(type)
    },
  }
  const storage = {
    _value: '',
    setItem(_key, value) {
      this._value = value
    },
    getItem() {
      return this._value
    },
  }

  const coordinator = new TabCoordinator({
    deviceId: 'device-a',
    tabId: 'tab-a',
    BroadcastChannelImpl: null,
    now: () => 1000,
    storage,
    storageEventTarget: target,
  })
  coordinator.start()
  coordinator.claim('control')

  let revoked = null
  coordinator.addEventListener('revoked', (event) => { revoked = event.detail }, { once: true })

  const message = JSON.parse(storage.getItem('quickdesk:tab-bus'))
  listeners.get('storage')({
    key: 'quickdesk:tab-bus',
    newValue: JSON.stringify({
      ...message,
      type: 'claim',
      tabId: 'tab-b',
      at: 1001,
    }),
  })

  assert.deepEqual(revoked, {
    resource: 'control',
    ownerTabId: 'tab-b',
  })

  coordinator.destroy()
})

test('storage fallback is used when BroadcastChannel construction is blocked', () => {
  const listeners = new Map()
  const target = {
    addEventListener(type, listener) {
      listeners.set(type, listener)
    },
    removeEventListener(type) {
      listeners.delete(type)
    },
  }
  const storage = {
    _value: '',
    setItem(_key, value) {
      this._value = value
    },
    getItem() {
      return this._value
    },
  }
  class BlockedBroadcastChannel {
    constructor() {
      throw new Error('blocked by browser policy')
    }
  }

  const coordinator = new TabCoordinator({
    deviceId: 'device-a',
    tabId: 'tab-a',
    now: () => 1000,
    BroadcastChannelImpl: BlockedBroadcastChannel,
    storage,
    storageEventTarget: target,
  })
  coordinator.start()
  coordinator.claim('control')

  assert.equal(JSON.parse(storage.getItem('quickdesk:tab-bus')).type, 'claim')
  assert.equal(listeners.has('storage'), true)

  coordinator.destroy()
})
