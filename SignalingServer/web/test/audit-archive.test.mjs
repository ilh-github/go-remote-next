// The archive panel has to tell three states apart, and two of them are
// errors that look alike at a glance:
//
//   1. not configured  - retention cannot archive at all (400 INVALID_REQUEST)
//   2. no file for day - the archive is on, that day simply has nothing (404)
//   3. a real failure  - network, 500, expired session
//
// Collapsing 1 and 2 is not cosmetic: it tells the operator to set an env var
// that is already set, and hides the fact that the archive itself is healthy.
import test from 'node:test'
import assert from 'node:assert/strict'
import './localstorage-stub.mjs'

const mod = await import('../src/views/audit-archive-errors.js')

const notConfigured = Object.assign(new Error('audit archive is not configured; set AUDIT_ARCHIVE_DIR'), {
  code: 'INVALID_REQUEST',
  status: 400,
})

const noFile = Object.assign(new Error('no archive file for 2026-10-21'), {
  code: 'NOT_FOUND',
  status: 404,
})

test('an unconfigured archive is reported as a configuration fact', () => {
  assert.equal(mod.archiveErrorKind(notConfigured), 'not_configured')
})

test('a day with no archive file is not the same as no archive at all', () => {
  // Regression: the message also contains "archive", so a text match on that
  // word reported "not configured" for a healthy archive.
  assert.equal(mod.archiveErrorKind(noFile), 'no_file')
})

test('an unrelated failure stays a failure', () => {
  assert.equal(mod.archiveErrorKind(Object.assign(new Error('boom'), { status: 500 })), 'error')
})

test('a plain string error without structure is still an error', () => {
  assert.equal(mod.archiveErrorKind(new Error('audit archive is not configured')), 'error')
})

test('an expired session is not mistaken for a missing archive', () => {
  assert.equal(mod.archiveErrorKind(Object.assign(new Error('session ended'), { status: 401 })), 'error')
})
