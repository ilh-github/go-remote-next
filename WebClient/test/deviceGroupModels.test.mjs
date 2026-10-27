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
  MAX_GROUP_DEVICES,
  describeGroupMembership,
} = await import('../src/api/deviceGroupModels.js')
const { userApi } = await import('../src/api/userApi.js')

// The membership ceiling is the server's rule - a group cannot contain more than
// 500 devices - and the save path already surfaces it correctly when the server
// refuses. What was missing is the selection side: nothing told the operator they
// had chosen too many until after the round trip, so they picked 501 devices,
// waited for the request, and got a rejection that undid their work.
//
// A model that knows the ceiling can say so while they are still choosing.
test('a selection over the ceiling is described before it is sent', () => {
  const overBy = 7
  const ids = Array.from({ length: MAX_GROUP_DEVICES + overBy }, (_, i) => `device-${i}`)

  const described = describeGroupMembership(ids)

  assert.equal(described.overLimit, true, 'a selection over the ceiling was not reported')
  assert.equal(described.excess, overBy,
    'the operator was not told how many devices they are over by')
  assert.equal(described.count, MAX_GROUP_DEVICES + overBy)
})

test('a selection at the ceiling is not over it', () => {
  // The boundary has to be exact in both directions: 500 is fine and 501 is not,
  // and an off-by-one here either blocks a legal selection or lets an illegal
  // one through to a guaranteed rejection.
  const atLimit = Array.from({ length: MAX_GROUP_DEVICES }, (_, i) => `device-${i}`)

  const described = describeGroupMembership(atLimit)

  assert.equal(described.overLimit, false)
  assert.equal(described.excess, 0)
})

test('duplicates do not count toward the ceiling', () => {
  // De-duplication happens before the count matters: selecting the same device
  // five times is one device, and charging it five times would refuse a
  // selection the server would have accepted.
  const ids = []
  for (let i = 0; i < MAX_GROUP_DEVICES; i += 1) ids.push('device-0', `device-${i}`)

  const described = describeGroupMembership(ids)

  assert.equal(described.overLimit, false,
    'duplicates were charged against the ceiling')
  assert.equal(described.count, MAX_GROUP_DEVICES)
})

// The model test above proves the rule; this one proves the page obeys it. A
// ceiling nobody consults is the same as no ceiling, and the wiring is the part
// that rots.
test('the devices page disables the save and warns while the selection is over', async () => {
  const source = await import('node:fs/promises').then(fs =>
    fs.readFile(new URL('../src/views/DevicesPage.vue', import.meta.url), 'utf8'))

  assert.match(source, /groupMembership\.overLimit/,
    'the page never consults the membership description')
  assert.match(source, /:disabled="groupDialog\.savingMembers \|\| groupMembership\.overLimit"/,
    'the save button is not disabled while the selection is over the ceiling')
  assert.match(source, /devices\.groups\.tooManyMembers/,
    'the operator is not told why the save is disabled')
})

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
