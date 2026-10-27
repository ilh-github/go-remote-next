// The audit list's query string.
//
// The server accepts six filters on /v1/admin/audit-logs: action, admin,
// username, actor_user_id, date_from, date_to. The UI sent five of them. The two
// it dropped are the pair an operator reaches for first when asked "what did
// this account do" - and `username` is the one that matches an end user's name
// as well as an administrator's, so without it a question about a user has no
// answer in this screen at all.
import test from 'node:test'
import assert from 'node:assert/strict'

const mod = await import('../src/views/audit-filter.js')

test('every filter the server documents is reachable', () => {
  const params = mod.auditLogParams({
    action: 'login',
    admin: 'root',
    username: 'tim',
    actorUserId: 42,
    dateFrom: '2026-10-01',
    dateTo: '2026-10-02',
  })
  assert.equal(params.get('action'), 'login')
  assert.equal(params.get('admin'), 'root')
  assert.equal(params.get('username'), 'tim')
  assert.equal(params.get('actor_user_id'), '42')
  assert.equal(params.get('date_from'), '2026-10-01')
  assert.equal(params.get('date_to'), '2026-10-02')
})

test('a blank filter is omitted rather than sent empty', () => {
  // An empty value is not "no filter" to a server doing a substring match: it
  // would match every row and quietly widen the result.
  const params = mod.auditLogParams({ action: '', admin: '   ', username: '' })
  for (const key of ['action', 'admin', 'username']) {
    assert.ok(!params.has(key), `${key} must not be sent when blank`)
  }
})

test('a whitespace-only username is trimmed away', () => {
  const params = mod.auditLogParams({ username: '   ' })
  assert.ok(!params.has('username'))
})

test('a non-numeric actor id is refused, not forwarded', () => {
  // The server answers a non-numeric actor_user_id with a 400 rather than
  // treating it as "no filter", so a typo would blank the whole list.
  assert.throws(() => mod.auditLogParams({ actorUserId: 'abc' }), /actor_user_id/i)
  assert.throws(() => mod.auditLogParams({ actorUserId: -1 }), /actor_user_id/i)
})

test('a numeric actor id given as a string is accepted', () => {
  // el-input binds strings, so the value often arrives as one.
  assert.equal(mod.auditLogParams({ actorUserId: '42' }).get('actor_user_id'), '42')
})

test('a reversed date window is refused before the request goes out', () => {
  assert.throws(
    () => mod.auditLogParams({ dateFrom: '2026-10-02', dateTo: '2026-10-01' }),
    /date_from/i,
  )
})

test('the same day is a valid window', () => {
  const params = mod.auditLogParams({ dateFrom: '2026-10-01', dateTo: '2026-10-01' })
  assert.equal(params.get('date_from'), '2026-10-01')
  assert.equal(params.get('date_to'), '2026-10-01')
})

test('pagination still travels with the filters', () => {
  const params = mod.auditLogParams({ cursor: 'c1', limit: 50 })
  assert.equal(params.get('cursor'), 'c1')
  assert.equal(params.get('limit'), '50')
})
