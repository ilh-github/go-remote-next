import assert from 'node:assert/strict'
import { execFile, spawn } from 'node:child_process'
import { mkdir, mkdtemp, rm } from 'node:fs/promises'
import net from 'node:net'
import os from 'node:os'
import path from 'node:path'
import { promisify } from 'node:util'
import { fileURLToPath } from 'node:url'
import { after, before, test } from 'node:test'

// QD-909, owner read half: the audit trail has to be readable by the person it
// belongs to.
//
// The write endpoints already refuse a device the caller does not own, so every
// row they produce carries that device's owner as its actor. What did not exist
// was a way for the owner to read those rows back - only the admin console could.
// That leaves "what did my controller copy at 14:03?" unanswerable by the one
// person who can act on the answer, which makes the trail a write-only log
// rather than an audit.
//
// The isolation half matters as much as the read: a second account is seeded on
// the same server so "my rows" can be proven to mean only mine.

const runFile = promisify(execFile)
const WEB_ROOT = path.resolve(fileURLToPath(new URL('../../', import.meta.url)))
const REPO_ROOT = path.resolve(WEB_ROOT, '..')
const SERVER_ROOT = path.join(REPO_ROOT, 'SignalingServer')
const API_KEY = 'owner-audit-e2e-api-key'
const ADMIN_PASSWORD = 'OwnerAuditAdmin-123!'
const USERNAME = 'owner-audit-e2e'
const PASSWORD = 'OwnerAuditUser-123!'
const OTHER_USERNAME = 'owner-audit-e2e-other'
const OTHER_PASSWORD = 'OwnerAuditOther-123!'
const DEVICE_ID = '678901345'
const DEVICE_NAME = 'Owner Audit Host'

let redis
let signaling
let tempRoot
let backendUrl

async function findFreePort() {
  const server = net.createServer()
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve))
  const { port } = server.address()
  await new Promise((resolve) => server.close(resolve))
  return port
}

function startProcess(command, args, { label, cwd, env }) {
  const child = spawn(command, args, { cwd, env, stdio: ['ignore', 'pipe', 'pipe'] })
  child.stdout.on('data', () => {})
  child.stderr.on('data', () => {})
  return child
}

async function stopProcess(child) {
  if (!child || child.exitCode !== null) return
  child.kill('SIGTERM')
  await new Promise((resolve) => {
    const timer = setTimeout(() => { child.kill('SIGKILL'); resolve() }, 5000)
    child.on('exit', () => { clearTimeout(timer); resolve() })
  })
}

async function waitForHealth(url, child) {
  for (let i = 0; i < 60; i += 1) {
    if (child.exitCode !== null) throw new Error('signaling exited before becoming healthy')
    try {
      const response = await fetch(url)
      if (response.ok) return
    } catch { /* not up yet */ }
    await new Promise((resolve) => setTimeout(resolve, 500))
  }
  throw new Error('signaling did not become healthy in 30s')
}

async function call(method, requestPath, body, token) {
  const response = await fetch(`${backendUrl}${requestPath}`, {
    method,
    headers: {
      'content-type': 'application/json',
      origin: 'http://127.0.0.1',
      ...(token ? { authorization: `Bearer ${token}` } : {}),
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  return { status: response.status, json: await response.json().catch(() => null) }
}

before(async () => {
  tempRoot = await mkdtemp(path.join(os.tmpdir(), 'quickdesk-owner-audit-e2e-'))
  const backendPort = await findFreePort()
  const redisPort = await findFreePort()
  backendUrl = `http://127.0.0.1:${backendPort}`
  const databasePath = path.join(tempRoot, 'data', 'quickdesk.db')
  await mkdir(path.dirname(databasePath), { recursive: true })

  const signalingBinary = path.join(tempRoot, 'signaling')
  const seedBinary = path.join(tempRoot, 'seed-smoke')
  await Promise.all([
    runFile('go', ['build', '-o', signalingBinary, './cmd/signaling'], {
      // 64 MiB, not 16: on a cold module cache `go build` prints one line per
      // module it downloads, which is tens of megabytes and overflows a smaller
      // buffer - the call then fails with "Command failed" before any assertion
      // has run. The first CI run of these tests failed exactly there.
      cwd: SERVER_ROOT, maxBuffer: 64 * 1024 * 1024,
    }),
    runFile('go', ['build', '-o', seedBinary, './cmd/seed-smoke'], {
      cwd: SERVER_ROOT, maxBuffer: 64 * 1024 * 1024,
    }),
  ])

  redis = startProcess('redis-server', [
    '--bind', '127.0.0.1', '--port', String(redisPort),
    '--notify-keyspace-events', 'Ex', '--save', '', '--appendonly', 'no', '--dir', tempRoot,
  ], { label: 'redis' })
  await new Promise((resolve) => setTimeout(resolve, 500))

  const sharedEnv = {
    ...process.env,
    SERVER_HOST: '127.0.0.1',
    SERVER_PORT: String(backendPort),
    DB_DRIVER: 'sqlite',
    DB_PATH: databasePath,
    REDIS_DRIVER: 'redis',
    REDIS_HOST: '127.0.0.1',
    REDIS_PORT: String(redisPort),
  }
  signaling = startProcess(signalingBinary, [], {
    cwd: path.dirname(databasePath),
    label: 'signaling',
    env: {
      ...sharedEnv,
      RUN_AUTO_MIGRATE: 'false',
      LOG_DIR: path.join(tempRoot, 'logs'),
      API_KEY,
      ALLOWED_ORIGINS: 'http://127.0.0.1',
      ADMIN_USER: 'admin',
      ADMIN_PASSWORD,
    },
  })
  await waitForHealth(`${backendUrl}/health`, signaling)

  const seedArgs = (username, password, deviceId, deviceName) => [
    '-server', backendUrl,
    '-api-key', API_KEY,
    '-admin-user', 'admin',
    '-admin-password', ADMIN_PASSWORD,
    '-username', username,
    '-password', password,
    '-device-id', deviceId,
    '-device-name', deviceName,
  ]
  await runFile(seedBinary, seedArgs(USERNAME, PASSWORD, DEVICE_ID, DEVICE_NAME), {
    cwd: path.dirname(databasePath), env: sharedEnv, maxBuffer: 16 * 1024 * 1024,
  })
  // A second account on the same server, so "my rows" can be shown to exclude
  // somebody else's rather than merely assumed to.
  await runFile(seedBinary, seedArgs(OTHER_USERNAME, OTHER_PASSWORD, '678901999', 'Other Host'), {
    cwd: path.dirname(databasePath), env: sharedEnv, maxBuffer: 16 * 1024 * 1024,
  })
})

after(async () => {
  await stopProcess(signaling)
  await stopProcess(redis)
  if (tempRoot) await rm(tempRoot, { recursive: true, force: true })
})

async function login(identifier, password) {
  const r = await call('POST', '/v1/auth/sessions', { identifier, password })
  assert.equal(r.status, 200, `login failed for ${identifier}: ${JSON.stringify(r.json)}`)
  return r.json.access_token
}

test('the owner reads back the transfers they reported', async () => {
  const token = await login(USERNAME, PASSWORD)
  const transferId = `owner-audit-clip-${Date.now()}`

  const report = await call('POST', '/v1/me/clipboard-audits', {
    device_id: DEVICE_ID,
    transfer_id: transferId,
    direction: 'upload',
    status: 'completed',
    mime_type: 'text/plain',
    bytes: 128,
  }, token)
  assert.equal(report.status, 204, `report status ${report.status}`)

  const fileId = `owner-audit-file-${Date.now()}`
  const fileReport = await call('POST', '/v1/me/file-transfer-audits', {
    device_id: DEVICE_ID,
    transfer_id: fileId,
    direction: 'download',
    status: 'completed',
    filename: 'quarterly.pdf',
    bytes_transferred: 4096,
    total_bytes: 4096,
  }, token)
  assert.equal(fileReport.status, 204, `file report status ${fileReport.status}`)

  const list = await call('GET', '/v1/me/audit-logs?limit=50', undefined, token)
  assert.equal(list.status, 200, `read status ${list.status}: ${JSON.stringify(list.json)}`)

  const items = list.json?.items ?? []
  const clipboard = items.find((row) => row.resource_id === transferId)
  assert.ok(clipboard, `the reported clipboard sync was not readable back: ${JSON.stringify(items)}`)
  // The direction is no longer part of the action name: one transfer id is one
  // audit row whichever way it went, so the direction lives in the details. The
  // aggregate action is what the filter is built on.
  assert.equal(clipboard.action, 'clipboard.sync')

  const details = JSON.parse(clipboard.details)
  assert.equal(details.bytes, 128, 'the byte count must survive the round trip')
  assert.equal(details.mime_type, 'text/plain')
  assert.deepEqual(details.directions, ['upload'], 'the direction must survive the round trip')

  const file = items.find((row) => row.resource_id === fileId)
  assert.ok(file, `the reported file transfer was not readable back: ${JSON.stringify(items)}`)
  assert.equal(file.action, 'file.transfer')
  assert.equal(JSON.parse(file.details).filename, 'quarterly.pdf')
})

// "My transfers" has to mean only mine. A second account on the same server
// reports its own transfer, and the first account must not see it - otherwise the
// audit trail is a surveillance tool rather than an attribution record.
test("another account's transfers are not in my trail", async () => {
  const token = await login(USERNAME, PASSWORD)
  const otherToken = await login(OTHER_USERNAME, OTHER_PASSWORD)

  const otherId = `owner-audit-other-${Date.now()}`
  const otherReport = await call('POST', '/v1/me/clipboard-audits', {
    device_id: '678901999',
    transfer_id: otherId,
    direction: 'upload',
    status: 'completed',
    mime_type: 'text/plain',
    bytes: 999,
  }, otherToken)
  assert.equal(otherReport.status, 204, `other account report status ${otherReport.status}`)

  const list = await call('GET', '/v1/me/audit-logs?limit=100', undefined, token)
  assert.equal(list.status, 200, `read status ${list.status}`)
  const items = list.json?.items ?? []
  assert.equal(items.some((row) => row.resource_id === otherId), false,
    "another account's transfer id was visible in my trail")
  assert.equal(items.some((row) => row.actor_user_id !== undefined && row.actor_user_id === 0), false,
    'a row with no actor leaked into an owner-scoped read')

  // The other account can read its own row, which is the same feature working.
  const otherList = await call('GET', '/v1/me/audit-logs?limit=100', undefined, otherToken)
  assert.equal(otherList.status, 200, `other read status ${otherList.status}`)
  const otherItems = otherList.json?.items ?? []
  assert.ok(otherItems.some((row) => row.resource_id === otherId),
    'the owning account could not read its own transfer back')
})

// An action filter narrows to one kind, and an action this trail does not hold is
// refused rather than answered with an empty page - "you did nothing" is a
// conclusion, and it must not be produced by a typo.
test('the action filter narrows and refuses what it does not cover', async () => {
  const token = await login(USERNAME, PASSWORD)

  const clipId = `owner-audit-filter-clip-${Date.now()}`
  const clip = await call('POST', '/v1/me/clipboard-audits', {
    device_id: DEVICE_ID,
    transfer_id: clipId,
    direction: 'upload',
    status: 'completed',
    mime_type: 'text/plain',
    bytes: 12,
  }, token)
  assert.equal(clip.status, 204, `clip report status ${clip.status}`)

  const fileId = `owner-audit-filter-file-${Date.now()}`
  const file = await call('POST', '/v1/me/file-transfer-audits', {
    device_id: DEVICE_ID,
    transfer_id: fileId,
    direction: 'upload',
    status: 'completed',
    filename: 'notes.txt',
    bytes_transferred: 5,
    total_bytes: 5,
  }, token)
  assert.equal(file.status, 204, `file report status ${file.status}`)

  const clipboardOnly = await call('GET',
    '/v1/me/audit-logs?action=clipboard.sync.upload&limit=50', undefined, token)
  assert.equal(clipboardOnly.status, 200, `filtered read status ${clipboardOnly.status}`)
  for (const row of clipboardOnly.json?.items ?? []) {
    assert.equal(row.action, 'clipboard.sync',
      `the action filter let through ${row.action}`)
  }
  assert.ok((clipboardOnly.json?.items ?? []).some((row) => row.resource_id === clipId),
    'the clipboard transfer was missing from its own filtered read')

  const unknown = await call('GET', '/v1/me/audit-logs?action=recording.delete', undefined, token)
  assert.equal(unknown.status, 400,
    `an action outside this trail was answered ${unknown.status}, want 400`)
})

// The read is paginated like every other v1 list, so a busy account does not get
// one unbounded response.
test('the read is paginated', async () => {
  const token = await login(USERNAME, PASSWORD)
  const first = await call('GET', '/v1/me/audit-logs?limit=1', undefined, token)
  assert.equal(first.status, 200, `first page status ${first.status}`)
  const items = first.json?.items ?? []
  assert.equal(items.length, 1, `asked for 1 and got ${items.length}`)

  const cursor = first.json?.next_cursor
  if (!cursor) return
  const second = await call('GET',
    `/v1/me/audit-logs?limit=1&cursor=${encodeURIComponent(cursor)}`, undefined, token)
  assert.equal(second.status, 200, `second page status ${second.status}`)
  const nextItems = second.json?.items ?? []
  assert.equal(nextItems.length, 1)
  assert.notEqual(nextItems[0].id, items[0].id, 'the second page repeated the first row')
})
