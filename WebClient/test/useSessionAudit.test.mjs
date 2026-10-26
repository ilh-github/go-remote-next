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
