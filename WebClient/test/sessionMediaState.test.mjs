import assert from 'node:assert/strict'
import test from 'node:test'

import { SessionMediaState } from '../js/sessionMediaState.js'

function fakeStream(id) {
  return { id, getVideoTracks: () => [{ id: `${id}-video` }] }
}

// A track that is still live when the session ends belongs to the machine the
// operator was just on. Merging it into the next session's video puts the old
// machine's audio under the new picture.
test('a live audio track from the previous session is not carried into the next one', () => {
  const state = new SessionMediaState()
  state.setAudioTrack({ id: 'audio-1', readyState: 'live' })

  state.reset()

  assert.equal(state.usableAudioTrack(), null,
    'the previous session audio track survived the reset')
})

test('a track whose readyState has moved on is not usable', () => {
  const state = new SessionMediaState()

  for (const readyState of ['ended', 'muted', 'inactive']) {
    state.setAudioTrack({ id: `audio-${readyState}`, readyState })
    assert.equal(state.usableAudioTrack(), null, `a ${readyState} track was treated as usable`)
  }
})

test('a live audio track is usable', () => {
  const state = new SessionMediaState()
  const track = { id: 'audio-1', readyState: 'live' }
  state.setAudioTrack(track)

  assert.equal(state.usableAudioTrack(), track)
})

// Stream ids are issued per session, so a map entry from the old session is
// unreachable at best. When ids do collide it attaches the wrong stream, which is
// worse than attaching nothing.
test('streams from the previous session do not resolve after a reset', () => {
  const state = new SessionMediaState()
  state.addStream('stream-a', fakeStream('stream-a'))
  state.selectStream('stream-a')

  state.reset()

  assert.equal(state.streamFor('stream-a'), null,
    'a stream from the previous session still resolves')
  assert.equal(state.selectedStream(), null,
    'the selected stream still resolves after the reset')
})

test('a selected id that this session never received resolves to null', () => {
  const state = new SessionMediaState()
  state.addStream('stream-a', fakeStream('stream-a'))

  // An id left over from the previous session.
  state.selectStream('stream-from-before')

  assert.equal(state.selectedStream(), null,
    'a stale selected id resolved to a stream')
})

test('streams received in this session resolve normally', () => {
  const state = new SessionMediaState()
  const stream = fakeStream('stream-a')
  state.addStream('stream-a', stream)
  state.selectStream('stream-a')

  assert.equal(state.streamFor('stream-a'), stream)
  assert.equal(state.selectedStream(), stream)
})

// An empty or missing id is the "nothing selected yet" case, not a lookup that
// should accidentally match a stream stored under a falsy key.
test('an empty stream id selects and resolves nothing', () => {
  const state = new SessionMediaState()

  state.selectStream('')
  state.addStream('', fakeStream('empty'))

  assert.equal(state.selectedStream(), null)
  assert.equal(state.streamFor(''), null)
})
