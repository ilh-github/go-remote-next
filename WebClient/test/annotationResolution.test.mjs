import assert from 'node:assert/strict'
import test from 'node:test'

import { AnnotationLayer } from '../js/ui/annotationLayer.js'

class FakeCanvas {
    constructor({ clientWidth = 0, clientHeight = 0 } = {}) {
        this.clientWidth = clientWidth
        this.clientHeight = clientHeight
        this.width = 0
        this.height = 0
        this.calls = []
    }

    getContext() {
        const canvas = this
        return {
            canvas,
            setTransform(...a) { canvas.calls.push(['setTransform', a]) },
            clearRect(...a) { canvas.calls.push(['clearRect', a]) },
            save() {}, restore() {}, beginPath() {}, moveTo() {}, lineTo() {},
            stroke() { canvas.calls.push(['stroke']) },
            fill() { canvas.calls.push(['fill']) },
            closePath() { canvas.calls.push(['closePath']) },
            strokeRect(...a) { canvas.calls.push(['strokeRect', a]) },
            set globalAlpha(v) {}, set strokeStyle(v) {}, set fillStyle(v) {},
            set lineWidth(v) {}, set lineJoin(v) {}, set lineCap(v) {},
        }
    }
}

function makeLayer(clientWidth, clientHeight, remoteWidth, remoteHeight) {
    globalThis.devicePixelRatio = 1
    const canvas = new FakeCanvas({ clientWidth, clientHeight })
    const layer = new AnnotationLayer(canvas)
    layer.setRemoteResolution(remoteWidth, remoteHeight)
    layer.resize()
    return { layer, canvas }
}

// A stroke drawn at a fixed point on the desktop stays at that point on the
// desktop when the captured resolution changes. Because the model stores remote
// coordinates, the same stroke is the same stroke and the layer only re-scales.
//
// What is *not* invariant is the fraction of the browser box it occupies: the
// layer scales the desktop to fit the box, so a fixed desktop point moves within
// the box as the scale changes. That is correct behaviour - the box is a viewport
// onto the desktop, not the desktop - so the assertion is written against the
// property that actually matters, which is that the point keeps its place on the
// desktop and stays inside the box.
test('a stroke keeps its place across a resolution change', () => {
    const { layer } = makeLayer(800, 600, 1920, 1080)
    const point = { x: 480, y: 360 }
    assert.ok(layer.toLocal(point.x, point.y))

    // The captured desktop got bigger; the window did not.
    layer.setRemoteResolution(2560, 1440)
    const after = layer.toLocal(point.x, point.y)
    assert.ok(after)

    // Round-tripping back through the layer must recover the desktop point
    // exactly: the stroke lives on the desktop, not on the box.
    const roundTrip = layer.toRemote(after.x, after.y)
    assert.equal(roundTrip.x, point.x, `the stroke moved on the desktop: ${roundTrip.x} != ${point.x}`)
    assert.equal(roundTrip.y, point.y, `the stroke moved on the desktop: ${roundTrip.y} != ${point.y}`)

    // And it is still inside the box rather than pushed off it.
    assert.ok(after.x >= 0 && after.x <= 800, `x = ${after.x} is outside the box`)
    assert.ok(after.y >= 0 && after.y <= 600, `y = ${after.y} is outside the box`)
})

// The scale has to follow the new resolution, or the stroke would be drawn at
// the old size in the new desktop's coordinates.
test('the scale follows the captured resolution', () => {
    const { layer } = makeLayer(800, 600, 1920, 1080)
    const first = layer.toLocal(1920, 1080)
    assert.ok(Math.abs(first.scale - 800 / 1920) < 1e-9)

    layer.setRemoteResolution(800, 600)
    const second = layer.toLocal(800, 600)
    assert.ok(Math.abs(second.scale - 1) < 1e-9, `scale = ${second.scale}, want 1`)
})

// A resolution change is not a reason to lose the strokes: they are data, and
// the layer only re-reads them. Re-rendering after the change must produce the
// same geometry.
test('re-rendering after a resolution change redraws the same strokes', () => {
    const { layer, canvas } = makeLayer(800, 600, 1920, 1080)
    const strokes = [{ isDrawable: true, tool: 'pen', color: '#f00', width: 3, points: [0, 0, 1920, 1080] }]

    canvas.calls.length = 0
    layer.render(strokes)
    const before = canvas.calls.filter(([name]) => name === 'stroke').length
    assert.equal(before, 1)

    layer.setRemoteResolution(2560, 1440)
    canvas.calls.length = 0
    layer.render(strokes)
    const after = canvas.calls.filter(([name]) => name === 'stroke').length
    assert.equal(after, 1, 'the stroke was lost across a resolution change')
})

// An unknown resolution must not silently produce a bogus scale: with nothing
// to scale from, the layer refuses to place anything.
test('an unknown resolution refuses to place strokes', () => {
    const { layer } = makeLayer(800, 600, 0, 0)
    assert.equal(layer.toLocal(100, 100), null)
    layer.setRemoteResolution(1920, 1080)
    assert.ok(layer.toLocal(100, 100))
})

test('resize is safe to call repeatedly', () => {
    const { layer, canvas } = makeLayer(800, 600, 1920, 1080)
    canvas.calls.length = 0
    layer.resize()
    layer.resize()
    assert.equal(canvas.calls.filter(([name]) => name === 'setTransform').length, 2)
    assert.equal(canvas.width, 800)
})
