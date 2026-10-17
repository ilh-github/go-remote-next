import assert from 'node:assert/strict'
import test from 'node:test'

globalThis.localStorage = {
  _values: new Map(),
  getItem(key) {
    return this._values.has(key) ? this._values.get(key) : null
  },
  setItem(key, value) {
    this._values.set(key, String(value))
  },
  removeItem(key) {
    this._values.delete(key)
  },
}

const {
  buildChannelPayload,
  buildMetricAlertPayload,
  buildOfflineAlertPayload,
  buildQuietHoursPayload,
  normalizeChannelList,
  normalizeMetricAlert,
  normalizeOfflineAlert,
  normalizeQuietHours,
  toDatetimeLocalValue,
} = await import('../src/api/notificationModels.js')
const { userApi } = await import('../src/api/userApi.js')

test('channel lists decode the stored events JSON and preserve signing metadata', () => {
  const result = normalizeChannelList({
    items: [{
      id: 7,
      name: 'Ops',
      url: 'https://example.test/hooks/quickdesk',
      events: '["device.offline.sustained","device.access.denied"]',
      enabled: true,
    }],
    supported_events: ['device.offline.sustained'],
    signature_header: 'X-Webhook-Signature',
    signature_scheme: 'sha256=<hmac-sha256 of the raw body>',
  })

  assert.deepEqual(result.items[0].events, [
    'device.offline.sustained',
    'device.access.denied',
  ])
  assert.equal(result.supportedEvents[0], 'device.offline.sustained')
  assert.equal(result.signatureHeader, 'X-Webhook-Signature')
})

test('channel payload trims user input and omits an empty PATCH secret', () => {
  assert.deepEqual(buildChannelPayload({
    name: ' Ops ',
    url: ' https://example.test/hooks/quickdesk ',
    secret: ' ',
    events: [' device.offline.sustained ', 'device.offline.sustained'],
    enabled: 1,
  }, { partial: true }), {
    name: 'Ops',
    url: 'https://example.test/hooks/quickdesk',
    events: ['device.offline.sustained'],
    enabled: true,
  })
})

test('quiet-hours form serializes local datetime controls as RFC3339', () => {
  const view = normalizeQuietHours({
    snoozed_until: '2026-10-17T04:00:00Z',
    quiet_hours: [{ days: [1, 2], start: '22:00', end: '07:00' }],
    timezone: 'Asia/Shanghai',
    active: true,
    active_until: '2026-10-17T04:00:00Z',
  })

  assert.equal(view.quiet_hours[0].days.length, 2)
  assert.deepEqual(buildQuietHoursPayload({
    snoozed_until: '2026-10-17T12:00',
    quiet_hours: view.quiet_hours,
    timezone: ' Asia/Shanghai ',
  }, { timeZoneOffsetMinutes: -480 }), {
    snoozed_until: '2026-10-17T04:00:00.000Z',
    quiet_hours: [{ days: [1, 2], start: '22:00', end: '07:00' }],
    timezone: 'Asia/Shanghai',
  })
})

test('server snooze instants round-trip into datetime-local controls', () => {
  assert.equal(
    toDatetimeLocalValue('2026-10-17T04:00:00Z', { timeZoneOffsetMinutes: -480 }),
    '2026-10-17T12:00',
  )
  assert.equal(toDatetimeLocalValue(null), '')
})

test('offline alert payload applies documented defaults and clears empty windows', () => {
  const view = normalizeOfflineAlert({ device_id: '123' })
  assert.equal(view.after_minutes, 10)
  assert.deepEqual(view.maintenance_windows, [])

  assert.deepEqual(buildOfflineAlertPayload({
    ...view,
    enabled: true,
    after_minutes: '15',
    repeat_minutes: '0',
    maintenance_windows: [],
    maintenance_timezone: '',
  }), {
    enabled: true,
    after_minutes: 15,
    repeat_minutes: 0,
    snoozed_until: null,
    maintenance_windows: [],
    maintenance_timezone: '',
  })
})

test('metric alert rules preserve current readings and serialize threshold settings', () => {
  const view = normalizeMetricAlert({
    device_id: ' device/1 ',
    enabled: true,
    cpu_percent: '85',
    memory_percent: 90,
    disk_percent: 92.5,
    sustained_minutes: '15',
    repeat_minutes: 60,
    snoozed_until: '2026-10-17T04:00:00Z',
    current: {
      captured_at: '2026-10-17T03:55:00Z',
      cpu_percent: 81.2,
      memory_percent: 67,
      disk_percent: 93.1,
      disk_mount: '/data',
    },
    states: {
      cpu: { exceeded_since: '2026-10-17T03:50:00Z', last_alerted_at: null },
      memory: { exceeded_since: null, last_alerted_at: null },
      disk: { exceeded_since: '2026-10-17T03:40:00Z', last_alerted_at: '2026-10-17T03:45:00Z' },
    },
  })

  assert.equal(view.device_id, 'device/1')
  assert.equal(view.cpu_percent, 85)
  assert.equal(view.sustained_minutes, 15)
  assert.equal(view.current.disk_mount, '/data')
  assert.equal(view.states.disk.last_alerted_at, '2026-10-17T03:45:00Z')

  assert.deepEqual(buildMetricAlertPayload({
    enabled: true,
    cpu_percent: '85',
    memory_percent: 90,
    disk_percent: 92.5,
    sustained_minutes: '15',
    repeat_minutes: 60,
    snoozed_until: '2026-10-17T12:00',
  }, { timeZoneOffsetMinutes: -480 }), {
    enabled: true,
    cpu_percent: 85,
    memory_percent: 90,
    disk_percent: 92.5,
    sustained_minutes: 15,
    repeat_minutes: 60,
    snoozed_until: '2026-10-17T04:00:00.000Z',
  })
})

test('notification API methods use owner-scoped v1 paths', async () => {
  const calls = []
  const original = userApi._req
  userApi._req = async (...args) => {
    calls.push(args)
    return { ok: true, data: {} }
  }
  try {
    await userApi.fetchNotificationChannels()
    await userApi.updateNotificationChannel(7, { name: 'Ops' })
    await userApi.fetchNotificationDeliveries(7, 20)
    await userApi.ackNotificationDelivery(7, 11)
    await userApi.retryFailedNotificationDeliveriesForUser({ dry_run: true })
    await userApi.updateNotificationQuietHours({ quiet_hours: [] })
    await userApi.updateDeviceOfflineAlert('device/1', { enabled: true })
    await userApi.fetchDeviceMetricAlert('device/2')
    await userApi.updateDeviceMetricAlert('device/2', { enabled: true })
    await userApi.deleteDeviceMetricAlert('device/2')
  } finally {
    userApi._req = original
  }

  assert.deepEqual(calls, [
    ['GET', '/v1/me/notification-channels'],
    ['PATCH', '/v1/me/notification-channels/7', { name: 'Ops' }],
    ['GET', '/v1/me/notification-channels/7/deliveries?limit=20'],
    ['POST', '/v1/me/notification-channels/7/deliveries/11/ack'],
    ['POST', '/v1/me/notification-channels:retry-failed', { dry_run: true }],
    ['PUT', '/v1/me/notification-quiet-hours', { quiet_hours: [] }],
    ['PUT', '/v1/me/devices/device%2F1/offline-alert', { enabled: true }],
    ['GET', '/v1/me/devices/device%2F2/metric-alert'],
    ['PUT', '/v1/me/devices/device%2F2/metric-alert', { enabled: true }],
    ['DELETE', '/v1/me/devices/device%2F2/metric-alert'],
  ])
})
