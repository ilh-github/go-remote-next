import assert from 'node:assert/strict'
import test from 'node:test'

globalThis.localStorage = {
  _values: new Map(),
  getItem(key) {
    return this._values.has(key) ? this._values.get(key) : null
  },
  setItem(key, value) {
    this._values.set(key, String(value))
  },
  removeItem(key) {
    this._values.delete(key)
  },
}

const {
  normalizeApprovalList,
  normalizeApprovalSummary,
} = await import('../src/api/accessApprovalModels.js')
const { userApi } = await import('../src/api/userApi.js')

test('approval list normalizes ids and keeps the security-relevant source fields', () => {
  const view = normalizeApprovalList({
    items: [{
      id: '12',
      device_id: '731905482',
      client_id: 'controller-a',
      client_ip: '203.0.113.7',
      source: 'device_code',
      status: 'pending',
      expires_at: '2026-10-17T05:00:00Z',
      created_at: '2026-10-17T04:58:00Z',
    }],
  })

  assert.equal(view[0].id, 12)
  assert.equal(view[0].client_ip, '203.0.113.7')
  assert.equal(view[0].source, 'device_code')
})

test('approval summary always exposes every lifecycle status', () => {
  const summary = normalizeApprovalSummary({
    since: '2026-10-16T00:00:00Z',
    total: 9,
    by_status: { pending: 2, denied: 7 },
    denied_sources: [{
      client_ip: '198.51.100.9',
      denials: 7,
      last_denied_at: '2026-10-17T04:50:00Z',
    }],
    denial_cooldown_seconds: 300,
    denial_cooldown_multiplier: 3,
    denial_cooldown_max_seconds: 3600,
  })

  assert.deepEqual(summary.byStatus, {
    pending: 2,
    approved: 0,
    denied: 7,
    expired: 0,
    consumed: 0,
  })
  assert.equal(summary.deniedSources[0].denials, 7)
  assert.equal(summary.denialCooldownSeconds, 300)
})

test('approval APIs use owner-scoped paths and optional reasons', async () => {
  const calls = []
  const original = userApi._req
  userApi._req = async (...args) => {
    calls.push(args)
    return { ok: true, data: {} }
  }
  try {
    await userApi.fetchAccessApprovals('device/1', {
      status: 'pending,denied',
      limit: 20,
    })
    await userApi.fetchAccessApprovalSummary('device/1', { limit: 10 })
    await userApi.approveAccessApproval('device/1', 12, 'recognized device')
    await userApi.denyAccessApproval('device/1', 13)
  } finally {
    userApi._req = original
  }

  assert.deepEqual(calls, [
    [
      'GET',
      '/v1/me/devices/device%2F1/access-approvals?status=pending%2Cdenied&limit=20',
    ],
    ['GET', '/v1/me/devices/device%2F1/access-approvals:summary?limit=10'],
    [
      'POST',
      '/v1/me/devices/device%2F1/access-approvals/12/approve',
      { reason: 'recognized device' },
    ],
    ['POST', '/v1/me/devices/device%2F1/access-approvals/13/deny', {}],
  ])
})
