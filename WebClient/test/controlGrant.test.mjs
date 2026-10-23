import assert from 'node:assert/strict'
import test from 'node:test'

import { ControlGrantState } from '../js/ui/control-grant.js'

// The default has to be "may drive", because that is what every device already
// does: a client that started out refusing input would break control on every
// machine whose owner never touched the setting.
test('starts out able to control', () => {
    const state = new ControlGrantState()
    assert.equal(state.canControl, true)
    assert.equal(state.mode, 'shared')
    assert.equal(state.holder, '')
})

// The server's answer is authoritative: an exclusive grant for somebody else
// means this client must stop sending input.
test('an exclusive grant held by somebody else stops input', () => {
    const state = new ControlGrantState()
    state.applyGrant({ control_mode: 'exclusive', control_holder: 'cli-a', can_control: false })
    assert.equal(state.canControl, false)
    assert.equal(state.mode, 'exclusive')
    assert.equal(state.holder, 'cli-a')
})

test('an exclusive grant held by this client allows input', () => {
    const state = new ControlGrantState()
    state.applyGrant({ control_mode: 'exclusive', control_holder: 'cli-b', can_control: true })
    assert.equal(state.canControl, true)
    assert.equal(state.holder, 'cli-b')
})

// Shared mode never names a holder, so a grant that omits one must not be read
// as "waiting for a holder".
test('a shared grant keeps input enabled', () => {
    const state = new ControlGrantState()
    state.applyGrant({ control_mode: 'shared', can_control: true })
    assert.equal(state.canControl, true)
    assert.equal(state.holder, '')
})

// A frame that does not say whether this client may control is not an answer,
// and guessing would either freeze control or hand it to the wrong tab.
test('ignores a frame with no can_control', () => {
    const state = new ControlGrantState()
    state.applyGrant({ control_mode: 'exclusive', control_holder: 'cli-a' })
    assert.equal(state.canControl, true)
    assert.equal(state.holder, '')
})

test('ignores a frame with an unknown mode', () => {
    const state = new ControlGrantState()
    state.applyGrant({ control_mode: 'single', control_holder: 'cli-a', can_control: false })
    assert.equal(state.canControl, true)
})

// A takeover has to be reported once, with the new holder named, so the UI can
// say who took over instead of just "input stopped".
test('a revoke names the new holder and fires once', () => {
    const state = new ControlGrantState()
    state.applyGrant({ control_mode: 'exclusive', control_holder: 'cli-a', can_control: true })

    const events = []
    state.addEventListener('revoked', (event) => events.push(event.detail))

    state.applyRevoked({ control_mode: 'exclusive', control_holder: 'cli-b' })
    assert.equal(events.length, 1)
    assert.equal(events[0].holder, 'cli-b')
    assert.equal(state.canControl, false)
})

// The same revoke arriving twice - a replayed frame, a reconnecting socket -
// must not put up a second notice.
test('a repeated revoke does not fire twice', () => {
    const state = new ControlGrantState()
    state.applyGrant({ control_mode: 'exclusive', control_holder: 'cli-a', can_control: true })
    let count = 0
    state.addEventListener('revoked', () => { count += 1 })

    state.applyRevoked({ control_mode: 'exclusive', control_holder: 'cli-b' })
    state.applyRevoked({ control_mode: 'exclusive', control_holder: 'cli-b' })
    assert.equal(count, 1)
})

// Taking control back is the mirror image, and it must not fire a revoke.
test('regaining control clears the revoked state', () => {
    const state = new ControlGrantState()
    state.applyGrant({ control_mode: 'exclusive', control_holder: 'cli-a', can_control: true })
    state.applyRevoked({ control_mode: 'exclusive', control_holder: 'cli-b' })
    assert.equal(state.canControl, false)

    let revoked = 0
    state.addEventListener('revoked', () => { revoked += 1 })
    state.applyGrant({ control_mode: 'exclusive', control_holder: 'cli-a', can_control: true })

    assert.equal(state.canControl, true)
    assert.equal(state.holder, 'cli-a')
    assert.equal(revoked, 0)
})

// A revoke in shared mode is not a thing that can happen; ignoring it keeps a
// client from freezing on a frame the server should never have sent.
test('ignores a revoke in shared mode', () => {
    const state = new ControlGrantState()
    let revoked = 0
    state.addEventListener('revoked', () => { revoked += 1 })
    state.applyRevoked({ control_mode: 'shared' })
    assert.equal(revoked, 0)
    assert.equal(state.canControl, true)
})

// handleMessage routes by frame type and swallows anything else, so it can be
// wired straight onto the signaling socket alongside the jingle traffic.
test('handleMessage routes control frames and ignores the rest', () => {
    const state = new ControlGrantState()
    const seen = []
    state.addEventListener('revoked', (event) => seen.push(event.detail.holder))

    state.handleMessage(JSON.stringify({
        type: 'control_grant',
        control_mode: 'exclusive',
        control_holder: 'cli-a',
        can_control: true,
    }))
    assert.equal(state.canControl, true)

    state.handleMessage(JSON.stringify({
        type: 'control_revoked',
        control_mode: 'exclusive',
        control_holder: 'cli-b',
    }))
    assert.deepEqual(seen, ['cli-b'])

    // Not ours, and not even JSON: neither may disturb the state.
    state.handleMessage(JSON.stringify({ type: 'auth_ok', session_id: 'sig_1' }))
    state.handleMessage('not json at all')
    assert.equal(state.canControl, false)
})
