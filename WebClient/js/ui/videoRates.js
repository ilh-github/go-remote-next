/**
 * ui/videoRates.js - deriving per-second rates from WebRTC stats reports.
 *
 * `getStats()` hands back cumulative counters, so a rate is always a difference
 * between two samples divided by the time between them. Two things make that
 * subtraction unsafe, and both happen in normal use rather than as errors:
 *
 * 1. Switching monitors replaces the media stream, and the new stream's counters
 *    start from zero. The difference is then large and negative, and the naive
 *    result is a negative frame rate and a negative bitrate on the stats overlay.
 * 2. The very first sample has no predecessor, so there is no difference to take
 *    and no rate to report.
 *
 * Both live here rather than inside the overlay because the rule is what is worth
 * testing, and the overlay can only be exercised by standing up a whole
 * RTCPeerConnection.
 */

/**
 * @param {object|null} previous the earlier video sample, or null on the first tick
 * @param {object} current   the current video sample
 * @param {number} timeDelta seconds between the two samples
 * @returns {{fps: number, bitrateKbps: number, packetRate: number}}
 */
export function deriveVideoRates(previous, current, timeDelta) {
  const rates = { fps: 0, bitrateKbps: 0, packetRate: 0 }
  if (!previous || !current) return rates
  if (!(timeDelta > 0)) return rates

  const framesDelta = counterDelta(current.framesDecoded, previous.framesDecoded)
  const bytesDelta = counterDelta(current.bytesReceived, previous.bytesReceived)
  const packetsDelta = counterDelta(current.packetsReceived, previous.packetsReceived)

  rates.fps = Math.round(framesDelta / timeDelta)
  rates.bitrateKbps = Math.round((bytesDelta * 8) / timeDelta / 1000)
  rates.packetRate = Math.round(packetsDelta / timeDelta)
  return rates
}

/**
 * The difference between two cumulative counters, or 0 when the counter went
 * backwards.
 *
 * A counter that decreased did so because whatever it was counting was replaced
 * rather than because time ran backwards, so the honest answer is "no rate can be
 * derived from these two samples" - which is 0, not a negative number. Reporting
 * a negative frame rate tells the operator the decoder is running backwards.
 *
 * @param {number|undefined} current
 * @param {number|undefined} previous
 * @returns {number}
 */
function counterDelta(current, previous) {
  const now = Number(current)
  const before = Number(previous)
  if (!Number.isFinite(now) || !Number.isFinite(before)) return 0
  const delta = now - before
  return delta > 0 ? delta : 0
}

/**
 * Whether a sample's counters are comparable with a previous one's.
 *
 * Used to decide whether the previous sample is still worth keeping: once the
 * stream is replaced, every later comparison against the old sample is
 * meaningless, so it is dropped rather than carried forward producing one bogus
 * rate per monitor switch.
 *
 * @param {object|null} previous
 * @param {object} current
 * @returns {boolean}
 */
export function isComparableSample(previous, current) {
  if (!previous || !current) return false
  return counterDelta(current.framesDecoded, previous.framesDecoded) > 0 ||
    counterDelta(current.bytesReceived, previous.bytesReceived) > 0
}
