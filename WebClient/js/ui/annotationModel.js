// Annotation strokes for a remote-control view.
//
// The model is deliberately separate from the canvas: strokes are plain data,
// which is what makes them shareable (a BroadcastChannel message, a replay) and
// testable without a DOM. The renderer in `annotation-layer.js` only reads this.
//
// A stroke is a polyline in *remote* coordinates - the resolution the captured
// desktop is in - not in canvas pixels. A viewer at a different window size
// must draw the same stroke in the same place on the desktop, so scaling belongs
// to the renderer, not to the stored data.

export const ANNOTATION_TOOL_PEN = 'pen'
export const ANNOTATION_TOOL_HIGHLIGHTER = 'highlighter'
export const ANNOTATION_TOOL_ARROW = 'arrow'
export const ANNOTATION_TOOL_RECT = 'rect'

const TOOLS = [
    ANNOTATION_TOOL_PEN,
    ANNOTATION_TOOL_HIGHLIGHTER,
    ANNOTATION_TOOL_ARROW,
    ANNOTATION_TOOL_RECT,
]

/** Hard ceiling on live strokes, so a stuck sender cannot grow without bound. */
export const MAX_STROKES = 200

/**
 * One drawn stroke.
 *
 * `id` is what makes a stroke removable and what a remote peer uses to tell a
 * new stroke from one it already has. `authorId` groups strokes by who drew
 * them, which is what a "clear mine" action needs.
 */
export class Stroke {
    constructor({
        id,
        tool = ANNOTATION_TOOL_PEN,
        color = '#ff3b30',
        width = 3,
        points = [],
        authorId = '',
    } = {}) {
        this.id = String(id || '')
        this.tool = tool
        this.color = color
        this.width = width
        // Flattened [x0, y0, x1, y1, ...] in remote coordinates: a JSON array
        // travels over BroadcastChannel without conversion, which a list of
        // {x, y} objects would not.
        this.points = Array.isArray(points) ? points.map(Number) : []
        this.authorId = String(authorId || '')
    }

    /** Whether the stroke has enough geometry to draw. */
    get isDrawable() {
        // Every tool needs at least two points: a rect and an arrow are defined
        // by their endpoints, and a freehand pen with one point renders nothing.
        return this.points.length >= 4
    }

    toJSON() {
        return {
            id: this.id,
            tool: this.tool,
            color: this.color,
            width: this.width,
            points: this.points.slice(),
            authorId: this.authorId,
        }
    }

    static fromJSON(value) {
        if (!value || typeof value !== 'object') return null
        const tool = TOOLS.includes(value.tool) ? value.tool : ANNOTATION_TOOL_PEN
        const stroke = new Stroke({
            id: value.id,
            tool,
            color: typeof value.color === 'string' ? value.color : '#ff3b30',
            width: Number.isFinite(value.width) ? value.width : 3,
            points: value.points,
            authorId: value.authorId,
        })
        return stroke.isDrawable ? stroke : null
    }
}

/**
 * The live set of strokes on one view, plus the rules that keep it bounded.
 */
export class AnnotationModel {
    constructor({ maxStrokes = MAX_STROKES, now = () => Date.now() } = {}) {
        this._maxStrokes = maxStrokes
        this._now = now
        this._strokes = new Map()
        this._order = []
        this._activeId = null
    }

    get strokes() {
        return this._order.map((id) => this._strokes.get(id)).filter(Boolean)
    }

    get activeStroke() {
        return this._activeId ? this._strokes.get(this._activeId) || null : null
    }

    /**
     * Begins a stroke. Returns the new stroke so the caller can append points
     * to it while the pointer is down.
     */
    begin({ id, tool, color, width, x, y, authorId }) {
        const stroke = new Stroke({ id, tool, color, width, authorId, points: [x, y] })
        this._strokes.set(stroke.id, stroke)
        this._order.push(stroke.id)
        this._activeId = stroke.id
        this._trim()
        return stroke
    }

    /** Extends the stroke in progress. Points outside the model are ignored. */
    extend(x, y) {
        const stroke = this.activeStroke
        if (!stroke) return null
        stroke.points.push(Number(x), Number(y))
        return stroke
    }

    /** Ends the stroke in progress and returns it, or null if none was active. */
    end() {
        const stroke = this.activeStroke
        this._activeId = null
        return stroke
    }

    /**
     * Adds a finished stroke, e.g. one received from another viewer. A stroke
     * whose id is already present is ignored, which is what makes replaying a
     * broadcast idempotent.
     */
    add(stroke) {
        if (!stroke || !stroke.id || this._strokes.has(stroke.id)) return false
        this._strokes.set(stroke.id, stroke)
        this._order.push(stroke.id)
        this._trim()
        return true
    }

    remove(id) {
        if (!this._strokes.delete(id)) return false
        this._order = this._order.filter((entry) => entry !== id)
        if (this._activeId === id) this._activeId = null
        return true
    }

    /** Removes every stroke by one author, leaving everybody else's intact. */
    clearAuthor(authorId) {
        const mine = this.strokes.filter((stroke) => stroke.authorId === authorId)
        for (const stroke of mine) this.remove(stroke.id)
        return mine.length
    }

    clear() {
        this._strokes.clear()
        this._order = []
        this._activeId = null
    }

    /**
     * Drops the oldest strokes once the ceiling is reached.
     *
     * Bounding is what keeps a long session from growing without limit, and
     * dropping the oldest rather than refusing new ones is what keeps the
     * newest annotation - the one the viewer is looking at right now - visible.
     */
    _trim() {
        while (this._order.length > this._maxStrokes) {
            const oldest = this._order.shift()
            this._strokes.delete(oldest)
        }
    }
}
