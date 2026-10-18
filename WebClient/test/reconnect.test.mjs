import test from 'node:test'
import assert from 'node:assert/strict'

import {
  ReconnectExhaustedError,
  isReconnectableAccessFailure,
  nonRetryableReconnectError,
  runReconnectAttempts,
} from '../js/signaling/reconnect.js'

test('reconnect succeeds after transient failures', async () => {
  const waits = []
  const attempts = []
  let calls = 0
  const result = await runReconnectAttempts(
    async ({ attempt, total }) => {
      attempts.push({ attempt, total })
      calls += 1
      if (calls < 3) throw new Error(`failure-${calls}`)
      return 'connected'
    },
    {
      delaysMs: [0, 10, 20],
      wait: async (ms) => { waits.push(ms) },
    },
  )

  assert.equal(result, 'connected')
  assert.deepEqual(waits, [10, 20])
  assert.deepEqual(attempts, [
    { attempt: 1, total: 3 },
    { attempt: 2, total: 3 },
    { attempt: 3, total: 3 },
  ])
})

test('reconnect reports exhaustion with the last transient error', async () => {
  const waits = []
  await assert.rejects(
    () => runReconnectAttempts(
      async ({ attempt }) => {
        throw new Error(`transient-${attempt}`)
      },
      {
        delaysMs: [0, 5, 15],
        wait: async (ms) => { waits.push(ms) },
      },
    ),
    (error) => {
      assert.equal(error instanceof ReconnectExhaustedError, true)
      assert.equal(error.attempts, 3)
      assert.equal(error.retryable, true)
      assert.equal(error.cause.message, 'transient-3')
      return true
    },
  )
  assert.deepEqual(waits, [5, 15])
})

test('non-retryable reconnect stops without consuming later attempts', async () => {
  const waits = []
  await assert.rejects(
    () => runReconnectAttempts(
      async () => {
        throw nonRetryableReconnectError('invalid', {
          code: 'INVALID_CODE',
          messageKey: 'status.invalidAccessCode',
        })
      },
      {
        delaysMs: [0, 1000, 3000],
        wait: async (ms) => { waits.push(ms) },
      },
    ),
    (error) => {
      assert.equal(error instanceof ReconnectExhaustedError, true)
      assert.equal(error.attempts, 1)
      assert.equal(error.retryable, false)
      assert.equal(error.cause.code, 'INVALID_CODE')
      assert.equal(error.cause.messageKey, 'status.invalidAccessCode')
      return true
    },
  )
  assert.deepEqual(waits, [])
})

test('only transient access failures are retried', () => {
  assert.equal(isReconnectableAccessFailure({ code: 'HOST_OFFLINE' }), true)
  assert.equal(isReconnectableAccessFailure({ code: 'ACCESS_VERIFY_FAILED' }), true)
  assert.equal(isReconnectableAccessFailure({ code: 'RATE_LIMITED' }), true)
  assert.equal(isReconnectableAccessFailure({ code: 'INVALID_CODE' }), false)
  assert.equal(isReconnectableAccessFailure({ code: 'ACCESS_POLICY_DENIED' }), false)
  assert.equal(isReconnectableAccessFailure(null), false)
})
