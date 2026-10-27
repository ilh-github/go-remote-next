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

test('devices page drives connection, favorites, logs, and private groups', async () => {
  const state = {
    devices: [
      {
        device_id: 'device-a',
        remark: 'Desk Alpha',
        online: true,
        logged_in: true,
        access_code: 'code-a',
      },
      {
        device_id: 'device-b',
        remark: 'Tablet',
        online: true,
        logged_in: true,
        access_code: '',
      },
      {
        device_id: 'device-c',
        remark: 'Idle Station',
        online: true,
        logged_in: false,
        access_code: 'code-c',
      },
      {
        device_id: 'device-d',
        remark: 'Offline Station',
        online: false,
        logged_in: false,
        access_code: 'code-d',
      },
    ],
    favorites: [
      {
        device_id: 'device-a',
        device_name: 'Desk Alpha',
        access_password: 'code-a',
      },
    ],
    logs: [
      {
        id: 1,
        device_id: 'device-a',
        device_name: 'Desk Alpha',
        status: 'success',
        duration: 125,
        created_at: '2026-10-18T10:00:00Z',
      },
      {
        id: 2,
        device_id: 'device-c',
        device_name: 'Old Laptop',
        status: 'failed',
        error_msg: 'access code rejected',
        duration: 0,
        created_at: '2026-10-18T09:30:00Z',
      },
      {
        id: 3,
        // Not in the bound device list at all, so there is no code to offer and
        // no way back to it: this row must render no connect button.
        device_id: 'device-gone',
        device_name: 'Retired Box',
        status: 'failed',
        error_msg: 'device not bound',
        duration: 0,
        created_at: '2026-10-18T08:00:00Z',
      },
    ],
    groups: [
      {
        id: 1,
        user_id: 7,
        name: 'Lab',
        description: 'Managed lab machines',
        color: '#10b981',
        device_ids: ['device-a'],
      },
      {
        id: 2,
        user_id: 7,
        name: 'Empty',
        description: '',
        color: '#64748b',
        device_ids: [],
      },
    ],
    nextGroupId: 3,
    verifyCounts: new Map(),
    deviceFetches: 0,
    mutations: [],
  }

  const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await context.addInitScript(() => {
    localStorage.setItem('quickdesk_lang', 'en-US')
    localStorage.setItem('quickdesk_user_access_token', 'devices-e2e-token')
    window.__openedRemote = []
    window.open = (url, target) => {
      window.__openedRemote.push({ url: String(url), target })
      return null
    }
  })
  const page = await context.newPage()
  const pageErrors = []
  page.on('pageerror', (error) => pageErrors.push(error.message))

  await page.route('**/v1/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const method = request.method()
    const json = () => request.postDataJSON()

    if (method === 'GET' && url.pathname === '/v1/me/devices') {
      state.deviceFetches += 1
      await fulfillJSON(route, { items: state.devices })
      return
    }
    if (method === 'GET' && url.pathname === '/v1/me/favorites') {
      await fulfillJSON(route, { items: state.favorites })
      return
    }
    if (method === 'GET' && url.pathname === '/v1/me/connections') {
      await fulfillJSON(route, { items: state.logs })
      return
    }
    if (method === 'GET' && url.pathname === '/v1/me/device-groups') {
      await fulfillJSON(route, { items: state.groups })
      return
    }

    // The history can lose rows as well as gain them: one entry by id, or the
    // whole list. Both are exercised by the delete/clear assertions below.
    const connectionDelete = url.pathname.match(/^\/v1\/me\/connections\/(\d+)$/)
    if (method === 'DELETE' && connectionDelete) {
      const id = Number(connectionDelete[1])
      const before = state.logs.length
      state.logs = state.logs.filter((log) => log.id !== id)
      state.mutations.push({ method, path: url.pathname })
      await fulfillJSON(route, { deleted: before !== state.logs.length })
      return
    }
    if (method === 'DELETE' && url.pathname === '/v1/me/connections') {
      const removed = state.logs.length
      state.mutations.push({ method, path: url.pathname })
      state.logs = []
      await fulfillJSON(route, { removed })
      return
    }

    const verify = url.pathname.match(/^\/v1\/devices\/([^/]+)\/access-code:verify$/)
    if (method === 'POST' && verify) {
      const deviceId = decodeURIComponent(verify[1])
      const count = (state.verifyCounts.get(deviceId) || 0) + 1
      state.verifyCounts.set(deviceId, count)
      await fulfillJSON(route, {
        signal_token: `signal-${deviceId}-${count}`,
        expires_at: '2026-10-18T12:00:00Z',
      })
      return
    }

    const devicePatch = url.pathname.match(/^\/v1\/me\/devices\/([^/]+)$/)
    if (method === 'PATCH' && devicePatch) {
      const deviceId = decodeURIComponent(devicePatch[1])
      const body = json()
      state.mutations.push({ method, path: url.pathname, body })
      const device = state.devices.find((item) => item.device_id === deviceId)
      if (device) device.remark = body.remark
      await fulfillJSON(route, device)
      return
    }
    if (method === 'DELETE' && devicePatch) {
      const deviceId = decodeURIComponent(devicePatch[1])
      state.mutations.push({ method, path: url.pathname })
      state.devices = state.devices.filter((item) => item.device_id !== deviceId)
      state.groups = state.groups.map((group) => ({
        ...group,
        device_ids: group.device_ids.filter((id) => id !== deviceId),
      }))
      await route.fulfill({ status: 204 })
      return
    }

    const favoriteDelete = url.pathname.match(/^\/v1\/me\/favorites\/([^/]+)$/)
    if (method === 'DELETE' && favoriteDelete) {
      const deviceId = decodeURIComponent(favoriteDelete[1])
      state.mutations.push({ method, path: url.pathname })
      state.favorites = state.favorites.filter((item) => item.device_id !== deviceId)
      await route.fulfill({ status: 204 })
      return
    }

    if (method === 'POST' && url.pathname === '/v1/me/device-groups') {
      const body = json()
      state.mutations.push({ method, path: url.pathname, body })
      const group = {
        id: state.nextGroupId++,
        user_id: 7,
        ...body,
        device_ids: [],
      }
      state.groups.push(group)
      await fulfillJSON(route, group)
      return
    }

    const groupMembers = url.pathname.match(/^\/v1\/me\/device-groups\/(\d+)\/devices$/)
    if (method === 'PUT' && groupMembers) {
      const groupId = Number(groupMembers[1])
      const body = json()
      state.mutations.push({ method, path: url.pathname, body })
      const group = state.groups.find((item) => item.id === groupId)
      if (group) group.device_ids = body.device_ids.slice()
      await fulfillJSON(route, group)
      return
    }

    const groupDelete = url.pathname.match(/^\/v1\/me\/device-groups\/(\d+)$/)
    if (method === 'DELETE' && groupDelete) {
      const groupId = Number(groupDelete[1])
      state.mutations.push({ method, path: url.pathname })
      state.groups = state.groups.filter((item) => item.id !== groupId)
      await route.fulfill({ status: 204 })
      return
    }

    await route.fulfill({
      status: 404,
      contentType: 'application/problem+json',
      body: JSON.stringify({ detail: `Unhandled E2E request: ${method} ${url.pathname}` }),
    })
  })

  try {
    await page.goto(`${baseUrl}/test/fixtures/devices.html`, { waitUntil: 'domcontentloaded' })
    await page.waitForFunction(() => window.__devicesMounted === true)
    await page.waitForFunction(
      () => document.querySelectorAll('.group-filter-chip').length === 3
        && document.querySelectorAll('.card')[0]?.textContent.includes('4 device(s)'),
    )

    assert.equal(await myDevicesCard(page).locator('.device-item').count(), 4)
    assert.equal(await favoritesCard(page).locator('.device-item').count(), 1)
    assert.equal(await logsCard(page).locator('.device-item').count(), 3)
    assert.equal(await groupCount(page, 'All'), '4')

    // The log rows are the only place an owner learns why a connection did not
    // happen, so the device name and the recorded failure reason both have to
    // reach the page. Counting rows alone would pass while both were dropped.
    const logText = await logsCard(page).textContent()
    assert.match(logText, /Desk Alpha/, 'a log row shows the device name the server sent')
    assert.match(logText, /2m5s/, 'a log row renders its duration')
    assert.match(logText, /access code rejected/, 'a failed log row shows the reason the server recorded')

    // QD-206 promises quick connect as well as the history. The history row
    // carries no access code by design (the API does not store one), so the
    // reconnect joins against the device list on device_id.
    await logsCard(page).getByRole('button', { name: 'Connect', exact: true }).first().click()
    await page.waitForFunction(() => window.__openedRemote.length === 1)
    const reconnect = await page.evaluate(() => {
      const parsed = new URL(window.__openedRemote[0].url, location.href)
      return {
        device: parsed.searchParams.get('device'),
        handoff: JSON.parse(sessionStorage.getItem(`quickdesk_remote_handoff__${parsed.searchParams.get('st')}`)),
      }
    })
    assert.equal(reconnect.device, 'device-a')
    assert.equal(reconnect.handoff.access_code, 'code-a')
    assert.equal(reconnect.handoff.device_id, 'device-a')

    // Two of the three rows are bound and carry a code; the third names a device
    // that is not bound at all, so there is no code to offer and it renders no
    // button rather than one that could only produce a "no access code" toast.
    assert.equal(
      await logsCard(page).getByRole('button', { name: 'Connect', exact: true }).count(),
      2,
    )

    // The history grows a row on every attempt, so it has to be able to lose one.
    // Deleting removes that row and only that row.
    await logsCard(page).getByTitle('Delete this entry').first().click()
    await page.waitForFunction(() => document.querySelectorAll('.card')[2].querySelectorAll('.device-item').length === 2)
    assert.equal(state.logs.length, 2)
    assert.equal(
      state.mutations.some(
        (item) => item.method === 'DELETE' && /^\/v1\/me\/connections\/\d+$/.test(item.path),
      ),
      true,
      'the delete went to the single-entry endpoint, not the collection',
    )

    // Clearing asks first -- it is the whole history and cannot be undone -- and
    // only proceeds once confirmed.
    page.once('dialog', (dialog) => dialog.accept())
    await page.getByRole('button', { name: 'Clear history', exact: true }).click()
    await page.waitForFunction(() => document.querySelectorAll('.card')[2].querySelectorAll('.device-item').length === 0)
    assert.equal(state.logs.length, 0)
    assert.equal(
      state.mutations.some(
        (item) => item.method === 'DELETE' && item.path === '/v1/me/connections',
      ),
      true,
    )
    await page.getByText('No connection logs', { exact: true }).waitFor()
    assert.equal(await groupCount(page, 'Lab'), '1')
    assert.equal(await groupCount(page, 'Empty'), '0')

    await groupChip(page, 'Lab').click()
    await waitForLocatorCount(myDevicesCard(page).locator('.device-item'), 1)
    assert.equal(await myDevicesCard(page).locator('.device-item').count(), 1)
    assert.equal(await page.getByText('Managed lab machines', { exact: true }).count(), 1)

    await groupChip(page, 'Empty').click()
    await waitForLocatorCount(myDevicesCard(page).locator('.device-item'), 0)
    assert.equal(await page.getByText('This group has no devices yet', { exact: true }).count(), 1)
    await page.getByRole('button', { name: 'Manage members', exact: true }).click()
    await page.locator('.group-dialog').waitFor()
    assert.equal(await page.locator('.group-list-row', { hasText: 'Empty' }).count(), 1)
    await page.getByTitle('Close').click()

    await page.getByRole('button', { name: 'Device groups', exact: true }).click()
    await page.getByRole('button', { name: 'New group', exact: true }).click()
    await page.locator('.group-fields input.form-input').first().fill('Office')
    await page.locator('.group-description-input').fill('Daily office machines')
    await page.locator('.color-swatch[title="#3b82f6"]').click()
    await page.locator('.group-editor .form-actions').first()
      .getByRole('button', { name: 'New group', exact: true })
      .click()
    await page.locator('.group-list-row', { hasText: 'Office' }).waitFor()
    await page.getByRole('button', { name: 'Select all', exact: true }).click()
    await page.getByRole('button', { name: 'Save members (4)', exact: true }).click()
    await waitFor(() => state.mutations.some(
      (item) => item.method === 'PUT' && item.body?.device_ids?.length === 4,
    ))
    await page.getByTitle('Close').click()
    assert.equal(await groupCount(page, 'Office'), '4')

    await groupChip(page, 'All').click()
    const alpha = deviceRow(page, 'Desk Alpha')
    await alpha.getByTitle('Set Remark').click()
    await page.locator('.dialog .form-input').fill('Desk Alpha 2')
    await page.getByRole('button', { name: 'Save', exact: true }).click()
    await page.getByText('Desk Alpha 2', { exact: true }).waitFor()
    assert.equal(state.devices.find((item) => item.device_id === 'device-a').remark, 'Desk Alpha 2')

    await deviceRow(page, 'Tablet').getByRole('button', { name: 'Connect', exact: true }).click()
    await page.waitForFunction(
      () => window.__deviceToasts.some((toast) => toast.message === 'No access code'),
    )

    await deviceRow(page, 'Desk Alpha 2').getByRole('button', { name: 'Connect', exact: true }).click()
    await page.waitForFunction(() => window.__openedRemote.length === 1)
    const opened = await page.evaluate(() => {
      const item = window.__openedRemote[0]
      const parsed = new URL(item.url, location.href)
      const token = parsed.searchParams.get('st')
      const handoff = JSON.parse(sessionStorage.getItem(`quickdesk_remote_handoff__${token}`))
      return {
        device: parsed.searchParams.get('device'),
        token,
        target: item.target,
        handoff,
      }
    })
    assert.equal(opened.device, 'device-a')
    assert.equal(opened.token, 'signal-device-a-1')
    assert.equal(opened.target, 'quickdesk_device-a')
    assert.equal(opened.handoff.access_code, 'code-a')
    assert.equal(opened.handoff.device_id, 'device-a')
    assert.equal(typeof opened.handoff.created_at, 'number')

    await favoritesCard(page).getByTitle('Remove Favorite').click()
    await page.getByText('No favorite devices', { exact: true }).waitFor()
    assert.equal(state.favorites.length, 0)

    page.once('dialog', (dialog) => dialog.accept())
    await deviceRow(page, 'Idle Station').getByTitle('Unbind').click()
    await page.waitForFunction(
      () => !document.body.textContent.includes('Idle Station'),
    )
    assert.equal(state.devices.length, 3)
    assert.equal(state.mutations.some(
      (item) => item.method === 'DELETE' && item.path === '/v1/me/devices/device-c',
    ), true)

    const fetchesBeforeRefresh = state.deviceFetches
    await page.getByRole('button', { name: /Refresh/ }).click()
    await waitFor(() => state.deviceFetches === fetchesBeforeRefresh + 1)
    assert.equal(await myDevicesCard(page).locator('.device-item').count(), 3)
    assert.deepEqual(pageErrors, [])
  } finally {
    await context.close()
  }
})

// QD-105: the account's device limit is now enforced server-side. The host binds
// itself on startup, so a refusal the owner only discovers by noticing a missing
// machine is the failure this warning exists to prevent. This drives the real
// DevicesPage against the real quota module, with the profile the server sends.
test('devices page shows the device limit and warns when it is reached', async () => {
  const state = {
    devices: [
      { device_id: 'device-a', remark: 'Desk Alpha', online: true, logged_in: true },
      { device_id: 'device-b', remark: 'Tablet', online: true, logged_in: true },
    ],
  }

  const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await context.addInitScript(() => {
    localStorage.setItem('quickdesk_lang', 'en-US')
    localStorage.setItem('quickdesk_user_access_token', 'devices-quota-e2e-token')
    // A quota of 2 with two bound machines: the warning has to appear without
    // anybody clicking anything, because that is the state a refused host lands
    // the owner in.
    localStorage.setItem('quickdesk_user_info', JSON.stringify({
      id: 7,
      username: 'quota-owner',
      device_count: 2,
    }))
  })
  const page = await context.newPage()
  const pageErrors = []
  page.on('pageerror', (error) => pageErrors.push(error.message))

  await page.route('**/v1/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    if (request.method() === 'GET' && url.pathname === '/v1/me/devices') {
      await fulfillJSON(route, { items: state.devices })
      return
    }
    await fulfillJSON(route, { items: [] })
  })

  try {
    await page.goto(`${baseUrl}/test/fixtures/devices.html`, { waitUntil: 'domcontentloaded' })
    await page.waitForFunction(() => window.__devicesMounted === true)
    await page.waitForFunction(
      () => document.querySelector('.device-quota-warning') !== null,
    )

    const heading = await myDevicesCard(page).locator('.card-heading').textContent()
    assert.ok(
      heading.includes('2 of 2 devices'),
      `heading should show the limit, got: ${heading}`,
    )
    assert.ok(
      heading.includes('Device limit reached'),
      `heading should warn that the limit is reached, got: ${heading}`,
    )
    assert.deepEqual(pageErrors, [])
  } finally {
    await context.close()
  }
})

// The same page with no quota configured must show neither the count nor the
// warning: 0 is the default every account is created with and means "no limit",
// so rendering it would tell a user with working machines that they are blocked.
test('devices page hides the limit when the account has none', async () => {
  const state = {
    devices: [
      { device_id: 'device-a', remark: 'Desk Alpha', online: true, logged_in: true },
    ],
  }

  const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await context.addInitScript(() => {
    localStorage.setItem('quickdesk_lang', 'en-US')
    localStorage.setItem('quickdesk_user_access_token', 'devices-noquota-e2e-token')
    localStorage.setItem('quickdesk_user_info', JSON.stringify({
      id: 8,
      username: 'unlimited-owner',
      device_count: 0,
    }))
  })
  const page = await context.newPage()
  const pageErrors = []
  page.on('pageerror', (error) => pageErrors.push(error.message))

  await page.route('**/v1/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    if (request.method() === 'GET' && url.pathname === '/v1/me/devices') {
      await fulfillJSON(route, { items: state.devices })
      return
    }
    await fulfillJSON(route, { items: [] })
  })

  try {
    await page.goto(`${baseUrl}/test/fixtures/devices.html`, { waitUntil: 'domcontentloaded' })
    await page.waitForFunction(() => window.__devicesMounted === true)
    await page.waitForFunction(
      () => document.querySelectorAll('.device-item').length === 1,
    )

    const heading = await myDevicesCard(page).locator('.card-heading').textContent()
    assert.ok(
      !heading.includes('of 0 devices'),
      `an unlimited account must not be shown a limit, got: ${heading}`,
    )
    assert.equal(
      await page.locator('.device-quota-warning').count(),
      0,
      'an unlimited account must not be warned about a limit',
    )
    assert.deepEqual(pageErrors, [])
  } finally {
    await context.close()
  }
})

function myDevicesCard(page) {
  return page.locator('.card').nth(0)
}

function favoritesCard(page) {
  return page.locator('.card').nth(1)
}

function logsCard(page) {
  return page.locator('.card').nth(2)
}

function groupChip(page, name) {
  return page.locator('.group-filter-chip', {
    hasText: name,
  })
}

async function groupCount(page, name) {
  return (await groupChip(page, name).locator('.group-filter-count').textContent()).trim()
}

function deviceRow(page, name) {
  return myDevicesCard(page).locator('.device-item', {
    has: page.locator('.device-name', { hasText: new RegExp(`^${name}$`) }),
  })
}

async function fulfillJSON(route, body) {
  await route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify(body),
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

async function waitForLocatorCount(locator, expected, timeout = 5000) {
  const deadline = Date.now() + timeout
  while (Date.now() < deadline) {
    if (await locator.count() === expected) return
    await new Promise((resolve) => setTimeout(resolve, 25))
  }
  throw new Error(`Timed out waiting for locator count ${expected}`)
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
