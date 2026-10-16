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
// short-lived guest code, the wrong-access-code rejection (which writes an
// access-log row and may notify the owner), the whole recording lifecycle (which
// the smoke creates and deletes itself) and the approval flow (which needs the
// device to be online and drives both the device-code and grant-backed triggers
// by switching approval_mode temporarily).
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
//   QD_SMOKE_APPROVAL_DENY        set to 1 to also drive the deny outcome; a
//                                 deny arms the per-(device, source) cooldown,
//                                 so a second run against the same device would
//                                 be refused while it lasts
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

async function call(method, path, { token, body, rawBody, contentType, expect, extraHeaders = {} } = {}) {
  const headers = { 'X-API-Key': apiKey, ...extraHeaders };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  // rawBody is for the two shapes JSON cannot express: a FormData multipart
  // upload (fetch sets its own boundary) and a streamed octet-stream body.
  let payload;
  if (rawBody !== undefined) {
    payload = rawBody;
    if (contentType) headers['Content-Type'] = contentType;
  } else if (body !== undefined) {
    payload = JSON.stringify(body);
  }
  let res;
  try {
    res = await fetch(server + path, {
      method,
      headers,
      body: payload,
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
//
// The field list is shared with the write path at the end of the file, so a
// column added to one place has to exist in the other.
const RECORDING_FIELDS = ['id', 'user_id', 'device_id', 'device_name', 'session_id', 'status', 'format',
  'artifact_path', 'artifact_url', 'thumbnail_path', 'thumbnail_url', 'duration_seconds',
  'size_bytes', 'started_at', 'created_at', 'updated_at'];
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
  for (const name of RECORDING_FIELDS) {
    field(rec, name, undefined, 'SessionRecording');
  }
  check(['recording', 'completed', 'failed'].includes(rec.status),
    `SessionRecording.status is "${rec.status}", spec enumerates recording/completed/failed`);
}
const recLimited = await call('GET', '/v1/me/recordings?limit=1', { token, expect: 200 });
const recLimitedItems = field(recLimited.json, 'items', 'object', 'GET /v1/me/recordings?limit=1');
check(Array.isArray(recLimitedItems) && recLimitedItems.length <= 1,
  'GET /v1/me/recordings?limit=1 returned more than one item');

// 11. Approval decisions (approve / deny / claim) - not-found and validation shapes.
// These POSTs use an id that cannot exist, so they pin the documented error
// contract without creating or deciding anything. The happy path needs a real
// pending approval (an online device plus approval_mode), so it is covered by
// the end-to-end run rather than this always-runnable smoke.
if (deviceId) {
  const ghostApprovalId = 2147483647;
  const approveGhost = await call('POST', `/v1/me/devices/${deviceId}/access-approvals/${ghostApprovalId}/approve`, { token, expect: 404 });
  problemShape(approveGhost.json, 'POST access-approvals/:id/approve (unknown id)');
  const denyGhost = await call('POST', `/v1/me/devices/${deviceId}/access-approvals/${ghostApprovalId}/deny`, { token, expect: 404 });
  problemShape(denyGhost.json, 'POST access-approvals/:id/deny (unknown id)');
  const claimGhost = await call('POST', `/v1/devices/${deviceId}/access-approvals:claim`, {
    body: { approval_id: ghostApprovalId, secret: 'openapi-smoke' }, expect: 404,
  });
  problemShape(claimGhost.json, 'POST access-approvals:claim (unknown id)');
  const claimMissing = await call('POST', `/v1/devices/${deviceId}/access-approvals:claim`, { body: {}, expect: 400 });
  problemShape(claimMissing.json, 'POST access-approvals:claim (missing secret)');
}

// 12. Recording playback tickets and the ticket-gated media route.
// Minting a ticket is stateless (a signed URL; nothing is written), so these
// checks stay in the read-only path. The media fetch is exercised with a
// one-byte range request rather than downloading the file.
const ticketedRecording = Array.isArray(recordingItems)
  ? recordingItems.find((r) => typeof r.artifact_path === 'string' && r.artifact_path !== '')
  : undefined;
if (ticketedRecording) {
  const recordingId = ticketedRecording.id;
  const artifactTicket = await call('POST', `/v1/me/recordings/${recordingId}/artifact-ticket`, { token, expect: 200 });
  const artifactUrl = field(artifactTicket.json, 'url', 'string', 'POST artifact-ticket');
  field(artifactTicket.json, 'expires_at', 'string', 'POST artifact-ticket');
  check(typeof artifactUrl === 'string' && artifactUrl.includes('/artifact?t='),
    `artifact ticket url is "${artifactUrl}", want a signed /artifact?t= path`);

  const thumbTicket = await call('POST', `/v1/me/recordings/${recordingId}/thumbnail-ticket`, { token, expect: 200 });
  const thumbUrl = field(thumbTicket.json, 'url', 'string', 'POST thumbnail-ticket');
  check(typeof thumbUrl === 'string' && thumbUrl.includes('/thumbnail?t='),
    `thumbnail ticket url is "${thumbUrl}", want a signed /thumbnail?t= path`);

  if (typeof artifactUrl === 'string' && artifactUrl.startsWith('/')) {
    // No Authorization and no API key: the signed ticket is the authorization.
    let ranged;
    try {
      ranged = await fetch(server + artifactUrl, { headers: { Range: 'bytes=0-0' } });
    } catch (err) {
      console.error(`transport error fetching a ticket-gated artifact: ${err.message}`);
      process.exit(4);
    }
    check(ranged.status === 200 || ranged.status === 206,
      `ticket-gated artifact fetch returned ${ranged.status}, want 200 or 206`);
  }

  const badTicket = await call('GET', `/v1/recordings/${recordingId}/artifact?t=not-a-ticket`, { expect: 401 });
  problemShape(badTicket.json, 'GET artifact by ticket (bad ticket)');
} else {
  console.log('note: no pre-existing recording carries a stored artifact, so the list-based playback checks were skipped (the write path still covers playback on a recording it creates itself)');
}

const ghostTicket = await call('POST', '/v1/me/recordings/2147483647/artifact-ticket', { token, expect: 404 });
problemShape(ghostTicket.json, 'POST artifact-ticket (unknown recording)');

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

  // 10. The recording write path: index a capture, patch it, upload media and
  // a preview, play the media back through a ticket, then delete the row.
  //
  // This is the half of the recordings surface a read-only run cannot reach,
  // and doing it here means a fresh account covers the whole lifecycle instead
  // of depending on a recording that some earlier run happened to leave behind.
  const newRecording = await call('POST', '/v1/me/recordings', {
    token,
    body: {
      device_id: deviceId,
      device_name: 'openapi-smoke',
      session_id: `smoke-${Date.now()}`,
      status: 'recording',
    },
    expect: 201,
  });
  for (const name of RECORDING_FIELDS) {
    field(newRecording.json, name, undefined, 'POST /v1/me/recordings');
  }
  const recordingId = newRecording.json.id;
  check(newRecording.json.status === 'recording',
    'POST /v1/me/recordings did not keep the status it was given');

  const missingStatus = await call('POST', '/v1/me/recordings', {
    token,
    body: { device_id: deviceId },
    expect: 400,
  });
  problemShape(missingStatus.json, 'POST /v1/me/recordings (missing status)');

  const oneRecording = await call('GET', `/v1/me/recordings/${recordingId}`, { token, expect: 200 });
  check(oneRecording.json.id === recordingId, 'GET recording returned a different row');

  const patched = await call('PATCH', `/v1/me/recordings/${recordingId}`, {
    token,
    body: { status: 'completed', duration_seconds: 3 },
    expect: 200,
  });
  check(patched.json.status === 'completed' && patched.json.duration_seconds === 3,
    'PATCH recording did not apply status/duration_seconds');
  const badPatch = await call('PATCH', `/v1/me/recordings/${recordingId}`, {
    token,
    body: { status: 'finished' },
    expect: 400,
  });
  problemShape(badPatch.json, 'PATCH /v1/me/recordings/:id (invalid status)');

  // A tiny but real MP4 header: the server allow-lists the extension rather
  // than sniffing video containers, and this pins that rule.
  const media = new Uint8Array([0, 0, 0, 24, 102, 116, 121, 112, 109, 112, 52, 50]);
  const mediaForm = new FormData();
  mediaForm.append('file', new Blob([media], { type: 'video/mp4' }), 'smoke.mp4');
  const uploaded = await call('POST', `/v1/me/recordings/${recordingId}/artifact`, {
    token,
    rawBody: mediaForm,
    expect: 200,
  });
  check(uploaded.json.size_bytes === media.length,
    `artifact size_bytes=${uploaded.json.size_bytes}, want ${media.length}`);
  check(uploaded.json.format === 'mp4', `artifact format=${uploaded.json.format}, want mp4`);
  check(typeof uploaded.json.artifact_url === 'string' && uploaded.json.artifact_url !== '',
    'artifact_url is empty after the upload');

  // The two documented rejections: an extension that is not on the allow-list,
  // and an empty body.
  const deniedExt = await call('POST', `/v1/me/recordings/${recordingId}/artifact?ext=.exe`, {
    token,
    rawBody: new Uint8Array([1, 2, 3]),
    contentType: 'application/octet-stream',
    expect: 400,
  });
  problemShape(deniedExt.json, 'POST artifact (denied extension)');
  const emptyUpload = await call('POST', `/v1/me/recordings/${recordingId}/artifact?ext=.mp4`, {
    token,
    rawBody: new Uint8Array([]),
    contentType: 'application/octet-stream',
    expect: 400,
  });
  problemShape(emptyUpload.json, 'POST artifact (empty body)');

  // A real 1x1 PNG, and a file that only claims to be one: the preview route
  // sniffs the bytes, so the second upload must be refused.
  const png = Buffer.from(
    'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8AAAwAB/AMQ/AMZ4zCfAAAAAElFTkSuQmCC',
    'base64',
  );
  const thumbForm = new FormData();
  thumbForm.append('file', new Blob([png], { type: 'image/png' }), 'smoke.png');
  const thumbUpload = await call('POST', `/v1/me/recordings/${recordingId}/thumbnail`, {
    token,
    rawBody: thumbForm,
    expect: 200,
  });
  check(typeof thumbUpload.json.thumbnail_path === 'string' && thumbUpload.json.thumbnail_path !== '',
    'thumbnail_path is empty after the preview upload');
  const fakeThumbForm = new FormData();
  fakeThumbForm.append('file', new Blob([new Uint8Array([1, 2, 3, 4])], { type: 'image/png' }), 'not-really.png');
  const fakeThumb = await call('POST', `/v1/me/recordings/${recordingId}/thumbnail`, {
    token,
    rawBody: fakeThumbForm,
    expect: 400,
  });
  problemShape(fakeThumb.json, 'POST thumbnail (bytes are not an image)');

  // Playback without an Authorization header, on the artifact that was just
  // uploaded: a one-byte range request rather than the whole file.
  const freshTicket = await call('POST', `/v1/me/recordings/${recordingId}/artifact-ticket`, { token, expect: 200 });
  const freshTicketUrl = field(freshTicket.json, 'url', 'string', 'POST artifact-ticket (fresh upload)');
  check(freshTicketUrl.includes('/artifact?t='),
    `fresh ticket url is "${freshTicketUrl}", want a signed /artifact?t= path`);
  const mediaRes = await fetch(new URL(freshTicketUrl, server), { headers: { Range: 'bytes=0-0' } });
  check(mediaRes.status === 206, `ticket media fetch returned ${mediaRes.status}, want 206`);
  check((mediaRes.headers.get('content-type') || '').startsWith('video/mp4'),
    `ticket media content-type is ${mediaRes.headers.get('content-type')}, want video/mp4`);
  const oneByte = new Uint8Array(await mediaRes.arrayBuffer());
  check(oneByte.length === 1, `ticket media fetch returned ${oneByte.length} bytes, want 1`);

  // Delete it again: the smoke leaves the account's recording index as it found
  // it, which is also the only assertion that covers the delete route.
  const deletedRecording = await call('DELETE', `/v1/me/recordings/${recordingId}`, { token, expect: 204 });
  check(deletedRecording.status === 204, 'DELETE recording did not return 204');
  const goneRecording = await call('GET', `/v1/me/recordings/${recordingId}`, { token, expect: 404 });
  problemShape(goneRecording.json, 'GET deleted recording');

  // 11. Second-factor approval: device-code and grant-backed triggers, both
  // outcomes.
  //
  // The read-only checks can pin the 404/400 shapes of these routes but not
  // what a real decision does, because that needs a pending request - and a
  // pending request needs a device that is online and requires approval. The
  // smoke provides both: it turns approval on for its own device, drives the
  // exchange, and puts the policy back immediately afterwards.
  //
  // Residue, stated rather than hidden: the device keeps three decided approval
  // rows (two approved-and-consumed, one denied) plus one consumed grant row
  // and their entries on the access timeline. They are the owner's own history
  // and the reason the flow is auditable.
  const originalApprovalMode = policy.json.approval_mode ?? 'off';
  const approvalPolicy = { ...policyPut, approval_mode: 'always' };
  const approvalOn = await call('PUT', `/v1/me/devices/${deviceId}/access-policy`, {
    token,
    body: approvalPolicy,
    expect: 200,
  });
  check(approvalOn.json.approval_mode === 'always',
    `PUT access-policy left approval_mode "${approvalOn.json.approval_mode}", want always`);

  // The device list carries the plaintext access code, which is what a
  // controller would have been told out of band.
  const approvalDevice = (deviceItems ?? []).find((d) => d.device_id === deviceId);
  const deviceAccessCode = approvalDevice?.access_code ?? '';
  check(typeof deviceAccessCode === 'string' && deviceAccessCode.length > 0,
    'the device list did not return an access_code to verify with');

  // A unique client id per run: denying arms the per-source cooldown, and a
  // fixed id would make the *next* run look like the same denied source.
  const approvalClient = `openapi-smoke-${Date.now()}`;
  const pending = await call('POST', `/v1/devices/${deviceId}/access-code:verify`, {
    body: { code: deviceAccessCode, client_id: approvalClient },
    expect: 202,
  });
  for (const name of ['approval_id', 'approval_secret', 'status', 'expires_at', 'poll_after_ms']) {
    field(pending.json, name, undefined, 'POST access-code:verify (approval pending)');
  }
  check(pending.json.status === 'pending',
    `approval status is "${pending.json.status}", want pending`);

  const pendingList = await call('GET', `/v1/me/devices/${deviceId}/access-approvals?status=pending`, { token, expect: 200 });
  const pendingRow = (pendingList.json?.items ?? []).find((a) => a.id === pending.json.approval_id);
  check(pendingRow !== undefined, 'the pending request is missing from the owner\'s approval list');
  if (pendingRow) {
    for (const name of ['id', 'device_id', 'client_id', 'client_ip', 'source', 'status', 'expires_at', 'created_at']) {
      field(pendingRow, name, undefined, 'AccessApproval');
    }
    check(pendingRow.source === 'device_code',
      `device-code approval source is "${pendingRow.source}", want device_code`);
  }

  // Before a decision the claim is "keep polling", not an error.
  const claimPending = await call('POST', `/v1/devices/${deviceId}/access-approvals:claim`, {
    body: { approval_id: pending.json.approval_id, secret: pending.json.approval_secret },
    expect: 202,
  });
  for (const name of ['status', 'expires_at', 'poll_after_ms']) {
    field(claimPending.json, name, undefined, 'POST access-approvals:claim (pending)');
  }
  check(claimPending.json.status === 'pending',
    `pending claim status is "${claimPending.json.status}", want pending`);

  const approved = await call('POST', `/v1/me/devices/${deviceId}/access-approvals/${pending.json.approval_id}/approve`, {
    token,
    body: { reason: 'openapi-smoke' },
    expect: 200,
  });
  check(approved.json.status === 'approved', `approve returned status "${approved.json.status}"`);
  check(typeof approved.json.decided_at === 'string', 'approve did not record decided_at');

  const claimed = await call('POST', `/v1/devices/${deviceId}/access-approvals:claim`, {
    body: { approval_id: pending.json.approval_id, secret: pending.json.approval_secret },
    expect: 200,
  });
  for (const name of ['signal_token', 'expires_at']) {
    field(claimed.json, name, undefined, 'POST access-approvals:claim (approved)');
  }
  check(typeof claimed.json.signal_token === 'string' && claimed.json.signal_token.length > 0,
    'the approved claim returned no signal_token');

  // The approval is single-use: the same secret cannot mint a second token.
  const reclaimed = await call('POST', `/v1/devices/${deviceId}/access-approvals:claim`, {
    body: { approval_id: pending.json.approval_id, secret: pending.json.approval_secret },
    expect: 409,
  });
  check(reclaimed.json?.status === 'consumed',
    `re-claim status is "${reclaimed.json?.status}", want consumed`);

  // The grant-backed trigger is a separate branch: approval_mode "grant" must
  // send guest-code sessions to the owner's decision queue while the device's
  // own code still passes. A one-use grant stays unconsumed across the pending
  // stage and is spent only when the approved claim actually mints a token.
  const grantPolicy = { ...policyPut, approval_mode: 'grant' };
  const grantOn = await call('PUT', `/v1/me/devices/${deviceId}/access-policy`, {
    token,
    body: grantPolicy,
    expect: 200,
  });
  check(grantOn.json.approval_mode === 'grant',
    `PUT access-policy left approval_mode "${grantOn.json.approval_mode}", want grant`);

  const approvalGrant = await call('POST', `/v1/me/devices/${deviceId}/access-grants`, {
    token,
    body: { label: 'openapi-smoke-approval', expires_in_minutes: 5, max_uses: 1 },
    expect: 201,
  });
  for (const name of ['id', 'device_id', 'code', 'expires_at', 'max_uses', 'uses', 'active', 'created_at']) {
    field(approvalGrant.json, name, undefined, 'POST access-grants (approval)');
  }
  check(typeof approvalGrant.json.code === 'string' && approvalGrant.json.code.length > 0,
    'approval grant returned no code');

  const grantClient = `${approvalClient}-grant`;
  const grantPending = await call('POST', `/v1/devices/${deviceId}/access-code:verify`, {
    body: { code: approvalGrant.json.code, client_id: grantClient },
    expect: 202,
  });
  for (const name of ['approval_id', 'approval_secret', 'status', 'expires_at', 'poll_after_ms']) {
    field(grantPending.json, name, undefined, 'POST access-code:verify (grant approval pending)');
  }
  check(grantPending.json.status === 'pending',
    `grant approval status is "${grantPending.json.status}", want pending`);

  const grantPendingList = await call('GET', `/v1/me/devices/${deviceId}/access-approvals?status=pending`, { token, expect: 200 });
  const grantApprovalRow = (grantPendingList.json?.items ?? []).find((a) => a.id === grantPending.json.approval_id);
  check(grantApprovalRow !== undefined, 'the grant-backed pending request is missing from the owner\'s list');
  if (grantApprovalRow) {
    check(grantApprovalRow.source === 'grant',
      `grant approval source is "${grantApprovalRow.source}", want grant`);
    check(grantApprovalRow.grant_id === approvalGrant.json.id,
      `grant approval grant_id is "${grantApprovalRow.grant_id}", want ${approvalGrant.json.id}`);
  }

  const grantApproved = await call('POST', `/v1/me/devices/${deviceId}/access-approvals/${grantPending.json.approval_id}/approve`, {
    token,
    body: { reason: 'openapi-smoke-grant' },
    expect: 200,
  });
  check(grantApproved.json.status === 'approved',
    `grant approve returned status "${grantApproved.json.status}"`);

  const grantClaimed = await call('POST', `/v1/devices/${deviceId}/access-approvals:claim`, {
    body: { approval_id: grantPending.json.approval_id, secret: grantPending.json.approval_secret },
    expect: 200,
  });
  field(grantClaimed.json, 'signal_token', 'string', 'POST access-approvals:claim (grant approved)');
  check(typeof grantClaimed.json.signal_token === 'string' && grantClaimed.json.signal_token.length > 0,
    'the grant-backed approved claim returned no signal_token');

  const grantList = await call('GET', `/v1/me/devices/${deviceId}/access-grants`, { token, expect: 200 });
  const spentGrant = (grantList.json?.items ?? []).find((g) => g.id === approvalGrant.json.id);
  check(spentGrant !== undefined, 'the approval grant disappeared from the owner\'s grant list');
  if (spentGrant) {
    check(spentGrant.uses === 1,
      `approval grant uses is ${spentGrant.uses}, want 1 after the approved claim`);
    check(spentGrant.active === false,
      'approval grant is still active after its only approved claim');
  }

  // The other outcome, on a fresh request: deny, then the claim is refused.
  //
  // Opt-in, because a deny arms the per-(device, source address) cooldown
  // (APPROVAL_DENIAL_COOLDOWN_SECONDS, five minutes by default): running it
  // unconditionally would make a second smoke run against the same device fail
  // before the approve leg even started. CI sets QD_SMOKE_APPROVAL_DENY=1
  // because its device is created fresh for every run and nothing follows it.
  if (process.env.QD_SMOKE_APPROVAL_DENY === '1') {
    // The grant leg above left approval_mode at "grant"; the deny leg is about
    // the device-code source, so switch it back to "always" explicitly.
    const denyOn = await call('PUT', `/v1/me/devices/${deviceId}/access-policy`, {
      token,
      body: { ...policyPut, approval_mode: 'always' },
      expect: 200,
    });
    check(denyOn.json.approval_mode === 'always',
      `PUT access-policy left approval_mode "${denyOn.json.approval_mode}", want always for deny`);
    const deniedRequest = await call('POST', `/v1/devices/${deviceId}/access-code:verify`, {
      body: { code: deviceAccessCode, client_id: `${approvalClient}-denied` },
      expect: 202,
    });
    const denied = await call('POST', `/v1/me/devices/${deviceId}/access-approvals/${deniedRequest.json.approval_id}/deny`, {
      token,
      body: { reason: 'openapi-smoke' },
      expect: 200,
    });
    check(denied.json.status === 'denied', `deny returned status "${denied.json.status}"`);
    const claimDenied = await call('POST', `/v1/devices/${deviceId}/access-approvals:claim`, {
      body: { approval_id: deniedRequest.json.approval_id, secret: deniedRequest.json.approval_secret },
      expect: 403,
    });
    problemShape(claimDenied.json, 'POST access-approvals:claim (denied)');
  }

  // Put the policy back exactly as it was before the flow.
  const approvalOff = await call('PUT', `/v1/me/devices/${deviceId}/access-policy`, {
    token,
    body: policyPut,
    expect: 200,
  });
  check(approvalOff.json.approval_mode === originalApprovalMode,
    `access-policy was not restored: approval_mode "${approvalOff.json.approval_mode}", want "${originalApprovalMode}"`);
}

if (failures.length > 0) {
  console.error(`FAIL: ${failures.length} of ${checks} documented assertions failed`);
  for (const f of failures) console.error('  - ' + f);
  process.exit(3);
}

const scope = smokeDevice ? 'read + write operations' : 'read-only operations';
console.log(`PASS: ${checks} documented assertions hold against ${server} (${scope})`);
if (!smokeDevice) {
  console.log('note: set QD_SMOKE_DEVICE=<device_id> to also check the offline-alert PUT, the rejected access-code shape and the recording write path');
}
