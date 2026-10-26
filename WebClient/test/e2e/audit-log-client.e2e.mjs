import assert from 'node:assert/strict'
import { execFile, spawn } from 'node:child_process'
import { mkdir, mkdtemp, rm } from 'node:fs/promises'
import net from 'node:net'
import os from 'node:os'
import path from 'node:path'
import { promisify } from 'node:util'
import { fileURLToPath } from 'node:url'
import { after, before, test } from 'node:test'

// QD-909, client half: the audit trail has to be readable through the client the
// user actually has, not only through a curl against the API.
//
// The server side is covered by owner-audit.e2e.mjs. What that does not prove is
// that a user can reach the data, and the checklist's definition of done requires
// a discoverable entry rather than an internal API. This drives the real
// `userApi.fetchAuditLogs` client method against a real server, so a rename in
// the client or a contract drift in the query string fails here.
//
// The client is loaded as a module rather than through a browser: the point under
// test is the client's request shape and its reading of the response, and a
// browser would add localStorage and origin setup that says nothing about either.

const runFile = promisify(execFile)

// `localStorage` is a getter-only global in Node, so the store is installed the
// only way that works rather than by assignment. It has to exist before the
// client module is imported: `new UserApi()` reads it in its constructor, so an
// import that happens first would throw before any test runs.
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
const WEB_ROOT = path.resolve(fileURLToPath(new URL('../../', import.meta.url)))
const REPO_ROOT = path.resolve(WEB_ROOT, '..')
const SERVER_ROOT = path.join(REPO_ROOT, 'SignalingServer')
const API_KEY = 'audit-client-e2e-api-key'
const ADMIN_PASSWORD = 'AuditClientAdmin-123!'
const USERNAME = 'audit-client-e2e'
const PASSWORD = 'AuditClientUser-123!'
const DEVICE_ID = '789012456'
const DEVICE_NAME = 'Audit Client Host'

let redis
let signaling
let tempRoot
let backendUrl
let userApi

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

before(async () => {
  tempRoot = await mkdtemp(path.join(os.tmpdir(), 'quickdesk-audit-client-e2e-'))
  const backendPort = await findFreePort()
  const redisPort = await findFreePort()
  backendUrl = `http://127.0.0.1:${backendPort}`
  const databasePath = path.join(tempRoot, 'data', 'quickdesk.db')
  await mkdir(path.dirname(databasePath), { recursive: true })

  const signalingBinary = path.join(tempRoot, 'signaling')
  const seedBinary = path.join(tempRoot, 'seed-smoke')
  await Promise.all([
    runFile('go', ['build', '-o', signalingBinary, './cmd/signaling'], {
      cwd: SERVER_ROOT, maxBuffer: 16 * 1024 * 1024,
    }),
    runFile('go', ['build', '-o', seedBinary, './cmd/seed-smoke'], {
      cwd: SERVER_ROOT, maxBuffer: 16 * 1024 * 1024,
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

  await runFile(seedBinary, [
    '-server', backendUrl,
    '-api-key', API_KEY,
    '-admin-user', 'admin',
    '-admin-password', ADMIN_PASSWORD,
    '-username', USERNAME,
    '-password', PASSWORD,
    '-device-id', DEVICE_ID,
    '-device-name', DEVICE_NAME,
  ], { cwd: path.dirname(databasePath), env: sharedEnv, maxBuffer: 16 * 1024 * 1024 })

  // The real client module, pointed at this server, with a session the seed
  // created. Loaded last so the base URL and token are already decided. The
  // module exports a singleton rather than a class, so this is the same instance
  // the app uses - which is what makes the request shape under test the real one.
  const { userApi: api } = await import('../../src/api/userApi.js')
  userApi = api
  userApi.setBaseUrl(backendUrl)

  // Node's fetch sends no `Origin`, and the server admits browser clients by
  // origin rather than by `X-API-Key` (see the note at the top of userApi.js).
  // So a request made from Node looks like a non-browser caller and is refused
  // before any handler runs. Wrapping fetch to add the header reproduces what a
  // browser does; it does not change what is under test, which is the client's
  // request shape and its reading of the response.
  const bareFetch = globalThis.fetch
  globalThis.fetch = (url, init = {}) => bareFetch(url, {
    ...init,
    headers: { ...(init.headers || {}), origin: 'http://127.0.0.1' },
  })
  const login = await fetch(`${backendUrl}/v1/auth/sessions`, {
    method: 'POST',
    headers: { 'content-type': 'application/json', origin: 'http://127.0.0.1' },
    body: JSON.stringify({ identifier: USERNAME, password: PASSWORD }),
  })
  assert.equal(login.status, 200, 'login failed')
  const session = await login.json()
  // The token is written the way the app writes it, so the client's own
  // `getToken()` finds it. `_saveSessionPayload` is the legacy client's private
  // method and does not exist here, which is why this is not a call into it.
  localStorage.setItem('quickdesk_user_access_token', session.access_token)
  assert.equal(userApi.getToken(), session.access_token,
    'the client could not read back the session it was given')
})

after(async () => {
  await stopProcess(signaling)
  await stopProcess(redis)
  if (tempRoot) await rm(tempRoot, { recursive: true, force: true })
})

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

async function token() {
  const r = await call('POST', '/v1/auth/sessions', { identifier: USERNAME, password: PASSWORD })
  return r.json.access_token
}

// The client sends the right query string and reads the envelope back. Driven
// through `userApi.fetchAuditLogs` rather than a hand-rolled fetch, because the
// request shape is what a rename or a contract drift would break.
test('the client reads back a transfer it reported', async () => {
  const tk = await token()
  const transferId = `client-audit-clip-${Date.now()}`
  const report = await call('POST', '/v1/me/clipboard-audits', {
    device_id: DEVICE_ID,
    transfer_id: transferId,
    direction: 'upload',
    status: 'completed',
    mime_type: 'text/plain',
    bytes: 64,
  }, tk)
  assert.equal(report.status, 204, `report status ${report.status}`)

  const read = await userApi.fetchAuditLogs({ limit: 50 })
  assert.equal(read.ok, true, `read failed: ${JSON.stringify(read)}`)
  const items = read.data?.items ?? []
  const row = items.find((candidate) => candidate.resource_id === transferId)
  assert.ok(row, `the reported transfer was not readable through the client: ${JSON.stringify(items)}`)
  assert.equal(row.action, 'clipboard.sync.upload')

  // The byte count survives the round trip through the client, which is the
  // number an investigator actually needs.
  const details = JSON.parse(row.details)
  assert.equal(details.bytes, 64)
})

// The action filter is an allow-list on the server. A client that offers what
// the server rejects turns a typo into an error the user cannot act on, so the
// client method has to be usable with a valid action and the server has to
// refuse an invalid one.
test('the client can filter by action and the server refuses an unknown one', async () => {
  const tk = await token()
  const read = await userApi.fetchAuditLogs({ action: 'clipboard.sync.upload', limit: 20 })
  assert.equal(read.ok, true, `filtered read failed: ${JSON.stringify(read)}`)
  for (const row of read.data?.items ?? []) {
    assert.equal(row.action, 'clipboard.sync.upload', `the filter let through ${row.action}`)
  }

  const raw = await fetch(`${backendUrl}/v1/me/audit-logs?action=recording.delete`, {
    headers: { origin: 'http://127.0.0.1', authorization: `Bearer ${tk}` },
  })
  assert.equal(raw.status, 400,
    `an action outside this trail was answered ${raw.status}, want 400`)
})

// Cursor pagination works through the client, so a busy account is not one
// unbounded response.
test('the client pages with the cursor', async () => {
  const first = await userApi.fetchAuditLogs({ limit: 1 })
  assert.equal(first.ok, true, `first page failed: ${JSON.stringify(first)}`)
  const items = first.data?.items ?? []
  assert.equal(items.length, 1, `asked for 1 and got ${items.length}`)

  const cursor = first.data?.next_cursor
  if (!cursor) return
  const second = await userApi.fetchAuditLogs({ limit: 1, cursor })
  assert.equal(second.ok, true, `second page failed: ${JSON.stringify(second)}`)
  const next = second.data?.items ?? []
  assert.equal(next.length, 1)
  assert.notEqual(next[0].id, items[0].id, 'the second page repeated the first row')
})

// The client sends no query string when nothing is asked for, so the server's
// own defaults apply rather than a client-side guess about them.
test('a read with no filters sends no query string', async () => {
  const requested = []
  const wrapped = globalThis.fetch
  globalThis.fetch = async (url, init) => {
    requested.push(String(url))
    return wrapped(url, init)
  }
  try {
    await userApi.fetchAuditLogs()
  } finally {
    globalThis.fetch = wrapped
  }
  const last = requested[requested.length - 1] || ''
  assert.ok(last.endsWith('/v1/me/audit-logs'),
    `an unfiltered read sent a query string: ${last}`)
})
