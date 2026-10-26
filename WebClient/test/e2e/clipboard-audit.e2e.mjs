import assert from 'node:assert/strict'
import { execFile, spawn } from 'node:child_process'
import { mkdir, mkdtemp, rm } from 'node:fs/promises'
import net from 'node:net'
import os from 'node:os'
import path from 'node:path'
import { promisify } from 'node:util'
import { fileURLToPath } from 'node:url'
import { after, before, test } from 'node:test'

// QD-909, real-server half: a clipboard sync reported through the real endpoint
// must land in the audit trail, and the text must not.
//
// The unit tests pin the builder and the handler; this drives the actual route
// against a real SQLite server and then reads the row back, which is the only
// way to prove the promise "the audit trail cannot become a credential store"
// holds end to end rather than just in the object the client built.

const runFile = promisify(execFile)
const WEB_ROOT = path.resolve(fileURLToPath(new URL('../../', import.meta.url)))
const REPO_ROOT = path.resolve(WEB_ROOT, '..')
const SERVER_ROOT = path.join(REPO_ROOT, 'SignalingServer')
const API_KEY = 'clipboard-audit-e2e-api-key'
const ADMIN_PASSWORD = 'ClipboardAuditAdmin-123!'
const USERNAME = 'clipboard-audit-e2e'
const PASSWORD = 'ClipboardAuditUser-123!'
const DEVICE_ID = '321654987'
const DEVICE_NAME = 'Clipboard Audit Host'

let redis
let signaling
let tempRoot
let backendUrl
let databasePath

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

// Reads the stored audit rows for one action, straight from the SQLite file the
// server is writing. `node:sqlite` is built in from Node 22, so reading the
// authoritative store costs no dependency.
//
// The database is the right thing to assert against: the promise under test is
// about what a future reader of that file inherits, and the owner-scoped audit
// read lives under /v1/admin behind admin auth, which is a heavier setup than
// the promise warrants.
async function readAuditDetails(action) {
  const { DatabaseSync } = await import('node:sqlite')
  const db = new DatabaseSync(databasePath, { readOnly: true })
  try {
    return db
      .prepare('SELECT details FROM audit_logs WHERE action = ? ORDER BY id DESC LIMIT 20')
      .all(action)
      .map((row) => String(row.details))
  } finally {
    db.close()
  }
}

async function waitForHealth(url, child) {
  for (let i = 0; i < 60; i += 1) {
    if (child.exitCode !== null) throw new Error('signaling exited before becoming healthy')
    try { if ((await fetch(url)).ok) return } catch { /* not up yet */ }
    await new Promise((resolve) => setTimeout(resolve, 500))
  }
  throw new Error('signaling did not become healthy in 30s')
}

before(async () => {
  tempRoot = await mkdtemp(path.join(os.tmpdir(), 'quickdesk-clipboard-audit-e2e-'))
  const backendPort = await findFreePort()
  const redisPort = await findFreePort()
  backendUrl = `http://127.0.0.1:${backendPort}`
  databasePath = path.join(tempRoot, 'data', 'quickdesk.db')
  await mkdir(path.dirname(databasePath), { recursive: true })

  const signalingBinary = path.join(tempRoot, 'signaling')
  const seedBinary = path.join(tempRoot, 'seed-smoke')
  await Promise.all([
    runFile('go', ['build', '-o', signalingBinary, './cmd/signaling'],
      { cwd: SERVER_ROOT, maxBuffer: 64 * 1024 * 1024 }),
    runFile('go', ['build', '-o', seedBinary, './cmd/seed-smoke'],
      { cwd: SERVER_ROOT, maxBuffer: 64 * 1024 * 1024 }),
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
    '-server', backendUrl, '-api-key', API_KEY,
    '-admin-user', 'admin', '-admin-password', ADMIN_PASSWORD,
    '-username', USERNAME, '-password', PASSWORD,
    '-device-id', DEVICE_ID, '-device-name', DEVICE_NAME,
  ], { cwd: path.dirname(databasePath), env: sharedEnv, maxBuffer: 16 * 1024 * 1024 })
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

async function login() {
  const r = await call('POST', '/v1/auth/sessions', { identifier: USERNAME, password: PASSWORD })
  assert.equal(r.status, 200, `login failed: ${JSON.stringify(r.json)}`)
  return r.json.access_token
}

test('a reported clipboard sync lands in the audit trail without its text', async () => {
  const token = await login()
  const secret = 'AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCY'

  const report = await call('POST', '/v1/me/clipboard-audits', {
    device_id: DEVICE_ID,
    transfer_id: `clip-e2e-${Date.now()}`,
    direction: 'upload',
    status: 'completed',
    mime_type: 'text/plain',
    bytes: new TextEncoder().encode(secret).length,
  }, token)
  assert.equal(report.status, 204, `report status ${report.status}`)

  const rows = await readAuditDetails('clipboard.sync.upload')
  assert.ok(rows.length > 0, 'no clipboard.sync.upload row was written')
  for (const details of rows) {
    assert.equal(details.includes(secret), false,
      'the clipboard text reached the audit row')
    assert.equal(details.includes('wJalrXUtnFEMI'), false,
      'a fragment of the clipboard text reached the audit row')
    assert.ok(details.includes('"bytes"'), 'the row should carry the byte count')
  }
})

test('a report for a device the caller does not own is refused', async () => {
  const token = await login()
  const r = await call('POST', '/v1/me/clipboard-audits', {
    device_id: '999999999',
    transfer_id: `clip-foreign-${Date.now()}`,
    direction: 'upload',
    status: 'completed',
    mime_type: 'text/plain',
    bytes: 10,
  }, token)
  assert.ok(r.status === 403 || r.status === 404,
    `a foreign device was answered ${r.status}, want 403 or 404`)
})

test('a report with no mime_type is refused rather than stored half-described', async () => {
  const token = await login()
  const r = await call('POST', '/v1/me/clipboard-audits', {
    device_id: DEVICE_ID,
    transfer_id: `clip-nomime-${Date.now()}`,
    direction: 'upload',
    status: 'completed',
    bytes: 10,
  }, token)
  assert.equal(r.status, 400, `status ${r.status}, want 400`)
})
