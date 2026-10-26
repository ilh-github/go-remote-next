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

// QD-908 follow-up: the allocation profile named `compileMessageFormat` in
// vue-i18n as the top self-size path, and merging the live state only removed a
// quarter of it. The rest was here: `resolveTiles` calls `t()` three times per
// tile on every recompute, and the recompute runs on every live-state tick.
// A reason string that only exists to be looked up when a tile is offline must
// not be compiled for the tiles that are live.
test('a live tile resolves without compiling its reason strings', () => {
  let calls = 0
  const translate = (key) => { calls += 1; return key }
  resolveTiles({
    selectedIds: ['a', 'b', 'c'],
    devices: [device('a'), device('b'), device('c')],
    live: {
      a: { state: 'live', url: 'remote.html?device=a' },
      b: { state: 'live', url: 'remote.html?device=b' },
      c: { state: 'live', url: 'remote.html?device=c' },
    },
    translate,
  })
  assert.equal(calls, 0,
    `a wall of live tiles compiled ${calls} reason strings it never shows`)
})

// The rest of the i18n cost. The allocation profile named `compileMessageFormat`
// as the top self-size path, and a tile template carries eight `$t()` bindings.
// Merging the live state removed a quarter of it; this is the remainder.
// `resolveTiles` returned a brand-new row object every recompute, so Vue saw
// every tile as changed and re-rendered the whole tile subtree - which re-runs
// all eight bindings - on every live-state tick, and the iframes tick twice a
// second each.
test('a row is reused when nothing about it changed', () => {
  const args = {
    selectedIds: ['a', 'b'],
    devices: [device('a'), device('b')],
    live: {
      a: { state: 'live', url: 'remote.html?device=a' },
      b: { state: 'live', url: 'remote.html?device=b' },
    },
    translate: (key) => key,
  }
  const first = resolveTiles(args)
  const second = resolveTiles(args)
  assert.equal(second[0], first[0], 'tile a was unchanged but got a new row')
  assert.equal(second[1], first[1], 'tile b was unchanged but got a new row')
})

// The counter is the one field that moves every tick, so it has to be applied
// even on a reused row - otherwise the number the operator watches goes stale.
test('a reused row still carries the advancing counter', () => {
  const args = {
    selectedIds: ['a'],
    devices: [device('a')],
    live: { a: { state: 'live', url: 'u', detail: '10' } },
    translate: (key) => key,
  }
  resolveTiles(args)
  args.live.a.detail = '11'
  const rows = resolveTiles(args)
  assert.equal(rows[0].detail, '11', 'the counter must stay live')
})

// A row whose state actually changed still has to be a new object, or the tile
// would never update. Reuse is an optimisation for unchanged rows, not a
// suppression of updates.
test('a row whose state changed is still a new object', () => {
  const args = {
    selectedIds: ['a'],
    devices: [device('a')],
    live: { a: { state: 'connecting', url: 'u' } },
    translate: (key) => key,
  }
  const before = resolveTiles(args)[0]
  args.live.a.state = 'live'
  const after = resolveTiles(args)[0]
  assert.notEqual(after, before)
  assert.equal(after.state, 'live')
})
