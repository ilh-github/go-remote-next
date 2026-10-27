import assert from 'node:assert/strict'
import test from 'node:test'

import { OriginalResolutionTracker } from '../js/ui/originalResolution.js'

// Switching monitors replaces the stream, and the new monitor usually has a
// different native resolution. "Original" has to follow the monitor the operator
// is on: offering the first monitor's geometry resizes the desktop away from what
// the video is showing.
test('original follows the monitor being viewed, not the first one seen', () => {
  const tracker = new OriginalResolutionTracker()

  tracker.observe(1920, 1080)
  assert.equal(tracker.width, 1920)
  assert.equal(tracker.height, 1080)

  // The operator switches to a 2560x1440 monitor.
  tracker.observe(2560, 1440)

  assert.equal(tracker.width, 2560, 'original still points at the first monitor')
  assert.equal(tracker.height, 1440)
})

test('the label reports the size that would actually be restored', () => {
  const tracker = new OriginalResolutionTracker()
  tracker.observe(1366, 768)
  tracker.observe(1920, 1080)

  assert.equal(tracker.label(), '1920 × 1080')
})

// A zero or missing size is the host not having told us yet, not a request to
// treat the desktop as 0x0. Honoring it would put "0 × 0" in the menu and make
// restoring it send a client resolution of nothing.
test('a zero or missing size is ignored, not recorded', () => {
  const tracker = new OriginalResolutionTracker()
  tracker.observe(1920, 1080)

  tracker.observe(0, 0)
  tracker.observe(undefined, undefined)
  tracker.observe(-1, 720)

  assert.equal(tracker.width, 1920, 'a zero size overwrote the known one')
  assert.equal(tracker.height, 1080)
})

test('before anything is known the label is empty rather than a size', () => {
  const tracker = new OriginalResolutionTracker()

  assert.equal(tracker.label(), '')
  assert.equal(tracker.width, 0)
  assert.equal(tracker.height, 0)
})
