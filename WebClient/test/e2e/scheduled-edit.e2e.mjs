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

// QD-607: a task with the wrong retry budget or the wrong timezone could only be
// fixed by deleting it and creating a new one, which threw away the outcome the
// owner needed in order to see it was failing. The edit now exists on the server;
// this drives the real page to prove the page uses it.
//
// It has to be the real page, because the failure this guards against is a dialog
// that opens empty - which a unit test on the payload builder passes straight
// through. The assertions below are on what the dialog shows and what it sends.

async function openPageWith(pageInit) {
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await context.addInitScript((task) => {
    window.__storedTask = task
  }, JSON.parse(JSON.stringify(storedTask)))
  await context.addInitScript(pageInit)
  const page = await context.newPage()
  const pageErrors = []
  page.on('pageerror', (error) => pageErrors.push(error.message))
  await page.goto(`${baseUrl}/test/fixtures/scheduled.html`, { waitUntil: 'domcontentloaded' })
  await page.waitForFunction(() => window.__scheduledMounted === true)
  return { context, page, pageErrors }
}

const storedTask = {
  id: 4242,
  kind: 'revoke_all_remote_sessions',
  enabled: true,
  next_run_at: '2026-10-27T09:05:00Z',
  repeat_seconds: 0,
  schedule: 'monthly@31@03:00',
  timezone: 'Asia/Shanghai',
  skip_holidays: true,
  max_attempts: 4,
  attempts: 0,
  last_run_at: null,
  last_status: '',
  last_error: '',
}

test('editing a task opens on the stored schedule and PATCHes the row', async () => {
  const { context, page, pageErrors } = await openPageWith(() => {
    window.__scheduledWrites = []
    const originalFetch = window.fetch
    window.fetch = (input, init) => {
      const url = typeof input === 'string' ? input : input.url
      const method = init?.method || 'GET'
      if (url.includes('/v1/me/scheduled-tasks') && method !== 'GET') {
        window.__scheduledWrites.push({ url: String(url), method, body: init?.body || '' })
      }
      if (url.includes('/v1/me/scheduled-tasks')) {
        return Promise.resolve(new Response(JSON.stringify({
          items: [JSON.parse(JSON.stringify(window.__storedTask))],
          supported_kinds: ['sign_out_all_sessions', 'revoke_all_remote_sessions', 'connect_prompt'],
          supported_schedules: ['daily@HH:MM'],
        }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
      }
      return originalFetch(input, init)
    }
  })

  // The row is on screen before anything is clicked, so the edit button is real.
  await page.waitForFunction(() => document.body.textContent.includes('Revoke all remote sessions'), null,
    { timeout: 15000 })
  await page.getByRole('button', { name: 'Edit' }).first().click()

  // The dialog has to open on the stored row, not on defaults. A calendar row
  // whose cadence reads "daily" is the empty-dialog bug wearing a disguise.
  await page.waitForFunction(() => document.querySelector('.task-dialog') !== null, null,
    { timeout: 10000 })
  const selected = await page.evaluate(() => {
    // The dialog holds several selects - kind, cadence, weekday - so the cadence
    // is picked by its option set rather than by position, which is what made
    // the first version of this assertion read the kind dropdown instead.
    const cadence = Array.from(document.querySelectorAll('.task-dialog select'))
      .find((node) => Array.from(node.options).some((option) => option.value === 'monthly'))
    const monthDay = Array.from(document.querySelectorAll('.task-dialog input[type="number"]'))
      .find((node) => Number(node.max) === 31)
    return {
      cadence: cadence?.value,
      monthDay: monthDay?.value,
      time: document.querySelector('.task-dialog input[type="time"]')?.value,
      timezone: Array.from(document.querySelectorAll('.task-dialog input[type="text"]'))
        .map((node) => node.value)
        .find((value) => value === 'Asia/Shanghai'),
    }
  })
  assert.equal(selected.cadence, 'monthly', `dialog cadence = ${selected.cadence}, want monthly`)
  assert.equal(selected.monthDay, '31', `dialog month day = ${selected.monthDay}, want 31`)
  assert.equal(selected.time, '03:00', `dialog time = ${selected.time}, want 03:00`)
  assert.equal(selected.timezone, 'Asia/Shanghai', 'the dialog dropped the stored timezone')

  // Save must PATCH the existing row rather than POST a new one: a POST here
  // would recreate the task and lose the id along with its outcome history.
  await page.getByRole('button', { name: 'Save changes' }).click()
  await page.waitForFunction(() => window.__scheduledWrites.length >= 1, null, { timeout: 10000 })

  const writes = await page.evaluate(() => window.__scheduledWrites)
  assert.equal(writes.length, 1, `expected exactly one write, got ${JSON.stringify(writes)}`)
  assert.equal(writes[0].method, 'PATCH', `write method = ${writes[0].method}, want PATCH`)
  assert.ok(
    writes[0].url.endsWith('/v1/me/scheduled-tasks/4242'),
    `write url = ${writes[0].url}, want the stored row`,
  )
  const body = JSON.parse(writes[0].body)
  assert.equal(body.schedule, 'monthly@31@03:00', `sent schedule = ${body.schedule}`)
  assert.equal(body.max_attempts, 4, `sent max_attempts = ${body.max_attempts}`)
  assert.ok(!('run_at' in body), 'a calendar row must not carry run_at')
  assert.ok(!('repeat_seconds' in body), 'a calendar row must not carry repeat_seconds')

  assert.deepEqual(pageErrors, [])
  await context.close()
})

// The create path must keep working, and must not be reachable through the edit
// button: the same dialog serves both, so a regression that leaves the editing id
// set would turn a "new task" into a PATCH of whatever row was edited last.
test('creating a task still POSTs a new row', async () => {
  const { context, page, pageErrors } = await openPageWith(() => {
    window.__scheduledWrites = []
    const originalFetch = window.fetch
    window.fetch = (input, init) => {
      const url = typeof input === 'string' ? input : input.url
      const method = init?.method || 'GET'
      if (url.includes('/v1/me/scheduled-tasks') && method !== 'GET') {
        window.__scheduledWrites.push({ url: String(url), method, body: init?.body || '' })
      }
      if (url.includes('/v1/me/scheduled-tasks')) {
        return Promise.resolve(new Response(JSON.stringify({
          items: [JSON.parse(JSON.stringify(window.__storedTask))],
          supported_kinds: ['sign_out_all_sessions', 'revoke_all_remote_sessions', 'connect_prompt'],
          supported_schedules: ['daily@HH:MM'],
        }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
      }
      return originalFetch(input, init)
    }
  })

  await page.waitForFunction(() => document.body.textContent.includes('Revoke all remote sessions'), null,
    { timeout: 15000 })
  // Edit first, then cancel, then create: the editing id must not survive.
  await page.getByRole('button', { name: 'Edit' }).first().click()
  await page.waitForFunction(() => document.querySelector('.task-dialog') !== null, null,
    { timeout: 10000 })
  await page.getByRole('button', { name: 'Cancel' }).click()
  await page.waitForFunction(() => document.querySelector('.task-dialog') === null, null,
    { timeout: 10000 })

  await page.getByRole('button', { name: /New scheduled task/ }).click()
  await page.waitForFunction(() => document.querySelector('.task-dialog') !== null, null,
    { timeout: 10000 })
  await page.getByRole('button', { name: 'Create task' }).click()
  await page.waitForFunction(() => window.__scheduledWrites.length >= 1, null, { timeout: 10000 })

  const writes = await page.evaluate(() => window.__scheduledWrites)
  assert.equal(writes.length, 1, `expected exactly one write, got ${JSON.stringify(writes)}`)
  assert.equal(writes[0].method, 'POST', `write method = ${writes[0].method}, want POST`)
  assert.ok(
    writes[0].url.endsWith('/v1/me/scheduled-tasks'),
    `write url = ${writes[0].url}, want the collection`,
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
