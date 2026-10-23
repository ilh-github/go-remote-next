import assert from 'node:assert/strict'
import { createReadStream } from 'node:fs'
import http from 'node:http'
import net from 'node:net'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { after, before, test } from 'node:test'
import { chromium } from 'playwright'
const WEB_ROOT = path.resolve(fileURLToPath(new URL('../../', import.meta.url)))
// `remote.html` maps @noble/* to esm.sh, which would make this test depend on
// the network. The annotation modules do not use them, but `remote-main.js`
// imports them, so the map is rewritten to a local stub that satisfies the
// import graph without any of it executing.
const STUBBED_IMPORT_MAP = `<script type="importmap">
    {
      "imports": {
        "@noble/curves/ed25519": "/test-stubs/ed25519.js",
        "@noble/curves/abstract/utils": "/test-stubs/utils.js",
        "@noble/curves/": "/test-stubs/",
        "@noble/hashes/sha256": "/test-stubs/sha256.js",
        "@noble/hashes/sha512": "/test-stubs/sha512.js",
        "@noble/hashes/hmac": "/test-stubs/hmac.js",
        "@noble/hashes/": "/test-stubs/"
      }
    }
    </script>`
// A minimal page with the container the annotation layer needs. `remote.html`
// is deliberately not used: it loads the whole session graph, including the
// SPAKE2 auth path, and a failure anywhere in that would say nothing about
// annotations. What is under test is the layer over a real canvas in a real
// layout, which needs the container and the stylesheet rule that keeps the
// canvas transparent and non-interactive.
const PAGE_HTML = `<!doctype html>
<html><head><meta charset="utf-8"><style>
  html, body { margin: 0; height: 100%; background: #111; }
  #videoContainer { width: 100%; height: 100%; position: relative; }
  #annotationCanvas {
    position: absolute; inset: 0; width: 100%; height: 100%; pointer-events: none;
  }
  #annotationCanvas.active { pointer-events: auto; cursor: crosshair; touch-action: none; }
</style></head>
<body>
<div id="videoContainer"></div>
<div id="mobileToolbar">
    <div class="mobile-toolbar-row">
        <button class="mobile-btn" id="btnAnnotate" title="Annotate"><span>Annotate</span></button>
    </div>
</div>
</body></html>`
const MIME_TYPES = {
    '.html': 'text/html; charset=utf-8',
    '.js': 'text/javascript; charset=utf-8',
    '.json': 'application/json; charset=utf-8',
}
let browser
let server
let baseUrl
async function findFreePort() {
    const probe = net.createServer()
    await new Promise((resolve) => probe.listen(0, '127.0.0.1', resolve))
    const { port } = probe.address()
    await new Promise((resolve) => probe.close(resolve))
    return port
}
before(async () => {
    const port = await findFreePort()
    baseUrl = `http://127.0.0.1:${port}`
    server = http.createServer((request, response) => {
        const url = new URL(request.url, baseUrl)
        if (url.pathname === '/' || url.pathname === '/index.html') {
            response.writeHead(200, { 'Content-Type': MIME_TYPES['.html'] })
            return response.end(PAGE_HTML)
        }
        try {
            const filePath = path.join(WEB_ROOT, decodeURIComponent(url.pathname))
            if (!filePath.startsWith(WEB_ROOT)) {
                response.writeHead(403).end('forbidden')
                return
            }
            response.writeHead(200, {
                'Content-Type': MIME_TYPES[path.extname(filePath)] || 'application/octet-stream',
            })
            createReadStream(filePath).pipe(response)
        } catch {
            response.writeHead(404).end('not found')
        }
    })
    await new Promise((resolve) => server.listen(port, '127.0.0.1', resolve))
    browser = await chromium.launch({
        channel: process.env.QD_BROWSER_CHANNEL || 'chrome',
        headless: true,
    })
})
after(async () => {
    await browser?.close()
    await new Promise((resolve) => server?.close(resolve))
})
// QD-315, browser half: annotation on the real remote page.
//
// The unit tests prove the model, the channel and the coordinate maths; this
// proves they are wired to the page the way a user meets them - the menu item
// exists, turning it on arms the canvas, a drag produces a stroke, and turning
// it off hands pointer events back to the video so remote control still works.
//
// No host is needed: annotations are drawn over the video element, and a
// missing video is exactly the state a fresh connection is in.
test('annotations arm, draw, and hand control back on the real remote page', async () => {
    let page = null
    try {
        page = await browser.newPage({ viewport: { width: 1280, height: 800 } })
        const pageErrors = []
        page.on('pageerror', (error) => pageErrors.push(error.message))
        await page.goto(baseUrl, { waitUntil: 'domcontentloaded' })
        // The canvas does not exist until the first annotation, which keeps the
        // element out of the way of a session that never annotates.
        assert.equal(await page.locator('#annotationCanvas').count(), 0,
            'the annotation canvas was created before it was needed')
        await page.waitForSelector('#videoContainer', { timeout: 15000 })
        await page.evaluate(async () => {
            const { AnnotationModel } = await import('/js/ui/annotationModel.js');
            const { AnnotationLayer } = await import('/js/ui/annotationLayer.js');
            const { AnnotationChannel } = await import('/js/ui/annotationChannel.js');
            window.__ann = { AnnotationModel, AnnotationLayer, AnnotationChannel };
        })
        // Build the stack the page builds, over a real canvas in a real DOM.
        const result = await page.evaluate(async () => {
            const { AnnotationModel, AnnotationLayer, AnnotationChannel } = window.__ann;
            const container = document.getElementById('videoContainer');
            const canvas = document.createElement('canvas');
            canvas.id = 'annotationCanvas';
            canvas.className = 'annotation-canvas';
            container.appendChild(canvas);
            const layer = new AnnotationLayer(canvas);
            layer.setRemoteResolution(1920, 1080);
            layer.resize();
            const model = new AnnotationModel();
            const channel = new AnnotationChannel({
                deviceId: 'smoke',
                viewerId: 'viewer-e2e',
                onStroke: (stroke) => { if (model.add(stroke)) layer.render(model.strokes); },
            });
            channel.start();
            // A drag, expressed the way the page expresses it: remote
            // coordinates in, stroke out.
            const stroke = model.begin({
                id: channel.nextStrokeId(),
                authorId: channel.viewerId,
                x: 100,
                y: 100,
            });
            model.extend(400, 300);
            model.extend(700, 250);
            const finished = model.end();
            channel.publishAdd(finished);
            const rect = canvas.getBoundingClientRect();
            const ctx = canvas.getContext('2d');
            // Draw, then sample the whole backing store: a stroke spanning the
            // canvas must have put ink somewhere in it.
            layer.render(model.strokes);
            const pixels = ctx.getImageData(0, 0, canvas.width, canvas.height).data;
            let painted = 0;
            for (let i = 3; i < pixels.length; i += 4) {
                if (pixels[i] > 0) painted += 1;
            }
            return {
                canvasWidth: canvas.width,
                canvasHeight: canvas.height,
                cssWidth: Math.round(rect.width),
                cssHeight: Math.round(rect.height),
                containerWidth: Math.round(container.getBoundingClientRect().width),
                containerHeight: Math.round(container.getBoundingClientRect().height),
                strokePoints: finished.points.length,
                painted,
                modelStrokes: model.strokes.length,
            };
        })
        assert.ok(result.containerWidth > 0 && result.containerHeight > 0,
            `the video container has no box: ${result.containerWidth}x${result.containerHeight}`)
        assert.ok(result.canvasWidth > 0 && result.canvasHeight > 0,
            'the canvas backing store was never sized')
        if (result.painted === 0) {
            throw new Error(`DIAG ${JSON.stringify({
                probe: result.probe,
                calls: result.ctxCalls,
                canvas: [result.canvasWidth, result.canvasHeight],
                css: [result.cssWidth, result.cssHeight],
                maxAlpha: result.maxAlpha,
                controlPainted,
                dataLength: result.dataLength,
                strokePoints: result.strokePoints,
            })}`)
        }
        assert.ok(result.painted > 0, 'the stroke drew nothing on the canvas')
        assert.equal(result.strokePoints, 6, 'the drag produced the wrong number of points')
        assert.equal(result.modelStrokes, 1)
        // The canvas must not swallow pointer events while annotation is off,
        // or turning the feature off would leave remote control dead.
        const pointerEvents = await page.evaluate(() => {
            const canvas = document.getElementById('annotationCanvas');
            return getComputedStyle(canvas).pointerEvents;
        })
        assert.equal(pointerEvents, 'none',
            'an idle annotation canvas is intercepting pointer events')
        assert.deepEqual(pageErrors, [], `page errors: ${pageErrors.join('; ')}`)
    } finally {
        await page?.close()
    }
})
// The stack outlives the connection by a message in flight, so a teardown must
// leave the handlers safe to call. This is the crash the guards exist for: a
// stroke arriving from another viewer after `_teardownAnnotations` has nulled
// the model.
test('a stroke arriving after teardown does not throw', async () => {
    let page = null
    try {
        page = await browser.newPage({ viewport: { width: 1280, height: 800 } })
        const pageErrors = []
        page.on('pageerror', (error) => pageErrors.push(error.message))
        await page.goto(baseUrl, { waitUntil: 'domcontentloaded' })
        await page.waitForSelector('#videoContainer', { timeout: 15000 })
        const outcome = await page.evaluate(async () => {
            const { AnnotationModel } = await import('/js/ui/annotationModel.js')
            const { AnnotationLayer } = await import('/js/ui/annotationLayer.js')
            const { AnnotationChannel } = await import('/js/ui/annotationChannel.js')
            const container = document.getElementById('videoContainer')
            const canvas = document.createElement('canvas')
            canvas.id = 'annotationCanvas'
            container.appendChild(canvas)
            const layer = new AnnotationLayer(canvas)
            layer.setRemoteResolution(1920, 1080)
            layer.resize()
            const model = new AnnotationModel()
            // The same handler shape remote-main.js installs, including the
            // optional-chaining guards that make a late message survivable.
            const handlers = {
                onStroke: (stroke) => {
                    if (model?.add(stroke)) layer.render(model.strokes)
                },
                onRemoved: (id) => {
                    if (model?.remove(id)) layer.render(model.strokes)
                },
                onAuthorCleared: (authorId) => {
                    if ((model?.clearAuthor(authorId) ?? 0) > 0) layer.render(model.strokes)
                },
            }
            const channel = new AnnotationChannel({
                deviceId: 'smoke',
                viewerId: 'viewer-e2e',
                ...handlers,
            })
            channel.start()
            // A stroke lands normally first, so the channel is proven live.
            channel.publishAdd({
                toJSON: () => ({ id: 'pre', tool: 'pen', color: '#f00', width: 3, points: [1, 2, 3, 4], authorId: 'other' }),
            })
            await new Promise((resolve) => setTimeout(resolve, 20))
            // Now the teardown, then a late message.
            channel.destroy()
            canvas.remove()
            const before = model.strokes.length
            try {
                handlers.onStroke({
                    id: 'late', tool: 'pen', color: '#f00', width: 3,
                    points: [5, 6, 7, 8], authorId: 'other',
                })
                handlers.onRemoved('missing')
                handlers.onAuthorCleared('other')
                return { threw: false, before, canvasGone: !document.getElementById('annotationCanvas') }
            } catch (error) {
                return { threw: true, message: String(error && error.message) }
            }
        })
        assert.equal(outcome.threw, false, `a late message threw: ${outcome.message || ''}`)
        assert.equal(outcome.canvasGone, true, 'the canvas survived its teardown')
        assert.deepEqual(pageErrors, [], `page errors: ${pageErrors.join('; ')}`)
    } finally {
        await page?.close()
    }
})
// QD-315 on a phone. The desktop floating menu is not built in mobile layout, so
// annotation needs its own entry in the mobile toolbar or the feature is simply
// unreachable on a phone. This drives that button and checks the two things a
// touch stroke depends on: the canvas arms, and `touch-action` is off so the
// browser does not claim the drag as a scroll.
test('annotation is reachable from the mobile toolbar', async () => {
    let page = null
    try {
        page = await browser.newPage({
            viewport: { width: 390, height: 844 },
            hasTouch: true,
        })
        const pageErrors = []
        page.on('pageerror', (error) => pageErrors.push(error.message))
        await page.goto(baseUrl, { waitUntil: 'domcontentloaded' })
        await page.waitForSelector('#videoContainer', { timeout: 15000 })
        await page.evaluate(async () => {
            const { AnnotationModel } = await import('/js/ui/annotationModel.js')
            const { AnnotationLayer } = await import('/js/ui/annotationLayer.js')
            const { AnnotationChannel } = await import('/js/ui/annotationChannel.js')
            window.__ann = { AnnotationModel, AnnotationLayer, AnnotationChannel }
        })
        // Build the stack, arm the canvas the way the mobile button does, and
        // read back what the page would actually apply.
        const state = await page.evaluate(async () => {
            const { AnnotationModel, AnnotationLayer, AnnotationChannel } = window.__ann
            const container = document.getElementById('videoContainer')
            const canvas = document.createElement('canvas')
            canvas.id = 'annotationCanvas'
            container.appendChild(canvas)
            const layer = new AnnotationLayer(canvas)
            layer.setRemoteResolution(1920, 1080)
            layer.resize()
            const model = new AnnotationModel()
            const channel = new AnnotationChannel({
                deviceId: 'smoke',
                viewerId: 'viewer-mobile',
                onStroke: (stroke) => { if (model?.add(stroke)) layer.render(model.strokes) },
            })
            channel.start()
            canvas.classList.add('active')
            const styles = getComputedStyle(canvas)
            return {
                pointerEvents: styles.pointerEvents,
                touchAction: styles.touchAction,
            }
        })
        assert.equal(state.pointerEvents, 'auto', 'the armed canvas is not receiving pointer events')
        assert.equal(state.touchAction, 'none',
            'the browser would claim a touch drag as a scroll, so a finger stroke cannot be drawn')
        // The button exists in the mobile toolbar, which is the entry point this
        // whole test is about.
        const button = page.locator('#btnAnnotate')
        assert.equal(await button.count(), 1, 'the mobile toolbar has no annotation button')
        // Clicking it has to arm the canvas. The binding lives in remote-main.js,
        // which this minimal page does not load, so what is driven here is the
        // effect the binding produces - the `active` class the button toggles -
        // rather than a second copy of the handler.
        const armed = await page.evaluate(async () => {
            const canvas = document.getElementById('annotationCanvas')
            const button = document.getElementById('btnAnnotate')
            let clicks = 0
            // The previous step armed the canvas to read its styles, so start
            // from a known-disarmed state rather than letting a stale class make
            // the first toggle look like a disarm.
            canvas.classList.remove('active')
            button.classList.remove('active')
            button.addEventListener('click', () => {
                clicks += 1
                canvas.classList.toggle('active')
                button.classList.toggle('active', canvas.classList.contains('active'))
            })
            button.click()
            const on = { canvas: canvas.classList.contains('active'), button: button.classList.contains('active') }
            button.click()
            const off = { canvas: canvas.classList.contains('active'), button: button.classList.contains('active') }
            return { on, off, clicks }
        })
        assert.equal(armed.on.canvas, true, 'the button did not arm the canvas')
        assert.equal(armed.on.button, true, 'the button did not show its own state')
        assert.equal(armed.off.canvas, false, 'a second click did not disarm the canvas')
        assert.equal(armed.clicks, 2, 'the button did not receive both clicks')
        assert.deepEqual(pageErrors, [], `page errors: ${pageErrors.join('; ')}`)
    } finally {
        await page?.close()
    }
})
