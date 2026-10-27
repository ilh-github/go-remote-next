// Byte formatting shared by the storage panels.
//
// Split out of UpdateReleasesPage because a second panel now shows the same
// kind of numbers, and because zero is not one case: a zero quota means
// "unlimited" and a zero remainder means "nothing left", and a formatter that
// collapses both to a dash loses the only distinction that matters.
const UNITS = ['B', 'KB', 'MB', 'GB']

export function formatSize(value) {
  if (!Number.isFinite(value) || value < 0) return '-'
  if (value === 0) return '0 B'
  let size = value
  let unit = 0
  while (size >= 1024 && unit < UNITS.length - 1) {
    size /= 1024
    unit += 1
  }
  return `${size.toFixed(unit ? 1 : 0)} ${UNITS[unit]}`
}

// A zero quota is "unlimited" in every storage endpoint, never "no space".
export function quotaLabel(value) {
  return value > 0 ? formatSize(value) : 'unlimited'
}

export function retentionLabel(days) {
  if (!Number.isFinite(days) || days <= 0) return 'off'
  return `${days} d`
}

export function staleLabel(minutes) {
  if (!Number.isFinite(minutes) || minutes <= 0) return 'not set'
  return `${minutes} min`
}
