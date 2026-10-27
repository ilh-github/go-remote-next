import assert from 'node:assert/strict'
import test from 'node:test'

// QD-906: the audio boundary. The host stops sending when it is told to, but
// the element that actually makes sound was never told - so an operator who
// clicked mute kept hearing the remote machine until the buffer drained.
const { applyAudioPreference, reapplyAudioPreference } = await import('../js/session/audio-state.js')

function fakeVideo() {
    return { muted: false }
}

test('muting reaches the element, not just the host', () => {
    const video = fakeVideo()
    const { hostEnable } = applyAudioPreference(video, false)

    assert.equal(video.muted, true,
        'the host was told to stop but the element kept playing its buffer')
    assert.equal(hostEnable, false, 'the host was not told to stop')
})

test('unmuting reaches the element too', () => {
    const video = fakeVideo()
    video.muted = true
    const { hostEnable } = applyAudioPreference(video, true)

    assert.equal(video.muted, false, 'the element stayed muted after unmuting')
    assert.equal(hostEnable, true, 'the host was not told to resume')
})

test('a missing element is not a crash', () => {
    // The toggle can fire before the video element exists, and a throw there
    // would take the whole handler down rather than just skipping the mute.
    assert.doesNotThrow(() => applyAudioPreference(null, false))
    assert.deepEqual(applyAudioPreference(null, false), { hostEnable: false })
})

// A reconnect assigns a fresh srcObject, which resets `muted` to the element's
// default. The operator's mute has to be re-applied, or the one moment the link
// is least stable is when the remote machine starts making noise again.
test('a reconnect does not lose the mute', () => {
    const video = fakeVideo()
    applyAudioPreference(video, false)
    assert.equal(video.muted, true)

    // A new stream is attached; the browser resets `muted`.
    video.muted = false
    reapplyAudioPreference(video, false)

    assert.equal(video.muted, true, 'the mute did not survive the reconnect')
})

test('a reconnect keeps audio on when the operator wanted it', () => {
    const video = fakeVideo()
    applyAudioPreference(video, true)

    video.muted = true
    reapplyAudioPreference(video, true)

    assert.equal(video.muted, false, 'a reconnect muted audio the operator asked for')
})

// The module test above proves the rule; this one proves remote-main.js obeys
// it. A correct helper nobody calls is the same as no helper, and the wiring is
// the part that rots.
test('the remote page mutes the element on toggle and restores the mute on a new stream', async () => {
    const source = await import('node:fs/promises').then(fs =>
        fs.readFile(new URL('../js/remote-main.js', import.meta.url), 'utf8'))

    // Strip the import lines first. Matching a bare call name would still match
    // the import after the call site is removed - the false green this guard
    // exists to prevent, and the one it produced on its first version.
    const body = source.replace(/^import .*$/gm, '')

    assert.match(body, /applyAudioPreference\(\s*\n?\s*document\.getElementById\('remoteVideo'\)/,
        'the audio toggle does not apply the preference to the element')
    assert.match(body, /reapplyAudioPreference\(video,/,
        'a new stream does not restore the operator audio preference')

    // The preference has to be read from the toolbar, and a missing toolbar must
    // not read as "muted" - the default is audio on.
    assert.match(body, /audioEnabled !== false/,
        'the restored preference treats an absent toolbar as muted')
})
