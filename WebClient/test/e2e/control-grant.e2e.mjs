import assert from 'node:assert/strict'
import { execFile, spawn } from 'node:child_process'
import { mkdir, mkdtemp, rm } from 'node:fs/promises'
import net from 'node:net'
import os from 'node:os'
import path from 'node:path'
import { promisify } from 'node:util'
import { fileURLToPath } from 'node:url'
import { after, before, test } from 'node:test'

// QD-314, signaling half: two real controllers on one exclusive device.
//
// The WebClient E2E proves the owner can choose exclusive control; this proves
// the server actually enforces the choice it was given. The interesting part is
// the takeover: the second controller must be handed the grant *and* the first
// must be told it lost it, because a holder that is never notified keeps typing
// into a machine it no longer owns - which is the whole failure mode the feature
// exists to prevent.

const runFile = promisify(execFile)
const WEB_ROOT = path.resolve(fileURLToPath(new URL('../../', import.meta.url)))
const REPO_ROOT = path.resolve(WEB_ROOT, '..')
const SERVER_ROOT = path.join(REPO_ROOT, 'SignalingServer')
const API_KEY = 'control-grant-e2e-api-key'
const ADMIN_PASSWORD = 'ControlGrantAdmin-123!'
const USERNAME = 'control-grant-e2e'
const PASSWORD = 'ControlGrantUser-123!'
const DEVICE_ID = '456789123'
const DEVICE_NAME = 'Control Grant Host'
const ACCESS_CODE = '135790'

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

/**
 * One controller's signaling socket. Frames are queued so a test can assert on
 * the exact order the server produced them rather than racing the wire.
 */
class SignalClient {
  constructor(signalToken, clientId) {
    this.clientId = clientId
    this.frames = []
    this.ws = new WebSocket(`${backendUrl.replace(/^http/, 'ws')}/v1/realtime/signal`)
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
  tempRoot = await mkdtemp(path.join(os.tmpdir(), 'quickdesk-control-grant-e2e-'))
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
})

after(async () => {
  await stopProcess(signaling)
  await stopProcess(redis)
  if (tempRoot) await rm(tempRoot, { recursive: true, force: true })
})

async function login() {
  const r = await call('POST', '/v1/auth/sessions', { identifier: USERNAME, password: PASSWORD })
  assert.equal(r.status, 200, `login failed: ${JSON.stringify(r.json)}`)
  return r.json.access_token
}

async function setControlMode(token, mode) {
  const r = await call('PUT', `/v1/me/devices/${DEVICE_ID}/access-policy`, {
    remote_disabled: false,
    allow_ips: [], deny_ips: [], max_sessions: 0,
    allowed_windows: [], timezone: '', exception_dates: [],
    approval_required: false, approval_mode: 'off', approval_trusted_ips: [],
    control_mode: mode,
  }, token)
  assert.equal(r.status, 200, `setting control_mode=${mode} failed: ${JSON.stringify(r.json)}`)
  return r.json
}

async function connectController(token, clientId) {
  const verified = await call('POST', `/v1/devices/${DEVICE_ID}/access-code:verify`, {
    code: ACCESS_CODE,
    client_id: clientId,
  })
  assert.equal(verified.status, 200, `verify failed for ${clientId}: ${JSON.stringify(verified.json)}`)
  const client = new SignalClient(verified.json.signal_token, clientId)
  await client.authenticate()
  return client
}

test('an exclusive device hands control to the second controller and tells the first it lost it', async () => {
  const token = await login()
  await setControlMode(token, 'exclusive')

  const first = await connectController(token, 'ctrl-first')
  const grant = await first.waitFor('control_grant', 5000)
  assert.ok(grant, 'the first controller got no control_grant')
  assert.equal(grant.control_mode, 'exclusive')
  assert.equal(grant.can_control, true, 'the first controller must be told it may drive')
  assert.equal(grant.control_holder, 'ctrl-first')

  const second = await connectController(token, 'ctrl-second')

  // The new holder is told it holds control, naming nobody but itself.
  const secondGrant = await second.waitFor('control_grant', 5000)
  assert.equal(secondGrant?.type, 'control_grant')
  assert.equal(secondGrant.can_control, true)
  assert.equal(secondGrant.control_holder, 'ctrl-second')

  // The previous holder is told it lost control, and who took it.
  const revoked = await first.waitFor('control_revoked', 5000)
  assert.equal(revoked?.type, 'control_revoked', `first controller got ${revoked?.type}, want control_revoked`)
  assert.equal(revoked.control_mode, 'exclusive')
  assert.equal(revoked.control_holder, 'ctrl-second',
    'the revoke must name the new holder so the UI can say who took over')

  first.close()
  second.close()
  await new Promise((resolve) => setTimeout(resolve, 300))
})

test('a shared device grants everybody and never notifies a revoke', async () => {
  const token = await login()
  await setControlMode(token, 'shared')

  const first = await connectController(token, 'ctrl-shared-a')
  const second = await connectController(token, 'ctrl-shared-b')

  for (const [client, name] of [[first, 'first'], [second, 'second']]) {
    const grant = await client.waitFor('control_grant', 5000)
    assert.equal(grant?.type, 'control_grant', `${name}: got ${grant?.type}`)
    assert.equal(grant.control_mode, 'shared')
    assert.equal(grant.can_control, true, `${name} must be allowed to drive in shared mode`)
    assert.equal(grant.control_holder, '', 'shared mode names no holder')
  }

  // Neither was notified of losing anything, because neither had it to lose.
  assert.equal(first.ofType('control_revoked').length, 0, 'shared mode sent a revoke')
  assert.equal(second.ofType('control_revoked').length, 0, 'shared mode sent a revoke')

  first.close()
  second.close()
  await new Promise((resolve) => setTimeout(resolve, 300))
})

test('a released exclusive machine frees control for the next asker', async () => {
  const token = await login()
  await setControlMode(token, 'exclusive')

  const first = await connectController(token, 'ctrl-release-a')
  await first.waitFor('control_grant', 5000)

  // Disconnecting is the release: the grant is tied to the socket that holds it.
  first.close()
  await new Promise((resolve) => setTimeout(resolve, 500))

  const second = await connectController(token, 'ctrl-release-b')
  const grant = await second.waitFor('control_grant', 5000)
  assert.ok(grant, 'the next controller got no control_grant')
  assert.equal(grant.can_control, true, 'a machine whose holder left must be drivable again')
  assert.equal(grant.control_holder, 'ctrl-release-b')

  second.close()
  await new Promise((resolve) => setTimeout(resolve, 300))
})
