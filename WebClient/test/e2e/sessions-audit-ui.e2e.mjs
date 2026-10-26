import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import net from 'node:net'
import { chromium } from 'playwright'
import { createServer } from 'vite'

// QD-909, UI half: the audit trail has to be reachable from the page a user is
// already looking at.
//
// owner-audit.e2e.mjs proves the server honours the read; audit-log-client.e2e.mjs
// proves the client method asks for it correctly. Neither proves a user can get
// to the data, and the checklist's definition of done requires a discoverable
// entry rather than an internal API. This drives the real Sessions page in a real
// browser and asserts on what renders.
//
// The panel is deliberately asserted in three states - loaded, empty, and failed
// - because the third is the one that matters: a read that failed must not look
// like a read that found nothing.

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

async function findFreePort() {
  const server = net.createServer()
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve))
  const { port } = server.address()
  await new Promise((resolve) => server.close(resolve))
  return port
}

function fulfillJSON(route, body, status = 200) {
  return route.fulfill({
    status,
    contentType: 'application/json',
    headers: { 'access-control-allow-origin': '*' },
    body: JSON.stringify(body),
  })
}

// The rows the mocked server returns. `details` is a JSON string, exactly as the
// real server sends it, so the client's parsing path is the one under test.
function auditItems() {
  return [
    {
      id: 11,
      action: 'clipboard.sync.upload',
      resource_type: 'clipboard_sync',
      resource_id: 'clip-ui-1',
      details: JSON.stringify({
        device_id: '123456789', status: 'completed', mime_type: 'text/plain', bytes: 1200,
      }),
      ip: '203.0.113.5',
      created_at: '2026-10-26T10:00:00Z',
    },
    {
      id: 12,
      action: 'file.transfer.download',
      resource_type: 'file_transfer',
      resource_id: 'file-ui-2',
      details: JSON.stringify({
        device_id: '123456789', status: 'completed', filename: 'report.pdf', bytes_transferred: 2048,
      }),
      ip: '203.0.113.5',
      created_at: '2026-10-26T10:05:00Z',
    },
  ]
}

async function openSessionsPage({ auditMode }) {
  // A fresh context per test. localStorage and route handlers live in the
  // context, so sharing one would let a previous test's token or mock decide
  // what this one sees - which is how a passing test starts failing only when
  // run together.
  const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await context.addInitScript(() => {
    localStorage.clear()
    localStorage.setItem('quickdesk_lang', 'en-US')
    localStorage.setItem('quickdesk_user_access_token', 'sessions-audit-token')
  })
  const page = await context.newPage()
  const pageErrors = []
  page.on('pageerror', (error) => pageErrors.push(error.message))

  let auditRequests = 0
  await page.route('**/v1/**', async (route) => {
    const url = new URL(route.request().url())
    const method = route.request().method()

    if (method === 'GET' && url.pathname === '/v1/me') {
      await fulfillJSON(route, { id: 1, username: 'audit-ui' })
      return
    }
    if (method === 'GET' && url.pathname === '/v1/me/devices') {
      await fulfillJSON(route, { items: [] })
      return
    }
    if (method === 'GET' && url.pathname === '/v1/me/remote-sessions') {
      await fulfillJSON(route, { items: [], next_cursor: null })
      return
    }
    if (method === 'GET' && url.pathname === '/v1/me/audit-logs') {
      auditRequests += 1
      if (auditMode === 'error') {
        await fulfillJSON(route, {
          type: 'https://quickdesk.io/problems/forbidden',
          title: 'Forbidden',
          status: 403,
          detail: 'Invalid or missing API key / origin not allowed',
        }, 403)
        return
      }
      const items = auditMode === 'empty' ? [] : auditItems()
      await fulfillJSON(route, { items, next_cursor: null })
      return
    }
    await fulfillJSON(route, { items: [] })
  })

  await page.goto(`${baseUrl}/#/sessions`, { waitUntil: 'networkidle' })
  // The page has its own loading state, and the audit toggle sits in the toolbar
  // that only appears once it is done. Waiting for the toolbar rather than a
  // fixed delay keeps the test from clicking a half-rendered page.
  await page.locator('.toolbar').first().waitFor({ state: 'visible', timeout: 10000 })
  return { page, context, pageErrors, requests: () => auditRequests }
}

// The entry point exists and opens the panel. A button that renders but does
// nothing would pass a "button exists" assertion, so this also waits for a row.
test('the sessions page exposes a transfer audit panel that lists rows', async () => {
  const { page, context, pageErrors } = await openSessionsPage({ auditMode: 'rows' })
  try {
    const toggle = page.getByRole('button', { name: 'Transfer audit' })
    assert.ok(await toggle.count(), 'the audit entry point is not on the page')
    await toggle.first().click()

    const panel = page.locator('.audit-panel')
    await panel.waitFor({ state: 'visible', timeout: 5000 })

    // Both kinds of data-channel payload are listed, and the row says what moved
    // and how much - never what was in it.
    const rows = panel.locator('.audit-row')
    await rows.first().waitFor({ state: 'attached', timeout: 5000 })
    // Both rows, not just the first: a page that rendered one would pass the
    // wait above and misreport the file transfer.
    await rows.nth(1).waitFor({ state: 'attached', timeout: 5000 })
    assert.equal(await rows.count(), 2, 'both reported transfers should be listed')

    const text = await panel.innerText()
    assert.ok(text.includes('Clipboard'), `the clipboard row is not described: ${text}`)
    assert.ok(text.includes('1.2 KB'), `the byte count is not readable: ${text}`)
    assert.ok(text.includes('report.pdf'), `the file name is not shown: ${text}`)
    assert.deepEqual(pageErrors, [])
  } finally {
    await context.close()
  }
})

// The state that matters: a failed read must not look like an empty one. If the
// panel rendered an empty list on failure, "the server refused" would read as
// "nothing was transferred", which is the opposite of what happened.
test('a failed audit read is shown as a failure, not as an empty list', async () => {
  const { page, context, pageErrors } = await openSessionsPage({ auditMode: 'error' })
  try {
    await page.getByRole('button', { name: 'Transfer audit' }).first().click()
    const panel = page.locator('.audit-panel')
    await panel.waitFor({ state: 'visible', timeout: 5000 })

    // Wait for the read to settle rather than reading the panel the instant it
    // opens: the loading line is what a panel shows before the request lands, and
    // asserting on it would be asserting on a race.
    await panel.locator('.audit-row, .audit-error, .hint')
      .filter({ hasNotText: 'Loading' })
      .first()
      .waitFor({ state: 'attached', timeout: 5000 })

    const text = await panel.innerText()
    assert.ok(text.includes('Could not read the audit trail'),
      `a failed read did not say so: ${text}`)
    assert.equal(await panel.locator('.audit-row').count(), 0,
      'a failed read rendered rows')
    // And it did not fall through to the empty state, which is a different claim.
    assert.ok(!text.includes('No file or clipboard transfers'),
      `a failed read rendered as "nothing transferred": ${text}`)
    assert.deepEqual(pageErrors, [])
  } finally {
    await context.close()
  }
})

// An empty successful read is the one case where an empty list is the truth, and
// it has to look different from a failure.
test('an empty audit read is shown as empty, not as a failure', async () => {
  const { page, context, pageErrors } = await openSessionsPage({ auditMode: 'empty' })
  try {
    await page.getByRole('button', { name: 'Transfer audit' }).first().click()
    const panel = page.locator('.audit-panel')
    await panel.waitFor({ state: 'visible', timeout: 5000 })
    await panel.locator('.audit-row, .audit-error, .hint')
      .filter({ hasNotText: 'Loading' })
      .first()
      .waitFor({ state: 'attached', timeout: 5000 })

    const text = await panel.innerText()
    assert.ok(text.includes('No file or clipboard transfers'),
      `an empty read did not say so: ${text}`)
    assert.ok(!text.includes('Could not read the audit trail'),
      `an empty read rendered as a failure: ${text}`)
    assert.deepEqual(pageErrors, [])
  } finally {
    await context.close()
  }
})

// The read is lazy: opening the page does not request the trail, and closing the
// panel does not either. A page that fetched on mount would pay for a request
// nobody asked for.
test('the audit is read on demand, not on page load', async () => {
  const { page, context, requests } = await openSessionsPage({ auditMode: 'rows' })
  try {
    await page.waitForTimeout(500)
    assert.equal(requests(), 0, 'the audit was read before anybody asked for it')

    await page.getByRole('button', { name: 'Transfer audit' }).first().click()
    const panel = page.locator('.audit-panel')
    await panel.waitFor({ state: 'visible', timeout: 5000 })
    await panel.locator('.audit-row').first().waitFor({ timeout: 5000 })
    assert.equal(requests(), 1, 'opening the panel did not read exactly once')

    // Closing and reopening does not re-read: the rows are already loaded.
    await page.getByRole('button', { name: 'Transfer audit' }).first().click()
    await panel.waitFor({ state: 'hidden', timeout: 5000 })
    await page.getByRole('button', { name: 'Transfer audit' }).first().click()
    await panel.waitFor({ state: 'visible', timeout: 5000 })
    assert.equal(requests(), 1, 'reopening a loaded panel re-read it')
  } finally {
    await context.close()
  }
})
