<template>
  <div class="approvals-page">
    <div class="page-heading">
      <div>
        <h2 class="page-title">{{ $t('approvals.title') }}</h2>
        <p class="page-copy">{{ $t('approvals.subtitle') }}</p>
      </div>
      <button class="btn btn-secondary btn-sm" :disabled="loading" @click="refresh">
        🔄 {{ $t('approvals.refresh') }}
      </button>
    </div>

    <div v-if="!authState.isLoggedIn" class="card">
      <div class="empty-state">
        <div class="empty-icon">🔒</div>
        <p>{{ $t('approvals.loginRequired') }}</p>
        <button class="btn btn-primary btn-sm login-button" @click="showLogin()">
          {{ $t('user.login') }}
        </button>
      </div>
    </div>

    <template v-else>
      <section class="control-band">
        <label class="field device-field">
          <span>{{ $t('approvals.device') }}</span>
          <select v-model="selectedDeviceId" class="form-input" @change="refresh">
            <option value="">{{ $t('approvals.pickDevice') }}</option>
            <option v-for="device in devices" :key="device.device_id" :value="device.device_id">
              {{ deviceName(device) }}
            </option>
          </select>
        </label>
        <div class="filter-field">
          <span>{{ $t('approvals.filter') }}</span>
          <div class="segmented" role="group" :aria-label="$t('approvals.filter')">
            <button
              v-for="option in filterOptions"
              :key="option.value"
              :class="['segment', { active: statusFilter === option.value }]"
              @click="setFilter(option.value)"
            >{{ $t(option.label) }}</button>
          </div>
        </div>
        <div class="live-note">
          <span class="live-dot"></span>
          {{ $t('approvals.liveNote') }}
        </div>
      </section>

      <div v-if="!selectedDeviceId" class="card">
        <div class="empty-state">
          <div class="empty-icon">🛂</div>
          <p>{{ $t('approvals.pickDeviceHint') }}</p>
        </div>
      </div>

      <template v-else>
        <div class="summary-strip">
          <div
            v-for="status in approvalStatuses"
            :key="status"
            :class="['summary-cell', status]"
          >
            <span class="summary-value">{{ summary.byStatus[status] || 0 }}</span>
            <span class="summary-label">{{ $t(`approvals.status.${status}`) }}</span>
          </div>
        </div>

        <div class="summary-detail-row">
          <div class="window-note">
            {{ $t('approvals.summary.window', {
              since: formatTime(summary.since),
              total: summary.total,
            }) }}
          </div>
          <div class="cooldown-note">
            {{ $t('approvals.summary.cooldown', {
              seconds: summary.denialCooldownSeconds,
              multiplier: summary.denialCooldownMultiplier,
              max: summary.denialCooldownMaxSeconds || $t('approvals.summary.uncapped'),
            }) }}
          </div>
        </div>

        <section v-if="summary.deniedSources.length" class="denied-band">
          <div class="denied-head">
            <strong>{{ $t('approvals.deniedSources.title') }}</strong>
            <span>{{ $t('approvals.deniedSources.subtitle') }}</span>
          </div>
          <div class="denied-list">
            <div v-for="source in summary.deniedSources" :key="source.clientIp" class="denied-source">
              <code>{{ source.clientIp }}</code>
              <span>{{ $t('approvals.deniedSources.count', { count: source.denials }) }}</span>
              <span>{{ formatTime(source.lastDeniedAt) }}</span>
            </div>
          </div>
        </section>

        <section class="surface">
          <div class="surface-head">
            <div>
              <h3>{{ $t('approvals.queue.title') }}</h3>
              <p>{{ $t('approvals.queue.subtitle') }}</p>
            </div>
            <span class="refresh-note">{{ $t('approvals.queue.refreshNote') }}</span>
          </div>

          <div v-if="loading && approvals.length === 0" class="loading-line">
            {{ $t('approvals.loading') }}
          </div>
          <div v-else-if="approvals.length === 0" class="empty-state compact">
            <div class="empty-icon">✓</div>
            <p>{{ $t('approvals.empty') }}</p>
            <p class="hint">{{ $t('approvals.emptyHint') }}</p>
          </div>
          <div v-else class="approval-list">
            <article
              v-for="approval in approvals"
              :key="approval.id"
              :class="['approval-row', approval.status]"
            >
              <div :class="['approval-rail', approval.status]"></div>
              <div class="approval-main">
                <div class="approval-title">
                  <span :class="['state-chip', approval.status]">
                    {{ $t(`approvals.status.${approval.status}`) }}
                  </span>
                  <strong>{{ sourceLabel(approval.source) }}</strong>
                  <span class="approval-id">#{{ approval.id }}</span>
                </div>
                <div class="requester-line">
                  <code>{{ approval.client_ip || $t('approvals.unknownIp') }}</code>
                  <span>{{ approval.client_id || $t('approvals.unknownClient') }}</span>
                </div>
                <div class="approval-meta">
                  <span>
                    <b>{{ $t('approvals.fields.requested') }}</b>
                    {{ formatTime(approval.created_at) }}
                  </span>
                  <span>
                    <b>{{ $t('approvals.fields.expires') }}</b>
                    {{ formatTime(approval.expires_at) }}
                  </span>
                  <span v-if="approval.decided_at">
                    <b>{{ $t('approvals.fields.decided') }}</b>
                    {{ formatTime(approval.decided_at) }}
                  </span>
                </div>
                <div v-if="approval.reason" class="reason-line">
                  <b>{{ $t('approvals.fields.reason') }}</b> {{ approval.reason }}
                </div>
              </div>
              <div v-if="approval.status === 'pending'" class="approval-actions">
                <button
                  class="btn btn-primary btn-sm"
                  :disabled="busyId === approval.id"
                  @click="openDecision(approval, true)"
                >{{ $t('approvals.approve') }}</button>
                <button
                  class="btn btn-danger btn-sm"
                  :disabled="busyId === approval.id"
                  @click="openDecision(approval, false)"
                >{{ $t('approvals.deny') }}</button>
              </div>
            </article>
          </div>
        </section>
      </template>
    </template>

    <div v-if="decision.show" class="dialog-overlay" @click.self="closeDecision">
      <div class="dialog decision-dialog">
        <div class="dialog-head">
          <div>
            <h3>{{ decision.approve ? $t('approvals.dialog.approveTitle') : $t('approvals.dialog.denyTitle') }}</h3>
            <p>{{ $t('approvals.dialog.subtitle', {
              ip: decision.approval?.client_ip || $t('approvals.unknownIp'),
            }) }}</p>
          </div>
          <button class="icon-close" :title="$t('approvals.close')" @click="closeDecision">✕</button>
        </div>
        <div class="decision-identity">
          <div>
            <span>{{ $t('approvals.fields.clientId') }}</span>
            <code>{{ decision.approval?.client_id || $t('approvals.unknownClient') }}</code>
          </div>
          <div>
            <span>{{ $t('approvals.fields.source') }}</span>
            <code>{{ sourceLabel(decision.approval?.source) }}</code>
          </div>
        </div>
        <label class="field">
          <span>{{ $t('approvals.fields.reason') }}</span>
          <textarea
            v-model="decision.reason"
            class="form-input reason-input"
            maxlength="255"
            :placeholder="decision.approve ? $t('approvals.dialog.approveReason') : $t('approvals.dialog.denyReason')"
          ></textarea>
          <small>{{ $t('approvals.dialog.reasonHint') }}</small>
        </label>
        <p v-if="!decision.approve" class="deny-warning">
          {{ $t('approvals.dialog.denyWarning', {
            seconds: summary.denialCooldownSeconds,
          }) }}
        </p>
        <div class="form-actions">
          <button
            :class="['btn', decision.approve ? 'btn-primary' : 'btn-danger']"
            :disabled="busyId === decision.approval?.id"
            @click="submitDecision"
          >{{ busyId === decision.approval?.id ? '...' : $t('approvals.dialog.confirm') }}</button>
          <button class="btn btn-secondary" :disabled="busyId === decision.approval?.id" @click="closeDecision">
            {{ $t('user.cancel') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import {
  inject,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref,
} from 'vue'
import { useI18n } from 'vue-i18n'
import { userApi } from '../api/userApi'
import { userSync } from '../api/userSync'
import {
  APPROVAL_STATUSES,
  normalizeApprovalList,
  normalizeApprovalSummary,
} from '../api/accessApprovalModels'

const { t } = useI18n()
const authState = inject('authState')
const showToast = inject('showToast')
const showLogin = inject('showLogin')

const devices = ref([])
const selectedDeviceId = ref('')
const statusFilter = ref('pending')
const approvals = ref([])
const summary = ref(normalizeApprovalSummary(null))
const loading = ref(false)
const busyId = ref(0)
const refreshPending = ref(false)
const approvalStatuses = APPROVAL_STATUSES
let refreshTimer = null
let pollTimer = null

const filterOptions = [
  { value: 'pending', label: 'approvals.filters.pending' },
  { value: 'approved,denied', label: 'approvals.filters.decided' },
  { value: 'expired', label: 'approvals.filters.expired' },
  { value: 'consumed', label: 'approvals.filters.consumed' },
  { value: 'pending,approved,denied,expired,consumed', label: 'approvals.filters.all' },
]

const decision = reactive({
  show: false,
  approve: true,
  approval: null,
  reason: '',
})

function deviceName(device) {
  return device.remark || device.device_name || device.device_id
}

function formatTime(value) {
  if (!value) return ''
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? '' : parsed.toLocaleString()
}

function sourceLabel(source) {
  if (source === 'device_code') return t('approvals.sources.deviceCode')
  if (source === 'grant') return t('approvals.sources.grant')
  return source || t('approvals.sources.unknown')
}

async function loadDevices() {
  const result = await userApi.fetchMyDevices()
  if (!result.ok) {
    showToast(result.error || t('approvals.loadFailed'), 'error')
    return
  }
  devices.value = Array.isArray(result.data?.items) ? result.data.items : []
  if (!selectedDeviceId.value && devices.value.length) {
    selectedDeviceId.value = devices.value[0].device_id
  }
}

async function loadApprovals() {
  if (!selectedDeviceId.value) {
    approvals.value = []
    summary.value = normalizeApprovalSummary(null)
    return
  }
  if (loading.value) {
    refreshPending.value = true
    return
  }
  loading.value = true
  const [listResult, summaryResult] = await Promise.all([
    userApi.fetchAccessApprovals(selectedDeviceId.value, {
      status: statusFilter.value,
      limit: 100,
    }),
    userApi.fetchAccessApprovalSummary(selectedDeviceId.value, { limit: 10 }),
  ])
  loading.value = false
  if (refreshPending.value) {
    refreshPending.value = false
    scheduleRefresh()
  }
  if (!listResult.ok) {
    showToast(listResult.error || t('approvals.loadFailed'), 'error')
    return
  }
  if (!summaryResult.ok) {
    showToast(summaryResult.error || t('approvals.loadFailed'), 'error')
    return
  }
  approvals.value = normalizeApprovalList(listResult.data)
  summary.value = normalizeApprovalSummary(summaryResult.data)
}

async function refresh() {
  await Promise.all([loadDevices(), loadApprovals()])
}

function setFilter(value) {
  if (statusFilter.value === value) return
  statusFilter.value = value
  approvals.value = []
  loadApprovals()
}

function scheduleRefresh() {
  clearTimeout(refreshTimer)
  refreshTimer = setTimeout(loadApprovals, 150)
}

function openDecision(approval, approve) {
  decision.show = true
  decision.approve = approve
  decision.approval = approval
  decision.reason = ''
}

function closeDecision() {
  if (busyId.value) return
  decision.show = false
  decision.approval = null
}

async function submitDecision() {
  const approval = decision.approval
  if (!approval) return
  busyId.value = approval.id
  const reason = decision.reason.trim()
  const result = decision.approve
    ? await userApi.approveAccessApproval(selectedDeviceId.value, approval.id, reason)
    : await userApi.denyAccessApproval(selectedDeviceId.value, approval.id, reason)
  busyId.value = 0
  if (!result.ok) {
    if (result.status === 409) {
      showToast(t('approvals.alreadyDecided', {
        status: t(`approvals.status.${result.data?.status || 'expired'}`),
      }), 'error')
    } else {
      showToast(result.error || t('approvals.decisionFailed'), 'error')
    }
    return
  }
  decision.show = false
  decision.approval = null
  showToast(
    decision.approve ? t('approvals.approvedToast') : t('approvals.deniedToast'),
    'success',
  )
  await loadApprovals()
}

function onApprovalChanged(event) {
  if (!event.detail?.device_id || event.detail.device_id === selectedDeviceId.value) {
    if (event.detail?.type === 'device.access_approval.requested') {
      showToast(t('approvals.newRequest'), 'info')
    }
    scheduleRefresh()
  }
}

onMounted(async () => {
  userSync.addEventListener('access-approval-changed', onApprovalChanged)
  if (authState.isLoggedIn) {
    await loadDevices()
    await loadApprovals()
    pollTimer = setInterval(loadApprovals, 15_000)
  }
})

onBeforeUnmount(() => {
  clearTimeout(refreshTimer)
  clearInterval(pollTimer)
  userSync.removeEventListener('access-approval-changed', onApprovalChanged)
})
</script>

<style scoped>
.approvals-page {
  max-width: 1140px;
}

.page-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 22px;
}

.page-heading .page-title {
  margin-bottom: 6px;
}

.page-copy {
  max-width: 760px;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.login-button {
  margin-top: 12px;
}

.control-band {
  display: grid;
  grid-template-columns: minmax(220px, 320px) 1fr auto;
  gap: 18px;
  align-items: end;
  margin-bottom: 16px;
  padding: 16px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  background: rgba(15, 22, 41, 0.7);
}

.field,
.filter-field {
  display: grid;
  gap: 6px;
  color: var(--text-secondary);
  font-size: 12px;
}

.segmented {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.segment {
  min-height: 38px;
  padding: 7px 11px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.025);
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 11px;
}

.segment.active {
  border-color: rgba(233, 69, 96, 0.5);
  background: rgba(233, 69, 96, 0.1);
  color: var(--text-primary);
}

.live-note {
  display: flex;
  align-items: center;
  gap: 7px;
  padding-bottom: 10px;
  color: var(--text-secondary);
  font-size: 10px;
  white-space: nowrap;
}

.live-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--success);
  box-shadow: 0 0 0 3px rgba(76, 175, 80, 0.12);
}

.summary-strip {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 1px;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.07);
}

.summary-cell {
  display: grid;
  gap: 3px;
  padding: 14px 16px;
  background: rgba(15, 22, 41, 0.85);
}

.summary-value {
  font-size: 24px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.summary-cell.denied .summary-value {
  color: #ffaaa3;
}

.summary-cell.approved .summary-value {
  color: #a9e9ad;
}

.summary-label {
  color: var(--text-secondary);
  font-size: 10px;
}

.summary-detail-row {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  margin: 8px 0 15px;
  color: var(--text-secondary);
  font-size: 10px;
}

.denied-band {
  margin-bottom: 16px;
  padding: 14px 16px;
  border-left: 2px solid rgba(244, 67, 54, 0.55);
  background: rgba(244, 67, 54, 0.045);
}

.denied-head {
  display: flex;
  justify-content: space-between;
  gap: 14px;
}

.denied-head strong {
  font-size: 12px;
}

.denied-head span {
  color: var(--text-secondary);
  font-size: 10px;
}

.denied-list {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
  margin-top: 10px;
}

.denied-source {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 7px 9px;
  border: 1px solid rgba(244, 67, 54, 0.2);
  border-radius: 6px;
  background: rgba(0, 0, 0, 0.1);
  color: var(--text-secondary);
  font-size: 10px;
}

.denied-source code {
  color: #ffc2bd;
}

.surface {
  padding: 22px;
  border: 1px solid rgba(255, 255, 255, 0.09);
  border-radius: 8px;
  background: var(--bg-card);
}

.surface-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}

.surface-head h3 {
  margin-bottom: 4px;
  font-size: 17px;
}

.surface-head p,
.refresh-note {
  color: var(--text-secondary);
  font-size: 11px;
  line-height: 1.55;
}

.approval-list {
  display: grid;
  gap: 1px;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.08);
}

.approval-row {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 18px;
  padding: 17px 16px 17px 20px;
  background: rgba(15, 22, 41, 0.74);
}

.approval-rail {
  position: absolute;
  inset: 0 auto 0 0;
  width: 3px;
  background: rgba(255, 255, 255, 0.13);
}

.approval-rail.pending {
  background: var(--warning);
}

.approval-rail.approved {
  background: var(--success);
}

.approval-rail.denied {
  background: var(--error);
}

.approval-title {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.approval-title strong {
  font-size: 14px;
}

.approval-id {
  color: var(--text-secondary);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 10px;
}

.requester-line {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 14px;
  margin-top: 8px;
}

.requester-line code {
  color: #bcd4ff;
  font-size: 13px;
}

.requester-line span {
  color: var(--text-secondary);
  font-size: 11px;
}

.approval-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 5px 18px;
  margin-top: 9px;
  color: var(--text-secondary);
  font-size: 10px;
}

.approval-meta b,
.reason-line b {
  color: #b8c3da;
  font-weight: 500;
}

.reason-line {
  margin-top: 9px;
  color: var(--text-secondary);
  font-size: 11px;
}

.approval-actions {
  display: flex;
  align-content: flex-start;
  gap: 7px;
}

.state-chip {
  display: inline-flex;
  min-height: 22px;
  align-items: center;
  padding: 2px 8px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 999px;
  color: var(--text-secondary);
  font-size: 10px;
  white-space: nowrap;
}

.state-chip.pending {
  border-color: rgba(255, 193, 7, 0.35);
  background: rgba(255, 193, 7, 0.09);
  color: #ffdc7a;
}

.state-chip.approved,
.state-chip.consumed {
  border-color: rgba(76, 175, 80, 0.35);
  background: rgba(76, 175, 80, 0.1);
  color: #a9e9ad;
}

.state-chip.denied {
  border-color: rgba(244, 67, 54, 0.36);
  background: rgba(244, 67, 54, 0.1);
  color: #ffaaa3;
}

.state-chip.expired {
  background: rgba(255, 255, 255, 0.04);
}

.btn-danger {
  border: 1px solid rgba(244, 67, 54, 0.3);
  background: rgba(244, 67, 54, 0.1);
  color: #ffaaa3;
}

.loading-line {
  padding: 24px 0;
  color: var(--text-secondary);
  font-size: 13px;
}

.empty-state.compact {
  padding: 28px 14px;
}

.dialog-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: grid;
  place-items: center;
  padding: 24px;
  background: rgba(5, 9, 20, 0.78);
}

.decision-dialog {
  width: min(560px, 100%);
  padding: 22px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 8px;
  background: var(--bg-card);
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.42);
}

.dialog-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 18px;
}

.dialog-head h3 {
  font-size: 18px;
}

.dialog-head p {
  margin-top: 4px;
  color: var(--text-secondary);
  font-size: 11px;
}

.icon-close {
  width: 32px;
  height: 32px;
  border: 0;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-primary);
  cursor: pointer;
}

.decision-identity {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-bottom: 16px;
}

.decision-identity div {
  display: grid;
  gap: 4px;
  padding: 10px;
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 7px;
  background: rgba(0, 0, 0, 0.1);
}

.decision-identity span {
  color: var(--text-secondary);
  font-size: 10px;
}

.decision-identity code {
  overflow-wrap: anywhere;
  color: #bcd4ff;
  font-size: 11px;
}

.reason-input {
  min-height: 110px;
  resize: vertical;
}

.field small {
  color: var(--text-secondary);
  font-size: 10px;
}

.deny-warning {
  margin-top: 12px;
  padding: 9px 11px;
  border-left: 2px solid rgba(244, 67, 54, 0.55);
  background: rgba(244, 67, 54, 0.045);
  color: #ffc2bd;
  font-size: 10px;
  line-height: 1.5;
}

.form-actions {
  display: flex;
  gap: 8px;
  margin-top: 18px;
}

@media (max-width: 880px) {
  .control-band {
    grid-template-columns: 1fr;
    align-items: stretch;
  }

  .live-note {
    padding: 0;
  }

  .summary-strip {
    grid-template-columns: repeat(3, 1fr);
  }
}

@media (max-width: 650px) {
  .page-heading,
  .surface-head,
  .summary-detail-row,
  .denied-head {
    flex-direction: column;
  }

  .approval-row {
    grid-template-columns: 1fr;
  }

  .summary-strip {
    grid-template-columns: 1fr 1fr;
  }

  .surface,
  .decision-dialog {
    padding: 16px;
  }

  .decision-identity {
    grid-template-columns: 1fr;
  }
}
</style>
