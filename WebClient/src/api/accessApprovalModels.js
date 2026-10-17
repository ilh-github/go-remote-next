const APPROVAL_STATUSES = ['pending', 'approved', 'denied', 'expired', 'consumed']

function number(value) {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : 0
}

export function normalizeApprovalList(data) {
  return Array.isArray(data?.items)
    ? data.items.map((row) => ({
      ...row,
      id: number(row?.id),
      client_id: row?.client_id || '',
      client_ip: row?.client_ip || '',
      source: row?.source || '',
      status: row?.status || '',
    }))
    : []
}

export function normalizeApprovalSummary(data) {
  const byStatus = {}
  for (const status of APPROVAL_STATUSES) {
    byStatus[status] = number(data?.by_status?.[status])
  }
  return {
    since: data?.since || null,
    total: number(data?.total),
    byStatus,
    deniedSources: Array.isArray(data?.denied_sources)
      ? data.denied_sources.map((row) => ({
        clientIp: row?.client_ip || '',
        denials: number(row?.denials),
        lastDeniedAt: row?.last_denied_at || null,
      }))
      : [],
    denialCooldownSeconds: number(data?.denial_cooldown_seconds),
    denialCooldownMultiplier: number(data?.denial_cooldown_multiplier) || 1,
    denialCooldownMaxSeconds: number(data?.denial_cooldown_max_seconds),
  }
}

export { APPROVAL_STATUSES }
