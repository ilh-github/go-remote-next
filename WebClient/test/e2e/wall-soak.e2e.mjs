import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import net from 'node:net'
import { chromium } from 'playwright'
import { createServer } from 'vite'

// The interval rule lives in a module so it can be tested without a two-hour
// run; the E2E only decides when to ask it.
import { buildSamplingPlan, intervalAt, denseResolution } from '../../js/soak/samplingPlan.js'

// The allocation-profile rules live in a module for the same reason: what
// counts as a period is the part worth testing, and it cannot be tested from
// inside a two-hour run.
import { periodicity, summarizeAllocationProfile } from '../../js/soak/allocationProfile.js'

const WEB_ROOT = new URL('../../', import.meta.url).pathname
const DEVICE_COUNT = positiveInteger(process.env.QD_WALL_DEVICES, 40)
const SOAK_SECONDS = positiveInteger(process.env.QD_WALL_SOAK_SECONDS, 12)
const SAMPLE_MS = positiveInteger(process.env.QD_WALL_SAMPLE_MS, 1000)
// Dense sampling inside one window of the soak, coarse outside it.
//
// The 2026-10-24 two-hour run showed FPS stepping down between 267 and 681
// seconds rather than sliding, and a one-second series over two hours cannot
// say what happened at the step - it only says the average is bad. These three
// variables carve out a window where the sampling interval drops to
// `DENSE_SAMPLE_MS`, so a short run inside the interesting stretch yields the
// resolution the full-length run cannot. Zero-width defaults (start == end)
// disable the feature, which is what every existing run gets.
const DENSE_FROM_MS = positiveInteger(process.env.QD_WALL_DENSE_FROM_MS, 0)
const DENSE_TO_MS = positiveInteger(process.env.QD_WALL_DENSE_TO_MS, 0)
const DENSE_SAMPLE_MS = positiveInteger(process.env.QD_WALL_DENSE_SAMPLE_MS, 100)
// A cheap independent peak tracker, on by default.
//
// The two-hour run reported `heapGrowthMb` of 535 MiB, and the 120-second dense
// run showed heap oscillating across 111% of its own mean while FPS and node
// count held still. That is the shape of GC firing between samples rather than
// of a leak, and the measurement is exposed to it twice: it samples only at the
// start and the end, and the samples in between are limited by how expensive a
// full sample is (200 viewers cost about seven seconds each, so a 200 ms dense
// interval is overset fourteen-fold).
//
// A tracker that does nothing but read `JSHeapUsedSize` on an interval costs a
// fraction of a full sample, so it can run many times more often and hold the
// maximum. If the peak is far above the final reading, GC timing is the
// explanation and the growth number is an artefact; if the peak tracks the final
// reading, the growth is real and the allocator hunt is worth doing. Either way
// the answer comes from a number rather than from an argument.
const PEAK_TRACK_MS = positiveInteger(process.env.QD_WALL_PEAK_TRACK_MS, 1000)
// How many heap readings the periodicity analysis keeps. The peak tracker runs
// for the whole soak, so an uncapped series is 7200 points on the documented
// two-hour run: unreadable in a summary and no faster to analyse. The most
// recent readings are the ones that matter, because a period is a property of
// the recent allocation behaviour rather than of the startup.
const HEAP_SERIES_LIMIT = positiveInteger(process.env.QD_WALL_HEAP_SERIES_LIMIT, 600)
const MIN_MAIN_FPS = positiveNumber(process.env.QD_WALL_MIN_MAIN_FPS, 1)
const MAX_HEAP_GROWTH_MB = positiveNumber(process.env.QD_WALL_MAX_HEAP_GROWTH_MB, 256)
// Startup for hundreds of animated iframes plus the final sampling pass takes
// minutes under load, so the node:test timeout needs headroom beyond the soak
// window itself.
//
// Two hours, not one. The documented release run is 200 viewers for 7200
// seconds and does not set this variable, so the default is what the gate
// actually gets: 7200s of soak plus only 3600s for setup and sampling. Waiting
// for 200 tiles to go live, then sampling three frames through the CDP, takes
// far longer under a two-hour load than it does during the 30-second smoke
// run this default was sized against. When the wrapper times out, node:test
// fails the whole thing — including the two hours that already passed.
const SOAK_TIMEOUT_HEADROOM_MS = positiveNumber(process.env.QD_WALL_TIMEOUT_HEADROOM_MS, 2 * 60 * 60 * 1000)
// Built once from the environment: a plan that was assembled per sample would
// make the interval a function of when the loop happened to ask, which is the
// opposite of reproducible.
const SAMPLING_PLAN = buildSamplingPlan({
  intervalMs: SAMPLE_MS,
  fromMs: DENSE_FROM_MS,
  toMs: DENSE_TO_MS,
  denseIntervalMs: DENSE_SAMPLE_MS,
})
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
  // The 2h+ gate outlives node:test's timeout, so the soak runs as a plain
  // awaited routine and the test only wraps setup plus assertions. Startup for
  // hundreds of iframes and the final CDP/frame sampling are slow, so the
  // wrapper timeout has to cover them, not just the soak window.
  { timeout: SOAK_SECONDS * 1000 + SOAK_TIMEOUT_HEADROOM_MS },
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
      assert.equal(await page.evaluate(() => !!document.querySelector('.wall-tools .primary')), true)
      const baseline = await performanceMetrics(cdp)
      const startedAt = Date.now()
      const startedFrames = await page.evaluate(() => window.__wallSoakMainFrames)
      const samples = []
      const peak = { heapBytes: baseline.JSHeapUsedSize || 0, nodes: baseline.Nodes || 0, atMs: 0 }
      let peakReads = 0
      let heapSeries = []
      let peakTracker = null
      // Sampling allocation across the whole soak rather than snapshotting the
      // heap at the end. A final snapshot says how much is live; it cannot say
      // what allocated it, and a periodic allocator that has already been
      // collected leaves no trace in one. `samplingHeap` is what costs something
      // small on an interval, so the profile covers the run instead of a moment.
      let allocationProfile = null
      try {
        await cdp.send('HeapProfiler.enable')
        await cdp.send('HeapProfiler.startSampling', {
          samplingInterval: 32768,
        })
      } catch (error) {
        // A browser without the domain is not a failed run: the heap series and
        // the FPS trace still gate it, and the profile is reported as absent
        // rather than as zero, because those are different statements.
        console.log('WALL_SOAK_ALLOCATION_PROFILE_UNAVAILABLE', String(error))
      }
      if (PEAK_TRACK_MS > 0) {
        // The peak tracker already reads the heap on its own interval, so the
        // series the periodicity analysis needs is a by-product rather than a
        // second measurement. Kept bounded: a two-hour run at one read per
        // second is 7200 points, and a series that large is neither readable nor
        // faster to analyse.
        heapSeries = []
        // Deliberately not awaited inside the sampling loop: the point is that
        // this runs independently of it, so a slow sample cannot starve it.
        peakTracker = setInterval(async () => {
          try {
            const m = await performanceMetrics(cdp)
            peakReads += 1
          if (heapSeries.length < HEAP_SERIES_LIMIT) {
            heapSeries.push({ atMs: Date.now() - startedAt, bytes: m.JSHeapUsedSize || 0 })
          }
            if ((m.JSHeapUsedSize || 0) > peak.heapBytes) {
              peak.heapBytes = m.JSHeapUsedSize || 0
              peak.nodes = m.Nodes || 0
              peak.atMs = Date.now() - startedAt
            }
          } catch { /* the CDP session is gone; the run is over anyway */ }
        }, PEAK_TRACK_MS)
      }
      // Time series behind the two numbers that gate the run. `mainFps` is an
      // average over the whole soak, so 0.77 cannot distinguish "slow from the
      // first second" from "fine until it collapsed"; a per-sample series can,
      // and that is the difference between a capacity limit and a leak.
      // The first sample is not a point: the window that precedes it covers
      // `Start rotation` and the first CDP read, during which frames advance
      // for reasons that have nothing to do with the soak. Counting it would
      // report a fake 0 and pin `worst` to second zero forever.
      let previousFrames = startedFrames
      let previousSampleAt = 0
      let index = 0
      // Measured, not assumed: how long one sample actually costs is what decides
      // whether a dense window is dense. Recorded as a rolling figure so the
      // summary can report the achieved resolution next to the configured one.
      let samplingCostMs = 0

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
        const now = Date.now()
        const frames = await page.evaluate(() => window.__wallSoakMainFrames)
        const metrics = await performanceMetrics(cdp)
        if (previousSampleAt > 0) {
          const elapsedSincePrevious = (now - previousSampleAt) / 1000
          samples[samples.length - 1].fps = elapsedSincePrevious > 0
            ? Number(((frames - previousFrames) / elapsedSincePrevious).toFixed(2))
            : 0
        }
        samples[samples.length - 1].heapMb = Number(
          (((metrics.JSHeapUsedSize || 0) / 1024 / 1024)).toFixed(1),
        )
        samples[samples.length - 1].nodes = metrics.Nodes || 0
        previousFrames = frames
        previousSampleAt = now
        index += 1
        // The interval is chosen per sample, not fixed for the run: inside the
        // dense window the loop tightens so a step change lands on several
        // points instead of being interpolated between two of them.
        // Rolling average of what a sample costs, so a slow start-up sample does
        // not permanently misreport the achievable resolution.
        const sampleCost = Date.now() - sampleStarted
        samplingCostMs = samplingCostMs === 0
          ? sampleCost
          : Math.round(samplingCostMs * 0.7 + sampleCost * 0.3)
        const elapsedNow = Date.now() - startedAt
        const interval = intervalAt(SAMPLING_PLAN, elapsedNow)
        const remaining = interval - (Date.now() - sampleStarted)
        if (remaining > 0) await new Promise((resolve) => setTimeout(resolve, remaining))
      }

      if (peakTracker) clearInterval(peakTracker)

      try {
        const taken = await cdp.send('HeapProfiler.stopSampling')
        allocationProfile = taken?.profile || null
        await cdp.send('HeapProfiler.disable')
      } catch (error) {
        console.log('WALL_SOAK_ALLOCATION_PROFILE_STOP_FAILED', String(error))
      }

      const rotateToolbar = await page.evaluate(() => {
        const wall = document.querySelector('.wall-tools')
        const buttons = [...(wall?.querySelectorAll('button') || [])]
          .map((button) => (button.textContent || '').trim())
        return {
          html: wall ? wall.innerHTML.slice(0, 400) : 'MISSING',
          buttons,
          tiles: document.querySelectorAll('.wall-tile').length,
          rotating: !!wall?.querySelector('.primary'),
        }
      })
      console.log('WALL_SOAK_TOOLBAR', JSON.stringify(rotateToolbar))
      await clickByName(page, 'Stop rotation')
      assert.equal(await page.evaluate(() => !document.querySelector('.wall-tools .primary')), true)
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
        }, { timeout: 120000 })
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
        // Echoed so a summary is self-describing: a trace read without knowing
        // where the dense window was cannot tell a real step from an artefact of
        // the sampling rate changing.
        dense: {
          fromMs: DENSE_FROM_MS,
          toMs: DENSE_TO_MS,
          intervalMs: DENSE_SAMPLE_MS,
          samples: samples.filter((sample) => (
            sample.elapsedMs >= DENSE_FROM_MS && sample.elapsedMs < DENSE_TO_MS
          )).length,
          // The interval that was configured is not the interval that was
          // achieved: under 200 viewers one sample costs seconds, so a 500 ms
          // window is overshot by however long the sampling itself takes. Both
          // numbers are reported, because a trace whose resolution is a property
          // of machine load while the summary claims it was configured is a lie
          // about its own evidence.
          achievedIntervalMs: Number(((Date.now() - startedAt) / Math.max(1, samples.length)).toFixed(0)),
          resolution: denseResolution(SAMPLING_PLAN, samplingCostMs),
        },
        mainFps: Number(mainFps.toFixed(2)),
        heapGrowthMb: Number(heapGrowthMb.toFixed(2)),
        heapEndMb: Number(((finalMetrics.JSHeapUsedSize || 0) / 1024 / 1024).toFixed(2)),
        nodesEnd: finalMetrics.Nodes || 0,
        sampledFrames,
        // The peak and how often it was read. `peakOverFinalMb` is the number
        // that decides the question: a large one says the growth figure is a GC
        // artefact and the allocator hunt is premature, a small one says the
        // growth is real.
        peak: {
          intervalMs: PEAK_TRACK_MS,
          reads: peakReads,
          atMs: peak.atMs,
          heapMb: Number((peak.heapBytes / 1024 / 1024).toFixed(1)),
          nodes: peak.nodes,
          peakOverFinalMb: Number(
            ((peak.heapBytes - (finalMetrics.JSHeapUsedSize || 0)) / 1024 / 1024).toFixed(1),
          ),
        },
        // What the heap series says about a repeating allocation source. This is
        // the question the 2026-10-24 run left open: FPS collapsed while the
        // heap was still climbing, so the allocator is outside the frame loop,
        // and "outside the frame loop" is a place to look rather than a name.
        //
        // A period is reported only when the correlation is decisive against
        // the series' own noise. The heap was measured swinging across 111% of
        // its mean while nothing was wrong, so a weak hit would be a finding in
        // name only. `periodMs` of 0 therefore means "no cycle established",
        // which is an answer, not a missing field.
        allocationCycle: {
          samples: heapSeries.length,
          ...periodicity(heapSeries),
        },
        // Where the allocation actually came from, when the browser could say.
        // `null` top means nothing was sampled, which is reported as absent
        // rather than as a zero share: a run that could not profile is not a run
        // that profiled nothing.
        allocationProfile: summarizeAllocationProfile(allocationProfile),
        pageErrors,
        // The series is what tells a gradual slide from a step change: the
        // average above cannot. Kept coarse on purpose - one point per sample
        // interval - so a two-hour run produces a readable trace rather than
        // thousands of points.
        trace: soakTrace(samples),
      }))

      assert.ok(elapsedMs >= SOAK_SECONDS * 1000)
      assert.ok(mainFps >= MIN_MAIN_FPS, `main animation fps ${mainFps.toFixed(2)}`)
      // The heap gate is on the peak, not on the start-to-end difference.
      //
      // The difference was shown to measure noise: three identical 200-viewer
      // runs produced -156, -11 and +296 MiB, and the peak sat 250 MiB above the
      // final reading in one of them and 13 MiB below it in another. A number
      // that swings 300 MiB under fixed parameters cannot gate anything - it
      // would fail random runs and hide real ones. The peak is the maximum over
      // the whole run, so it is not an artefact of when GC happened to fire
      // between two samples, and a wall that genuinely holds hundreds of
      // viewers has to hold that maximum.
      //
      // Both figures are still reported, so the trace keeps the shape and the
      // gate keeps a number it can defend.
      const peakGrowthMb = (peak.heapBytes - (baseline.JSHeapUsedSize || 0)) / 1024 / 1024
      assert.ok(
        peakGrowthMb <= MAX_HEAP_GROWTH_MB,
        `JS heap peaked ${peakGrowthMb.toFixed(1)} MiB above baseline (start-to-end ${heapGrowthMb.toFixed(1)} MiB)`,
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

// Reduces the per-sample series to a trace a human can read: one point per
// bucket, plus the first, last and worst points. A run that degrades steadily
// shows a falling `fps` line with a rising `heapMb` one; a step change shows a
// flat line that drops once, which is a different bug.
function soakTrace(samples) {
  const withMetrics = samples.filter((sample) => typeof sample.fps === 'number')
  if (withMetrics.length === 0) return []
  const buckets = 12
  const perBucket = Math.max(1, Math.floor(withMetrics.length / buckets))
  const points = []
  for (let start = 0; start < withMetrics.length; start += perBucket) {
    const bucket = withMetrics.slice(start, start + perBucket)
    points.push({
      atSeconds: Math.round(bucket[bucket.length - 1].elapsedMs / 1000),
      fps: Number((bucket.reduce((sum, s) => sum + s.fps, 0) / bucket.length).toFixed(2)),
      heapMb: Number((bucket[bucket.length - 1].heapMb || 0).toFixed(1)),
      nodes: bucket[bucket.length - 1].nodes || 0,
    })
  }
  const worst = withMetrics.reduce((low, s) => (s.fps < low.fps ? s : low), withMetrics[0])
  return {
    points,
    worst: {
      atSeconds: Math.round(worst.elapsedMs / 1000),
      fps: worst.fps,
      heapMb: worst.heapMb,
    },
    firstFps: withMetrics[0].fps,
    lastFps: withMetrics[withMetrics.length - 1].fps,
  }
}

// A wall with hundreds of iframes makes Playwright's locator engine slow enough
// to time out, and getByRole() additionally depends on the accessibility tree.
// Clicking is done in one page-side query instead.
async function clickByName(page, name) {
  return page.evaluate((label) => {
    const button = [...document.querySelectorAll('button')]
      .find((candidate) => (candidate.textContent || '').trim() === label)
    if (!button) return false
    button.click()
    return true
  }, name)
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
