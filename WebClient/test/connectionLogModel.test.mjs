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
// The clear-history confirmation promises to report how many rows went. The
// server already returns it (`DELETE /v1/me/connections` answers `{removed: N}`),
// and the page's own comment says the confirmation uses that number "rather than
// a generic 'cleared'" - but the code shows a fixed string. An operator who
// clears 137 entries is told only "History cleared", which is the same message
// they would get for one.
test('clearing reports how many rows actually went', async () => {
  const { describeClearResult } = await import('../src/connectionLogModel.js')

  const many = describeClearResult({ removed: 137 })
  assert.equal(many.key, 'devices.logsClearedCount',
    'clearing a real number of rows fell back to the generic message')
  assert.deepEqual(many.params, { count: 137 })
})

// Zero is not an error. The operator confirmed a clear, the server answered that
// nothing was there - telling them that is more honest than a generic success,
// and it explains why the list is still empty rather than leaving them to wonder.
test('clearing nothing is reported as nothing, not as a generic success', async () => {
  const { describeClearResult } = await import('../src/connectionLogModel.js')

  const none = describeClearResult({ removed: 0 })
  assert.equal(none.key, 'devices.logsClearedNone',
    'a clear that removed nothing claimed a generic success')
  assert.deepEqual(none.params, {})
})

// A response that lost its count must not turn into "cleared 0" - that would be
// a specific claim about the server's answer when the answer is unknown.
test('a missing count falls back to the generic message', async () => {
  const { describeClearResult } = await import('../src/connectionLogModel.js')

  for (const response of [undefined, null, {}, { removed: null }, { removed: 'many' }]) {
    const described = describeClearResult(response)
    assert.equal(described.key, 'devices.logsCleared',
      `a response without a usable count (${JSON.stringify(response)}) claimed a number`)
  }
})

// The model test above proves the rule; this proves the page obeys it. A correct
// helper nobody calls is the same as no helper, and the wiring is the part that
// rots - the comment promising the count was already there while the code showed
// a fixed string.
test('the devices page shows the count the server returned', async () => {
  const source = await import('node:fs/promises').then(fs =>
    fs.readFile(new URL('../src/views/DevicesPage.vue', import.meta.url), 'utf8'))

  assert.match(source, /describeClearResult\(r\.data\)/,
    'the clear confirmation does not consult the server count')
  assert.match(source, /showToast\(t\(cleared\.key, cleared\.params\)/,
    'the described result never reaches the toast')
  assert.doesNotMatch(source, /showToast\(t\('devices\.logsCleared'\)/,
    'the generic message is still hard-coded')
})
