// QD-106/QD-909: the owner's view of attempts to reach their machine.
//
// The server records every access attempt at verification time - who, from
// where, and what happened - and has served them at
// `GET /v1/me/devices/:id/access-logs` for a while. Nothing consumed it, so the
// question an owner actually asks, "is somebody trying to get into my machine?",
// had no answer in any interface. The session detail shows the access log for
// sessions that started; a failed attempt never creates a session, so the rows
// that matter most were the ones with nowhere to appear.
//
// Rows are written by the server, never by the controller, so a client cannot
// hide or forge its own attempts. This module's job is to say what each
// canonical result means, because "no session" covers several very different
// things: a wrong code is somebody guessing, while an offline host is the code
// being right and the machine not being there.

/**
 * The outcomes the UI distinguishes. They are not the server's result strings -
 * those are finer-grained, and several of them collapse to the same thing an
 * operator needs to know.
 */
export const ACCESS_LOG_OUTCOME = {
  Granted: 'granted',
  Denied: 'denied',
  Pending: 'pending',
  Offline: 'offline',
  Unknown: 'unknown',
}

// Results that mean the code was right and something else refused. They are
// denials, but they are the owner's own rules working - not somebody guessing -
// so flagging them would drown the rows that matter.
const policyDenials = new Set([
  'denied_policy',
  'denied_session_limit',
  'denied_revoked',
  'denied_approval',
])

// Results that mean the code was wrong or the attempts were too many. These are
// the ones an owner scanning the list is looking for.
const guessDenials = new Set(['denied_invalid_code', 'denied_rate_limited'])

function text(value) {
  if (value === undefined || value === null) return ''
  return String(value).trim()
}

/**
 * Describes one access-log row.
 *
 * @param {object} row a row from `GET /v1/me/devices/:id/access-logs`
 * @returns {{
 *   id: number,
 *   at: string,
 *   ip: string,
 *   clientId: string,
 *   sessionId: string,
 *   result: string,
 *   reason: string,
 *   outcome: string,
 *   denied: boolean,
 *   suspicious: boolean,
 * }}
 */
export function describeAccessLog(row) {
  const log = row || {}
  const result = text(log.result)

  let outcome = ACCESS_LOG_OUTCOME.Unknown
  let suspicious = false
  if (result === 'granted' || result === 'session_started') {
    outcome = ACCESS_LOG_OUTCOME.Granted
  } else if (guessDenials.has(result)) {
    outcome = ACCESS_LOG_OUTCOME.Denied
    suspicious = true
  } else if (policyDenials.has(result)) {
    outcome = ACCESS_LOG_OUTCOME.Denied
  } else if (result === 'pending_approval') {
    outcome = ACCESS_LOG_OUTCOME.Pending
  } else if (result === 'host_offline') {
    outcome = ACCESS_LOG_OUTCOME.Offline
  }

  return {
    id: log.id,
    at: text(log.created_at),
    ip: text(log.ip),
    clientId: text(log.client_id),
    sessionId: text(log.session_id),
    result,
    reason: text(log.reason),
    outcome,
    denied: outcome === ACCESS_LOG_OUTCOME.Denied,
    suspicious,
  }
}

/**
 * Describes a page of rows, keeping the server's newest-first order.
 *
 * @param {Array<object>} items
 * @returns {Array<object>}
 */
export function describeAccessLogs(items) {
  return Array.isArray(items) ? items.map(describeAccessLog) : []
}
