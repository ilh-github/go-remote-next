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
  buildDeviceGroupPayload,
  filterDevicesByGroup,
  normalizeDeviceGroupList,
  normalizeGroupDeviceIds,
} = await import('../src/api/deviceGroupModels.js')
const { userApi } = await import('../src/api/userApi.js')

test('group list normalizes ids, text fields, and de-duplicates membership', () => {
  const result = normalizeDeviceGroupList({
    items: [{
      id: '7',
      user_id: '3',
      name: ' Office ',
      description: ' Headquarters ',
      color: ' #1abc9c ',
      device_ids: ['device-b', ' device-a ', 'device-b', ''],
      device_count: 9,
    }],
  })

  assert.deepEqual(result.items, [{
    id: 7,
    user_id: 3,
    name: 'Office',
    description: 'Headquarters',
    color: '#1abc9c',
    device_ids: ['device-b', 'device-a'],
    device_count: 2,
  }])
})

test('group filtering returns all bound devices or only current members', () => {
  const devices = [
    { device_id: 'device-a', remark: 'Desk' },
    { device_id: 'device-b', remark: 'Lab' },
    { device_id: 'device-c', remark: 'Lobby' },
  ]

  assert.deepEqual(
    filterDevicesByGroup(devices, null).map((device) => device.device_id),
    ['device-a', 'device-b', 'device-c'],
  )
  assert.deepEqual(
    filterDevicesByGroup(devices, {
      id: 7,
      device_ids: ['device-c', 'missing-device', 'device-a'],
    }).map((device) => device.device_id),
    ['device-a', 'device-c'],
  )
})

test('membership normalization trims, de-duplicates, and preserves selection order', () => {
  assert.deepEqual(
    normalizeGroupDeviceIds([' device-b ', 'device-a', 'device-b', '', null]),
    ['device-b', 'device-a'],
  )
  assert.deepEqual(normalizeGroupDeviceIds(null), [])
})

test('group payload trims editable fields for create and patch', () => {
  assert.deepEqual(buildDeviceGroupPayload({
    name: ' Office ',
    description: ' Headquarters ',
    color: ' #1abc9c ',
  }), {
    name: 'Office',
    description: 'Headquarters',
    color: '#1abc9c',
  })
})

test('device group APIs use owner-scoped paths and normalized membership payloads', async () => {
  const calls = []
  const original = userApi._req
  userApi._req = async (...args) => {
    calls.push(args)
    return { ok: true, data: {} }
  }
  try {
    await userApi.fetchDeviceGroups()
    await userApi.createDeviceGroup({
      name: 'Office',
      description: 'HQ',
      color: '#1abc9c',
    })
    await userApi.updateDeviceGroup(7, { name: 'Lab' })
    await userApi.replaceDeviceGroupDevices(7, ['device-b', ' device-a ', 'device-b'])
    await userApi.deleteDeviceGroup(7)
  } finally {
    userApi._req = original
  }

  assert.deepEqual(calls, [
    ['GET', '/v1/me/device-groups'],
    ['POST', '/v1/me/device-groups', {
      name: 'Office',
      description: 'HQ',
      color: '#1abc9c',
    }],
    ['PATCH', '/v1/me/device-groups/7', { name: 'Lab' }],
    ['PUT', '/v1/me/device-groups/7/devices', {
      device_ids: ['device-b', 'device-a'],
    }],
    ['DELETE', '/v1/me/device-groups/7'],
  ])
})
