import assert from 'node:assert/strict'
import test from 'node:test'

// The remote cursor is drawn from raw host pixels, and the host sends it in
// whatever bit depth its platform produces. The renderer has to know which
// depth it is looking at, because guessing turns the cursor into noise the
// operator cannot aim with.

class FakeCanvas {
  constructor() {
    this.width = 0
    this.height = 0
    this.imageData = null
    this.put = null
  }
  getContext() {
    const canvas = this
    return {
      createImageData(w, h) {
        canvas.imageData = { width: w, height: h, data: new Uint8ClampedArray(w * h * 4) }
        return canvas.imageData
      },
      putImageData(imageData) { canvas.put = imageData },
    }
  }
  toDataURL() { return 'data:image/png;base64,STUB' }
}

function makeRenderer() {
  const container = { style: {} }
  const canvases = []
  globalThis.document = {
    createElement: (tag) => {
      assert.equal(tag, 'canvas')
      const canvas = new FakeCanvas()
      canvases.push(canvas)
      return canvas
    },
  }
  return { container, canvases, renderer: null }
}

// A 32-bit BGRA cursor is the case the renderer was written for. It must swap
// the red and blue channels, because the host sends blue first and the canvas
// wants red first.
test('a 32-bit BGRA cursor has its channels swapped', async () => {
  const { CursorRenderer } = await import('../js/ui/cursor-renderer.js')
  const { container, canvases, renderer } = makeRenderer()
  const r = new CursorRenderer(container)

  const width = 2
  const height = 1
  const data = new Uint8Array([
    10, 20, 30, 255,   // BGRA: B=10 G=20 R=30 A=255
    40, 50, 60, 128,
  ])

  r.updateCursor({ width, height, hotspotX: 0, hotspotY: 0, data })

  const canvas = canvases[0]
  const pixels = canvas.put.data
  assert.equal(pixels[0], 30, 'red channel was not taken from the third byte')
  assert.equal(pixels[1], 20, 'green channel moved')
  assert.equal(pixels[2], 10, 'blue channel was not taken from the first byte')
  assert.equal(pixels[3], 255, 'alpha channel moved')

  r.destroy()
})

// A cursor whose data is shorter than width*height*4 must not silently drop
// the pixels it does have, and must not read past the end of the buffer: the
// tail of the image would be transparent, which reads as "the cursor
// disappeared" rather than "the cursor is partly broken".
test('a truncated cursor leaves the missing pixels transparent', async () => {
  const { CursorRenderer } = await import('../js/ui/cursor-renderer.js')
  const { container, canvases, renderer } = makeRenderer()
  const r = new CursorRenderer(container)

  const width = 2
  const height = 1
  const data = new Uint8Array([10, 20, 30, 255])  // only the first pixel

  r.updateCursor({ width, height, hotspotX: 0, hotspotY: 0, data })

  const pixels = canvases[0].put.data
  assert.equal(pixels[0], 30)
  assert.equal(pixels[3], 255)
  assert.equal(pixels[7], 0, 'the missing pixel was filled with something')

  r.destroy()
})

// The hotspot is where the click lands, so it has to stay inside the cursor.
// A hotspot the host reports outside the bitmap would offset every click by
// that much.
test('the hotspot is clamped inside the cursor bitmap', async () => {
  const { CursorRenderer } = await import('../js/ui/cursor-renderer.js')
  const { container, canvases, renderer } = makeRenderer()
  const r = new CursorRenderer(container)

  r.updateCursor({
    width: 16, height: 16,
    hotspotX: 99, hotspotY: -5,
    data: new Uint8Array(16 * 16 * 4).fill(255),
  })

  assert.match(
    container.style.cursor,
    /url\(data:image\/png;base64,STUB\) 15 0, auto/,
    `unexpected cursor style: ${container.style.cursor}`,
  )

  r.destroy()
})

// An empty or zero-sized shape is the host saying "no custom cursor". It must
// be ignored rather than crashing on a 0x0 canvas, which would take the whole
// video panel down with it.
test('an empty cursor shape is ignored', async () => {
  const { CursorRenderer } = await import('../js/ui/cursor-renderer.js')
  const { container, canvases, renderer } = makeRenderer()
  const r = new CursorRenderer(container)

  assert.doesNotThrow(() => r.updateCursor(null))
  assert.doesNotThrow(() => r.updateCursor({ width: 0, height: 0, data: new Uint8Array() }))
  assert.equal(canvases.length, 0, 'a canvas was created for a cursor that does not exist')

  r.destroy()
})
