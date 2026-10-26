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
    const base = {
      deviceId: id,
      name: device ? (device.remark || device.device_name || id) : id,
      url: state.url || '',
      detail: state.detail || '',
      reason: '',
      state: 'connecting',
    }
    if (!device) {
      rows.push({ ...base, state: 'error', reason: t('wall.tileUnknown') })
      continue
    }
    if (!device.online) {
      rows.push({ ...base, url: '', state: 'offline', reason: t('wall.reasonOffline') })
      continue
    }
    if (!device.logged_in) {
      rows.push({ ...base, url: '', state: 'idle', reason: t('wall.reasonIdle') })
      continue
    }
    rows.push({ ...base, state: state.state || 'connecting', reason: state.reason || '' })
  }
  return rows
}
