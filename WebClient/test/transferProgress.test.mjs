import assert from 'node:assert/strict'
import test from 'node:test'

import { TransferProgress, reconcileTotal } from '../js/transferProgress.js'

// The host announces the total at the start and is free to omit it afterwards.
// A progress event without one must not redraw a transfer that had reached 80%
// back at 0%.
test('a progress event without a total keeps the known one', () => {
  const progress = new TransferProgress()
  progress.update(0, 1000)
  progress.update(800, 1000)

  progress.update(900, undefined)

  assert.equal(progress.totalBytes, 1000, 'the known total was discarded')
  assert.equal(progress.percent(), 90, 'the bar was reset by an event that omitted the total')
})

test('a later event that does carry a total updates it', () => {
  const progress = new TransferProgress()
  progress.update(0, 1000)

  progress.update(500, 2000)

  assert.equal(progress.totalBytes, 2000)
  assert.equal(progress.percent(), 25)
})

// A zero total on an event means "this event did not say", not "the transfer
// became empty" - otherwise a mid-transfer event would collapse the bar.
test('a zero total on a later event does not collapse the bar', () => {
  const progress = new TransferProgress()
  progress.update(0, 1000)
  progress.update(600, 1000)

  progress.update(700, 0)

  assert.equal(progress.totalBytes, 1000)
  assert.equal(progress.percent(), 70)
})

test('a zero-byte transfer announced at the start is complete', () => {
  const progress = new TransferProgress()
  progress.announceTotal(0)

  assert.equal(progress.percent(), 100)
  assert.equal(progress.isComplete, true)
})

// The start of the transfer is where a zero total is a statement about the file.
// A zero arriving later is the event declining to say, and must not turn a
// half-finished transfer into a "complete" one.
test('a zero on a later event is not read as an empty transfer', () => {
  const progress = new TransferProgress()
  progress.announceTotal(1000)
  progress.update(400, 1000)

  progress.update(500, 0)

  assert.equal(progress.percent(), 50, 'a mid-transfer zero was read as empty')
  assert.equal(progress.isComplete, false)
})

test('a transfer whose total is still unknown reports zero', () => {
  const progress = new TransferProgress()
  progress.update(500, undefined)

  assert.equal(progress.totalBytes, 0)
  assert.equal(progress.percent(), 0)
})

test('the percentage never leaves 0-100', () => {
  const progress = new TransferProgress()
  progress.update(0, 100)

  // A host reporting more bytes than it announced must not overflow the bar.
  progress.update(250, 100)
  assert.equal(progress.percent(), 100)

  const negative = new TransferProgress()
  negative.update(-50, 100)
  assert.equal(negative.percent(), 0)
})

test('isComplete only once every byte has crossed', () => {
  const progress = new TransferProgress()
  progress.announceTotal(100)
  assert.equal(progress.isComplete, false)

  progress.update(99, 100)
  assert.equal(progress.isComplete, false)

  progress.update(100, 100)
  assert.equal(progress.isComplete, true)
})

test('a transfer whose total was never announced is not complete', () => {
  const progress = new TransferProgress()
  progress.update(500, undefined)

  assert.equal(progress.isComplete, false)
})

test('reconcileTotal keeps the larger of what is known', () => {
  assert.equal(reconcileTotal(1000, 2000), 2000)
  assert.equal(reconcileTotal(1000, 0), 1000)
  assert.equal(reconcileTotal(1000, undefined), 1000)
  assert.equal(reconcileTotal(0, 0), 0)
  assert.equal(reconcileTotal(0, 500), 500)
})
