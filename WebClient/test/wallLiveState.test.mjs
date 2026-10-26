import assert from 'node:assert/strict'
import test from 'node:test'

// QD-908: the 40-viewer 90-second soak attributed 29.6% of sampled allocation
// to compileMessageFormat in vue-i18n, and the heap series carried a decisive
// 2008ms period. The period matches the 2000ms `getStats` interval in the
// controlled-side iframe almost exactly, and the allocation does not depend on
// the frame loop - both point at the same path.
//
// Each iframe posts its frame count every 500ms. onWallMessage answered by
// rebuilding the whole `live[deviceId]` object, so every tile re-rendered on
// every counter tick, and a tile template carries eight `$t()` bindings that
// each compile an i18n message. A frame counter that exists to be displayed was
// therefore re-running the message compiler for the whole wall, twice a second.
//
// The fix is to merge the reported fields into the existing entry instead of
// replacing it, so a tick that changes nothing but `detail` is not a re-render.

const { mergeLiveState } = await import('../js/wall/liveState.js')

// The bug, pinned: a message whose only change is the frame counter must not
// produce a new object. A new object is what makes Vue re-render the tile, and
// re-rendering the tile is what runs the message compiler.
test('a tick that changes only the frame counter keeps the same object', () => {
  const live = { 'soak-001': { state: 'live', detail: '1534', url: 'remote.html?device=soak-001' } }
  const before = live['soak-001']
  const changed = mergeLiveState(live, {
    device: 'soak-001', state: 'live', detail: '1536',
  })
  assert.equal(changed, false, 'nothing observable changed, so no re-render is owed')
  assert.equal(live['soak-001'], before, 'the entry must be the same object')
  assert.equal(live['soak-001'].detail, '1536', 'the counter still has to advance')
})

// A real state change must still produce a new object, or the tile would never
// update. The optimisation is about not re-rendering when nothing visible
// changed, not about suppressing updates.
test('a state change still produces a new object', () => {
  const live = { 'soak-001': { state: 'connecting', detail: '' } }
  const before = live['soak-001']
  const changed = mergeLiveState(live, {
    device: 'soak-001', state: 'live', detail: '1',
  })
  assert.equal(changed, true, 'connecting -> live is a real change')
  assert.notEqual(live['soak-001'], before)
  assert.equal(live['soak-001'].state, 'live')
})

// The counter has to advance even though the object is reused: the whole point
// is that the number stays live without the tile re-rendering for it.
test('the frame counter advances on a reused object', () => {
  const live = { a: { state: 'live', detail: '10' } }
  for (const detail of ['11', '12', '13']) {
    mergeLiveState(live, { device: 'a', state: 'live', detail })
    assert.equal(live.a.detail, detail)
  }
})

// An empty detail must not be treated as a change: the iframe reports
// `detail: ''` before its first frame, and the wall would otherwise re-render on
// every one of those too.
test('an empty counter is not a change', () => {
  const live = { a: { state: 'live', detail: '5' } }
  const before = live.a
  assert.equal(mergeLiveState(live, { device: 'a', state: 'live', detail: '' }), false)
  assert.equal(live.a, before)
})

// A first sighting still has to create the entry.
test('a device seen for the first time is recorded', () => {
  const live = {}
  assert.equal(mergeLiveState(live, { device: 'b', state: 'live', detail: '7' }), true)
  assert.equal(live.b.state, 'live')
  assert.equal(live.b.detail, '7')
})

// A message for a device nobody selected is ignored, exactly as before.
test('an unselected device is ignored', () => {
  const live = {}
  assert.equal(mergeLiveState(live, { device: 'c', state: 'live', detail: '7' }, ['a']), false)
  assert.deepEqual(live, {})
})

// Malformed input must not throw: the wall receives postMessage from every
// iframe on the page, and one bad frame must not take the wall down.
test('a malformed message is ignored', () => {
  const live = { a: { state: 'live', detail: '1' } }
  assert.equal(mergeLiveState(live, null), false)
  assert.equal(mergeLiveState(live, {}), false)
  assert.equal(mergeLiveState(live, { device: 'a' }), false)
  assert.equal(live.a.detail, '1', 'a message with no state or detail leaves the entry alone')
})
