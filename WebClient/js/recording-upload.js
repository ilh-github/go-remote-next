const DEFAULT_CHUNK_SIZE = 4 * 1024 * 1024
const DEFAULT_RETRY_DELAYS_MS = [0, 800, 2000, 5000]
const RETRYABLE_STATUS = new Set([429, 500, 502, 503, 504])

export class RecordingUploadError extends Error {
  constructor(message, { status = 0, code = '' } = {}) {
    super(message)
    this.name = 'RecordingUploadError'
    this.status = status
    this.code = code
  }
}

function normalizeExtension(ext) {
  const value = String(ext || '').trim().toLowerCase()
  if (!value) throw new RecordingUploadError('recording upload requires a file extension')
  return value.startsWith('.') ? value : `.${value}`
}

function parseOffset(response) {
  const raw = response.headers.get('Upload-Offset')
  if (raw == null || raw === '') return null
  const value = Number(raw)
  if (!Number.isSafeInteger(value) || value < 0) {
    throw new RecordingUploadError('server returned an invalid Upload-Offset', {
      status: response.status,
    })
  }
  return value
}

async function responseError(response) {
  let detail = ''
  let code = ''
  try {
    const body = await response.json()
    detail = body?.detail || body?.title || ''
    code = body?.code || ''
  } catch {
    // A proxy or gateway may return HTML or an empty body.
  }
  const suffix = detail ? `: ${detail}` : ''
  return new RecordingUploadError(`recording upload failed: HTTP ${response.status}${suffix}`, {
    status: response.status,
    code,
  })
}

function wait(ms) {
  if (!ms) return Promise.resolve()
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function throwIfAborted(signal) {
  if (signal?.aborted) {
    throw new DOMException('recording upload aborted', 'AbortError')
  }
}

async function fetchWithRetry({
  fetchImpl,
  url,
  init,
  retryDelaysMs,
  signal,
  onRetry,
}) {
  const delays = retryDelaysMs.length ? retryDelaysMs : DEFAULT_RETRY_DELAYS_MS
  let lastError = null

  for (let attempt = 0; attempt < delays.length; attempt += 1) {
    throwIfAborted(signal)
    if (delays[attempt] > 0) await wait(delays[attempt])
    try {
      const response = await fetchImpl(url, init)
      if (!RETRYABLE_STATUS.has(response.status) || attempt === delays.length - 1) {
        return response
      }
      lastError = await responseError(response)
      onRetry?.({ attempt: attempt + 1, delayMs: delays[attempt + 1], error: lastError })
    } catch (error) {
      lastError = error
      if (error?.name === 'AbortError' || attempt === delays.length - 1) throw error
      onRetry?.({ attempt: attempt + 1, delayMs: delays[attempt + 1], error })
    }
  }

  throw lastError || new RecordingUploadError('recording upload failed')
}

export async function uploadRecordingArtifact({
  url,
  token,
  blob,
  ext,
  chunkSize = DEFAULT_CHUNK_SIZE,
  fetchImpl = globalThis.fetch,
  retryDelaysMs = DEFAULT_RETRY_DELAYS_MS,
  signal,
  onProgress,
  onRetry,
}) {
  if (!url) throw new RecordingUploadError('recording upload requires a URL')
  if (!token) throw new RecordingUploadError('recording upload requires an access token')
  if (!blob || typeof blob.slice !== 'function' || !Number.isSafeInteger(blob.size)) {
    throw new RecordingUploadError('recording upload requires a Blob')
  }
  if (blob.size <= 0) throw new RecordingUploadError('recording upload cannot be empty')
  if (!Number.isSafeInteger(chunkSize) || chunkSize <= 0) {
    throw new RecordingUploadError('recording upload chunk size must be positive')
  }
  if (typeof fetchImpl !== 'function') {
    throw new RecordingUploadError('recording upload requires fetch')
  }

  const extension = normalizeExtension(ext)
  const total = blob.size
  const requestUrl = `${url}${url.includes('?') ? '&' : '?'}ext=${encodeURIComponent(extension)}`
  let offset = 0

  while (offset < total) {
    throwIfAborted(signal)
    const start = offset
    const end = Math.min(start + chunkSize, total)
    const response = await fetchWithRetry({
      fetchImpl,
      url: requestUrl,
      init: {
        method: 'PATCH',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/octet-stream',
          'Upload-Offset': String(start),
          'Upload-Length': String(total),
        },
        body: blob.slice(start, end),
        signal,
      },
      retryDelaysMs,
      signal,
      onRetry,
    })

    if (response.status === 409) {
      const authoritative = parseOffset(response)
      if (authoritative == null || authoritative > total) {
        throw new RecordingUploadError('server returned an unusable resume offset', {
          status: 409,
        })
      }
      offset = authoritative
      onProgress?.({ uploaded: offset, total, percent: Math.round((offset / total) * 100) })
      continue
    }

    if (!response.ok) throw await responseError(response)

    const acknowledged = parseOffset(response)
    const nextOffset = acknowledged ?? end
    if (nextOffset < start || nextOffset > total) {
      throw new RecordingUploadError('server acknowledged an invalid upload offset', {
        status: response.status,
      })
    }
    offset = nextOffset
    onProgress?.({ uploaded: offset, total, percent: Math.round((offset / total) * 100) })

    if (response.status === 200) {
      try {
        return await response.json()
      } catch {
        throw new RecordingUploadError('server returned an unreadable recording', { status: 200 })
      }
    }
  }

  throw new RecordingUploadError('upload ended without a completed recording')
}
