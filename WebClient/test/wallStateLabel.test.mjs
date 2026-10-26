import assert from 'node:assert/strict'
import test from 'node:test'

// QD-908: `stateLabel` called `t()` on every render of every tile badge, and the
// allocation profile put vue-i18n's message compiler at the top of the wall's
// self-size allocation. There are only a handful of tile states, so the mapping
// is worth resolving once per state rather than once per render per tile.
//
// The behaviour being preserved is the fallback: a state with no message shows
// the raw state, because a tile that says "wall.state.weird" is worse than one
// that says "weird".

const { makeStateLabel } = await import('../js/wall/stateLabel.js')

test('a known state resolves to its message', () => {
  const label = makeStateLabel((key) => `msg:${key}`)
  assert.equal(label('live'), 'msg:wall.state.live')
  assert.equal(label('offline'), 'msg:wall.state.offline')
})

// The i18n contract in this codebase: `t()` returns the key itself when there is
// no message, and the tile must show the state rather than the key.
test('an unknown state falls back to the raw state', () => {
  const label = makeStateLabel((key) => key)
  assert.equal(label('connecting'), 'connecting')
  assert.equal(label('something-new'), 'something-new')
})

// The point of the cache: the resolver runs once per distinct state, not once
// per call. A wall of 200 tiles re-renders constantly, and the resolver is the
// expensive half.
test('the resolver runs once per state, not once per call', () => {
  let calls = 0
  const label = makeStateLabel((key) => { calls += 1; return `msg:${key}` })
  for (let i = 0; i < 200; i += 1) label('live')
  assert.equal(calls, 1, `200 renders ran the resolver ${calls} times`)
})

// A locale change has to invalidate the cache, or the wall would keep showing the
// old language. The caller drops the cache by rebuilding the resolver, which is
// what the third argument is for.
test('a different locale rebuilds the mapping', () => {
  let calls = 0
  const make = (locale) => makeStateLabel((key) => { calls += 1; return `${locale}:${key}` })
  const en = make('en')
  assert.equal(en('live'), 'en:wall.state.live')
  const zh = make('zh')
  assert.equal(zh('live'), 'zh:wall.state.live')
  assert.equal(calls, 2, 'each locale resolves its own mapping')
})
