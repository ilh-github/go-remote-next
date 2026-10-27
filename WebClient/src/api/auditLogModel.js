// QD-909: reading the data-channel audit trail back.
//
// The write endpoints report metadata only - there is deliberately no field for
// the clipboard text, because a row that could say what was copied would turn
// the audit trail into a credential store every later reader of the database
// inherits. `details` comes back from the server as a JSON *string*, which is
// what keeps that property visible to a client: a typed field would hide it.
//
// This module is the read side's boundary. It parses that string and refuses to
// guess: a row whose details will not parse is surfaced as an unparsed row
// rather than dropped, because "this transfer happened and we cannot describe
// it" is a fact an investigator needs, and silently omitting the row would make
// the list look complete when it is not.

// One row per transfer, whichever way it went. The direction used to be part of
// the action name, which made a transfer reported with both directions produce
// two rows; it lives in the row's `directions` now.
export const AUDIT_ACTION_FILE = 'file.transfer'
export const AUDIT_ACTION_CLIPBOARD = 'clipboard.sync'

// The direction-scoped names stay valid filters, because "what did I upload" is
// a real question - they are just not row identities any more.
export const AUDIT_ACTION_FILE_UPLOAD = 'file.transfer.upload'
export const AUDIT_ACTION_FILE_DOWNLOAD = 'file.transfer.download'
export const AUDIT_ACTION_CLIPBOARD_UPLOAD = 'clipboard.sync.upload'
export const AUDIT_ACTION_CLIPBOARD_DOWNLOAD = 'clipboard.sync.download'

// Every action the server accepts as an `action` filter. Kept here so a client
// cannot send one the server will reject with a 400.
export const AUDIT_ACTIONS = [
  AUDIT_ACTION_FILE,
  AUDIT_ACTION_CLIPBOARD,
  AUDIT_ACTION_FILE_UPLOAD,
  AUDIT_ACTION_FILE_DOWNLOAD,
  AUDIT_ACTION_CLIPBOARD_UPLOAD,
  AUDIT_ACTION_CLIPBOARD_DOWNLOAD,
]

/**
 * @param {string} action
 * @returns {boolean} whether the server accepts this action as a filter
 */
export function isAuditAction(action) {
  return AUDIT_ACTIONS.includes(action)
}

/**
 * Parses one audit row into the shape a list renders.
 *
 * @param {object} row  one item from GET /v1/me/audit-logs
 * @returns {{
 *   id: number,
 *   action: string,
 *   kind: 'file'|'clipboard'|'unknown',
 *   direction: string,
 *   transferId: string,
 *   status: string,
 *   bytes: number|null,
 *   filename: string|null,
 *   mimeType: string|null,
 *   error: string|null,
 *   at: string,
 *   ip: string,
 *   detailsParsed: boolean,
 * }}
 */
export function describeAuditRow(row) {
  const action = typeof row?.action === 'string' ? row.action : ''
  const kind = action.startsWith('file.transfer')
    ? 'file'
    : action.startsWith('clipboard.sync')
      ? 'clipboard'
      : 'unknown'
  const direction = action.endsWith('.upload')
    ? 'upload'
    : action.endsWith('.download')
      ? 'download'
      : ''

  const base = {
    id: row?.id ?? 0,
    action,
    kind,
    direction,
    transferId: typeof row?.resource_id === 'string' ? row.resource_id : '',
    status: '',
    bytes: null,
    filename: null,
    mimeType: null,
    error: null,
    at: typeof row?.created_at === 'string' ? row.created_at : '',
    ip: typeof row?.ip === 'string' ? row.ip : '',
    detailsParsed: false,
  }

  // `details` is a JSON string. Anything else - an object from a caller that
  // already parsed it, or a missing field - is taken as-is when it is an object,
  // and reported as unparsed when it is neither.
  let details = null
  if (typeof row?.details === 'string') {
    try {
      const parsed = JSON.parse(row.details)
      details = parsed && typeof parsed === 'object' ? parsed : null
    } catch {
      details = null
    }
  } else if (row?.details && typeof row.details === 'object') {
    details = row.details
  }

  if (!details) return base

  return {
    ...base,
    detailsParsed: true,
    status: typeof details.status === 'string' ? details.status : '',
    bytes: Number.isFinite(details.bytes)
      ? details.bytes
      : Number.isFinite(details.bytes_transferred)
        ? details.bytes_transferred
        : null,
    filename: typeof details.filename === 'string' ? details.filename : null,
    mimeType: typeof details.mime_type === 'string' ? details.mime_type : null,
    error: typeof details.error === 'string' && details.error ? details.error : null,
  }
}

/**
 * Describes a page of rows, keeping the server's newest-first order.
 *
 * @param {Array<object>} items
 * @returns {Array<object>}
 */
export function describeAuditRows(items) {
  return Array.isArray(items) ? items.map(describeAuditRow) : []
}

/**
 * The one line a list row shows.
 *
 * Deliberately says "a transfer happened" and never what was in it - the same
 * boundary the server enforces, repeated here so the UI cannot be the thing that
 * leaks it.
 *
 * @param {object} described  a row from describeAuditRow
 * @returns {string}
 */
export function auditRowLabel(described) {
  if (!described) return ''
  const what = described.kind === 'file' ? 'File' : described.kind === 'clipboard' ? 'Clipboard' : 'Transfer'
  const where = described.direction === 'upload' ? 'to the host' : 'to the controller'
  const size = described.bytes === null ? '' : ` (${formatBytes(described.bytes)})`
  const name = described.filename ? ` ${described.filename}` : ''
  return `${what} ${described.direction || 'transfer'} ${where}${name}${size}`
}

function formatBytes(bytes) {
  if (!Number.isFinite(bytes) || bytes < 0) return ''
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}
