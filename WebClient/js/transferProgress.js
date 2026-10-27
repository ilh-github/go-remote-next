/**
 * transferProgress.js - how far along a file transfer is, from progress events.
 *
 * Progress events carry `bytesSent` and `totalBytes`, but not reliably: the host
 * is free to omit the total on a later event once it has been announced at the
 * start. The audit state already keeps the last known total for exactly that
 * reason, while the progress readout used the raw event value - so a transfer
 * that had reached 80% was redrawn at 0% by the very next event, and the operator
 * watched the bar jump backwards for the rest of the upload.
 *
 * Two rules fall out of that:
 *
 * 1. The total is remembered, and a missing or zero one never undoes what was
 *    already known.
 * 2. "This transfer is empty" and "this event did not say" both arrive as 0, and
 *    they mean opposite things - the first is complete, the second is unknown.
 *    They are told apart by when the zero arrived: `announceTotal` is the start
 *    of the transfer, where a zero is a statement about the file.
 */

export class TransferProgress {
  constructor() {
    this.bytesSent = 0
    this.totalBytes = 0
    // Whether the total has ever been announced. Without this, an empty transfer
    // and an unknown one are the same state and cannot be reported differently.
    this.totalKnown = false
  }

  /**
   * Records the total announced when the transfer started, including zero.
   *
   * @param {number} totalBytes
   */
  announceTotal(totalBytes) {
    const total = Number(totalBytes)
    if (!Number.isFinite(total) || total < 0) return
    this.totalBytes = total
    this.totalKnown = true
  }

  /**
   * Applies one progress event.
   *
   * @param {number} bytesSent how many bytes have crossed
   * @param {number} [totalBytes] the total, when the event carries a usable one
   */
  update(bytesSent, totalBytes) {
    const sent = Number(bytesSent)
    this.bytesSent = Number.isFinite(sent) && sent > 0 ? sent : 0

    // Only a positive total replaces what is known. A zero here is the event
    // declining to say, not the transfer becoming empty.
    const total = Number(totalBytes)
    if (Number.isFinite(total) && total > 0) {
      this.totalBytes = total
      this.totalKnown = true
    }
  }

  /**
   * Percentage complete, 0-100.
   *
   * A transfer that was announced as empty is complete, because there is nothing
   * left to send. One whose total has never been announced reports 0 rather than
   * a guess.
   *
   * @returns {number}
   */
  percent() {
    if (this.totalBytes <= 0) {
      return this.totalKnown ? 100 : 0
    }
    const pct = Math.round((this.bytesSent / this.totalBytes) * 100)
    return Math.max(0, Math.min(100, pct))
  }

  /** Whether every byte has crossed. */
  get isComplete() {
    return this.totalKnown && this.bytesSent >= this.totalBytes
  }
}

/**
 * The remembered total, for callers that keep their own state alongside the
 * readout and must not be told a smaller one than they already have.
 *
 * @param {number} known    the total held so far
 * @param {number} incoming the total from the event
 * @returns {number}
 */
export function reconcileTotal(known, incoming) {
  const value = Number(incoming)
  if (Number.isFinite(value) && value > 0) return value
  const held = Number(known)
  return Number.isFinite(held) && held > 0 ? held : 0
}
