/**
 * ui/originalResolution.js - which desktop size the "original" menu entry means.
 *
 * The floating toolbar's resolution menu has an entry that restores the size the
 * operator is currently looking at, and it was built from the first resolution the
 * toolbar was told about. That is right for the first monitor and wrong for every
 * one after it: switching monitors replaces the stream, the new monitor usually
 * has a different native size, and the entry keeps offering the first monitor's
 * geometry. Restoring it then resizes the desktop away from what the video shows.
 *
 * The rule is that "original" tracks the monitor being viewed, so it lives here
 * rather than inside the toolbar: the rule is what is worth testing, and the
 * toolbar can only be exercised by building the whole menu.
 */

export class OriginalResolutionTracker {
  constructor() {
    this._width = 0
    this._height = 0
  }

  get width() {
    return this._width
  }

  get height() {
    return this._height
  }

  /**
   * @param {number} width  width of the desktop now being viewed
   * @param {number} height height of the desktop now being viewed
   */
  observe(width, height) {
    if (!(width > 0) || !(height > 0)) return
    this._width = width
    this._height = height
  }

  /**
   * The label the menu shows next to the entry, so the operator can see what
   * restoring will do before doing it.
   *
   * @returns {string} e.g. "1920 × 1080", or an empty string when nothing is known
   */
  label() {
    if (!(this._width > 0) || !(this._height > 0)) return ''
    return `${this._width} × ${this._height}`
  }
}
