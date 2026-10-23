import assert from 'node:assert/strict'
import { execFile, spawn } from 'node:child_process'
import { mkdir, mkdtemp, rm } from 'node:fs/promises'
import net from 'node:net'
import os from 'node:os'
import path from 'node:path'
import { promisify } from 'node:util'
import { fileURLToPath } from 'node:url'
import { after, before, test } from 'node:test'
import { chromium } from 'playwright'

// QD-314: the owner-facing half of exclusive control.
//
// The server side grants and announces control; this drives the one place an
// owner can actually choose it. The interesting risk is the full-replace
// endpoint: a form that PUT only `control_mode` would wipe the IP lists, the
// session cap, the windows and the approval settings in the same request, so the
// test writes a policy with real values in those fields first and asserts they
// survive the switch.

const runFile = promisify(execFile)
const WEB_ROOT = path.resolve(fileURLToPath(new URL('../../', import.meta.url)))
const REPO_ROOT = path.resolve(WEB_ROOT, '..')
const SERVER_ROOT = path.join(REPO_ROOT, 'SignalingServer')
const DIST_ROOT = path.join(WEB_ROOT, 'dist')
const API_KEY = 'control-mode-e2e-api-key'
const ADMIN_PASSWORD = 'ControlModeAdmin-123!'
const USERNAME = 'control-mode-e2e'
const PASSWORD = 'ControlModeUser-123!'
const DEVICE_ID = '987654321'
const DEVICE_NAME = 'Control Mode Host'

// The server only admits requests from an allowed origin, and `fetch` from node
// sends none - so the HTTP half of the assertions has to declare the same origin
// the browser uses. Without it every call is a 403 that says nothing about the
// feature under test.
const ORIGIN_HEADERS = () => ({ origin: frontendUrl })

let browser
let redis
let signaling
let webServer
let tempRoot
let backendUrl
let frontendUrl

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
  child.on('exit', (code) => {
    if (code !== 0 && code !== null) console.log(`[${label}] exited with ${code}`)
  })
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

async function startStaticServer(root, port) {
  const { createServer } = await import('node:http')
  const { readFile } = await import('node:fs/promises')
  const mime = {
    '.html': 'text/html; charset=utf-8',
    '.js': 'text/javascript; charset=utf-8',
    '.mjs': 'text/javascript; charset=utf-8',
    '.css': 'text/css; charset=utf-8',
    '.json': 'application/json; charset=utf-8',
    '.svg': 'image/svg+xml',
    '.png': 'image/png',
    '.ico': 'image/x-icon',
    '.woff2': 'font/woff2',
  }
  const server = createServer(async (request, response) => {
    try {
      const url = new URL(request.url, `http://127.0.0.1:${port}`)
      const relative = url.pathname === '/' ? '/index.html' : decodeURIComponent(url.pathname)
      const filePath = path.join(root, relative)
      if (!filePath.startsWith(root)) {
        response.writeHead(403).end('forbidden')
        return
      }
      const body = await readFile(filePath)
      const type = mime[path.extname(filePath).toLowerCase()] || 'application/octet-stream'
      response.writeHead(200, { 'content-type': type }).end(body)
    } catch {
      response.writeHead(404).end('not found')
    }
  })
  await new Promise((resolve) => server.listen(port, '127.0.0.1', resolve))
  return server
}

async function closeServer(server) {
  if (!server) return
  await new Promise((resolve) => server.close(resolve))
}

before(async () => {
  tempRoot = await mkdtemp(path.join(os.tmpdir(), 'quickdesk-control-mode-e2e-'))
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
      ALLOWED_ORIGINS: frontendUrl,
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

// Logs in through the real form and returns the page plus the bearer token the
// client stored, so the HTTP half of the assertions can talk to the same server.
async function loginAndOpenDevices(context) {
  await context.addInitScript(({ server }) => {
    localStorage.setItem('quickdesk_lang', 'en-US')
    localStorage.setItem('quickdesk_signaling_url', server)
  }, { server: backendUrl.replace(/^http/, 'ws') })
  const page = await context.newPage()
  await page.goto(`${frontendUrl}/#/devices`, { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('.nav-header', { timeout: 15000 })
  await page.locator('.nav-header').click()
  await page.locator('input[autocomplete="username"]').fill(USERNAME)
  await page.locator('input[autocomplete="current-password"]').fill(PASSWORD)
  await page.locator('.dialog .btn-primary').click()
  await page.waitForFunction(() => !!localStorage.getItem('quickdesk_user_access_token'))
  await page.waitForSelector('.device-item', { timeout: 15000 })
  return page
}

test('the owner can switch a device to exclusive control without losing the rest of the policy', async () => {
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  const page = await loginAndOpenDevices(context)
  const pageErrors = []
  page.on('pageerror', (error) => pageErrors.push(error.message))

  // Seed a policy with real values in the fields a one-field form would destroy,
  // so the switch has something to preserve.
  const token = await page.evaluate(() => localStorage.getItem('quickdesk_user_access_token'))
  const seed = await fetch(`${backendUrl}/v1/me/devices/${DEVICE_ID}/access-policy`, {
    method: 'PUT',
    headers: { 'content-type': 'application/json', authorization: `Bearer ${token}`, ...ORIGIN_HEADERS() },
    body: JSON.stringify({
      remote_disabled: false,
      allow_ips: ['203.0.113.4'],
      deny_ips: [],
      max_sessions: 3,
      allowed_windows: [],
      timezone: 'Asia/Shanghai',
      exception_dates: [],
      approval_required: false,
      approval_mode: 'off',
      approval_trusted_ips: [],
      control_mode: 'shared',
    }),
  })
  assert.equal(seed.status, 200, 'seeding the policy must succeed')

  // The dialog opens on the stored mode, not on an assumption.
  await page.locator('.device-item', { hasText: DEVICE_ID })
    .locator('button[title="Control"]').click()
  await page.waitForSelector('.control-mode-option')
  const sharedChecked = await page.locator(`input[value="shared"]`).isChecked()
  assert.equal(sharedChecked, true, 'a shared device must open on shared')

  // Nothing to save yet: the button is disabled until the choice actually changes.
  const saveDisabledBefore = await page
    .locator('.dialog .btn-primary')
    .isDisabled()
  assert.equal(saveDisabledBefore, true, 'save must be disabled before a change')

  await page.locator(`input[value="exclusive"]`).check()
  const saveEnabled = await page.locator('.dialog .btn-primary').isEnabled()
  assert.equal(saveEnabled, true, 'save must enable once the choice changes')

  await page.locator('.dialog .btn-primary').click()
  await page.waitForSelector('.dialog-overlay', { state: 'detached', timeout: 15000 })

  // The stored row is now exclusive, and every other field survived.
  const stored = await fetch(`${backendUrl}/v1/me/devices/${DEVICE_ID}/access-policy`, {
    headers: { authorization: `Bearer ${token}`, ...ORIGIN_HEADERS() },
  })
  assert.equal(stored.status, 200)
  const policy = await stored.json()
  assert.equal(policy.control_mode, 'exclusive', 'the switch must be stored')
  assert.deepEqual(policy.allow_ips, ['203.0.113.4'], 'the IP list must survive')
  assert.equal(policy.max_sessions, 3, 'the session cap must survive')
  assert.equal(policy.timezone, 'Asia/Shanghai', 'the timezone must survive')
  assert.equal(policy.approval_mode, 'off', 'the approval mode must survive')

  // Switching back is the mirror image, and it must also preserve the rest.
  await page.locator('.device-item', { hasText: DEVICE_ID })
    .locator('button[title="Control"]').click()
  await page.waitForSelector('.control-mode-option')
  assert.equal(
    await page.locator('input[value="exclusive"]').isChecked(),
    true,
    'reopening must show the mode that was stored',
  )
  await page.locator('input[value="shared"]').check()
  await page.locator('.dialog .btn-primary').click()
  await page.waitForSelector('.dialog-overlay', { state: 'detached', timeout: 15000 })

  const reverted = await (await fetch(`${backendUrl}/v1/me/devices/${DEVICE_ID}/access-policy`, {
    headers: { authorization: `Bearer ${token}`, ...ORIGIN_HEADERS() },
  })).json()
  assert.equal(reverted.control_mode, 'shared', 'switching back must be stored')
  assert.deepEqual(reverted.allow_ips, ['203.0.113.4'], 'the IP list must survive the switch back')
  assert.equal(reverted.max_sessions, 3, 'the session cap must survive the switch back')

  assert.deepEqual(pageErrors, [], `page errors: ${pageErrors.join('; ')}`)
  await context.close()
})

test('a device that never had a policy opens on shared and can still be switched', async () => {
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  const page = await loginAndOpenDevices(context)
  const token = await page.evaluate(() => localStorage.getItem('quickdesk_user_access_token'))

  // Clear whatever the previous test left so this device really has no policy.
  const cleared = await fetch(`${backendUrl}/v1/me/devices/${DEVICE_ID}/access-policy`, {
    method: 'PUT',
    headers: { 'content-type': 'application/json', authorization: `Bearer ${token}`, ...ORIGIN_HEADERS() },
    body: JSON.stringify({
      remote_disabled: false, allow_ips: [], deny_ips: [], max_sessions: 0,
      allowed_windows: [], timezone: '', exception_dates: [], approval_required: false,
      approval_mode: 'off', approval_trusted_ips: [], control_mode: 'shared',
    }),
  })
  assert.equal(cleared.status, 200)

  await page.locator('.device-item', { hasText: DEVICE_ID })
    .locator('button[title="Control"]').click()
  await page.waitForSelector('.control-mode-option')
  assert.equal(
    await page.locator('input[value="shared"]').isChecked(),
    true,
    'an unconfigured device must open on shared',
  )

  await page.locator('input[value="exclusive"]').check()
  await page.locator('.dialog .btn-primary').click()
  await page.waitForSelector('.dialog-overlay', { state: 'detached', timeout: 15000 })

  const stored = await (await fetch(`${backendUrl}/v1/me/devices/${DEVICE_ID}/access-policy`, {
    headers: { authorization: `Bearer ${token}`, ...ORIGIN_HEADERS() },
  })).json()
  assert.equal(stored.control_mode, 'exclusive')

  await context.close()
})
