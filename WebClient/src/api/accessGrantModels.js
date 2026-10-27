// The guest-access grant form.
//
// The server has implemented create / list / delete for time-boxed guest codes
// all along. A grant is an alternative to the device's own rotating access code:
// it lives for a bounded window, may start a bounded number of sessions, and can
// be bound to one controller so a leaked code is useless elsewhere. Neither
// control end has an entry point, so until now the only way to exercise any of it
// was a hand-written request.
//
// Three of the five options are validated here rather than left to the server,
// because each server rejection is correct and arrives too late to be useful: a
// malformed key is only discovered when the guest tries to connect, a bad address
// only when the code is redeemed, and a public key without a client id produces a
// binding that holds while its "who" column stays empty.

// A raw Ed25519 public key is 32 bytes, base64-encoded with the standard
// alphabet - the same shape the server's ParseControllerPublicKey requires.
const PUBLIC_KEY_BYTES = 32

function decodeBase64(value) {
  const cleaned = String(value ?? '').trim()
  if (!cleaned) return null
  // Reject anything outside the standard alphabet rather than letting atob's
  // leniency produce a different byte string than the server will decode.
  if (!/^[A-Za-z0-9+/]+={0,2}$/.test(cleaned)) return null
  try {
    const binary = atob(cleaned)
    if (binary.length !== PUBLIC_KEY_BYTES) return null
    return binary
  } catch {
    return null
  }
}

function parseAddress(entry) {
  const value = String(entry ?? '').trim()
  if (!value) return null
  if (value.includes('/')) {
    // CIDR: four octets, a slash, then a prefix length the octets can express.
    const match = /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})\/(\d{1,2})$/.exec(value)
    if (!match) return null
    const octets = match.slice(1, 5).map(Number)
    if (octets.some((o) => o > 255)) return null
    if (Number(match[5]) > 32) return null
    return value
  }
  const octets = value.split('.')
  if (octets.length !== 4) return null
  if (!octets.every((o) => /^\d{1,3}$/.test(o) && Number(o) <= 255)) return null
  return value
}

export function buildAccessGrantPayload(form = {}) {
  const minutes = Number(form.expiresInMinutes)
  if (!Number.isFinite(minutes) || minutes <= 0) {
    // A grant without a lifetime never expires, which is worse than never
    // issuing it - and a window that already ended is refused by every
    // redemption the guest attempts.
    throw new Error('expires_in_minutes must be a positive number of minutes')
  }

  const payload = { expires_in_minutes: Math.floor(minutes) }

  // 0 is the server's "no cap within the window", and a blank input has to send
  // that rather than a number the user never entered.
  const rawUses = form.maxUses
  if (rawUses !== undefined && rawUses !== null && String(rawUses).trim() !== '') {
    const uses = Number(rawUses)
    if (!Number.isInteger(uses) || uses < 0) {
      throw new Error('max_uses must be zero (no cap) or a positive integer')
    }
    payload.max_uses = uses
  } else {
    payload.max_uses = 0
  }

  const label = String(form.label ?? '').trim()
  if (label) payload.label = label

  const clientId = String(form.boundClientId ?? '').trim()
  const publicKey = String(form.boundClientPublicKey ?? '').trim()

  if (publicKey) {
    if (!decodeBase64(publicKey)) {
      throw new Error('bound_client_public_key must be base64 of a raw 32-byte Ed25519 public key')
    }
    // The key proves possession of a key; without a client id to name the holder
    // the binding would hold while the audit column stayed empty.
    if (!clientId) {
      throw new Error('bound_client_public_key requires bound_client_id')
    }
    payload.bound_client_public_key = publicKey
  }

  if (clientId) payload.bound_client_id = clientId

  const rawIps = Array.isArray(form.boundIps) ? form.boundIps : []
  const addresses = []
  for (const entry of rawIps) {
    const parsed = parseAddress(entry)
    // A dropped entry is not the same as an absent one: the user typed an
    // address, so silently discarding it would narrow the binding without
    // saying so - and the owner would believe a leaked code is useless
    // somewhere it is not.
    if (!parsed) {
      throw new Error(`bound_ips: "${String(entry).trim()}" is not a valid IP address or CIDR block`)
    }
    addresses.push(parsed)
  }
  if (addresses.length > 0) {
    payload.bound_ips = JSON.stringify(addresses)
  }

  return payload
}
