# QuickDesk Signaling Server v1 API

> Updated: 2026-05-12 (v1 refactor; aligned with `docs/dev/信令服务器API重构方案.md` §2.2)
> All routes are version-prefixed under `/v1`. Pre-refactor `/api/v1/*` endpoints are removed.

This document is the canonical client-facing reference for the SignalingServer
HTTP/WS surface. Server source of truth lives in
[`SignalingServer/cmd/signaling/main.go`](../cmd/signaling/main.go); the design
rationale for every decision is in
[`QuickDesk/docs/dev/信令服务器API重构方案.md`](../../docs/dev/信令服务器API重构方案.md).

---

## Table of contents

1. [Conventions](#1-conventions)
2. [Authentication layers](#2-authentication-layers)
3. [Public surface](#3-public-surface)
4. [Auth (register / login / refresh / password reset)](#4-auth)
5. [Current user (`/v1/me/*`)](#5-current-user-v1me)
6. [Devices (host / client side)](#6-devices)
7. [WebSockets](#7-websockets)
8. [Admin surface](#8-admin-surface)
9. [Error format](#9-error-format)
10. [Rate limits](#10-rate-limits)
11. [Realtime event types](#11-realtime-event-types)

---

## 1. Conventions

### 1.1 Wire format

- All requests and responses are `application/json` unless noted.
- All field names on the wire are **`snake_case`** (`device_id`, `access_code`,
  `last_seen_at`, `next_cursor`, `device_count`, `channel_type`, `date_from`,
  …). i18n dictionary keys and language-internal identifiers may keep their
  native casing — only fields that cross the network boundary are normalized.
- Timestamps are RFC 3339 / ISO 8601 with timezone (`2026-05-12T08:52:39Z`).
- All non-2xx responses are RFC 7807 problem details with
  `Content-Type: application/problem+json` (see [§9](#9-error-format)).

### 1.2 Envelopes

- **Single object**: bare object — `{"id": 1, …}`.
- **List**: cursor-paginated — `{"items": [...], "next_cursor": "..." | null, "total": 123?}`.
- **Errors**: RFC 7807 (see §9).

### 1.3 Pagination

List endpoints accept:

- `cursor` — opaque token from the previous response's `next_cursor`. Empty/absent
  means "first page".
- `limit` — page size (default 20, max 100).

Server returns `next_cursor: null` when the cursor reaches the end. Cursors are
order-sensitive: changing sort/filter parameters invalidates the cursor stack —
clients should reset.

### 1.4 Request ID

Every HTTP request gets an `X-Request-ID` header:

- Client may send `traceparent` / `X-Request-ID`; server propagates it.
- Server-generated ULIDs otherwise.
- Error responses echo it as `trace_id` for correlation.

---

## 2. Authentication layers

Three orthogonal layers:

| Layer | Credential | Where it goes | Lifetime |
|---|---|---|---|
| **Server admission** | `X-API-Key` (compile-injected on official binaries; runtime overridable) | All `/v1/*` HTTP requests | Compile-time fixed (or admin override) |
| **Device identity** | `device_secret` (per-device, server-issued at provision time) | `Authorization: Bearer <device_secret>` (host-side device APIs only) | Permanent until rotation |
| **User identity** | `access_token` (short) + `refresh_token` (long) | `Authorization: Bearer <access_token>` (user APIs); refresh only at `POST /v1/auth/tokens:refresh` | access 2 h, refresh 30 d |

Notes:

- `X-API-Key` is the global "official binary admits to official server" gate.
  When the server has no API key configured, the middleware short-circuits to
  pass-through — self-hosted deployments work without any client config.
- WebSocket upgrades (`/v1/realtime/*`) are **not** subject to `X-API-Key`,
  because browsers can't attach custom headers to a WS handshake. They
  authenticate via first-frame auth (§7).
- `device_secret` and the user `access_token` are completely independent. An
  unbound device can connect host signaling without any user; a logged-in user
  can read their devices without any device's secret.
- The "client" role of access-code verify accepts **either** `X-API-Key`
  **or** an `Origin` in the admin-configured `allowed_origins` whitelist —
  this is what lets a browser-only WebClient call the verify endpoint.

---

## 3. Public surface

### 3.1 `GET /health`

Liveness + readiness probe. Public, no auth.

```json
{
  "status": "ok",
  "version": "v2.10.0",
  "components": {
    "postgres": "ok",
    "redis": "ok"
  }
}
```

Returns `503` with `status: "degraded"` if any component is non-`ok`. The
`version` value comes from the `Version` ldflag (`go build -ldflags
"-X main.Version=$(git describe --tags --always)"`); defaults to `"dev"` when
unset.

Use as Kubernetes readiness/liveness probe directly.

### 3.2 `GET /v1/preset`

Returns the unified preset bundle (announcements, links, WebClient URL,
minimum native-client version). Consumed by `PresetManager` in Qt and by the
WebClient bootstrap.

```json
{
  "announcement": { "zh_CN": "...", "en_US": "..." },
  "links":        { "zh_CN": [...], "en_US": [...] },
  "webclient_url": "https://web.quickdesk.cc",
  "min_version":   "2.10"
}
```

### 3.3 `GET /v1/settings/public`

Subset of the runtime settings safe for unauthenticated clients (site name,
brand, login logo URL, registration enabled flag, etc.).

### 3.4 `GET /v1/features`

Feature flag map.

```json
{
  "sms_enabled":      true,
  "register_enabled": true
}
```

### 3.5 `POST /v1/verification-codes`

Single entrypoint for all SMS scenarios. The legacy `/api/v1/sms/send` route is
removed.

Request:

```json
{ "phone": "13800138000", "scene": "login" }
```

Valid `scene` values: `login`, `register`, `reset_password`, `bind_phone`.

Response:

```json
{ "request_id": "vc_01H...", "expires_at": "2026-05-12T08:57:39Z" }
```

Rate limits per phone (Redis `qd:sms:rate:{phone}`):

- 1 / minute
- 3 / 10 minutes
- 10 / 24 hours

---

## 4. Auth

All endpoints under `/v1/auth/*` are public (no user token required) but still
sit behind `X-API-Key` when the server has one configured.

### 4.1 `POST /v1/auth/register`

Registers a new account and **automatically logs in** — the response carries
the same envelope as `/v1/auth/sessions` so the client doesn't need to
re-prompt for password.

Request:

```json
{
  "username":  "barry",
  "password":  "...",
  "phone":     "13800138000",
  "email":     "barry@example.com",
  "sms_code":  "123456"
}
```

`phone` / `email` / `sms_code` are optional unless the server requires phone
verification on register (see `/v1/features.register_enabled` semantics).

Response (200):

```json
{
  "user": {
    "id":         1,
    "username":   "barry",
    "phone":      "13800138000",
    "email":      "barry@example.com",
    "level":      "V1",
    "device_count": 0,
    "channel_type": "全球",
    "status":     true,
    "created_at": "2026-05-12T08:52:39Z",
    "updated_at": "2026-05-12T08:52:39Z"
  },
  "access_token":       "...",
  "access_expires_at":  "2026-05-12T10:52:39Z",
  "refresh_token":      "...",
  "refresh_expires_at": "2026-06-11T08:52:39Z"
}
```

### 4.2 `POST /v1/auth/sessions` — password login

```json
{ "identifier": "barry", "password": "..." }
```

`identifier` may be username, phone, or email. Response is identical to
`POST /v1/auth/register`.

### 4.3 `POST /v1/auth/sessions:sms` — SMS login

```json
{ "phone": "13800138000", "sms_code": "123456" }
```

Same response envelope.

### 4.4 `POST /v1/auth/tokens:refresh`

```json
{ "refresh_token": "..." }
```

Response:

```json
{
  "access_token":       "...",
  "access_expires_at":  "...",
  "refresh_token":      "...",
  "refresh_expires_at": "..."
}
```

**Rotation semantics**: the server invalidates the old refresh token
(Redis `GETDEL`) and issues a new one in the same family. Reusing a
previously rotated refresh token causes the entire family to be revoked and
returns `401 REFRESH_INVALID` — the leaked client must re-login.

### 4.5 `POST /v1/auth/password-resets`

```json
{ "phone": "13800138000" }
```

Sends an SMS code (scene `reset_password`).

### 4.6 `POST /v1/auth/password-resets:confirm`

```json
{ "phone": "13800138000", "sms_code": "123456", "new_password": "..." }
```

On success, every existing session of the user is revoked (clients receive a
`session.revoked` realtime event and must clear local credentials).

---

## 5. Current user (`/v1/me/*`)

Requires `Authorization: Bearer <access_token>`. On `401 TOKEN_EXPIRED`,
clients must silently call `/v1/auth/tokens:refresh` and retry once; a second
401 means the session has ended and the client should clear local state and
return to the login flow.

### 5.1 Profile

| Method | Path | Body | Notes |
|---|---|---|---|
| `GET` | `/v1/me` | — | Returns the user object (same shape as `register`). |
| `PUT` | `/v1/me/password` | `{old_password, new_password}` | Revokes all of the user's sessions. |
| `PUT` | `/v1/me/username` | `{username}` | |
| `PUT` | `/v1/me/phone` | `{phone, sms_code}` | SMS verification required. |
| `PUT` | `/v1/me/email` | `{email}` | |

### 5.2 Sessions

| Method | Path | Notes |
|---|---|---|
| `GET` | `/v1/me/sessions` | Lists active sessions (`{id, user_agent, ip, last_seen_at, created_at}`). |
| `DELETE` | `/v1/me/sessions/current` | Logs out the current session. |
| `DELETE` | `/v1/me/sessions/:session_id` | Kicks another session. |

### 5.3 My devices

The unified envelope is `{items: [DeviceItem], next_cursor}`. Every
`DeviceItem` has the canonical shape:

```jsonc
{
  "device_id":   "176017615",
  "device_name": "Barry-PC",
  "remark":      "office desktop",
  "online":      true,    // derived: hb && wsconn — never persisted
  "logged_in":   true,    // derived: logged_in_intent && online
  "access_code": "633732",
  "os":          "win",
  "os_version":  "11",
  "app_version": "2.10.0",
  "last_seen_at":"2026-05-12T08:52:39Z"
}
```

| Method | Path | Body | Notes |
|---|---|---|---|
| `GET` | `/v1/me/devices?cursor=&limit=` | — | Cursor-paginated list. |
| `POST` | `/v1/me/devices` | `{device_id}` | Bind / take-over. Idempotent — re-binding a device already owned by `me` is a 200 no-op. Take-over publishes `device.ownership.lost` to the previous owner. |
| `GET` | `/v1/me/devices/:device_id` | — | One-shot device detail. |
| `PATCH` | `/v1/me/devices/:device_id` | `{remark?, device_name?}` | Partial update. |
| `DELETE` | `/v1/me/devices/:device_id` | — | Unbind (clears `user_id` and `logged_in_intent`). |
| `DELETE` | `/v1/me/devices/:device_id/session` | — | Marks `logged_in_intent = false` without unbinding. Used by the Qt logout flow. |
| `GET` | `/v1/me/devices/:device_id/access-logs?cursor=&limit=` | — | Owner-only timeline of access to **this device**: `{items: [{id, device_id, owner_id, client_id, session_id, ip, result, reason, created_at}], next_cursor}`. |

### 5.3.1 Device access log (who tried to reach my machine)

Every `POST /v1/devices/:device_id/access-code:verify` writes one row, so the
device owner can answer "did anyone try to open my machine, from where, and
did it work". The controller cannot fabricate or suppress these rows — they are
written server-side, never reported by the client. `result` is one of:

| `result` | Meaning |
|---|---|
| `granted` | Access code matched and the host was online; a `signal_token` was minted. |
| `denied_invalid_code` | Wrong access code. |
| `denied_rate_limited` | Blocked by the per-IP / per-device / per-(device,ip) limiter, or the failure counter tripped. |
| `host_offline` | Code matched but the host was not connected; no session started. |
| `session_started` | The controller actually opened the signaling channel with the token from a `granted` attempt — i.e. the attempt turned into a real control session. Recorded by the realtime handler; `session_id` identifies that session. |
| `denied_policy` | The device's own access policy refused the client (remote access disabled, or the client IP is outside `allow_ips`). See 5.3.3. |
| `denied_session_limit` | The access code was accepted, but the device already had as many controllers as `max_sessions` allows. Recorded when the controller opens its signaling socket. |

`owner_id` is the owner at the moment of the attempt (`0` when the device is
unbound). Rows are newest-first and paginated by the `next_cursor` keyset.
Each attempt also fans a `device.access.denied` / `device.access.granted` event
to that owner's realtime WS stream, and a real control session adds
`device.access.session_started`; rejected attempts additionally land in the
admin audit log.

### 5.3.2 Notification channels (per-user webhooks)

Admin webhooks only carry system-scope events, so a user could not be told
that someone tried to open *their* device. A notification channel is a
user-owned webhook endpoint that receives that user's own device events.

| Method | Path | Body | Notes |
|---|---|---|---|
| `GET` | `/v1/me/notification-channels` | — | `{items, supported_events, signature_header, signature_scheme}`. The signing secret is never returned. |
| `POST` | `/v1/me/notification-channels` | `{name?, url, secret?, events: [...], enabled?}` | `enabled` defaults to `true`. `events` must come from `supported_events`. |
| `PATCH` | `/v1/me/notification-channels/:id` | same as create | Omit `enabled` to keep the current state; omit `secret` to keep the stored one. |
| `DELETE` | `/v1/me/notification-channels/:id` | — | Owner-scoped. |
| `POST` | `/v1/me/notification-channels/:id/test` | — | Fires a synthetic `test` event so the endpoint can be verified without waiting for device traffic. |
| `GET` | `/v1/me/notification-channels/:id/deliveries?limit=` | — | Delivery history for the channel (newest first, default 50, max 200). Owner-scoped. |
| `POST` | `/v1/me/notification-channels/:id/deliveries/:deliveryId/retry` | — | Re-arms a parked `failed` delivery (dead-letter replay). Returns `400` for rows that are not `failed`. Owner-scoped. |
| `POST` | `/v1/me/notification-channels/:id/deliveries:retry-failed` | optional body | Re-arms parked `failed` deliveries for the channel. Owner-scoped. |
| `POST` | `/v1/me/notification-channels:retry-failed` | optional body | Re-arms parked `failed` deliveries the caller owns, **across all of their channels** (one call after an outage that hit several endpoints). |

The two bulk-replay endpoints take the same optional body. Omitting it replays
everything in scope, which is how they behaved before filters existed.

| Field | Effect |
|---|---|
| `event_type` | only rows of this event type |
| `device_id` | only rows about this device. Rows written before the `device_id` column existed have an empty value and only match an unfiltered replay |
| `limit` | re-arm at most this many rows, **oldest first**, so repeated calls drain the backlog deterministically. `0` = no cap; anything above `1000` is clamped to `1000` and the response reports the effective value; negative is a `400` |
| `spread_over_minutes` | pace the replay across this many minutes instead of making the whole backlog due at once. `0` = everything due immediately (the default); anything above `1440` is clamped and the response reports the effective value; negative is a `400` |
| `dry_run` | count what *would* be re-armed and change nothing |

Response: `{success, matched, requeued, dry_run, limit, spread_over_minutes,
filters}`. `matched` is how many parked rows satisfy the filters before
`limit`; `requeued` is how many were actually re-armed (always `0` on a dry
run). The dry run exists because re-arming wakes the delivery worker
immediately, and "replay everything" is rarely the right first move when one
endpoint is broken and another is healthy.

`spread_over_minutes` is the tool for the other half of that problem: after an
outage the whole backlog is due at once, which can knock the endpoint over again
the moment it recovers. A spread replay re-arms every matched row as `pending`
straight away but staggers `next_attempt_at` across the window, so the existing
worker delivers the backlog in waves. The first wave is due one bucket interval
after the call and the last at the end of the window; nothing is due at the
moment of the call.

Payload shape (same as admin webhooks):

```json
{"event": "device.access.denied", "timestamp": "2026-10-16T03:35:23Z", "data": {"device_id": "...", "ip": "...", "reason": "..."}}
```

Supported events today: `device.access.denied`, `device.access.granted`,
`device.access.session_started`, `device.online.changed`,
`device.offline.sustained` (§5.3.6), `device.bound`, `device.unbound`,
`device.access_approval.requested`, `device.access_approval.decided`.

When `secret` is set the raw body is signed with HMAC-SHA256 and sent as
`X-Webhook-Signature: sha256=<hex>`, so receivers can share one verification
routine with the admin webhooks. URLs must be absolute `http(s)`; operators
exposing this as multi-tenant SaaS should also apply an egress allowlist (a
user-supplied URL is an SSRF surface).

**Delivery is durable (at-least-once).** Every event is first written to the
`notification_deliveries` outbox, so a channel that is briefly down no longer
loses the event. A background worker polls the outbox every 2s, claims due
rows with `SELECT ... FOR UPDATE SKIP LOCKED` (safe across two server
instances) and posts them one at a time:

- the claim is **per-channel fair**: phase one picks the channels holding the
  oldest due work, phase two takes at most 5 rows from each. Ordering the whole
  table by id would let one channel with a deep backlog fill every batch, so
  every other endpoint's notifications would wait behind it no matter how small
  their own backlog is. The trade-off is that a batch comes out shorter than 20
  rows when few channels have due work;
- success (`2xx`) → `status: "delivered"`, `delivered_at` is stamped;
- failure → the row is rescheduled with exponential backoff
  (2s, 4s, 8s … capped at 5m);
- after 8 attempts the row is parked as `status: "failed"` (kept, never
  silently dropped) and still shows up in the history.

Each attempt also records `last_status` (the HTTP code, `0` on a transport
error) and `last_error` on both the row and the channel row. Because delivery
is *at-least-once*, a receiver that fails after processing may see an event
twice — consumers should treat `event` + `timestamp` + `data.log_id` as an
idempotency key. The `:id/deliveries` endpoint is the operator-facing log:
it shows what was queued, how many attempts it took, and whether it is still
`pending`, `delivering`, `delivered` or `failed`.

The claim is written as `SELECT ... FOR UPDATE SKIP LOCKED`, so the outbox is
safe to drain from more than one worker or more than one server instance — two
workers never take the same row. A row that exhausts its budget is parked as
`failed` rather than dropped; `POST .../deliveries/:deliveryId/retry` re-arms
one for another delivery pass (resetting its attempt budget to zero).

Retention is opt-in: `NOTIFICATION_RETENTION_DAYS` (default `0` = keep history
forever) deletes finished rows — `delivered` and `failed` only — whose
`updated_at` is older than the window. In-flight rows (`pending`, `delivering`)
are never touched, so a long backoff is not mistaken for history. The sweep runs
every `NOTIFICATION_SWEEP_INTERVAL_MINUTES` (default `60`); when retention is `0`
the worker is not started at all.

Set `NOTIFICATION_ARCHIVE_DIR` to make retention **archive before deleting**:
expired rows are appended (one JSON object per line) to a daily
`notification-deliveries-YYYY-MM-DD.jsonl` under that directory and only removed
after the file is flushed to disk. If the archive cannot be written the purge
aborts with an error and the rows stay, so retention can never be the step that
destroys the only copy of a delivery history. Leave it empty for the plain
delete.

### 5.3.3 Device access policy (who is allowed to connect)

The access log records who *tried*; the policy decides who *may*. The owner of
the controlled machine sets it, and it is enforced on the device-side
`access-code:verify` endpoint **before** the access code is compared — a
blocked client never gets to guess codes.

| Method | Path | Body | Notes |
|---|---|---|---|
| `GET` | `/v1/me/devices/:device_id/access-policy` | — | Returns `{device_id, remote_disabled, allow_ips: [], max_sessions, allowed_windows: [], timezone, exception_dates: [], approval_required, approval_mode, approval_trusted_ips: [], updated_at?}`. A device with no policy returns `remote_disabled: false, allow_ips: [], max_sessions: 0, allowed_windows: [], exception_dates: [], approval_required: false, approval_mode: "off", approval_trusted_ips: []`. |
| `PUT` | `/v1/me/devices/:device_id/access-policy` | `{remote_disabled: bool, allow_ips: ["10.0.0.0/8"], max_sessions: 1, allowed_windows: [{"days":[1,2,3,4,5],"start":"09:00","end":"18:00"}], timezone: "Asia/Shanghai", exception_dates: [{"holiday":"春节","mode":"deny"},{"repeat":"lunar","lunar_month":8,"lunar_day":15,"mode":"deny"},{"date":"2026-10-17","mode":"allow"}], approval_mode: "grant", approval_trusted_ips: ["10.211.55.0/24"]}` | Full replace. IP entries must be bare IPs or CIDR blocks; `0` max sessions means unlimited; negative is rejected. An unknown `approval_mode` is a `400`. |

`max_sessions` is a concurrency cap on *controllers* (viewers count too, the
host socket does not). It is enforced when a controller authenticates its
signaling socket — after the access code was already accepted — so the log
reads `granted` followed by `denied_session_limit`, and the rejected socket
gets `SESSION_LIMIT_REACHED` **instead of** `auth_ok`.

`allowed_windows` restricts *when* sessions may start, in the policy's
`timezone` (IANA name, default = the server's local zone; the server embeds
the IANA database so this works on Windows and scratch containers):

- `days` are `0`=Sunday … `6`=Saturday; omit `days` for "every day";
- `start`/`end` are `HH:MM` wall-clock times, `end` exclusive;
- `start` later than `end` wraps past midnight and the day filter applies to
  the day the window *starts* (`{"days":[5],"start":"22:00","end":"06:00"}`
  covers Friday night through Saturday 06:00);
- an empty list means "any time".

`exception_dates` override the weekly schedule for specific calendar dates,
evaluated in the same `timezone`:

- `mode: "deny"` blocks the covered day(s) (holidays, maintenance windows) even
  if a window would otherwise allow them — reason
  `today is blocked by an access exception date`;
- `mode: "allow"` permits the covered day(s) and **overrides**
  `allowed_windows` (a one-off work day);
- an `allow` and a `deny` entry may not cover the same day (overlap is rejected
  as ambiguous, with the two offending indexes named); same-mode overlap is
  allowed. An empty list means "no exceptions".

Each entry carries **exactly one** of three date sources:

1. **Fixed date** — `date` is `YYYY-MM-DD`; `end_date` is an optional
   **inclusive** range end and defaults to `date`; `repeat: "yearly"` makes it
   recur every year on the same month-day(s), and a yearly range may wrap the
   new year (`2026-12-30` → `2026-01-02`). A non-yearly range must be ordered.
2. **Statutory holiday** — `holiday` names an entry in the server's holiday
   calendar, e.g. `"春节"`, `"国庆节"`, `"元旦"`. The dates come from the data,
   not from the policy, so the whole holiday break is one entry:
   `{"holiday":"春节","mode":"deny"}`. `holiday_day: "work"` selects the
   make-up workdays (调休上班) instead of the days off, which is how a Saturday
   that the government turned into a workday can be admitted by a weekly
   window that would otherwise skip it:
   `{"holiday":"春节","holiday_day":"work","mode":"allow"}`. An unknown holiday
   name is rejected at write time rather than silently never matching.
3. **Lunar date** — `repeat: "lunar"` with `lunar_month` (1–12) and `lunar_day`
   (1–30) follows the lunar calendar, so `{"repeat":"lunar","lunar_month":8,"lunar_day":15,"mode":"deny"}`
   blocks 中秋 every year without an annual edit. `12/30` falls back to `12/29`
   when that month is short, which keeps a 除夕 entry working across years.

The holiday calendar is the 2026 CN arrangement compiled into the binary
(sourced from the State Council notice) plus, when `HOLIDAY_CALENDAR_DIR` is
set, every `YYYY.json` found there. The files use the public `holiday-cn`
format (`{year, papers, days:[{name,date,isOffDay}]}`), so next year's file can
be dropped in and the server restarted without a rebuild.

Exception dates only decide the *time* rule. `remote_disabled` and `allow_ips`
are still evaluated first, so an allow date cannot bypass either of them.

`approval_mode` adds a second factor on top of a correct access code
(§5.3.5): instead of a signal token, the session becomes a pending request the
owner must approve. It is not a substitute for the code — the code is still
compared, and the policy gate above still runs. The mode decides *which*
sessions are worth interrupting the owner for:

| `approval_mode` | Asks for approval when… |
|---|---|
| `off` (default) | never |
| `always` | every session, whatever the credential |
| `grant` | the session used a time-boxed guest grant, not the device's own code — the "I lent someone a code" case |
| `untrusted_ip` | the controller is outside `approval_trusted_ips` — the "someone from somewhere new" case |

`approval_trusted_ips` is a list of IPs/CIDR blocks (same syntax as
`allow_ips`) and is **required** for `untrusted_ip`; an empty list there is a
`400` rather than a silent "trust everyone". `approval_required: true` is kept
for compatibility and means `approval_mode: "always"`; the response always
reports the resolved mode, so `{"approval_required": true}` reads back as
`{"approval_required": true, "approval_mode": "always"}`. An unrecognised mode
is rejected at write time — a typo must never quietly turn the gate off.

Approval requests carry a `trigger` (`always` / `grant` / `untrusted_ip`) on
the `device.access_approval.requested` event, and the access timeline says
which rule asked, e.g. `waiting for owner approval (guest code)`.

A controller outside every window is rejected at verify time with
`ACCESS_POLICY_DENIED` / "outside the allowed access window" and shows up as
`denied_policy` on the access timeline — the access code is never compared.

Rejected attempts are recorded in the access log as `denied_policy` (with the
reason) and fan out as the usual `device.access.denied` event, so notification
channels subscribed to denials see policy blocks too. Policy changes are
audited as `device.access_policy.changed`.

### 5.3.4 Device access grants (time-boxed guest codes)

Handing a guest the device's own access code means handing them a code that
keeps working. A grant is a separate, time-boxed (optionally single-use) code
the owner can issue, hand out, and revoke.

| Method | Path | Body | Notes |
|---|---|---|---|
| `GET` | `/v1/me/devices/:device_id/access-grants` | — | `{items: [...]}` newest first. Each item carries the derived `active` flag (so clients never re-implement the expiry/quota/revocation rules), the `bound_client_id` / `bound_ips` binding, and `last_used_client_id` / `last_used_ip` showing who actually redeemed it. |
| `POST` | `/v1/me/devices/:device_id/access-grants` | `{label?, expires_in_minutes, max_uses?, code?, bound_client_id?, bound_ips?}` | `expires_in_minutes` must be positive; `max_uses` defaults to `0` = unlimited within the window; `code` is generated (8 chars, unambiguous alphabet) when omitted. Returns `201` with the code, which the owner must relay to the guest. A `bound_ips` entry that is not an IP/CIDR or a `bound_client_id` over 64 characters is a `400`. |
| `DELETE` | `/v1/me/devices/:device_id/access-grants/:grant_id` | — | Revokes. Revoking an already-revoked grant is a no-op. |

**Optional guest binding.** A grant is a bearer code by default: whoever holds
it can redeem it. `bound_client_id` and `bound_ips` narrow that:

| Field | Enforced against | Strength |
|---|---|---|
| `bound_client_id` | the `client_id` the controller sends to `access-code:verify` | client-asserted friction: it stops a code being pasted into a different client, but a client can claim any id, so it is not proof of identity |
| `bound_ips` | the address the server actually saw | a real boundary: a forwarded code stays useless from anywhere else |

Both are optional and ANDed when both are set; empty means "anyone holding the
code", which is how grants behaved before the fields existed. A controller that
matches the code but not the binding gets the ordinary `403 INVALID_CODE` — the
same answer as a wrong code, so a leaked code does not confirm itself.

`access-code:verify` accepts **either** the device's access code or the code of
a usable grant. Two properties matter:

- **The policy still wins.** The device policy (remote switch, IP allow list,
  windows, exception dates) is evaluated before the code comparison, so a grant
  can never open a device the owner has switched off or restricted.
- **A use is only counted when a session actually starts.** Matching a grant
  does not consume it; the counter is incremented after the host-online check,
  so a guest who arrives while the host is offline does not burn a use. The
  increment re-checks the window and quota inside the `UPDATE`, so two
  controllers racing the last slot of a single-use grant cannot both win — the
  loser gets `403 INVALID_CODE`.

Admissions via a grant land on the access timeline as `granted` with the reason
`access grant <id>`, so the owner can tell a guest code from their own code.

Grants also emit realtime events to the owner (§11):

- `device.access_grant.changed` on create and revoke (`action: "created"` /
  `"revoked"`). It deliberately omits the code — the owner already received it
  in the create response and events must not carry credentials. This event is
  also written to the admin audit trail, the same way a policy change is.
- `device.access_grant.used` when a grant admits a session, carrying
  `{grant_id, label, uses, max_uses, client_id}`. Successful admissions are
  intentionally *not* audited (that would add an admin row per healthy session);
  they are visible on the owner-scoped access timeline instead.

### 5.3.5 Device access approvals (second factor)

`approval_mode` on the device policy (§5.3.3) turns "right access code" into
"right access code **and** the owner said yes", for the sessions the owner
chose to gate (`always`, `grant`, or `untrusted_ip`). It exists for the case
the code alone cannot cover: a guest who was given a code walks up at a bad
moment, or someone connects from an address the owner does not recognise.

The flow:

1. the controller calls `access-code:verify` (§6.5) with a correct code or
   grant while the host is online;
2. instead of a signal token, the server creates a pending approval and
   answers `202` with `{approval_id, approval_secret, status: "pending",
   expires_at, poll_after_ms}`. The secret is shown **once** and stored only as
   a SHA-256 hash;
3. the owner sees it on `GET /v1/me/devices/:device_id/access-approvals`
   (the request also fans out as `device.access_approval.requested`), and
   approves or denies;
4. the controller polls `POST /v1/devices/:device_id/access-approvals:claim`
   with `{approval_id, secret}` until it is granted, then receives the usual
   `{signal_token, expires_at}`.

| Method | Path | Auth | Body / notes |
|---|---|---|---|
| `GET` | `/v1/me/devices/:device_id/access-approvals` | user token | `?status=pending&limit=50`. `status` is a comma-separated subset of `pending,approved,denied,expired,consumed` and defaults to `pending`. Returns `{items: [...]}` newest first. |
| `POST` | `/v1/me/devices/:device_id/access-approvals/:approval_id/approve` | user token | Optional `{reason}`. `200` with the updated row. A request that was already decided (or timed out) returns `409` with the current `status` instead of silently flipping it. |
| `POST` | `/v1/me/devices/:device_id/access-approvals/:approval_id/deny` | user token | Optional `{reason}`. Same `200` / `409` shape. |
| `POST` | `/v1/devices/:device_id/access-approvals:claim` | `X-API-Key` | `{approval_id, secret}` from step 2. `202` while pending, `200` with a signal token once approved, `403` when denied or expired, `404` for an unknown id *or* a wrong secret (they are deliberately indistinguishable), `409` when the approval was already used. |

Rules that matter:

- The approval is **single-use**. The `approved → consumed` transition is a
  single `UPDATE` re-checking the status, so two controllers racing the same
  approval cannot both mint a token — the loser gets `409`.
- **The host is re-checked at claim time.** An approval is not consumed while
  the host is offline; the controller gets `409 HOST_OFFLINE` and can claim
  again once the device is back.
- **A grant still spends its use only when the session can start.** For a
  grant-backed approval, the grant counter is incremented during the claim,
  after the host-online check — the same property as the non-approval path.
- A pending approval **expires** after 2 minutes (`DeviceAccessApprovalDefaultTTL`);
  the client then sees `403` on claim and the row reads `expired`.
- Access-timeline rows read `pending_approval` on request, `denied_approval`
  when the owner refuses, and `granted` when the claim succeeds. Approving or
  denying is also written to the admin audit trail (it is an admission
  decision), while healthy admissions stay owner-scoped.

### 5.3.6 Device offline alerts ("tell me when this stays offline")

`device.online.changed` fires the moment a heartbeat stops, which is the same
event whether the machine is rebooting, asleep, temporarily unplugged or dead.
An offline alert adds the missing dimension — **how long** — as an owner-set
rule per device:

| Method | Path | Auth | Body / notes |
|---|---|---|---|
| `GET` | `/v1/me/devices/:device_id/offline-alert` | user token | The rule, or the defaults (`enabled: false`, `after_minutes: 10`) when none is configured. Also returns the evaluator's episode state. |
| `PUT` | `/v1/me/devices/:device_id/offline-alert` | user token | `{enabled, after_minutes, repeat_minutes?, snoozed_until?}`. `after_minutes` is 1…10080 (7 days); omitting it (or sending `0`) stores the default of 10. `repeat_minutes` is 0 (one alert per outage, the default) or 1…10080. `snoozed_until` is an RFC3339 instant; omitting or sending `null` clears an existing snooze. Out-of-range values are `400`. |
| `DELETE` | `/v1/me/devices/:device_id/offline-alert` | user token | `204`. Idempotent: deleting a rule that does not exist is still `204`. |

Response:

```json
{"device_id": "256983850", "enabled": true, "after_minutes": 10, "repeat_minutes": 0,
 "offline_since": "2026-10-16T04:10:17Z", "last_alerted_at": "2026-10-16T04:11:17Z"}
```

A background evaluator (`OFFLINE_ALERT_INTERVAL_SECONDS`, default 30; `0`
disables it) turns presence into an episode state machine:

1. **first** tick that sees the device offline records `offline_since` — this
   is what makes "offline for ten minutes" answerable, because Redis only knows
   whether the heartbeat exists *right now*;
2. while `now - offline_since < after_minutes` nothing happens;
3. once the threshold is crossed the evaluator publishes
   `device.offline.sustained` and stamps `last_alerted_at`. With
   `repeat_minutes: 0` that is the only message for the episode; with a
   positive value it keeps reminding at that interval until the device comes
   back. The event is user-scoped, so it reaches the owner's realtime stream
   and — if a channel subscribes to it — the durable notification outbox;
4. when the device comes back, `offline_since` and `last_alerted_at` are
   cleared, so the **next** outage alerts again instead of being suppressed by
   the first one.

Payload of the alert:

```json
{"device_id": "256983850", "device_name": "办公机", "offline_since": "...",
 "offline_minutes": 11, "threshold_minutes": 10,
 "reminder": false, "repeat_minutes": 0}
```

`reminder` distinguishes "this device just crossed the threshold" from "this is
still down" (`offline_minutes` is `> threshold_minutes` on a reminder), so a
receiver can route the two differently without diffing timestamps.

Four behaviours are worth knowing because the obvious implementation gets
them wrong:

- **A deliberately logged-out device is not an outage.** When
  `devices.logged_in` is false the episode is cleared and nothing is published;
  alerting about a machine the owner parked would only train them to ignore the
  alerts.
- **Changing the threshold does not restart the clock.** `offline_since` is
  preserved across a `PUT`, so raising the threshold from 1 to 10 minutes keeps
  the episode's real start; lowering it re-evaluates immediately.
- **Disabling the rule clears the episode.** Switching it off means "stop
  telling me about this device", so switching it back on gives a clean slate
  rather than an alert about an outage that began while it was off.
- **A snooze silences messages, not the clock.** `snoozed_until` suppresses
  publishing while it is in the future, but `offline_since` keeps running and
  the moment it expires the alert (or the next reminder) goes out — a
  maintenance window should not make the outage invisible, only quiet.

Both `offline_since` and `last_alerted_at` are written with conditional
`UPDATE`s (`... WHERE offline_since IS NULL` and `... WHERE alerted_at IS NULL`),
so two signaling instances cannot both start the same episode or both publish
one alert for it.

### 5.4 Connections (history)

| Method | Path | Body |
|---|---|---|
| `GET` | `/v1/me/connections?cursor=&limit=&since=` | — |
| `POST` | `/v1/me/connections` | `{device_id, duration, status: "success"\|"failed", error_msg?}` |
| `GET` | `/v1/me/recordings?cursor=&limit=` | — |
| `POST` | `/v1/me/recordings` | `{device_id, device_name?, session_id?, status, format?, artifact_path?, artifact_url?, thumbnail_path?, duration_seconds?, size_bytes?, started_at?, ended_at?}` |
| `GET` | `/v1/me/recordings/:id` | — |
| `PATCH` | `/v1/me/recordings/:id` | `{status?, format?, artifact_path?, artifact_url?, thumbnail_path?, duration_seconds?, size_bytes?, ended_at?}` |
| `DELETE` | `/v1/me/recordings/:id` | Deletes the row **and the stored media/preview files** it points at. |
| `POST` | `/v1/me/recordings/:id/artifact` | Media bytes. `multipart/form-data` with a `file` part, or a raw body plus `?ext=.mp4\|.mkv\|.webm\|.h264\|.grf`. |
| `GET` | `/v1/me/recordings/:id/artifact` | Streams the stored media back. Owner only. |
| `POST` | `/v1/me/recordings/:id/artifact-ticket` | Mints `{url, expires_at}` for playback without an `Authorization` header. |
| `POST` | `/v1/me/recordings/:id/thumbnail` | Preview image (JPEG or PNG) for the recordings list. `multipart/form-data` with a `file` part, or a raw body plus `?ext=.jpg\|.jpeg\|.png`. |
| `GET` | `/v1/me/recordings/:id/thumbnail` | Streams the stored preview back. Owner only. |
| `POST` | `/v1/me/recordings/:id/thumbnail-ticket` | Mints `{url, expires_at}` for the preview, same rules as the artifact ticket. |

`session_recordings` is the control-plane index for remote-session media. The
recording engine owns the artifact and reports its metadata through
`POST /v1/me/recordings`. The bytes themselves are pushed with
`POST /v1/me/recordings/:id/artifact` and read back through the matching `GET`
route — there is deliberately **no static directory mount**, every download is
ownership-checked.

Signed playback links: `<video src>`, `window.open` and `<a download>` cannot
carry an `Authorization` header, so those two routes have ticket variants.
`POST .../artifact-ticket` and `POST .../thumbnail-ticket` return a URL of the
form `/v1/recordings/:id/artifact?t=...` (or `/thumbnail?t=...`) signed with
`RECORDING_URL_SECRET` (falling back to `API_KEY`), valid for
`RECORDING_TICKET_TTL_SECONDS` (default `600`). The two routes live outside the
API-key and user-auth groups because a top-level browser navigation carries
neither header; the ticket is the credential. Each ticket is scoped to one
media kind, so a playback link cannot be replayed against the preview route.

Upload behaviour worth knowing:

- the stored file name is derived from `(user_id, recording_id)` and a
  whitelisted extension (`.mp4 .m4v .mkv .webm .h264 .grf`); the uploaded
  filename never reaches the filesystem, so traversal is not possible;
- bytes are written to `<name>.part` and renamed, so a dropped upload cannot
  leave a half-written artifact under the real name;
- the upload sets `artifact_url` (server-relative), `size_bytes` and `format`,
  and flips `status` from `recording` to `completed`. A `failed` status is left
  alone so a client can still report failure;
- the size limit is `RECORDING_MAX_UPLOAD_MB` (default `2048`), and files live
  under `RECORDING_DIR` (default `data/recordings`).

Previews follow the same storage rules with a much smaller budget: the cap is
2 MiB (`min(2 MiB, RECORDING_MAX_UPLOAD_MB)`), the file is named
`thumb-<recording_id>.<ext>` next to the media, and only `.jpg`/`.jpeg`/`.png`
are accepted. Unlike an artifact, the bytes are **sniffed** (JPEG `FF D8 FF`,
PNG signature) and the declared extension must match them, because the
recordings list renders this image inline. A successful upload writes
`thumbnail_url` and removes any preview left under the other extension. Only
the relative URL is stored, so `thumbnail_url` is never an absolute host and
the value survives a redeploy behind a different name.

Status lifecycle: a client opens the row as `recording`, uploads the bytes, and
the server flips it to `completed` when they arrive. If the upload never lands the
client marks the row `failed`. If a client dies mid-capture the row is repaired by
the stale sweep below, so `recording` is never a terminal state.

Retention is opt-in: `RECORDING_RETENTION_DAYS` (default `0` = keep media
forever) deletes media files older than the window while **keeping the index
row**; `artifact_path`/`artifact_url`/`size_bytes` are cleared, so playback
stops being offered and the artifact route answers `404`. A stored preview is
deleted by the same pass and `thumbnail_path`/`thumbnail_url` are cleared with
it, so a purge never leaves an image orphaned on disk. `RECORDING_STALE_MINUTES`
(default `240`) is how long `recording` may stay open before it is marked
`failed`. Both passes run every `RECORDING_SWEEP_INTERVAL_MINUTES` (default `30`)
and can be triggered on demand through the admin endpoints below.

Deleting a recording (`DELETE /v1/me/recordings/:id` or the admin equivalent)
drops the media and the preview along with the row. The same pass also reaps
**orphan files** - stored files under `RECORDING_DIR` whose index row is gone,
for example because an unlink failed earlier - once they are more than an hour
old. Upload temporaries (`.part`) are never matched, so an in-flight upload
cannot be reaped.

### 5.5 Favorites

| Method | Path | Body |
|---|---|---|
| `GET` | `/v1/me/favorites` | — |
| `POST` | `/v1/me/favorites` | `{device_id, device_name?, access_password?}` |
| `PATCH` | `/v1/me/favorites/:device_id` | `{device_name?, access_password?}` |
| `DELETE` | `/v1/me/favorites/:device_id` | — |

`FavoriteItem` shape: `{device_id, device_name, access_password, created_at}`.

---

## 6. Devices

These endpoints serve the **device side** (Chromium host, headless agents,
official Qt instances acting on behalf of the host). All require
`X-API-Key`; most additionally require `Authorization: Bearer <device_secret>`.

### 6.1 `POST /v1/devices:provision`

First-run provisioning. Allocates `(device_id, device_secret)` and persists
the secret hash.

Auth: `X-API-Key` only.

Request:

```json
{
  "device_uuid":  "01H...-deterministic-from-hardware",
  "os":           "win",
  "os_version":   "11",
  "app_version":  "2.10.0"
}
```

Response (200):

```json
{
  "device_id":     "176017615",
  "device_secret": "<plaintext, returned only here>",
  "device_uuid":   "01H...",
  "issued_at":     "2026-05-12T08:52:39Z"
}
```

Re-provisioning the same `device_uuid` is allowed and rotates the secret
(old secret becomes invalid). Owner / access-code / `logged_in_intent` are
preserved.

### 6.2 `POST /v1/devices/:device_id/heartbeat`

Auth: `X-API-Key` + `Authorization: Bearer <device_secret>`.

Request (all fields optional):

```json
{ "app_version": "2.10.0", "os": "win", "stats": { "cpu": 12.5 } }
```

Response:

```json
{
  "server_time":                "2026-05-12T08:52:39Z",
  "turn_config_version":        4,
  "suggested_heartbeat_interval_sec": 30
}
```

The server may set a `Retry-After: <n>` header to throttle the host. Clients
**must** honor it.

The host watches `turn_config_version` for changes; on bump, it re-fetches
`/v1/ice-config` (this is the only signal a host needs to pick up admin TURN
changes — see §2.19 of the design doc).

### 6.3 `POST /v1/devices/:device_id/signal-tokens`

Auth: `X-API-Key` + `Authorization: Bearer <device_secret>`.

Issues a one-shot `signal_token` valid for 300 seconds, used by the host to
authenticate its `/v1/realtime/signal` first frame. **Hosts must request a
fresh token before every WS connection — never cache.**

Response:

```json
{ "signal_token": "stk_01H...", "expires_at": "2026-05-12T08:57:39Z" }
```

### 6.4 `PUT /v1/devices/:device_id/access-code`

Auth: `X-API-Key` + `Authorization: Bearer <device_secret>`.

Reports the current plaintext access code. Issued by Qt (which receives the
`device_secret` over native-messaging from the Chromium host at process
startup; held only in Qt process memory, never persisted).

Request:

```json
{ "access_code": "633732" }
```

Response: `204 No Content`. Server publishes `device.access_code.changed` to
the owning user's realtime events stream (the event payload **never** contains
the plaintext code — only a `changed` flag).

### 6.5 `POST /v1/devices/:device_id/access-code:verify`

The **client-side** verify entrypoint. Used by the WebClient (browser) and Qt
to validate an access code and obtain a one-shot `signal_token`.

`code` may be the device's access code **or** the code of an active access grant
(§5.3.4). The device policy gate runs first either way, and a grant only spends
one of its uses once a session actually starts.

Auth: `X-API-Key` **or** `Origin` in the admin-configured `allowed_origins`
whitelist (whichever the request can satisfy). Browsers depend on the second
form.

Request:

```json
{ "code": "633732" }
```

Response (200):

```json
{ "signal_token": "stk_01H...", "expires_at": "2026-05-12T08:53:39Z" }
```

When the device policy has `approval_required` (§5.3.3), a correct code returns
`202` instead, and no token is minted until the owner approves and the
controller claims the approval (§5.3.5):

```json
{
  "approval_id": 42,
  "approval_secret": "9c5cabcd...",
  "status": "pending",
  "expires_at": "2026-10-16T00:40:18Z",
  "poll_after_ms": 2000
}
```

Error path:

| HTTP | `code` | Meaning |
|---|---|---|
| `404` | `DEVICE_NOT_FOUND` | No such `device_id` registered |
| `409` | `HOST_OFFLINE` | Device exists but is not online (no host hb + wsconn) |
| `403` | `INVALID_CODE` | Access code is wrong (counts toward rate-limit budget) |
| `403` | `TOO_MANY_ATTEMPTS` | Rate limit exceeded; honor the `Retry-After` header |

Rate limits (Redis `INCR`-based, 60-second windows):

- `(device_id, ip)` — 5 wrong attempts → 403 `TOO_MANY_ATTEMPTS`
- `ip` (any device) — 60 total attempts → 429
- `device_id` (any ip) — 30 wrong attempts → 403

### 6.6 `GET /v1/ice-config`

Returns the global ICE configuration. Auth: `X-API-Key` (or hosts may
additionally present `Authorization: Bearer <device_secret>`).

```json
{
  "ice_servers": [
    { "urls": ["stun:stun.l.google.com:19302"] },
    { "urls": ["turn:turn.example.com:3478"], "username": "...", "credential": "..." }
  ],
  "turn_config_version": 4
}
```

---

## 7. WebSockets

Two long-lived endpoints. Both authenticate via **first-frame auth** because
browsers can't attach custom headers to WS upgrades.

Both endpoints `close(4401)` connections that don't send a valid auth frame
within 5 seconds of the upgrade.

### 7.1 `GET /v1/realtime/events` — user event stream

Used by Qt and the WebClient to receive snapshot + incremental updates for
my devices and favorites.

Handshake:

```
1. Client opens WS.
2. Client sends:   {"type":"auth", "access_token":"...", "since_rev": 4217?}
3. Server replies: {"type":"auth_ok", "server_rev": 4218}
4. Server immediately sends a snapshot OR (if `since_rev` is recent enough)
  replays missed events from Redis stream `qd:events:user:{user_id}`.
5. Server sends:  {"type":"bootstrap_done", "server_rev":4218}
```

Snapshot frame:

```json
{
  "type": "snapshot",
  "server_rev": 4218,
  "data": {
    "devices":   [DeviceItem, ...],
    "favorites": [FavoriteItem, ...]
  }
}
```

Clients must **replace** local state on snapshot — never merge. Subsequent
events with `device.*` / `favorite.*` types are patches and must be applied
in-place. Clients must apply snapshot/replay/events only when their
`server_rev` is newer than the last applied revision; stale frames are ignored.
Clients must not depend on HTTP `/v1/me/devices` responses to order or repair
realtime state, because HTTP and WS frames can arrive in either order.

If the gap is too large (Redis stream has been trimmed), the server replies
`{type:"snapshot_required"}` and then sends a fresh authoritative `snapshot`
on the same connection. Clients may also choose to reconnect without
`since_rev`, but they must remain compatible with the same-connection snapshot
path.

`session.revoked` events on this stream mean the user's session has been
terminated by an admin (or family rotation detected a leak). Clients should
clear local credentials and exit to login.

### 7.2 `GET /v1/realtime/signal` — SDP/ICE relay

Used by hosts and clients (browser / native) for WebRTC signaling.

Handshake:

```
1. Client/host opens WS.
2. First frame:
   {"type":"auth", "signal_token":"...", "role":"host"|"client",
    "device_id":"176017615", "client_id":"cli-..." }     // client_id for clients only
3. Server replies: {"type":"auth_ok", "session_id":"..."} on success.
4. SDP/ICE relay begins (existing jingle envelope reused).
```

Sending SDP/ICE before `auth_ok` causes the server to drop and close the
connection. `signal_token` is single-use — it is `GETDEL`'d at first-frame
auth and may not be re-presented on a different WS.

A host that goes offline mid-session causes the server to push
`{"type":"error", "code":"PEER_DISCONNECTED"}` to all bound clients and close
their WS. The client should record a `connections` entry with
`status: "failed"`.

---

## 8. Admin surface

Mounted at `/v1/admin/*`. All endpoints require an admin access token; writes
additionally require 2FA-enabled super-admin (see §2.16 of the design doc).
The admin web SPA in [`SignalingServer/web/`](../web/) is the reference
consumer. Client implementers should consult the admin web source for exact
request shapes; this section documents only the route table.

### 8.1 Admin auth

| Method | Path | Body | Notes |
|---|---|---|---|
| `POST` | `/v1/admin/auth/sessions` | `{username, password, totp_code?}` | If account has 2FA enabled and `totp_code` is omitted, server returns `401 TOTP_REQUIRED` with `pre_token`. |
| `POST` | `/v1/admin/auth/sessions:totp` | `{pre_token, totp_code}` | Two-step completion. |
| `POST` | `/v1/admin/auth/tokens:refresh` | `{refresh_token}` | |
| `DELETE` | `/v1/admin/auth/sessions/current` | — | |

### 8.2 Admin accounts (CRUD over admins themselves)

| Method | Path |
|---|---|
| `GET` | `/v1/admin/admins` |
| `POST` | `/v1/admin/admins` |
| `GET` | `/v1/admin/admins/:id` |
| `PATCH` | `/v1/admin/admins/:id` |
| `DELETE` | `/v1/admin/admins/:id` |
| `POST` | `/v1/admin/admins/me/2fa/setup` |
| `POST` | `/v1/admin/admins/me/2fa/verify` |
| `DELETE` | `/v1/admin/admins/me/2fa` |

### 8.3 Business users

| Method | Path | Notes |
|---|---|---|
| `GET` | `/v1/admin/users` | Filters: `channel_type`, `status`, `level`, `q`. |
| `POST` | `/v1/admin/users` | |
| `POST` | `/v1/admin/users:batch` | `{ids: [...], op: "enable"\|"disable"\|"delete"\|"set_level"}` |
| `GET` | `/v1/admin/users/:id` | |
| `GET` | `/v1/admin/users/:id/details` | Includes bound devices + active sessions. |
| `PATCH` | `/v1/admin/users/:id` | |
| `DELETE` | `/v1/admin/users/:id` | Cascades: `UPDATE devices SET user_id=NULL, logged_in_intent=false WHERE user_id=?`. |
| `POST` | `/v1/admin/users/:id/sessions:revoke` | Kicks every session of that user. |
| `PATCH` | `/v1/admin/users/:id/device-count` | `{deviceCount: N}` (kept camelCase for backward compat — the only such field). |

### 8.4 Devices

| Method | Path | Notes |
|---|---|---|
| `GET` | `/v1/admin/devices` | |
| `POST` | `/v1/admin/devices:batch` | `{ids: [...], op: "delete"\|"assign_group"\|"remove_group"}` |
| `GET` | `/v1/admin/devices/:device_id` | |
| `DELETE` | `/v1/admin/devices/:device_id` | Hard delete. |
| `POST` | `/v1/admin/devices/:device_id/unbind` | Force-unbind, keep device record. |
| `POST` | `/v1/admin/devices/:device_id/secret:rotate` | Invalidates `device_secret`; host re-provisions on next 401. |

### 8.5 Bindings / groups / stats / audit / preset / settings / webhooks

| Method | Path |
|---|---|
| `GET` | `/v1/admin/device-bindings` |
| `GET` `POST` `PATCH` `DELETE` | `/v1/admin/groups[/:id]` |
| `GET` `POST` `DELETE` | `/v1/admin/groups/:id/devices` |
| `GET` | `/v1/admin/stats` |
| `GET` | `/v1/admin/system/status` |
| `GET` | `/v1/admin/connections` |
| `GET` | `/v1/admin/activity` |
| `GET` | `/v1/admin/trends?range=24h\|7d\|30d` |
| `GET` | `/v1/admin/audit-logs` (filters `date_from` / `date_to` / `q`) |
| `GET` `DELETE` | `/v1/admin/recordings[/:id]` (list filters `user_id` / `device_id` / `status` / `q`; delete writes audit action `recording.delete`) |
| `GET` | `/v1/admin/recordings/:id/artifact` (cross-user download; audit action `recording.download`) |
| `GET` | `/v1/admin/recordings/:id/thumbnail` (cross-user preview; audit action `recording.thumbnail.download`) |
| `GET` | `/v1/admin/recordings:storage` — `{artifact_count, total_bytes, retention_days, stale_after_minutes}` |
| `POST` | `/v1/admin/recordings:sweep` — runs the stale + retention + orphan pass now; returns `{stale_marked_failed, artifacts_purged, bytes_freed, orphans_removed}` and audits `recording.sweep` |
| `GET` | `/v1/admin/notification-deliveries:stats` — `{total, by_status, retention_days, archive_dir}`; `by_status` always carries all four states (`pending`/`delivering`/`delivered`/`failed`), zeroes included |
| `POST` | `/v1/admin/notification-deliveries:sweep` — runs the outbox retention pass now. Optional body `{older_than_days: N}` overrides the configured window. Returns `{purged, older_than_days, cutoff, archive_dir}` and audits `notification_delivery.sweep`. `400` when neither the config nor the body supplies a window; on an archive write failure nothing is deleted and the response is an error |
| `GET` `PUT` | `/v1/admin/preset` |
| `GET` `PUT` | `/v1/admin/settings` |
| `GET` `POST` `PATCH` `DELETE` | `/v1/admin/webhooks[/:id]` |
| `POST` | `/v1/admin/webhooks/:id/test` |

> **Why `webhooks/:id/test` and not `webhooks/:id:test`** — gin's httprouter
> cannot register a literal suffix on a wildcard segment. The AIP-136-style
> `:test` would silently fail to register. Sub-resource form is RESTful and
> consistent with what gin can express. See §6 W1 in the design doc for the
> historical record. All other "colon action" endpoints (`sessions:sms`,
> `tokens:refresh`, `users:batch`, `secret:rotate`, …) have a
> literal segment before the colon and work fine. The `2fa/setup` and
> `2fa/verify` endpoints are exceptions — they use sub-resource form because
> `2fa:setup` vs `2fa:verify` would be two competing wildcards on the same
> segment (same issue as `webhooks/:id:test`).

---

## 9. Error format

Every non-2xx response is RFC 7807 problem-details with
`Content-Type: application/problem+json`:

```json
{
  "type":     "https://quickdesk.io/problems/device-not-found",
  "title":    "Device not found",
  "status":   404,
  "detail":   "Device 176017615 is not registered",
  "instance": "/v1/me/devices/176017615",
  "code":     "DEVICE_NOT_FOUND",
  "trace_id": "01H..."
}
```

Common `code` values:

| Code | HTTP | When |
|---|---|---|
| `INVALID_REQUEST` | 400 | Body / query validation failed |
| `UNAUTHORIZED` | 401 | Missing or malformed auth |
| `TOKEN_EXPIRED` | 401 | `access_token` expired — try refresh once |
| `TOKEN_INVALID` | 401 | `access_token` revoked or never existed |
| `REFRESH_INVALID` | 401 | `refresh_token` expired / leaked / replayed |
| `TOTP_REQUIRED` | 401 | Admin login needs second factor |
| `FORBIDDEN` | 403 | Authenticated but not allowed |
| `INVALID_CODE` | 403 | Wrong access code (counts toward rate budget) |
| `TOO_MANY_ATTEMPTS` | 403 | Rate limit hit; see `Retry-After` |
| `DEVICE_NOT_FOUND` | 404 | No such `device_id` |
| `NOT_FOUND` | 404 | Resource not found |
| `CONFLICT` | 409 | Generic conflict |
| `HOST_OFFLINE` | 409 | Verify reached an offline host |
| `RATE_LIMITED` | 429 | IP-level rate limit |
| `INTERNAL` | 500 | Server bug — file an issue with `trace_id` |
| `UNAVAILABLE` | 503 | Component degraded; see `/health` |

Clients must read both `status` and `code`. Display `detail` to the user when
no localized i18n key matches `code`.

---

## 10. Rate limits

| Surface | Window | Quota |
|---|---|---|
| `POST /v1/verification-codes` per phone | 1 min / 10 min / 24 h | 1 / 3 / 10 |
| `POST /v1/devices/:id/access-code:verify` per `(device, ip)` (errors only) | 60 s | 5 |
| same per `ip` (any device) | 60 s | 60 |
| same per `device_id` (errors only, any ip) | 60 s | 30 |
| `POST /v1/devices/:id/heartbeat` per device | min interval | 1 s |
| `POST /v1/devices/:id/signal-tokens` per device | min interval | 1 s |

Hits return `403 TOO_MANY_ATTEMPTS` or `429 RATE_LIMITED` with `Retry-After`.

---

## 11. Realtime event types

Pushed over `/v1/realtime/events` to the owning user. Event payloads never
include access-code plaintext, `device_secret`, or `refresh_token`.

| `type` | When | Payload sketch |
|---|---|---|
| `device.online.changed` | hb-TTL expiry or wsconn DEL → `online` flips | `{device_id, online}` |
| `device.offline.sustained` | the device stayed offline past its owner-set threshold (§5.3.6), published once per outage | `{device_id, device_name, offline_since, offline_minutes, threshold_minutes}` |
| `device.session.updated` | `logged_in_intent` flipped (bind / clear-session) | `{device_id, online, logged_in}` |
| `device.access_code.changed` | host reported a new code | `{device_id, changed: true}` (no plaintext) |
| `device.bound` / `device.added` | new device for this user | `{device_id}` (clients fetch detail once if not present) |
| `device.ownership.lost` | another account took over this device | `{device_id}` |
| `device.unbound` | unbound by user or admin | `{device_id}` |
| `device.remark.changed` / `device.display_name.changed` | `PATCH /v1/me/devices/:id` | `{device_id, remark?, device_name?}` |
| `favorite.added` / `favorite.updated` / `favorite.removed` | favorites mutated | `{device_id, ...}` |
| `recording.created` | `POST /v1/me/recordings` | `{recording_id, device_id, session_id, status, format}` |
| `recording.updated` | `PATCH /v1/me/recordings/:id` | `{recording_id, device_id, session_id, status}` |
| `recording.deleted` | `DELETE /v1/me/recordings/:id` | `{recording_id}` |
| `session.revoked` | this user's session was killed | `{}` |
| `device.secret.rotated` | admin rotated `device_secret` (sent to host owner) | `{device_id}` (host should re-provision on next 401) |
| `device.access.granted` / `device.access.denied` / `device.access.session_started` | an access attempt or session start on this user's device (§5.3.1) | `{device_id, client_id, ip, result, reason?, log_id}` |
| `device.access_policy.changed` | owner changed the device access policy (§5.3.3) | `{device_id, remote_disabled, allow_ips, max_sessions, allowed_windows, timezone, exception_dates}` |
| `device.access_grant.changed` | owner created or revoked a guest grant (§5.3.4) | `{action: "created"\|"revoked", grant_id, label, expires_at, max_uses, uses}` (never the code) |
| `device.access_grant.used` | a grant just admitted a session | `{grant_id, label, uses, max_uses, client_id}` |
| `device.access_approval.requested` | a correct code is waiting for the owner's yes/no (§5.3.5) | `{approval_id, client_id, client_ip, source, trigger, expires_at}`, where `trigger` is `always` / `grant` / `untrusted_ip` (never the secret) |
| `device.access_approval.decided` | owner approved or denied a waiting session | `{approval_id, status: "approved"\|"denied", source, client_id, reason?}` |

Event envelope (always present):

```json
{
  "id":         "evt_01HXY...",
  "type":       "device.session.updated",
  "ts":         "2026-05-12T08:52:39Z",
  "server_rev": 4218,
  "data":       { ... }
}
```

`server_rev` increments per published event and supports the resume protocol
(§7.1).

---

## Appendix: removed pre-refactor routes

For migration awareness only. Do not call these on a v1 server — they 404.

```
/api/v1/user/register                  →  /v1/auth/register
/api/v1/user/login                     →  /v1/auth/sessions
/api/v1/user/login-sms                 →  /v1/auth/sessions:sms
/api/v1/user/logout                    →  DELETE /v1/me/sessions/current
/api/v1/user/reset-password            →  /v1/auth/password-resets[:confirm]
/api/v1/user/me                        →  /v1/me
/api/v1/user/devices                   →  /v1/me/devices
/api/v1/user/devices/unbind            →  DELETE /v1/me/devices/:id
/api/v1/user/devices/auto-bind         →  POST /v1/me/devices
/api/v1/user/devices/record            →  POST /v1/me/connections
/api/v1/user/devices/logs              →  GET /v1/me/connections
/api/v1/user/favorites                 →  /v1/me/favorites
/api/v1/devices/register               →  /v1/devices:provision
/api/v1/devices/:id/status             →  derived from realtime events
/api/v1/auth/verify                    →  /v1/devices/:id/access-code:verify
/api/v1/ice-config                     →  /v1/ice-config
/api/v1/sms/send                       →  /v1/verification-codes
/signal/:device_id                     →  /v1/realtime/signal (first-frame auth)
/host/:device_id                       →  /v1/realtime/signal
/client/:device_id/:access_code        →  /v1/realtime/signal (after :verify)
/api/v1/user/sync?token=               →  /v1/realtime/events (first-frame auth)
```
