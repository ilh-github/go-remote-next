/**
 * files/statusLine.js - which message the file manager's status line shows.
 *
 * Three writers target one line: upload progress, the busy/loading lifecycle,
 * and the listing that follows an operation. They arrive in that order, so the
 * last one wins by default - and the loser is the progress readout. A
 * multi-chunk upload reports 100%, then the post-upload refresh reports busy
 * ("loading"), then the listing reports the item count, and the operator never
 * sees confirmation that the file finished transferring. The item count that
 * replaced it says how many rows are listed, not that anything arrived.
 *
 * The rule is that a message announces what it is *about*, and a later message
 * only displaces an earlier one when it is about the same subject. Progress is
 * about a transfer; loading and item counts are about the listing. They do not
 * overwrite each other, and the transfer's final state is what stays.
 */

export const StatusSubject = {
  /** A transfer is in flight or has just finished. */
  Transfer: 'transfer',
  /** The listing is being read. */
  Listing: 'listing',
  /**
   * An operation on one entry: created, renamed, trashed. It is neither a
   * transfer nor the listing, so it displaces either - but it is its own
   * subject rather than `Other`, because two of them in a row are about
   * different entries and the newer one is the news.
   */
  Entry: 'entry',
  /** Anything else: an error, a one-off confirmation, an idle hint. */
  Other: 'other',
}

/**
 * Decides the subject of a status message from the key that produced it.
 *
 * The panel already knows which key it is rendering; passing it in is what lets
 * the rule be stated once instead of at each call site.
 *
 * @param {string} key an i18n key such as 'files.uploading' or 'files.loading'
 * @returns {string} one of StatusSubject
 */
export function subjectForKey(key) {
  const name = String(key || '')
  if (name.startsWith('files.uploading')) return StatusSubject.Transfer
  if (name.startsWith('files.downloading')) return StatusSubject.Transfer
  if (name.startsWith('files.uploaded')) return StatusSubject.Transfer
  if (name.startsWith('files.downloaded')) return StatusSubject.Transfer
  if (name.startsWith('files.loading')) return StatusSubject.Listing
  if (name.startsWith('files.itemCount')) return StatusSubject.Listing
  if (name.startsWith('files.created')) return StatusSubject.Entry
  if (name.startsWith('files.renamed')) return StatusSubject.Entry
  if (name.startsWith('files.trashed')) return StatusSubject.Entry
  return StatusSubject.Other
}

/**
 * A status line that keeps a transfer's readout until another transfer
 * displaces it.
 *
 * Listing messages do not clear a transfer readout, because the refresh that
 * produces them is part of finishing the transfer rather than a separate thing
 * the operator needs to be told about instead.
 */
export class StatusLine {
  constructor() {
    this._message = ''
    this._subject = null
    this._error = false
  }

  get message() {
    return this._message
  }

  get error() {
    return this._error
  }

  /**
   * @param {string} message the rendered text
   * @param {string} key     the i18n key it came from, used to place it
   * @param {boolean} isError
   */
  set(message, key, isError = false) {
    const subject = subjectForKey(key)
    this._message = message
    this._subject = subject
    this._error = isError
  }

  /**
   * Whether a message with this key may replace what is currently shown.
   *
   * @param {string} key
   * @returns {boolean}
   */
  accepts(key) {
    const incoming = subjectForKey(key)
    // A listing message does not displace a transfer's readout: the refresh that
    // produces it is part of finishing that transfer, not separate news.
    if (incoming === StatusSubject.Listing && this._subject === StatusSubject.Transfer) {
      return false
    }
    // The same reasoning covers an entry confirmation. Renaming a file in
    // another directory is not news about the transfer that just finished, and
    // letting it through is what erased the "100%" the operator was reading.
    if (incoming === StatusSubject.Entry && this._subject === StatusSubject.Transfer) {
      return false
    }
    return true
  }
}
