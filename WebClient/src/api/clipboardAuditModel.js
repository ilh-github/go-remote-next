// QD-909: the reporting half of clipboard auditing.
//
// Clipboard bytes travel peer to peer over the Remoting data channel, so the
// control plane can only record what a controller tells it afterwards. That is
// acceptable for attribution, and it is why this module exists as a builder
// rather than letting callers assemble the body inline: the one thing that must
// never be assemblable is a report carrying the clipboard text.
//
// A clipboard is by definition whatever the user last copied - routinely a
// password, a recovery code or a private message. So the report takes the text
// only to measure it, and returns an object with no field that could hold it.

export const CLIPBOARD_AUDIT_UPLOAD = 'upload'
export const CLIPBOARD_AUDIT_DOWNLOAD = 'download'

const DIRECTIONS = [CLIPBOARD_AUDIT_UPLOAD, CLIPBOARD_AUDIT_DOWNLOAD]

/**
 * Builds the report body, or null when it is missing something the server would
 * reject. Refusing locally is deliberate: a report with no device, no transfer
 * id or no MIME type describes nothing, and sending it only burns a round trip
 * and a 400.
 *
 * @param {object} input
 * @param {string} input.deviceId
 * @param {string} input.transferId  stable id that makes retries idempotent
 * @param {string} input.direction   'upload' | 'download'
 * @param {string} input.status      'completed' | 'failed'
 * @param {string} input.mimeType
 * @param {string} input.text        the clipboard text; measured, never returned
 * @returns {object|null}
 */
export function buildClipboardAuditReport({
    deviceId = '',
    transferId = '',
    direction = '',
    status = '',
    mimeType = '',
    text = '',
} = {}) {
    const clean = (value) => (typeof value === 'string' ? value.trim() : '')

    const device = clean(deviceId)
    const id = clean(transferId)
    const way = clean(direction)
    const state = clean(status)
    const mime = clean(mimeType)

    if (!device || !id || !way || !state || !mime) return null
    if (!DIRECTIONS.includes(way)) return null
    if (state !== 'completed' && state !== 'failed') return null

    return {
        deviceId: device,
        transferId: id,
        direction: way,
        status: state,
        mimeType: mime,
        // Byte length, not character length: the number should describe what
        // crossed the data channel, and a non-ASCII clipboard is longer in bytes
        // than in characters.
        bytes: typeof text === 'string' ? new TextEncoder().encode(text).length : 0,
    }
}
