import assert from 'node:assert/strict'
import test from 'node:test'

// localStorage is a getter-only global in Node, so the store is installed the
// only way that works. It has to exist before the client module is imported:
// `new UserApi()` reads it in its constructor.
Object.defineProperty(globalThis, 'localStorage', {
  value: {
    _v: new Map(),
    getItem(key) { return this._v.has(key) ? this._v.get(key) : null },
    setItem(key, value) { return this._v.set(key, String(value)) },
    removeItem(key) { return this._v.delete(key) },
  },
  configurable: true,
  writable: true,
})

const { describeSessionDetail, resolveRecordingUrl } = await import('../src/sessionDetailModel.js')

// QD-208 promises the session detail aggregates the session, its recordings and
// its access log. The server does return all three -- `detailEnvelope` puts the
// full recordings array in the payload -- but the UI rendered only
// `counts.recordings` and listed nothing, while the access log got a full list.
// A count tells the owner a recording exists; it does not tell them which one,
// when it was made, or whether it has media to play.

const sampleDetail = () => ({
  session: { session_id: 'sig_a', device_id: '700000001', active: false },
  recordings: [
    { id: 11, session_id: 'sig_a', status: 'completed', duration: 90, created_at: '2026-10-18T10:00:00Z', artifact_url: '/media/a.mp4' },
    { id: 12, session_id: 'sig_a', status: 'recording', duration: 0, created_at: '2026-10-18T10:05:00Z' },
  ],
  access_log: [
    { id: 21, session_id: 'sig_a', result: 'session_started', ip: '203.0.113.7', created_at: '2026-10-18T09:59:00Z' },
  ],
  counts: { recordings: 2, access_log: 1 },
})

test('the detail describes its recordings rather than only counting them', () => {
  const view = describeSessionDetail(sampleDetail())
  assert.equal(view.recordings.length, 2)
  assert.equal(view.counts.recordings, 2)
})

test('a recording row carries what the owner needs to recognise it', () => {
  const [first] = describeSessionDetail(sampleDetail()).recordings
  assert.equal(first.id, 11)
  assert.equal(first.status, 'completed')
  assert.match(first.duration, /1m30s/)
  assert.equal(first.createdAt, '2026-10-18T10:00:00Z')
})

// Playability is the difference between "there is a recording" and "there is
// something to watch". A row with no media must not offer a play button.
test('only a recording with media is playable', () => {
  const rows = describeSessionDetail(sampleDetail()).recordings
  assert.equal(rows[0].playable, true)
  assert.equal(rows[1].playable, false)
})

test('an absent recordings array is an empty list, not a crash', () => {
  const view = describeSessionDetail({ session: {}, access_log: [], counts: {} })
  assert.deepEqual(view.recordings, [])
  assert.deepEqual(view.accessLog, [])
})

test('a zero recording count is rendered as zero, not blank', () => {
  const view = describeSessionDetail({
    session: {},
    recordings: [],
    access_log: [],
    counts: { recordings: 0, access_log: 0 },
  })
  assert.equal(view.counts.recordings, 0)
  assert.equal(view.recordings.length, 0)
})

// The counts the server sends are derived from the arrays, so a payload whose
// counts disagree with its arrays must render the arrays: the rows are what the
// owner acts on.
test('the arrays win over the counts when they disagree', () => {
  const detail = sampleDetail()
  detail.counts = { recordings: 99, access_log: 99 }
  const view = describeSessionDetail(detail)
  assert.equal(view.recordings.length, 2)
  assert.equal(view.accessLog.length, 1)
})

test('a relative artifact url is anchored against the http form of the server', () => {
  const url = resolveRecordingUrl({ artifact_url: '/media/a.mp4' }, 'wss://signal.example.com')
  assert.equal(url, 'https://signal.example.com/media/a.mp4')
})

test('an absolute artifact url is left alone', () => {
  const url = resolveRecordingUrl({ artifact_url: 'https://cdn.example.com/a.mp4' }, 'wss://signal.example.com')
  assert.equal(url, 'https://cdn.example.com/a.mp4')
})

test('a row with no artifact has no url, and so is not playable', () => {
  assert.equal(resolveRecordingUrl({ artifact_url: '' }, 'wss://signal.example.com'), null)
  assert.equal(resolveRecordingUrl(null, 'wss://signal.example.com'), null)
})

test('the access log keeps being described', () => {
  const view = describeSessionDetail(sampleDetail())
  assert.equal(view.accessLog.length, 1)
  assert.equal(view.accessLog[0].result, 'session_started')
})
