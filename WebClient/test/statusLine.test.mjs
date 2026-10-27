import assert from 'node:assert/strict'
import test from 'node:test'

import { StatusLine, StatusSubject, subjectForKey } from '../js/files/statusLine.js'

// A multi-chunk upload's last progress readout is followed by the refresh that
// upload triggers. Without a rule, the refresh's "loading" and then the item
// count replace it, and the operator never sees confirmation that the file
// finished transferring.
test('the refresh that follows an upload does not replace its progress', () => {
  const line = new StatusLine()

  line.set('report.bin 100%', 'files.uploading')
  assert.equal(line.accepts('files.loading'), false, 'the refresh cleared the progress')
  assert.equal(line.accepts('files.itemCount'), false, 'the listing cleared the progress')

  line.set('report.bin 100%', 'files.uploading')
  assert.equal(line.message, 'report.bin 100%')
})

test('a listing message shows when no transfer is in flight', () => {
  const line = new StatusLine()

  line.set('12 items', 'files.itemCount')
  assert.equal(line.accepts('files.loading'), true)
  assert.equal(line.accepts('files.itemCount'), true)
})

// The readout has to move on eventually: a second upload must be able to report
// its own progress, otherwise the line is stuck on the first file forever.
test('a second transfer replaces the first one', () => {
  const line = new StatusLine()

  line.set('first.bin 100%', 'files.uploading')
  assert.equal(line.accepts('files.uploading'), true)

  line.set('second.bin 40%', 'files.uploading')
  assert.equal(line.message, 'second.bin 40%')
})

// An error is the one message that must always get through, including over a
// progress readout: "the transfer failed" is more important than "the transfer
// is at 40%".
test('an error replaces whatever was showing', () => {
  const line = new StatusLine()

  line.set('big.bin 40%', 'files.uploading')
  line.set('The host refused the chunk', 'files.error', true)

  assert.equal(line.message, 'The host refused the chunk')
  assert.equal(line.error, true)
})

// Downloads are transfers too, and they go through the same refresh.
test('a download progress readout survives the refresh as well', () => {
  const line = new StatusLine()

  line.set('data.bin 100%', 'files.downloaded')
  assert.equal(line.accepts('files.loading'), false)
  assert.equal(line.accepts('files.itemCount'), false)
})

test('every key maps to a known subject', () => {
  const keys = [
    'files.uploading', 'files.downloading', 'files.uploaded', 'files.downloaded',
    'files.loading', 'files.itemCount', 'files.error', 'files.renamed',
  ]
  for (const key of keys) {
    assert.ok(
      Object.values(StatusSubject).includes(subjectForKey(key)),
      `${key} mapped to an unknown subject`,
    )
  }
})
