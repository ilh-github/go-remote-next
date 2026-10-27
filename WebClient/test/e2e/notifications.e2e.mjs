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

// QD-904: NotificationsPage (1652 lines, the largest page in the app) had 18 model
// tests and no browser E2E. What is asserted here is the part a model test cannot
// see: that the two safety controls reach the request, and that a preview is never
// reported as a replay. Reporting "requeued 40" over a dry run would tell the
// operator the backlog was drained when nothing was touched.
test('notifications page keeps a preview and a replay distinguishable', async () => {
  const state = {
    channels: [
      { id: 7, name: 'Ops webhook', url: 'https://example.invalid/hook', events: ['device.offline.sustained'], enabled: true },
    ],
    deliveries: [
      { id: 101, event_type: 'device.offline.sustained', status: 'failed', attempts: 4, created_at: '2026-10-27T09:00:00Z' },
      { id: 102, event_type: 'device.offline.sustained', status: 'failed', attempts: 2, created_at: '2026-10-27T09:10:00Z' },
    ],
    retries: [],
  }

  const context = await browser.newContext({ viewport: { width: 1440, height: 2000 } })
  await context.addInitScript(() => {
    localStorage.setItem('quickdesk_user_access_token', 'notif-e2e-token')
    window.__notifToasts = []
    window.__showLoginCalls = 0
  })
  const page = await context.newPage()
  const pageErrors = []
  page.on('pageerror', (error) => pageErrors.push(error.message))

  // Decisions are observed from the page, so they are mirrored there.
  const mirrorRetries = () => page.evaluate((r) => { window.__notifRetries = r }, state.retries)

  await page.route('**/v1/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const method = request.method()
    const json = () => request.postDataJSON()

    if (method === 'GET' && url.pathname === '/v1/me/devices') {
      await fulfillJSON(route, { items: [{ device_id: '700000001', remark: 'Lab Bench', online: true }] })
      return
    }
    if (method === 'GET' && url.pathname === '/v1/me/notification-channels') {
      await fulfillJSON(route, { items: state.channels })
      return
    }
    if (method === 'GET' && url.pathname === '/v1/me/notification-quiet-hours') {
      await fulfillJSON(route, { enabled: false })
      return
    }
    if (method === 'GET' && /\/notification-channels\/\d+\/deliveries(\?|$)/.test(url.pathname)) {
      await fulfillJSON(route, { items: state.deliveries })
      return
    }
    if (method === 'POST' && /\/deliveries:retry-failed$/.test(url.pathname)) {
      const body = json()
      state.retries.push(body)
      mirrorRetries()
      const dry = body.dry_run === true
      await fulfillJSON(route, dry
        ? { dry_run: true, matched: 40, requeued: 0 }
        : { dry_run: false, matched: 40, requeued: 40 })
      return
    }
    if (method === 'POST' && /\/deliveries\/\d+\/retry$/.test(url.pathname)) {
      await fulfillJSON(route, { id: 101, status: 'pending' })
      return
    }
    if (method === 'POST' && /\/deliveries\/\d+\/ack$/.test(url.pathname)) {
      await fulfillJSON(route, { id: 101, status: 'acked' })
      return
    }
    await fulfillJSON(route, {})
  })

  await page.goto(`${baseUrl}/test/fixtures/notifications.html`, { waitUntil: 'domcontentloaded' })
  await page.waitForFunction(() => window.__notifMounted === true)
  await page.waitForSelector('.channel-row')

  // Open the deliveries panel for the only channel.
  await page.locator('article').first().locator('button', { hasText: /^Deliveries$/i }).click()
  await page.waitForSelector('.delivery-table')

  // The preview is on by default. A mis-click must not drain the backlog.
  const dryToggle = page.locator('.dry-run-toggle input[type=checkbox]')
  assert.equal(await dryToggle.isChecked(), true,
    'the preview was not on by default, so a mis-click would drain the backlog')

  // --- Preview: nothing is touched, and the toast says so ---
  // `.head-actions` is used by the channel-list header too, so the buttons are
  // picked by their label rather than by position. The bulk action is the one
  // whose label flips with the preview toggle.
  // The bulk action's label flips with the preview toggle, so it is identified by
  // either of the two strings it can carry. Both are matched as substrings.
  // The deliveries section is identified by its heading, which is unique on the
  // page. Scoping matters: `.head-actions` also appears in the channel-list
  // header, and the table has its own per-row retry button with a stable label.
  const deliverySection = page.locator('section.surface').filter({
    hasText: 'Delivery history',
  })
  const bulkButton = deliverySection.locator('.head-actions button').filter({
    hasText: /Preview replay|Retry this channel/,
  })
  await bulkButton.first().click()

  await page.waitForFunction(() => (window.__notifRetries || []).length > 0)
  assert.equal(state.retries.length, 1)
  assert.equal(state.retries[0].dry_run, true,
    'the preview toggle did not reach the request')

  await page.waitForFunction(() => window.__notifToasts.length > 0)
  let toast = (await page.evaluate(() => window.__notifToasts)).at(-1)
  assert.match(toast.message, /40/, 'the preview toast does not report how many rows it covered')
  // The word that matters: a preview says what would happen, not what did.
  assert.doesNotMatch(toast.message, /requeued|re-?queued/i,
    `a preview was reported as a replay: ${toast.message}`)

  // --- A real replay: the toggle off, the toast says it happened ---
  // The label flips with the toggle, which is the visible half of the same state
  // the request carries.
  await dryToggle.uncheck()
  assert.match(await bulkButton.first().innerText(), /Retry this channel/,
    'the button label did not follow the preview toggle')
  await bulkButton.first().click()

  await page.waitForFunction(() => (window.__notifRetries || []).length === 2)
  assert.equal(state.retries[1].dry_run, false,
    'turning the preview off did not reach the request')

  // Wait for the replay's own toast rather than reading the last one present:
  // the preview's toast is still the newest one for a moment, and a replay that
  // has not been reported yet must not be mistaken for one that was misreported.
  await page.waitForFunction(() => window.__notifToasts.some(
    (entry) => /requeued|re-?queued/i.test(entry.message)))
  toast = (await page.evaluate(() => window.__notifToasts)).at(-1)
  assert.match(toast.message, /40/, 'the replay toast does not report the count')
  assert.match(toast.message, /requeued|re-?queued/i,
    `a real replay was not reported as one: ${toast.message}`)

  assert.deepEqual(pageErrors, [])
})

async function fulfillJSON(route, body) {
  await route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify(body),
  })
}

function findFreePort() {
  return new Promise((resolve, reject) => {
    const server = net.createServer()
    server.once('error', reject)
    server.listen(0, '127.0.0.1', () => {
      const { port } = server.address()
      server.close(() => resolve(port))
    })
  })
}
