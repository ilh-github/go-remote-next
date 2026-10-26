/**
 * soak/allocationProfile.js - QD-908: what is allocating, and on what period.
 *
 * The 2026-10-24 two-hour wall soak established two facts that together rule
 * out the obvious explanations. FPS stepped down between 267 and 681 seconds
 * while the DOM node count peaked at 267s and stopped growing, so the frame
 * collapse was not the wall accumulating nodes. And FPS went to zero while the
 * heap was still climbing linearly, so whatever allocates does not depend on
 * the main-thread frame loop. Both point at a source outside the frame path.
 *
 * The checklist's next step is therefore "查周期性分配源", and that needs two
 * numbers rather than an argument: a call path with a share of total
 * allocation, and a period that survives the heap series' own noise. The peak
 * tracker already showed the heap oscillating across 111% of its own mean while
 * nothing was wrong, so a period is only a finding if it is decisive against
 * that noise.
 *
 * Kept out of the E2E file for the same reason as samplingPlan.js: these are
 * the rules worth testing, and neither can be tested without a two-hour run if
 * it lives inside one.
 */

/**
 * Flattens a CDP HeapProfiler sampling profile into the path that allocated the
 * most on its own.
 *
 * "Self" size rather than total, because a path that is large only because its
 * children are large names a container, not an allocator. The share is reported
 * alongside the bytes because it is what decides whether a path is worth
 * chasing at all.
 *
 * @param {{head: object}} profile the HeapProfiler sampling profile
 * @returns {{totalBytes: number, top: object|null}}
 */
export function summarizeAllocationProfile(profile) {
  const head = profile?.head
  if (!head) return { totalBytes: 0, top: null }

  let totalBytes = 0
  let top = null
  const stack = [head]
  while (stack.length > 0) {
    const entry = stack.pop()
    const self = Number(entry.selfSize) || 0
    totalBytes += self
    if (self > 0 && (top === null || self > top.selfBytes)) {
      top = {
        selfBytes: self,
        callFrame: entry.callFrame,
        children: (entry.children || []).length,
      }
    }
    for (const child of entry.children || []) stack.push(child)
  }
  if (top && totalBytes > 0) {
    top.share = Number((top.selfBytes / totalBytes).toFixed(3))
  }
  return { totalBytes, top }
}

/**
 * Finds the dominant period in a time series, if it has one.
 *
 * Autocorrelation against its own mean, then the strongest non-zero lag. The
 * mean is subtracted first: a series with a rising trend correlates with itself
 * at every lag, which would report a period for a straight line. The zero lag
 * is excluded because it is trivially the strongest and means nothing.
 *
 * The strength threshold is what keeps this honest. The heap series is noisy by
 * nature - the peak tracker measured it swinging across 111% of its own mean
 * while FPS and node count held still - so a weak correlation is reported as no
 * period rather than as a finding.
 *
 * @param {Array<{atMs: number, bytes: number}>} samples
 * @returns {{periodMs: number, strength: number}}
 */
export function periodicity(samples) {
  const usable = (samples || []).filter(
    (sample) => Number.isFinite(sample?.bytes) && Number.isFinite(sample?.atMs),
  )
  // A period needs at least two full cycles to be distinguishable from one
 // outlier, and three points to have a lag at all.
  if (usable.length < 12) return { periodMs: 0, strength: 0 }

  const sorted = [...usable].sort((left, right) => left.atMs - right.atMs)
  const intervalMs = medianInterval(sorted)
  if (!(intervalMs > 0)) return { periodMs: 0, strength: 0 }

  const values = sorted.map((sample) => sample.bytes)
  const mean = values.reduce((sum, value) => sum + value, 0) / values.length
  const centered = values.map((value) => value - mean)
  const energy = centered.reduce((sum, value) => sum + value * value, 0)
  if (energy === 0) return { periodMs: 0, strength: 0 }

  let best = { lag: 0, strength: 0 }
  for (let lag = 1; lag < Math.floor(values.length / 2); lag += 1) {
    let dot = 0
    for (let index = 0; index + lag < values.length; index += 1) {
      dot += centered[index] * centered[index + lag]
    }
    const strength = dot / energy
    if (strength > best.strength) best = { lag, strength }
  }

  // Below this the correlation is not separable from the noise the heap series
  // is known to carry, so it is reported as no period at all. A number that
  // looks like a finding but is not one is worse than no number.
  const decisive = 0.5
  if (best.strength < decisive) return { periodMs: 0, strength: best.strength }
  return { periodMs: Math.round(best.lag * intervalMs), strength: Number(best.strength.toFixed(3)) }
}

function medianInterval(sorted) {
  const gaps = []
  for (let index = 1; index < sorted.length; index += 1) {
    const gap = sorted[index].atMs - sorted[index - 1].atMs
    if (gap > 0) gaps.push(gap)
  }
  if (gaps.length === 0) return 0
  gaps.sort((left, right) => left - right)
  const middle = Math.floor(gaps.length / 2)
  return gaps.length % 2 === 1 ? gaps[middle] : (gaps[middle - 1] + gaps[middle]) / 2
}
