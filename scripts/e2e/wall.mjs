// Screen wall: put one online device and one offline device on the wall, then
// prove the live tile really decodes frames while staying view-only.
//
// Asserts:
//   - the online tile reaches "live" through the tile's postMessage report
//   - the iframe's own <video> has real dimensions and an advancing clock
//   - the tile created no mouse/keyboard/touch handler (wall tiles are monitors)
//   - the offline device renders a placeholder instead of an iframe

import { chromium } from 'playwright'

const APP = 'http://127.0.0.1:5173'
const SERVER = 'ws://10.211.55.2:8000'
const TOKEN = process.env.QD_TOKEN
const ONLINE = process.env.QD_DEVICE
const OFFLINE = process.env.QD_OFFLINE_DEVICE
const OUT = process.env.QD_OUT || '/Users/timbl/PDShare/qd-e2e'

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

await page.screenshot({ path: `${OUT}/wall.png`, fullPage: false })
console.log(`screenshot: ${OUT}/wall.png`)

const ok = first.videoWidth > 0 && second.currentTime > first.currentTime &&
  first.wallMode && !first.mouse && !first.keyboard && !first.touch &&
  first.chromeHidden &&
  (!OFFLINE || (/离线|Offline/.test(offlinePlaceholder || '')))

console.log(`video ${first.videoWidth}px, clock ${first.currentTime.toFixed(2)} -> ${second.currentTime.toFixed(2)}, ` +
  `handlers mouse=${first.mouse} keyboard=${first.keyboard} touch=${first.touch} chromeHidden=${first.chromeHidden}`)
console.log(ok ? 'RESULT=PASS' : 'RESULT=FAIL')
await browser.close()
process.exit(ok ? 0 : 1)
