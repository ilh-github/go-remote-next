/**
 * soak/samplingPlan.js - QD-908: which sampling interval applies at a moment.
 *
 * The 2026-10-24 two-hour wall soak showed main-thread FPS stepping down between
 * 267 and 681 seconds rather than sliding, while DOM node count peaked at 267s
 * and stopped growing. A one-second series across two hours can say the average
 * is bad; it cannot say what happened at the step, because the step falls
 * between two of its points. Carving out a dense window - fine sampling inside
 * a stretch of interest, coarse everywhere else - is what turns "it collapsed
 * somewhere in there" into "here is the second it collapsed".
 *
 * Kept out of the E2E file on purpose: the rule that decides the interval is the
 * part worth testing, and a function buried inside a two-hour test cannot be
 * tested without running two hours.
 */

/**
 * Builds a sampling plan from raw configuration.
 *
 * A window that does not actually cover any time is dropped rather than
 * normalised into something that looks configured: `from >= to` would silently
 * sample nothing densely, and a run whose summary claims a dense window while
 * taking every sample at the coarse rate is a misleading record.
 *
 * @param {object} config
 * @param {number} config.intervalMs         the coarse, whole-run interval
 * @param {number} [config.fromMs]           dense window start
 * @param {number} [config.toMs]             dense window end
 * @param {number} [config.denseIntervalMs]  interval inside the window
 * @returns {{intervalMs: number, window: object|null}}
 */
export function buildSamplingPlan({
  intervalMs,
  fromMs = 0,
  toMs = 0,
  denseIntervalMs = 0,
} = {}) {
  const coarse = positive(intervalMs, 1000)
  if (!(fromMs >= 0) || !(toMs > fromMs)) {
    return { intervalMs: coarse, window: null }
  }
  // A dense interval that is not actually finer than the coarse one would spend
  // the window's extra sampling budget for no resolution, so it is refused
  // rather than silently widening the sample count.
  const dense = positive(denseIntervalMs, 0)
  if (dense <= 0 || dense >= coarse) {
    return { intervalMs: coarse, window: null }
  }
  return {
    intervalMs: coarse,
    window: { fromMs, toMs, intervalMs: dense },
  }
}

/**
 * The interval that applies at `elapsedMs` into the run.
 *
 * The window is half-open `[fromMs, toMs)`. A closed upper bound would make the
 * boundary second ambiguous - it belongs to the dense window or to the coarse
 * one, and a plan that cannot answer that cannot be reasoned about.
 *
 * @param {{intervalMs: number, window: object|null}} plan
 * @param {number} elapsedMs
 * @returns {number}
 */
export function intervalAt(plan, elapsedMs) {
  if (!plan) return 1000
  if (!plan.window) return plan.intervalMs
  const { fromMs, toMs, intervalMs } = plan.window
  if (elapsedMs >= fromMs && elapsedMs < toMs) return intervalMs
  return plan.intervalMs
}

/**
 * How many samples the plan will produce over a run of `durationMs`, assuming
 * each sample takes no time.
 *
 * The assumption is stated because it is what makes the number useful: it is the
 * floor, and the real count is lower by however long each sample's own work
 * takes. A summary that reported this as the actual count would overstate the
 * resolution of the trace it ships with.
 *
 * @param {{intervalMs: number, window: object|null}} plan
 * @param {number} durationMs
 * @returns {{total: number, dense: number}}
 */
export function plannedSampleCount(plan, durationMs) {
  const coarse = plan ? plan.intervalMs : 1000
  if (!(durationMs > 0)) return { total: 0, dense: 0 }
  const window = plan && plan.window
  if (!window) {
    return { total: Math.floor(durationMs / coarse), dense: 0 }
  }
  const denseStart = Math.max(0, Math.min(window.fromMs, durationMs))
  const denseEnd = Math.max(denseStart, Math.min(window.toMs, durationMs))
  const denseSpan = denseEnd - denseStart
  const coarseSpan = durationMs - denseSpan
  return {
    total: Math.floor(coarseSpan / coarse) + Math.floor(denseSpan / window.intervalMs),
    dense: Math.floor(denseSpan / window.intervalMs),
  }
}

/**
 * Whether the plan can actually deliver the resolution it promises.
 *
 * The interval a plan *asks* for and the interval a run *achieves* are different
 * numbers once sampling itself costs time. Under 200 viewers one sample takes
 * about seven seconds (page.evaluate plus a CDP metrics read), so a 500 ms dense
 * interval is overshot fourteen-fold and the window yields six points instead of
 * the two hundred and forty it was configured for. A plan that cannot say so
 * ships a trace whose resolution is a property of machine load rather than of
 * configuration, and a summary that reports the configured interval as if it
 * were the achieved one is a lie about its own evidence.
 *
 * This reports the gap so the E2E can carry it in the summary rather than
 * silently producing a dense window that is not dense.
 *
 * @param {{intervalMs: number, window: object|null}} plan
 * @param {number} sampleCostMs  how long one sample actually takes
 * @returns {{achievable: boolean, effectiveMs: number, requestedMs: number}}
 */
export function denseResolution(plan, sampleCostMs) {
  const requested = plan && plan.window ? plan.window.intervalMs : (plan ? plan.intervalMs : 1000)
  // A caller with no plan has nothing to fall short of, so the question does not
  // apply: reporting "not achievable" there would make the coarse default look
  // broken by a cost it was never trying to beat.
  if (!plan) {
    return { achievable: true, effectiveMs: requested, requestedMs: requested }
  }
  const cost = positive(sampleCostMs, 0)
  const effective = Math.max(requested, cost)
  return {
    achievable: effective <= requested,
    effectiveMs: effective,
    requestedMs: requested,
  }
}

function positive(value, fallback) {
  return typeof value === 'number' && Number.isFinite(value) && value > 0 ? value : fallback
}
