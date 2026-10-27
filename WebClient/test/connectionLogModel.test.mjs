import assert from 'node:assert/strict'
import test from 'node:test'

const { describeConnectionLog } = await import('../src/connectionLogModel.js')

// The connection log is the only place an owner learns why a connection did not
// happen. The API returns device_name, connect_ip and error_msg on every row, so
// a row that shows only the raw device id throws away the one field that
// explains a failure -- and a failed row is exactly the row someone opens this
// list to read.

test('a log row is labelled with the device name when the server sent one', () => {
  const text = describeConnectionLog({ device_id: '700000001', device_name: 'Desk Alpha' })
  assert.match(text.name, /Desk Alpha/)
  // The id stays visible: it is what the owner matches against the device list
  // and what a support conversation is about.
  assert.match(text.detail, /700000001/)
})

test('a row without a device name falls back to the id rather than going blank', () => {
  const text = describeConnectionLog({ device_id: '700000002', device_name: '' })
  assert.equal(text.name, '700000002')
})

test('a failed row carries the reason the server recorded', () => {
  const text = describeConnectionLog({
    device_id: '700000003',
    status: 'failed',
    error_msg: 'access code rejected',
  })
  assert.match(text.reason, /access code rejected/)
})

test('a successful row has no reason to show', () => {
  const text = describeConnectionLog({
    device_id: '700000004',
    status: 'success',
    error_msg: '',
  })
  assert.equal(text.reason, '')
})

// A failure with no recorded reason must not render the word "undefined" or an
// empty parenthesis; the row has to stay readable.
test('a failed row with no recorded reason stays readable', () => {
  const text = describeConnectionLog({
    device_id: '700000005',
    status: 'failed',
    error_msg: '',
  })
  assert.equal(text.reason, '')
  assert.doesNotMatch(text.reason, /undefined|null/)
})

test('duration renders as minutes and seconds, and is absent at zero', () => {
  assert.match(describeConnectionLog({ duration: 125 }).duration, /2m5s/)
  assert.equal(describeConnectionLog({ duration: 0 }).duration, '')
})
