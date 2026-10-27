// What the notification delivery panel shows, from
// GET /v1/admin/notification-deliveries:stats.
//
// The interesting number is not `total`: a channel that answers 200 and drops
// the event is indistinguishable from one that works unless you look at
// `unacked`, which is rows that were delivered and never confirmed. That
// distinction is the whole reason the ack loop exists.
import test from 'node:test'
import assert from 'node:assert/strict'

const mod = await import('../src/views/notification-stats-model.js')

const base = {
  total: 12,
  by_status: { pending: 3, delivering: 1, delivered: 6, failed: 2 },
  acked: 4,
  unacked: 5,
  retention_days: 30,
  archive_retention_days: 90,
  archive_dir: '/var/qd/notifications',
}

test('all four delivery states are shown, zeroes included', () => {
  // The server always sends all four; a panel that filters out zeroes makes
  // "no failures" look the same as "not asked".
  const rows = mod.statusRows(base)
  assert.deepEqual(rows, [
    ['pending', 3],
    ['delivering', 1],
    ['delivered', 6],
    ['failed', 2],
  ])
})

test('an absent state still shows as zero rather than vanishing', () => {
  const rows = mod.statusRows({ by_status: { pending: 1, delivered: 2 } })
  assert.deepEqual(rows, [
    ['pending', 1],
    ['delivering', 0],
    ['delivered', 2],
    ['failed', 0],
  ])
})

test('unacked is the number that says nobody confirmed', () => {
  const s = mod.summary(base)
  assert.equal(s.unacked, 5)
  assert.equal(s.acked, 4)
})

test('a delivered-but-never-confirmed channel is flagged', () => {
  // 2xx does not mean anybody saw it. This is the state the ack loop exists for.
  assert.equal(mod.summary(base).needsAttention, true)
})

test('a fully confirmed queue is not flagged', () => {
  const s = mod.summary({ ...base, delivered: 6, acked: 6, unacked: 0 })
  assert.equal(s.needsAttention, false)
})

test('a stuck queue is flagged even with nothing unacked', () => {
  const s = mod.summary({
    ...base,
    by_status: { pending: 9, delivering: 0, delivered: 0, failed: 0 },
    acked: 0,
    unacked: 0,
  })
  assert.equal(s.needsAttention, true)
})

test('retention windows read as disabled when unset, not as zero days', () => {
  const s = mod.summary({ ...base, retention_days: 0, archive_retention_days: 0 })
  assert.equal(s.retention, 'off')
  assert.equal(s.archiveRetention, 'off')
})

test('a missing archive directory is called out rather than left blank', () => {
  assert.equal(mod.summary({ ...base, archive_dir: '' }).archiveDir, 'not set')
  assert.equal(mod.summary(base).archiveDir, '/var/qd/notifications')
})
