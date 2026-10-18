export const DEFAULT_RECONNECT_DELAYS_MS = [0, 1000, 3000, 7000]

const RETRYABLE_ACCESS_FAILURES = new Set([
  'ACCESS_VERIFY_FAILED',
  'HOST_OFFLINE',
  'RATE_LIMITED',
  'TOO_MANY_ATTEMPTS',
])

export class ReconnectExhaustedError extends Error {
  constructor(message, { attempts, cause, retryable = true } = {}) {
    super(message || 'reconnect attempts exhausted')
    this.name = 'ReconnectExhaustedError'
    this.attempts = attempts
    this.cause = cause
    this.retryable = retryable
  }
}

export function isReconnectableAccessFailure(failure) {
  const code = String(failure?.code || '').trim().toUpperCase()
  return RETRYABLE_ACCESS_FAILURES.has(code)
}

export function nonRetryableReconnectError(message, cause) {
  const error = new Error(message || 'reconnect cannot be retried')
  error.retryable = false
  error.cause = cause
  error.code = cause?.code
  error.messageKey = cause?.messageKey
  return error
}

export async function runReconnectAttempts(attempt, options = {}) {
  const delays = Array.isArray(options.delaysMs) && options.delaysMs.length
    ? options.delaysMs
    : DEFAULT_RECONNECT_DELAYS_MS
  const onAttempt = options.onAttempt || (() => {})
  const onFailure = options.onFailure || (() => {})
  const wait = options.wait || ((ms) => new Promise((resolve) => setTimeout(resolve, ms)))

  let lastError = null
  for (let index = 0; index < delays.length; index += 1) {
    const attemptNumber = index + 1
    const delayMs = Math.max(0, Number(delays[index]) || 0)
    if (delayMs > 0) await wait(delayMs)
    onAttempt({ attempt: attemptNumber, total: delays.length, delayMs })
    try {
      return await attempt({ attempt: attemptNumber, total: delays.length })
    } catch (error) {
      lastError = error
      const retryable = error?.retryable !== false
      onFailure({ attempt: attemptNumber, total: delays.length, error, retryable })
      if (!retryable) {
        throw new ReconnectExhaustedError(error?.message, {
          attempts: attemptNumber,
          cause: error,
          retryable: false,
        })
      }
    }
  }

  throw new ReconnectExhaustedError(lastError?.message, {
    attempts: delays.length,
    cause: lastError,
    retryable: true,
  })
}
