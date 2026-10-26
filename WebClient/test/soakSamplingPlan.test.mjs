import assert from 'node:assert/strict'
import test from 'node:test'

// QD-908: the dense-sampling window is the difference between "FPS collapsed
// somewhere between 267 and 681 seconds" and "here is the second it collapsed".
// The rule that picks the interval lives in soak/samplingPlan.js so it can be
// tested without a two-hour run.

const { buildSamplingPlan, intervalAt, plannedSampleCount, denseResolution } =
  await import('../js/soak/samplingPlan.js')

// The default every existing run gets: no window configured, one interval for
// the whole run. A run that never asked for dense sampling must not silently
// acquire one.
test('no window configured means one interval for the whole run', () => {
  const plan = buildSamplingPlan({ intervalMs: 1000 })
  assert.equal(plan.window, null)
  assert.equal(intervalAt(plan, 0), 1000)
  assert.equal(intervalAt(plan, 267_000), 1000)
  assert.equal(intervalAt(plan, 681_000), 1000)
})

// The documented case: the 2026-10-24 run stepped down between 267 and 681
// seconds, so that is the window a follow-up run carves out.
test('the dense window tightens the interval inside it only', () => {
  const plan = buildSamplingPlan({
    intervalMs: 1000,
    fromMs: 267_000,
    toMs: 681_000,
    denseIntervalMs: 100,
  })
  assert.ok(plan.window, 'a configured window must survive')
  assert.equal(intervalAt(plan, 0), 1000, 'before the window: coarse')
  assert.equal(intervalAt(plan, 266_999), 1000, 'just before: coarse')
  assert.equal(intervalAt(plan, 267_000), 100, 'window start is inclusive')
  assert.equal(intervalAt(plan, 500_000), 100, 'inside: dense')
  assert.equal(intervalAt(plan, 680_999), 100, 'last dense second')
  assert.equal(intervalAt(plan, 681_000), 1000, 'window end is exclusive')
  assert.equal(intervalAt(plan, 7_200_000), 1000, 'after: coarse')
})

// A half-open upper bound is what makes the boundary answerable. If both bounds
// were inclusive the boundary second would belong to two intervals at once.
test('the window is half-open so the boundary second is unambiguous', () => {
  const plan = buildSamplingPlan({
    intervalMs: 1000, fromMs: 10_000, toMs: 20_000, denseIntervalMs: 200,
  })
  assert.equal(intervalAt(plan, 20_000), 1000, 'the end second is coarse, not both')
})

// A window that covers no time must be dropped rather than half-configured:
// `from >= to` would otherwise claim a dense window while sampling nothing
// densely, and the summary would carry that claim.
test('a window that covers no time is dropped', () => {
  for (const [fromMs, toMs] of [[5000, 5000], [6000, 5000]]) {
    const plan = buildSamplingPlan({ intervalMs: 1000, fromMs, toMs, denseIntervalMs: 100 })
    assert.equal(plan.window, null,
      `from=${fromMs} to=${toMs} must not produce a window`)
    assert.equal(intervalAt(plan, 5500), 1000)
  }
})

// A dense interval that is not finer than the coarse one buys no resolution but
// still spends the sampling budget, so it is refused rather than accepted as a
// no-op that looks configured.
test('a dense interval no finer than the coarse one is refused', () => {
  for (const denseIntervalMs of [1000, 2000, 0, -5]) {
    const plan = buildSamplingPlan({
      intervalMs: 1000, fromMs: 0, toMs: 1000, denseIntervalMs,
    })
    assert.equal(plan.window, null,
      `dense=${denseIntervalMs} must not be accepted against a coarse 1000`)
  }
})

// A negative start is dropped too: a window beginning before the run exists
// cannot be honoured, and clamping it would silently move the dense stretch.
test('a negative window start is dropped', () => {
  const plan = buildSamplingPlan({
    intervalMs: 1000, fromMs: -1, toMs: 1000, denseIntervalMs: 100,
  })
  assert.equal(plan.window, null)
})

// The sample count is what tells an operator what resolution a run will actually
// produce, and the dense portion has to be counted at its own rate.
test('the planned sample count splits coarse and dense spans', () => {
  const plan = buildSamplingPlan({
    intervalMs: 1000, fromMs: 100_000, toMs: 200_000, denseIntervalMs: 100,
  })
  const counts = plannedSampleCount(plan, 300_000)
  // 100s coarse before the window + 100s dense at 100ms (=1000) + 100s coarse
  // after it: the dense span is counted at its own rate, and both coarse spans
  // at the coarse one.
  assert.equal(counts.dense, 1000)
  assert.equal(counts.total, 1200, '200 coarse + 1000 dense')

  // A window that runs past the end of the soak is clamped, not extrapolated.
  const clamped = plannedSampleCount(plan, 150_000)
  assert.equal(clamped.dense, 500, 'only the part inside the run counts')
  assert.equal(clamped.total, 600, '100 coarse before + 500 dense')
})

// A run with no window reports only the coarse count, so the dense figure being
// zero is a fact about the plan rather than a missing field.
test('the sample count without a window reports no dense samples', () => {
  const plan = buildSamplingPlan({ intervalMs: 1000 })
  assert.deepEqual(plannedSampleCount(plan, 60_000), { total: 60, dense: 0 })
})

// A plan is optional at the call site: a caller that never built one still gets
// the coarse default rather than a crash.
test('intervalAt tolerates no plan', () => {
  assert.equal(intervalAt(null, 5000), 1000)
  assert.equal(intervalAt(undefined, 5000), 1000)
})

// The interval a plan asks for and the interval a run achieves are different
// numbers once sampling itself costs time. Under 200 viewers one sample takes
// about seven seconds, so a 500 ms dense interval is overshot fourteen-fold and
// a window configured for 240 points yields six. Reporting the configured
// interval as if it were the achieved one makes the trace's resolution a
// property of machine load while the summary claims it was configured.
test('a dense window sampling cannot deliver reports the gap', () => {
  const plan = buildSamplingPlan({
    intervalMs: 5000, fromMs: 0, toMs: 120000, denseIntervalMs: 500,
  })
  // A cheap sample: the plan's own interval is the binding constraint.
  const cheap = denseResolution(plan, 100)
  assert.equal(cheap.achievable, true)
  assert.equal(cheap.effectiveMs, 500, 'the requested interval is what is used')

  // The measured 200-viewer cost: the sample dominates, so the window is not
  // actually dense even though it is configured to be.
  const expensive = denseResolution(plan, 7000)
  assert.equal(expensive.achievable, false)
  assert.equal(expensive.effectiveMs, 7000,
    'the effective interval must be the real one, not the configured one')
  assert.equal(expensive.requestedMs, 500)
})

// A sample exactly as expensive as the requested interval is still achievable:
// the loop keeps up, it just has no slack.
test('a sample that costs exactly the interval is achievable', () => {
  const plan = buildSamplingPlan({
    intervalMs: 1000, fromMs: 0, toMs: 10000, denseIntervalMs: 200,
  })
  const result = denseResolution(plan, 200)
  assert.equal(result.achievable, true)
  assert.equal(result.effectiveMs, 200)
})

// With no window the question is about the coarse interval, so a caller that
// never configured dense sampling still gets an honest answer.
test('resolution without a window reports the coarse interval', () => {
  const plan = buildSamplingPlan({ intervalMs: 1000 })
  const result = denseResolution(plan, 250)
  assert.equal(result.achievable, true)
  assert.equal(result.requestedMs, 1000)
  assert.equal(result.effectiveMs, 1000)
})

// A caller with no plan at all must not crash: the coarse default is the answer.
test('resolution tolerates no plan', () => {
  const result = denseResolution(null, 5000)
  assert.equal(result.achievable, true)
  assert.equal(result.requestedMs, 1000)
})
