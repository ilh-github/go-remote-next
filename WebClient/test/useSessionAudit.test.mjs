import assert from 'node:assert/strict'
import test from 'node:test'

// `localStorage` is a getter-only global in Node, so the store is installed the
// only way that works. It has to exist before the client module is imported:
// `new UserApi()` reads it in its constructor, so an import that runs first
// throws before any test starts.
Object.defineProperty(globalThis, 'localStorage', {
  value: {
    _v: new Map(),
    getItem(key) { return this._v.has(key) ? this._v.get(key) : null },
    setItem(key, value) { this._v.set(key, String(value)) },
    removeItem(key) { this._v.delete(key) },
  },
  configurable: true,
  writable: true,
})

// QD-909: the client-side loading rules for the audit trail.
//
// The rules under test are about honesty rather than about fetching. A read that
// fails has to look different from a read that found nothing, because "no
// transfers" and "we could not ask" are different facts and only one of them is
// evidence - and a component that renders both as an empty list makes the second
// one look like the first.

const { userApi } = await import('../src/api/userApi.js')
const { useSessionAudit } = await import('../src/composables/useSessionAudit.js')

function stubFetchAuditLogs(impl) {
  const original = userApi.fetchAuditLogs
  userApi.fetchAuditLogs = impl
  return () => { userApi.fetchAuditLogs = original }
}

function stubFetchArchivedAuditLogs(impl) {
  const original = userApi.fetchArchivedAuditLogs
  userApi.fetchArchivedAuditLogs = impl
  return () => { userApi.fetchArchivedAuditLogs = original }
}

// QD-110: retention moves expired rows into per-day files, so an empty live table
// is not the whole answer. The panel has to ask the archive, and say which one it
// is showing - "nothing was transferred" and "nothing is in the live table any
// more" are different facts.
test('an empty live table falls back to the archive and says so', async () => {
  const restoreLive = stubFetchAuditLogs(async () => ({ ok: true, data: { items: [] } }))
  const restoreArchive = stubFetchArchivedAuditLogs(async () => ({
    ok: true,
    data: {
      items: [
        {
          id: 9,
          action: 'file.transfer.upload',
          resource_id: 'tr-archived',
          details: JSON.stringify({ status: 'completed', filename: 'old.pdf', bytes_transferred: 99 }),
          created_at: '2026-09-01T10:00:00Z',
        },
      ],
    },
  }))
  try {
    const audit = useSessionAudit()
    await audit.load()

    assert.equal(audit.error.value, '')
    assert.equal(audit.loaded.value, true)
    assert.equal(audit.rows.value.length, 1, 'the archived row must reach the panel')
    assert.equal(audit.rows.value[0].transferId, 'tr-archived')
    assert.equal(audit.source.value, 'archive', 'the panel must say the rows came from the archive')
  } finally {
    restoreLive()
    restoreArchive()
  }
})

// A populated live table must not cost a second request: the archive is only
// asked when the live read came back empty.
test('a populated live table does not consult the archive', async () => {
  let archiveCalls = 0
  const restoreLive = stubFetchAuditLogs(async () => ({
    ok: true,
    data: {
      items: [
        {
          id: 1,
          action: 'clipboard.sync.upload',
          resource_id: 'clip-1',
          details: JSON.stringify({ status: 'completed', bytes: 42, mime_type: 'text/plain' }),
          created_at: '2026-10-26T10:00:00Z',
        },
      ],
    },
  }))
  const restoreArchive = stubFetchArchivedAuditLogs(async () => {
    archiveCalls += 1
    return { ok: true, data: { items: [] } }
  })
  try {
    const audit = useSessionAudit()
    await audit.load()

    assert.equal(audit.rows.value.length, 1)
    assert.equal(audit.source.value, 'live')
    assert.equal(archiveCalls, 0, 'a populated live table should not trigger an archive read')
  } finally {
    restoreLive()
    restoreArchive()
  }
})

// An archive that cannot be read must not turn a successful live read into a
// failure: an unset AUDIT_ARCHIVE_DIR is a server configuration fact, not
// something the owner did wrong.
test('an unreadable archive leaves a successful live read intact', async () => {
  const restoreLive = stubFetchAuditLogs(async () => ({ ok: true, data: { items: [] } }))
  const restoreArchive = stubFetchArchivedAuditLogs(async () => ({
    ok: false, error: 'audit archive is not configured',
  }))
  try {
    const audit = useSessionAudit()
    await audit.load()

    assert.equal(audit.error.value, '')
    assert.equal(audit.loaded.value, true)
    assert.equal(audit.rows.value.length, 0)
    assert.equal(audit.source.value, 'live')
  } finally {
    restoreLive()
    restoreArchive()
  }
})

test('a successful read leaves the rows described and marks it loaded', async () => {
  const restore = stubFetchAuditLogs(async () => ({
    ok: true,
    data: {
      items: [
        {
          id: 1,
          action: 'clipboard.sync.upload',
          resource_id: 'clip-1',
          details: JSON.stringify({ status: 'completed', bytes: 42, mime_type: 'text/plain' }),
          created_at: '2026-10-26T10:00:00Z',
        },
      ],
    },
  }))
  try {
    const audit = useSessionAudit()
    await audit.load()

    assert.equal(audit.error.value, '')
    assert.equal(audit.loading.value, false)
    assert.equal(audit.loaded.value, true, 'a completed read must be marked as one')
    assert.equal(audit.rows.value.length, 1)
    assert.equal(audit.rows.value[0].bytes, 42)
    assert.equal(audit.rows.value[0].kind, 'clipboard')
  } finally {
    restore()
  }
})

// The case that matters: a failed read must not render as an empty list. If it
// did, "the server refused" would read as "nothing was transferred", which is
// the opposite of what happened.
test('a failed read is an error, not an empty list', async () => {
  const restore = stubFetchAuditLogs(async () => ({ ok: false, error: 'session ended' }))
  try {
    const audit = useSessionAudit()
    await audit.load()

    assert.equal(audit.error.value, 'session ended')
    assert.equal(audit.rows.value.length, 0)
    assert.equal(audit.loaded.value, false,
      'a failed read must not be marked loaded, or it renders as "no transfers"')
  } finally {
    restore()
  }
})

// A read that returns no rows is a fact worth showing: this account moved
// nothing. It is the one case where an empty list is the truth.
test('an empty successful read is loaded with no rows', async () => {
  const restore = stubFetchAuditLogs(async () => ({ ok: true, data: { items: [] } }))
  try {
    const audit = useSessionAudit()
    await audit.load()

    assert.equal(audit.error.value, '')
    assert.equal(audit.loaded.value, true)
    assert.equal(audit.rows.value.length, 0)
  } finally {
    restore()
  }
})

// A response with no items array at all is treated as empty rather than thrown:
// a contract that omits the field must not take the page down.
test('a response with no items array is empty, not a crash', async () => {
  const restore = stubFetchAuditLogs(async () => ({ ok: true, data: {} }))
  try {
    const audit = useSessionAudit()
    await audit.load()

    assert.equal(audit.error.value, '')
    assert.equal(audit.loaded.value, true)
    assert.deepEqual(audit.rows.value, [])
  } finally {
    restore()
  }
})

// A read that throws rather than returning a failure object is still an error
// message, not an unhandled rejection that leaves the spinner running.
test('a thrown read becomes an error and stops the spinner', async () => {
  const restore = stubFetchAuditLogs(async () => { throw new Error('network down') })
  try {
    const audit = useSessionAudit()
    await audit.load()

    assert.equal(audit.error.value, 'network down')
    assert.equal(audit.loading.value, false, 'the spinner must stop')
    assert.equal(audit.loaded.value, false)
  } finally {
    restore()
  }
})

// Loading clears the previous rows rather than leaving them under a new heading.
// Showing the previous read's transfers while a new one is in flight is a lie
// the user cannot detect.
test('a new load clears the previous rows', async () => {
  let call = 0
  const restore = stubFetchAuditLogs(async () => {
    call += 1
    if (call === 1) {
      return {
        ok: true,
        data: {
          items: [{
            id: 1, action: 'file.transfer.upload', resource_id: 'file-1',
            details: JSON.stringify({ status: 'completed', filename: 'a.txt', bytes_transferred: 5 }),
          }],
        },
      }
    }
    return { ok: false, error: 'refused' }
  })
  try {
    const audit = useSessionAudit()
    await audit.load()
    assert.equal(audit.rows.value.length, 1)

    await audit.load()
    assert.equal(audit.rows.value.length, 0,
      'a failed reload left the previous rows on screen')
    assert.equal(audit.error.value, 'refused')
  } finally {
    restore()
  }
})

test('reset returns the composable to its unread state', async () => {
  const restore = stubFetchAuditLogs(async () => ({
    ok: true,
    data: {
      items: [{
        id: 1, action: 'clipboard.sync.upload', resource_id: 'clip-1',
        details: JSON.stringify({ status: 'completed', bytes: 1 }),
      }],
    },
  }))
  try {
    const audit = useSessionAudit()
    await audit.load()
    audit.reset()

    assert.deepEqual(audit.rows.value, [])
    assert.equal(audit.loaded.value, false)
    assert.equal(audit.error.value, '')
    assert.equal(audit.loading.value, false)
  } finally {
    restore()
  }
})

// QD-110: the server pages at 200 and hands back a `next` cursor. The panel asks
// for 100 and stops there, so an account with more transfers than one page shows
// only the newest slice - with nothing on screen saying there is more. The owner
// reading "these are my transfers" is actually reading "these are my newest
// transfers", and the difference matters exactly when they are looking for
// something old.
test('a read that has more pages follows the cursor', async () => {
  const restore = stubFetchAuditLogs(async ({ cursor }) => {
    if (cursor === 'page-2') {
      return { ok: true, data: { items: [{ id: 2, action: 'file.transfer' }], next: '' } }
    }
    return { ok: true, data: { items: [{ id: 1, action: 'file.transfer' }], next: 'page-2' } }
  })
  const restoreArchive = stubFetchArchivedAuditLogs(async () => ({ ok: true, data: { items: [] } }))
  try {
    const audit = useSessionAudit()
    await audit.load()

    assert.equal(audit.rows.value.length, 2,
      'the second page was never fetched, so an account with more transfers than one page shows only the newest')
    assert.equal(audit.rows.value[0].id, 1)
    assert.equal(audit.rows.value[1].id, 2)
  } finally {
    restore()
    restoreArchive()
  }
})

test('a read with no further page stops after one request', async () => {
  let calls = 0
  const restore = stubFetchAuditLogs(async () => {
    calls += 1
    return { ok: true, data: { items: [{ id: 1, action: 'file.transfer' }], next: '' } }
  })
  const restoreArchive = stubFetchArchivedAuditLogs(async () => ({ ok: true, data: { items: [] } }))
  try {
    const audit = useSessionAudit()
    await audit.load()

    assert.equal(calls, 1, 'a read with no next cursor issued another request')
    assert.equal(audit.rows.value.length, 1)
  } finally {
    restore()
    restoreArchive()
  }
})

test('a failing later page does not discard the pages already read', async () => {
  // The first page succeeded, so those rows are real. Throwing them away because
  // the second request failed would turn a partial read into an empty one, which
  // is the same lie the error path is careful not to tell.
  const restore = stubFetchAuditLogs(async ({ cursor }) => {
    if (cursor === 'page-2') return { ok: false, error: 'the second page failed' }
    return { ok: true, data: { items: [{ id: 1, action: 'file.transfer' }], next: 'page-2' } }
  })
  const restoreArchive = stubFetchArchivedAuditLogs(async () => ({ ok: true, data: { items: [] } }))
  try {
    const audit = useSessionAudit()
    await audit.load()

    assert.equal(audit.rows.value.length, 1,
      'a failed later page discarded the rows the first page already returned')
  } finally {
    restore()
    restoreArchive()
  }
})

test('a server that returns the same cursor forever does not spin', async () => {
  // The guard is the `next === cursor` check. Without it a server that hands back
  // the same cursor on every page - or one that never clears it - turns the read
  // into an unbounded loop, and the panel never finishes loading.
  let calls = 0
  const restore = stubFetchAuditLogs(async () => {
    calls += 1
    if (calls > 20) return { ok: true, data: { items: [], next: '' } }
    return { ok: true, data: { items: [{ id: calls, action: 'file.transfer' }], next: 'stuck' } }
  })
  const restoreArchive = stubFetchArchivedAuditLogs(async () => ({ ok: true, data: { items: [] } }))
  try {
    const audit = useSessionAudit()
    await audit.load()

    assert.ok(calls <= 3,
      `the read issued ${calls} requests against a server repeating one cursor`)
  } finally {
    restore()
    restoreArchive()
  }
})
