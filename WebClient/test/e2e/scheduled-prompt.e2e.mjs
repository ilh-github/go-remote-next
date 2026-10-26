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

// QD-607's connect_prompt was published by the server and subscribed by a
// notification channel, and then stopped: nothing in the client consumed it. This
// drives the real page over the real realtime path, because a unit test on the
// dispatch switch would pass while the page still ignored the event.
test('a scheduled connection prompt reaches the page the owner is looking at', async () => {
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await context.addInitScript(() => {
    window.__scheduledToasts = []
    window.__scheduledRequests = []
    const originalFetch = window.fetch
    window.fetch = (input, init) => {
      const url = typeof input === 'string' ? input : input.url
      window.__scheduledRequests.push({ url: String(url), method: init?.method || 'GET' })
      if (String(url).includes('/v1/me/scheduled-tasks')) {
        return Promise.resolve(new Response(JSON.stringify({
          items: [{
            id: 77,
            kind: 'connect_prompt',
            enabled: true,
            next_run_at: '2026-10-27T01:00:00Z',
            repeat_seconds: 0,
            max_attempts: 1,
            attempts: 0,
            schedule: '',
            timezone: '',
            skip_holidays: false,
            last_run_at: null,
            last_status: '',
            last_error: '',
          }],
          supported_kinds: ['sign_out_all_sessions', 'revoke_all_remote_sessions', 'connect_prompt'],
          supported_schedules: ['daily@HH:MM'],
        }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
      }
      return originalFetch(input, init)
    }
  })

  const page = await context.newPage()
  const pageErrors = []
  page.on('pageerror', (error) => pageErrors.push(error.message))

  await page.goto(`${baseUrl}/test/fixtures/scheduled.html`, { waitUntil: 'domcontentloaded' })
  await page.waitForFunction(() => window.__scheduledMounted === true)

  // The kind has to render as a label, not as its wire name.
  await page.waitForFunction(() => document.body.textContent.includes('Prompt to connect'), null,
    { timeout: 15000 })

  // Fire the event the way the server does, through the same dispatcher the real
  // socket uses.
  await page.evaluate(async () => {
    const { userSync } = await import('/src/api/userSync.js')
    userSync._onMessage({
      data: JSON.stringify({
        type: 'device.connect_prompted',
        server_rev: 99,
        data: { task_id: 77 },
      }),
    })
  })

  // The toast is the part the owner sees. Without it the prompt is a server-side
  // fact nobody is told about, which is the whole reason this round exists.
  await page.waitForFunction(
    () => (window.__scheduledToasts || []).some((toast) => /scheduled connection is due/i.test(toast.message)),
    null,
    { timeout: 10000 },
  )

  const toasts = await page.evaluate(() => window.__scheduledToasts)
  assert.ok(
    toasts.some((toast) => /scheduled connection is due/i.test(toast.message)),
    `no prompt toast; got ${JSON.stringify(toasts)}`,
  )

  // And the list has to refresh, so the owner can see which task it was. The page
  // debounces a refresh by 150ms, so this waits for the second fetch rather than
  // racing it - a test that checks too early fails on timing, not on behaviour.
  await page.waitForFunction(
    () => window.__scheduledRequests.filter((request) => (
      request.url.includes('/v1/me/scheduled-tasks')
    )).length >= 2,
    null,
    { timeout: 10000 },
  )

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
