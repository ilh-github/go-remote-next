import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import net from 'node:net'
import { chromium } from 'playwright'
import { createServer } from 'vite'

const WEB_ROOT = new URL('../../', import.meta.url).pathname
const DEVICES = [
  {
    device_id: 'device-a',
    remark: 'Device A',
    online: true,
    logged_in: true,
    access_code: 'code-a',
  },
  {
    device_id: 'device-b',
    remark: 'Device B',
    online: true,
    logged_in: true,
    access_code: 'code-b',
  },
  {
    device_id: 'device-c',
    remark: 'Device C',
    online: false,
    logged_in: false,
    access_code: 'code-c',
  },
]

let browser
let vite
let baseUrl

before(async () => {
  browser = await chromium.launch({
    channel: process.env.QD_BROWSER_CHANNEL || 'chrome',
    headless: true,
  })
  const port = await findFreePort()
  vite = await createServer({
    root: WEB_ROOT,
    logLevel: 'silent',
    server: {
      host: '127.0.0.1',
      port,
      strictPort: true,
    },
  })
  await vite.listen()
  baseUrl = `http://127.0.0.1:${port}`
})

after(async () => {
  await vite?.close()
  await browser?.close()
})

test('screen wall manages live tiles from device APIs through a real browser', async () => {
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await context.addInitScript(({ devices }) => {
    localStorage.setItem('quickdesk_lang', 'en-US')
    localStorage.setItem('quickdesk_user_access_token', 'wall-e2e-token')
    localStorage.setItem(
      'quickdesk_wall_devices',
      JSON.stringify(devices.map((device) => device.device_id)),
    )
  }, { devices: DEVICES })

  const page = await context.newPage()
  const pageErrors = []
  const verifyCounts = new Map()
  page.on('pageerror', (error) => pageErrors.push(error.message))

  await page.route('**/v1/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    if (request.method() === 'GET' && url.pathname === '/v1/me/devices') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ items: DEVICES }),
      })
      return
    }

    const verify = url.pathname.match(/^\/v1\/devices\/([^/]+)\/access-code:verify$/)
    if (request.method() === 'POST' && verify) {
      const deviceId = decodeURIComponent(verify[1])
      verifyCounts.set(deviceId, (verifyCounts.get(deviceId) || 0) + 1)
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          signal_token: `signal-${deviceId}-${verifyCounts.get(deviceId)}`,
          expires_at: '2026-10-18T12:00:00Z',
        }),
      })
      return
    }

    await route.fulfill({
      status: 404,
      contentType: 'application/problem+json',
      body: JSON.stringify({ detail: `Unhandled E2E request: ${request.method()} ${url.pathname}` }),
    })
  })
  await page.route(/\/remote\.html(?:\?.*)?$/, async (route) => {
    const url = new URL(route.request().url())
    const deviceId = url.searchParams.get('device') || ''
    await route.fulfill({
      status: 200,
      contentType: 'text/html',
      body: embeddedSessionFixture(deviceId),
    })
  })

  try {
    await page.goto(`${baseUrl}/test/fixtures/wall.html`, { waitUntil: 'domcontentloaded' })
    await page.waitForFunction(() => window.__wallMounted === true)
    await page.waitForFunction(
      () => document.querySelectorAll('.wall-tile').length === 3
        && document.querySelectorAll('.tile-badge.live').length === 2,
    )

    assert.equal(await page.locator('.wall-tile').count(), 3)
    assert.equal(await page.locator('.wall-count').textContent(), '3 device(s) on the wall')
    assert.equal(await tile(page, 'Device C').getByText('Device is offline').count(), 1)
    assert.deepEqual(
      await page.evaluate(() => JSON.parse(localStorage.getItem('quickdesk_wall_devices'))),
      ['device-a', 'device-b', 'device-c'],
    )

    await tile(page, 'Device A').getByTitle('Play audio').click()
    await embedded(page, 'device-a').waitForFunction(() => window.__wallAudioEnabled === true)
    assert.equal(await tile(page, 'Device A').evaluate((element) => element.classList.contains('audio')), true)

    await tile(page, 'Device B').getByTitle('Play audio').click()
    await embedded(page, 'device-a').waitForFunction(() => window.__wallAudioEnabled === false)
    await embedded(page, 'device-b').waitForFunction(() => window.__wallAudioEnabled === true)
    assert.equal(await page.locator('.wall-tile.audio').count(), 1)
    assert.equal(await tile(page, 'Device B').evaluate((element) => element.classList.contains('audio')), true)

    await tile(page, 'Device A').getByTitle('Unlock input').click()
    await embedded(page, 'device-a').waitForFunction(() => window.__wallControlArmed === true)
    await tile(page, 'Device B').getByTitle('Unlock input').click()
    await embedded(page, 'device-a').waitForFunction(() => window.__wallControlArmed === false)
    await embedded(page, 'device-b').waitForFunction(() => window.__wallControlArmed === true)
    assert.equal(await page.locator('.wall-tile.armed').count(), 1)
    assert.equal(await tile(page, 'Device B').evaluate((element) => element.classList.contains('armed')), true)

    await tile(page, 'Device A').getByTitle('Enlarge').click()
    assert.equal(await tile(page, 'Device A').evaluate((element) => element.classList.contains('focused')), true)
    await tile(page, 'Device A').getByTitle('Exit enlarge').click()
    assert.equal(await page.locator('.wall-tile.focused').count(), 0)

    const firstCount = verifyCounts.get('device-a')
    await tile(page, 'Device A').getByTitle('Reconnect').click()
    await waitFor(() => verifyCounts.get('device-a') === firstCount + 1)
    await page.waitForFunction(
      () => document.querySelectorAll('.tile-badge.live').length === 2,
    )

    await page.locator('.wall-cols select').selectOption('3')
    await page.waitForFunction(
      () => document.querySelector('.wall-grid')?.style.getPropertyValue('--wall-cols') === '3',
    )
    assert.equal(await page.evaluate(() => localStorage.getItem('quickdesk_wall_columns')), '3')

    await tile(page, 'Device B').getByTitle('Remove from wall').click()
    await page.waitForFunction(() => document.querySelectorAll('.wall-tile').length === 2)
    assert.equal(await page.locator('.wall-tile.audio').count(), 0)
    assert.deepEqual(
      await page.evaluate(() => JSON.parse(localStorage.getItem('quickdesk_wall_devices'))),
      ['device-a', 'device-c'],
    )

    const refreshedCount = verifyCounts.get('device-a')
    await page.getByRole('button', { name: /Refresh/ }).click()
    await waitFor(() => verifyCounts.get('device-a') === refreshedCount + 1)
    assert.deepEqual(pageErrors, [])
  } finally {
    await context.close()
  }
})

function tile(page, name) {
  return page.locator('.wall-tile', {
    has: page.locator('.tile-name', { hasText: name }),
  })
}

function embedded(page, deviceId) {
  return page.frame({
    url: (url) => url.pathname.endsWith('/remote.html')
      && url.searchParams.get('device') === deviceId,
  })
}

async function waitFor(predicate, timeout = 5000) {
  const deadline = Date.now() + timeout
  while (Date.now() < deadline) {
    if (predicate()) return
    await new Promise((resolve) => setTimeout(resolve, 25))
  }
  throw new Error('Timed out waiting for condition')
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

function embeddedSessionFixture(deviceId) {
  return `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>${deviceId}</title></head>
<body>
  <div id="state">connecting</div>
  <script>
    const device = ${JSON.stringify(deviceId)}
    window.__wallAudioEnabled = false
    window.__wallControlArmed = false
    window.__wallMessages = []
    window.addEventListener('message', (event) => {
      if (event.origin !== location.origin) return
      if (event.data?.device !== device) return
      window.__wallMessages.push(event.data)
      if (event.data.type === 'quickdesk-wall-audio') {
        window.__wallAudioEnabled = event.data.enabled === true
      }
      if (event.data.type === 'quickdesk-wall-control') {
        window.__wallControlArmed = event.data.armed === true
      }
    })
    parent.postMessage({
      type: 'quickdesk-wall',
      device,
      state: 'live',
      detail: '1280x720',
    }, location.origin)
  </script>
</body>
</html>`
}
