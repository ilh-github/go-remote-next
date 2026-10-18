import test from 'node:test'
import assert from 'node:assert/strict'

import { uploadRecordingArtifact } from '../js/recording-upload.js'

function recordingResponse(status, offset, recording = {}) {
  return new Response(JSON.stringify({ id: 7, ...recording }), {
    status,
    headers: {
      'Content-Type': 'application/json',
      'Upload-Offset': String(offset),
    },
  })
}

test('recording upload resumes from the authoritative offset after a conflict', async () => {
  const blob = new Blob(['abcdefghij'], { type: 'video/webm' })
  const requests = []
  const responses = [
    recordingResponse(202, 4),
    recordingResponse(409, 6),
    recordingResponse(202, 8),
    recordingResponse(200, 10, { status: 'completed', size_bytes: 10 }),
  ]
  const progress = []

  const saved = await uploadRecordingArtifact({
    url: '/v1/me/recordings/7/artifact',
    token: 'tok',
    blob,
    ext: 'webm',
    chunkSize: 4,
    fetchImpl: async (url, init) => {
      requests.push({
        url,
        method: init.method,
        offset: Number(init.headers['Upload-Offset']),
        length: Number(init.headers['Upload-Length']),
        bodySize: init.body.size,
      })
      return responses.shift()
    },
    onProgress: (state) => progress.push(state.uploaded),
  })

  assert.deepEqual(
    requests.map(({ offset, length, bodySize }) => ({ offset, length, bodySize })),
    [
      { offset: 0, length: 10, bodySize: 4 },
      { offset: 4, length: 10, bodySize: 4 },
      { offset: 6, length: 10, bodySize: 4 },
      { offset: 8, length: 10, bodySize: 2 },
    ],
  )
  assert.deepEqual(progress, [4, 6, 8, 10])
  assert.equal(saved.status, 'completed')
  assert.equal(saved.size_bytes, 10)
})

test('recording upload retries transport errors without advancing its offset', async () => {
  const blob = new Blob(['abcdefgh'], { type: 'video/webm' })
  const requestOffsets = []
  const retries = []
  let attempt = 0

  const saved = await uploadRecordingArtifact({
    url: '/v1/me/recordings/9/artifact',
    token: 'tok',
    blob,
    ext: '.webm',
    chunkSize: 4,
    retryDelaysMs: [0, 0, 0],
    fetchImpl: async (_url, init) => {
      requestOffsets.push(Number(init.headers['Upload-Offset']))
      attempt += 1
      if (attempt === 1) throw new TypeError('connection reset')
      if (attempt === 2) return recordingResponse(202, 4)
      return recordingResponse(200, 8, { status: 'completed', size_bytes: 8 })
    },
    onRetry: (state) => retries.push(state.attempt),
  })

  assert.deepEqual(requestOffsets, [0, 0, 4])
  assert.deepEqual(retries, [1])
  assert.equal(saved.status, 'completed')
})

test('recording upload does not retry a payload rejection', async () => {
  const blob = new Blob(['abcdefgh'], { type: 'video/webm' })
  let calls = 0

  await assert.rejects(
    uploadRecordingArtifact({
      url: '/v1/me/recordings/11/artifact',
      token: 'tok',
      blob,
      ext: 'webm',
      chunkSize: 4,
      fetchImpl: async () => {
        calls += 1
        return new Response(JSON.stringify({ detail: 'artifact too large' }), {
          status: 413,
          headers: { 'Content-Type': 'application/problem+json' },
        })
      },
    }),
    (error) => error.status === 413 && error.message.includes('artifact too large'),
  )

  assert.equal(calls, 1)
})

test('recording upload reports the last transport error after retry exhaustion', async () => {
  const blob = new Blob(['abcdefgh'], { type: 'video/webm' })
  const requestOffsets = []
  const retries = []
  let calls = 0

  await assert.rejects(
    uploadRecordingArtifact({
      url: '/v1/me/recordings/12/artifact',
      token: 'tok',
      blob,
      ext: 'webm',
      chunkSize: 4,
      retryDelaysMs: [0, 0, 0],
      fetchImpl: async (_url, init) => {
        calls += 1
        requestOffsets.push(Number(init.headers['Upload-Offset']))
        throw new TypeError(`connection reset ${calls}`)
      },
      onRetry: (state) => retries.push(state.attempt),
    }),
    /connection reset 3/,
  )

  assert.deepEqual(requestOffsets, [0, 0, 0])
  assert.deepEqual(retries, [1, 2])
})

test('recording upload reports retryable HTTP failure after the final attempt', async () => {
  const blob = new Blob(['abcdefgh'], { type: 'video/webm' })
  let calls = 0

  await assert.rejects(
    uploadRecordingArtifact({
      url: '/v1/me/recordings/13/artifact',
      token: 'tok',
      blob,
      ext: 'webm',
      chunkSize: 4,
      retryDelaysMs: [0, 0],
      fetchImpl: async () => {
        calls += 1
        return new Response(JSON.stringify({ detail: 'gateway unavailable' }), {
          status: 503,
          headers: { 'Content-Type': 'application/problem+json' },
        })
      },
    }),
    (error) => error.status === 503 && error.message.includes('gateway unavailable'),
  )

  assert.equal(calls, 2)
})
