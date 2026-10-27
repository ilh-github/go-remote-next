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

test('bound addresses are sent as the JSON array the server parses', () => {
  const payload = mod.buildAccessGrantPayload({ expiresInMinutes: 30, boundIps: ['203.0.113.5', '10.0.0.0/8'] })
  assert.deepEqual(JSON.parse(payload.bound_ips), ['203.0.113.5', '10.0.0.0/8'])
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
