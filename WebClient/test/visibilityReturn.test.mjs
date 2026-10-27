import assert from 'node:assert/strict'
import test from 'node:test'

// The release path tears the input handlers down on every way an operator stops
// paying attention. These pin the return path that puts them back - the half
// that was missing, which is why clipboard sync died on the first blur.
const { watchForReturn } = await import('../js/input/visibilityRelease.js')

function fakeWindow() {
    const listeners = {}
    const document = {
        hidden: false,
        addEventListener(event, fn) { (listeners[event] ||= []).push(fn) },
        removeEventListener(event, fn) {
            if (!listeners[event]) return
            listeners[event] = listeners[event].filter((f) => f !== fn)
        },
    }
    const window = {
        document,
        addEventListener(event, fn) { (listeners[event] ||= []).push(fn) },
        removeEventListener(event, fn) {
            if (!listeners[event]) return
            listeners[event] = listeners[event].filter((f) => f !== fn)
        },
    }
    return {
        window,
        document,
        fire(event) { for (const fn of listeners[event] || []) fn() },
        count(event) { return (listeners[event] || []).length },
    }
}

test('becoming visible again fires the return', () => {
    const fake = fakeWindow()
    let returns = 0
    watchForReturn(fake.window, () => { returns += 1 })

    fake.fire('visibilitychange')

    assert.equal(returns, 1, 'becoming visible again did not restore the handlers')
})

test('a visibilitychange that leaves the page hidden does not fire', () => {
    const fake = fakeWindow()
    fake.document.hidden = true
    let returns = 0
    watchForReturn(fake.window, () => { returns += 1 })

    fake.fire('visibilitychange')

    assert.equal(returns, 0, 'a still-hidden page was treated as the operator returning')
})

test('regaining window focus fires the return', () => {
    // A window can stay visible while another application takes focus, so blur
    // is not the only way to stop driving and focus is not the only way back.
    const fake = fakeWindow()
    let returns = 0
    watchForReturn(fake.window, () => { returns += 1 })

    fake.fire('focus')

    assert.equal(returns, 1, 'regaining focus did not restore the handlers')
})

test('detaching stops the returns', () => {
    const fake = fakeWindow()
    let returns = 0
    const detach = watchForReturn(fake.window, () => { returns += 1 })

    detach()
    fake.fire('visibilitychange')
    fake.fire('focus')

    assert.equal(returns, 0, 'a detached watcher still fired')
})

test('a target without addEventListener is not a crash', () => {
    // The module is used in tests and in environments where window is a stub.
    assert.doesNotThrow(() => watchForReturn(null, () => {}))
    assert.doesNotThrow(() => watchForReturn({}, () => {}))
})
