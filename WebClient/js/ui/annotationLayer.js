// Draws annotation strokes over the remote video.
//
// The layer is a separate canvas positioned over the video element rather than
// drawing into the video's own canvas: the video is owned by the decoder and is
// being redrawn every frame, so anything drawn there would be wiped on the next
// one. A sibling canvas keeps annotations stable across frames.
//
// Everything the model stores is in *remote* coordinates, so this layer is where
// the remote-to-local scale is applied. That is what lets a viewer at one window
// size and another at a different size both put a stroke in the same place on
// the captured desktop.

const STROKE_STYLES = {
    pen: { alpha: 1, lineJoin: 'round', lineCap: 'round', shadow: false },
    highlighter: { alpha: 0.35, lineJoin: 'round', lineCap: 'butt', shadow: false },
    arrow: { alpha: 1, lineJoin: 'round', lineCap: 'round', shadow: false },
    rect: { alpha: 1, lineJoin: 'miter', lineCap: 'butt', shadow: false },
}

/** Arrowhead size as a multiple of the stroke width. */
const ARROW_HEAD_RATIO = 3

export class AnnotationLayer {
    /**
     * @param {HTMLCanvasElement} canvas  overlay canvas, sized to its container
     */
    constructor(canvas) {
        this.canvas = canvas
        this.ctx = canvas ? canvas.getContext('2d') : null
        this._remoteWidth = 0
        this._remoteHeight = 0
    }

    /**
     * Records the captured desktop's resolution. Without it a stroke cannot be
     * placed: there is nothing to scale from.
     */
    setRemoteResolution(width, height) {
        this._remoteWidth = Number(width) || 0
        this._remoteHeight = Number(height) || 0
    }

    /** Sizes the backing store to the element's box, so lines are not blurry. */
    resize() {
        if (!this.canvas || !this.ctx) return
        const ratio = globalThis.devicePixelRatio || 1
        const width = this.canvas.clientWidth || 0
        const height = this.canvas.clientHeight || 0
        if (width === 0 || height === 0) return
        this.canvas.width = Math.round(width * ratio)
        this.canvas.height = Math.round(height * ratio)
        this.ctx.setTransform(ratio, 0, 0, ratio, 0, 0)
    }

    /**
     * Converts a point in remote coordinates to canvas pixels.
     *
     * The scale is uniform and derived from whichever axis is the binding
     * constraint, so a stroke keeps its aspect ratio on a differently shaped
     * window instead of being stretched to fill it.
     */
    toLocal(x, y) {
        if (!this._remoteWidth || !this._remoteHeight) return null
        const width = this.canvas?.clientWidth || 0
        const height = this.canvas?.clientHeight || 0
        if (!width || !height) return null
        const scale = Math.min(width / this._remoteWidth, height / this._remoteHeight)
        const offsetX = (width - this._remoteWidth * scale) / 2
        const offsetY = (height - this._remoteHeight * scale) / 2
        return { x: offsetX + x * scale, y: offsetY + y * scale, scale }
    }

    /** Converts a canvas point back to remote coordinates. */
    toRemote(x, y) {
        if (!this._remoteWidth || !this._remoteHeight) return null
        const width = this.canvas?.clientWidth || 0
        const height = this.canvas?.clientHeight || 0
        if (!width || !height) return null
        const scale = Math.min(width / this._remoteWidth, height / this._remoteHeight)
        const offsetX = (width - this._remoteWidth * scale) / 2
        const offsetY = (height - this._remoteHeight * scale) / 2
        return {
            x: Math.round((x - offsetX) / scale),
            y: Math.round((y - offsetY) / scale),
        }
    }

    /** Redraws every stroke. Cheap enough to call on each pointer move. */
    render(strokes) {
        if (!this.ctx) return
        const width = this.canvas?.clientWidth || 0
        const height = this.canvas?.clientHeight || 0
        this.ctx.clearRect(0, 0, width, height)
        if (!Array.isArray(strokes)) return
        for (const stroke of strokes) this._drawStroke(stroke)
    }

    _drawStroke(stroke) {
        if (!stroke || !stroke.isDrawable) return
        const style = STROKE_STYLES[stroke.tool] || STROKE_STYLES.pen
        const points = []
        for (let i = 0; i + 1 < stroke.points.length; i += 2) {
            const local = this.toLocal(stroke.points[i], stroke.points[i + 1])
            if (local) points.push(local)
        }
        if (points.length === 0) return

        this.ctx.save()
        this.ctx.globalAlpha = style.alpha
        this.ctx.strokeStyle = stroke.color
        this.ctx.fillStyle = stroke.color
        this.ctx.lineWidth = Math.max(1, stroke.width)
        this.ctx.lineJoin = style.lineJoin
        this.ctx.lineCap = style.lineCap

        if (stroke.tool === 'rect') {
            const first = points[0]
            const last = points[points.length - 1]
            this.ctx.strokeRect(first.x, first.y, last.x - first.x, last.y - first.y)
        } else if (stroke.tool === 'arrow') {
            this._strokePath(points)
            const from = points[points.length - 2] || points[0]
            const to = points[points.length - 1]
            this._drawArrowHead(from, to, stroke.width)
        } else {
            this._strokePath(points)
        }

        this.ctx.restore()
    }

    _strokePath(points) {
        this.ctx.beginPath()
        this.ctx.moveTo(points[0].x, points[0].y)
        if (points.length === 1) {
            // A single point would draw nothing as a path, so it becomes a dot -
            // which is what a tap with the pen looks like.
            this.ctx.lineTo(points[0].x + 0.01, points[0].y)
        }
        for (let i = 1; i < points.length; i += 1) {
            this.ctx.lineTo(points[i].x, points[i].y)
        }
        this.ctx.stroke()
    }

    _drawArrowHead(from, to, width) {
        const angle = Math.atan2(to.y - from.y, to.x - from.x)
        const size = Math.max(6, width * ARROW_HEAD_RATIO)
        this.ctx.beginPath()
        this.ctx.moveTo(to.x, to.y)
        this.ctx.lineTo(
            to.x - size * Math.cos(angle - Math.PI / 6),
            to.y - size * Math.sin(angle - Math.PI / 6),
        )
        this.ctx.lineTo(
            to.x - size * Math.cos(angle + Math.PI / 6),
            to.y - size * Math.sin(angle + Math.PI / 6),
        )
        this.ctx.closePath()
        this.ctx.fill()
    }

    clear() {
        if (!this.ctx) return
        const width = this.canvas?.clientWidth || 0
        const height = this.canvas?.clientHeight || 0
        this.ctx.clearRect(0, 0, width, height)
    }
}
