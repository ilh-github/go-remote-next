// The audit list's query string.
//
// The server documents six filters for /v1/admin/audit-logs: action, admin,
// username, actor_user_id, date_from and date_to. `username` is the one that
// matches an end user's account as well as an administrator's, in one
// case-insensitive pass - so without it, "what did this account do" has no
// answer on this screen at all, which is the question the screen exists for.
//
// Two things are validated here rather than left to the server, because both
// server answers are technically correct and practically useless:
//   * a blank value is omitted. A server doing a substring match reads "" as
//     "matches everything", which quietly widens the result instead of
//     narrowing it.
//   * a reversed date window is refused. The server answers with an empty
//     page, which sends the operator hunting for rows that cannot exist.
export function auditLogParams(filters = {}) {
  const params = new URLSearchParams()

  const text = (value, key) => {
    const trimmed = String(value ?? '').trim()
    if (trimmed) params.set(key, trimmed)
  }

  text(filters.action, 'action')
  text(filters.admin, 'admin')
  text(filters.username, 'username')

  // el-input binds strings, so the id usually arrives as one. A non-numeric or
  // non-positive value is a caller mistake: the server answers it with a 400
  // rather than treating it as "no filter", so a typo would blank the list.
  const rawActor = filters.actorUserId
  if (rawActor !== undefined && rawActor !== null && String(rawActor).trim() !== '') {
    const actor = Number(String(rawActor).trim())
    if (!Number.isInteger(actor) || actor <= 0) {
      throw new Error('actor_user_id must be a positive integer')
    }
    params.set('actor_user_id', String(actor))
  }

  const from = String(filters.dateFrom ?? '').trim()
  const to = String(filters.dateTo ?? '').trim()
  if (from && to && from > to) {
    throw new Error('date_from must not be later than date_to')
  }
  if (from) params.set('date_from', from)
  if (to) params.set('date_to', to)

  // Pagination rides along so the caller builds one query string, not two.
  if (filters.cursor) params.set('cursor', String(filters.cursor))
  if (filters.limit) params.set('limit', String(filters.limit))
  // The table's page size is the same knob under an older name.
  if (filters.size && !filters.limit) params.set('limit', String(filters.size))

  return params
}
