import assert from 'node:assert/strict'
import test from 'node:test'

// QD-105, client half. `users.device_count` sat in the database and in /v1/me from
// the start while nothing read it, so an account could be "at its limit" in the
// admin console and keep binding machines anyway. The server now enforces it;
// these tests pin the rule the device list renders from.

const {
  deviceQuota,
  deviceQuotaReached,
  deviceQuotaRemaining,
  deviceQuotaSummary,
} = await import('../js/devices/quota.js')

// 0 is the default every account is created with, and it means "no limit". If it
// were read as a hard zero the device list would tell a user with three working
// machines that they are over a limit that does not exist.
test('a quota of zero means no limit', () => {
  assert.equal(deviceQuota({ device_count: 0 }), null)
  assert.equal(deviceQuota({}), null)
  assert.equal(deviceQuota(null), null)
  assert.equal(deviceQuota(undefined), null)
  assert.equal(deviceQuota({ device_count: -3 }), null)
  assert.equal(deviceQuota({ device_count: 'many' }), null)
})

test('a positive quota is reported as-is', () => {
  assert.equal(deviceQuota({ device_count: 5 }), 5)
})

test('remaining counts down and never goes negative', () => {
  assert.equal(deviceQuotaRemaining({ device_count: 5 }, 2), 3)
  assert.equal(deviceQuotaRemaining({ device_count: 1 }, 1), 0)
  // A takeover can transiently leave the count above the quota if an admin
  // lowered it, and the UI must not then display a negative number of slots.
  assert.equal(deviceQuotaRemaining({ device_count: 2 }, 7), 0)
  assert.equal(deviceQuotaRemaining({ device_count: 0 }, 7), null)
})

test('the summary carries the limit, the count and the slots left', () => {
  assert.deepEqual(deviceQuotaSummary({ device_count: 3 }, 1), {
    limited: true,
    owned: 1,
    quota: 3,
    remaining: 2,
  })
  assert.deepEqual(deviceQuotaSummary({ device_count: 0 }, 4), {
    limited: false,
    owned: 4,
    quota: null,
    remaining: null,
  })
})

// The device list has to be able to warn before a bind is attempted: the host
// binds itself on startup, and a refusal the owner only learns about by noticing
// a missing machine is the failure this whole change exists to prevent.
test('quota reached is true only when a limit exists and is full', () => {
  assert.equal(deviceQuotaReached({ device_count: 1 }, 1), true)
  assert.equal(deviceQuotaReached({ device_count: 2 }, 1), false)
  assert.equal(deviceQuotaReached({ device_count: 0 }, 9), false)
  assert.equal(deviceQuotaReached(null, 9), false)
})
