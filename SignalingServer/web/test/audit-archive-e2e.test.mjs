// The same three states, but read off a real server response rather than a
// hand-built Error, so the handler's status codes and the UI's classification
// cannot drift apart unnoticed.
//
// Needs a running admin API. Skipped (not failed) when it is absent, because a
// missing dev server is an environment fact, not a defect.
import test from 'node:test'
import assert from 'node:assert/strict'
import './localstorage-stub.mjs'

const BASE = process.env.QD_ADMIN_BASE || 'http://127.0.0.1:18099'
const { problemToError } = await import('../src/api/auth.js')
const { archiveErrorKind } = await import('../src/views/audit-archive-errors.js')

let token = null
let reachable = true

test('sign in', async (t) => {
  try {
    const res = await fetch(`${BASE}/v1/admin/auth/sessions`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: process.env.QD_ADMIN_USER || 'admin',
        password: process.env.QD_ADMIN_PASSWORD || 'admin',
      }),
    })
    if (!res.ok) { reachable = false; return t.skip(`login HTTP ${res.status}`) }
    token = (await res.json()).access_token
    assert.ok(token)
  } catch {
    reachable = false
    t.skip('admin API not reachable')
  }
})

test('a day with no archive file is an empty answer, not a missing configuration', async (t) => {
  if (!reachable) return t.skip('no server')
  const res = await fetch(`${BASE}/v1/admin/audit-archive:search?date=2099-01-01`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  assert.equal(res.status, 404)
  const err = problemToError(res, await res.json())
  assert.equal(archiveErrorKind(err), 'no_file')
})

test('a malformed date stays an error', async (t) => {
  if (!reachable) return t.skip('no server')
  const res = await fetch(`${BASE}/v1/admin/audit-archive:search?date=nonsense`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  assert.equal(res.status, 400)
  const err = problemToError(res, await res.json())
  // A bad date is also a 400, and must not be reported as "archive not
  // configured" - only the configuration error is.
  assert.equal(archiveErrorKind(err), 'error')
})
