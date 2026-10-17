import assert from 'node:assert/strict'
import test from 'node:test'

import { WallAudioSelector } from '../js/ui/wall-audio.js'

test('a new wall exposes no audio owner', () => {
  const selector = new WallAudioSelector()

  assert.equal(selector.activeId, '')
})

test('selecting a tile makes it the only audio owner', () => {
  const selector = new WallAudioSelector()

  const previous = selector.setActive('device-a')

  assert.equal(previous, '')
  assert.equal(selector.activeId, 'device-a')
})

test('switching tiles returns the previous owner for revocation', () => {
  const selector = new WallAudioSelector()
  selector.setActive('device-a')

  const previous = selector.setActive('device-b')

  assert.equal(previous, 'device-a')
  assert.equal(selector.activeId, 'device-b')
})

test('selecting the current audio owner keeps it active without revocation', () => {
  const selector = new WallAudioSelector()
  selector.setActive('device-a')

  const previous = selector.setActive('device-a')

  assert.equal(previous, '')
  assert.equal(selector.activeId, 'device-a')
})

test('clearing a matching tile stops its audio', () => {
  const selector = new WallAudioSelector()
  selector.setActive('device-a')

  const previous = selector.clear('device-a')

  assert.equal(previous, 'device-a')
  assert.equal(selector.activeId, '')
})

test('clearing another tile leaves the current audio owner untouched', () => {
  const selector = new WallAudioSelector()
  selector.setActive('device-a')

  const previous = selector.clear('device-b')

  assert.equal(previous, '')
  assert.equal(selector.activeId, 'device-a')
})

test('clearing the whole wall returns the last audio owner', () => {
  const selector = new WallAudioSelector()
  selector.setActive('device-a')

  const previous = selector.clear()

  assert.equal(previous, 'device-a')
  assert.equal(selector.activeId, '')
})
