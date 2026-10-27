// The operator's audio decision, in one place.
//
// There are two things "turn the audio off" has to reach, and for a while only
// one of them did. The host stops sending audio when it receives
// `audioControl {enable: false}` - that half was always there. The local
// `<video>` element, which is what actually makes sound, was never touched: the
// operator clicked mute, the host stopped streaming, and whatever the element
// was already holding kept playing until the buffer ran out. On a quiet link
// that is a few seconds of a machine the operator just asked to be silent.
//
// The wall path already did both - `_setWallAudio` sends the control message and
// sets `video.muted`. The single-session path did one. This module is the shared
// half, so the two cannot drift apart again.

/**
 * Applies the operator's audio decision to the local element and reports what
 * the host should be told.
 *
 * @param {HTMLVideoElement|null} video the element the remote stream is attached to
 * @param {boolean} enabled whether the operator wants audio
 * @returns {{hostEnable: boolean}} what to send to the host
 */
export function applyAudioPreference(video, enabled) {
    // Muting the element is not optional even when the host is about to stop
    // sending: the element keeps playing whatever it already buffered, and the
    // operator asked for silence now, not for silence once the buffer drains.
    if (video) video.muted = !enabled
    return { hostEnable: enabled }
}

/**
 * Re-applies the audio preference after a stream is attached.
 *
 * A reconnect rebuilds the combined stream and assigns a fresh srcObject, which
 * resets `muted` to the element's default (false). Without this the operator's
 * mute does not survive a reconnect - the one moment the link is least stable
 * is exactly when the remote machine starts making noise again.
 *
 * @param {HTMLVideoElement|null} video
 * @param {boolean} enabled the operator's current preference
 */
export function reapplyAudioPreference(video, enabled) {
    if (video) video.muted = !enabled
}
