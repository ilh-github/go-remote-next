// Reading the audit sweep response.
//
// The sweep runs six independent bounds and each reports its own removal count.
// Collapsing them into one "purged N rows" would hide which bound did the work,
// and that is precisely the answer to "why did my history disappear".
//
// The distinction this model exists to keep: a bound that was NOT configured is
// absent from the list, while a bound that ran and removed nothing is listed
// with a zero. "Not set" and "ran, found nothing" are different facts.
function positive(value) {
  const n = Number(value)
  return Number.isFinite(n) && n > 0 ? n : 0
}

export function boundRows(r) {
  const rows = []
  const age = positive(r?.older_than_days)
  if (age > 0) rows.push({ label: `age ${age}d`, value: Number(r.purged) || 0 })

  const actor = positive(r?.max_rows_per_actor)
  if (actor > 0) rows.push({ label: `per-actor ${actor}`, value: Number(r.actor_cap_removed) || 0 })

  const device = positive(r?.max_rows_per_device)
  if (device > 0) rows.push({ label: `per-device ${device}`, value: Number(r.device_cap_removed) || 0 })

  const action = positive(r?.max_rows_per_action)
  if (action > 0) rows.push({ label: `per-action ${action}`, value: Number(r.action_cap_removed) || 0 })

  const table = positive(r?.max_rows)
  if (table > 0) rows.push({ label: `table-wide ${table}`, value: Number(r.cap_removed) || 0 })

  const archive = positive(r?.archive_retention_days)
  if (archive > 0) rows.push({ label: `archive ${archive}d`, value: Number(r.archive_removed) || 0 })

  return rows
}

export function summary(r) {
  const rows = boundRows(r)
  const totalRemoved = rows.reduce((sum, row) => sum + row.value, 0)
  return {
    rows,
    totalRemoved,
    empty: totalRemoved === 0,
    cutoff: r?.cutoff || '',
    moved: Number(r?.archive_moved) || 0,
    deleted: Number(r?.archive_deleted) || 0,
    coldDir: r?.archive_cold_dir ? r.archive_cold_dir : 'not set',
  }
}
