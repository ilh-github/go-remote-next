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
        <button
          v-if="deviceFilter"
          class="btn btn-danger btn-sm"
          :disabled="clearBusy"
          @click="clearDevice"
        >{{ clearBusy ? '...' : $t('sessions.clearDevice') }}</button>
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
                  <b>{{ $t('sessions.recordings') }}:</b> {{ detail.counts?.recordings ?? 0 }}
                </div>
                <div class="detail-line">
                  <b>{{ $t('sessions.accessLog') }}:</b> {{ detail.counts?.access_log ?? 0 }}
                </div>
                <div v-if="(detail.access_log ?? []).length" class="detail-list">
                  <div v-for="log in detail.access_log" :key="log.id" class="detail-row">
                    <span>{{ formatTime(log.created_at) }}</span>
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
const busy = ref('')
const clearBusy = ref(false)
const blockedSources = ref([])
const banRemoving = ref('')

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
.ban-panel-title {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 8px;
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
