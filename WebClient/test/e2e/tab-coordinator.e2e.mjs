import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { createReadStream } from 'node:fs'
import { stat } from 'node:fs/promises'
import http from 'node:http'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'

const WEB_ROOT = path.resolve(fileURLToPath(new URL('../../', import.meta.url)))

let browser

before(async () => {
  browser = await chromium.launch({
    channel: process.env.QD_BROWSER_CHANNEL || 'chrome',
    headless: true,
  })
})

after(async () => {
  await browser?.close()
})

test('real tabs coordinate control and audio with BroadcastChannel', async () => {
  const server = await startWebServer()
  const context = await browser.newContext()
  const pageErrors = []

  try {
    const base = `http://127.0.0.1:${server.address().port}`
    const first = await openTab(context, base, pageErrors, {
      device: 'device-a',
      tab: 'tab-a',
    })
    const second = await openTab(context, base, pageErrors, {
      device: 'device-a',
      tab: 'tab-b',
    })

    await first.evaluate(() => {
      window.__tab.claim('control')
      window.__tab.claim('audio')
    })
    assert.equal(await first.evaluate(() => window.__tab.has('control')), true)
    assert.equal(await first.evaluate(() => window.__tab.has('audio')), true)

    await second.evaluate(() => window.__tab.claim('control'))
    await first.waitForFunction(() => window.__tab.revoked.length === 1)
    assert.deepEqual(await first.evaluate(() => window.__tab.revoked[0]), {
      resource: 'control',
      ownerTabId: 'tab-b',
    })
    assert.equal(await first.evaluate(() => window.__tab.has('control')), false)
    assert.equal(await first.evaluate(() => window.__tab.has('audio')), true)
    assert.equal(await second.evaluate(() => window.__tab.has('control')), true)

    await first.evaluate(() => window.__tab.claim('control'))
    await second.waitForFunction(() => window.__tab.revoked.length === 1)
    assert.equal(await second.evaluate(() => window.__tab.has('control')), false)
    assert.equal(await first.evaluate(() => window.__tab.has('control')), true)

    await first.evaluate(() => window.__tab.release('control'))
    await second.evaluate(() => window.__tab.claim('control'))
    await second.waitForTimeout(50)
    assert.equal(await second.evaluate(() => window.__tab.has('control')), true)
    assert.equal(await first.evaluate(() => window.__tab.revoked.length), 1)

    const isolated = await openTab(context, base, pageErrors, {
      device: 'device-b',
      tab: 'tab-c',
    })
    await isolated.evaluate(() => window.__tab.claim('control'))
    await first.waitForTimeout(150)
    assert.equal(await first.evaluate(() => window.__tab.revoked.length), 1)
    assert.equal(await first.evaluate(() => window.__tab.has('audio')), true)

    await second.evaluate(() => window.__tab.claim('audio'))
    await first.waitForFunction(() => window.__tab.revoked.length === 2)
    assert.deepEqual(await first.evaluate(() => window.__tab.revoked[1]), {
      resource: 'audio',
      ownerTabId: 'tab-b',
    })
    assert.equal(await first.evaluate(() => window.__tab.has('audio')), false)
    assert.deepEqual(pageErrors, [])
  } finally {
    await context.close()
    await closeServer(server)
  }
})

async function openTab(context, base, pageErrors, { device, tab }) {
  const page = await context.newPage()
  page.on('pageerror', (error) => pageErrors.push(error.message))
  await page.goto(
    `${base}/tab-coordinator-fixture.html?device=${encodeURIComponent(device)}&tab=${encodeURIComponent(tab)}`,
    { waitUntil: 'domcontentloaded' },
  )
  await page.waitForFunction(() => window.__tabReady === true)
  return page
}

async function startWebServer() {
  const server = http.createServer(async (req, res) => {
    const url = new URL(req.url, 'http://127.0.0.1')
    if (url.pathname === '/tab-coordinator-fixture.html') {
      res.writeHead(200, {
        'Content-Type': 'text/html; charset=utf-8',
        'Cache-Control': 'no-store',
      })
      res.end(FIXTURE_HTML)
      return
    }

    const filePath = path.resolve(WEB_ROOT, `.${decodeURIComponent(url.pathname)}`)
    const relative = path.relative(WEB_ROOT, filePath)
    if (relative.startsWith('..') || path.isAbsolute(relative)) {
      res.writeHead(403)
      res.end('forbidden')
      return
    }
    try {
      const info = await stat(filePath)
      if (!info.isFile()) throw new Error('not a file')
      res.writeHead(200, {
        'Content-Type': filePath.endsWith('.js')
          ? 'text/javascript; charset=utf-8'
          : 'application/octet-stream',
        'Cache-Control': 'no-store',
      })
      createReadStream(filePath).pipe(res)
    } catch {
      res.writeHead(404)
      res.end('not found')
    }
  })
  await new Promise((resolve, reject) => {
    server.once('error', reject)
    server.listen(0, '127.0.0.1', resolve)
  })
  return server
}

async function closeServer(server) {
  await new Promise((resolve, reject) => {
    server.close((error) => (error ? reject(error) : resolve()))
  })
}

const FIXTURE_HTML = `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>TabCoordinator E2E</title></head>
<body>
  <script type="module">
    import { TabCoordinator } from '/js/ui/tab-coordinator.js'

    const params = new URLSearchParams(location.search)
    const coordinator = new TabCoordinator({
      deviceId: params.get('device') || 'device-a',
      tabId: params.get('tab') || 'tab-a',
      channelName: 'quickdesk:e2e-tab-bus',
    })
    const revoked = []
    coordinator.addEventListener('revoked', (event) => {
      revoked.push(event.detail)
    })
    coordinator.start()

    window.__tab = {
      revoked,
      claim: (resource) => coordinator.claim(resource),
      release: (resource) => coordinator.release(resource),
      has: (resource) => coordinator.hasClaim(resource),
      destroy: () => coordinator.destroy(),
    }
    window.__tabReady = true
  </script>
</body>
</html>`
