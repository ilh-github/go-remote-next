import assert from 'node:assert/strict'
import test from 'node:test'

// The wall resolves one row per selected device from the device list and the
// per-device live state. It used to do `devices.find(...)` inside the computed
// that mapped over the selection, which is quadratic in the number of tiles -
// and a wall is a grid of hundreds of tiles by design, so that is the common
// case rather than an edge. These tests pin the behaviour and the cost.

const { indexDevices, resolveTiles } = await import('../js/wall/tileModel.js')

function device(id, overrides = {}) {
  return { device_id: id, remark: `host-${id}`, online: true, logged_in: true, ...overrides }
}

test('the index maps every device by its id', () => {
  const index = indexDevices([device('a'), device('b')])
  assert.deepEqual(Object.keys(index).sort(), ['a', 'b'])
  assert.equal(index.a.remark, 'host-a')
})

// A device list that is not an array, or a row without an id, must not throw:
// the wall renders from whatever the API returned, and a poll that came back
// half-formed must not take the page down.
test('a malformed device list yields an empty index rather than a crash', () => {
  assert.deepEqual(indexDevices(undefined), {})
  assert.deepEqual(indexDevices(null), {})
  assert.deepEqual(indexDevices('nope'), {})
  assert.deepEqual(indexDevices([{ remark: 'no id' }]), {})
})

test('an online device resolves to a connecting tile carrying its url', () => {
  const rows = resolveTiles({
    selectedIds: ['a'],
    devices: [device('a')],
    live: { a: { state: 'connecting', url: 'remote.html?device=a' } },
  })
  assert.equal(rows.length, 1)
  assert.equal(rows[0].state, 'connecting')
  assert.equal(rows[0].url, 'remote.html?device=a')
  assert.equal(rows[0].name, 'host-a')
})

// A device the caller no longer owns is an error row, not a silent gap: the
// tile has to say so rather than rendering as a placeholder that looks like it
// is still loading.
test('an unknown device is an error row, not a silent gap', () => {
  const rows = resolveTiles({ selectedIds: ['gone'], devices: [], translate: (k) => k })
  assert.equal(rows[0].state, 'error')
  assert.equal(rows[0].reason, 'wall.tileUnknown')
})

test('an offline device drops its url so a stale token is not retried', () => {
  const rows = resolveTiles({
    selectedIds: ['a'],
    devices: [device('a', { online: false })],
    live: { a: { url: 'remote.html?device=a', state: 'connecting' } },
    translate: (k) => k,
  })
  assert.equal(rows[0].state, 'offline')
  assert.equal(rows[0].url, '', 'an offline device must not keep a stale url')
  assert.equal(rows[0].reason, 'wall.reasonOffline')
})

test('a device that is not logged in is idle, also without its url', () => {
  const rows = resolveTiles({
    selectedIds: ['a'],
    devices: [device('a', { logged_in: false })],
    live: { a: { url: 'remote.html?device=a' } },
    translate: (k) => k,
  })
  assert.equal(rows[0].state, 'idle')
  assert.equal(rows[0].url, '')
  assert.equal(rows[0].reason, 'wall.reasonIdle')
})

// The order of the selection is the display order. A wall that reordered its
// tiles between renders would move the tile the operator was reaching for.
test('rows keep the selection order', () => {
  const rows = resolveTiles({
    selectedIds: ['c', 'a', 'b'],
    devices: [device('a'), device('b'), device('c')],
  })
  assert.deepEqual(rows.map((row) => row.deviceId), ['c', 'a', 'b'])
})

// The name prefers the remark, then the device name, then the id - the same
// precedence the inline version had, so a rename does not change what a tile
// shows.
test('the tile name prefers remark, then device name, then id', () => {
  const rows = resolveTiles({
    selectedIds: ['a', 'b', 'c'],
    devices: [
      device('a', { remark: 'front desk' }),
      device('b', { remark: '', device_name: 'lab machine' }),
      device('c', { remark: '', device_name: '' }),
    ],
  })
  assert.deepEqual(rows.map((row) => row.name), ['front desk', 'lab machine', 'c'])
})

// The cost is pinned structurally, not by timing.
//
// A timing guard was tried first and is worth recording why it was dropped: V8
// optimises a 400-element `find` well enough that the quadratic version finished
// in 9.1 ms against the indexed version's 3.6 ms - both pass any bound worth
// setting, so the test could not tell them apart and would have been a guard
// that is always green. What does discriminate is counting how many times the
// device list is *touched*: an indexed build reads each device once, a per-tile
// scan reads the list once per tile.
test('the device list is indexed once rather than scanned per tile', () => {
  let comparisons = 0
  const ids = []
  const devices = []
  for (let i = 0; i < 200; i += 1) {
    const id = `dev-${String(i).padStart(3, '0')}`
    ids.push(id)
    devices.push({ device_id: id, remark: id, online: true, logged_in: true })
  }

  // A list whose every comparison is counted. This is the operation the
  // quadratic version performed once per tile; the indexed version performs it
  // never, because it builds a lookup instead of searching.
  const countingDevices = new Proxy(devices, {
    get(target, prop, receiver) {
      // Only `find`. Iterating the list to build an index reads `length` and the
      // iterator once per pass, which is the cost being kept; a per-tile scan
      // additionally calls `find` once per tile, and that is the operation the
      // count has to catch.
      if (prop === 'find') {
        comparisons += 1
      }
      return Reflect.get(target, prop, receiver)
    },
  })

  const rows = resolveTiles({ selectedIds: ids, devices: countingDevices })
  assert.equal(rows.length, 200)

  // An indexed build never searches, so `find` is never called. The per-tile
  // scan this replaced called it once per tile - 200 times here.
  assert.equal(comparisons, 0,
    `the device list was searched ${comparisons} times; a per-tile scan searches it once per tile`)
})
