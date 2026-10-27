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

// QD-904: the approvals page had model-level tests but no browser E2E, and it is
// the page where a wrong click has a consequence - approve admits a session onto
// a machine, deny puts the source into a cooldown. What is asserted here is the
// part a unit test cannot see: that the dialog the operator fills in is the
// dialog for the approval they clicked, and that a decision the server refuses
// reports the server's own status rather than a generic failure.
test('approvals page drives approve, deny, and an already-decided conflict', async () => {
  const state = {
    approvals: [
      {
        id: 11,
        device_id: '700000001',
        client_id: 'grant-event-probe',
        client_ip: '203.0.113.9',
        source: 'access_grant',
        status: 'pending',
        reason: 'guest access',
        created_at: '2026-10-27T10:00:00Z',
      },
      {
        id: 12,
        device_id: '700000001',
        client_id: 'second-requester',
        client_ip: '198.51.100.4',
        source: 'pair_code',
        status: 'pending',
        reason: '',
        created_at: '2026-10-27T10:05:00Z',
      },
    ],
    // The server's answer to the second approve attempt: someone else got there
    // first, and it says which way.
    conflictStatus: 'denied',
    decisions: [],
  }

  const context = await browser.newContext({ viewport: { width: 1280, height: 1000 } })
  await context.addInitScript(() => {
    localStorage.setItem('quickdesk_user_access_token', 'approvals-e2e-token')
    window.__approvalToasts = []
    window.__showLoginCalls = 0
  })
  const page = await context.newPage()
  const pageErrors = []
  page.on('pageerror', (error) => pageErrors.push(error.message))

  // The decisions live in Node, but waitForFunction runs in the page. Mirror them
  // so the waits can observe them.
  const mirrorDecisions = () => page.evaluate((d) => {
    window.__approvalDecisions = d
  }, state.decisions)

  await page.route('**/v1/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const method = request.method()
    const json = () => request.postDataJSON()

    if (method === 'GET' && url.pathname === '/v1/me/devices') {
      await fulfillJSON(route, {
        items: [{ device_id: '700000001', remark: 'Lab Bench', online: true }],
      })
      return
    }
    if (method === 'GET' && /\/v1\/me\/devices\/[^/]+\/access-approvals$/.test(url.pathname)) {
      await fulfillJSON(route, { items: state.approvals })
      return
    }
    if (method === 'GET' && /\/v1\/me\/devices\/[^/]+\/access-approvals:summary$/.test(url.pathname)) {
      await fulfillJSON(route, {
        since: '2026-10-27T09:00:00Z',
        total: state.approvals.length,
        by_status: { pending: state.approvals.length, approved: 0, denied: 0, expired: 0, consumed: 0 },
        denied_sources: [],
        denial_cooldown_seconds: 300,
        denial_cooldown_multiplier: 3,
        denial_cooldown_max_seconds: 3600,
      })
      return
    }
    if (method === 'POST' && /\/access-approvals\/\d+\/approve$/.test(url.pathname)) {
      const id = Number(url.pathname.match(/\/access-approvals\/(\d+)\/approve$/)[1])
      state.decisions.push({ id, action: 'approve', reason: json().reason || '' })
      mirrorDecisions()
      // The second approve on the same row is refused the way the server does it.
      const already = state.decisions.filter((d) => d.id === id).length > 1
      if (already) {
        await route.fulfill({
          status: 409,
          contentType: 'application/json',
          body: JSON.stringify({ message: 'This request was already decided', status: state.conflictStatus }),
        })
        return
      }
      await fulfillJSON(route, { id, status: 'approved' })
      return
    }
    if (method === 'POST' && /\/access-approvals\/\d+\/deny$/.test(url.pathname)) {
      const id = Number(url.pathname.match(/\/access-approvals\/(\d+)\/deny$/)[1])
      state.decisions.push({ id, action: 'deny', reason: json().reason || '' })
      mirrorDecisions()
      await fulfillJSON(route, { id, status: 'denied' })
      return
    }
    await fulfillJSON(route, {})
  })

  await page.goto(`${baseUrl}/test/fixtures/access-approvals.html`, { waitUntil: 'domcontentloaded' })
  await page.waitForFunction(() => window.__approvalsMounted === true)
  await page.waitForSelector('.approval-row')

  // Both pending rows are listed, each with its own approve and deny.
  assert.equal(await page.locator('.approval-row').count(), 2)

  // --- Approve the first row ---
  // The click targets the button inside the first row, not the first button on
  // the page: the two rows are identical in shape, and a page-level selector
  // would pass while deciding the wrong approval.
  await page.locator('.approval-row').nth(0).locator('button.btn-primary').click()
  await page.waitForSelector('.dialog')

  // The dialog must name the approval that was clicked.
  const dialogText = await page.locator('.dialog').innerText()
  assert.match(dialogText, /grant-event-probe/, 'the dialog is not for the approval that was clicked')
  assert.match(dialogText, /203\.0\.113\.9/, 'the dialog does not show the requester address')

  // Approving is the default intent, so the confirm button must be the primary
  // one; deny is the dangerous one and must be the danger-styled one.
  const confirm = page.locator('.form-actions button').first()
  assert.match(await confirm.getAttribute('class'), /btn-primary/)

  await page.locator('.reason-input').fill('approved for the maintenance window')
  await confirm.click()

  await page.waitForFunction(() => (window.__approvalDecisions || []).length > 0)
  assert.equal(state.decisions.length, 1)
  assert.equal(state.decisions[0].id, 11)
  assert.equal(state.decisions[0].action, 'approve')
  assert.equal(state.decisions[0].reason, 'approved for the maintenance window')

  // --- Deny the second row ---
  await page.locator('.approval-row').nth(1).locator('button.btn-danger').click()
  await page.waitForSelector('.dialog')
  const denyText = await page.locator('.dialog').innerText()
  assert.match(denyText, /second-requester/, 'the deny dialog is not for the second approval')

  // Denying carries a cooldown consequence, so the warning must name it.
  assert.match(denyText, /300/, 'the deny dialog does not warn about the cooldown')

  await page.locator('.form-actions button').first().click()
  await page.waitForFunction(() => (window.__approvalDecisions || []).length === 2)
  assert.equal(state.decisions[1].id, 12)
  assert.equal(state.decisions[1].action, 'deny')

  // --- An already-decided row reports the server's status, not a generic failure ---
  // Re-approving row 11 gets a 409 naming the status someone else set. The toast
  // must carry that status; "decision failed" would send the operator looking for
  // a network problem.
  await page.locator('.approval-row').nth(0).locator('button.btn-primary').click()
  await page.waitForSelector('.dialog')
  await page.locator('.form-actions button').first().click()

  await page.waitForFunction(() => window.__approvalToasts.length >= 3)
  const toasts = await page.evaluate(() => window.__approvalToasts)
  const conflictToast = toasts.find((entry) => entry.type === 'error')
  assert.ok(conflictToast, 'a refused decision produced no toast')
  assert.match(conflictToast.message, /denied/i,
    `the conflict toast does not name the server's status: ${conflictToast.message}`)

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
