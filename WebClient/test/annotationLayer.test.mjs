import assert from 'node:assert/strict'
import test from 'node:test'

import { AnnotationLayer } from '../js/ui/annotationLayer.js'
import { AnnotationModel } from '../js/ui/annotationModel.js'

// A canvas stand-in with the box properties the layer reads. `width`/`height`
// are the backing store the layer writes; `clientWidth`/`clientHeight` are the
// CSS box it draws into, which is what the coordinate maths uses.
class FakeCanvas {
    constructor({ clientWidth = 0, clientHeight = 0, devicePixelRatio = 1 } = {}) {
        this.clientWidth = clientWidth
        this.clientHeight = clientHeight
        this.width = 0
        this.height = 0
        this.calls = []
        this._ratio = devicePixelRatio
    }

    getContext() {
        const canvas = this
        return {
            canvas,
            setTransform(...args) { canvas.calls.push(['setTransform', args]) },
            clearRect(...args) { canvas.calls.push(['clearRect', args]) },
            save() { canvas.calls.push(['save']) },
            restore() { canvas.calls.push(['restore']) },
            beginPath() { canvas.calls.push(['beginPath']) },
            moveTo(...args) { canvas.calls.push(['moveTo', args]) },
            lineTo(...args) { canvas.calls.push(['lineTo', args]) },
            stroke() { canvas.calls.push(['stroke']) },
            fill() { canvas.calls.push(['fill']) },
            closePath() { canvas.calls.push(['closePath']) },
            strokeRect(...args) { canvas.calls.push(['strokeRect', args]) },
            set globalAlpha(value) { canvas.calls.push(['globalAlpha', value]) },
            set strokeStyle(value) { canvas.calls.push(['strokeStyle', value]) },
            set fillStyle(value) { canvas.calls.push(['fillStyle', value]) },
            set lineWidth(value) { canvas.calls.push(['lineWidth', value]) },
            set lineJoin(value) { canvas.calls.push(['lineJoin', value]) },
            set lineCap(value) { canvas.calls.push(['lineCap', value]) },
        }
    }
}

function makeLayer({ clientWidth, clientHeight, remoteWidth, remoteHeight, devicePixelRatio = 1 }) {
    const canvas = new FakeCanvas({ clientWidth, clientHeight, devicePixelRatio })
    globalThis.devicePixelRatio = devicePixelRatio
    const layer = new AnnotationLayer(canvas)
    layer.setRemoteResolution(remoteWidth, remoteHeight)
    layer.resize()
    return { layer, canvas }
}

// A stroke drawn at the same remote point must land at the same place on the
// desktop no matter how large the viewer's window is. That is the whole reason
// the model stores remote coordinates.
test('the same remote point maps to the same relative place at any window size', () => {
    const small = makeLayer({ clientWidth: 800, clientHeight: 600, remoteWidth: 1920, remoteHeight: 1080 })
    const large = makeLayer({ clientWidth: 1600, clientHeight: 1200, remoteWidth: 1920, remoteHeight: 1080 })

    const point = { x: 960, y: 540 }
    const a = small.layer.toLocal(point.x, point.y)
    const b = large.layer.toLocal(point.x, point.y)

    assert.ok(a && b)
    // Halfway across the remote desktop is halfway across the canvas in both.
    assert.ok(Math.abs(a.x / 800 - b.x / 1600) < 1e-9)
    assert.ok(Math.abs(a.y / 600 - b.y / 1200) < 1e-9)
})

// The scale is uniform, so a circle drawn on the desktop stays a circle rather
// than being stretched to fill a differently shaped window.
test('the scale is uniform and letterboxes the remainder', () => {
    const { layer, canvas } = makeLayer({
        clientWidth: 1000, clientHeight: 1000, remoteWidth: 1920, remoteHeight: 1080,
    })
    const topLeft = layer.toLocal(0, 0)
    const bottomRight = layer.toLocal(1920, 1080)

    assert.ok(Math.abs(topLeft.x) < 1e-9, 'the desktop is wider than the box, so it fills the width')
    assert.ok(topLeft.y > 0, 'the desktop is shorter than the box, so it is centred vertically')
    assert.ok(Math.abs(bottomRight.x - 1000) < 1e-9)
    assert.ok(bottomRight.y < 1000)
})

test('toRemote inverts toLocal', () => {
    const { layer } = makeLayer({
        clientWidth: 900, clientHeight: 700, remoteWidth: 1600, remoteHeight: 900,
    })
    const local = layer.toLocal(400, 300)
    const back = layer.toRemote(local.x, local.y)
    assert.equal(back.x, 400)
    assert.equal(back.y, 300)
})

// Without the captured resolution there is nothing to scale from, and guessing
// would put every stroke in the wrong place.
test('coordinates are unavailable until the remote resolution is known', () => {
    const canvas = new FakeCanvas({ clientWidth: 800, clientHeight: 600 })
    globalThis.devicePixelRatio = 1
    const layer = new AnnotationLayer(canvas)
    layer.resize()
    assert.equal(layer.toLocal(10, 10), null)
    assert.equal(layer.toRemote(10, 10), null)
})

// The backing store follows the CSS box times the pixel ratio, so a line is not
// blurry on a HiDPI display.
test('resize scales the backing store by the device pixel ratio', () => {
    const { canvas } = makeLayer({
        clientWidth: 800, clientHeight: 600, remoteWidth: 1920, remoteHeight: 1080, devicePixelRatio: 2,
    })
    assert.equal(canvas.width, 1600)
    assert.equal(canvas.height, 1200)
    assert.deepEqual(canvas.calls[0], ['setTransform', [2, 0, 0, 2, 0, 0]])
})

test('render clears before drawing', () => {
    const { layer, canvas } = makeLayer({
        clientWidth: 800, clientHeight: 600, remoteWidth: 1920, remoteHeight: 1080,
    })
    canvas.calls.length = 0
    layer.render([])
    assert.deepEqual(canvas.calls[0], ['clearRect', [0, 0, 800, 600]])
})

// A stroke that is not drawable must be skipped rather than producing a bogus
// path - the model already refuses them, and this is the second line of defence.
test('an undrawable stroke is skipped', () => {
    const { layer, canvas } = makeLayer({
        clientWidth: 800, clientHeight: 600, remoteWidth: 1920, remoteHeight: 1080,
    })
    canvas.calls.length = 0
    layer.render([{ isDrawable: false, points: [1, 2] }])
    assert.equal(canvas.calls.filter(([name]) => name === 'stroke').length, 0)
})

// The highlighter is semi-transparent, which is what makes it a highlighter
// rather than a second pen.
test('the highlighter draws translucent', () => {
    const { layer, canvas } = makeLayer({
        clientWidth: 800, clientHeight: 600, remoteWidth: 1920, remoteHeight: 1080,
    })
    canvas.calls.length = 0
    layer.render([{ isDrawable: true, tool: 'highlighter', color: '#ff0', width: 12, points: [0, 0, 100, 100] }])
    const alphas = canvas.calls.filter(([name]) => name === 'globalAlpha').map(([, value]) => value)
    assert.ok(alphas.length > 0)
    assert.ok(alphas.every((alpha) => alpha > 0 && alpha < 1), `alpha was ${alphas.join(',')}`)
})

test('a rect is stroked as a rectangle rather than a path', () => {
    const { layer, canvas } = makeLayer({
        clientWidth: 800, clientHeight: 600, remoteWidth: 1920, remoteHeight: 1080,
    })
    canvas.calls.length = 0
    layer.render([{ isDrawable: true, tool: 'rect', color: '#f00', width: 3, points: [0, 0, 1920, 1080] }])
    const rects = canvas.calls.filter(([name]) => name === 'strokeRect')
    assert.equal(rects.length, 1, 'a rect must be drawn with strokeRect')
    const [x, y, w, h] = rects[0][1]
    // 1920x1080 into an 800x600 box scales to 800x450 and is centred, so the
    // top edge sits at the 75px letterbox rather than at zero.
    assert.ok(Math.abs(x) < 1e-9)
    assert.equal(y, 75)
    assert.ok(Math.abs(w - 800) < 1e-9)
    assert.ok(Math.abs(h - 450) < 1e-9)
})

// An arrowhead is filled, not stroked: a stroked triangle outline reads as a
// chevron rather than an arrow.
test('an arrow draws a filled head', () => {
    const { layer, canvas } = makeLayer({
        clientWidth: 800, clientHeight: 600, remoteWidth: 1920, remoteHeight: 1080,
    })
    canvas.calls.length = 0
    layer.render([{ isDrawable: true, tool: 'arrow', color: '#f00', width: 4, points: [0, 0, 1920, 1080] }])
    assert.equal(canvas.calls.filter(([name]) => name === 'stroke').length, 1, 'the shaft is stroked')
    assert.equal(canvas.calls.filter(([name]) => name === 'fill').length, 1, 'the head is filled')
    assert.equal(canvas.calls.filter(([name]) => name === 'closePath').length, 1)
})

test('a full round trip through the model renders', () => {
    const { layer, canvas } = makeLayer({
        clientWidth: 800, clientHeight: 600, remoteWidth: 1920, remoteHeight: 1080,
    })
    const model = new AnnotationModel()
    const stroke = model.begin({ id: 's1', authorId: 'a', x: 100, y: 100 })
    model.extend(500, 400)
    model.extend(900, 300)
    model.end()
    assert.equal(stroke.points.length, 6)

    canvas.calls.length = 0
    layer.render(model.strokes)
    assert.equal(canvas.calls.filter(([name]) => name === 'stroke').length, 1)
})

test('clear empties the canvas', () => {
    const { layer, canvas } = makeLayer({
        clientWidth: 800, clientHeight: 600, remoteWidth: 1920, remoteHeight: 1080,
    })
    canvas.calls.length = 0
    layer.clear()
    assert.deepEqual(canvas.calls, [['clearRect', [0, 0, 800, 600]]])
})
