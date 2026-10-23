import assert from 'node:assert/strict'
import test from 'node:test'

import {
    AnnotationModel,
    Stroke,
    MAX_STROKES,
    ANNOTATION_TOOL_PEN,
    ANNOTATION_TOOL_RECT,
} from '../js/ui/annotationModel.js'

function draw(model, { id, tool, authorId, points }) {
    const stroke = model.begin({ id, tool, authorId, x: points[0], y: points[1] })
    for (let i = 2; i < points.length; i += 2) model.extend(points[i], points[i + 1])
    return model.end()
}

// Points are stored in remote coordinates, which is what lets a viewer at a
// different window size draw the same stroke in the same place on the desktop.
test('a stroke keeps the coordinates it was given', () => {
    const model = new AnnotationModel()
    const stroke = draw(model, {
        id: 's1', authorId: 'viewer-a', points: [10, 20, 30, 40, 50, 60],
    })
    assert.deepEqual(stroke.points, [10, 20, 30, 40, 50, 60])
    assert.equal(model.strokes.length, 1)
})

test('a stroke with a single point is not drawable', () => {
    const stroke = new Stroke({ id: 's', points: [5, 5] })
    assert.equal(stroke.isDrawable, false)
})

// Broadcasting the same stroke twice - a replayed message, a late joiner asking
// for state - must not duplicate it.
test('adding the same stroke twice keeps one', () => {
    const model = new AnnotationModel()
    const stroke = draw(model, { id: 's1', authorId: 'a', points: [1, 2, 3, 4] })
    assert.equal(model.add(stroke.toJSON()), false, 'the stroke it already has was re-added')
    assert.equal(model.strokes.length, 1)
})

test('a stroke round-trips through JSON', () => {
    const original = draw(new AnnotationModel(), {
        id: 's1', tool: ANNOTATION_TOOL_RECT, authorId: 'a', points: [1, 2, 3, 4],
    })
    const restored = Stroke.fromJSON(JSON.parse(JSON.stringify(original.toJSON())))
    assert.deepEqual(restored.points, original.points)
    assert.equal(restored.tool, ANNOTATION_TOOL_RECT)
    assert.equal(restored.authorId, 'a')
})

// An unknown tool from a newer peer must not crash the renderer: it falls back
// to the pen rather than being dropped or throwing.
test('an unknown tool falls back to the pen', () => {
    const restored = Stroke.fromJSON({ id: 's', tool: 'laser', points: [1, 2, 3, 4] })
    assert.equal(restored.tool, ANNOTATION_TOOL_PEN)
})

test('a malformed stroke is refused rather than half-drawn', () => {
    assert.equal(Stroke.fromJSON(null), null)
    assert.equal(Stroke.fromJSON('nonsense'), null)
    assert.equal(Stroke.fromJSON({ id: 's', points: [1] }), null)
})

// One viewer clearing their own annotations must not wipe everybody else's.
test('clearing one author leaves the others intact', () => {
    const model = new AnnotationModel()
    draw(model, { id: 'a1', authorId: 'viewer-a', points: [1, 2, 3, 4] })
    draw(model, { id: 'a2', authorId: 'viewer-a', points: [5, 6, 7, 8] })
    draw(model, { id: 'b1', authorId: 'viewer-b', points: [9, 10, 11, 12] })

    const removed = model.clearAuthor('viewer-a')
    assert.equal(removed, 2)
    assert.equal(model.strokes.length, 1)
    assert.equal(model.strokes[0].authorId, 'viewer-b')
})

// The ceiling has to hold, and it has to drop the oldest rather than refuse the
// newest: the annotation a viewer is looking at right now is the newest one.
test('the stroke ceiling drops the oldest', () => {
    const model = new AnnotationModel({ maxStrokes: 3 })
    for (let i = 0; i < 5; i += 1) {
        draw(model, { id: `s${i}`, authorId: 'a', points: [i, i, i + 1, i + 1] })
    }
    assert.equal(model.strokes.length, 3)
    assert.deepEqual(model.strokes.map((s) => s.id), ['s2', 's3', 's4'])
})

test('the default ceiling is the documented one', () => {
    const model = new AnnotationModel()
    for (let i = 0; i < MAX_STROKES + 5; i += 1) {
        draw(model, { id: `s${i}`, authorId: 'a', points: [i, i, i + 1, i + 1] })
    }
    assert.equal(model.strokes.length, MAX_STROKES)
})

test('only one stroke is in progress at a time', () => {
    const model = new AnnotationModel()
    model.begin({ id: 's1', authorId: 'a', x: 1, y: 1 })
    model.extend(2, 2)
    model.begin({ id: 's2', authorId: 'a', x: 3, y: 3 })
    assert.equal(model.activeStroke.id, 's2')
    model.extend(4, 4)
    assert.equal(model.activeStroke.points.length, 4)
})

test('extending with no stroke in progress is a no-op', () => {
    const model = new AnnotationModel()
    assert.equal(model.extend(1, 2), null)
})

test('removing the stroke in progress clears it', () => {
    const model = new AnnotationModel()
    draw(model, { id: 's1', authorId: 'a', points: [1, 2, 3, 4] })
    assert.equal(model.activeStroke, null, 'a finished stroke is no longer in progress')
    assert.equal(model.remove('missing'), false)
    assert.equal(model.remove('s1'), true)
    assert.equal(model.strokes.length, 0)
})

test('clearing everything empties the model', () => {
    const model = new AnnotationModel()
    draw(model, { id: 'a1', authorId: 'a', points: [1, 2, 3, 4] })
    draw(model, { id: 'b1', authorId: 'b', points: [5, 6, 7, 8] })
    model.clear()
    assert.equal(model.strokes.length, 0)
    assert.equal(model.activeStroke, null)
})
