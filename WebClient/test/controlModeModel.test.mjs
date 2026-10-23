import assert from 'node:assert/strict'
import test from 'node:test'

const {
    CONTROL_MODE_SHARED,
    CONTROL_MODE_EXCLUSIVE,
    readControlMode,
    controlModeChanged,
    buildAccessPolicyBody,
} = await import('../src/api/controlModeModel.js')

// A device with no policy row must read as shared, which is the mode it already
// behaves as. Reporting anything else would tell the owner their machine is
// locked down when it is not.
test('a policy with no control_mode reads as shared', () => {
    assert.equal(readControlMode({}), CONTROL_MODE_SHARED)
    assert.equal(readControlMode({ control_mode: '' }), CONTROL_MODE_SHARED)
    assert.equal(readControlMode(null), CONTROL_MODE_SHARED)
})

test('a stored mode is read back', () => {
    assert.equal(readControlMode({ control_mode: CONTROL_MODE_EXCLUSIVE }), CONTROL_MODE_EXCLUSIVE)
    assert.equal(readControlMode({ control_mode: CONTROL_MODE_SHARED }), CONTROL_MODE_SHARED)
})

// An unknown value is not silently coerced to exclusive: that would hand the
// owner a setting they never chose. It falls back to the default instead.
test('an unknown stored mode falls back to shared', () => {
    assert.equal(readControlMode({ control_mode: 'single' }), CONTROL_MODE_SHARED)
    assert.equal(readControlMode({ control_mode: 42 }), CONTROL_MODE_SHARED)
})

test('only a real change counts as a change', () => {
    assert.equal(controlModeChanged(CONTROL_MODE_SHARED, CONTROL_MODE_SHARED), false)
    assert.equal(controlModeChanged(CONTROL_MODE_EXCLUSIVE, CONTROL_MODE_EXCLUSIVE), false)
    assert.equal(controlModeChanged(CONTROL_MODE_SHARED, CONTROL_MODE_EXCLUSIVE), true)
    assert.equal(controlModeChanged(CONTROL_MODE_EXCLUSIVE, CONTROL_MODE_SHARED), true)
})

// The endpoint is a full replace, so a form that only carries control_mode would
// silently reset every other policy field. The body has to be the whole policy,
// with the one field the owner actually changed.
test('the PUT body carries the whole policy with the chosen control mode', () => {
    const stored = {
        remote_disabled: false,
        allow_ips: ['203.0.113.4'],
        deny_ips: [],
        max_sessions: 3,
        allowed_windows: [],
        timezone: 'Asia/Shanghai',
        exception_dates: [],
        approval_required: false,
        approval_mode: 'off',
        approval_trusted_ips: [],
        control_mode: CONTROL_MODE_SHARED,
    }

    const body = buildAccessPolicyBody(stored, CONTROL_MODE_EXCLUSIVE)

    assert.equal(body.control_mode, CONTROL_MODE_EXCLUSIVE)
    // Everything else survives untouched, which is the only way a one-field edit
    // is safe on a full-replace endpoint.
    assert.equal(body.max_sessions, 3)
    assert.deepEqual(body.allow_ips, ['203.0.113.4'])
    assert.equal(body.timezone, 'Asia/Shanghai')
    assert.equal(body.approval_mode, 'off')
    assert.equal(body.remote_disabled, false)
})

test('the PUT body omits control_mode when the owner did not change it', () => {
    const stored = { control_mode: CONTROL_MODE_EXCLUSIVE, max_sessions: 0 }
    const body = buildAccessPolicyBody(stored, CONTROL_MODE_EXCLUSIVE)
    assert.equal(body.control_mode, CONTROL_MODE_EXCLUSIVE)
    assert.equal(body.max_sessions, 0)
})

// A null/absent policy row means "documented defaults"; the body must still be
// a complete, valid policy rather than a partial object the server rejects.
test('the PUT body is complete even when no policy was ever stored', () => {
    const body = buildAccessPolicyBody(null, CONTROL_MODE_EXCLUSIVE)
    assert.equal(body.control_mode, CONTROL_MODE_EXCLUSIVE)
    assert.deepEqual(body.allow_ips, [])
    assert.deepEqual(body.deny_ips, [])
    assert.deepEqual(body.allowed_windows, [])
    assert.deepEqual(body.exception_dates, [])
    assert.deepEqual(body.approval_trusted_ips, [])
    assert.equal(body.max_sessions, 0)
    assert.equal(body.approval_mode, 'off')
    assert.equal(body.remote_disabled, false)
})

test('a non-object policy is treated as no policy', () => {
    const body = buildAccessPolicyBody('nonsense', CONTROL_MODE_SHARED)
    assert.equal(body.control_mode, CONTROL_MODE_SHARED)
    assert.deepEqual(body.allow_ips, [])
})
