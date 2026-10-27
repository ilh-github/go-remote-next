/**
 * input/visibilityRelease.js - which browser events must tell the host to drop
 * its held keys.
 *
 * The host keeps its own record of which keys and buttons are down, and it has
 * no way to notice that this controller stopped paying attention. The Qt client
 * guards four conditions - losing focus, the window going inactive, the view
 * becoming invisible, and the window itself deactivating - because a system
 * shortcut or a focus change can swallow the matching key-up and the next person
 * at the machine types into a stuck modifier.
 *
 * The web client only listened for blur on the element that receives keys, which
 * covers clicking elsewhere on the page and nothing else. Switching browser tabs,
 * switching applications, minimising, or the screen locking all leave the host
 * holding whatever was down, and none of them produce a blur on that element.
 *
 * The rule is stated here rather than inline so the set of events is a thing that
 * can be asserted: an event added later to the browser's model, or one removed by
 * a platform, changes this list and the test says which way.
 */

/**
 * The events that mean "this controller is no longer the thing the operator is
 * driving with", and therefore that the host must be told to release.
 *
 * @returns {Array<{event: string, when: (state: object) => boolean}>}
 */
export function releaseTriggers() {
  return [
    {
      // Fires on tab switch, application switch, minimise, and screen lock. It
      // is the one event that covers every way the page stops being visible,
      // and it is also the only one that fires when the screen locks - which
      // produces no window blur at all.
      event: 'visibilitychange',
      when: (state) => state.hidden === true,
    },
    {
      // The window itself losing activation. Distinct from visibility: a window
      // can stay visible while another application takes focus, and the operator
      // is then typing into the remote machine by accident.
      event: 'blur',
      when: () => true,
      target: 'window',
    },
    {
      // The page is going away. `beforeunload` is not reliable on mobile
      // browsers and does not fire when the tab is discarded, so the page-hide
      // path is the one that actually runs there.
      event: 'pagehide',
      when: () => true,
      target: 'window',
    },
  ]
}

/**
 * Whether a given browser event should cause a release.
 *
 * @param {string} eventName the event that fired
 * @param {object} state     the relevant document/window state, e.g. `{hidden: true}`
 * @returns {boolean}
 */
export function shouldReleaseFor(eventName, state = {}) {
  return releaseTriggers().some(
    (trigger) => trigger.event === eventName && trigger.when(state),
  )
}

/**
 * Attaches the release triggers to the window.
 *
 * @param {Window} target
 * @param {() => void} onRelease called once per event that warrants a release
 * @returns {() => void} detach
 */
export function watchForRelease(target, onRelease) {
  if (!target || typeof target.addEventListener !== 'function') {
    return () => {}
  }

  const detachers = []
  for (const trigger of releaseTriggers()) {
    const scope = trigger.target === 'window' ? target : target.document || target
    const handler = () => {
      if (!trigger.when(readState(scope))) return
      onRelease()
    }
    scope.addEventListener(trigger.event, handler)
    detachers.push(() => scope.removeEventListener(trigger.event, handler))
  }
  return () => detachers.forEach((detach) => detach())
}

function readState(scope) {
  return { hidden: scope.hidden === true, focused: scope.focused !== false }
}
