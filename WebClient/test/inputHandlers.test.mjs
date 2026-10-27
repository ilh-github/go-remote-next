import assert from 'node:assert/strict'
import test from 'node:test'

import { MouseHandler } from '../js/input/mouse-handler.js'
import { KeyboardHandler } from '../js/input/keyboard-handler.js'

// The handlers reach for `document.activeElement` to decide whether the press
// needs to move focus back onto the video. Node has no document, so provide the
// one field they read; everything else about the element is faked below.
globalThis.document = { activeElement: null }

// The video element is the only thing the handlers are allowed to touch: they
// read its box and its intrinsic video size, and nothing else. A minimal
// stand-in is therefore enough to exercise the coordinate maths.
class FakeElement {
  constructor({ clientWidth = 0, clientHeight = 0, videoWidth = 0, videoHeight = 0 } = {}) {
    this.clientWidth = clientWidth
    this.clientHeight = clientHeight
    this.videoWidth = videoWidth
    this.videoHeight = videoHeight
    this.listeners = new Map()
    this.focusCalls = 0
  }

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
    const preventers = []
    for (const entry of (this.listeners.get(type) || []).slice()) {
      entry.fn({
        preventDefault() { preventers.push(type) },
        stopPropagation() {},
        ...event,
      })
    }
    return preventers
  }

  getBoundingClientRect() {
    return { left: 0, top: 0, width: this.clientWidth, height: this.clientHeight }
  }

  getAttribute(name) { return null }
  setAttribute() {}
  focus() { this.focusCalls += 1 }
  get tabIndex() { return 0 }
}

// A recorder that stands in for the data channel. What the handlers decide to
// send is the behaviour under test, so the recorder keeps the raw objects.
function makeRecorder() {
  const events = []
  return {
    events,
    sendMouseEvent(e) { events.push({ kind: 'mouse', ...e }) },
    sendKeyEvent(e) { events.push({ kind: 'key', ...e }) },
    sendTextEvent(t) { events.push({ kind: 'text', text: t }) },
  }
}

function makeMouse({ clientWidth, clientHeight, videoWidth, videoHeight, remoteWidth, remoteHeight }) {
  const element = new FakeElement({ clientWidth, clientHeight, videoWidth, videoHeight })
  const recorder = makeRecorder()
  const handler = new MouseHandler(element, recorder, element)
  handler.setRemoteResolution(remoteWidth, remoteHeight)
  handler.enable()
  return { element, recorder, handler }
}

// Before the first track arrives there is no honest resolution, and a guess is
// worse than nothing: 1920x1080 is a plausible desktop, so nothing looks broken
// while every click is quietly wrong. Dropping the event is the correct answer,
// and it is what makes the caller's early return safe to rely on.
test('a click with no known resolution is dropped, not injected at a guess', () => {
  const element = new FakeElement({
    clientWidth: 1000, clientHeight: 500,
    videoWidth: 1920, videoHeight: 1080,
  })
  const recorder = makeRecorder()
  const handler = new MouseHandler(element, recorder, element)

  // No setRemoteResolution call: this is the window between the session coming
  // up and the first track arriving.
  handler.enable()

  element.emit('mousedown', { clientX: 500, clientY: 250, button: 0, pointerId: 1 })

  assert.equal(recorder.events.length, 0,
    'a click was injected with a guessed resolution')

  handler.destroy()
})

// The point of the letterbox maths is that a click in the black bar must not
// become a click on the desktop. Returning null for those is the only thing
// stopping an off-screen coordinate from being injected at the edge.
test('a click in the letterbox does not reach the remote desktop', () => {
  const { element, recorder } = makeMouse({
    clientWidth: 1000, clientHeight: 500,
    videoWidth: 1920, videoHeight: 1080,
    remoteWidth: 1920, remoteHeight: 1080,
  })

  // The 16:9 video fills the height (500) and is 888 wide, so the bars are 56
  // pixels each. Landing in the left bar is outside the picture.
  const prevented = element.emit('mousedown', { clientX: 10, clientY: 250, button: 0 })

  assert.equal(recorder.events.length, 0)
  assert.ok(prevented.includes('mousedown'), 'the click must still be swallowed, not handed to the page')
})

test('a click at the picture edge maps to the last desktop column', () => {
  const { element, recorder } = makeMouse({
    clientWidth: 1000, clientHeight: 500,
    videoWidth: 1920, videoHeight: 1080,
    remoteWidth: 1920, remoteHeight: 1080,
  })

  // The picture is 888.9 wide and centred, so its right edge sits at 944.4.
  // Clicking just inside that edge must land on the last desktop column, not
  // one short of it, and never past it.
  element.emit('mousedown', { clientX: 944, clientY: 250, button: 0 })

  assert.equal(recorder.events.length, 1)
  assert.equal(recorder.events[0].x, 1919)
  assert.equal(recorder.events[0].y, 540)
  assert.equal(recorder.events[0].button, 1)
})

// Clamping matters when the rendered picture is a fraction of a pixel wider or
// taller than an exact multiple of the remote desktop. Rounding can then land
// on remoteWidth, which is a pixel the desktop does not have.
test('a coordinate rounded past the desktop is clamped into it', () => {
  // An exact 16:9 box means no letterbox at all, so every CSS pixel of the box
  // is a pixel of the picture.
  const { element, recorder } = makeMouse({
    clientWidth: 1920, clientHeight: 1080,
    videoWidth: 1920, videoHeight: 1080,
    remoteWidth: 1920, remoteHeight: 1080,
  })

  // The very last CSS pixel is at 1919, which is also the last desktop column.
  // Sub-pixel rendering can put the pointer at 1920 in a real browser, and a
  // desktop pixel that does not exist must not be requested.
  element.emit('mousemove', { clientX: 1920, clientY: 1080 })

  assert.equal(recorder.events.length, 1)
  assert.equal(recorder.events[0].x, 1919)
  assert.equal(recorder.events[0].y, 1079)
})

// Wheel deltas arrive in three different units depending on the browser and
// input device, and only one of them is already in pixels. A "line" delta
// reaching the host as a single pixel would make scrolling feel dead.
test('wheel deltas are normalised to pixels for every deltaMode', () => {
  const modes = [
    { deltaMode: 0, deltaY: 120, expected: 120 },
    { deltaMode: 1, deltaY: 3, expected: 120 },
    { deltaMode: 2, deltaY: 1, expected: 800 },
  ]

  for (const mode of modes) {
    const { element, recorder } = makeMouse({
      clientWidth: 1000, clientHeight: 500,
      videoWidth: 1920, videoHeight: 1080,
      remoteWidth: 1920, remoteHeight: 1080,
    })
    element.emit('wheel', { clientX: 500, clientY: 250, deltaX: 0, deltaY: mode.deltaY, deltaMode: mode.deltaMode })

    assert.equal(recorder.events.length, 1, `deltaMode ${mode.deltaMode} produced nothing`)
    assert.equal(recorder.events[0].wheelDeltaY, -mode.expected)
    assert.equal(recorder.events[0].wheelTicksY, -mode.expected / 120)
  }
})

// Switching monitors replaces the stream under the same <video> element. The
// element keeps reporting the *previous* stream's intrinsic size until the new
// metadata arrives, and the coordinate maths prefers that stale size over the
// resolution it was told. Every click then lands at a fraction of where the
// operator aimed, on a machine whose layout just changed.
test('a resolution told to the handler wins over a stale videoWidth', () => {
  const element = new FakeElement({
    clientWidth: 1000, clientHeight: 500,
    videoWidth: 1920, videoHeight: 1080,
  })
  const recorder = makeRecorder()
  const handler = new MouseHandler(element, recorder, element)

  handler.setRemoteResolution(1000, 500)
  handler.enable()

  element.emit('mousemove', { clientX: 750, clientY: 250 })

  assert.equal(recorder.events.length, 1)
  // With the stale 1920x1080 the picture is letterboxed to 888.9 wide and the
  // click maps to x=1000. With the told 1000x500 the picture fills the box and
  // the same click maps to x=750.
  assert.equal(recorder.events[0].x, 750, 'the stale video size was used instead of the new resolution')
  assert.equal(recorder.events[0].y, 250)

  handler.destroy()
})

// A drag that wanders off the video element must keep reporting the button as
// held. Without pointer capture the mouseup lands on whatever is under the
// cursor, the handler never sees it, and the remote machine is left with a
// button stuck down until the next click happens to land on the video again.
test('a drag that leaves the element still ends on the video', () => {
  const { element, recorder, handler } = makeMouse({
    clientWidth: 1000, clientHeight: 500,
    videoWidth: 1920, videoHeight: 1080,
    remoteWidth: 1920, remoteHeight: 1080,
  })

  const captured = []
  element.setPointerCapture = (id) => captured.push(id)
  element.releasePointerCapture = () => {}

  // Every MouseEvent carries a pointerId in a real browser, and it is what the
  // capture call needs.
  element.emit('mousedown', { clientX: 500, clientY: 250, button: 0, pointerId: 1 })
  assert.deepEqual(captured, [1], 'the press must capture the pointer')

  // The pointer has left the element; the browser retargets subsequent events
  // to the capturing element, which is this one.
  element.emit('mousemove', { clientX: 700, clientY: 300, pointerId: 1 })
  element.emit('mouseup', { clientX: 700, clientY: 300, button: 0, pointerId: 1 })

  const presses = recorder.events.filter((e) => e.kind === 'mouse' && e.button === 1)
  assert.equal(presses.filter((e) => e.buttonDown === true).length, 1)
  assert.equal(presses.filter((e) => e.buttonDown === false).length, 1, 'the release never reached the handler')

  handler.destroy()
})

// The same stuck-button problem on the keyboard side is worse: a key held when
// the window loses focus stays down on the remote machine, so the next person
// to type gets every keystroke shifted or controlled.
test('keys held when focus is lost are released to the host', () => {
  const element = new FakeElement()
  const recorder = makeRecorder()
  const handler = new KeyboardHandler(element, recorder)
  handler.enable()

  element.emit('keydown', { code: 'ShiftLeft', key: 'Shift', getModifierState: () => false })
  element.emit('keydown', { code: 'KeyA', key: 'a', getModifierState: () => false })

  const held = recorder.events.filter((e) => e.pressed === true)
  assert.equal(held.length, 2)

  // The window is switched away mid-hold. The host must be told both keys are up.
  element.emit('blur', {})

  const released = recorder.events.filter((e) => e.pressed === false)
  assert.equal(released.length, 2, 'keys held across the blur were never released')
  assert.deepEqual(released.map((e) => e.usbKeycode).sort(), [0x070004, 0x0700e1].sort())
})

// Tearing the session down while a key is down leaves the same stuck key on
// the host, because the element the listeners were bound to goes away with the
// panel. Releasing on destroy is the only place left to do it.
test('destroy releases keys that are still held', () => {
  const element = new FakeElement()
  const recorder = makeRecorder()
  const handler = new KeyboardHandler(element, recorder)
  handler.enable()

  element.emit('keydown', { code: 'ControlLeft', key: 'Control', getModifierState: () => false })

  handler.destroy()

  const released = recorder.events.filter((e) => e.pressed === false)
  assert.equal(released.length, 1, 'a held modifier was left down on the host')
  assert.equal(released[0].usbKeycode, 0x0700e0)
})
