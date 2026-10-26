/**
 * scheduled/kindLabels.js - the label a scheduled-task kind renders as.
 *
 * The map was inline in ScheduledTasksPage.vue with entries for the original two
 * kinds, so the third kind this project added rendered as its raw wire name in a
 * table cell - the same half-wired pattern the realtime event label map had a
 * round earlier. It lives here so a new kind without a label is a failing test
 * rather than something an operator notices.
 */

// The kinds the signaling server registers, in the order they were added. Kept
// explicit rather than derived from an API response so a test can assert the map
// has not drifted from what the server accepts.
export const SCHEDULED_KIND_LABEL_KEYS = Object.freeze([
  'sign_out_all_sessions',
  'revoke_all_remote_sessions',
  'connect_prompt',
])

// What the create dialog offers before the server's supported_kinds arrives. It
// is the same list rather than a second copy, because the previous version was a
// second copy: it was written for the original two kinds, so the third was
// invisible in the picker until the list loaded, and a user who opened the dialog
// on a slow connection could not schedule the feature at all.
export const fallbackScheduledKinds = Object.freeze([...SCHEDULED_KIND_LABEL_KEYS])

const labelKeys = {
  sign_out_all_sessions: 'signOutAllSessions',
  revoke_all_remote_sessions: 'revokeAllRemoteSessions',
  connect_prompt: 'connectPrompt',
}

/**
 * The i18n key for a kind, or the unknown label when the build has not heard of
 * it. A kind that renders as an empty cell is worse than one that says it is
 * unknown.
 *
 * @param {string} kind the wire task kind
 * @returns {string}
 */
export function scheduledKindLabelKey(kind) {
  const key = typeof kind === 'string' ? kind.trim() : ''
  return labelKeys[key] || 'unknown'
}
