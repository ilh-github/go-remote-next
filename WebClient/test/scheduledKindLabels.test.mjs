import assert from 'node:assert/strict'
import test from 'node:test'

// QD-607 added a third task kind, connect_prompt. The label map was inline in
// ScheduledTasksPage.vue and had entries for the original two, so the new kind
// rendered as its raw wire name - the same half-wired pattern the event label map
// had one round earlier. It lives in a module now so the coverage is a test.

const { scheduledKindLabelKey, SCHEDULED_KIND_LABEL_KEYS, fallbackScheduledKinds } =
  await import('../js/scheduled/kindLabels.js')

test('every server kind has a label key', () => {
  assert.equal(scheduledKindLabelKey('sign_out_all_sessions'), 'signOutAllSessions')
  assert.equal(scheduledKindLabelKey('revoke_all_remote_sessions'), 'revokeAllRemoteSessions')
  assert.equal(scheduledKindLabelKey('connect_prompt'), 'connectPrompt')
})

// An unknown kind must not render as an empty cell, and must not pretend to be a
// known one either: a kind this build has not heard of should say so.
test('an unknown kind falls back to the unknown label', () => {
  assert.equal(scheduledKindLabelKey('some_future_kind'), 'unknown')
  assert.equal(scheduledKindLabelKey(''), 'unknown')
  assert.equal(scheduledKindLabelKey(undefined), 'unknown')
})

// The map must not drift from the kinds the server actually registers. This is
// the assertion that would have caught the gap a round ago.
test('the map covers exactly the kinds the server registers', () => {
  assert.deepEqual(
    [...SCHEDULED_KIND_LABEL_KEYS].sort(),
    ['connect_prompt', 'revoke_all_remote_sessions', 'sign_out_all_sessions'],
  )
})

// The fallback list the create dialog shows before the server's supported_kinds
// arrives. It was written for the original two kinds, so a third kind is invisible
// in the picker until the list loads - and a user who opens the dialog on a slow
// connection cannot schedule the feature at all.
test('the fallback list covers every kind the server registers', () => {
  for (const kind of SCHEDULED_KIND_LABEL_KEYS) {
    assert.ok(
      fallbackScheduledKinds.includes(kind),
      `${kind} is missing from the create dialog's fallback list`,
    )
  }
})
