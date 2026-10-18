const ACCESS_CODE_FAILURE_KEYS = {
  ACCESS_POLICY_DENIED: 'status.accessDenied',
  FORBIDDEN: 'status.accessDenied',
  DEVICE_NOT_OWNED: 'status.accessDenied',
  DEVICE_NOT_FOUND: 'status.deviceNotFound',
  HOST_OFFLINE: 'status.reason.HOST_OFFLINE',
  INVALID_CODE: 'status.invalidAccessCode',
  TOO_MANY_ATTEMPTS: 'status.tooManyAttempts',
  RATE_LIMITED: 'status.tooManyAttempts',
  CONTROLLER_PROOF_REQUIRED: 'status.controllerProofRequired',
  CONTROLLER_PROOF_INVALID: 'status.controllerProofRequired',
}

function verificationCode(result) {
  return String(result?.code || result?.data?.code || '').trim().toUpperCase()
}

/**
 * Convert access-code:verify response into a stable UI failure.
 *
 * Returns null only when the response contains a usable signal token. An
 * approval-pending response is a normal 202, but it is still a non-connected
 * state and must not fall through to "missing parameters".
 */
export function describeAccessCodeFailure(result) {
  if (result?.ok === true && result.data?.signal_token) return null

  if (result?.status === 202 || result?.data?.status === 'pending') {
    return {
      code: 'APPROVAL_PENDING',
      messageKey: 'status.approvalPending',
    }
  }

  const code = verificationCode(result)
  return {
    code: code || 'ACCESS_VERIFY_FAILED',
    messageKey: ACCESS_CODE_FAILURE_KEYS[code] || 'status.accessVerifyFailed',
  }
}
