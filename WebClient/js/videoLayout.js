/**
 * videoLayout.js - which display a fresh video layout means.
 *
 * The host sends a layout listing every capturable display, and the client has to
 * answer "which one am I showing". Three inputs decide it, in order:
 *
 * 1. The stream the operator already selected. Switching monitors must survive a
 *    layout update, or every unrelated layout change yanks them back to the
 *    primary display.
 * 2. The host's primary screen, when nothing was selected before.
 * 3. The first entry, as a last resort.
 *
 * The empty case is the one worth being careful about. A host with no selectable
 * displays is a real state - every display can be detached, or the layout can
 * arrive before any per-display track exists - and the answer is "nothing", not
 * "keep whatever we had". Reporting the stale list as current leaves the display
 * strip showing monitors that no longer exist, and clicking one selects a stream
 * id the host will never send.
 */

/**
 * @param {Array<object>} displays per-display tracks, already filtered to those
 *   with a media stream id
 * @param {object} options
 * @param {string} [options.selectedStreamId] the stream currently selected
 * @param {number} [options.primaryScreenId]    the host's primary screen id
 * @returns {{track: object|null, index: number}}
 */
export function selectDisplayTrack(displays, options = {}) {
  if (!Array.isArray(displays) || displays.length === 0) {
    return { track: null, index: -1 }
  }

  const { selectedStreamId, primaryScreenId } = options

  if (selectedStreamId) {
    const kept = displays.findIndex((track) => track.mediaStreamId === selectedStreamId)
    if (kept >= 0) return { track: displays[kept], index: kept }
  }

  if (primaryScreenId !== undefined) {
    const primary = displays.findIndex((track) => track.screenId === primaryScreenId)
    if (primary >= 0) return { track: displays[primary], index: primary }
  }

  return { track: displays[0], index: 0 }
}

/**
 * Whether a layout that resolved to nothing should still update the display list.
 *
 * It should. An empty layout is the host saying "there are no displays right
 * now", and the display strip has to reflect that rather than keep showing the
 * previous set. Only a layout with no per-display information at all is a
 * non-event.
 *
 * @param {Array<object>} rawTracks the tracks as they arrived
 * @returns {boolean}
 */
export function shouldUpdateDisplayList(rawTracks) {
  return Array.isArray(rawTracks)
}
