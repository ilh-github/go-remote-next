import assert from 'node:assert/strict'
import test from 'node:test'

import { runUploadBatch, uploadBatchStatus } from '../js/files/uploadBatch.js'

function files(...names) {
  return names.map((name) => ({ name }))
}

// A batch where the third of four files fails must still upload the fourth, and
// must report that three arrived. Stopping at the first failure would leave the
// operator with a partial directory and no way to know which part is missing.
test('the batch continues past a failure and reports what landed', async () => {
  const attempted = []
  const result = await runUploadBatch(files('a.bin', 'b.bin', 'c.bin', 'd.bin'), (file) => {
    attempted.push(file.name)
    if (file.name === 'c.bin') return Promise.reject(new Error('host refused c.bin'))
    return Promise.resolve()
  })

  assert.deepEqual(attempted, ['a.bin', 'b.bin', 'c.bin', 'd.bin'],
    'the batch stopped at the first failure')
  assert.deepEqual(result.succeeded, ['a.bin', 'b.bin', 'd.bin'])
  assert.equal(result.failure.message, 'host refused c.bin')
})

test('a partial batch reports both halves, not just the count', async () => {
  const result = await runUploadBatch(files('a.bin', 'b.bin', 'c.bin'), (file) => {
    if (file.name === 'b.bin') return Promise.reject(new Error('disk full'))
    return Promise.resolve()
  })

  const status = uploadBatchStatus(result, 3)

  assert.equal(status.key, 'files.uploadPartial')
  assert.equal(status.params.done, 2)
  assert.equal(status.params.total, 3)
  assert.equal(status.params.name, 'disk full')
  assert.equal(status.isError, true, 'a partial failure was not reported as an error')
})

// A clean batch is a plain success: no error styling, no failure text.
test('a clean batch reports the count with no error', async () => {
  const result = await runUploadBatch(files('a.bin', 'b.bin'), () => Promise.resolve())

  const status = uploadBatchStatus(result, 2)

  assert.equal(status.key, 'files.uploadedCount')
  assert.equal(status.params.count, 2)
  assert.equal(status.isError, false)
})

// Nothing landed, so there is no partial success to describe and the caller
// rethrows the original error for the existing error path to render.
test('a batch where nothing landed produces no partial message', async () => {
  const result = await runUploadBatch(files('a.bin', 'b.bin'), () =>
    Promise.reject(new Error('host unreachable')))

  assert.deepEqual(result.succeeded, [])
  assert.equal(uploadBatchStatus(result, 2), null)
})

// The first failure is the one named, because it is the one that explains the
// batch. Naming the last would point at whichever file happened to be last.
test('the first failure is the one reported', async () => {
  const result = await runUploadBatch(files('a.bin', 'b.bin', 'c.bin'), (file) => {
    if (file.name !== 'b.bin') return Promise.reject(new Error(`failed ${file.name}`))
    return Promise.resolve()
  })

  assert.equal(result.failure.message, 'failed a.bin')
})

// A non-Error throw is still reported as a message rather than being dropped.
test('a non-Error throw is still reported', async () => {
  const result = await runUploadBatch(files('a.bin', 'b.bin'), (file) => {
    if (file.name === 'a.bin') return Promise.reject('plain string failure')
    return Promise.resolve()
  })

  const status = uploadBatchStatus(result, 2)
  assert.equal(status.params.name, 'plain string failure')
})

test('an empty selection produces no message at all', async () => {
  const result = await runUploadBatch([], () => Promise.resolve())

  assert.deepEqual(result.succeeded, [])
  assert.equal(uploadBatchStatus(result, 0), null)
})
