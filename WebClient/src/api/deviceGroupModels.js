function trimmed(value) {
  return typeof value === 'string' ? value.trim() : ''
}

function positiveInteger(value) {
  const parsed = Number(value)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : 0
}

export function normalizeGroupDeviceIds(values) {
  if (!Array.isArray(values)) return []
  return [...new Set(values.map(trimmed).filter(Boolean))]
}

export function normalizeDeviceGroupList(data) {
  return {
    items: Array.isArray(data?.items)
      ? data.items.map((row) => {
        const deviceIds = normalizeGroupDeviceIds(row?.device_ids)
        return {
          ...row,
          id: positiveInteger(row?.id),
          user_id: positiveInteger(row?.user_id),
          name: trimmed(row?.name),
          description: trimmed(row?.description),
          color: trimmed(row?.color),
          device_ids: deviceIds,
          device_count: deviceIds.length,
        }
      })
      : [],
  }
}

export function filterDevicesByGroup(devices, group) {
  const rows = Array.isArray(devices) ? devices : []
  if (!group) return rows.slice()
  const memberIds = new Set(normalizeGroupDeviceIds(group.device_ids))
  return rows.filter((device) => memberIds.has(device?.device_id))
}

export function buildDeviceGroupPayload(form) {
  return {
    name: trimmed(form?.name),
    description: trimmed(form?.description),
    color: trimmed(form?.color),
  }
}
