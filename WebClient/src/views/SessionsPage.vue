<template>
  <div>
    <h2 class="page-title">{{ $t('sessions.title') }}</h2>

    <div v-if="!authState.isLoggedIn" class="card">
      <div class="empty-state">
        <div class="empty-icon">🔒</div>
        <p>{{ $t('sessions.loginRequired') }}</p>
        <button class="btn btn-primary btn-sm" style="margin-top:12px;" @click="showLogin()">
          {{ $t('user.login') }}
        </button>
      </div>
    </div>

    <template v-else>
      <div class="toolbar">
        <div class="segmented" role="group" :aria-label="$t('sessions.title')">
          <button
            v-for="option in statusOptions"
            :key="option.value"
            :class="['segment', { active: status === option.value }]"
            @click="setStatus(option.value)"
          >{{ $t(option.label) }}</button>
        </div>
        <select v-model="deviceFilter" class="select" @change="refresh">
          <option value="">{{ $t('sessions.device.all') }}</option>
          <option v-for="d in devices" :key="d.device_id" :value="d.device_id">
            {{ deviceName(d) }}
          </option>
        </select>
        <button class="btn btn-secondary btn-sm" :disabled="loading" @click="refresh">
          🔄 {{ $t('sessions.refresh') }}
        </button>
        <!-- QD-909: the audit trail was write-only until now - transfers could be
             reported but never read back from the client. Toggling rather than
             always-open because the list is only interesting when somebody is
             investigating, and it costs a request. -->
        <button
          class="btn btn-secondary btn-sm"
          :class="{ active: auditOpen }"
          @click="toggleAudit"
        >{{ $t('sessions.transferAudit') }}</button>
        <button
          v-if="deviceFilter"
          class="btn btn-danger btn-sm"
          :disabled="clearBusy"
          @click="clearDevice"
        >{{ clearBusy ? '...' : $t('sessions.clearDevice') }}</button>
      </div>

      <div v-if="auditOpen" class="card audit-panel">
        <div class="audit-panel-heading">
          <div class="audit-panel-title">{{ $t('sessions.transferAuditTitle') }}</div>
          <!-- QD-110: the owner can read this trail, and now take it with them.
               Offered whenever the panel is open, including on an empty list, so
               the control is findable rather than something that appears only
               once there is something to export. -->
          <button
            class="btn btn-secondary btn-sm"
            :disabled="auditExporting"
            @click="exportAudit"
          >{{ auditExporting ? '...' : $t('sessions.exportAudit') }}</button>
        </div>
        <!-- A ref is unwrapped by the template, so these are `audit.loading` and
             not `audit.loading.value` - the latter is the script-side spelling and
             renders nothing here. -->
        <div v-if="auditLoading" class="hint">{{ $t('sessions.loading') }}</div>
        <!-- A failed read is shown as a failure, not as an empty list: "no
             transfers" and "we could not ask" are different facts. -->
        <div v-else-if="auditError" class="hint audit-error">
          {{ $t('sessions.auditReadFailed') }}: {{ auditError }}
        </div>
        <div v-else-if="auditRows.length === 0" class="hint">
          {{ $t('sessions.auditEmpty') }}
        </div>
        <template v-else>
        <!-- QD-110: retention moves expired rows into per-day files, so an empty
             live table is not the whole answer. The label sits above the rows
             rather than in place of them: these rows exist, they have just left
             the live table, and showing the label instead of them would be the
             same "empty" lie one branch up. -->
        <div v-if="auditSource === 'archive'" class="hint">
          {{ $t('sessions.auditFromArchive') }}
        </div>
        <div class="audit-list">
          <div v-for="row in auditRows" :key="row.id" class="audit-row">
            <span>{{ formatTime(row.at) }}</span>
            <span>{{ auditRowLabel(row) }}</span>
            <span v-if="row.error" class="audit-row-error">{{ row.error }}</span>
            <span v-else-if="row.status === 'failed'" class="audit-row-error">
              {{ $t('sessions.auditFailed') }}
            </span>
          </div>
        </div>
        </template>
      </div>

      <div v-if="deviceFilter" class="card ban-panel">
        <div class="ban-panel-title">{{ $t('sessions.blockedSources') }}</div>
        <p v-if="blockedSources.length === 0" class="hint">{{ $t('sessions.noBlockedSources') }}</p>
        <div v-else class="ban-list">
          <span v-for="source in blockedSources" :key="source" class="ban-chip">
            <code>{{ source }}</code>
            <button
              class="btn btn-secondary btn-sm"
              :disabled="banRemoving === source"
              @click="unbanSource(source)"
            >{{ banRemoving === source ? '...' : $t('sessions.unbanSource') }}</button>
          </span>
        </div>
      </div>

      <!-- Guest grants (QD-107). The server has served create/list/delete all
           along and neither control end could reach them, so the only way to
           issue a bounded code was a hand-written request. A live row has to be
           distinguishable from a revoked or expired one: showing a dead code as
           usable sends the owner to hand it to a guest who is then refused. -->
      <div v-if="deviceFilter" class="card ban-panel">
        <div class="ban-panel-title">{{ $t('sessions.guestGrants') }}</div>
        <p class="hint">{{ $t('sessions.guestGrantsHint') }}</p>

        <div class="grant-form">
          <input
            v-model="grantForm.label"
            class="form-input"
            type="text"
            :placeholder="$t('sessions.grantLabelPlaceholder')"
          >
          <input
            v-model.number="grantForm.expiresInMinutes"
            class="form-input form-input-narrow"
            type="number"
            min="1"
            step="1"
            :placeholder="$t('sessions.grantExpires')"
          >
          <input
            v-model.number="grantForm.maxUses"
            class="form-input form-input-narrow"
            type="number"
            min="0"
            step="1"
            :placeholder="$t('sessions.grantMaxUses')"
          >
          <input
            v-model="grantForm.boundClientId"
            class="form-input"
            type="text"
            :placeholder="$t('sessions.grantBoundClient')"
          >
          <input
            v-model="grantForm.boundIps"
            class="form-input"
            type="text"
            :placeholder="$t('sessions.grantBoundIpsPlaceholder')"
          >
          <button class="btn btn-primary btn-sm" :disabled="grantBusy" @click="createGrant">
            {{ grantBusy ? '...' : $t('sessions.createGrant') }}
          </button>
        </div>

        <p v-if="grants.length === 0" class="hint">{{ $t('sessions.grantsEmpty') }}</p>
        <div v-else class="grant-list">
          <div v-for="grant in grants" :key="grant.id" class="grant-row">
            <code class="grant-code">{{ grant.code }}</code>
            <span class="grant-meta">
              {{ $t('sessions.grantRemaining') }}:
              {{ grant.unlimited ? $t('sessions.grantUnlimited') : grant.remainingUses }}
            </span>
            <span class="grant-meta">{{ $t('sessions.grantExpiresAt') }}: {{ formatGrantTime(grant.expiresAt) }}</span>
            <span class="grant-meta">{{ $t('sessions.grantBinding') }}: {{ grant.binding }}</span>
            <span v-if="!grant.usable" class="grant-dead">{{ $t(grant.stateKey) }}</span>
            <button
              v-if="grant.usable"
              class="btn btn-secondary btn-sm"
              :disabled="grantBusy"
              @click="revokeGrant(grant)"
            >{{ $t('sessions.revokeGrant') }}</button>
          </div>
        </div>
      </div>

      <p v-if="notice" class="notice">{{ notice }}</p>
      <p v-if="error" class="hint error">{{ error }}</p>

      <div class="card">
        <div v-if="loading && sessions.length === 0" class="empty-state">
          <div class="empty-icon">⏳</div>
          <p>{{ $t('sessions.loading') }}</p>
        </div>
        <div v-else-if="sessions.length === 0" class="empty-state">
          <div class="empty-icon">🛰️</div>
          <p>{{ emptyText }}</p>
          <p class="hint">{{ $t('sessions.emptyHint') }}</p>
        </div>
        <div v-else>
          <div v-for="row in sessions" :key="row.session_id" class="session-item">
            <div :class="['status-dot', row.active ? 'active' : 'ended']"></div>
            <div class="session-info" @click="toggleDetail(row)">
              <div class="session-name">
                {{ deviceNameById(row.device_id) }}
                <span :class="['badge', row.active ? 'active' : 'ended']">
                  {{ row.active ? $t('sessions.active') : $t('sessions.ended') }}
                </span>
                <span v-if="!row.active && row.end_reason" class="badge reason">{{ row.end_reason }}</span>
              </div>
              <div class="session-meta">
                <span>{{ formatTime(row.started_at) }}</span>
                <span v-if="row.duration_seconds != null">{{ formatDuration(row.duration_seconds) }}</span>
                <span v-if="row.client_id">{{ row.client_id }}</span>
                <span v-if="row.client_ip">{{ row.client_ip }}</span>
              </div>
            </div>
            <div class="session-actions">
              <button class="btn btn-secondary btn-sm" @click="toggleDetail(row)">
                {{ expanded === row.session_id ? $t('sessions.hideDetails') : $t('sessions.details') }}
              </button>
              <button
                v-if="row.active"
                class="btn btn-danger btn-sm"
                :disabled="busy === row.session_id"
                @click="revoke(row)"
              >{{ busy === row.session_id ? '...' : $t('sessions.revoke') }}</button>
              <button
                v-if="row.active && row.client_ip"
                class="btn btn-secondary btn-sm"
                :disabled="busy === row.session_id"
                @click="banSource(row)"
              >{{ $t('sessions.banSource') }}</button>
            </div>

            <div v-if="expanded === row.session_id" class="session-detail">
              <div v-if="detailLoading" class="hint">{{ $t('sessions.loading') }}</div>
              <template v-else-if="detail">
                <div class="detail-line">
                  <b>{{ $t('sessions.recordings') }}:</b> {{ describedDetail.counts.recordings }}
                </div>
                <!-- The recordings themselves, not just how many there are: a count
                     tells the owner one exists, it does not tell them which, when, or
                     whether there is anything to play. -->
                <div v-if="describedDetail.recordings.length" class="detail-list">
                  <div v-for="rec in describedDetail.recordings" :key="rec.id" class="detail-row">
                    <span>{{ formatTime(rec.createdAt) }}</span>
                    <span>{{ rec.status }}</span>
                    <span v-if="rec.duration">{{ rec.duration }}</span>
                    <button
                      v-if="rec.playable"
                      class="btn btn-secondary btn-sm"
                      @click="playSessionRecording(rec)"
                    >{{ $t('sessions.playRecording') }}</button>
                    <span v-else class="hint">{{ $t('sessions.noMedia') }}</span>
                  </div>
                </div>
                <div class="detail-line">
                  <b>{{ $t('sessions.accessLog') }}:</b> {{ describedDetail.counts.accessLog }}
                </div>
                <div v-if="describedDetail.accessLog.length" class="detail-list">
                  <div v-for="log in describedDetail.accessLog" :key="log.id" class="detail-row">
                    <span>{{ formatTime(log.createdAt) }}</span>
                    <span>{{ log.result }}</span>
                    <span>{{ log.ip }}</span>
                  </div>
                </div>
              </template>
            </div>
          </div>
          <div v-if="hasMore" style="text-align:center;margin-top:12px;">
            <button class="btn btn-secondary btn-sm" :disabled="loading" @click="loadMore">
              {{ $t('sessions.loadMore') }}
            </button>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { userApi } from '../api/userApi'
import { userSync } from '../api/userSync'
import { describeSessionDetail, resolveRecordingUrl } from '../sessionDetailModel'
import { buildAccessGrantPayload, grantRow } from '../api/accessGrantModels'
import { useSessionAudit } from '../composables/useSessionAudit'
import { auditRowLabel } from '../api/auditLogModel'

const { t } = useI18n()
const showLogin = inject('showLogin')
const authState = inject('authState')

const sessions = ref([])
const nextCursor = ref('')
const loading = ref(false)
const error = ref('')
const notice = ref('')
const status = ref('active')
const deviceFilter = ref('')
const devices = ref([])
const expanded = ref('')
const detail = ref(null)
const detailLoading = ref(false)

// QD-208: the server sends the recordings and access-log rows themselves, so the
// detail is described for display rather than read raw in the template. The
// arrays win over the counts when the two disagree, because the rows are what the
// owner acts on.
const describedDetail = computed(() => (detail.value ? describeSessionDetail(detail.value) : null))
const busy = ref('')
const clearBusy = ref(false)
const blockedSources = ref([])
// Guest grants (QD-107). Defaults are deliberately conservative: a one-hour
// window and one session, so an accidental submit does not mint an open-ended
// code for the device.
const grants = ref([])
const grantBusy = ref(false)
const grantForm = ref({ label: '', expiresInMinutes: 60, maxUses: 1, boundClientId: '', boundIps: '' })
const banRemoving = ref('')

// QD-909: the data-channel audit trail, read back for the first time. See the
// composable's header for why this is labelled as the account's trail rather
// than the session's - the row's resource id is a transfer id, and the server has
// no session filter, so the narrower question is not one the contract answers.
//
// The refs are unwrapped here rather than reached through in the template. Vue
// unwraps a ref returned from setup, but not a ref nested inside an object that
// setup returned: `audit.loading` in a template reads the Ref wrapper itself,
// which is always truthy, so a loading panel would never leave its loading state.
// Spreading the refs into top-level names is what makes the template see values.
const auditState = useSessionAudit()
const auditRows = auditState.rows
const auditLoading = auditState.loading
const auditError = auditState.error
const auditLoaded = auditState.loaded
const auditSource = auditState.source

const statusOptions = [
  { value: 'active', label: 'sessions.filter.active' },
  { value: 'ended', label: 'sessions.filter.ended' },
  { value: 'all', label: 'sessions.filter.all' },
]

const hasMore = computed(() => !!nextCursor.value)
const emptyText = computed(() => {
  if (status.value === 'active') return t('sessions.emptyActive')
  if (status.value === 'ended') return t('sessions.emptyEnded')
  return t('sessions.empty')
})

function deviceName(d) {
  return d?.remark || d?.device_name || d?.device_id || ''
}

function deviceNameById(id) {
  return deviceName(devices.value.find((d) => d.device_id === id)) || id
}

function formatTime(ts) {
  if (!ts) return ''
  return new Date(ts).toLocaleString()
}

function formatDuration(seconds) {
  const s = Math.max(0, Math.floor(seconds || 0))
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = s % 60
  const pad = (n) => String(n).padStart(2, '0')
  return h > 0 ? `${h}:${pad(m)}:${pad(sec)}` : `${m}:${pad(sec)}`
}

const auditOpen = ref(false)
const auditExporting = ref(false)

// QD-110: the same rows the panel lists, as a file. The server names it from its
// own clock, so the file is named by when the export was made rather than by
// whatever the browser thought the time was.
async function exportAudit() {
  auditExporting.value = true
  try {
    const r = await userApi.exportAuditLogs({ format: 'csv' })
    if (!r.ok) {
      showToast(r.error || t('sessions.auditExportFailed'), 'error')
      return
    }
    const url = URL.createObjectURL(r.blob)
    const link = document.createElement('a')
    link.href = url
    link.download = r.filename
    link.click()
    URL.revokeObjectURL(url)
  } finally {
    auditExporting.value = false
  }
}

async function toggleAudit() {
  auditOpen.value = !auditOpen.value
  // Loaded on first open, and on every open after a failure, so a retry is one
  // click rather than a page reload. A successful read is not repeated.
  if (auditOpen.value && !auditLoaded.value && !auditLoading.value) {
    await auditState.load()
  }
}

async function loadDevices() {
  const r = await userApi.fetchMyDevices()
  if (r.ok) devices.value = r.data?.items ?? []
}

async function load(reset = true) {
  if (loading.value) return
  loading.value = true
  error.value = ''
  const r = await userApi.fetchRemoteSessions({
    status: status.value,
    deviceId: deviceFilter.value,
    cursor: reset ? '' : nextCursor.value,
    limit: 50,
  })
  loading.value = false
  if (sessionRefreshPending) {
    sessionRefreshPending = false
    scheduleSessionRefresh()
  }
  if (!r.ok) {
    error.value = r.error || t('sessions.loadFailed')
    return
  }
  const items = r.data?.items ?? []
  sessions.value = reset ? items : [...sessions.value, ...items]
  nextCursor.value = r.data?.next_cursor || ''
}

async function refresh() {
  notice.value = ''
  expanded.value = ''
  detail.value = null
  await load(true)
  await loadBlockedSources()
  await loadGrants()
}

function loadMore() {
  return load(false)
}

function setStatus(value) {
  if (status.value === value) return
  status.value = value
  refresh()
}

async function toggleDetail(row) {
  if (expanded.value === row.session_id) {
    expanded.value = ''
    detail.value = null
    return
  }
  expanded.value = row.session_id
  detail.value = null
  detailLoading.value = true
  const r = await userApi.fetchRemoteSession(row.session_id)
  if (expanded.value !== row.session_id) return
  detailLoading.value = false
  if (r.ok) detail.value = r.data
}

// Playing a recording from the session detail. A browser cannot attach the Bearer
// token to a <video src> or window.open, so the server is asked for a short-lived
// signed URL first; if tickets are unavailable on this deployment the plain
// artifact URL still works when the server is reached without the API-key gate.
async function playSessionRecording(rec) {
  if (!rec?.playable) {
    showToast(t('sessions.noMediaHint'), 'info')
    return
  }
  let url = resolveRecordingUrl(rec, userApi.getServerUrl())
  try {
    const r = await userApi.createRecordingTicket(rec.id)
    if (r.ok && r.data && r.data.url) {
      url = resolveRecordingUrl({ ...rec, artifact_url: r.data.url }, userApi.getServerUrl())
    }
  } catch { /* fall through to the direct URL */ }
  if (!url) {
    showToast(t('sessions.noMediaHint'), 'info')
    return
  }
  window.open(url, `quickdesk_recording_${rec.id}`)
}

async function revoke(row) {
  if (!confirm(t('sessions.revokeConfirm', { device: deviceNameById(row.device_id) }))) return
  busy.value = row.session_id
  notice.value = ''
  error.value = ''
  const r = await userApi.revokeRemoteSession(row.session_id)
  busy.value = ''
  if (!r.ok) {
    error.value = r.error || t('sessions.revokeFailed')
    return
  }
  notice.value = t('sessions.revoked')
  if (expanded.value === row.session_id) {
    expanded.value = ''
    detail.value = null
  }
  await load(true)
}

async function banSource(row) {
  if (!row.client_ip) return
  if (!confirm(t('sessions.banConfirm', { ip: row.client_ip }))) return
  busy.value = row.session_id
  notice.value = ''
  error.value = ''
  const r = await userApi.banAccessSource(row.device_id, row.client_ip)
  busy.value = ''
  if (!r.ok) {
    error.value = r.error || t('sessions.banFailed')
    return
  }
  notice.value = t('sessions.banned', { ip: row.client_ip })
  await loadBlockedSources()
}

async function loadBlockedSources() {
  if (!deviceFilter.value) {
    blockedSources.value = []
    return
  }
  blockedSources.value = []
  const r = await userApi.fetchAccessPolicy(deviceFilter.value)
  if (r.ok) blockedSources.value = r.data?.deny_ips ?? []
}

function formatGrantTime(value) {
  if (!value) return '-'
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? '-' : parsed.toLocaleString()
}

async function loadGrants() {
  if (!deviceFilter.value) {
    grants.value = []
    return
  }
  const r = await userApi.listAccessGrants(deviceFilter.value)
  if (!r.ok) {
    error.value = r.error || t('sessions.grantsLoadFailed')
    grants.value = []
    return
  }
  // Rows are derived rather than stored raw, so "is this usable right now" is
  // answered in one place instead of being re-decided by every template site.
  grants.value = (r.data?.items ?? []).map(grantRow)
}

async function createGrant() {
  if (!deviceFilter.value) return
  grantBusy.value = true
  error.value = ''
  notice.value = ''
  let payload
  try {
    payload = buildAccessGrantPayload({
      label: grantForm.value.label,
      expiresInMinutes: grantForm.value.expiresInMinutes,
      maxUses: grantForm.value.maxUses,
      boundClientId: grantForm.value.boundClientId,
      boundIps: grantForm.value.boundIps
        ? grantForm.value.boundIps.split(',').map((entry) => entry.trim()).filter(Boolean)
        : [],
    })
  } catch (e) {
    grantBusy.value = false
    error.value = e.message
    return
  }
  const r = await userApi.createAccessGrant(deviceFilter.value, payload)
  grantBusy.value = false
  if (!r.ok) {
    error.value = r.error || t('sessions.grantCreateFailed')
    return
  }
  // The code is shown once by the server on creation and is the one thing the
  // owner has to communicate, so it goes straight into the confirmation.
  notice.value = t('sessions.grantCreated', {
    code: r.data?.code ?? '',
    minutes: payload.expires_in_minutes,
  })
  await loadGrants()
}

async function revokeGrant(grant) {
  if (!deviceFilter.value) return
  if (!confirm(t('sessions.grantRevokeConfirm'))) return
  grantBusy.value = true
  error.value = ''
  const r = await userApi.deleteAccessGrant(deviceFilter.value, grant.id)
  grantBusy.value = false
  if (!r.ok) {
    error.value = r.error || t('sessions.grantRevokeFailed')
    return
  }
  notice.value = t('sessions.grantRevoked')
  await loadGrants()
}

async function unbanSource(source) {
  if (!deviceFilter.value) return
  if (!confirm(t('sessions.unbanConfirm', { ip: source }))) return
  banRemoving.value = source
  error.value = ''
  const r = await userApi.removeAccessBan(deviceFilter.value, source)
  banRemoving.value = ''
  if (!r.ok) {
    error.value = r.error || t('sessions.unbanFailed')
    return
  }
  notice.value = t('sessions.unbanned', { ip: source })
  blockedSources.value = r.data?.deny_ips ?? []
}

function onAccessPolicyChanged(event) {
  const detail = event.detail || {}
  if (!deviceFilter.value || detail.device_id !== deviceFilter.value) return
  if (Array.isArray(detail.deny_ips)) {
    blockedSources.value = detail.deny_ips.slice()
    return
  }
  loadBlockedSources()
}

let sessionRefreshTimer = null
let sessionRefreshPending = false

function scheduleSessionRefresh() {
  if (loading.value) {
    sessionRefreshPending = true
    return
  }
  if (sessionRefreshTimer) return
  sessionRefreshTimer = setTimeout(() => {
    sessionRefreshTimer = null
    load(true)
  }, 150)
}

function onRemoteSessionChanged(event) {
  const data = event.detail?.data || {}
  if (deviceFilter.value && data.device_id !== deviceFilter.value) return
  scheduleSessionRefresh()
}

async function clearDevice() {
  const id = deviceFilter.value
  if (!id) return
  if (!confirm(t('sessions.clearConfirm', { device: deviceNameById(id) }))) return
  clearBusy.value = true
  notice.value = ''
  error.value = ''
  const r = await userApi.revokeDeviceSessions(id)
  clearBusy.value = false
  if (!r.ok) {
    error.value = r.error || t('sessions.clearFailed')
    return
  }
  const data = r.data || {}
  notice.value = t('sessions.clearDone', {
    revoked: data.revoked ?? 0,
    failed: data.failed ?? 0,
  }) + (data.truncated ? ` ${t('sessions.clearTruncated')}` : '')
  await load(true)
}

let pollTimer = null

function startPolling() {
  pollTimer = setInterval(() => {
    if (document.hidden) return
    if (status.value !== 'active') return
    if (loading.value || busy.value || clearBusy.value) return
    load(true)
  }, 10000)
}

watch(() => authState.isLoggedIn, (loggedIn) => {
  if (!loggedIn) {
    sessions.value = []
    return
  }
  loadDevices()
  load(true)
})

onMounted(() => {
  if (authState.isLoggedIn) {
    loadDevices()
    load(true)
  }
  userSync.addEventListener('access-policy-changed', onAccessPolicyChanged)
  userSync.addEventListener('remote-session-changed', onRemoteSessionChanged)
  startPolling()
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
  if (sessionRefreshTimer) clearTimeout(sessionRefreshTimer)
  userSync.removeEventListener('access-policy-changed', onAccessPolicyChanged)
  userSync.removeEventListener('remote-session-changed', onRemoteSessionChanged)
})
</script>

<style scoped>
.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  margin-bottom: 12px;
}
.segmented {
  display: inline-flex;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 8px;
  overflow: hidden;
}
.segment {
  background: transparent;
  border: 0;
  color: var(--text-secondary);
  padding: 6px 14px;
  font-size: 13px;
  cursor: pointer;
}
.segment.active {
  background: var(--accent);
  color: #fff;
}
.select {
  background: rgba(255, 255, 255, 0.08);
  color: var(--text-primary);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 8px;
  padding: 6px 10px;
  font-size: 13px;
}
.btn-danger {
  background: rgba(244, 67, 54, 0.18);
  color: var(--error);
}
.session-item {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  padding: 12px 0;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}
.session-item:last-child {
  border-bottom: 0;
}
.session-info {
  flex: 1;
  min-width: 200px;
  cursor: pointer;
}
.session-name {
  font-size: 14px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.session-meta {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 4px;
}
.session-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.session-detail {
  flex-basis: 100%;
  background: rgba(255, 255, 255, 0.04);
  border-radius: 8px;
  padding: 10px 12px;
  font-size: 13px;
}
.detail-line {
  margin-bottom: 4px;
}
.detail-list {
  margin-top: 6px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  color: var(--text-secondary);
  font-size: 12px;
}
.detail-row {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}
.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}
.status-dot.active {
  background: var(--success);
}
.status-dot.ended {
  background: var(--text-disabled);
}
.badge {
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
}
.badge.active {
  background: rgba(76, 175, 80, 0.15);
  color: var(--success);
}
.badge.ended {
  background: rgba(255, 255, 255, 0.1);
  color: var(--text-secondary);
}
.badge.reason {
  background: rgba(233, 69, 96, 0.15);
  color: var(--accent);
}
.ban-panel {
  margin-bottom: 12px;
}
.grant-form {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}
.form-input-narrow { width: 110px; }
.grant-list { display: flex; flex-direction: column; gap: 6px; }
.grant-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  font-size: 13px;
}
.grant-code { font-weight: 600; }
.grant-meta { color: var(--el-text-color-secondary, #6b7280); }
.grant-dead { color: var(--el-color-danger, #dc2626); }
.ban-panel-title {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 8px;
}
/* QD-909: the transfer audit panel. Deliberately the same shape as the access
   log list above it - one line per event, time first - so a reader scanning a
   session's records does not have to relearn the layout for the third kind. */
.audit-panel-title {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 8px;
}
/* The export sits beside the title rather than under it: it acts on the whole
   panel, and burying it below the rows would make it look like a per-row action. */
.audit-panel-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.audit-panel-heading .audit-panel-title { margin-bottom: 0; }
.audit-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 320px;
  overflow-y: auto;
}
.audit-row {
  display: grid;
  grid-template-columns: minmax(150px, auto) minmax(0, 1fr) minmax(0, auto);
  gap: 12px;
  align-items: baseline;
  font-size: 13px;
  padding: 6px 8px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.03);
}
.audit-row-error {
  color: var(--error, #f44336);
}
.audit-error {
  color: var(--error, #f44336);
}
.ban-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.ban-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.04);
}
.ban-chip code {
  font-size: 12px;
}
.notice {
  color: var(--success);
  font-size: 13px;
  margin-bottom: 10px;
}
</style>
