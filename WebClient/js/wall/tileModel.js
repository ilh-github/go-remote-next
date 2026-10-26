/**
 * wall/tileModel.js - resolving the wall's tile rows from devices and live state.
 *
 * The lookup used to be `devices.find(...)` inside a `computed` that mapped over
 * every selected device, which is quadratic in the number of tiles: 200 viewers
 * meant 40,000 comparisons per recompute, and the recompute ran on every device
 * poll and every live-state update. A wall is a grid of hundreds of tiles by
 * design, so the quadratic term is the common case rather than an edge.
 *
 * Indexing the devices once turns the same work into one pass, which is what
 * this module does. It is kept out of the component because the cost of the
 * lookup is the part worth testing, and a function buried in a Vue `computed`
 * can only be measured by rendering hundreds of tiles.
 */

/**
 * Indexes devices by id so a tile can find its device without scanning.
 *
 * A plain object rather than a Map because the keys are the server's device ids,
 * which are fixed-format strings - so there is no prototype-collision concern to
 * design around, and an object is what the rest of this codebase passes around.
 *
 * @param {Array<object>} devices
 * @returns {Record<string, object>}
 */
export function indexDevices(devices) {
  const index = {}
  if (!Array.isArray(devices)) return index
  for (const device of devices) {
    if (device && typeof device.device_id === 'string') {
      index[device.device_id] = device
    }
  }
  return index
}

/**
 * Builds the tile rows for one render.
 *
 * `live` holds per-device session state reported back over postMessage; it is
 * read-only here so the same state object can be shared across recomputes
 * without this function copying it.
 *
 * @param {object} args
 * @param {string[]} args.selectedIds  device ids to show, in display order
 * @param {Array<object>} args.devices the caller's devices
 * @param {Record<string, object>} [args.live] per-device live state
 * @param {(key: string) => string} [args.translate] message resolver
 * @returns {Array<object>} one row per selected id, in the same order
 */
export function resolveTiles({ selectedIds, devices, live = {}, translate }) {
  const t = typeof translate === 'function' ? translate : (key) => key
  const index = indexDevices(devices)
  const rows = []
  for (const id of selectedIds) {
    const device = index[id]
    const state = live[id] || {}
    // The previous row is reused when nothing visible about it changed. This is
    // not a micro-optimisation: the allocation profile named vue-i18n's
    // `compileMessageFormat` as the top self-size path, and a tile template
    // carries eight `$t()` bindings. A brand-new row object every recompute made
    // Vue re-render the whole tile subtree - re-running all eight - on every
    // live-state tick, and each iframe ticks twice a second.
    const previous = previousRows.get(id)
    // Resolved first, compared second. Comparing the half-built row would
    // compare defaults against resolved values and report every tile as
    // changed, which is the very thing being fixed.
    const resolved = resolveOneRow({ id, device, state, t })
    const base = {
      deviceId: id,
      name: device ? (device.remark || device.device_name || id) : id,
      url: state.url || '',
      detail: state.detail || '',
      reason: '',
      state: 'connecting',
    }
    // The counter is the one field whose job is to move on every tick, so it is
    // applied even when the row is reused - otherwise the number an operator
    // watches goes stale while everything else correctly holds still.
    const next = previous && sameRow(previous, resolved)
      ? Object.assign(previous, { detail: resolved.detail })
      : resolved
    previousRows.set(id, next)
    rows.push(next)
  }
  return rows
}

// The row a tile renders, resolved from the device and its live state. Split out
// so the reuse decision compares two finished rows rather than a finished one
// against a half-built one.
function resolveOneRow({ id, device, state, t }) {
  const base = {
    deviceId: id,
    name: device ? (device.remark || device.device_name || id) : id,
    url: state.url || '',
    detail: state.detail || '',
    reason: '',
    state: 'connecting',
  }
  if (!device) return { ...base, state: 'error', reason: t('wall.tileUnknown') }
  if (!device.online) return { ...base, url: '', state: 'offline', reason: t('wall.reasonOffline') }
  if (!device.logged_in) return { ...base, url: '', state: 'idle', reason: t('wall.reasonIdle') }
  return { ...base, state: state.state || 'connecting', reason: state.reason || '' }
}

// Rows are cached per device id so a recompute can tell "unchanged" from
// "changed" without the caller holding on to the previous array. A Map rather
// than a field on the row because the row is what the caller renders, and it
// must stay a plain object.
const previousRows = new Map()

// Everything a row shows other than the counter. The counter is excluded on
// purpose: it is expected to differ on every tick, and including it would make
// every row look changed.
function sameRow(previous, base) {
  return previous.deviceId === base.deviceId
    && previous.name === base.name
    && previous.url === base.url
    && previous.state === base.state
    && previous.reason === base.reason
}
