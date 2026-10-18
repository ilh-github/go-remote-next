import assert from 'node:assert/strict'
import { execFile, spawn } from 'node:child_process'
import { createReadStream } from 'node:fs'
import { mkdir, mkdtemp, rm, stat } from 'node:fs/promises'
import http from 'node:http'
import net from 'node:net'
import os from 'node:os'
import path from 'node:path'
import { promisify } from 'node:util'
import { fileURLToPath } from 'node:url'
import { after, before, test } from 'node:test'
import { chromium } from 'playwright'

const runFile = promisify(execFile)
const WEB_ROOT = path.resolve(fileURLToPath(new URL('../../', import.meta.url)))
const REPO_ROOT = path.resolve(WEB_ROOT, '..')
const SERVER_ROOT = path.join(REPO_ROOT, 'SignalingServer')
const DIST_ROOT = path.join(WEB_ROOT, 'dist')
const API_KEY = 'selfhost-e2e-api-key'
const ADMIN_PASSWORD = 'SelfhostAdmin-123!'
const USERNAME = 'selfhost-e2e'
const PASSWORD = 'SelfhostUser-123!'
const DEVICE_ID = '123456789'
const DEVICE_NAME = 'Selfhost Host'

let browser
let redis
let signaling
let webServer
let tempRoot
let backendUrl
let frontendUrl

before(async () => {
  tempRoot = await mkdtemp(path.join(os.tmpdir(), 'quickdesk-selfhost-e2e-'))
  const backendPort = await findFreePort()
  const redisPort = await findFreePort()
  const frontendPort = await findFreePort()
  backendUrl = `http://127.0.0.1:${backendPort}`
  frontendUrl = `http://127.0.0.1:${frontendPort}`
  const databasePath = path.join(tempRoot, 'data', 'quickdesk.db')
  await mkdir(path.dirname(databasePath), { recursive: true })

  const signalingBinary = path.join(tempRoot, 'signaling')
  const seedBinary = path.join(tempRoot, 'seed-smoke')
  await Promise.all([
    runFile('go', ['build', '-o', signalingBinary, './cmd/signaling'], {
      cwd: SERVER_ROOT,
      maxBuffer: 16 * 1024 * 1024,
    }),
    runFile('go', ['build', '-o', seedBinary, './cmd/seed-smoke'], {
      cwd: SERVER_ROOT,
      maxBuffer: 16 * 1024 * 1024,
    }),
  ])

  redis = startProcess('redis-server', [
    '--bind', '127.0.0.1',
    '--port', String(redisPort),
    '--notify-keyspace-events', 'Ex',
    '--save', '',
    '--appendonly', 'no',
    '--dir', tempRoot,
  ], { label: 'redis' })
  await waitForRedis(redisPort)

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
      ALLOWED_ORIGINS: frontendUrl,
      ADMIN_USER: 'admin',
      ADMIN_PASSWORD,
    },
  })
  await waitForHealth(`${backendUrl}/health`, signaling)
  await waitForRedisInstance(redisPort)

  await runFile(seedBinary, [
    '-server', backendUrl,
    '-api-key', API_KEY,
    '-admin-user', 'admin',
    '-admin-password', ADMIN_PASSWORD,
    '-username', USERNAME,
    '-password', PASSWORD,
    '-device-id', DEVICE_ID,
    '-device-name', DEVICE_NAME,
  ], {
    cwd: path.dirname(databasePath),
    env: sharedEnv,
    maxBuffer: 16 * 1024 * 1024,
  })

  webServer = await startStaticServer(DIST_ROOT, frontendPort)
  browser = await chromium.launch({
    channel: process.env.QD_BROWSER_CHANNEL || 'chrome',
    headless: true,
  })
})

after(async () => {
  await browser?.close()
  await closeServer(webServer)
  await stopProcess(signaling)
  await stopProcess(redis)
  if (tempRoot) await rm(tempRoot, { recursive: true, force: true })
})

test('production WebClient uses a real self-hosted SQLite SignalingServer', async () => {
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await context.addInitScript(({ server, deviceId }) => {
    localStorage.setItem('quickdesk_lang', 'en-US')
    localStorage.setItem('quickdesk_signaling_url', server)
    window.__openedRemote = []
    window.open = (url, target) => {
      window.__openedRemote.push({ url: String(url), target })
      return null
    }
    sessionStorage.setItem('quickdesk_e2e_device_id', deviceId)
  }, {
    server: backendUrl.replace(/^http/, 'ws'),
    deviceId: DEVICE_ID,
  })
  const page = await context.newPage()
  const pageErrors = []
  const responses = []
  let realtimeOpened = false
  let realtimeSnapshot = false
  page.on('pageerror', (error) => pageErrors.push(error.message))
  page.on('websocket', (socket) => {
    const url = new URL(socket.url())
    if (url.pathname !== '/v1/realtime/events') return
    realtimeOpened = true
    socket.on('framereceived', (event) => {
      try {
        const frame = JSON.parse(String(event.payload))
        if (
          frame.type === 'snapshot'
          && frame.data?.devices?.some((device) => device.device_id === DEVICE_ID)
        ) {
          realtimeSnapshot = true
        }
      } catch { /* binary or non-JSON frame */ }
    })
  })
  page.on('response', (response) => {
    const url = new URL(response.url())
    if (url.pathname.startsWith('/v1/')) {
      responses.push({ path: url.pathname, status: response.status() })
    }
  })

  try {
    await page.goto(`${frontendUrl}/#/remote`, { waitUntil: 'domcontentloaded' })
    await page.locator('.nav-header').click()
    await page.locator('input[autocomplete="username"]').fill(USERNAME)
    await page.locator('input[autocomplete="current-password"]').fill(PASSWORD)
    await page.locator('.dialog .btn-primary').click()
    await page.waitForFunction(
      () => !!localStorage.getItem('quickdesk_user_access_token'),
    )
    await page.locator('.dialog').waitFor({ state: 'detached' })

    await page.locator('a.nav-item', { hasText: 'Devices' }).click()
    await page.waitForFunction(() => location.hash === '#/devices')
    const deviceRow = page.locator('.device-item', {
      has: page.locator('.device-name', { hasText: DEVICE_NAME }),
    })
    await deviceRow.waitFor()
    assert.equal(await deviceRow.locator('.device-status.online').count(), 1)
    assert.equal(await deviceRow.getByText('Offline', { exact: true }).count(), 0)

    await page.getByRole('button', { name: 'Device groups', exact: true }).click()
    await page.locator('.group-list-pane .group-create-button').click()
    await page.locator('.group-fields input.form-input').first().fill('Selfhost Lab')
    await page.locator('.group-description-input').fill('Created against the real SQLite server')
    await page.locator('.group-editor .form-actions').first()
      .getByRole('button', { name: 'New group', exact: true })
      .click()
    await page.locator('.group-list-row', { hasText: 'Selfhost Lab' }).waitFor()
    await page.getByRole('button', { name: 'Select all', exact: true }).click()
    await page.getByRole('button', { name: 'Save members (1)', exact: true }).click()
    await page.waitForFunction(() => {
      const chip = [...document.querySelectorAll('.group-filter-chip')]
        .find((element) => element.textContent.includes('Selfhost Lab'))
      return chip?.querySelector('.group-filter-count')?.textContent.trim() === '1'
    })
    await page.getByTitle('Close').click()

    await deviceRow.getByRole('button', { name: 'Connect', exact: true }).click()
    await page.waitForFunction(() => window.__openedRemote.length === 1)
    const opened = await page.evaluate(() => {
      const item = window.__openedRemote[0]
      const parsed = new URL(item.url, location.href)
      const token = parsed.searchParams.get('st')
      return {
        device: parsed.searchParams.get('device'),
        server: parsed.searchParams.get('server'),
        token,
        target: item.target,
        handoff: JSON.parse(sessionStorage.getItem(`quickdesk_remote_handoff__${token}`)),
      }
    })
    assert.equal(opened.device, DEVICE_ID)
    assert.equal(opened.server, backendUrl.replace(/^http/, 'ws'))
    assert.match(opened.token, /^[A-Za-z0-9._-]+$/)
    assert.equal(opened.target, `quickdesk_${DEVICE_ID}`)
    assert.equal(opened.handoff.access_code, '135790')
    assert.equal(opened.handoff.device_id, DEVICE_ID)

    assert.equal(findStatus(responses, '/v1/auth/sessions'), 200)
    assert.ok(
      findStatus(responses, '/v1/me/devices') === 200
        || (realtimeOpened && realtimeSnapshot),
      `device list came from neither HTTP nor realtime: ${JSON.stringify(responses)}`,
    )
    assert.equal(findStatus(responses, '/v1/me/device-groups'), 200)
    assert.equal(
      responses.some((response) => (
        response.path === '/v1/me/device-groups' && response.status === 201
      )),
      true,
    )
    assert.equal(
      findStatus(responses, '/v1/me/device-groups/1/devices'),
      200,
    )
    assert.equal(
      findStatus(responses, `/v1/devices/${DEVICE_ID}/access-code:verify`),
      200,
    )
    assert.deepEqual(pageErrors, [])
  } finally {
    await context.close()
  }
})

function startProcess(command, args, { cwd = REPO_ROOT, env = process.env, label }) {
  const child = spawn(command, args, {
    cwd,
    env,
    stdio: ['ignore', 'pipe', 'pipe'],
  })
  child.output = ''
  child.stdout.on('data', (chunk) => { child.output += chunk })
  child.stderr.on('data', (chunk) => { child.output += chunk })
  child.label = label
  return child
}

async function waitForRedis(port) {
  const deadline = Date.now() + 10000
  while (Date.now() < deadline) {
    if (redis?.exitCode !== null) {
      throw new Error(`redis exited early:\n${redis?.output || ''}`)
    }
    try {
      const reply = await redisPing(port)
      if (reply === 'PONG') return
    } catch { /* retry */ }
    await sleep(50)
  }
  throw new Error(`timed out waiting for Redis on ${port}`)
}

function redisPing(port) {
  return new Promise((resolve, reject) => {
    const socket = net.createConnection({ host: '127.0.0.1', port })
    let data = ''
    socket.setTimeout(1000)
    socket.once('connect', () => socket.write('PING\r\n'))
    socket.on('data', (chunk) => {
      data += chunk
      if (data.includes('PONG')) {
        socket.end()
        resolve('PONG')
      }
    })
    socket.once('timeout', () => {
      socket.destroy()
      reject(new Error('redis ping timeout'))
    })
    socket.once('error', reject)
  })
}

async function waitForHealth(url, processInfo) {
  const deadline = Date.now() + 60000
  while (Date.now() < deadline) {
    if (processInfo.exitCode !== null) {
      throw new Error(`signaling exited early:\n${processInfo.output}`)
    }
    try {
      const response = await fetch(url)
      if (response.ok) {
        const body = await response.json()
        if (body.status === 'ok' && body.components?.sqlite === 'ok') return
      }
    } catch { /* retry */ }
    await sleep(100)
  }
  throw new Error(`timed out waiting for ${url}\n${processInfo.output}`)
}

async function waitForRedisInstance(port) {
  const deadline = Date.now() + 10000
  while (Date.now() < deadline) {
    const { stdout } = await runFile('redis-cli', [
      '-h', '127.0.0.1',
      '-p', String(port),
      '--scan',
      '--pattern', 'qd:presence:instance:*',
    ])
    if (stdout.trim()) return
    await sleep(100)
  }
  throw new Error('signaling instance never registered in Redis')
}

async function startStaticServer(root, port) {
  const server = http.createServer(async (req, res) => {
    const url = new URL(req.url, 'http://127.0.0.1')
    let pathname = decodeURIComponent(url.pathname)
    if (pathname === '/') pathname = '/index.html'
    let filePath = path.resolve(root, `.${pathname}`)
    const relative = path.relative(root, filePath)
    if (relative.startsWith('..') || path.isAbsolute(relative)) {
      res.writeHead(403)
      res.end('forbidden')
      return
    }
    try {
      const info = await stat(filePath)
      if (!info.isFile()) throw new Error('not a file')
    } catch {
      if (path.extname(pathname)) {
        res.writeHead(404)
        res.end('not found')
        return
      }
      filePath = path.join(root, 'index.html')
    }
    res.writeHead(200, {
      'Content-Type': MIME_TYPES[path.extname(filePath)] || 'application/octet-stream',
      'Cache-Control': 'no-store',
    })
    createReadStream(filePath).pipe(res)
  })
  await new Promise((resolve, reject) => {
    server.once('error', reject)
    server.listen(port, '127.0.0.1', resolve)
  })
  return server
}

async function closeServer(server) {
  if (!server) return
  await new Promise((resolve, reject) => {
    server.close((error) => (error ? reject(error) : resolve()))
  })
}

async function stopProcess(processInfo) {
  if (!processInfo || processInfo.exitCode !== null) return
  await new Promise((resolve) => {
    const timer = setTimeout(() => {
      processInfo.kill('SIGKILL')
      resolve()
    }, 5000)
    processInfo.once('exit', () => {
      clearTimeout(timer)
      resolve()
    })
    processInfo.kill('SIGTERM')
  })
}

function findStatus(responses, pathName) {
  return responses.find((response) => response.path === pathName)?.status || 0
}

async function findFreePort() {
  const server = net.createServer()
  await new Promise((resolve, reject) => {
    server.once('error', reject)
    server.listen(0, '127.0.0.1', resolve)
  })
  const port = server.address().port
  await new Promise((resolve, reject) => {
    server.close((error) => (error ? reject(error) : resolve()))
  })
  return port
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

const MIME_TYPES = {
  '.css': 'text/css; charset=utf-8',
  '.html': 'text/html; charset=utf-8',
  '.ico': 'image/x-icon',
  '.js': 'text/javascript; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.png': 'image/png',
}
