import assert from 'node:assert/strict'
import { execFile, spawn } from 'node:child_process'
import { mkdir, mkdtemp, rm } from 'node:fs/promises'
import net from 'node:net'
import os from 'node:os'
import path from 'node:path'
import { promisify } from 'node:util'
import { fileURLToPath } from 'node:url'
import { after, before, test } from 'node:test'

// QD-314, fleet half: two real signaling processes, one exclusive device.
//
// control-grant.e2e.mjs proves the server enforces exclusive control inside one
// process. That is not where the bug is. The holder lives in that process's
// memory, so a second instance admitting the second controller answers "you may
// drive it" as well - two people typing into the same machine, invisibly, which
// is the exact failure the feature exists to prevent. Only two real processes
// sharing one Redis can show that, so this test starts two.
//
// Both instances share the SQLite database and Redis; they differ only in port
// and INSTANCE_ID, which is how a real deployment is shaped.

const runFile = promisify(execFile)
const WEB_ROOT = path.resolve(fileURLToPath(new URL('../../', import.meta.url)))
const REPO_ROOT = path.resolve(WEB_ROOT, '..')
const SERVER_ROOT = path.join(REPO_ROOT, 'SignalingServer')
const API_KEY = 'control-fleet-e2e-api-key'
const ADMIN_PASSWORD = 'ControlFleetAdmin-123!'
const USERNAME = 'control-fleet-e2e'
const PASSWORD = 'ControlFleetUser-123!'
const DEVICE_ID = '567890234'
const DEVICE_NAME = 'Control Fleet Host'
const ACCESS_CODE = '135790'

let redis
let signalingA
let signalingB
let tempRoot
let urlA
let urlB

async function findFreePort() {
  const server = net.createServer()
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve))
  const { port } = server.address()
  await new Promise((resolve) => server.close(resolve))
  return port
}

function startProcess(command, args, { label, cwd, env }) {
  const child = spawn(command, args, { cwd, env, stdio: ['ignore', 'pipe', 'pipe'] })
  // FLEET_E2E_DEBUG surfaces the child processes' output, which is how a
  // failing boot is diagnosed: a silent spawn tells you nothing about whether
  // Redis came up or the second instance lost a SQLITE_BUSY race.
  if (process.env.FLEET_E2E_DEBUG) {
    child.stdout.on('data', (d) => console.log(`[${label}] ${String(d).trim()}`))
    child.stderr.on('data', (d) => console.log(`[${label}-err] ${String(d).trim()}`))
    return child
  }
  child.stdout.on('data', () => {})
  child.stderr.on('data', () => {})
  return child
}

async function stopProcess(child) {
  if (!child || child.exitCode !== null) return
  child.kill('SIGTERM')
  await new Promise((resolve) => {
    const timer = setTimeout(() => {
      child.kill('SIGKILL')
      resolve()
    }, 5000)
    child.on('exit', () => {
      clearTimeout(timer)
      resolve()
    })
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

async function call(baseUrl, method, requestPath, body, token) {
  const response = await fetch(`${baseUrl}${requestPath}`, {
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

/**
 * One controller's signaling socket. Frames are queued so a test can assert on
 * the exact order the server produced them rather than racing the wire.
 */
class SignalClient {
  constructor(baseUrl, signalToken, clientId) {
    this.clientId = clientId
    this.baseUrl = baseUrl
    this.frames = []
    this.ws = new WebSocket(`${baseUrl.replace(/^http/, 'ws')}/v1/realtime/signal`)
    this.ws.addEventListener('message', (event) => {
      let frame = null
      try { frame = JSON.parse(String(event.data)) } catch { return }
      this.frames.push(frame)
    })
    this.ready = new Promise((resolve, reject) => {
      this.ws.addEventListener('open', resolve)
      this.ws.addEventListener('error', () => reject(new Error('signal socket failed to open')))
    })
    this.signalToken = signalToken
  }

  async authenticate() {
    await this.ready
    this.ws.send(JSON.stringify({
      type: 'auth',
      signal_token: this.signalToken,
      role: 'client',
      device_id: DEVICE_ID,
      client_id: this.clientId,
      client_id: this.clientId,
    }))
    // The handshake frame is consumed here, so a caller's first `next()` is the
    // first frame the server sent *after* the socket was admitted - which is
    // where the control grant lives.
    const auth = await this.waitFor('auth_ok', 10000)
    assert.ok(auth, `${this.clientId}: no auth_ok arrived`)
  }

  /**
   * Waits for a frame of one type and removes it from the queue. Polling rather
   * than a push-based waiter, because the order the server emits `auth_ok` and
   * `control_grant` is close enough that a waiter registered between them would
   * race - and a race in the test says nothing about the server.
   */
  async waitFor(type, timeoutMs = 5000) {
    const deadline = Date.now() + timeoutMs
    for (;;) {
      const index = this.frames.findIndex((frame) => frame.type === type)
      if (index >= 0) return this.frames.splice(index, 1)[0]
      if (Date.now() >= deadline) return null
      await new Promise((resolve) => setTimeout(resolve, 20))
    }
  }

  ofType(type) {
    return this.frames.filter((frame) => frame.type === type)
  }

  close() {
    try { this.ws.close() } catch { /* already gone */ }
  }
}

before(async () => {
  tempRoot = await mkdtemp(path.join(os.tmpdir(), 'quickdesk-control-fleet-e2e-'))
  const portA = await findFreePort()
  const portB = await findFreePort()
  const redisPort = await findFreePort()
  urlA = `http://127.0.0.1:${portA}`
  urlB = `http://127.0.0.1:${portB}`
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
    DB_DRIVER: 'sqlite',
    DB_PATH: databasePath,
    REDIS_DRIVER: 'redis',
    REDIS_HOST: '127.0.0.1',
    REDIS_PORT: String(redisPort),
    LOG_DIR: path.join(tempRoot, 'logs'),
    API_KEY,
    ALLOWED_ORIGINS: 'http://127.0.0.1',
    ADMIN_USER: 'admin',
    ADMIN_PASSWORD,
    // A short TTL bounds the debris a crashed instance leaves behind; no test
    // below waits on expiry, so this is not part of what is being asserted.
    CONTROL_GRANT_TTL_SECONDS: '60',
  }

  // Started one at a time on purpose. Both instances automigrate the same
  // SQLite file at boot (sqlite forces it), and two concurrent AutoMigrate runs
  // on one file is SQLITE_BUSY - a property of the test fixture, not of the
  // fleet, so the second process waits for the first to be serving.
  signalingA = startProcess(signalingBinary, [], {
    cwd: path.dirname(databasePath),
    label: 'signaling-a',
    env: { ...sharedEnv, SERVER_PORT: String(portA), INSTANCE_ID: 'fleet-a' },
  })
  await waitForHealth(`${urlA}/health`, signalingA)

  signalingB = startProcess(signalingBinary, [], {
    cwd: path.dirname(databasePath),
    label: 'signaling-b',
    env: { ...sharedEnv, SERVER_PORT: String(portB), INSTANCE_ID: 'fleet-b' },
  })
  await waitForHealth(`${urlB}/health`, signalingB)

  await runFile(seedBinary, [
    '-server', urlA,
    '-api-key', API_KEY,
    '-admin-user', 'admin',
    '-admin-password', ADMIN_PASSWORD,
    '-username', USERNAME,
    '-password', PASSWORD,
    '-device-id', DEVICE_ID,
    '-device-name', DEVICE_NAME,
  ], { cwd: path.dirname(databasePath), env: sharedEnv, maxBuffer: 16 * 1024 * 1024 })
})

after(async () => {
  await stopProcess(signalingA)
  await stopProcess(signalingB)
  await stopProcess(redis)
  if (tempRoot) await rm(tempRoot, { recursive: true, force: true })
})

async function login() {
  const r = await call(urlA, 'POST', '/v1/auth/sessions', { identifier: USERNAME, password: PASSWORD })
  if (r.status !== 200) {
    throw new Error(`login failed: status=${r.status} body=${JSON.stringify(r.json)}`)
  }
  return r.json.access_token
}

async function setControlMode(token, mode) {
  const r = await call(urlA, 'PUT', `/v1/me/devices/${DEVICE_ID}/access-policy`, {
    remote_disabled: false,
    allow_ips: [], deny_ips: [], max_sessions: 0,
    allowed_windows: [], timezone: '', exception_dates: [],
    approval_required: false, approval_mode: 'off', approval_trusted_ips: [],
    control_mode: mode,
  }, token)
  assert.equal(r.status, 200, `setting control_mode=${mode} failed: ${JSON.stringify(r.json)}`)
  return r.json
}

// The access-code check is a write against the shared database, so the token it
// mints works on either instance. Which instance the socket lands on is decided
// by the URL the client connects to, which is the variable under test.
async function connectController(baseUrl, clientId) {
  const verified = await call(baseUrl, 'POST', `/v1/devices/${DEVICE_ID}/access-code:verify`, {
    code: ACCESS_CODE,
    client_id: clientId,
  })
  if (verified.status !== 200) {
    throw new Error(`verify failed for ${clientId} on ${baseUrl}: status=${verified.status} body=${JSON.stringify(verified.json)}`)
  }
  const client = new SignalClient(baseUrl, verified.json.signal_token, clientId)
  await client.authenticate()
  return client
}

test('an exclusive device has one holder across two real instances', async () => {
  const token = await login()
  await setControlMode(token, 'exclusive')

  // Deliberately different instances: this is the case the single-process test
  // cannot reach, and the one that produced two people driving one machine.
  const first = await connectController(urlA, 'ctrl-a')
  const grant = await first.waitFor('control_grant', 5000)
  assert.ok(grant, 'the first controller got no control_grant')
  assert.equal(grant.control_mode, 'exclusive')
  assert.equal(grant.can_control, true)
  assert.equal(grant.control_holder, 'ctrl-a')

  const second = await connectController(urlB, 'ctrl-b')
  const secondGrant = await second.waitFor('control_grant', 5000)
  assert.ok(secondGrant, 'the second controller got no control_grant')
  // B asked, so B holds it now.
  assert.equal(secondGrant.control_holder, 'ctrl-b')
  assert.equal(secondGrant.can_control, true)

  // And A is told it lost it, naming the new holder - even though that holder
  // was recorded by a process A has never spoken to.
  const revoked = await first.waitFor('control_revoked', 5000)
  assert.equal(revoked?.type, 'control_revoked',
    `the displaced holder got ${revoked?.type}, want control_revoked`)
  assert.equal(revoked.control_holder, 'ctrl-b',
    'the revoke must name the new holder so the UI can say who took over')

  first.close()
  second.close()
  await new Promise((resolve) => setTimeout(resolve, 300))
})

// The holder is a fact about the device, not about a process, so a takeover from
// either side has to be visible from the other.
test('a takeover from either instance is visible to the other', async () => {
  const token = await login()
  await setControlMode(token, 'exclusive')

  const first = await connectController(urlB, 'ctrl-x')
  await first.waitFor('control_grant', 5000)

  // Take it back from the *other* instance.
  const second = await connectController(urlA, 'ctrl-y')
  const secondGrant = await second.waitFor('control_grant', 5000)
  assert.equal(secondGrant.control_holder, 'ctrl-y')
  assert.equal(secondGrant.can_control, true)

  const revoked = await first.waitFor('control_revoked', 5000)
  assert.equal(revoked?.type, 'control_revoked',
    `the displaced holder got ${revoked?.type}, want control_revoked`)
  assert.equal(revoked.control_holder, 'ctrl-y')

  first.close()
  second.close()
  await new Promise((resolve) => setTimeout(resolve, 300))
})

// A holder that disconnected must free the machine for the other instance. If
// the release only lands in the holding process, the next controller waits for
// a turn that never comes.
test('a release frees the machine for the other instance', async () => {
  const token = await login()
  await setControlMode(token, 'exclusive')

  const first = await connectController(urlA, 'ctrl-hold')
  await first.waitFor('control_grant', 5000)
  first.close()
  // The socket teardown is what releases, and it runs on A. Give it a moment
  // rather than asserting on a timing that is not what is under test.
  await new Promise((resolve) => setTimeout(resolve, 500))

  const second = await connectController(urlB, 'ctrl-next')
  const grant = await second.waitFor('control_grant', 5000)
  assert.equal(grant.control_holder, 'ctrl-next',
    'a machine whose holder left must be free for the next controller')
  assert.equal(grant.can_control, true)

  second.close()
  await new Promise((resolve) => setTimeout(resolve, 300))
})
