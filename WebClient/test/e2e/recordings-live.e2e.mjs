import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import net from 'node:net'
import { chromium } from 'playwright'
import { createServer } from 'vite'

const WEB_ROOT = new URL('../../', import.meta.url).pathname

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
    server: { host: '127.0.0.1', port, strictPort: true },
  })
  await vite.listen()
  baseUrl = `http://127.0.0.1:${port}`
})

after(async () => {
  await vite?.close()
  await browser?.close()
})

// The recordings page refreshed only in onMounted. The server publishes
// recording.created / updated / deleted, so a recording another controller
// finishes or deletes never reached the open page - and nothing was wrong with
// the page itself, which is why it survived: it looks complete until somebody
// else touches a recording.
//
// This drives the real page through the real dispatcher, because a unit test on
// the dispatch switch would pass while the page still ignored the event.
test('a recording another controller changes reaches the open page', async () => {
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await context.addInitScript(() => {
    window.__recordingFetches = 0
    window.__recordingsToasts = []
    const originalFetch = window.fetch
    window.fetch = (input, init) => {
      const url = typeof input === 'string' ? input : input.url
      if (String(url).includes('/v1/me/recordings')) {
        window.__recordingFetches += 1
        return Promise.resolve(new Response(JSON.stringify({
          items: [{
            id: 4242,
            device_id: '700000001',
            status: 'completed',
            size_bytes: 2048,
            duration_seconds: 12,
            started_at: '2026-10-26T01:00:00Z',
            ended_at: '2026-10-26T01:00:12Z',
            artifact_url: '',
            thumbnail_url: '',
          }],
          next_cursor: '',
        }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
      }
      return originalFetch(input, init)
    }
  })

  const page = await context.newPage()
  const pageErrors = []
  page.on('pageerror', (error) => pageErrors.push(error.message))

  await page.goto(`${baseUrl}/test/fixtures/recordings.html`, { waitUntil: 'domcontentloaded' })
  await page.waitForFunction(() => window.__recordingsMounted === true)

  // The page renders its list first, so a refresh is observable as a second
  // fetch rather than as a silent no-op.
  await page.waitForFunction(() => window.__recordingFetches >= 1, null, { timeout: 15000 })

  // Fire the event the way the server does, through the same dispatcher the real
  // socket uses.
  await page.evaluate(async () => {
    const { userSync } = await import('/src/api/userSync.js')
    userSync._onMessage({
      data: JSON.stringify({
        type: 'recording.deleted',
        server_rev: 99,
        data: { recording_id: 4242, device_id: '700000001' },
      }),
    })
  })

  await page.waitForFunction(() => window.__recordingFetches >= 2, null, { timeout: 10000 })

  const fetches = await page.evaluate(() => window.__recordingFetches)
  assert.ok(fetches >= 2, `the list was fetched ${fetches} times; a change should refresh it`)

  assert.deepEqual(pageErrors, [])
  await context.close()
})

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
