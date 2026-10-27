import assert from 'node:assert/strict'
import test from 'node:test'

import { deriveVideoRates, isComparableSample } from '../js/ui/videoRates.js'

// A monitor switch replaces the stream, and the new stream's counters start from
// zero. The difference is then large and negative, and the naive subtraction puts
// a negative frame rate on the stats overlay - the operator reads it as "the
// decoder is running backwards".
test('a counter that went backwards yields no rate, not a negative one', () => {
  const previous = { framesDecoded: 5000, bytesReceived: 900000, packetsReceived: 4000 }
  const current = { framesDecoded: 12, bytesReceived: 3000, packetsReceived: 20 }

  const rates = deriveVideoRates(previous, current, 1)

  assert.equal(rates.fps, 0, `fps was ${rates.fps}`)
  assert.equal(rates.bitrateKbps, 0, `bitrate was ${rates.bitrateKbps}`)
  assert.equal(rates.packetRate, 0, `packet rate was ${rates.packetRate}`)
})

test('normal counters give the expected per-second rates', () => {
  const previous = { framesDecoded: 1000, bytesReceived: 100000, packetsReceived: 500 }
  const current = { framesDecoded: 1030, bytesReceived: 112500, packetsReceived: 560 }

  const rates = deriveVideoRates(previous, current, 1)

  assert.equal(rates.fps, 30)
  assert.equal(rates.bitrateKbps, 100)
  assert.equal(rates.packetRate, 60)
})

// The first sample has no predecessor, so there is no difference to take.
test('the first sample reports no rates', () => {
  const rates = deriveVideoRates(null, { framesDecoded: 30, bytesReceived: 5000 }, 1)

  assert.equal(rates.fps, 0)
  assert.equal(rates.bitrateKbps, 0)
})

// A stats poll that somehow arrives with the same timestamp would divide by zero
// and produce Infinity, which the overlay would render as a number nobody can act
// on.
test('a zero or negative time delta reports no rates', () => {
  const previous = { framesDecoded: 10, bytesReceived: 100, packetsReceived: 10 }
  const current = { framesDecoded: 20, bytesReceived: 200, packetsReceived: 20 }

  for (const delta of [0, -1]) {
    const rates = deriveVideoRates(previous, current, delta)
    assert.equal(rates.fps, 0, `delta ${delta} produced fps ${rates.fps}`)
    assert.equal(Number.isFinite(rates.bitrateKbps), true)
  }
})

// Missing fields are the common case rather than an error: an audio-only report,
// or a browser that does not expose framesDecoded, produces a sample with holes.
test('missing counters are treated as no movement', () => {
  const rates = deriveVideoRates({}, { framesDecoded: 30 }, 1)

  assert.equal(rates.fps, 0)
  assert.equal(rates.bitrateKbps, 0)
  assert.equal(rates.packetRate, 0)
})

test('a sample whose counters only went backwards is not comparable', () => {
  const previous = { framesDecoded: 5000, bytesReceived: 900000 }
  const current = { framesDecoded: 12, bytesReceived: 3000 }

  assert.equal(isComparableSample(previous, current), false)
})

test('a sample that moved forward is comparable', () => {
  const previous = { framesDecoded: 100 }
  const current = { framesDecoded: 130 }

  assert.equal(isComparableSample(previous, current), true)
})

test('a first or missing sample is not comparable', () => {
  assert.equal(isComparableSample(null, { framesDecoded: 1 }), false)
  assert.equal(isComparableSample({ framesDecoded: 1 }, null), false)
})
