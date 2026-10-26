/**
 * wall/stateLabel.js - QD-908: resolving a tile's state label once.
 *
 * The allocation profile for a 40-viewer soak put vue-i18n's message compiler at
 * the top of the wall's self-size allocation, and `stateLabel` was one of the
 * paths feeding it: it called `t()` from a template expression, so every tile
 * badge resolved its message again on every render, and a wall of hundreds of
 * tiles re-renders on every live-state tick.
 *
 * There are only a handful of tile states, so the mapping is worth resolving
 * once. The fallback behaviour is preserved exactly, because it is the part an
 * operator sees: a state with no message shows the raw state rather than the
 * message key.
 */

/**
 * Builds a state-label resolver that memoises per distinct state.
 *
 * @param {(key: string) => string} translate the i18n resolver
 * @returns {(state: string) => string}
 */
export function makeStateLabel(translate) {
  const t = typeof translate === 'function' ? translate : (key) => key
  const cache = new Map()
  return (state) => {
    const key = String(state)
    if (cache.has(key)) return cache.get(key)
    const message = t(`wall.state.${key}`)
    // The codebase's i18n contract: a missing message returns the key itself, so
    // the tile shows the state rather than an internal key.
    const label = message === `wall.state.${key}` ? key : message
    cache.set(key, label)
    return label
  }
}
