// Turning the audit sweep response into something an operator can read.
//
// The sweep has six independent bounds - age, per-actor, per-device, per-action,
// table-wide rows, archive files - and each reports its own removal count. A
// single "purged N rows" would hide which bound did the work, and the answer to
// "why did my history disappear" is exactly that.
import test from 'node:test'
import assert from 'node:assert/strict'

const mod = await import('../src/views/audit-sweep-model.js')

const base = {
  purged: 120,
  older_than_days: 30,
  cutoff: '2026-09-27T00:00:00Z',
  max_rows: 0,
  cap_removed: 0,
  max_rows_per_actor: 500,
  actor_cap_removed: 40,
  max_rows_per_device: 0,
  device_cap_removed: 0,
  max_rows_per_action: 0,
  action_cap_removed: 0,
  archive_retention_days: 90,
  archive_removed: 3,
  archive_moved: 0,
  archive_deleted: 0,
  archive_dir: '/var/qd/audit',
  archive_cold_dir: '',
}

test('every bound that was configured is reported, with its own count', () => {
  const rows = mod.boundRows(base)
  const byLabel = Object.fromEntries(rows.map((r) => [r.label, r.value]))
  assert.equal(byLabel['age 30d'], 120)
  assert.equal(byLabel['per-actor 500'], 40)
  assert.equal(byLabel['archive 90d'], 3)
})

test('a bound that was not configured is not listed as zero work', () => {
  // max_rows=0 means "not set", so showing "table-wide 0 rows" would claim a
  // bound ran and found nothing.
  const labels = mod.boundRows(base).map((r) => r.label)
  assert.ok(!labels.some((l) => l.startsWith('table-wide')))
  assert.ok(!labels.some((l) => l.startsWith('per-device')))
})

test('a configured bound that removed nothing is still listed', () => {
  // This one is different from above: the bound ran, and "ran, found nothing"
  // is the answer the operator needs.
  const rows = mod.boundRows({ ...base, max_rows_per_actor: 0, actor_cap_removed: 0, max_rows: 1000, cap_removed: 0 })
  assert.ok(rows.some((r) => r.label === 'table-wide 1000'))
})

test('a sweep that removed nothing says so instead of an empty summary', () => {
  const s = mod.summary({ ...base, purged: 0, actor_cap_removed: 0, archive_removed: 0 })
  assert.equal(s.totalRemoved, 0)
  assert.equal(s.empty, true)
})

test('total counts every bound, not just the age purge', () => {
  const s = mod.summary(base)
  assert.equal(s.totalRemoved, 163)
})

test('the archive destination is named when files were moved to cold storage', () => {
  const s = mod.summary({ ...base, archive_moved: 2, archive_cold_dir: '/mnt/cold' })
  assert.equal(s.coldDir, '/mnt/cold')
  assert.equal(s.moved, 2)
})

test('no cold directory configured reads as not set', () => {
  assert.equal(mod.summary(base).coldDir, 'not set')
})

test('the cutoff is shown so an operator can tell what window ran', () => {
  assert.equal(mod.summary(base).cutoff, '2026-09-27T00:00:00Z')
})
