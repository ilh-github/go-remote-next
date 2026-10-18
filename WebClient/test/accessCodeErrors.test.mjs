import assert from 'node:assert/strict'
import test from 'node:test'

import { describeAccessCodeFailure } from '../js/auth/access-code-errors.js'

test('access verification distinguishes policy denial from bad credentials', () => {
  assert.deepEqual(
    describeAccessCodeFailure({ ok: false, code: 'ACCESS_POLICY_DENIED', status: 403 }),
    { code: 'ACCESS_POLICY_DENIED', messageKey: 'status.accessDenied' },
  )
  assert.deepEqual(
    describeAccessCodeFailure({ ok: false, code: 'INVALID_CODE', status: 403 }),
    { code: 'INVALID_CODE', messageKey: 'status.invalidAccessCode' },
  )
})

test('access verification preserves offline, rate-limit, and proof failures', () => {
  assert.deepEqual(
    describeAccessCodeFailure({ ok: false, code: 'HOST_OFFLINE', status: 409 }),
    { code: 'HOST_OFFLINE', messageKey: 'status.reason.HOST_OFFLINE' },
  )
  assert.deepEqual(
    describeAccessCodeFailure({ ok: false, data: { code: 'TOO_MANY_ATTEMPTS' }, status: 429 }),
    { code: 'TOO_MANY_ATTEMPTS', messageKey: 'status.tooManyAttempts' },
  )
  assert.deepEqual(
    describeAccessCodeFailure({ ok: false, code: 'CONTROLLER_PROOF_INVALID', status: 403 }),
    { code: 'CONTROLLER_PROOF_INVALID', messageKey: 'status.controllerProofRequired' },
  )
})

test('approval pending is not reported as missing parameters', () => {
  assert.deepEqual(
    describeAccessCodeFailure({
      ok: true,
      status: 202,
      data: { approval_id: 7, status: 'pending' },
    }),
    { code: 'APPROVAL_PENDING', messageKey: 'status.approvalPending' },
  )
})

test('successful verification needs a token and unknown failures stay actionable', () => {
  assert.equal(
    describeAccessCodeFailure({
      ok: true,
      status: 200,
      data: { signal_token: 'one-shot' },
    }),
    null,
  )
  assert.deepEqual(
    describeAccessCodeFailure({ ok: false, status: 502 }),
    { code: 'ACCESS_VERIFY_FAILED', messageKey: 'status.accessVerifyFailed' },
  )
})
