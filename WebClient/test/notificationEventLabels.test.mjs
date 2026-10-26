import assert from 'node:assert/strict'
import test from 'node:test'

// QD-607's connect_prompt publishes device.connect_prompted, and a client that
// receives it has to be able to say what it is. The label map was inline in the
// Vue file, so an unlisted event type fell through to the raw wire name - which
// is a UI that shows "device.connect_prompted" to an operator instead of a
// sentence. It lives in a module now so the coverage is a test rather than a
// hope.

const { eventLabelKeys, eventLabelKey } = await import('../js/notifications/eventLabels.js')

test('the connect prompt has a label', () => {
  assert.equal(eventLabelKeys['device.connect_prompted'], 'connectPrompted')
})

// The fallback matters as much as the mapping: a type this build has never heard
// of must still render something, or a new server event blanks the row.
test('an unknown event falls back to the raw type', () => {
  assert.equal(eventLabelKey('device.something_new'), 'device.something_new')
  assert.equal(eventLabelKey(''), '')
  assert.equal(eventLabelKey(undefined), '')
})

// Every mapped key has to be a non-empty label: a mapping to '' would render an
// empty cell, which is worse than the raw name.
test('no label maps to an empty string', () => {
  for (const [type, key] of Object.entries(eventLabelKeys)) {
    assert.ok(key && key.length > 0, `${type} maps to an empty label`)
  }
})

// The existing labels must not be lost in the move.
test('the pre-existing labels survive', () => {
  assert.equal(eventLabelKeys['scheduled_task.failed'], 'scheduledTaskFailed')
  assert.equal(eventLabelKeys['device.metric.alert'], 'metricAlert')
  assert.equal(eventLabelKeys['device.access.denied'], 'accessDenied')
})
