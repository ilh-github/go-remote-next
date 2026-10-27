function trimmed(value) {
  return typeof value === 'string' ? value.trim() : ''
}

function positiveInteger(value) {
  const parsed = Number(value)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : 0
}

// A group cannot contain more than 500 devices. The server is the authority and
// rejects a larger selection; the point of knowing the number here is to say so
// while the operator is still choosing, rather than after a round trip that
// undoes their work.
export const MAX_GROUP_DEVICES = 500

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

/**
 * Describes a membership selection against the ceiling.
 *
 * The count is taken after de-duplication, because that is the number the server
 * will actually be asked to store: charging a repeated device five times would
 * refuse a selection the server would have accepted.
 *
 * @param {Array<string>} values the raw selection
 * @returns {{count: number, overLimit: boolean, excess: number}}
 */
export function describeGroupMembership(values) {
  const ids = normalizeGroupDeviceIds(values)
  const excess = Math.max(0, ids.length - MAX_GROUP_DEVICES)
  return {
    count: ids.length,
    overLimit: excess > 0,
    excess,
  }
}
