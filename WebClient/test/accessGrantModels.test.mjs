// The guest-access grant form.
//
// The server has implemented create/list/delete for time-boxed guest codes all
// along, with five options on the way in: how long it lives, how many sessions
// it may start, and three ways to bind it to a specific controller. Both control
// ends - the web client and the Qt build - have no entry point at all, so the
// only thing that can exercise this is a hand-written request. That is the gap
// these tests are written against.
import assert from 'node:assert/strict'
import test from 'node:test'

const mod = await import('../src/api/accessGrantModels.js')

test('a grant carries its lifetime and its use cap', () => {
  const payload = mod.buildAccessGrantPayload({ expiresInMinutes: 60, maxUses: 3 })
  assert.equal(payload.expires_in_minutes, 60)
  assert.equal(payload.max_uses, 3)
})

test('a lifetime is required, because a grant without one never expires', () => {
  // The server rejects a non-positive lifetime, and a code that outlives its
  // purpose is worse than one that was never issued.
  assert.throws(() => mod.buildAccessGrantPayload({ expiresInMinutes: 0 }), /expires/i)
  assert.throws(() => mod.buildAccessGrantPayload({}), /expires/i)
})

test('an omitted use cap means unlimited within the window', () => {
  // 0 is the server's "no cap", and a blank input must send that rather than a
  // number the user never entered.
  assert.equal(mod.buildAccessGrantPayload({ expiresInMinutes: 30 }).max_uses, 0)
  assert.equal(mod.buildAccessGrantPayload({ expiresInMinutes: 30, maxUses: '' }).max_uses, 0)
})

test('a negative use cap is refused rather than read as unlimited', () => {
  assert.throws(() => mod.buildAccessGrantPayload({ expiresInMinutes: 30, maxUses: -1 }), /max_uses/i)
})

test('a label travels with the grant so the owner can tell codes apart', () => {
  assert.equal(mod.buildAccessGrantPayload({ expiresInMinutes: 30, label: '  repair visit  ' }).label, 'repair visit')
})

test('a blank label is omitted rather than sent empty', () => {
  assert.ok(!('label' in mod.buildAccessGrantPayload({ expiresInMinutes: 30, label: '   ' })))
})

test('a client binding restricts the code to one controller', () => {
  const payload = mod.buildAccessGrantPayload({ expiresInMinutes: 30, boundClientId: 'tab-7' })
  assert.equal(payload.bound_client_id, 'tab-7')
  assert.ok(!('bound_client_id' in mod.buildAccessGrantPayload({ expiresInMinutes: 30 })))
})

test('a public key binding requires a client id to name the holder', () => {
  // The key proves possession; without a client id the "who" column would stay
  // empty while the binding still held, so the two would disagree.
  assert.throws(
    () => mod.buildAccessGrantPayload({ expiresInMinutes: 30, boundClientPublicKey: 'AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8=' }),
    /bound_client_id/i,
  )
})

test('a public key binding travels with its client id', () => {
  const payload = mod.buildAccessGrantPayload({
    expiresInMinutes: 30,
    boundClientId: 'tab-7',
    boundClientPublicKey: 'AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8=',
  })
  assert.equal(payload.bound_client_public_key, 'AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8=')
})

test('a malformed public key is refused before the request goes out', () => {
  // The server rejects a key it cannot parse, and the owner would only find out
  // by handing a useless code to a guest.
  assert.throws(
    () => mod.buildAccessGrantPayload({ expiresInMinutes: 30, boundClientId: 't', boundClientPublicKey: 'not base64 !!' }),
    /bound_client_public_key/i,
  )
})

test('bound addresses are sent as the array the request struct takes', () => {
  // A JSON array, not a stringified one: the server's field is []string, and the
  // text column it eventually lands in is an internal detail. Sending the encoded
  // string fails with a 400 that names the Go type.
  const payload = mod.buildAccessGrantPayload({ expiresInMinutes: 30, boundIps: ['203.0.113.5', '10.0.0.0/8'] })
  assert.deepEqual(payload.bound_ips, ['203.0.113.5', '10.0.0.0/8'])
  assert.ok(!('bound_ips' in mod.buildAccessGrantPayload({ expiresInMinutes: 30, boundIps: [] })))
})

test('a malformed bound address is refused', () => {
  assert.throws(
    () => mod.buildAccessGrantPayload({ expiresInMinutes: 30, boundIps: ['not-an-ip'] }),
    /bound_ips/i,
  )
})

test('a grant that is already expired cannot be created', () => {
  // A window that ends before it starts is a caller mistake; the server would
  // accept it and refuse every redemption.
  assert.throws(() => mod.buildAccessGrantPayload({ expiresInMinutes: -30 }), /expires/i)
})

// The list half. A grant row has to answer three questions the raw API row does
// not: is it still usable right now, how much of its budget is left, and what is
// it bound to. Getting the first one wrong is the dangerous case - a revoked or
// expired code shown as live invites the owner to hand it to a guest who will
// then be refused, with no explanation on either side.
test('a live grant reports as usable with its remaining budget', () => {
  const future = new Date(Date.now() + 3600_000).toISOString()
  const row = mod.grantRow({ id: 1, code: 'AAAA-1111', expires_at: future, max_uses: 3, uses: 1 })
  assert.equal(row.usable, true)
  assert.equal(row.remainingUses, 2)
  assert.equal(row.unlimited, false)
})

test('a grant with no cap reports as unlimited rather than zero left', () => {
  const future = new Date(Date.now() + 3600_000).toISOString()
  const row = mod.grantRow({ id: 2, expires_at: future, max_uses: 0, uses: 9 })
  assert.equal(row.unlimited, true)
  assert.equal(row.usable, true)
})

test('an expired grant is not usable even with budget left', () => {
  const past = new Date(Date.now() - 1000).toISOString()
  const row = mod.grantRow({ id: 3, expires_at: past, max_uses: 5, uses: 0 })
  assert.equal(row.usable, false)
  assert.equal(row.reason, 'expired')
})

test('a revoked grant is not usable, and says so distinctly', () => {
  // Revoked and expired need different words: one is a choice the owner made,
  // the other is time passing.
  const future = new Date(Date.now() + 3600_000).toISOString()
  const row = mod.grantRow({ id: 4, expires_at: future, revoked_at: future })
  assert.equal(row.usable, false)
  assert.equal(row.reason, 'revoked')
})

test('a grant that has spent its budget is not usable', () => {
  const future = new Date(Date.now() + 3600_000).toISOString()
  const row = mod.grantRow({ id: 5, expires_at: future, max_uses: 2, uses: 2 })
  assert.equal(row.usable, false)
  assert.equal(row.reason, 'used_up')
})

test('the binding is summarised for the row, not left as raw JSON', () => {
  // The response sends bound_ips as a real array; this is the shape that matters.
  const future = new Date(Date.now() + 3600_000).toISOString()
  const row = mod.grantRow({
    id: 6,
    expires_at: future,
    bound_client_id: 'tab-7',
    bound_ips: ['203.0.113.5', '10.0.0.0/8'],
  })
  assert.match(row.binding, /tab-7/)
  assert.match(row.binding, /203\.0\.113\.5/)
  assert.match(row.binding, /10\.0\.0\.0\/8/)
})

test('a row whose addresses arrived as a JSON string is still read', () => {
  // A row that has been through a store that flattens the array arrives this
  // way. Refusing it would blank the binding for a grant that does have one.
  const future = new Date(Date.now() + 3600_000).toISOString()
  const row = mod.grantRow({
    id: 11,
    expires_at: future,
    bound_ips: '["203.0.113.5"]',
  })
  assert.match(row.binding, /203\.0\.113\.5/)
})

test('a grant with no binding says anyone holding the code', () => {
  const future = new Date(Date.now() + 3600_000).toISOString()
  assert.match(mod.grantRow({ id: 7, expires_at: future }).binding, /anyone|任何/i)
})

test('a malformed binding does not break the row', () => {
  // One bad row must not take the whole list down with a parse error.
  const future = new Date(Date.now() + 3600_000).toISOString()
  const row = mod.grantRow({ id: 8, expires_at: future, bound_ips: 'not json' })
  assert.equal(row.usable, true)
  assert.ok(typeof row.binding === 'string' && row.binding.length > 0)
})

test('the code is shown, because the owner has to communicate it', () => {
  const future = new Date(Date.now() + 3600_000).toISOString()
  assert.equal(mod.grantRow({ id: 9, code: 'ZZZZ-9999', expires_at: future }).code, 'ZZZZ-9999')
})

test('a missing expiry is treated as unusable, not as never expiring', () => {
  // A row with no expiry cannot be judged live; guessing would show a code as
  // usable that the server may refuse.
  const row = mod.grantRow({ id: 10 })
  assert.equal(row.usable, false)
})

// The row carries its own i18n key rather than leaving the template to build one
// from the machine reason. `used_up` and `grantStateUsedUp` do not compose by any
// rule, so a template that concatenates renders the raw key the moment the two
// drift apart - which is exactly what the first version of this did.
test('every unusable state carries a key the i18n files actually define', () => {
  const past = new Date(Date.now() - 1000).toISOString()
  const future = new Date(Date.now() + 3600_000).toISOString()
  const rows = [
    mod.grantRow({ id: 1 }),
    mod.grantRow({ id: 2, expires_at: future, revoked_at: future }),
    mod.grantRow({ id: 3, expires_at: past }),
    mod.grantRow({ id: 4, expires_at: future, max_uses: 1, uses: 1 }),
  ]
  const keys = rows.map((row) => row.stateKey)
  assert.deepEqual(keys, [
    'sessions.grantStateUnknown',
    'sessions.grantStateRevoked',
    'sessions.grantStateExpired',
    'sessions.grantStateUsedUp',
  ])
  for (const key of keys) {
    assert.match(key, /^sessions\.grantState[A-Z]/)
  }
})

test('a usable row carries no state key, so the template renders nothing', () => {
  const future = new Date(Date.now() + 3600_000).toISOString()
  assert.equal(mod.grantRow({ id: 5, expires_at: future }).stateKey, '')
})
