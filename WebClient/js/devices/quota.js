/**
 * devices/quota.js - what the account's device limit means for the device list.
 *
 * QD-105 carries two halves: devices are isolated per owner (every /v1/me route is
 * owner-scoped, which has always held), and the account has a device quota. The
 * quota column existed and was admin-editable from the first day, but nothing
 * read it, so the number in the admin console was decorative. The server now
 * refuses a bind past the limit; this module is the client half that lets the
 * owner see the limit before they hit it.
 *
 * A quota of 0 means "no limit configured" - that is the default every account is
 * created with, and rendering it as "0 of 0 devices" would tell a user with three
 * working machines that they are over a limit that does not exist.
 */

/**
 * The account's device limit, or null when there is none.
 *
 * @param {{device_count?: number}|null|undefined} user the /v1/me user object
 * @returns {number|null}
 */
export function deviceQuota(user) {
  const raw = Number(user?.device_count)
  if (!Number.isInteger(raw) || raw <= 0) return null
  return raw
}

/**
 * How many more machines this account can bind, or null when unlimited.
 *
 * @param {{device_count?: number}|null|undefined} user the /v1/me user object
 * @param {number} owned devices the account currently owns
 * @returns {number|null}
 */
export function deviceQuotaRemaining(user, owned) {
  const quota = deviceQuota(user)
  if (quota === null) return null
  const count = Number.isInteger(owned) && owned > 0 ? owned : 0
  return Math.max(0, quota - count)
}

/**
 * The device list's view of the limit: whether there is one, how many machines
 * are owned, and how many slots are left. The owner is the person who has to act
 * when a bind is refused, so all three have to be available together.
 *
 * @param {{device_count?: number}|null|undefined} user the /v1/me user object
 * @param {number} owned devices the account currently owns
 * @returns {{limited: boolean, owned: number, quota: number|null, remaining: number|null}}
 */
export function deviceQuotaSummary(user, owned) {
  const quota = deviceQuota(user)
  const count = Number.isInteger(owned) && owned > 0 ? owned : 0
  return {
    limited: quota !== null,
    owned: count,
    quota,
    remaining: deviceQuotaRemaining(user, count),
  }
}

/**
 * Whether a bind would be refused before it is attempted. The device list uses
 * this to say so up front rather than letting the owner discover it by watching a
 * machine fail to appear.
 *
 * @param {{device_count?: number}|null|undefined} user the /v1/me user object
 * @param {number} owned devices the account currently owns
 * @returns {boolean}
 */
export function deviceQuotaReached(user, owned) {
  const remaining = deviceQuotaRemaining(user, owned)
  return remaining !== null && remaining === 0
}
