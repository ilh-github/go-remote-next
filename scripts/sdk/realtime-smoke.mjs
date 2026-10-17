#!/usr/bin/env node
// Conformance smoke for SignalingServer/api/asyncapi.json (the two realtime
// WebSockets).
//
// The OpenAPI smoke pins the HTTP contract by asserting documented fields
// against a running server; this does the same for the WebSocket handshakes and
// the event envelope. It deliberately reads the AsyncAPI document and drives
// its assertions from it - the expected frame order comes from
// `x-server-handshake`, the event catalogue from `EventType.enum`, the snapshot
// device fields from `SnapshotDevice.required` - so the document cannot rot
// unnoticed while a hand-written copy of it stays green.
//
// Usage:
//   QD_IDENTIFIER=user QD_PASSWORD=secret scripts/sdk/realtime-smoke.mjs
//   QD_IDENTIFIER=user QD_PASSWORD=secret QD_SMOKE_DEVICE=126735214 scripts/sdk/realtime-smoke.mjs
//
// Env:
//   QD_SERVER       default http://127.0.0.1:8000 (ws:// is derived from it)
//   QD_API_KEY      default qd-local-dev-key (HTTP only; the WS routes take no key)
//   QD_IDENTIFIER / QD_PASSWORD   required
//   QD_SMOKE_DEVICE                optional; makes the script trigger a real event
//   QD_SMOKE_REMOTE_SESSION        set to 1 to drive a real client signaling
//                                  session; the device must be online
//   QD_SMOKE_ACCESS_CODE           access code for that session, default 135790
//
// Needs Node 22+: it uses the built-in WebSocket, so the script stays
// dependency-free the way the HTTP smoke is.
//
// Exit codes: 0 = the documented shapes hold, 2 = missing environment or an old
// Node, 3 = a documented shape is missing or mistyped, 4 = the server could not
// be reached or behaved unexpectedly.

import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const contractPath = join(here, '..', '..', 'SignalingServer', 'api', 'asyncapi.json');
const contract = JSON.parse(readFileSync(contractPath, 'utf8'));

const server = (process.env.QD_SERVER || 'http://127.0.0.1:8000').replace(/\/$/, '');
const wsBase = server.replace(/^http/, 'ws');
const apiKey = process.env.QD_API_KEY || 'qd-local-dev-key';
const identifier = process.env.QD_IDENTIFIER;
const password = process.env.QD_PASSWORD;
const smokeDevice = process.env.QD_SMOKE_DEVICE || '';

if (!identifier || !password) {
  console.error('QD_IDENTIFIER and QD_PASSWORD are required');
  process.exit(2);
}

if (typeof WebSocket !== 'function') {
  // Node only exposes a global WebSocket from 22 (21 behind a flag). Saying so
  // beats a ReferenceError from deep inside the first connection attempt.
  console.error(`this smoke needs Node 22+ for the built-in WebSocket; running ${process.version}`);
  process.exit(2);
}

let checks = 0;
const failures = [];
const check = (condition, message) => {
  checks += 1;
  if (!condition) failures.push(message);
};
const field = (obj, name, type, where) => {
  const present = obj !== null && typeof obj === 'object' && name in obj;
  check(present, `${where}: missing documented field "${name}"`);
  if (!present) return undefined;
  const value = obj[name];
  if (type && typeof value !== type) {
    check(false, `${where}: field "${name}" is ${typeof value}, contract says ${type}`);
  }
  return value;
};

const fatal = (where, detail) => {
  console.error(`${where}: ${detail}`);
  process.exit(4);
};

// --- contract plumbing -------------------------------------------------------

const schemas = contract?.components?.schemas ?? {};
const eventsChannel = contract?.channels?.events ?? {};
const signalChannel = contract?.channels?.signal ?? {};
const eventTypes = schemas.EventType?.enum ?? [];

function handshakeFor(channel, name) {
  const steps = channel['x-server-handshake'] ?? [];
  return steps.find((step) => step.replace(/\?$|\*$/, '') === name);
}
const optionalStep = (step) => step.endsWith('?');
const repeatedStep = (step) => step.endsWith('*');

// --- websocket plumbing ------------------------------------------------------

// A tiny frame queue: the server writes its bootstrap frames back to back, so a
// listener that only resolves one waiter at a time would drop the rest.
function openChannel(path) {
  const ws = new WebSocket(`${wsBase}${path}`);
  const queue = [];
  const waiters = [];
  let closed = null;

  const deliver = (frame) => {
    const waiter = waiters.shift();
    if (waiter) waiter.resolve(frame);
    else queue.push(frame);
  };

  ws.addEventListener('message', (event) => {
    const raw = typeof event.data === 'string' ? event.data : Buffer.from(event.data).toString('utf8');
    let frame = null;
    try {
      frame = JSON.parse(raw);
    } catch {
      frame = { type: '<unparseable>', raw };
    }
    deliver(frame);
  });
  ws.addEventListener('close', (event) => {
    closed = { code: event.code, reason: event.reason };
    while (waiters.length > 0) waiters.shift().reject(new Error(`closed with ${event.code}`));
  });
  ws.addEventListener('error', () => {
    closed = closed ?? { code: 0, reason: 'transport error' };
  });

  return {
    ws,
    closed: () => closed,
    send(frame) {
      ws.send(JSON.stringify(frame));
    },
    next(timeoutMs = 5000) {
      if (queue.length > 0) return Promise.resolve(queue.shift());
      return new Promise((resolve, reject) => {
        const timer = setTimeout(() => reject(new Error('timed out waiting for a frame')), timeoutMs);
        waiters.push({
          resolve: (frame) => {
            clearTimeout(timer);
            resolve(frame);
          },
          reject: (err) => {
            clearTimeout(timer);
            reject(err);
          },
        });
      });
    },
    close() {
      try {
        ws.close();
      } catch {
        // already gone
      }
    },
  };
}

async function openAuthenticated(where) {
  const channel = openChannel(eventsChannel.address);
  try {
    await new Promise((resolve, reject) => {
      const timer = setTimeout(() => reject(new Error('open timeout')), 5000);
      channel.ws.addEventListener('open', () => {
        clearTimeout(timer);
        resolve();
      });
      channel.ws.addEventListener('error', () => {
        clearTimeout(timer);
        reject(new Error('connect failed'));
      });
    });
  } catch (err) {
    fatal(where, err.message);
  }
  return channel;
}

// --- HTTP helpers ------------------------------------------------------------

async function call(method, path, body, token) {
  const headers = { 'X-API-Key': apiKey };
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  if (token) headers.Authorization = `Bearer ${token}`;
  let res;
  try {
    res = await fetch(server + path, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch (err) {
    // A transport failure is "the server could not be reached" (exit 4), not a
    // contract violation - and an unhandled rejection would exit 1 with a stack
    // trace, which is neither.
    fatal(`${method} ${path}`, err.message);
  }
  const text = await res.text();
  let json = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    // non-JSON bodies only matter where the caller expects JSON
  }
  return { status: res.status, json };
}

// --- 1. the contract itself --------------------------------------------------

check(handshakeFor(eventsChannel, 'auth_ok') !== undefined,
  'asyncapi.json: the events channel does not document an auth_ok handshake step');
check(handshakeFor(eventsChannel, 'bootstrap_done') !== undefined,
  'asyncapi.json: the events channel does not document a bootstrap_done step');
check(handshakeFor(signalChannel, 'auth_ok') !== undefined,
  'asyncapi.json: the signal channel does not document an auth_ok step');
check(eventTypes.length > 0, 'asyncapi.json: EventType.enum is empty');
check((schemas.EventEnvelope?.required ?? []).includes('type'),
  'asyncapi.json: EventEnvelope does not require `type`');
const snapshotDeviceRequired = schemas.SnapshotDevice?.required ?? [];
check(snapshotDeviceRequired.length > 0,
  'asyncapi.json: SnapshotDevice documents no required fields');

// --- 2. the session token the events channel authenticates with --------------

const login = await call('POST', '/v1/auth/sessions', { identifier, password });
if (login.status !== 200) fatal('POST /v1/auth/sessions', `expected 200, got ${login.status}`);
const accessToken = login.json?.access_token;
if (typeof accessToken !== 'string' || accessToken === '') {
  fatal('POST /v1/auth/sessions', 'the login response carried no access_token');
}

// --- 3. events channel: bootstrap -------------------------------------------

const events = await openAuthenticated('GET /v1/realtime/events');
events.send({ type: 'auth', access_token: accessToken });

const authOK = await events.next().catch((err) => fatal('events handshake', err.message));
check(authOK.type === 'auth_ok', `events: first frame is "${authOK.type}", documented first frame is auth_ok`);
check(typeof authOK.server_rev === 'number', 'events: auth_ok has no numeric server_rev');
check(typeof authOK.data?.family_id === 'string' && authOK.data.family_id !== '',
  'events: auth_ok.data.family_id is missing - the stream cannot say which session family it represents');

const snapshotRequired = handshakeFor(eventsChannel, 'snapshot_required');
let frame = await events.next().catch((err) => fatal('events bootstrap', err.message));
if (snapshotRequired !== undefined && optionalStep(snapshotRequired) && frame.type === 'snapshot_required') {
  check(typeof frame.server_rev === 'number', 'events: snapshot_required has no numeric server_rev');
  frame = await events.next().catch((err) => fatal('events bootstrap', err.message));
}
check(frame.type === 'snapshot', `events: bootstrap frame is "${frame.type}", documented sequence has snapshot`);
if (frame.type === 'snapshot') {
  check(Array.isArray(frame.data?.devices), 'events: snapshot.data.devices is not an array');
  check(Array.isArray(frame.data?.favorites), 'events: snapshot.data.favorites is not an array');
  for (const device of frame.data?.devices ?? []) {
    for (const name of snapshotDeviceRequired) {
      field(device, name, undefined, 'snapshot devices[]');
    }
    check(typeof device.online === 'boolean', 'snapshot devices[]: online is not a boolean');
    check(typeof device.logged_in === 'boolean', 'snapshot devices[]: logged_in is not a boolean');
  }
}
const bootstrapDone = await events.next().catch((err) => fatal('events bootstrap', err.message));
check(bootstrapDone.type === 'bootstrap_done',
  `events: frame after the snapshot is "${bootstrapDone.type}", documented sequence ends the bootstrap with bootstrap_done`);

// --- 4. events channel: a real event ----------------------------------------

// A policy event alone does not prove the session lifecycle reaches the events
// stream. When enabled, authenticate a real client signaling socket and verify
// that its open/close transitions arrive with the same session id.
let remoteSessionChecked = false;
if (smokeDevice && process.env.QD_SMOKE_REMOTE_SESSION === '1') {
  const accessCode = process.env.QD_SMOKE_ACCESS_CODE || '135790';
  const clientId = `realtime-smoke-${Date.now()}`;
  const verified = await call('POST',
    `/v1/devices/${smokeDevice}/access-code:verify`, {
      code: accessCode,
      client_id: clientId,
    });
  check(verified.status === 200,
    `POST access-code:verify: expected 200, got ${verified.status}`);
  const signalToken = verified.json?.signal_token;
  check(typeof signalToken === 'string' && signalToken !== '',
    'POST access-code:verify returned no signal_token');

  if (verified.status === 200 && typeof signalToken === 'string' && signalToken) {
    const sessionSignal = openChannel(signalChannel.address);
    await new Promise((resolve, reject) => {
      sessionSignal.ws.addEventListener('open', resolve);
      sessionSignal.ws.addEventListener('error', () => reject(new Error('connect failed')));
    }).catch((err) => fatal('signal (remote session)', err.message));
    sessionSignal.send({
      type: 'auth',
      signal_token: signalToken,
      role: 'client',
      device_id: smokeDevice,
      client_id: clientId,
    });
    const signalAuth = await sessionSignal.next()
      .catch((err) => fatal('signal (remote session)', err.message));
    check(signalAuth.type === 'auth_ok',
      `signal (remote session): first frame is "${signalAuth.type}", want auth_ok`);
    const sessionId = signalAuth.session_id || '';
    check(sessionId !== '', 'signal (remote session): auth_ok carried no session_id');

    let started = null;
    for (let i = 0; i < 15 && started === null; i++) {
      const candidate = await events.next(1000).catch(() => null);
      if (candidate?.type === 'remote_session.started' &&
          candidate.data?.session_id === sessionId) {
        started = candidate;
      }
    }
    check(started !== null,
      'events: no remote_session.started arrived for the real client socket');
    if (started !== null) {
      check(started.data?.device_id === smokeDevice,
        `events: remote_session.started device_id is "${started.data?.device_id}"`);
      check(started.data?.client_id === clientId,
        `events: remote_session.started client_id is "${started.data?.client_id}"`);
      check(started.data?.active === true,
        'events: remote_session.started active is not true');
    }

    sessionSignal.close();

    let ended = null;
    for (let i = 0; i < 15 && ended === null; i++) {
      const candidate = await events.next(1000).catch(() => null);
      if (candidate?.type === 'remote_session.ended' &&
          candidate.data?.session_id === sessionId) {
        ended = candidate;
      }
    }
    check(ended !== null,
      'events: no remote_session.ended arrived after the client socket closed');
    if (ended !== null) {
      check(ended.data?.device_id === smokeDevice,
        `events: remote_session.ended device_id is "${ended.data?.device_id}"`);
      check(ended.data?.end_reason === 'disconnected',
        `events: remote_session.ended end_reason is "${ended.data?.end_reason}"`);
      check(ended.data?.active === false,
        'events: remote_session.ended active is not false');
    }
    remoteSessionChecked = true;
  }
}

// Nothing is published by an idle server, so the smoke makes one happen. A
// recording is created and deleted again, which publishes recording.created and
// recording.deleted and leaves nothing behind.
let triggered = false;
if (smokeDevice) {
  const created = await call('POST', '/v1/me/recordings',
    { device_id: smokeDevice, device_name: 'realtime-smoke', status: 'recording' }, accessToken);
  check(created.status === 201, `POST /v1/me/recordings: expected 201, got ${created.status}`);
  if (created.status === 201 && created.json?.id) {
    const deleted = await call('DELETE', `/v1/me/recordings/${created.json.id}`, undefined, accessToken);
    check(deleted.status === 204, `DELETE /v1/me/recordings/${created.json.id}: expected 204, got ${deleted.status}`);
    triggered = true;
  }
}

if (triggered) {
  let event = null;
  for (let i = 0; i < 10 && event === null; i++) {
    const candidate = await events.next(1000).catch(() => null);
    if (candidate !== null) event = candidate;
  }
  check(event !== null, 'events: no event arrived after one was published');
  if (event !== null) {
    check(event.type !== '<unparseable>', 'events: a live frame was not JSON');
    check(eventTypes.includes(event.type),
      `events: live event type "${event.type}" is not in the documented catalogue`);
    for (const name of schemas.EventEnvelope?.required ?? []) {
      field(event, name, undefined, 'live event envelope');
    }
    check(typeof event.id === 'string' && event.id !== '',
      'events: a published event carries no stream id');
    check(typeof event.server_rev === 'number',
      'events: a published event carries no numeric server_rev');
    check(Number.isFinite(Date.parse(event.ts ?? '')),
      `events: event ts is "${event.ts}", contract says date-time`);
  }
} else {
  console.log('note: set QD_SMOKE_DEVICE=<device_id> to also check a real event arriving');
}
if (smokeDevice && !remoteSessionChecked) {
  console.log('note: set QD_SMOKE_REMOTE_SESSION=1 with an online device to check remote_session.started/ended');
}

// The WebClient updates its blocked-source list from this event, so pin the
// event type and the fields it consumes against the live server. A full PUT is
// used with the policy just read: it is semantically a no-op and the smoke
// restores the same values it found.
let policyTriggered = false;
if (smokeDevice) {
  const policy = await call('GET',
    `/v1/me/devices/${smokeDevice}/access-policy`, undefined, accessToken);
  check(policy.status === 200,
    `GET access-policy: expected 200, got ${policy.status}`);
  if (policy.status === 200) {
    const put = await call('PUT',
      `/v1/me/devices/${smokeDevice}/access-policy`, {
        remote_disabled: policy.json?.remote_disabled ?? false,
        allow_ips: policy.json?.allow_ips ?? [],
        deny_ips: policy.json?.deny_ips ?? [],
        max_sessions: policy.json?.max_sessions ?? 0,
        allowed_windows: policy.json?.allowed_windows ?? [],
        timezone: policy.json?.timezone ?? '',
        exception_dates: policy.json?.exception_dates ?? [],
        approval_required: policy.json?.approval_required ?? false,
        approval_mode: policy.json?.approval_mode ?? 'off',
        approval_trusted_ips: policy.json?.approval_trusted_ips ?? [],
      }, accessToken);
    check(put.status === 200, `PUT access-policy: expected 200, got ${put.status}`);
    policyTriggered = put.status === 200;
  }
}

if (policyTriggered) {
  let policyEvent = null;
  for (let i = 0; i < 10 && policyEvent === null; i++) {
    const candidate = await events.next(1000).catch(() => null);
    if (candidate?.type === 'device.access_policy.changed') policyEvent = candidate;
  }
  check(policyEvent !== null,
    'events: no device.access_policy.changed event arrived after PUT access-policy');
  if (policyEvent !== null) {
    check(eventTypes.includes(policyEvent.type),
      `events: live event type "${policyEvent.type}" is not in the documented catalogue`);
    check(policyEvent.data?.device_id === smokeDevice,
      `events: policy event device_id is "${policyEvent.data?.device_id}", want "${smokeDevice}"`);
    check(Array.isArray(policyEvent.data?.deny_ips),
      'events: policy event deny_ips is not an array');
  }
}

events.close();

// --- 5. events channel: a bad token is refused, not ignored -----------------

const badEvents = openChannel(eventsChannel.address);
await new Promise((resolve, reject) => {
  badEvents.ws.addEventListener('open', resolve);
  badEvents.ws.addEventListener('error', () => reject(new Error('connect failed')));
}).catch((err) => fatal('events (bad token)', err.message));
badEvents.send({ type: 'auth', access_token: 'not-a-real-token' });
const badFrame = await badEvents.next().catch((err) => fatal('events (bad token)', err.message));
check(badFrame.type === 'error', `events (bad token): first frame is "${badFrame.type}", want error`);
check(badFrame.data?.code === 'TOKEN_INVALID',
  `events (bad token): code is "${badFrame.data?.code}", want TOKEN_INVALID`);
badEvents.close();

// --- 6. signal channel: the handshake is documented and enforced ------------

const signal = openChannel(signalChannel.address);
await new Promise((resolve, reject) => {
  signal.ws.addEventListener('open', resolve);
  signal.ws.addEventListener('error', () => reject(new Error('connect failed')));
}).catch((err) => fatal('signal (bad token)', err.message));
signal.send({ type: 'auth', signal_token: 'not-a-real-token', device_id: '000000000', role: 'client' });
const signalFrame = await signal.next().catch((err) => fatal('signal (bad token)', err.message));
check(signalFrame.type === 'error', `signal (bad token): first frame is "${signalFrame.type}", want error`);
check(signalFrame.code === 'AUTH_INVALID',
  `signal (bad token): code is "${signalFrame.code}", want AUTH_INVALID`);
check(typeof signalFrame.detail === 'string' && signalFrame.detail !== '',
  'signal (bad token): error carries no detail');
signal.close();

// --- report ------------------------------------------------------------------

if (failures.length > 0) {
  console.error(`FAIL: ${failures.length} of ${checks} realtime assertions failed`);
  for (const failure of failures) console.error('  - ' + failure);
  process.exit(3);
}

console.log(`PASS: ${checks} realtime assertions hold against ${wsBase} (${contractPath})`);
