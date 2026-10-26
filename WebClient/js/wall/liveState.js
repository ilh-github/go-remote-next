/**
 * wall/liveState.js - QD-908: applying a live-state message without a re-render.
 *
 * A 40-viewer 90-second soak attributed 29.6% of sampled allocation to
 * `compileMessageFormat` in vue-i18n, and the heap series carried a decisive
 * 2008ms period. Neither depends on the frame loop, which is what pointed here:
 * every iframe posts its frame count every 500ms, and the wall answered each
 * post by replacing the whole `live[deviceId]` object. A replaced object is a
 * re-render, and a tile template carries eight `$t()` bindings that each compile
 * an i18n message - so a counter that exists to be displayed was re-running the
 * message compiler for the whole wall, twice a second.
 *
 * Merging into the existing entry instead of replacing it keeps the counter
 * live without making every tick a render. The module is separate from the
 * component because "does a tick cause a re-render" is exactly the property
 * worth testing, and it is invisible in a page of hundreds of tiles.
 */

/**
 * Applies one `quickdesk-wall` message to the live-state map.
 *
 * Returns whether the entry changed identity, which is what a caller uses to
 * decide whether anything is owed a re-render. A message whose only difference
 * is the frame counter mutates the counter in place and reports no change: the
 * number still advances, it just does not take the tile with it.
 *
 * @param {Record<string, object>} live per-device live state, mutated in place
 * @param {object} message the postMessage payload
 * @param {string[]} [selectedIds] when given, only these devices are recorded
 * @returns {boolean} true when the entry was created or replaced
 */
export function mergeLiveState(live, message, selectedIds) {
  if (!live || typeof live !== 'object') return false
  if (!message || typeof message !== 'object') return false
  const device = typeof message.device === 'string' ? message.device : ''
  if (!device) return false
  if (Array.isArray(selectedIds) && !selectedIds.includes(device)) return false

  const state = typeof message.state === 'string' ? message.state : ''
  const detail = typeof message.detail === 'string' ? message.detail : ''
  const url = typeof message.url === 'string' ? message.url : ''
  const reason = typeof message.reason === 'string' ? message.reason : ''

  const existing = live[device]
  if (!existing) {
    if (!state && !detail && !url && !reason) return false
    live[device] = {
      state: state || 'connecting',
      detail,
      url,
      reason,
    }
    return true
  }

  // A change is anything other than the frame counter. Everything else - the
  // session state, the reason a tile is in it, the url it points at - is visible
  // in the template and has to produce a new object for the tile to update.
  const visibleChange =
    (state && state !== existing.state) ||
    (reason && reason !== existing.reason) ||
    (url && url !== existing.url)
  if (visibleChange) {
    live[device] = {
      ...existing,
      state: state || existing.state,
      detail: detail || existing.detail,
      url: url || existing.url,
      reason: reason || existing.reason,
    }
    return true
  }

  // Nothing visible changed. The counter still advances, in place: it is the
  // one field whose job is to move on every tick.
  if (detail && detail !== existing.detail) {
    existing.detail = detail
  }
  return false
}
