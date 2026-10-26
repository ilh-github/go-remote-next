import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { createReadStream } from 'node:fs'
import { readFile, stat } from 'node:fs/promises'
import http from 'node:http'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'

const WEB_ROOT = path.resolve(fileURLToPath(new URL('../../', import.meta.url)))
const LOCAL_IMPORT_MAP = `<script type="importmap">
    {
      "imports": {
        "@noble/curves/ed25519": "/vendor/noble/curves/ed25519.js",
        "@noble/curves/abstract/utils": "/vendor/noble/curves/abstract/utils.js",
        "@noble/curves/": "/vendor/noble/curves/",
        "@noble/hashes/sha256": "/vendor/noble/hashes/sha256.js",
        "@noble/hashes/sha512": "/vendor/noble/hashes/sha512.js",
        "@noble/hashes/hmac": "/vendor/noble/hashes/hmac.js",
        "@noble/hashes/": "/vendor/noble/hashes/"
      }
    }
    </script>`

const MIME_TYPES = {
  '.css': 'text/css; charset=utf-8',
  '.html': 'text/html; charset=utf-8',
  '.ico': 'image/x-icon',
  '.js': 'text/javascript; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.map': 'application/json; charset=utf-8',
  '.png': 'image/png',
  '.svg': 'image/svg+xml',
  '.woff2': 'font/woff2',
}

let browser

before(async () => {
  browser = await chromium.launch({
    channel: process.env.QD_BROWSER_CHANNEL || 'chrome',
    headless: true,
  })
})

after(async () => {
  await browser?.close()
})

// QD-907 asks for evidence on a genuinely bad link, not only a socket that is
// told to close. The peer connection here is faked because WebRTC is not what
// this test is about, but the signaling WebSocket is a real browser socket
// speaking the real protocol, so the client opens it, sends auth and waits for
// a reply that may or may not arrive.
//
// The failure mode worth guarding is the one where a bad link is mistaken for a
// dead one. Losing the auth reply makes the transport reject the connection as
// "closed before auth", and that rejection has to stay on the retryable path:
// if it were treated as final, an operator on a lossy connection would be told
// the session is over while the server is still reachable.
test('keeps retrying when the auth reply is lost on a lossy link', async (t) => {
  // Deterministic, not random: a reproducible run is what makes a failure
  // actionable. The first reply is lost, which is the case that matters - it is
  // the difference between "retry again" and "give up now".
  const dropPattern = [true, false]
  for (const latencyMs of [0, 150]) {
    await t.test(`first reply lost, ${latencyMs}ms latency`, async () => {
      const harness = await createHarness()
      try {
        await harness.openRemote()
        await harness.waitForConnected()

        harness.state.signalLoss = { latencyMs, dropPattern, attempts: 0 }
        await harness.closeCurrentSocket()

        // The lost reply costs one attempt; the next one lands. Four is the
        // whole budget, so this proves the loss was survivable rather than
        // terminal.
        await until(
          () => harness.state.authFrames.length >= 2,
          { timeoutMs: 25000, label: 'a second auth attempt after the lost reply' },
        )
        await harness.waitForConnected({ expectedSignalConnections: 3 })

        assert.equal(
          await harness.reconnectTaskActive(),
          false,
          'the client must settle once the link answers, not keep retrying',
        )
        assert.equal(
          await harness.statusText(),
          'Connected',
          'the session must come back rather than surface the exhausted message',
        )
        assert.ok(
          !harness.state.connections.some((entry) => entry.error_msg === 'reconnect_exhausted'),
          'a lost auth reply must not be reported as an exhausted reconnect',
        )
      } finally {
        await harness.close()
      }
    })
  }
})

// The other side of the same boundary: loss that never lets up must still end
// in the exhausted state, with the retries it is supposed to have made. Without
// this the test above could pass by the client giving up early and calling it a
// recovery.
test('still reports exhaustion when the link never answers', async () => {
  const harness = await createHarness()
  try {
    await harness.openRemote()
    await harness.waitForConnected()

    // Every reply is lost, so the four attempts all fail the same way.
    harness.state.signalLoss = { latencyMs: 0, dropPattern: [true], attempts: 0 }
    await harness.closeCurrentSocket()

    await until(
      () => harness.state.authFrames.length >= 4,
      { timeoutMs: 30000, label: 'four attempts against a link that never answers' },
    )
    await harness.page.waitForFunction(
      () => document.querySelector('#connStatus')?.textContent
        === 'Connection could not be restored after multiple attempts. Please reconnect.',
      null,
      { timeout: 15000 },
    )
    await until(
      () => harness.state.connections.some((entry) => entry.error_msg === 'reconnect_exhausted'),
      { timeoutMs: 5000, label: 'exhausted connection audit row' },
    )

    assert.equal(await harness.reconnectTaskActive(), false)
    assert.equal(await harness.dotClass(), 'conn-dot failed')
  } finally {
    await harness.close()
  }
})

// QD-907's remaining boundary: sustained jitter. A link that answers late but
// does answer is neither the clean case nor the dead one, and it is the shape a
// real operator on a bad connection lives in for minutes at a time. The client
// has to treat it as usable: recover, settle, and stop asking.
//
// The distinction under test is between "slow" and "broken". The retry backoff
// exists so a link that never answers is not hammered; if jitter were mistaken
// for failure, an operator on a merely bad link would be told the session is
// over while the server is still there.
test('survives sustained jitter and settles instead of retrying', async (t) => {
  // Late, but never lost, across four consecutive reconnects. The pattern is
  // fixed rather than random so a failure names a specific attempt.
  const jitter = [120, 260, 400, 180]
  await t.test('every reply is late and none is lost', async () => {
    const harness = await createHarness()
    try {
      await harness.openRemote()
      await harness.waitForConnected()

      for (const [index, latencyMs] of jitter.entries()) {
        harness.state.signalLoss = {
          latencyMs,
          dropPattern: [false],
          attempts: 0,
        }
        await harness.closeCurrentSocket()

        // One new socket per reconnect, so the count is how many attempts the
        // client actually made - it must be exactly one per injected drop, not
        // a retry storm on top of it.
        await until(
          () => harness.sockets.length === index + 2,
          { timeoutMs: 25000, label: `reconnect ${index + 1} of ${jitter.length}` },
        )
        await harness.waitForConnected({ expectedSignalConnections: index + 2 })

        assert.equal(
          await harness.reconnectTaskActive(),
          false,
          `attempt ${index + 1}: a link that answered must not leave a retry running`,
        )
        assert.equal(
          await harness.statusText(),
          'Connected',
          `attempt ${index + 1}: a slow reply is not a failed session`,
        )
        assert.ok(
          !harness.state.connections.some((entry) => entry.error_msg === 'reconnect_exhausted'),
          `attempt ${index + 1}: jitter must not be recorded as an exhausted reconnect`,
        )
      }
    } finally {
      await harness.close()
    }
  })

  // The other end of jitter: late *and* eventually lost. The client must spend
  // its whole retry budget rather than giving up on the first slow reply, and
  // must land in the exhausted state once the link stops answering.
  //
  // Every attempt is dropped here, but each one hangs first - which is the
  // point. A link that stalls before failing is where a client with short
  // patience declares the session dead while the server is still reachable, and
  // the backoff has to be what carries it to a real answer.
  await t.test('jitter that ends in a hang still exhausts', async () => {
    const harness = await createHarness()
    try {
      await harness.openRemote()
      await harness.waitForConnected()

      harness.state.signalLoss = {
        latencyMs: 150,
        dropPattern: [true],
        attempts: 0,
      }
      await harness.closeCurrentSocket()

      // Four attempts is the whole budget, so waiting for all of them is what
      // proves the client did not settle for "one slow reply, therefore broken".
      await until(
        () => harness.state.authFrames.length >= 5,
        { timeoutMs: 60000, label: 'the full retry budget against a hanging link' },
      )
      await harness.page.waitForFunction(
        () => document.querySelector('#connStatus')?.textContent
          === 'Connection could not be restored after multiple attempts. Please reconnect.',
        null,
        { timeout: 15000 },
      )
      const exhausted = await harness.state.connections.filter(
        (entry) => entry.error_msg === 'reconnect_exhausted',
      ).length
      assert.equal(exhausted, 1, `expected one exhausted row, got ${exhausted}`)
      assert.equal(await harness.dotClass(), 'conn-dot failed')
    } finally {
      await harness.close()
    }
  })
})

test('recovers on the first three reconnect attempts and ignores stale session events', async (t) => {
  for (const successAt of [1, 2, 3]) {
    await t.test(`attempt ${successAt}`, async () => {
      const harness = await createHarness()
      try {
        await harness.openRemote()
        await harness.waitForConnected()
        await harness.page.evaluate(() => {
          window.__staleSession = window.remoteDesktopApp.session
        })

        harness.state.signalFailuresRemaining = successAt - 1
        await harness.closeCurrentSocket()

        await until(
          () => harness.state.verifyCalls === successAt
            && harness.state.authFrames.length === successAt + 1,
          { timeoutMs: 15000, label: `attempt ${successAt} reconnect handshakes` },
        )
        await harness.waitForConnected({ expectedSignalConnections: successAt + 1 })

        const tokens = harness.state.authFrames.map((frame) => frame.signal_token)
        assert.deepEqual(
          tokens,
          [
            'initial-token',
            ...Array.from({ length: successAt }, (_, index) => `retry-token-${index + 1}`),
          ],
        )

        if (successAt === 1) {
          await harness.page.evaluate(() => {
            window.__staleSession._emitEvent('signalingDisconnected', {
              code: 1012,
              reason: 'late event from replaced socket',
            })
            window.__staleSession._emitEvent('stateChange', {
              oldState: 'ACCEPTING',
              newState: 'FAILED',
              reason: 'PEER_DISCONNECTED',
            })
          })
          await harness.page.waitForTimeout(250)
          assert.equal(await harness.statusText(), 'Connected')
          assert.equal(await harness.reconnectTaskActive(), false)
        }
      } finally {
        await harness.close()
      }
    })
  }
})

test('shows a stable exhausted state after all four reconnect attempts fail', async () => {
  const harness = await createHarness()
  try {
    await harness.openRemote()
    await harness.waitForConnected()

    harness.state.signalFailuresRemaining = 100
    await harness.closeCurrentSocket()

    await until(
      () => harness.state.verifyCalls === 4 && harness.state.authFrames.length === 5,
      { timeoutMs: 20000, label: 'four failed reconnect attempts' },
    )
    await harness.page.waitForFunction(
      () => document.querySelector('#connStatus')?.textContent
        === 'Connection could not be restored after multiple attempts. Please reconnect.',
      null,
      { timeout: 10000 },
    )
    await until(
      () => harness.state.connections.some((entry) => entry.error_msg === 'reconnect_exhausted'),
      { timeoutMs: 5000, label: 'exhausted connection audit row' },
    )

    assert.equal(await harness.reconnectTaskActive(), false)
    assert.equal(await harness.dotClass(), 'conn-dot failed')
  } finally {
    await harness.close()
  }
})

test('stops immediately for a non-retryable access policy rejection', async () => {
  const harness = await createHarness()
  try {
    await harness.openRemote()
    await harness.waitForConnected()

    harness.state.accessMode = 'denied'
    await harness.closeCurrentSocket()

    await harness.page.waitForFunction(
      () => document.querySelector('#connStatus')?.textContent
        === 'The device access policy denied this connection',
      null,
      { timeout: 5000 },
    )
    await until(
      () => harness.state.connections.some((entry) => entry.error_msg === 'reconnect_rejected'),
      { timeoutMs: 5000, label: 'rejected connection audit row' },
    )

    assert.equal(harness.state.verifyCalls, 1)
    assert.equal(harness.state.authFrames.length, 1)
    assert.equal(await harness.reconnectTaskActive(), false)
  } finally {
    await harness.close()
  }
})

// The non-retryable case that used to be swallowed. A rejection raised *inside*
// the session connect - not while issuing the token - was caught by _connect,
// which answers false, so _attemptReconnect turned it into a generic
// "reconnect attempt failed" and the retry loop spent the whole budget
// rediscovering the same rejection. Four attempts where one would do, and an
// audit trail that says exhausted instead of rejected.
test('stops immediately for a non-retryable rejection inside the session connect', async () => {
  const harness = await createHarness()
  try {
    await harness.openRemote()
    await harness.waitForConnected()

    // The next connect fails the way a policy rejection does: the transport
    // closes before auth, which is where the retryable marker lives.
    harness.state.signalLoss = {
      latencyMs: 0,
      dropPattern: [true],
      attempts: 0,
      rejectAsNonRetryable: true,
    }
    await harness.closeCurrentSocket()

    // What the operator sees has to name the rejection, not the generic
    // exhausted message: the difference between "your access was denied" and
    // "the network kept failing" is the difference between fixing a policy and
    // waiting for a link.
    // The status has to name the rejection, not the generic exhausted message:
    // "signaling auth failed" sends the operator to their credentials, while
    // "could not be restored after multiple attempts" sends them to wait for a
    // network that was never broken.
    await harness.page.waitForFunction(
      () => (document.querySelector('#connStatus')?.textContent || '').includes('Signaling auth failed'),
      null,
      { timeout: 10000 },
    )
    await until(
      () => harness.state.connections.some((entry) => entry.error_msg === 'reconnect_rejected'),
      { timeoutMs: 5000, label: 'rejected connection audit row' },
    )

    // One attempt, not four. The initial connect is the only auth frame, so a
    // second one means the client retried a rejection it had already been told
    // was final.
    assert.equal(await harness.reconnectTaskActive(), false)
    assert.ok(
      harness.state.authFrames.length <= 2,
      `expected the rejection to stop the run, got ${harness.state.authFrames.length} auth frames`,
    )
  } finally {
    await harness.close()
  }
})

test('keeps a recording upload alive while signaling disconnects and reconnects', async (t) => {
  const harness = await createHarness()
  try {
    await harness.openRemote()
    await harness.waitForConnected()

    const supportsRecordingUpload = await harness.page.evaluate(
      () => typeof window.remoteDesktopApp?._uploadRecording === 'function',
    )
    if (!supportsRecordingUpload) {
      t.skip('recording upload is not present in this WebClient baseline')
      return
    }

    await harness.page.evaluate(() => {
      window.__uploadPromise = window.remoteDesktopApp._uploadRecording({
        blob: new Blob(['reconnect-upload'], { type: 'video/webm' }),
        mimeType: 'video/webm',
        durationMs: 1000,
        poster: Promise.resolve(null),
      })
    })

    await harness.state.secondArtifactStarted.promise
    await harness.closeCurrentSocket()
    await harness.waitForConnected({ expectedSignalConnections: 2 })
    assert.equal(harness.state.artifactAttempts, 2)
    assert.equal(await harness.reconnectTaskActive(), false)

    harness.state.releaseArtifact.resolve()
    const saved = await harness.page.evaluate(() => window.__uploadPromise)

    assert.equal(saved.id, 'recording-1')
    assert.equal(saved.size_bytes, 16)
    assert.equal(await harness.page.evaluate(() => window.remoteDesktopApp._recordingUploading), false)
    assert.match(await harness.logText(), /Recording uploaded and indexed/)
  } finally {
    harness.state.releaseArtifact.resolve()
    await harness.close()
  }
})

// QD-907's other remaining boundary: the process itself loses the network.
//
// This is a different failure from a socket closing, and the difference is what
// makes it worth its own test. A close arrives as an event the client can react
// to. A process that goes offline gets no event at all - the socket just stops
// working, and the client finds out only when it next tries to use it. So the
// behaviour under test is whether the client notices at all, and whether coming
// back online is enough to recover without the operator reloading the page.
test('recovers after the whole process loses and regains the network', async () => {
  const harness = await createHarness()
  try {
    await harness.openRemote()
    await harness.waitForConnected()

    await harness.setProcessOffline(true)
    // Give the browser time to fire its own offline event and for the socket to
    // notice, rather than racing it: a check made too early passes because
    // nothing has happened yet.
    await new Promise((resolve) => setTimeout(resolve, 1500))

    await harness.setProcessOffline(false)

    // Back online is the trigger. The client has to reconnect on its own - a
    // page that only recovers because the operator reloaded it has not handled
    // this case at all. Two sockets is the proof: the second one is the client
    // dialling again after the network returned.
    await harness.waitForConnected({ expectedSignalConnections: 2 })
    assert.equal(
      await harness.reconnectTaskActive(),
      false,
      'the client must settle once the network is back, not keep retrying',
    )
    assert.equal(
      await harness.statusText(),
      'Connected',
      'regaining the network must restore the session without a reload',
    )
    assert.ok(
      !harness.state.connections.some((entry) => entry.error_msg === 'reconnect_exhausted'),
      'a process-level outage must not be recorded as an exhausted reconnect',
    )
  } finally {
    await harness.close()
  }
})

async function createHarness() {
  const state = {
    accessMode: 'success',
    artifactAttempts: 0,
    authFrames: [],
    connections: [],
    signalFailuresRemaining: 0,
    signalLoss: null,
    secondArtifactStarted: deferred(),
    releaseArtifact: deferred(),
  }
  const server = await startWebServer(state)
  const baseUrl = `http://127.0.0.1:${server.address().port}`
  const wsUrl = `ws://127.0.0.1:${server.address().port}`
  const context = await browser.newContext({
    viewport: { width: 1280, height: 800 },
  })
  const sockets = []
  const pageErrors = []

  await context.addInitScript(({ serverUrl, accessCode, initialToken }) => {
    try {
      localStorage.setItem('quickdesk_signaling_url', serverUrl)
      localStorage.setItem('quickdesk_user_access_token', 'e2e-user-token')
      localStorage.setItem('quickdesk_lang', 'en-US')
      sessionStorage.setItem(
        `quickdesk_remote_handoff__${initialToken}`,
        JSON.stringify({ access_code: accessCode }),
      )
    } catch {
      // about:blank has an opaque origin; the next navigation gets real storage.
    }

    window.__fakePeerConnections = []
    class FakeRTCPeerConnection {
      constructor() {
        this.connectionState = 'new'
        this.iceConnectionState = 'new'
        this.iceGatheringState = 'complete'
        this.signalingState = 'stable'
        this.localDescription = null
        this.remoteDescription = null
        this.onicecandidate = null
        this.oniceconnectionstatechange = null
        this.onicegatheringstatechange = null
        this.onconnectionstatechange = null
        this.ontrack = null
        this.ondatachannel = null
        window.__fakePeerConnections.push(this)
      }

      addTransceiver() {
        return {
          setCodecPreferences() {},
        }
      }

      async createOffer() {
        return { type: 'offer', sdp: 'v=0\r\nm=video 9 UDP/TLS/RTP/SAVPF 96\r\n' }
      }

      async setLocalDescription(description) {
        this.localDescription = description
        this.signalingState = 'have-local-offer'
      }

      async setRemoteDescription(description) {
        this.remoteDescription = description
        this.signalingState = 'stable'
      }

      async addIceCandidate() {}

      async getStats() {
        return new Map()
      }

      close() {
        this.connectionState = 'closed'
        this.iceConnectionState = 'closed'
      }

      setIceState(nextState) {
        this.iceConnectionState = nextState
        this.connectionState = nextState === 'connected' ? 'connected' : this.connectionState
        this.oniceconnectionstatechange?.()
        this.onconnectionstatechange?.()
      }
    }

    Object.defineProperty(window, 'RTCPeerConnection', {
      configurable: true,
      value: FakeRTCPeerConnection,
    })
  }, {
    serverUrl: wsUrl,
    accessCode: 'e2e-access-code',
    initialToken: 'initial-token',
  })

  await context.routeWebSocket(
    (url) => url.pathname === '/v1/realtime/signal',
    async (socket) => {
      const entry = { socket, auth: null, payloads: [] }
      sockets.push(entry)
      socket.onMessage(async (message) => {
        const text = typeof message === 'string' ? message : message.toString()
        let frame
        try {
          frame = JSON.parse(text)
        } catch {
          entry.payloads.push(text)
          return
        }

        if (frame?.type === 'auth') {
          entry.auth = frame
          state.authFrames.push(frame)
          // A lossy link, decided per auth attempt rather than per socket: the
          // browser opens a real WebSocket either way, so what a bad link
          // changes is whether the reply comes back and how late it is. A
          // deterministic sequence keeps the run reproducible, which a random
          // drop rate would not.
          const loss = state.signalLoss
          if (loss) {
            const attempt = loss.attempts
            loss.attempts += 1
            const shouldDrop = loss.dropPattern[attempt % loss.dropPattern.length]
            if (shouldDrop) {
              if (loss.latencyMs > 0) await sleep(loss.latencyMs)
              // The server states why before it hangs up, which is what makes
              // this a rejection rather than a loss: the client is told the
              // credential is bad, and retrying it would ask the same question.
              if (loss.rejectAsNonRetryable) {
                socket.send(JSON.stringify({
                  type: 'error',
                  code: 'AUTH_INVALID',
                  message: 'The device access policy denied this connection',
                }))
              }
              await socket.close({ code: 1006, reason: 'injected packet loss' })
              return
            }
            if (loss.latencyMs > 0) await sleep(loss.latencyMs)
          }
          if (state.signalFailuresRemaining > 0) {
            state.signalFailuresRemaining -= 1
            await socket.close({ code: 1012, reason: 'injected signaling failure' })
            return
          }
          socket.send(JSON.stringify({ type: 'auth_ok' }))
          return
        }
        entry.payloads.push(frame)
      })
    },
  )

  const page = await context.newPage()
  page.on('pageerror', (error) => pageErrors.push(error.message))

  const harness = {
    state,
    context,
    page,
    sockets,
    // The whole browser process loses the network, which is a different failure
    // from a socket closing: there is no close event to react to, only the
    // browser's own offline state.
    async setProcessOffline(offline) {
      await context.setOffline(offline)
    },
    async openRemote() {
      await page.goto(
        `${baseUrl}/remote.html?server=${encodeURIComponent(wsUrl)}&device=e2e-device&st=initial-token`,
        { waitUntil: 'domcontentloaded' },
      )
    },
    async closeCurrentSocket() {
      await until(() => sockets.length > 0, { label: 'signaling socket' })
      const current = sockets.at(-1)
      await current.socket.close({ code: 1012, reason: 'injected network loss' })
    },
    async waitForConnected({ expectedSignalConnections } = {}) {
      if (expectedSignalConnections != null) {
        await until(
          () => sockets.length === expectedSignalConnections,
          { timeoutMs: 15000, label: `${expectedSignalConnections} signaling connections` },
        )
      }
      await page.waitForFunction(
        () => !!window.remoteDesktopApp?.session?.transport?.isConnected(),
        null,
        { timeout: 15000 },
      )
      await page.evaluate(() => {
        const peer = window.__fakePeerConnections.at(-1)
        if (!peer) throw new Error('missing fake RTCPeerConnection')
        peer.setIceState('connected')
      })
      await page.waitForFunction(
        () => document.querySelector('#connStatus')?.textContent === 'Connected',
        null,
        { timeout: 5000 },
      )
    },
    async statusText() {
      return page.locator('#connStatus').textContent()
    },
    async dotClass() {
      return page.locator('#connDot').getAttribute('class')
    },
    async reconnectTaskActive() {
      return page.evaluate(() => !!window.remoteDesktopApp?._reconnectTask)
    },
    async logText() {
      return page.locator('#logContainer').innerText()
    },
    async close() {
      await context.close()
      await closeServer(server)
      assert.deepEqual(pageErrors, [])
    },
  }
  return harness
}

async function startWebServer(state) {
  const server = http.createServer(async (req, res) => {
    const url = new URL(req.url, 'http://127.0.0.1')

    try {
      if (req.method === 'GET' && url.pathname === '/v1/ice-config') {
        return sendJson(res, 200, { ice_servers: [] })
      }

      const verifyMatch = url.pathname.match(/^\/v1\/devices\/([^/]+)\/access-code:verify$/)
      if (req.method === 'POST' && verifyMatch) {
        await drain(req)
        state.verifyCalls = (state.verifyCalls || 0) + 1
        if (state.accessMode === 'denied') {
          return sendJson(res, 403, {
            code: 'ACCESS_POLICY_DENIED',
            detail: 'injected policy rejection',
          })
        }
        return sendJson(res, 200, {
          signal_token: `retry-token-${state.verifyCalls}`,
          expires_at: new Date(Date.now() + 60000).toISOString(),
        })
      }

      if (req.method === 'POST' && url.pathname === '/v1/me/connections') {
        state.connections.push(JSON.parse(await readBody(req)))
        return sendJson(res, 200, { ok: true })
      }

      if (req.method === 'POST' && url.pathname === '/v1/me/recordings') {
        await drain(req)
        return sendJson(res, 200, { id: 'recording-1' })
      }

      if (req.method === 'PATCH' && url.pathname === '/v1/me/recordings/recording-1/artifact') {
        const total = Number(req.headers['upload-length']) || 0
        await drain(req)
        state.artifactAttempts += 1
        if (state.artifactAttempts === 1) {
          return sendJson(res, 503, { detail: 'injected transient upload failure' })
        }
        state.secondArtifactStarted.resolve()
        await state.releaseArtifact.promise
        res.writeHead(200, {
          'Content-Type': 'application/json; charset=utf-8',
          'Upload-Offset': String(total),
        })
        res.end(JSON.stringify({
          id: 'recording-1',
          size_bytes: total,
          status: 'completed',
        }))
        return
      }

      if (url.pathname.startsWith('/v1/')) {
        return sendJson(res, 404, { code: 'NOT_FOUND', detail: url.pathname })
      }

      let filePath
      if (url.pathname.startsWith('/vendor/noble/')) {
        filePath = resolveVendorPath(url.pathname)
      } else {
        filePath = path.resolve(WEB_ROOT, `.${decodeURIComponent(url.pathname)}`)
      }
      if (!isInside(filePath, WEB_ROOT) && !filePath.startsWith(path.join(WEB_ROOT, 'node_modules'))) {
        return sendText(res, 403, 'forbidden')
      }

      let info
      try {
        info = await stat(filePath)
      } catch {
        if (!path.extname(filePath)) {
          filePath += '.js'
          info = await stat(filePath)
        } else {
          throw new Error('not found')
        }
      }
      if (info.isDirectory()) {
        filePath = path.join(filePath, 'index.html')
      }

      if (url.pathname === '/remote.html') {
        const html = await readFile(filePath, 'utf8')
        const rewritten = html.replace(
          /<script type="importmap">[\s\S]*?<\/script>/,
          LOCAL_IMPORT_MAP,
        )
        res.writeHead(200, {
          'Content-Type': MIME_TYPES['.html'],
          'Cache-Control': 'no-store',
        })
        return res.end(rewritten)
      }

      res.writeHead(200, {
        'Content-Type': MIME_TYPES[path.extname(filePath)] || 'application/octet-stream',
        'Cache-Control': 'no-store',
      })
      createReadStream(filePath).pipe(res)
    } catch {
      sendText(res, 404, 'not found')
    }
  })

  await new Promise((resolve, reject) => {
    server.once('error', reject)
    server.listen(0, '127.0.0.1', resolve)
  })
  return server
}

function resolveVendorPath(pathname) {
  const match = pathname.match(/^\/vendor\/noble\/(curves|hashes)\/(.+)$/)
  if (!match) throw new Error('invalid vendor path')
  const packageName = match[1] === 'curves' ? '@noble/curves' : '@noble/hashes'
  const relative = match[2].replace(/^\/+/, '')
  return path.resolve(WEB_ROOT, 'node_modules', packageName, 'esm', relative)
}

function isInside(candidate, root) {
  const relative = path.relative(root, candidate)
  return relative === '' || (!relative.startsWith('..') && !path.isAbsolute(relative))
}

function sendJson(res, status, body) {
  res.writeHead(status, { 'Content-Type': MIME_TYPES['.json'] })
  res.end(JSON.stringify(body))
}

function sendText(res, status, body) {
  res.writeHead(status, { 'Content-Type': 'text/plain; charset=utf-8' })
  res.end(body)
}

function readBody(req) {
  return new Promise((resolve, reject) => {
    const chunks = []
    req.on('data', (chunk) => chunks.push(chunk))
    req.on('end', () => resolve(Buffer.concat(chunks).toString('utf8') || '{}'))
    req.on('error', reject)
  })
}

async function drain(req) {
  for await (const _ of req) {
    // Consume request bodies before delaying or responding.
  }
}

function deferred() {
  let resolve
  let reject
  const promise = new Promise((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

async function until(predicate, { timeoutMs = 5000, intervalMs = 25, label = 'condition' } = {}) {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    if (await predicate()) return
    await new Promise((resolve) => setTimeout(resolve, intervalMs))
  }
  throw new Error(`timed out waiting for ${label}`)
}

async function closeServer(server) {
  await new Promise((resolve, reject) => {
    server.close((error) => (error ? reject(error) : resolve()))
  })
}
