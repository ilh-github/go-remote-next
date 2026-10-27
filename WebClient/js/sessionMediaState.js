/**
 * sessionMediaState.js - the media state that belongs to one session.
 *
 * A reconnect tears the session down and builds a new one, and three pieces of
 * state have to go with the old one rather than survive into the new:
 *
 * - the cached audio track, which is held separately from the video streams and
 *   re-attached to every combined stream built afterwards. A track from the old
 *   session that is still `live` will be merged into the new session's video, and
 *   the operator hears the previous machine's audio under the current picture.
 * - the map of received streams, keyed by the old session's stream ids. A new
 *   session issues its own ids, so a stale entry is unreachable at best and, when
 *   ids collide, attaches the wrong stream.
 * - the selected stream id, which names a stream the old session owned. Selecting
 *   a display after the reconnect resolves it against the new map and silently
 *   finds nothing.
 *
 * These live here rather than as bare fields on the controller because "which
 * fields are session-scoped" is the thing that has to be right, and a field added
 * later that nobody clears is exactly the bug. A test can drive this module; it
 * cannot easily drive the controller.
 */

/**
 * Holds the media state for the session currently connected.
 */
export class SessionMediaState {
  constructor() {
    this.reset()
  }

  /** Drops everything that belonged to the previous session. */
  reset() {
    this.audioTrack = null
    this.receivedStreams = new Map()
    this.selectedStreamId = ''
  }

  /**
   * @param {MediaStreamTrack|null} track the host's audio track, if it has sent one
   */
  setAudioTrack(track) {
    this.audioTrack = track || null
  }

  /**
   * The audio track, but only when it is still usable. A track whose readyState
   * has moved on belongs to a session that has ended, and merging it would put
   * the old session's audio under the new session's video.
 *
   * @returns {MediaStreamTrack|null}
   */
  usableAudioTrack() {
    const track = this.audioTrack
    if (!track) return null
    if (track.readyState !== 'live') return null
    return track
  }

  /**
   * @param {string} streamId id of the MediaStream a video track arrived on
   * @param {MediaStream} stream
   */
  addStream(streamId, stream) {
    if (!streamId || !stream) return
    this.receivedStreams.set(streamId, stream)
  }

  /**
   * @param {string} streamId
   * @returns {MediaStream|null}
   */
  streamFor(streamId) {
    if (!streamId) return null
    return this.receivedStreams.get(streamId) || null
  }

  /**
   * @param {string} streamId
   */
  selectStream(streamId) {
    this.selectedStreamId = streamId || ''
  }

  /**
   * The stream currently selected, resolved against the streams this session has
   * actually received. An id left over from a previous session resolves to null
   * rather than to a stream that no longer belongs here.
   *
   * @returns {MediaStream|null}
   */
  selectedStream() {
    return this.streamFor(this.selectedStreamId)
  }
}
