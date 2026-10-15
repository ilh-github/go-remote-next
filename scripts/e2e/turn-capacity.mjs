// Minimal TURN client harness for capacity + relay-quality checks against a
// coturn server running in `use-auth-secret` mode (the same mode the signaling
// server issues credentials for).
//
// Capacity run (N concurrent allocations):
//   QD_TURN_URL=turn:127.0.0.1:3478 QD_TURN_SECRET=qd-turn-secret-dev \
//   QD_TURN_CLIENTS=50 node turn-capacity.mjs
//
// Relay-quality run (two allocations actually exchange data through the relay):
//   QD_TURN_URL=... QD_TURN_QUALITY=1 QD_TURN_PACKETS=200 QD_TURN_PAYLOAD=1024 \
//   node turn-capacity.mjs
//
// Credentials come from QD_TURN_SECRET (minted locally) or, if unset, from the
// signaling server's GET /v1/ice-config (QD_SIGNALING_URL + QD_API_KEY).
//
// Exit codes: 0 = assertions held, 2 = assertion failed, 3 = no credentials,
// 4 = could not reach the TURN server.

import dgram from 'node:dgram';
import crypto from 'node:crypto';

const MAGIC_COOKIE = 0x2112a442;

// Counts allocations that had to be retried because coturn still had the
// previous 5-tuple in a closing state (see allocate()).
let transientRetries = 0;

// STUN/TURN message types we care about.
const T = {
  Allocate: 0x0003,
  Refresh: 0x0004,
  SendInd: 0x0016,
  DataInd: 0x0017,
  AllocateSuccess: 0x0103,
  AllocateError: 0x0113,
  CreatePermission: 0x0008,
  CreatePermissionSuccess: 0x0108,
  CreatePermissionError: 0x0118,
  RefreshSuccess: 0x0104,
};

const A = {
  Username: 0x0006,
  MessageIntegrity: 0x0008,
  ErrorCode: 0x0009,
  Realm: 0x0014,
  Nonce: 0x0015,
  XorRelayed: 0x0016,
  XorPeer: 0x0012,
  Data: 0x0013,
  RequestedTransport: 0x0019,
  XorMapped: 0x0020,
  Fingerprint: 0x8028,
  Lifetime: 0x000d,
};

// ---------------------------------------------------------------------------
// Credentials
// ---------------------------------------------------------------------------

function mintCredential(secret, suffix = 'quickdesk', ttlSeconds = 86400) {
  const username = `${Math.floor(Date.now() / 1000) + ttlSeconds}:${suffix}`;
  const credential = crypto
    .createHmac('sha1', secret)
    .update(username)
    .digest('base64');
  return { username, credential };
}

async function credentialFromSignaling() {
  const base = process.env.QD_SIGNALING_URL ?? 'http://127.0.0.1:8000';
  const apiKey = process.env.QD_API_KEY ?? 'qd-local-dev-key';
  const res = await fetch(`${base}/v1/ice-config`, { headers: { 'X-API-Key': apiKey } });
  if (!res.ok) {
    throw new Error(`ice-config failed: HTTP ${res.status}`);
  }
  const body = await res.json();
  const turn = (body.ice_servers ?? []).find((s) => (s.urls ?? []).some((u) => u.startsWith('turn:')));
  if (!turn) {
    throw new Error('ice-config returned no turn: server');
  }
  const url = turn.urls.find((u) => u.startsWith('turn:'));
  return { url, username: turn.username, credential: turn.credential, lifetime: body.lifetime_duration };
}

function turnEndpoint(url) {
  // turn:host:port?transport=udp
  const withoutScheme = url.replace(/^turns?:/, '').split('?')[0];
  const [host, port] = withoutScheme.split(':');
  return { host, port: Number(port ?? 3478) };
}

// ---------------------------------------------------------------------------
// STUN/TURN message encoding
// ---------------------------------------------------------------------------

function attr(type, value) {
  const padding = (4 - (value.length % 4)) % 4;
  const buf = Buffer.alloc(4 + value.length + padding);
  buf.writeUInt16BE(type, 0);
  buf.writeUInt16BE(value.length, 2);
  value.copy(buf, 4);
  return buf;
}

function u32(value) {
  const b = Buffer.alloc(4);
  b.writeUInt32BE(value >>> 0, 0);
  return b;
}

/** Long-term credential key: MD5(username ":" realm ":" password). */
function longTermKey(username, realm, password) {
  return crypto.createHash('md5').update(`${username}:${realm}:${password}`).digest();
}

function header(type, length, txId) {
  const buf = Buffer.alloc(20);
  buf.writeUInt16BE(type, 0);
  buf.writeUInt16BE(length, 2);
  buf.writeUInt32BE(MAGIC_COOKIE, 4);
  txId.copy(buf, 8);
  return buf;
}

/** Build a message with a trailing MESSAGE-INTEGRITY when key is given. */
function buildMessage(type, txId, attrs, key = null) {
  const attrsBuf = Buffer.concat(attrs);
  if (!key) {
    return Buffer.concat([header(type, attrsBuf.length, txId), attrsBuf]);
  }
  const integrityLen = 24; // 4-byte attr header + 20-byte HMAC-SHA1
  const length = attrsBuf.length + integrityLen;
  const head = header(type, length, txId);
  const mac = crypto.createHmac('sha1', key).update(Buffer.concat([head, attrsBuf])).digest();
  return Buffer.concat([head, attrsBuf, attr(A.MessageIntegrity, mac)]);
}

function parseMessage(buf) {
  const type = buf.readUInt16BE(0);
  const length = buf.readUInt16BE(2);
  const txId = buf.subarray(8, 20);
  const attrs = [];
  let off = 20;
  const end = Math.min(20 + length, buf.length);
  while (off + 4 <= end) {
    const at = buf.readUInt16BE(off);
    const al = buf.readUInt16BE(off + 2);
    const value = buf.subarray(off + 4, off + 4 + al);
    attrs.push({ type: at, value, offset: off });
    off += 4 + al + ((4 - (al % 4)) % 4);
  }
  return { type, length, txId, attrs, raw: buf };
}

function findAttr(msg, type) {
  return msg.attrs.find((a) => a.type === type);
}

function verifyIntegrity(msg, key) {
  const mi = findAttr(msg, A.MessageIntegrity);
  if (!mi) {
    return false;
  }
  const withMi = Buffer.alloc(20);
  msg.raw.copy(withMi, 0, 0, 20);
  withMi.writeUInt16BE(mi.offset - 20 + 24, 2);
  const mac = crypto
    .createHmac('sha1', key)
    .update(Buffer.concat([withMi, msg.raw.subarray(20, mi.offset)]))
    .digest();
  return crypto.timingSafeEqual(mac, mi.value);
}

function parseXorAddress(value) {
  const family = value[1];
  const port = value.readUInt16BE(2) ^ (MAGIC_COOKIE >>> 16);
  if (family === 0x01) {
    const mask = Buffer.alloc(4);
    mask.writeUInt32BE(MAGIC_COOKIE, 0);
    const addr = Buffer.from(value.subarray(4, 8).map((b, i) => b ^ mask[i]));
    return { address: Array.from(addr).join('.'), port, family: 'ipv4' };
  }
  // IPv6: the mask is the magic cookie followed by the transaction id; we only
  // need this for completeness, the harness runs over IPv4.
  return { address: value.subarray(4, 20).toString('hex'), port, family: 'ipv6' };
}

function xorAddressAttr(type, { address, port }) {
  const parts = address.split('.').map(Number);
  const b = Buffer.alloc(8);
  b[0] = 0;
  b[1] = 0x01;
  b.writeUInt16BE(port ^ (MAGIC_COOKIE >>> 16), 2);
  const mask = Buffer.alloc(4);
  mask.writeUInt32BE(MAGIC_COOKIE, 0);
  for (let i = 0; i < 4; i += 1) {
    b[4 + i] = parts[i] ^ mask[i];
  }
  return attr(type, b);
}

// ---------------------------------------------------------------------------
// Transport: one UDP socket, promises matched by transaction id
// ---------------------------------------------------------------------------

class TurnSocket {
  constructor(host, port) {
    this.socket = dgram.createSocket('udp4');
    this.host = host;
    this.port = port;
    this.waiters = new Map();
    this.indications = [];
    this.indicationHandlers = [];
    this.socket.on('message', (buf) => this.#onMessage(buf));
  }

  #onMessage(buf) {
    let msg;
    try {
      msg = parseMessage(buf);
    } catch {
      return;
    }
    if (msg.type === T.DataInd) {
      this.indications.push(msg);
      for (const h of this.indicationHandlers) {
        h(msg);
      }
      return;
    }
    const key = msg.txId.toString('hex');
    const waiter = this.waiters.get(key);
    if (waiter) {
      this.waiters.delete(key);
      clearTimeout(waiter.timer);
      waiter.resolve(msg);
    }
  }

  send(buf) {
    return new Promise((resolve, reject) => {
      this.socket.send(buf, this.port, this.host, (err) => (err ? reject(err) : resolve()));
    });
  }

  request(type, attrs, key, timeoutMs) {
    const txId = crypto.randomBytes(12);
    const buf = buildMessage(type, txId, attrs, key);
    const idHex = txId.toString('hex');
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.waiters.delete(idHex);
        reject(new Error(`timeout waiting for reply to 0x${type.toString(16)}`));
      }, timeoutMs);
      this.waiters.set(idHex, { resolve, timer });
      this.send(buf).catch((err) => {
        clearTimeout(timer);
        this.waiters.delete(idHex);
        reject(err);
      });
    });
  }

  close() {
    try {
      this.socket.close();
    } catch {
      /* already closed */
    }
  }
}

function errorCode(msg) {
  const e = findAttr(msg, A.ErrorCode);
  if (!e) {
    return null;
  }
  return { code: e.value[2] * 100 + e.value[3], reason: e.value.subarray(4).toString('utf8') };
}

// ---------------------------------------------------------------------------
// Operations
// ---------------------------------------------------------------------------

async function allocate({ host, port, username, password }, timeoutMs = 4000, attempt = 1) {
  const sock = new TurnSocket(host, port);
  const started = process.hrtime.bigint();

  // 1. Unauthenticated Allocate -> expects 401 with REALM + NONCE.
  const challenge = await sock.request(
    T.Allocate,
    [attr(A.RequestedTransport, Buffer.from([17, 0, 0, 0]))],
    null,
    timeoutMs,
  );
  if (challenge.type !== T.AllocateError || errorCode(challenge)?.code !== 401) {
    const err = errorCode(challenge);
    sock.close();
    // coturn keeps a released allocation's 5-tuple in a closing state for about
    // a second. A later run that rebinds the same ephemeral source port hits it
    // and gets 437 "Mismatched allocation: wrong transaction ID" instead of a
    // 401. A real client just retries on a fresh socket, so mirror that here.
    if (err?.code === 437 && attempt < 4) {
      transientRetries += 1;
      await new Promise((r) => setTimeout(r, 40));
      return allocate({ host, port, username, password }, timeoutMs, attempt + 1);
    }
    throw new Error(
      `expected 401 challenge, got type 0x${challenge.type.toString(16)}${err ? ` (${err.code} ${err.reason})` : ''}`,
    );
  }
  const realm = findAttr(challenge, A.Realm).value.toString('utf8');
  const nonce = findAttr(challenge, A.Nonce).value.toString('utf8');
  const key = longTermKey(username, realm, password);

  // 2. Authenticated Allocate.
  const reply = await sock.request(
    T.Allocate,
    [
      attr(A.RequestedTransport, Buffer.from([17, 0, 0, 0])),
      attr(A.Username, Buffer.from(username, 'utf8')),
      attr(A.Realm, Buffer.from(realm, 'utf8')),
      attr(A.Nonce, Buffer.from(nonce, 'utf8')),
    ],
    key,
    timeoutMs,
  );
  const elapsedMs = Number(process.hrtime.bigint() - started) / 1e6;
  if (reply.type !== T.AllocateSuccess) {
    sock.close();
    const err = errorCode(reply);
    throw new Error(`allocate rejected: ${err ? `${err.code} ${err.reason}` : `type 0x${reply.type.toString(16)}`}`);
  }
  const relayed = parseXorAddress(findAttr(reply, A.XorRelayed).value);
  return {
    socket: sock,
    key,
    realm,
    nonce,
    username,
    relayed,
    mapped: parseXorAddress(findAttr(reply, A.XorMapped).value),
    integrityOK: verifyIntegrity(reply, key),
    elapsedMs,
  };
}

async function createPermission(alloc, peer) {
  const reply = await authedRequest(
    alloc,
    T.CreatePermission,
    [xorAddressAttr(A.XorPeer, peer)],
    T.CreatePermissionSuccess,
  );
  if (reply.type !== T.CreatePermissionSuccess) {
    const err = errorCode(reply);
    throw new Error(`create-permission failed: ${err ? `${err.code} ${err.reason}` : `type 0x${reply.type.toString(16)}`}`);
  }
  return reply;
}

/**
 * Send an authenticated request. coturn may rotate the nonce, so a 438 (Stale
 * Nonce) is answered once with the nonce echoed in the error. Without this a
 * follow-up request silently fails and the allocation is never released.
 */
async function authedRequest(alloc, type, extraAttrs, expectSuccess) {
  const attempt = (nonce) =>
    alloc.socket.request(
      type,
      [
        ...extraAttrs,
        attr(A.Username, Buffer.from(alloc.username, 'utf8')),
        attr(A.Realm, Buffer.from(alloc.realm, 'utf8')),
        attr(A.Nonce, Buffer.from(nonce, 'utf8')),
      ],
      alloc.key,
      4000,
    );

  let reply = await attempt(alloc.nonce);
  if (reply.type !== expectSuccess && errorCode(reply)?.code === 438) {
    const fresh = findAttr(reply, A.Nonce);
    if (fresh) {
      alloc.nonce = fresh.value.toString('utf8');
      reply = await attempt(alloc.nonce);
    }
  }
  return reply;
}

function sendData(alloc, peer, payload) {
  // Send indications are not authenticated.
  const txId = crypto.randomBytes(12);
  const buf = buildMessage(
    T.SendInd,
    txId,
    [xorAddressAttr(A.XorPeer, peer), attr(A.Data, payload)],
    null,
  );
  return alloc.socket.send(buf);
}

/**
 * Release an allocation (Refresh with LIFETIME=0). Without this coturn keeps
 * the allocation for its full lifetime, which both inflates the global count
 * and - more subtly - makes a later run that reuses the same ephemeral source
 * port collide with the stale allocation and get `437 Mismatched allocation`.
 */
async function deallocate(alloc) {
  try {
    const reply = await authedRequest(alloc, T.Refresh, [attr(A.Lifetime, u32(0))], T.RefreshSuccess);
    if (reply.type !== T.RefreshSuccess) {
      return false;
    }
    return true;
  } catch {
    return false;
  } finally {
    alloc.socket.close();
  }
}

function percentile(sorted, p) {
  if (sorted.length === 0) {
    return 0;
  }
  const idx = Math.min(sorted.length - 1, Math.max(0, Math.ceil((p / 100) * sorted.length) - 1));
  return sorted[idx];
}

// ---------------------------------------------------------------------------
// Runs
// ---------------------------------------------------------------------------

async function runCapacity({ host, port }, credFor) {
  const clients = Number(process.env.QD_TURN_CLIENTS ?? 50);
  const expectAllocated = process.env.QD_TURN_EXPECT_ALLOCATED
    ? Number(process.env.QD_TURN_EXPECT_ALLOCATED)
    : null;

  process.stdout.write(`opening ${clients} concurrent TURN allocations at ${host}:${port}\n`);
  const started = Date.now();
  const results = await Promise.allSettled(
    // A real deployment mints one credential per device; issuing a distinct
    // username per client keeps coturn's per-user quota out of the measurement.
    Array.from({ length: clients }, (_, i) => allocate({ host, port, ...credFor(i) })),
  );
  const wallMs = Date.now() - started;

  const ok = [];
  const failures = new Map();
  const latencies = [];
  let integrityFailures = 0;
  for (const r of results) {
    if (r.status === 'fulfilled') {
      ok.push(r.value);
      latencies.push(r.value.elapsedMs);
      if (!r.value.integrityOK) {
        integrityFailures += 1;
      }
    } else {
      const reason = String(r.reason?.message ?? r.reason);
      failures.set(reason, (failures.get(reason) ?? 0) + 1);
    }
  }
  latencies.sort((a, b) => a - b);

  const relayed = [...new Set(ok.map((a) => `${a.relayed.address}:${a.relayed.port}`))];
  const released = (await Promise.all(ok.map(deallocate))).filter(Boolean).length;
  console.log(`\n=== TURN capacity ===`);
  console.log(`allocated:   ${ok.length}/${clients}`);
  console.log(`failed:      ${clients - ok.length}`);
  console.log(`wall time:   ${wallMs} ms`);
  console.log(`alloc latency: p50 ${percentile(latencies, 50).toFixed(1)} ms | p95 ${percentile(latencies, 95).toFixed(1)} ms | max ${percentile(latencies, 100).toFixed(1)} ms`);
  console.log(`distinct relayed addresses: ${relayed.length}`);
  console.log(`MESSAGE-INTEGRITY verified on replies: ${ok.length - integrityFailures}/${ok.length}`);
  console.log(`allocations released: ${released}/${ok.length}`);
  console.log(`transient 437 retries: ${transientRetries}`);
  if (failures.size > 0) {
    console.log('failure reasons:');
    for (const [reason, count] of failures) {
      console.log(`  ${count}x ${reason}`);
    }
  }

  if (integrityFailures > 0) {
    console.error(`FAIL: ${integrityFailures} allocation replies failed MESSAGE-INTEGRITY`);
    return 2;
  }
  if (released !== ok.length) {
    console.error(`FAIL: ${ok.length - released} allocations were not released; later runs will collide`);
    return 2;
  }
  if (expectAllocated !== null && ok.length !== expectAllocated) {
    console.error(`FAIL: expected ${expectAllocated} allocations, got ${ok.length}`);
    return 2;
  }
  if (ok.length !== clients) {
    console.error(`FAIL: ${clients - ok.length} allocations did not succeed`);
    return 2;
  }
  console.log('PASS');
  return 0;
}

async function runQuality({ host, port }, credFor) {
  const packets = Number(process.env.QD_TURN_PACKETS ?? 200);
  const payloadSize = Number(process.env.QD_TURN_PAYLOAD ?? 1024);

  console.log(`relay-quality run: ${packets} packets x ${payloadSize} B between two allocations`);
  const a = await allocate({ host, port, ...credFor(0) });
  const b = await allocate({ host, port, ...credFor(1) });
  console.log(`A relayed ${a.relayed.address}:${a.relayed.port}  B relayed ${b.relayed.address}:${b.relayed.port}`);

  // Each side must permit the other's relayed address.
  await createPermission(a, b.relayed);
  await createPermission(b, a.relayed);

  let received = 0;
  let bytes = 0;
  b.socket.indicationHandlers.push((msg) => {
    const data = findAttr(msg, A.Data);
    if (data) {
      received += 1;
      bytes += data.value.length;
    }
  });

  // Prime: some stacks drop the first datagram before the permission settles.
  await sendData(a, b.relayed, Buffer.from('prime'));
  await new Promise((r) => setTimeout(r, 50));

  b.socket.indications.length = 0;
  received = 0;
  bytes = 0;

  const started = Date.now();
  for (let i = 0; i < packets; i += 1) {
    const payload = Buffer.alloc(payloadSize);
    payload.writeUInt32BE(i, 0);
    await sendData(a, b.relayed, payload);
    if (i % 20 === 0) {
      await new Promise((r) => setTimeout(r, 2));
    }
  }
  // Give the relay a moment to drain.
  await new Promise((r) => setTimeout(r, 300));
  const elapsedMs = Date.now() - started;

  const lossPercent = ((packets - received) / packets) * 100;
  const kbps = (bytes * 8) / elapsedMs;

  console.log(`\n=== TURN relay quality ===`);
  console.log(`sent:      ${packets} packets, ${packets * payloadSize} B`);
  console.log(`received:  ${received} packets, ${bytes} B`);
  console.log(`loss:      ${lossPercent.toFixed(1)} %`);
  console.log(`goodput:   ~${kbps.toFixed(0)} kbps over ${elapsedMs} ms`);
  console.log(`A integrity: ${a.integrityOK}  B integrity: ${b.integrityOK}`);

  await Promise.all([deallocate(a), deallocate(b)]);

  const minReceived = Number(process.env.QD_TURN_EXPECT_RECEIVED ?? Math.floor(packets * 0.9));
  if (received < minReceived) {
    console.error(`FAIL: relayed ${received}/${packets}, below the ${minReceived} floor`);
    return 2;
  }
  console.log('PASS');
  return 0;
}

async function main() {
  const secret = process.env.QD_TURN_SECRET;
  let creds;
  if (secret) {
    creds = {
      url: process.env.QD_TURN_URL ?? 'turn:127.0.0.1:3478',
      ...mintCredential(secret, 'quality'),
    };
  } else {
    try {
      creds = await credentialFromSignaling();
    } catch (err) {
      console.error(`no credentials: ${err.message}`);
      return 3;
    }
  }
  const { host, port } = turnEndpoint(creds.url);
  const credFor = (i) => {
    if (!secret) {
      return { username: creds.username, password: creds.credential };
    }
    const c = mintCredential(secret, `client-${i}`);
    return { username: c.username, password: c.credential };
  };
  const opts = { host, port };
  console.log(`credentials: user=${creds.username} (secret ${secret ? 'minted locally' : 'from signaling'})`);

  try {
    return process.env.QD_TURN_QUALITY
      ? await runQuality(opts, credFor)
      : await runCapacity(opts, credFor);
  } catch (err) {
    console.error(`FAIL: ${err.message}`);
    return 4;
  }
}

process.exit(await main());
