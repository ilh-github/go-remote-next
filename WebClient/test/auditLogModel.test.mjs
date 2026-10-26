import assert from 'node:assert/strict'
import test from 'node:test'

// QD-909, read side. The server stores `details` as a JSON string and never
// stores the clipboard text; these tests pin that the client side of the same
// boundary holds, including the case that is easiest to get wrong - a row whose
// details will not parse must stay visible rather than disappear.

const {
  describeAuditRow,
  describeAuditRows,
  auditRowLabel,
  isAuditAction,
  AUDIT_ACTIONS,
} = await import('../src/api/auditLogModel.js')

test('a clipboard row is described without ever exposing its text', () => {
  const described = describeAuditRow({
    id: 7,
    action: 'clipboard.sync.upload',
    resource_type: 'clipboard_sync',
    resource_id: 'clip-42',
    details: JSON.stringify({
      device_id: '123456789',
      status: 'completed',
      mime_type: 'text/plain',
      bytes: 42,
    }),
    ip: '203.0.113.9',
    created_at: '2026-10-26T10:00:00Z',
  })

  assert.equal(described.kind, 'clipboard')
  assert.equal(described.direction, 'upload')
  assert.equal(described.transferId, 'clip-42')
  assert.equal(described.status, 'completed')
  assert.equal(described.bytes, 42)
  assert.equal(described.mimeType, 'text/plain')
  assert.equal(described.detailsParsed, true)
  // The contract has no field that could hold the text, so the described row
  // has nowhere to put it. Asserted by shape rather than by substring: 'text'
  // appears legitimately inside the mime type `text/plain`, so a naive
  // "does not contain 'text'" check would fail on a correct row.
  assert.deepEqual(Object.keys(described).sort(), [
    'action', 'at', 'bytes', 'detailsParsed', 'direction', 'error', 'filename',
    'id', 'ip', 'kind', 'mimeType', 'status', 'transferId',
  ])
  // And no value on the row is the clipboard content itself.
  for (const value of Object.values(described)) {
    if (typeof value === 'string') {
      assert.equal(value.includes('hunter2'), false, 'a value carried row content')
    }
  }
})

test('a file row carries its name and byte counts', () => {
  const described = describeAuditRow({
    id: 8,
    action: 'file.transfer.download',
    resource_id: 'file-9',
    details: JSON.stringify({
      status: 'completed',
      filename: 'quarterly.pdf',
      bytes_transferred: 2048,
      total_bytes: 4096,
    }),
    created_at: '2026-10-26T10:01:00Z',
  })

  assert.equal(described.kind, 'file')
  assert.equal(described.direction, 'download')
  assert.equal(described.filename, 'quarterly.pdf')
  assert.equal(described.bytes, 2048, 'bytes_transferred is what actually crossed')
})

// A failed transfer keeps its reason. A trail that only shows successes cannot
// answer "did anything reach this machine".
test('a failed transfer keeps its reason', () => {
  const described = describeAuditRow({
    action: 'file.transfer.upload',
    resource_id: 'file-err',
    details: JSON.stringify({
      status: 'failed',
      filename: 'a.txt',
      error: 'Host rejected the transfer',
    }),
    created_at: '2026-10-26T10:02:00Z',
  })

  assert.equal(described.status, 'failed')
  assert.equal(described.error, 'Host rejected the transfer')
})

// The case that is easiest to get wrong. A row whose details will not parse is
// still a row that exists: dropping it would make the list look complete when it
// is not, and an investigator would read the absence as "nothing happened".
test('a row whose details will not parse stays visible', () => {
  const described = describeAuditRow({
    id: 12,
    action: 'clipboard.sync.download',
    resource_id: 'clip-broken',
    details: 'not json at all',
    created_at: '2026-10-26T10:03:00Z',
  })

  assert.equal(described.detailsParsed, false)
  assert.equal(described.status, '', 'an unparsed row must not claim a status')
  assert.equal(described.bytes, null)
  // What it still knows: that a transfer of this id happened, and when.
  assert.equal(described.transferId, 'clip-broken')
  assert.equal(described.at, '2026-10-26T10:03:00Z')
})

// A caller that already parsed the object (a test, or a future API shape) is
// taken as-is rather than rejected.
test('an already-parsed details object is accepted', () => {
  const described = describeAuditRow({
    action: 'clipboard.sync.upload',
    resource_id: 'clip-obj',
    details: { status: 'completed', bytes: 5 },
  })
  assert.equal(described.detailsParsed, true)
  assert.equal(described.bytes, 5)
})

// A row with no details at all is described rather than throwing: the row exists
// and the list has to render it.
test('a row with no details is described, not thrown', () => {
  const described = describeAuditRow({
    id: 3,
    action: 'file.transfer.upload',
    resource_id: 'file-none',
    created_at: '2026-10-26T10:04:00Z',
  })
  assert.equal(described.transferId, 'file-none')
  assert.equal(described.detailsParsed, false)
  assert.equal(described.bytes, null)
})

test('the order of the page is preserved', () => {
  const rows = describeAuditRows([
    { id: 1, action: 'file.transfer.upload', resource_id: 'a', details: '{}' },
    { id: 2, action: 'clipboard.sync.upload', resource_id: 'b', details: '{}' },
  ])
  assert.deepEqual(rows.map((row) => row.id), [1, 2])
})

test('a non-array page is an empty list, not a crash', () => {
  assert.deepEqual(describeAuditRows(undefined), [])
  assert.deepEqual(describeAuditRows(null), [])
})

// The label has to describe the transfer without describing its contents. This
// is the same boundary as the server's, repeated on the client so the UI cannot
// be the thing that leaks.
test('the label says what moved and how much, never what', () => {
  const clipboard = auditRowLabel(describeAuditRow({
    action: 'clipboard.sync.upload',
    resource_id: 'clip-1',
    details: JSON.stringify({ status: 'completed', bytes: 1200, mime_type: 'text/plain' }),
  }))
  assert.ok(clipboard.includes('Clipboard'), clipboard)
  assert.ok(clipboard.includes('upload'), clipboard)
  assert.ok(clipboard.includes('1.2 KB'), `the byte count should be readable: ${clipboard}`)

  const file = auditRowLabel(describeAuditRow({
    action: 'file.transfer.download',
    resource_id: 'file-1',
    details: JSON.stringify({ status: 'completed', filename: 'report.pdf', bytes_transferred: 2048 }),
  }))
  assert.ok(file.includes('report.pdf'), file)
  assert.ok(file.includes('2.0 KB'), file)
})

// An unparsed row still gets a label. A blank line in the list would read as
// "nothing happened here" rather than "we could not describe this".
test('an unparsed row still gets a label', () => {
  const label = auditRowLabel(describeAuditRow({
    action: 'clipboard.sync.upload',
    resource_id: 'clip-broken',
    details: 'broken',
  }))
  assert.ok(label.includes('Clipboard'), label)
  assert.ok(label.length > 0, 'an unparsed row must not render as an empty line')
})

// The action filter is an allow-list on the server, so the client must not offer
// anything else - a 400 is a wasted round trip and a confusing error.
test('only the four server actions are offered as filters', () => {
  assert.equal(AUDIT_ACTIONS.length, 4)
  for (const action of AUDIT_ACTIONS) {
    assert.equal(isAuditAction(action), true, action)
  }
  for (const action of ['recording.delete', 'session.revoke', '', 'file.transfer']) {
    assert.equal(isAuditAction(action), false, `${action} must not be offered`)
  }
})
