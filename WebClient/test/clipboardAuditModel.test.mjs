import assert from 'node:assert/strict'
import test from 'node:test'

const {
    buildClipboardAuditReport,
    CLIPBOARD_AUDIT_UPLOAD,
    CLIPBOARD_AUDIT_DOWNLOAD,
} = await import('../src/api/clipboardAuditModel.js')

// QD-909. The clipboard is the one payload whose contents are routinely a
// password, a recovery code or a private message, so the report has to be able
// to say "a transfer happened" without being able to say what it was. These
// tests pin that the text never reaches the object that gets sent.
test('the report never carries the clipboard text', () => {
    const secret = 'correct-horse-battery-staple-9f3a'
    const report = buildClipboardAuditReport({
        deviceId: '123456789',
        transferId: 'clip-1',
        direction: CLIPBOARD_AUDIT_UPLOAD,
        status: 'completed',
        mimeType: 'text/plain',
        text: secret,
    })

    assert.deepEqual(Object.keys(report).sort(), [
        'bytes', 'deviceId', 'direction', 'mimeType', 'status', 'transferId',
    ])
    assert.equal(JSON.stringify(report).includes(secret), false)
    assert.equal(report.bytes, secret.length)
})

test('direction constants map to the server vocabulary', () => {
    assert.equal(CLIPBOARD_AUDIT_UPLOAD, 'upload')
    assert.equal(CLIPBOARD_AUDIT_DOWNLOAD, 'download')
})

test('bytes is the encoded size, not the character count', () => {
    // A non-ASCII clipboard is longer in bytes than in characters, and the
    // number the audit row carries is meant to describe what crossed the wire.
    const text = '密码'
    const report = buildClipboardAuditReport({
        deviceId: '123456789',
        transferId: 'clip-2',
        direction: CLIPBOARD_AUDIT_DOWNLOAD,
        status: 'completed',
        mimeType: 'text/plain',
        text,
    })
    assert.equal(report.bytes, new TextEncoder().encode(text).length)
    assert.ok(report.bytes > text.length)
})

// A report with nothing to identify it is not a report: the server would reject
    // it, and sending it would only burn a round trip.
test('an unusable report is refused locally', () => {
    for (const bad of [
        { deviceId: '', transferId: 'c', direction: 'upload', status: 'completed', mimeType: 'text/plain', text: 'x' },
        { deviceId: 'd', transferId: '', direction: 'upload', status: 'completed', mimeType: 'text/plain', text: 'x' },
        { deviceId: 'd', transferId: 'c', direction: '', status: 'completed', mimeType: 'text/plain', text: 'x' },
        { deviceId: 'd', transferId: 'c', direction: 'upload', status: '', mimeType: 'text/plain', text: 'x' },
        { deviceId: 'd', transferId: 'c', direction: 'upload', status: 'completed', mimeType: '', text: 'x' },
    ]) {
        assert.equal(buildClipboardAuditReport(bad), null, `expected null for ${JSON.stringify(bad)}`)
    }
})

test('an empty clipboard still produces a valid report', () => {
    const report = buildClipboardAuditReport({
        deviceId: '123456789',
        transferId: 'clip-empty',
        direction: CLIPBOARD_AUDIT_UPLOAD,
        status: 'completed',
        mimeType: 'text/plain',
        text: '',
    })
    assert.equal(report.bytes, 0)
    assert.equal(report.direction, 'upload')
})

test('a failed sync is reported with its status', () => {
    const report = buildClipboardAuditReport({
        deviceId: '123456789',
        transferId: 'clip-failed',
        direction: CLIPBOARD_AUDIT_DOWNLOAD,
        status: 'failed',
        mimeType: 'text/plain',
        text: 'whatever was in flight',
    })
    assert.equal(report.status, 'failed')
    assert.equal(report.direction, 'download')
})
