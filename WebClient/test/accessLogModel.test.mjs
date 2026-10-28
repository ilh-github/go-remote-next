import assert from 'node:assert/strict'
import test from 'node:test'

// QD-106/QD-909: the server records every access attempt - who tried, from
// where, and whether it worked - and has served them at
// `GET /v1/me/devices/:id/access-logs` for a while. No client ever called it.
//
// That left the one question an owner actually asks - "is somebody trying to get
// into my machine?" - unanswerable from any UI. The session detail shows access
// logs for sessions that started, but a failed attempt never creates a session,
// so the attempts that matter most were the ones with nowhere to appear.
//
// Rows are written by the server at verification time, never by the controller,
// so a client cannot hide or forge its own attempts. The model's job is to say
// what each canonical result means, because `denied_invalid_code` and
// `host_offline` are both "no session" and only one of them is somebody trying.
const { describeAccessLog, ACCESS_LOG_OUTCOME } = await import('../src/api/accessLogModel.js')

test('a wrong access code is reported as a denial, not a failure to connect', () => {
  const described = describeAccessLog({
    id: 1,
    result: 'denied_invalid_code',
    ip: '203.0.113.9',
    client_id: 'web-abc',
    created_at: '2026-10-18T09:59:00Z',
  })

  assert.equal(described.outcome, ACCESS_LOG_OUTCOME.Denied)
  assert.equal(described.denied, true)
  // The distinction that matters: somebody was guessing, so this is the row an
  // owner scanning the list needs to see stand out.
  assert.equal(described.suspicious, true, 'a wrong code was not flagged as a guess')
  assert.equal(described.ip, '203.0.113.9')
})

test('rate limiting is suspicious too, because it means repeated guesses', () => {
  const described = describeAccessLog({ id: 2, result: 'denied_rate_limited', ip: '203.0.113.10' })

  assert.equal(described.outcome, ACCESS_LOG_OUTCOME.Denied)
  assert.equal(described.suspicious, true)
})

test('a host that was offline is not a denial and not suspicious', () => {
  // The code matched; the machine just was not there. Reporting it as a failed
  // attempt would teach the owner to ignore the column that matters.
  const described = describeAccessLog({ id: 3, result: 'host_offline', ip: '203.0.113.11' })

  assert.equal(described.outcome, ACCESS_LOG_OUTCOME.Offline)
  assert.equal(described.denied, false, 'an offline host was reported as a denial')
  assert.equal(described.suspicious, false)
})

test('a policy or session-limit refusal is a denial but not a guess', () => {
  // The code was right, so these are not somebody guessing - they are the
  // owner's own rules doing their job. Marking them suspicious would drown the
  // real guesses.
  for (const result of ['denied_policy', 'denied_session_limit', 'denied_revoked']) {
    const described = describeAccessLog({ id: 4, result })
    assert.equal(described.outcome, ACCESS_LOG_OUTCOME.Denied, `${result} was not a denial`)
    assert.equal(described.suspicious, false, `${result} was wrongly flagged as a guess`)
  }
})

test('a pending approval is neither granted nor denied', () => {
  const described = describeAccessLog({ id: 5, result: 'pending_approval' })
  assert.equal(described.outcome, ACCESS_LOG_OUTCOME.Pending)
  assert.equal(described.denied, false)
})

test('a granted attempt and a started session both read as granted', () => {
  for (const result of ['granted', 'session_started']) {
    assert.equal(describeAccessLog({ id: 6, result }).outcome, ACCESS_LOG_OUTCOME.Granted,
      `${result} was not read as granted`)
  }
})

test('an unknown result is surfaced as unknown rather than guessed at', () => {
  // The server can grow a new result. Defaulting it to "granted" would hide a
  // refusal; defaulting it to "denied" would cry wolf. Neither is knowledge.
  const described = describeAccessLog({ id: 7, result: 'some_future_result' })
  assert.equal(described.outcome, ACCESS_LOG_OUTCOME.Unknown)
  assert.equal(described.denied, false)
  assert.equal(described.suspicious, false)
})

test('a row missing its fields does not render as undefined', () => {
  const described = describeAccessLog({})
  assert.equal(described.ip, '')
  assert.equal(described.clientId, '')
  assert.equal(described.at, '')
  assert.equal(described.outcome, ACCESS_LOG_OUTCOME.Unknown)
})

// The model tests above prove the mapping. These prove the endpoint is actually
// reached and the panel actually renders it - the half that was missing for as
// long as the server has been serving these rows.
test('the client calls the owner-scoped access-logs route', async () => {
  const source = await import('node:fs/promises').then(fs =>
    fs.readFile(new URL('../src/api/userApi.js', import.meta.url), 'utf8'))

  assert.match(source, /fetchAccessLogs\(/,
    'the client has no method for the access-logs endpoint')
  assert.match(source, /\/v1\/me\/devices\/\$\{encodeURIComponent\(deviceId\)\}\/access-logs/,
    'the access-logs method does not target the owner-scoped route')
})

test('the sessions page loads and renders the access attempts', async () => {
  const source = await import('node:fs/promises').then(fs =>
    fs.readFile(new URL('../src/views/SessionsPage.vue', import.meta.url), 'utf8'))

  assert.match(source, /fetchAccessLogs\(deviceFilter\.value/,
    'the panel never asks the server for the attempts')
  // The loader existing is not the same as the loader running: a fetch nobody
  // calls leaves the panel permanently empty.
  assert.match(source, /await loadAccessLogs\(\)/,
    'the access-log loader is never called, so the panel would always be empty')
  assert.match(source, /describeAccessLogs\(/,
    'the rows reach the template without going through the model')
  assert.match(source, /sessions\.accessAttempts/,
    'the panel has no title, so the rows are still unreachable from the UI')
  assert.match(source, /v-for="entry in accessLogs"/,
    'the panel does not render the rows it loads')
  // A guess has to be actionable from the row it appears on; otherwise the owner
  // reads the IP and has to retype it into a different form.
  assert.match(source, /banAccessLogSource\(/,
    'a suspicious row offers no way to act on the address it names')
})
