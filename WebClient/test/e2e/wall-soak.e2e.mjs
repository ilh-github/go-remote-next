import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import net from 'node:net'
import { chromium } from 'playwright'
import { createServer } from 'vite'

const WEB_ROOT = new URL('../../', import.meta.url).pathname
const DEVICE_COUNT = positiveInteger(process.env.QD_WALL_DEVICES, 40)
const SOAK_SECONDS = positiveInteger(process.env.QD_WALL_SOAK_SECONDS, 12)
const SAMPLE_MS = positiveInteger(process.env.QD_WALL_SAMPLE_MS, 1000)
const MIN_MAIN_FPS = positiveNumber(process.env.QD_WALL_MIN_MAIN_FPS, 1)
const MAX_HEAP_GROWTH_MB = positiveNumber(process.env.QD_WALL_MAX_HEAP_GROWTH_MB, 256)
const DEVICES = Array.from({ length: DEVICE_COUNT }, (_, index) => {
  const suffix = String(index + 1).padStart(3, '0')
  return {
    device_id: `soak-${suffix}`,
    remark: `Soak ${suffix}`,
    online: true,
    logged_in: true,
    access_code: `code-${suffix}`,
  }
})

let browser
let vite
let baseUrl

before(async () => {
  browser = await chromium.launch({
    channel: process.env.QD_BROWSER_CHANNEL || 'chrome',
    headless: true,
    args: [
      '--disable-background-timer-throttling',
      '--disable-backgrounding-occluded-windows',
      '--disable-renderer-backgrounding',
    ],
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

test(
  `screen wall stays live with ${DEVICE_COUNT} viewers for ${SOAK_SECONDS}s`,
  { timeout: (SOAK_SECONDS + 900) * 1000 },
  async () => {
    const context = await browser.newContext({ viewport: { width: 1600, height: 1000 } })
    await context.addInitScript(({ devices }) => {
      localStorage.setItem('quickdesk_lang', 'en-US')
      localStorage.setItem('quickdesk_user_access_token', 'wall-soak-token')
      localStorage.setItem(
        'quickdesk_wall_devices',
        JSON.stringify(devices.map((device) => device.device_id)),
      )
      localStorage.setItem('quickdesk_wall_columns', '4')
      localStorage.setItem('quickdesk_wall_rotate_seconds', '2')
    }, { devices: DEVICES })

    const page = await context.newPage()
    const pageErrors = []
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
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            signal_token: `signal-${deviceId}`,
            expires_at: '2026-10-18T12:00:00Z',
          }),
        })
        return
      }
      await route.fulfill({
        status: 404,
        contentType: 'application/problem+json',
        body: JSON.stringify({ detail: `Unhandled request: ${request.method()} ${url.pathname}` }),
      })
    })
    await page.route(/\/remote\.html(?:\?.*)?$/, async (route) => {
      const url = new URL(route.request().url())
      await route.fulfill({
        status: 200,
        contentType: 'text/html',
        body: animatedSessionFixture(url.searchParams.get('device') || ''),
      })
    })

    const cdp = await context.newCDPSession(page)
    await cdp.send('Performance.enable')

    try {
      await page.goto(`${baseUrl}/test/fixtures/wall.html`, { waitUntil: 'domcontentloaded' })
      await page.waitForFunction(() => window.__wallMounted === true)
      await page.waitForFunction(
        ({ count }) => (
          document.querySelectorAll('.wall-tile').length === count
          && document.querySelectorAll('.tile-badge.live').length === count
        ),
        { count: DEVICE_COUNT },
        { timeout: Math.max(60000, DEVICE_COUNT * 1000) },
      )

      await page.evaluate(() => {
        window.__wallSoakMainFrames = 0
        const tick = () => {
          window.__wallSoakMainFrames += 1
          requestAnimationFrame(tick)
        }
        requestAnimationFrame(tick)
      })

      await clickByName(page, 'Start rotation')
      const baseline = await performanceMetrics(cdp)
      const startedAt = Date.now()
      const startedFrames = await page.evaluate(() => window.__wallSoakMainFrames)
      const samples = []
      let index = 0

      while (Date.now() - startedAt < SOAK_SECONDS * 1000) {
        const sampleStarted = Date.now()
        const target = index % Math.min(DEVICE_COUNT, 20)
        const dom = await page.evaluate(({ targetIndex }) => {
          const tiles = [...document.querySelectorAll('.wall-tile')]
          const audio = tiles.filter((tile) => tile.classList.contains('audio')).length
          const live = tiles.filter((tile) => (
            tile.querySelector('.tile-badge')?.classList.contains('live')
          )).length
          const button = [...(tiles[targetIndex]?.querySelectorAll('.tile-btn') || [])]
            .find((candidate) => candidate.title === 'Play audio')
          button?.click()
          return { live, audio, tiles: tiles.length }
        }, { targetIndex: target })
        assert.equal(dom.tiles, DEVICE_COUNT)
        assert.equal(dom.live, DEVICE_COUNT)
        assert.ok(dom.audio <= 1)

        samples.push({
          elapsedMs: Date.now() - startedAt,
          live: dom.live,
          audio: dom.audio,
        })
        index += 1
        const remaining = SAMPLE_MS - (Date.now() - sampleStarted)
        if (remaining > 0) await new Promise((resolve) => setTimeout(resolve, remaining))
      }

      await clickByName(page, 'Stop rotation')
      const elapsedMs = Date.now() - startedAt
      const endedFrames = await page.evaluate(() => window.__wallSoakMainFrames)
      const finalMetrics = await performanceMetrics(cdp)
      const mainFps = (endedFrames - startedFrames) / (elapsedMs / 1000)
      const heapGrowthMb = (
        (finalMetrics.JSHeapUsedSize || 0) - (baseline.JSHeapUsedSize || 0)
      ) / 1024 / 1024

      const sampledFrames = []
      for (const index of uniqueIndexes(DEVICE_COUNT, 3)) {
        const device = DEVICES[index]
        const frame = page.frame({
          url: (url) => (
            url.pathname.endsWith('/remote.html')
            && url.searchParams.get('device') === device.device_id
          ),
        })
        assert.ok(frame, `missing iframe for ${device.device_id}`)
        const value = await frame.evaluate(() => ({
          frames: window.__wallSoakFrames,
          audio: window.__wallAudioEnabled,
        }))
        sampledFrames.push({ device: device.device_id, frames: value.frames, audio: value.audio })
      }

      console.log('WALL_SOAK_SUMMARY', JSON.stringify({
        devices: DEVICE_COUNT,
        durationSeconds: elapsedMs / 1000,
        samples: samples.length,
        mainFps: Number(mainFps.toFixed(2)),
        heapGrowthMb: Number(heapGrowthMb.toFixed(2)),
        heapEndMb: Number(((finalMetrics.JSHeapUsedSize || 0) / 1024 / 1024).toFixed(2)),
        nodesEnd: finalMetrics.Nodes || 0,
        sampledFrames,
        pageErrors,
      }))

      assert.ok(elapsedMs >= SOAK_SECONDS * 1000)
      assert.ok(mainFps >= MIN_MAIN_FPS, `main animation fps ${mainFps.toFixed(2)}`)
      assert.ok(
        heapGrowthMb <= MAX_HEAP_GROWTH_MB,
        `JS heap grew ${heapGrowthMb.toFixed(1)} MiB`,
      )
      assert.ok(sampledFrames.every((sample) => sample.frames > 1))
      assert.deepEqual(pageErrors, [])
      assert.ok(samples.length >= 2, `only ${samples.length} samples`)
    } finally {
      await cdp.detach().catch(() => {})
      await context.close()
    }
  },
)

async function performanceMetrics(cdp) {
  const result = await cdp.send('Performance.getMetrics')
  return Object.fromEntries(result.metrics.map((metric) => [metric.name, metric.value]))
}

// A wall with hundreds of iframes makes Playwright's actionability checks slow
// and flaky, so the click goes straight to the button element.
async function clickByName(page, name) {
  return page.getByRole('button', { name, exact: true }, { timeout: 120000 })
    .evaluate((button) => button.click())
}

function uniqueIndexes(count, wanted) {
  if (count <= wanted) return Array.from({ length: count }, (_, index) => index)
  return [...new Set([
    0,
    Math.floor((count - 1) / 2),
    count - 1,
  ])]
}

function positiveInteger(value, fallback) {
  const parsed = Number(value)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : fallback
}

function positiveNumber(value, fallback) {
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : fallback
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

function animatedSessionFixture(deviceId) {
  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>${deviceId}</title>
  <style>
    html, body { margin: 0; width: 100%; height: 100%; background: #050505; overflow: hidden; }
    canvas { width: 100%; height: 100%; display: block; }
  </style>
</head>
<body>
  <canvas width="320" height="180"></canvas>
  <script>
    const device = ${JSON.stringify(deviceId)}
    const canvas = document.querySelector('canvas')
    const context = canvas.getContext('2d')
    window.__wallSoakFrames = 0
    window.__wallAudioEnabled = false
    window.__wallControlArmed = false
    const tick = (now) => {
      window.__wallSoakFrames += 1
      const hue = Math.floor((now / 20 + device.length * 13) % 360)
      context.fillStyle = 'hsl(' + hue + ' 45% 18%)'
      context.fillRect(0, 0, canvas.width, canvas.height)
      context.fillStyle = 'hsl(' + ((hue + 120) % 360) + ' 70% 55%)'
      context.fillRect((now / 10) % canvas.width, 70, 24, 40)
      requestAnimationFrame(tick)
    }
    requestAnimationFrame(tick)
    window.addEventListener('message', (event) => {
      if (event.origin !== location.origin || event.data?.device !== device) return
      if (event.data.type === 'quickdesk-wall-audio') {
        window.__wallAudioEnabled = event.data.enabled === true
      }
      if (event.data.type === 'quickdesk-wall-control') {
        window.__wallControlArmed = event.data.armed === true
      }
    })
    setInterval(() => {
      parent.postMessage({
        type: 'quickdesk-wall',
        device,
        state: 'live',
        detail: String(window.__wallSoakFrames),
      }, location.origin)
    }, 500)
  </script>
</body>
</html>`
}
