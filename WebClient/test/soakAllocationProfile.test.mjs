import assert from 'node:assert/strict'
import test from 'node:test'

// QD-908: the 2026-10-24 run established that heap growth and FPS collapse are
// two independent degradations - FPS went to zero while heap was still climbing
// linearly, so the allocation source does not depend on the main-thread frame
// loop. The next question the checklist names is "查周期性分配源": which code
// keeps allocating, and does it do so on a period?
//
// A heap that grows without an attributed source is an argument; an allocation
// profile that names a call path and a period is evidence.

const { summarizeAllocationProfile, periodicity } = await import('../js/soak/allocationProfile.js')

// CDP's HeapProfiler reports a sampling profile as a list of nodes, each with a
// self size and a tree of children. The shape the test needs is the shape the
// CDP produces, so the fixtures are built that way rather than into whatever
// shape is convenient.
function node(selfSize, children = [], callFrame = { functionName: 'anon', url: 'x.js', lineNumber: 1 }) {
  return { callFrame, selfSize, children }
}

// The shape the 2026-10-24 run pointed at: something allocating on a steady
// period, independent of the frame loop. A profile with one dominant path and a
// series that repeats at a fixed interval is what that looks like.
test('a profile names the call path that allocated the most', () => {
  const profile = {
    head: node(1000, [
      node(50, [], { functionName: 'small', url: 'a.js', lineNumber: 1 }),
      node(9000, [], { functionName: 'allocator', url: 'b.js', lineNumber: 42 }),
    ]),
  }
  const summary = summarizeAllocationProfile(profile)
  assert.equal(summary.totalBytes, 10_050)
  assert.equal(summary.top.selfBytes, 9000)
  assert.equal(summary.top.callFrame.functionName, 'allocator')
  // The share is what decides whether a path is worth chasing: a top entry that
  // is 90% of all allocation points somewhere, one that is 3% does not.
  assert.equal(summary.top.share, 0.896)
})

// An empty profile must not crash and must not claim a top entry: "nothing was
// sampled" and "nothing allocated" are different statements.
test('an empty profile reports no top entry', () => {
  const summary = summarizeAllocationProfile({ head: node(0) })
  assert.equal(summary.totalBytes, 0)
  assert.equal(summary.top, null)
})

// Periodicity is the actual question. The heap series is noisy - the peak
// tracker already showed it oscillating across 111% of its own mean while
// nothing was wrong - so a period has to survive that noise to be a finding.
// A steady 500ms allocation cycle must be found; white noise must not be.
test('a steady allocation cycle is found in a noisy series', () => {
  // A 500ms period with noise on top: the kind of series a real run produces.
  const samples = []
  for (let i = 0; i < 200; i += 1) {
    const phase = (i % 5) / 5
    const signal = Math.sin(phase * 2 * Math.PI) * 100
    const noise = ((i * 37) % 11) - 5
    samples.push({ atMs: i * 100, bytes: 1000 + signal + noise })
  }
  const result = periodicity(samples)
  assert.ok(result.periodMs > 0, 'a repeating signal must yield a period')
  assert.ok(Math.abs(result.periodMs - 500) <= 50, `period ${result.periodMs}, want about 500`)
  assert.ok(result.strength > 0.5, `strength ${result.strength} must be decisive`)
})

// The negative case matters as much: if any series produced a period, the
// finding would be worthless. Deterministic pseudo-noise must come back with no
// period rather than a flattering one.
test('a series with no cycle reports no period', () => {
  // A multiplicative hash of the index, not a linear recurrence: `(i * k) % m`
  // is a ramp with a sawtooth, which correlates with itself at the sawtooth's
  // own period and would "find" a cycle in pure noise. Mixing the bits first is
  // what makes this a fair negative control.
  const samples = []
  for (let i = 0; i < 200; i += 1) {
    let hash = (i + 1) * 2654435761
    hash ^= hash >>> 15
    hash = Math.imul(hash, 2246822507)
    hash ^= hash >>> 13
    samples.push({ atMs: i * 100, bytes: 1000 + (hash >>> 0) % 1013 })
  }
  const result = periodicity(samples)
  assert.equal(result.periodMs, 0, `no cycle expected, got ${result.periodMs}ms`)
})

// Too few samples to establish a period must say so rather than return a number
// that looks like a finding.
test('too few samples cannot establish a period', () => {
  assert.equal(periodicity([{ atMs: 0, bytes: 1 }, { atMs: 100, bytes: 2 }]).periodMs, 0)
  assert.equal(periodicity([]).periodMs, 0)
})
