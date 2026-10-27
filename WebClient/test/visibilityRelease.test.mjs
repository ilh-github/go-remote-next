import assert from 'node:assert/strict'
import test from 'node:test'

import { shouldReleaseFor, watchForRelease } from '../js/input/visibilityRelease.js'

// Switching tabs, switching applications, minimising and locking the screen all
// leave the host holding whatever was down, and none of them produce a blur on
// the element that receives keys - which is the only thing the client used to
// listen for.
test('a tab switch releases the held keys', () => {
  assert.equal(shouldReleaseFor('visibilitychange', { hidden: true }), true)
})

test('the window losing focus releases the held keys', () => {
  assert.equal(shouldReleaseFor('blur', {}), true)
})

test('the page going away releases the held keys', () => {
  assert.equal(shouldReleaseFor('pagehide', {}), true)
})

// Becoming visible again must not release: the operator has come back and the
// keys they were holding are keys they still mean to hold.
test('becoming visible again does not release', () => {
  assert.equal(shouldReleaseFor('visibilitychange', { hidden: false }), false)
})

test('an unrelated event does not release', () => {
  assert.equal(shouldReleaseFor('resize', {}), false)
  assert.equal(shouldReleaseFor('scroll', {}), false)
  assert.equal(shouldReleaseFor('focus', {}), false)
})

// The triggers must actually be attached, and detached. A watcher that is never
// detached keeps firing after the session is over, which is the same leak the
// pointer-capture fix addressed from the other side.
test('the watcher fires for the triggers and can be detached', () => {
  const listeners = new Map()
  const scope = {
    hidden: false,
    addEventListener(type, fn) {
      if (!listeners.has(type)) listeners.set(type, [])
      listeners.get(type).push(fn)
    },
    removeEventListener(type, fn) {
      const list = listeners.get(type) || []
      const i = list.indexOf(fn)
      if (i >= 0) list.splice(i, 1)
    },
  }
  const target = { document: scope, addEventListener: scope.addEventListener, removeEventListener: scope.removeEventListener }

  let releases = 0
  const detach = watchForRelease(target, () => { releases += 1 })

  // The document-scoped visibilitychange and the window-scoped blur/pagehide.
  assert.equal(listeners.has('visibilitychange'), true, 'no visibility listener was attached')
  assert.equal(listeners.has('blur'), true, 'no window blur listener was attached')
  assert.equal(listeners.has('pagehide'), true, 'no pagehide listener was attached')

  scope.hidden = true
  for (const fn of listeners.get('visibilitychange')) fn()
  assert.equal(releases, 1, 'hiding the page did not release')

  for (const fn of listeners.get('blur')) fn()
  assert.equal(releases, 2, 'losing window focus did not release')

  detach()
  for (const fn of (listeners.get('blur') || [])) fn()
  assert.equal(releases, 2, 'a detached watcher still fired')
})

// A visible document must not release on the visibilitychange event itself.
test('a visibilitychange while still visible does not release', () => {
  const listeners = new Map()
  const scope = {
    hidden: false,
    addEventListener(type, fn) {
      if (!listeners.has(type)) listeners.set(type, [])
      listeners.get(type).push(fn)
    },
    removeEventListener() {},
  }
  const target = { document: scope, addEventListener: scope.addEventListener, removeEventListener: scope.removeEventListener }

  let releases = 0
  watchForRelease(target, () => { releases += 1 })

  for (const fn of listeners.get('visibilitychange')) fn()
  assert.equal(releases, 0, 'a visibilitychange with hidden=false released anyway')
})

// A target with no addEventListener must not throw: the watcher is attached from
// a code path that also runs in non-browser contexts during tests.
test('a target without addEventListener is tolerated', () => {
  assert.doesNotThrow(() => watchForRelease(null, () => {}))
  assert.doesNotThrow(() => watchForRelease({}, () => {}))
})
