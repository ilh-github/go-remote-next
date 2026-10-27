import assert from 'node:assert/strict'
import test from 'node:test'

import { selectDisplayTrack, shouldUpdateDisplayList } from '../js/videoLayout.js'

function track(mediaStreamId, extra = {}) {
  return { mediaStreamId, width: 1920, height: 1080, ...extra }
}

const displays = [
  track('stream-a', { screenId: 1 }),
  track('stream-b', { screenId: 2 }),
  track('stream-c', { screenId: 3 }),
]

// Switching monitors must survive a layout update, or every unrelated change
// yanks the operator back to the primary display.
test('the selected stream survives a layout update', () => {
  const result = selectDisplayTrack(displays, { selectedStreamId: 'stream-b', primaryScreenId: 1 })

  assert.equal(result.index, 1)
  assert.equal(result.track.mediaStreamId, 'stream-b')
})

test('the primary screen is used when nothing was selected', () => {
  const result = selectDisplayTrack(displays, { primaryScreenId: 3 })

  assert.equal(result.index, 2)
  assert.equal(result.track.mediaStreamId, 'stream-c')
})

test('the first display is the last resort', () => {
  const result = selectDisplayTrack(displays, {})

  assert.equal(result.index, 0)
  assert.equal(result.track.mediaStreamId, 'stream-a')
})

// A selected stream that the new layout no longer contains must fall through to
// the primary rather than silently showing nothing: the monitor was unplugged or
// re-created, and the operator is still looking at the machine.
test('a selected stream missing from the new layout falls through', () => {
  const result = selectDisplayTrack(displays, { selectedStreamId: 'stream-gone', primaryScreenId: 2 })

  assert.equal(result.index, 1)
  assert.equal(result.track.mediaStreamId, 'stream-b')
})

// The empty case is a real state, and the answer is "nothing" rather than "keep
// what we had": reporting the stale list as current leaves the display strip
// showing monitors that no longer exist.
test('an empty layout resolves to nothing', () => {
  const result = selectDisplayTrack([], { selectedStreamId: 'stream-a', primaryScreenId: 1 })

  assert.equal(result.track, null)
  assert.equal(result.index, -1)
})

test('a non-array layout resolves to nothing', () => {
  assert.equal(selectDisplayTrack(undefined, {}).track, null)
  assert.equal(selectDisplayTrack(null, {}).track, null)
})

test('an empty layout still updates the display list', () => {
  assert.equal(shouldUpdateDisplayList([]), true,
    'an empty layout was treated as a non-event, leaving the stale list showing')
})

test('a layout with no track information is a non-event', () => {
  assert.equal(shouldUpdateDisplayList(undefined), false)
  assert.equal(shouldUpdateDisplayList(null), false)
})
