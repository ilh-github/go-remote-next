import assert from 'node:assert/strict'
import test from 'node:test'

import { addTrackOnce } from '../js/protocol/media-stream.js'

function fakeStream(initialTracks = []) {
  const tracks = initialTracks.slice()
  return {
    tracks,
    addTrack(track) {
      tracks.push(track)
    },
    getTracks() {
      return tracks
    },
  }
}

test('a late audio track is added to the stream already attached to the video', () => {
  const videoTrack = { id: 'video' }
  const audioTrack = { id: 'audio' }
  const stream = fakeStream([videoTrack])

  const added = addTrackOnce(stream, audioTrack)

  assert.equal(added, true)
  assert.deepEqual(stream.tracks, [videoTrack, audioTrack])
})

test('the same audio track is not merged twice', () => {
  const audioTrack = { id: 'audio' }
  const stream = fakeStream([audioTrack])

  const added = addTrackOnce(stream, audioTrack)

  assert.equal(added, false)
  assert.deepEqual(stream.tracks, [audioTrack])
})
