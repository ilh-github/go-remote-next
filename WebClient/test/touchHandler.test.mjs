import assert from 'node:assert/strict'
import test from 'node:test'

import { TouchHandler } from '../js/input/touch-handler.js'
import { MouseButton } from '../js/protocol/protobuf-messages.js'

class FakeElement {
  constructor() {
    this.listeners = new Map()
    this.style = {}
    this.removed = false
    this.children = []
    this.clientWidth = 400
    this.clientHeight = 800
  }

  remove() { this.removed = true }

  addEventListener(type, fn, opts) {
    if (!this.listeners.has(type)) this.listeners.set(type, [])
    this.listeners.get(type).push({ fn, opts })
  }

  removeEventListener(type, fn) {
    const list = this.listeners.get(type) || []
    const idx = list.findIndex((entry) => entry.fn === fn)
    if (idx >= 0) list.splice(idx, 1)
  }

  emit(type, event = {}) {
    for (const entry of (this.listeners.get(type) || []).slice()) {
      entry.fn({ preventDefault() {}, ...event })
    }
  }

  getBoundingClientRect() {
    return { left: 0, top: 0, width: this.clientWidth || 400, height: this.clientHeight || 800 }
  }

  get parentElement() { return this._parent || null }
  set parentElement(el) { this._parent = el }

  getAttribute() { return null }
  setAttribute() {}
  appendChild(child) {
    this.children.push(child)
    child.parentElement = this
  }
  removeChild(child) {
    const i = this.children.indexOf(child)
    if (i >= 0) this.children.splice(i, 1)
  }
}

// The touch handler reads `document.activeElement` to dismiss the virtual
// keyboard before a gesture, and creates a cursor element to show where the
// remote pointer is. Both need a document, so provide one up front.
globalThis.document = {
  activeElement: null,
  createElement: (tag) => {
    const el = new FakeElement()
    el.tagName = tag.toUpperCase()
    return el
  },
  body: new FakeElement(),
}
function makeHandler({ remoteWidth = 1920, remoteHeight = 1080 } = {}) {
  const target = new FakeElement()
  const events = []
  const dcHandler = {
    sendMouseEvent(e) { events.push({ kind: 'mouse', ...e }) },
  }
  // The handler renders its cursor dot into the video's parent, so the video
  // fake needs a container to hang it on.
  const container = new FakeElement()
  target.parentElement = container

  const handler = new TouchHandler(target, target, dcHandler)
  handler.setRemoteResolution(remoteWidth, remoteHeight)
  handler.enable()
  return { target, events, handler }
}

function touch(id, x, y) {
  return { identifier: id, clientX: x, clientY: y }
}

function touches(...list) {
  return { length: list.length, ...list, item: (i) => list[i] }
}

// A touch that starts a drag and then gets cancelled by the OS (an incoming
// call, a notification, the browser taking the gesture over) must release the
// button. The remote machine has no idea the touch is gone.
//
// The drag gesture here is the platform-standard one: tap once to click, then
// hold and move on a second tap. The first tap already produced the click, so
// the second tap's movement is what presses the button.
test('a cancelled touch releases the button it was holding', () => {
  const { target, events, handler } = makeHandler()

  // First tap: click.
  target.emit('touchstart', { touches: touches(touch(1, 200, 400)) })
  target.emit('touchend', { touches: touches() })

  // Second tap, held and moved: this presses the button.
  target.emit('touchstart', { touches: touches(touch(1, 200, 400)) })
  target.emit('touchmove', { touches: touches(touch(1, 260, 460)) })

  // The click produced a press/release pair; the drag then pressed the button
  // and is still holding it. What matters is that a press with no matching
  // release is what the cancel has to clean up.
  const presses = events.filter((e) => e.button === MouseButton.BUTTON_LEFT && e.buttonDown === true)
  const releases = events.filter((e) => e.button === MouseButton.BUTTON_LEFT && e.buttonDown === false)
  assert.equal(presses.length, releases.length + 1, 'the drag did not leave a held button')

  target.emit('touchcancel', { touches: touches() })

  const released = events.filter((e) => e.button === MouseButton.BUTTON_LEFT && e.buttonDown === false)
  assert.equal(released.length, presses.length, 'a cancelled drag left the button stuck down on the host')

  handler.destroy()
})

// Lifting one finger of a two-finger pinch must not leave a half-finished
// gesture behind: the remaining finger becomes a new single-finger gesture, and
// if the user then drags, the host should not see a phantom press.
test('lifting one finger of a pinch starts a clean single-finger gesture', () => {
  const { target, events, handler } = makeHandler()

  target.emit('touchstart', { touches: touches(touch(1, 200, 400), touch(2, 220, 400)) })
  target.emit('touchmove', { touches: touches(touch(1, 190, 390), touch(2, 230, 410)) })

  const beforeLift = events.filter((e) => e.button === MouseButton.BUTTON_LEFT).length

  target.emit('touchend', { touches: touches(touch(2, 230, 410)) })

  // The surviving finger continues. Whatever happens next must not produce a
  // stray click: the user lifted a finger, they did not tap.
  target.emit('touchend', { touches: touches() })

  const afterLift = events.filter((e) => e.button === MouseButton.BUTTON_LEFT)
  assert.equal(afterLift.length, beforeLift, 'lifting a pinch finger produced a phantom click')
  assert.ok(afterLift.every((e) => e.buttonDown === false), 'a phantom press was left down')

  handler.destroy()
})

// Disposing the handler mid-drag is the same stuck-button problem: the touch
// that was holding the button is gone along with the element.
test('destroying mid-drag releases the held button', () => {
  const { target, events, handler } = makeHandler()

  target.emit('touchstart', { touches: touches(touch(1, 200, 400)) })
  target.emit('touchend', { touches: touches() })

  target.emit('touchstart', { touches: touches(touch(1, 200, 400)) })
  target.emit('touchmove', { touches: touches(touch(1, 260, 460)) })

  // The click produced its own press/release; the drag is still holding one.
  const held = events.filter((e) => e.buttonDown === true).length -
               events.filter((e) => e.buttonDown === false).length
  assert.equal(held, 1)

  handler.destroy()

  assert.equal(events.filter((e) => e.buttonDown === false).length,
    events.filter((e) => e.buttonDown === true).length,
    'a held button survived the handler teardown')
})

// The cursor element is what shows the finger where the remote pointer is.
// Leaving it behind after teardown pins a stale dot over a dead session.
test('teardown removes the on-screen cursor element', () => {
  const created = []
  globalThis.document = {
    activeElement: null,
    createElement: (tag) => {
      const el = new FakeElement()
      el.tagName = tag.toUpperCase()
      created.push(el)
      return el
    },
    body: new FakeElement(),
  }

  const { target, handler } = makeHandler()
  assert.ok(created.length > 0, 'no cursor element was created')

  handler.destroy()

  const cursor = created.find((el) => el.tagName === 'DIV')
  assert.equal(cursor.removed, true, 'the cursor element was never removed from the page')
})
