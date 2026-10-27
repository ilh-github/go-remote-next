import assert from 'node:assert/strict'
import test from 'node:test'

const { describeConnectionLog } = await import('../src/connectionLogModel.js')

// QD-206 promises connection history *and* quick connect. The history list is
// the one an owner opens to find "the machine I was on earlier", and until now
// it had no way back: the device list and the favourites card both had a connect
// button, the history rows had none.
//
// The access code cannot come from the history row -- the API deliberately does
// not store it, which is the right call and matches upstream issue #18's
// "do NOT persist verification-code". So the row is joined against the device
// list on device_id, which does return the code.

test('a history row for a bound device can be reconnected', () => {
  const devices = [{ device_id: '700000001', access_code: 'code-a' }]
  const row = describeConnectionLog(
    { device_id: '700000001', status: 'success' },
    devices,
  )
  assert.equal(row.canConnect, true)
  assert.equal(row.accessCode, 'code-a')
})

test('a history row for a device no longer bound cannot be reconnected', () => {
  const devices = [{ device_id: '700000009', access_code: 'code-z' }]
  const row = describeConnectionLog(
    { device_id: '700000002', status: 'success' },
    devices,
  )
  assert.equal(row.canConnect, false)
  assert.equal(row.accessCode, '')
})

// A device that is present but carries no code is not connectable either: the
// connect button has to verify a code, so offering it would only produce a
// "no access code" toast.
test('a bound device without an access code is not offered a connect button', () => {
  const devices = [{ device_id: '700000003', access_code: '' }]
  const row = describeConnectionLog(
    { device_id: '700000003', status: 'success' },
    devices,
  )
  assert.equal(row.canConnect, false)
})

// The join is on device_id, so a row whose id matches nothing must not silently
// borrow a neighbour's code.
test('the join matches on device id only', () => {
  const devices = [
    { device_id: '700000010', access_code: 'first' },
    { device_id: '700000011', access_code: 'second' },
  ]
  assert.equal(
    describeConnectionLog({ device_id: '700000011' }, devices).accessCode,
    'second',
  )
})

test('a history row with no device list at all is still described, just not connectable', () => {
  const row = describeConnectionLog({ device_id: '700000012', status: 'failed' })
  assert.equal(row.canConnect, false)
  assert.equal(row.name, '700000012')
})
