import assert from 'node:assert/strict'

import { uploadRecordingArtifact } from '../../WebClient/js/recording-upload.js'

const baseUrl = (process.env.QD_BASE_URL || '').replace(/\/+$/, '')
if (!baseUrl) {
  console.error('QD_BASE_URL is required')
  process.exit(2)
}

async function jsonRequest(path, { method = 'GET', token = '', body } = {}) {
  const response = await fetch(`${baseUrl}${path}`, {
    method,
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  const text = await response.text()
  let payload = null
  try {
    payload = text ? JSON.parse(text) : null
  } catch {
    payload = text
  }
  if (!response.ok) {
    throw new Error(`${method} ${path} failed: HTTP ${response.status} ${text}`)
  }
  return payload
}

const suffix = `${Date.now()}-${process.pid}`
const registration = await jsonRequest('/v1/auth/register', {
  method: 'POST',
  body: {
    username: `recording-upload-${suffix}`,
    password: `Recording-${suffix}!`,
  },
})
const token = registration.access_token
assert.ok(token, 'registration returned no access token')

const row = await jsonRequest('/v1/me/recordings', {
  method: 'POST',
  token,
  body: {
    device_id: 'recording-upload-e2e',
    session_id: `recording-upload-${suffix}`,
    status: 'recording',
    format: 'webm',
  },
})

const chunkSize = 64 * 1024
const bytes = Buffer.allocUnsafe(chunkSize * 3 + 17)
for (let index = 0; index < bytes.length; index += 1) bytes[index] = index % 251
const blob = new Blob([bytes], { type: 'video/webm' })
const progress = []

const saved = await uploadRecordingArtifact({
  url: `${baseUrl}/v1/me/recordings/${row.id}/artifact`,
  token,
  blob,
  ext: 'webm',
  chunkSize,
  onProgress: ({ uploaded }) => progress.push(uploaded),
})

assert.equal(saved.status, 'completed')
assert.equal(saved.size_bytes, bytes.length)
assert.equal(progress.at(-1), bytes.length)
assert.ok(progress.length >= 4, `expected multiple upload chunks, got ${progress.length}`)

const fresh = await jsonRequest(`/v1/me/recordings/${row.id}`, { token })
assert.equal(fresh.status, 'completed')
assert.equal(fresh.size_bytes, bytes.length)

const download = await fetch(`${baseUrl}/v1/me/recordings/${row.id}/artifact`, {
  headers: { Authorization: `Bearer ${token}` },
})
assert.equal(download.status, 200)
const downloaded = Buffer.from(await download.arrayBuffer())
assert.deepEqual(downloaded, bytes)

await jsonRequest(`/v1/me/recordings/${row.id}`, { method: 'DELETE', token })
console.log(`recording upload smoke passed: id=${row.id} bytes=${bytes.length} chunks=${progress.length}`)
