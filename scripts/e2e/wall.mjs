// Screen wall: put one online device and one offline device on the wall, then
// prove the live tile really decodes frames while staying view-only.
//
// Asserts:
//   - the online tile reaches "live" through the tile's postMessage report
//   - the iframe's own <video> has real dimensions and an advancing clock
//   - the tile created no mouse/keyboard/touch handler (wall tiles are monitors)
//   - the offline device renders a placeholder instead of an iframe

import { chromium } from 'playwright'
import { execFileSync } from 'node:child_process'

const APP = 'http://127.0.0.1:5173'
const SERVER = 'ws://10.211.55.2:8000'
const TOKEN = process.env.QD_TOKEN
const ONLINE = process.env.QD_DEVICE
const OFFLINE = process.env.QD_OFFLINE_DEVICE
const OUT = process.env.QD_OUT || '/Users/timbl/PDShare/qd-e2e'
// When set, the harness also proves that arming a tile sends real keystrokes to
// the guest (the input target writes its text to a spool file we can read back).
const GUEST = process.env.QD_GUEST_UUID

function guestText() {
  if (!GUEST) return ''
  // The target spool is a plain file: reading it with `type` avoids the UI
  // Automation path, which can wedge when the guest's automation host is busy.
  return execFileSync('prlctl', [
    'exec', GUEST, '--current-user', 'cmd', '/c', 'type C:\\QuickDeskTest\\input-target.txt',
  ], { encoding: 'utf8', timeout: 20000 })
}

const browser = await chromium.launch({ channel: 'chrome', headless: true })
const context = await browser.newContext({ viewport: { width: 1600, height: 1000 } })
const page = await context.newPage()
page.on('pageerror', (e) => console.log('[pageerror]', e.message))

await page.goto(APP, { waitUntil: 'domcontentloaded' })
await page.evaluate(({ s, t }) => {
  localStorage.setItem('quickdesk_signaling_url', s)
  localStorage.setItem('quickdesk_user_access_token', t)
  // Start from an empty wall so the run is deterministic.
  localStorage.removeItem('quickdesk_wall_devices')
}, { s: SERVER, t: TOKEN })
await page.reload({ waitUntil: 'domcontentloaded' })
await page.goto(`${APP}/#/wall`, { waitUntil: 'networkidle' })
await page.waitForTimeout(1500)

await page.getByRole('button', { name: /添加设备|Add devices/ }).first().click()
await page.locator('.picker-label', { hasText: ONLINE }).first().click()
if (OFFLINE) await page.locator('.picker-label', { hasText: OFFLINE }).first().click()

const tileFor = (id) => page.locator('.wall-tile').filter({ hasText: id }).first()
await tileFor(ONLINE).waitFor({ timeout: 15000 })

// The tile reports "live" from inside the iframe once metadata arrives.
let live = ''
const deadline = Date.now() + 90000
while (Date.now() < deadline) {
  live = (await tileFor(ONLINE).locator('.tile-badge').textContent()) || ''
  if (/画面中|Live/.test(live)) break
  await page.waitForTimeout(1500)
}
console.log(`online tile state: ${live.trim()}`)

const frame = page.frames().find((f) => f.url().includes('wall=1') && f.url().includes(ONLINE))
if (!frame) throw new Error('the online tile never created a wall iframe')

const sample = async () => frame.evaluate(() => {
  const v = document.getElementById('remoteVideo')
  const app = window.remoteDesktopApp
  const bar = document.getElementById('connectionBar')
  return {
    videoWidth: v ? v.videoWidth : 0,
    currentTime: v ? v.currentTime : 0,
    mouse: !!app?.mouseHandler,
    keyboard: !!app?.keyboardHandler,
    touch: !!app?.touchHandler,
    wallMode: !!app?._wallMode,
    // The tile must not even offer its own controls.
    chromeHidden: !bar || getComputedStyle(bar).display === 'none',
  }
})
const first = await sample()
await page.waitForTimeout(3000)
const second = await sample()
console.log('iframe sample:', JSON.stringify({ first, second }))

let offlinePlaceholder = null
if (OFFLINE) {
  offlinePlaceholder = await tileFor(OFFLINE).locator('.tile-placeholder').textContent()
  console.log(`offline tile placeholder: ${(offlinePlaceholder || '').trim()}`)
  console.log(`offline tile has iframe: ${await tileFor(OFFLINE).locator('iframe').count() > 0}`)
}

// ---- input arming -------------------------------------------------------
// Default is view-only; unlocking one tile must create the input handlers and
// actually deliver keystrokes, and locking it again must remove them.
let input = null

// The tile reports both connected and live, and the arm button is only enabled
// while one of those holds, so retry briefly instead of racing the badge.
async function armTile(maxTries = 8) {
  for (let i = 0; i < maxTries; i++) {
    const btn = tileFor(ONLINE).getByTitle(/解锁输入|Unlock input/)
    if (await btn.count() && await btn.isEnabled()) {
      await btn.click({ timeout: 5000 }).catch(() => {})
      await page.waitForTimeout(1200)
    }
    if (await tileFor(ONLINE).locator('.tile-armed').count()) return true
    await page.waitForTimeout(1500)
  }
  return false
}

if (await tileFor(ONLINE).getByTitle(/解锁输入|Unlock input/).count()) {
  console.log('before arming:', JSON.stringify(await tileFor(ONLINE).evaluate((el) => ({
    badge: el.querySelector('.tile-badge')?.textContent,
    buttons: [...el.querySelectorAll('.tile-btn')].map((b) => ({ title: b.title, disabled: b.disabled })),
  }))))
  const before = guestText()
  // Lower-case on purpose: the point of the assertion is delivery through the
  // armed tile, not the keyboard's shift handling.
  const token = `qd-wall-${Date.now().toString().slice(-6)}`
  const armedOk = await armTile()
  console.log(`armed=${armedOk}`)
  const armedBadge = (await tileFor(ONLINE).locator('.tile-armed').textContent()) || ''
  const armed = await sample()
  await frame.evaluate(() => document.getElementById('videoContainer')?.focus())
  await page.keyboard.type(token, { delay: 40 })
  await page.waitForTimeout(1500)
  const after = guestText()
  await tileFor(ONLINE).getByTitle(/锁定输入|Lock input/).click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(1000)
  const locked = await sample()
  input = {
    armedOk,
    armedBadge: armedBadge.trim(),
    armedMouse: armed.mouse,
    armedKeyboard: armed.keyboard,
    lockedMouse: locked.mouse,
    lockedKeyboard: locked.keyboard,
    token,
    delivered: after.toLowerCase().includes(token.toLowerCase()),
    beforeTail: before.slice(-30),
    afterTail: after.slice(-30),
  }
  console.log('input arming:', JSON.stringify(input))
  await page.screenshot({ path: `${OUT}/wall-armed.png`, fullPage: false })
}

await page.screenshot({ path: `${OUT}/wall.png`, fullPage: false })
console.log(`screenshot: ${OUT}/wall.png`)

const ok = first.videoWidth > 0 && second.currentTime > first.currentTime &&
  first.wallMode && !first.mouse && !first.keyboard && !first.touch &&
  first.chromeHidden &&
  (!input || (input.armedOk && input.armedMouse && input.armedKeyboard && !input.lockedMouse && !input.lockedKeyboard && input.delivered)) &&
  (!OFFLINE || (/离线|Offline/.test(offlinePlaceholder || '')))

console.log(`video ${first.videoWidth}px, clock ${first.currentTime.toFixed(2)} -> ${second.currentTime.toFixed(2)}, ` +
  `handlers mouse=${first.mouse} keyboard=${first.keyboard} touch=${first.touch} chromeHidden=${first.chromeHidden}`)
console.log(ok ? 'RESULT=PASS' : 'RESULT=FAIL')
await browser.close()
process.exit(ok ? 0 : 1)
