// The recording storage numbers an operator acts on.
//
// The server reports two different byte totals on purpose: indexed_bytes is the
// database index sum, total_bytes walks the real directory including thumbnails
// and .part staging files. Showing only the index tells the operator the disk is
// emptier than it is, which is the number that decides whether a quota is about
// to be hit. Both are shown, and the gap is named when there is one.
import test from 'node:test'
import assert from 'node:assert/strict'

const mod = await import('../src/views/recording-storage-model.js')
const sizes = await import('../src/views/storage-size.js')

const base = {
  artifact_count: 6,
  indexed_bytes: 56791553,
  total_bytes: 56795649,
  file_count: 7,
  quota_bytes: 0,
  remaining_bytes: 0,
  over_quota: false,
  retention_days: 30,
  stale_after_minutes: 120,
}

test('both byte totals are reported, not just the index', () => {
  const rows = mod.storageRows(base)
  assert.equal(rows.indexed, '54.2 MB')
  assert.equal(rows.disk, '54.2 MB')
})

test('a gap between index and disk is named, not silently averaged', () => {
  // Thumbnails and staged uploads are real bytes the index does not know about.
  assert.equal(mod.storageGap(base).hidden, 4096)
})

test('a quota of zero means unlimited rather than full', () => {
  const rows = mod.storageRows({ ...base, quota_bytes: 0 })
  assert.equal(rows.quota, 'unlimited')
})

test('a quota with room left reports the remainder', () => {
  const rows = mod.storageRows({ ...base, quota_bytes: 100000, remaining_bytes: 43047 })
  assert.equal(rows.quota, '97.7 KB')
  assert.equal(rows.remaining, '42.0 KB')
  assert.equal(rows.over, false)
})

test('an over-quota server is flagged even when remaining is negative', () => {
  const rows = mod.storageRows({ ...base, quota_bytes: 1000, remaining_bytes: 0, over_quota: true })
  assert.equal(rows.over, true)
  assert.equal(rows.remaining, '0 B')
})

test('a zero retention window reads as disabled, not as zero days left', () => {
  assert.equal(sizes.retentionLabel(0), 'off')
  assert.equal(sizes.retentionLabel(30), '30 d')
})

test('a missing stale window does not render as "0 min"', () => {
  assert.equal(sizes.staleLabel(0), 'not set')
  assert.equal(sizes.staleLabel(120), '120 min')
})

test('a negative remainder means no quota is set, not that the disk is full', () => {
  // The server reports -1 when no quota is configured. Rendering that as "0 B"
  // remaining would tell the operator a full disk when nothing is capped.
  const rows = mod.storageRows({ ...base, quota_bytes: 0, remaining_bytes: -1 })
  assert.equal(rows.remaining, 'not set')
})

test('a quota that is genuinely full reports a real zero', () => {
  const rows = mod.storageRows({ ...base, quota_bytes: 1000, remaining_bytes: 0, over_quota: true })
  assert.equal(rows.remaining, '0 B')
})

test('a zero-byte artifact reports 0 B rather than a dash', () => {
  // An empty recording is a real fact, not missing data, and a dash reads as
  // "unknown" on a size column.
  assert.equal(sizes.formatSize(0), '0 B')
})

test('a missing or non-numeric size stays unknown', () => {
  assert.equal(sizes.formatSize(undefined), '-')
  assert.equal(sizes.formatSize(null), '-')
  assert.equal(sizes.formatSize('abc'), '-')
  assert.equal(sizes.formatSize(NaN), '-')
})

test('a negative size is not silently reported as zero', () => {
  // Only -1 has a meaning (no quota set); anything else negative is a bug and
  // should not look like a legitimate measurement.
  assert.equal(sizes.formatSize(-5), '-')
})
