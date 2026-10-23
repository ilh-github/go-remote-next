// Owner-facing helpers for the `control_mode` half of the access policy.
//
// The policy endpoint is a full replace: every field not sent is reset to its
// zero value. That is the whole reason this module exists - a control-mode
// switch that PUT only its own field would silently wipe the owner's IP lists,
// session cap, windows and approval settings in the same request.

export const CONTROL_MODE_SHARED = 'shared'
export const CONTROL_MODE_EXCLUSIVE = 'exclusive'

const KNOWN_MODES = [CONTROL_MODE_SHARED, CONTROL_MODE_EXCLUSIVE]

/**
 * The mode a stored policy describes. Anything unrecognised - a missing row, an
 * empty column, a value from a newer server - reads as shared, because that is
 * what such a device already does. Falling back to exclusive instead would tell
 * the owner their machine is locked down when it is not.
 *
 * @param {object|null} policy  the GET /access-policy response
 * @returns {string}
 */
export function readControlMode(policy) {
    if (!policy || typeof policy !== 'object') return CONTROL_MODE_SHARED
    const mode = policy.control_mode
    return KNOWN_MODES.includes(mode) ? mode : CONTROL_MODE_SHARED
}

/**
 * Whether saving is worth a request at all.
 * @param {string} from  currently stored mode
 * @param {string} to    mode chosen in the form
 * @returns {boolean}
 */
export function controlModeChanged(from, to) {
    return readControlMode({ control_mode: from }) !== readControlMode({ control_mode: to })
}

/**
 * Builds the PUT body: the stored policy with `control_mode` replaced.
 *
 * The stored row is the source of truth for every other field, so a policy that
 * was never saved contributes its documented defaults rather than nothing.
 *
 * @param {object|null} policy  the GET /access-policy response
 * @param {string} controlMode  the mode chosen in the form
 * @returns {object}
 */
export function buildAccessPolicyBody(policy, controlMode) {
    const stored = policy && typeof policy === 'object' ? policy : {}
    return {
        remote_disabled: Boolean(stored.remote_disabled),
        allow_ips: Array.isArray(stored.allow_ips) ? stored.allow_ips.slice() : [],
        deny_ips: Array.isArray(stored.deny_ips) ? stored.deny_ips.slice() : [],
        max_sessions: Number.isFinite(stored.max_sessions) ? stored.max_sessions : 0,
        allowed_windows: Array.isArray(stored.allowed_windows) ? stored.allowed_windows.slice() : [],
        timezone: typeof stored.timezone === 'string' ? stored.timezone : '',
        exception_dates: Array.isArray(stored.exception_dates) ? stored.exception_dates.slice() : [],
        approval_required: Boolean(stored.approval_required),
        approval_mode: typeof stored.approval_mode === 'string' && stored.approval_mode
            ? stored.approval_mode
            : 'off',
        approval_trusted_ips: Array.isArray(stored.approval_trusted_ips)
            ? stored.approval_trusted_ips.slice()
            : [],
        control_mode: readControlMode({ control_mode: controlMode }),
    }
}
