#!/usr/bin/env node
// Conformance smoke for SignalingServer/api/openapi.yaml.
//
// The spec is the contract business systems generate clients from, so it has to
// be checked against a running server rather than trusted: this script calls
// the documented operations and asserts the fields the spec promises exist with
// the promised types. If the server renames a field or changes its shape, this
// fails instead of quietly invalidating the spec.
//
// Read-only by default. Give QD_SMOKE_DEVICE=<device_id> and it also covers the
// documented operations with side effects - the offline-alert and access-policy
// PUTs (restored to exactly the prior state afterwards), issuing then revoking a
// short-lived guest code, and the wrong-access-code rejection (which writes an
// access-log row and may notify the owner).
//
// Usage:
//   QD_IDENTIFIER=user QD_PASSWORD=secret scripts/sdk/openapi-smoke.mjs
//   QD_IDENTIFIER=user QD_PASSWORD=secret QD_SMOKE_DEVICE=126735214 scripts/sdk/openapi-smoke.mjs
//
// Env:
//   QD_SERVER      default http://127.0.0.1:8000
//   QD_API_KEY     default qd-local-dev-key
//   QD_IDENTIFIER / QD_PASSWORD   required
//   QD_SMOKE_DEVICE               optional; enables the write-path checks
//
// Exit codes: 0 = the documented shapes hold, 3 = a documented field is missing
// or mistyped, 4 = the server could not be reached or returned an unexpected
// status.

const server = (process.env.QD_SERVER || 'http://127.0.0.1:8000').replace(/\/$/, '');
const apiKey = process.env.QD_API_KEY || 'qd-local-dev-key';
const identifier = process.env.QD_IDENTIFIER;
const password = process.env.QD_PASSWORD;
const smokeDevice = process.env.QD_SMOKE_DEVICE || '';

if (!identifier || !password) {
  console.error('QD_IDENTIFIER and QD_PASSWORD are required');
  process.exit(2);
}

let checks = 0;
const failures = [];

function check(condition, message) {
  checks += 1;
  if (!condition) failures.push(message);
}

// field asserts that a documented property exists and, when a type is given,
// has that JSON type. `null` is accepted for the nullable fields the spec
// declares as ["string","null"].
function field(obj, name, type, where) {
  const present = obj !== null && typeof obj === 'object' && name in obj;
  check(present, `${where}: missing documented field "${name}"`);
  if (!present) return undefined;
  const value = obj[name];
  if (type && value !== null && typeof value !== type) {
    check(false, `${where}: field "${name}" is ${typeof value}, spec says ${type}`);
  }
  return value;
}

async function call(method, path, { token, body, expect, extraHeaders = {} } = {}) {
  const headers = { 'X-API-Key': apiKey, ...extraHeaders };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  let res;
  try {
    res = await fetch(server + path, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch (err) {
    console.error(`transport error calling ${method} ${path}: ${err.message}`);
    process.exit(4);
  }
  const text = await res.text();
  let json = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    // Non-JSON bodies only matter if the caller expected JSON.
  }
  if (expect !== undefined && res.status !== expect) {
    console.error(`${method} ${path}: expected ${expect}, got ${res.status}`);
    console.error(text.slice(0, 400));
    process.exit(4);
  }
  return { status: res.status, json, text, headers: res.headers };
}

function problemShape(json, where, expectedCode) {
  field(json, 'type', 'string', where);
  field(json, 'title', 'string', where);
  field(json, 'status', 'number', where);
  const code = field(json, 'code', 'string', where);
  if (expectedCode) {
    check(code === expectedCode, `${where}: code is "${code}", spec documents "${expectedCode}"`);
  }
}

// 1. POST /v1/auth/sessions
const login = await call('POST', '/v1/auth/sessions', {
  body: { identifier, password },
  expect: 200,
});
const token = field(login.json, 'access_token', 'string', 'POST /v1/auth/sessions');
field(login.json, 'access_expires_at', 'string', 'POST /v1/auth/sessions');
field(login.json, 'refresh_token', 'string', 'POST /v1/auth/sessions');
field(login.json, 'refresh_expires_at', 'string', 'POST /v1/auth/sessions');
const loginUser = field(login.json, 'user', 'object', 'POST /v1/auth/sessions');
for (const name of ['id', 'username', 'status']) {
  field(loginUser, name, name === 'status' ? 'boolean' : name === 'id' ? 'number' : 'string', 'AuthSessionResponse.user');
}

// 2. GET /v1/me
const me = await call('GET', '/v1/me', { token, expect: 200 });
for (const name of ['id', 'username', 'status', 'device_count', 'channel_type', 'created_at']) {
  field(me.json, name, undefined, 'GET /v1/me');
}

// 3. GET /v1/me/devices
const devices = await call('GET', '/v1/me/devices', { token, expect: 200 });
const deviceItems = field(devices.json, 'items', 'object', 'GET /v1/me/devices');
check(Array.isArray(deviceItems), 'GET /v1/me/devices: items is not an array');
if (Array.isArray(deviceItems) && deviceItems.length > 0) {
  const device = deviceItems[0];
  for (const name of ['device_id', 'device_name', 'remark', 'online', 'logged_in', 'access_code']) {
    field(device, name, undefined, 'GET /v1/me/devices items[]');
  }
  check(typeof device.online === 'boolean', 'Device.online is not a boolean');
  check(typeof device.logged_in === 'boolean', 'Device.logged_in is not a boolean');
}

const deviceId = smokeDevice || (Array.isArray(deviceItems) && deviceItems[0]?.device_id) || '';

// 4. GET /v1/me/devices/{device_id}/offline-alert
const alertBefore = await call('GET', `/v1/me/devices/${deviceId}/offline-alert`, { token, expect: 200 });
for (const name of ['device_id', 'enabled', 'after_minutes', 'repeat_minutes', 'maintenance_windows', 'in_maintenance_window']) {
  field(alertBefore.json, name, undefined, 'GET offline-alert');
}
check(Array.isArray(alertBefore.json.maintenance_windows), 'offline-alert: maintenance_windows is not an array');
check(typeof alertBefore.json.in_maintenance_window === 'boolean', 'offline-alert: in_maintenance_window is not a boolean');

// 5. GET /v1/me/notification-channels
const channels = await call('GET', '/v1/me/notification-channels', { token, expect: 200 });
const channelItems = field(channels.json, 'items', 'object', 'GET notification-channels');
check(Array.isArray(channelItems), 'GET notification-channels: items is not an array');
field(channels.json, 'supported_events', 'object', 'GET notification-channels');
field(channels.json, 'signature_header', 'string', 'GET notification-channels');
field(channels.json, 'signature_scheme', 'string', 'GET notification-channels');
if (Array.isArray(channelItems) && channelItems.length > 0) {
  const ch = channelItems[0];
  for (const name of ['id', 'user_id', 'name', 'url', 'events', 'enabled', 'last_status']) {
    field(ch, name, undefined, 'NotificationChannel');
  }
  // The spec documents `events` as a JSON string, not an array; if the server
  // ever starts sending an array this has to be an explicit spec change.
  check(typeof ch.events === 'string', 'NotificationChannel.events is not a string');
}

// 6. GET /v1/me/devices/{device_id}/access-policy
const policy = await call('GET', `/v1/me/devices/${deviceId}/access-policy`, { token, expect: 200 });
for (const name of ['device_id', 'remote_disabled', 'allow_ips', 'max_sessions', 'allowed_windows', 'timezone', 'exception_dates', 'approval_mode', 'approval_trusted_ips']) {
  field(policy.json, name, undefined, 'GET access-policy');
}
for (const list of ['allow_ips', 'allowed_windows', 'exception_dates', 'approval_trusted_ips']) {
  check(Array.isArray(policy.json[list]), `access-policy: ${list} is not an array`);
}

// 7. GET /v1/me/devices/{device_id}/access-grants
const grants = await call('GET', `/v1/me/devices/${deviceId}/access-grants`, { token, expect: 200 });
const grantItems = field(grants.json, 'items', 'object', 'GET access-grants');
check(Array.isArray(grantItems), 'GET access-grants: items is not an array');
if (Array.isArray(grantItems) && grantItems.length > 0) {
  for (const name of ['id', 'device_id', 'code', 'expires_at', 'max_uses', 'uses', 'active', 'bound_client_id', 'bound_ips', 'created_at']) {
    field(grantItems[0], name, undefined, 'AccessGrant');
  }
  check(Array.isArray(grantItems[0].bound_ips), 'AccessGrant.bound_ips is not an array');
}

// 8. GET /v1/me/devices/{device_id}/access-approvals
const approvals = await call('GET', `/v1/me/devices/${deviceId}/access-approvals`, { token, expect: 200 });
const approvalItems = field(approvals.json, 'items', 'object', 'GET access-approvals');
check(Array.isArray(approvalItems), 'GET access-approvals: items is not an array');
if (Array.isArray(approvalItems) && approvalItems.length > 0) {
  for (const name of ['id', 'device_id', 'client_id', 'client_ip', 'source', 'status', 'expires_at', 'created_at']) {
    field(approvalItems[0], name, undefined, 'AccessApproval');
  }
}
const badStatus = await call('GET', `/v1/me/devices/${deviceId}/access-approvals?status=nope`, { token, expect: 400 });
problemShape(badStatus.json, 'GET access-approvals (unknown status)');

// 9. GET /v1/me/devices/{device_id}/access-approvals:summary
const summary = await call('GET', `/v1/me/devices/${deviceId}/access-approvals:summary`, { token, expect: 200 });
field(summary.json, 'since', 'string', 'GET access-approvals:summary');
field(summary.json, 'total', 'number', 'GET access-approvals:summary');
field(summary.json, 'denial_cooldown_seconds', 'number', 'GET access-approvals:summary');
const byStatus = field(summary.json, 'by_status', 'object', 'GET access-approvals:summary');
for (const state of ['pending', 'approved', 'denied', 'expired', 'consumed']) {
  field(byStatus, state, 'number', 'AccessApprovalSummary.by_status');
}
const deniedSources = field(summary.json, 'denied_sources', 'object', 'GET access-approvals:summary');
check(Array.isArray(deniedSources), 'GET access-approvals:summary: denied_sources is not an array');
if (Array.isArray(deniedSources) && deniedSources.length > 0) {
  field(deniedSources[0], 'client_ip', 'string', 'AccessApprovalSummary.denied_sources[]');
  field(deniedSources[0], 'denials', 'number', 'AccessApprovalSummary.denied_sources[]');
}
const badSince = await call('GET', `/v1/me/devices/${deviceId}/access-approvals:summary?since=nope`, { token, expect: 400 });
problemShape(badSince.json, 'GET access-approvals:summary (bad since)');

// 10. GET /v1/me/recordings (cursor-paginated index)
const recordings = await call('GET', '/v1/me/recordings', { token, expect: 200 });
const recordingItems = field(recordings.json, 'items', 'object', 'GET /v1/me/recordings');
check(Array.isArray(recordingItems), 'GET /v1/me/recordings: items is not an array');
// The spec deliberately documents no `total` here (the endpoint does not count
// the whole set per page), so a new `total` field is a contract change.
check(!('total' in recordings.json),
  'GET /v1/me/recordings: envelope carries total, but the spec documents no count on this endpoint');
if ('next_cursor' in recordings.json) {
  field(recordings.json, 'next_cursor', 'string', 'GET /v1/me/recordings');
}
if (Array.isArray(recordingItems) && recordingItems.length > 0) {
  const rec = recordingItems[0];
  for (const name of ['id', 'user_id', 'device_id', 'device_name', 'session_id', 'status', 'format',
    'artifact_path', 'artifact_url', 'thumbnail_path', 'thumbnail_url', 'duration_seconds',
    'size_bytes', 'started_at', 'created_at', 'updated_at']) {
    field(rec, name, undefined, 'SessionRecording');
  }
  check(['recording', 'completed', 'failed'].includes(rec.status),
    `SessionRecording.status is "${rec.status}", spec enumerates recording/completed/failed`);
}
const recLimited = await call('GET', '/v1/me/recordings?limit=1', { token, expect: 200 });
const recLimitedItems = field(recLimited.json, 'items', 'object', 'GET /v1/me/recordings?limit=1');
check(Array.isArray(recLimitedItems) && recLimitedItems.length <= 1,
  'GET /v1/me/recordings?limit=1 returned more than one item');

if (smokeDevice) {
  // 6. PUT the rule back unchanged, then restore the prior state exactly.
  const putBody = {
    enabled: alertBefore.json.enabled,
    after_minutes: alertBefore.json.after_minutes,
    repeat_minutes: alertBefore.json.repeat_minutes,
    maintenance_windows: alertBefore.json.maintenance_windows ?? [],
    maintenance_timezone: alertBefore.json.maintenance_timezone ?? '',
  };
  const put = await call('PUT', `/v1/me/devices/${deviceId}/offline-alert`, { token, body: putBody, expect: 200 });
  for (const name of ['device_id', 'enabled', 'after_minutes', 'repeat_minutes', 'maintenance_windows', 'in_maintenance_window']) {
    field(put.json, name, undefined, 'PUT offline-alert');
  }
  check(put.json.after_minutes === putBody.after_minutes, 'PUT offline-alert did not echo after_minutes');

  // The GET view cannot distinguish "no rule" from "a disabled rule", so
  // whether a row existed before is inferred from updated_at: the defaults come
  // back without one. If there was no rule, delete the one the PUT created so
  // the smoke leaves the account exactly as it found it.
  if (alertBefore.json.updated_at === undefined || alertBefore.json.updated_at === null) {
    const del = await call('DELETE', `/v1/me/devices/${deviceId}/offline-alert`, { token, expect: 204 });
    check(del.status === 204, 'DELETE offline-alert did not return 204');
  }

  // 7. PUT the access policy back unchanged. A full replace with what was just
  // read is a no-op semantically, so nothing about the device is loosened or
  // tightened by running the smoke.
  const policyPut = {
    remote_disabled: policy.json.remote_disabled,
    allow_ips: policy.json.allow_ips ?? [],
    max_sessions: policy.json.max_sessions,
    allowed_windows: policy.json.allowed_windows ?? [],
    timezone: policy.json.timezone ?? '',
    exception_dates: policy.json.exception_dates ?? [],
    approval_required: policy.json.approval_required ?? false,
    approval_mode: policy.json.approval_mode ?? 'off',
    approval_trusted_ips: policy.json.approval_trusted_ips ?? [],
  };
  const policyRes = await call('PUT', `/v1/me/devices/${deviceId}/access-policy`, { token, body: policyPut, expect: 200 });
  for (const name of ['device_id', 'remote_disabled', 'allow_ips', 'max_sessions', 'allowed_windows', 'timezone', 'exception_dates', 'approval_mode', 'approval_trusted_ips']) {
    field(policyRes.json, name, undefined, 'PUT access-policy');
  }
  check(policyRes.json.approval_mode === policyPut.approval_mode,
    `PUT access-policy changed approval_mode: ${policyRes.json.approval_mode} != ${policyPut.approval_mode}`);
  check(policyRes.json.max_sessions === policyPut.max_sessions, 'PUT access-policy did not echo max_sessions');

  // 8. Issue a short-lived guest code and revoke it. Revocation (not deletion)
  // is the documented way to disable a grant, so this leaves one inert
  // `active: false` row behind rather than pretending to erase it.
  const created = await call('POST', `/v1/me/devices/${deviceId}/access-grants`, {
    token,
    body: { label: 'openapi-smoke', expires_in_minutes: 5, max_uses: 1 },
    expect: 201,
  });
  for (const name of ['id', 'device_id', 'code', 'expires_at', 'max_uses', 'uses', 'active', 'created_at']) {
    field(created.json, name, undefined, 'POST access-grants');
  }
  check(typeof created.json.code === 'string' && created.json.code.length > 0,
    'POST access-grants returned no code');
  const revoked = await call('DELETE', `/v1/me/devices/${deviceId}/access-grants/${created.json.id}`, { token, expect: 200 });
  field(revoked.json, 'success', 'boolean', 'DELETE access-grants');

  // 9. POST access-code:verify with a code that cannot be right, to pin the
  // documented rejection shape.
  const verify = await call('POST', `/v1/devices/${deviceId}/access-code:verify`, {
    body: { code: '000000', client_id: 'openapi-smoke' },
    expect: 403,
  });
  problemShape(verify.json, 'POST access-code:verify (rejected)', verify.json?.code);
}

if (failures.length > 0) {
  console.error(`FAIL: ${failures.length} of ${checks} documented assertions failed`);
  for (const f of failures) console.error('  - ' + f);
  process.exit(3);
}

const scope = smokeDevice ? 'read + write operations' : 'read-only operations';
console.log(`PASS: ${checks} documented assertions hold against ${server} (${scope})`);
if (!smokeDevice) {
  console.log('note: set QD_SMOKE_DEVICE=<device_id> to also check the offline-alert PUT and the rejected access-code shape');
}
