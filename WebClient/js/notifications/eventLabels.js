/**
 * notifications/eventLabels.js - the label a realtime event renders as.
 *
 * The map used to be inline in NotificationsPage.vue. That meant an event type
 * the build had not heard of fell through to the raw wire name, so an operator
 * saw `device.connect_prompted` in a table cell instead of a sentence - and
 * nothing checked it, because the only way to notice was to look at the page.
 *
 * It lives here so the mapping is a test rather than a hope, and so the
 * fallback is deliberate rather than an accident of a missing object key.
 */

// Event type -> i18n key under `notifications.eventNames.*`. A type that is not
// here renders as its raw name, which is the honest answer for something this
// build cannot describe.
export const eventLabelKeys = {
  'device.access.denied': 'accessDenied',
  'device.access.granted': 'accessGranted',
  'device.access.session': 'accessSession',
  'device.online.changed': 'onlineChanged',
  'device.offline.sustained': 'offlineSustained',
  'device.bound': 'bound',
  'device.unbound': 'unbound',
  'device.access_approval.requested': 'approvalRequested',
  'device.access_approval.decided': 'approvalDecided',
  'scheduled_task.failed': 'scheduledTaskFailed',
  'device.metric.alert': 'metricAlert',
  // QD-607: a scheduled task told the owner's clients that a connection is
  // due. The server publishes it; a client has to be able to name it.
  'device.connect_prompted': 'connectPrompted',
}

/**
 * The i18n key for an event type, or the type itself when there is none.
 *
 * @param {string} eventType the wire event type
 * @returns {string}
 */
export function eventLabelKey(eventType) {
  const type = typeof eventType === 'string' ? eventType.trim() : ''
  if (!type) return ''
  return eventLabelKeys[type] || type
}
