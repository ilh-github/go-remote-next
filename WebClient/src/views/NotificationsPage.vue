<template>
  <div class="notifications-page">
    <div class="page-heading">
      <div>
        <h2 class="page-title">{{ $t('notifications.title') }}</h2>
        <p class="page-copy">{{ $t('notifications.subtitle') }}</p>
      </div>
      <button class="btn btn-secondary btn-sm" :disabled="loading" @click="refreshAll">
        🔄 {{ $t('notifications.refresh') }}
      </button>
    </div>

    <div v-if="!authState.isLoggedIn" class="card">
      <div class="empty-state">
        <div class="empty-icon">🔒</div>
        <p>{{ $t('notifications.loginRequired') }}</p>
        <button class="btn btn-primary btn-sm login-button" @click="showLogin()">
          {{ $t('user.login') }}
        </button>
      </div>
    </div>

    <template v-else>
      <section class="surface">
        <div class="surface-head">
          <div>
            <h3>{{ $t('notifications.channels.title') }}</h3>
            <p>{{ $t('notifications.channels.subtitle') }}</p>
          </div>
          <div class="head-actions">
            <button class="btn btn-secondary btn-sm" :disabled="retryingAll" @click="retryAllFailed">
              {{ retryingAll ? '...' : $t('notifications.channels.retryAll') }}
            </button>
            <button class="btn btn-primary btn-sm" @click="openCreateChannel">
              {{ $t('notifications.channels.create') }}
            </button>
          </div>
        </div>

        <div v-if="channelsLoading && channels.length === 0" class="loading-line">
          {{ $t('notifications.loading') }}
        </div>
        <div v-else-if="channels.length === 0" class="empty-state compact">
          <div class="empty-icon">🔔</div>
          <p>{{ $t('notifications.channels.empty') }}</p>
          <p class="hint">{{ $t('notifications.channels.emptyHint') }}</p>
        </div>
        <div v-else class="channel-list">
          <article v-for="channel in channels" :key="channel.id" class="channel-row">
            <div class="channel-main">
              <div class="channel-title-line">
                <strong>{{ channel.name || $t('notifications.channels.unnamed') }}</strong>
                <span :class="['state-chip', channel.enabled ? 'enabled' : 'paused']">
                  {{ channel.enabled ? $t('notifications.channels.enabled') : $t('notifications.channels.paused') }}
                </span>
                <span v-if="channel.last_status" :class="['state-chip', statusTone(channel)]">
                  HTTP {{ channel.last_status }}
                </span>
              </div>
              <div class="channel-url">{{ channel.url }}</div>
              <div class="event-tags">
                <span v-for="event in channel.events" :key="event" class="event-tag" :title="event">
                  {{ eventLabel(event) }}
                </span>
              </div>
              <div class="channel-meta">
                <span>
                  {{ $t('notifications.channels.lastTriggered') }}:
                  {{ channel.last_triggered ? formatTime(channel.last_triggered) : $t('notifications.never') }}
                </span>
                <span>{{ $t('notifications.channels.secretHint') }}</span>
              </div>
            </div>
            <div class="row-actions">
              <button class="btn btn-secondary btn-sm" @click="toggleChannel(channel)">
                {{ channel.enabled ? $t('notifications.channels.pause') : $t('notifications.channels.enable') }}
              </button>
              <button class="btn btn-secondary btn-sm" @click="testChannel(channel)">
                {{ $t('notifications.channels.test') }}
              </button>
              <button class="btn btn-secondary btn-sm" @click="openDeliveries(channel)">
                {{ $t('notifications.channels.deliveries') }}
              </button>
              <button class="btn btn-secondary btn-sm" @click="openEditChannel(channel)">
                {{ $t('notifications.edit') }}
              </button>
              <button class="btn btn-danger btn-sm" @click="deleteChannel(channel)">
                {{ $t('notifications.delete') }}
              </button>
            </div>
          </article>
        </div>
        <div class="signature-line">
          {{ $t('notifications.channels.signature', {
            header: signatureHeader || 'X-Webhook-Signature',
            scheme: signatureScheme || 'sha256=<hmac-sha256 of the raw body>',
          }) }}
        </div>
      </section>

      <section class="surface">
        <div class="surface-head">
          <div>
            <h3>{{ $t('notifications.quiet.title') }}</h3>
            <p>{{ $t('notifications.quiet.subtitle') }}</p>
          </div>
          <span :class="['state-chip', quietMeta.active ? 'warning' : 'enabled']">
            {{ quietMeta.active ? $t('notifications.quiet.active') : $t('notifications.quiet.inactive') }}
          </span>
        </div>
        <div class="field-grid two">
          <label class="field">
            <span>{{ $t('notifications.quiet.snoozeUntil') }}</span>
            <input v-model="quietForm.snoozed_until" class="form-input" type="datetime-local" />
            <small>{{ $t('notifications.quiet.snoozeHint') }}</small>
          </label>
          <label class="field">
            <span>{{ $t('notifications.timezone') }}</span>
            <input
              v-model="quietForm.timezone"
              class="form-input"
              type="text"
              placeholder="Asia/Shanghai"
            />
            <small>{{ $t('notifications.timezoneHint') }}</small>
          </label>
        </div>
        <div class="subsection">
          <div class="subsection-title">{{ $t('notifications.quiet.windows') }}</div>
          <WindowEditor
            v-model="quietForm.quiet_hours"
            :empty-label="$t('notifications.windows.none')"
            :hint="$t('notifications.quiet.windowHint')"
          />
        </div>
        <div class="form-actions">
          <button class="btn btn-primary" :disabled="quietSaving" @click="saveQuietHours">
            {{ quietSaving ? '...' : $t('notifications.save') }}
          </button>
          <button class="btn btn-secondary" :disabled="quietSaving" @click="clearQuietHours">
            {{ $t('notifications.quiet.clear') }}
          </button>
          <span v-if="quietMeta.active_until" class="hint">
            {{ $t('notifications.quiet.until', { time: formatTime(quietMeta.active_until) }) }}
          </span>
        </div>
      </section>

      <section class="surface">
        <div class="surface-head">
          <div>
            <h3>{{ $t('notifications.offline.title') }}</h3>
            <p>{{ $t('notifications.offline.subtitle') }}</p>
          </div>
          <span
            v-if="offlineMeta.in_maintenance_window"
            class="state-chip warning"
          >{{ $t('notifications.offline.inMaintenance') }}</span>
        </div>
        <label class="field">
          <span>{{ $t('notifications.offline.device') }}</span>
          <select v-model="selectedDeviceId" class="form-input" @change="loadOfflineAlert">
            <option value="">{{ $t('notifications.offline.pickDevice') }}</option>
            <option v-for="device in devices" :key="device.device_id" :value="device.device_id">
              {{ deviceName(device) }}
            </option>
          </select>
        </label>

        <div v-if="selectedDeviceId" class="offline-form">
          <div class="field-grid three">
            <label class="field toggle-field">
              <span>{{ $t('notifications.offline.enabled') }}</span>
              <label class="switch-row">
                <input v-model="offlineForm.enabled" type="checkbox" />
                <span>{{ offlineForm.enabled ? $t('notifications.on') : $t('notifications.off') }}</span>
              </label>
            </label>
            <label class="field">
              <span>{{ $t('notifications.offline.afterMinutes') }}</span>
              <input v-model.number="offlineForm.after_minutes" class="form-input" type="number" min="1" max="10080" />
              <small>{{ $t('notifications.offline.afterHint') }}</small>
            </label>
            <label class="field">
              <span>{{ $t('notifications.offline.repeatMinutes') }}</span>
              <input v-model.number="offlineForm.repeat_minutes" class="form-input" type="number" min="0" max="10080" />
              <small>{{ $t('notifications.offline.repeatHint') }}</small>
            </label>
          </div>
          <div class="field-grid two">
            <label class="field">
              <span>{{ $t('notifications.offline.snoozeUntil') }}</span>
              <input v-model="offlineForm.snoozed_until" class="form-input" type="datetime-local" />
            </label>
            <label class="field">
              <span>{{ $t('notifications.timezone') }}</span>
              <input
                v-model="offlineForm.maintenance_timezone"
                class="form-input"
                type="text"
                placeholder="Asia/Shanghai"
              />
            </label>
          </div>
          <div class="subsection">
            <div class="subsection-title">{{ $t('notifications.offline.maintenanceWindows') }}</div>
            <WindowEditor
              v-model="offlineForm.maintenance_windows"
              :empty-label="$t('notifications.windows.none')"
              :hint="$t('notifications.offline.windowHint')"
              default-start="02:00"
              default-end="03:00"
            />
          </div>
          <div v-if="offlineMeta.offline_since" class="meta-strip">
            {{ $t('notifications.offline.since', { time: formatTime(offlineMeta.offline_since) }) }}
            <span v-if="offlineMeta.last_alerted_at">
              · {{ $t('notifications.offline.lastAlerted', { time: formatTime(offlineMeta.last_alerted_at) }) }}
            </span>
          </div>
          <div class="form-actions">
            <button class="btn btn-primary" :disabled="offlineSaving" @click="saveOfflineAlert">
              {{ offlineSaving ? '...' : $t('notifications.save') }}
            </button>
            <button class="btn btn-secondary" :disabled="offlineSaving" @click="removeOfflineAlert">
              {{ $t('notifications.offline.reset') }}
            </button>
          </div>
        </div>
      </section>

      <section class="surface">
        <div class="surface-head">
          <div>
            <h3>{{ $t('notifications.metricAlert.title') }}</h3>
            <p>{{ $t('notifications.metricAlert.subtitle') }}</p>
          </div>
          <span :class="['state-chip', metricForm.enabled ? 'enabled' : 'paused']">
            {{ metricForm.enabled ? $t('notifications.metricAlert.enabled') : $t('notifications.metricAlert.disabled') }}
          </span>
        </div>
        <label class="field">
          <span>{{ $t('notifications.metricAlert.device') }}</span>
          <select v-model="metricDeviceId" class="form-input" @change="loadMetricAlert">
            <option value="">{{ $t('notifications.metricAlert.pickDevice') }}</option>
            <option v-for="device in devices" :key="device.device_id" :value="device.device_id">
              {{ deviceName(device) }}
            </option>
          </select>
        </label>

        <div v-if="metricDeviceId" class="metric-alert-form">
          <div class="metric-current-grid">
            <article class="metric-reading">
              <span>{{ $t('notifications.metricAlert.cpu') }}</span>
              <strong>{{ metricPercent(metricMeta.current.cpu_percent) }}</strong>
              <small>{{ $t('notifications.metricAlert.threshold', { value: metricForm.cpu_percent }) }}</small>
            </article>
            <article class="metric-reading">
              <span>{{ $t('notifications.metricAlert.memory') }}</span>
              <strong>{{ metricPercent(metricMeta.current.memory_percent) }}</strong>
              <small>{{ $t('notifications.metricAlert.threshold', { value: metricForm.memory_percent }) }}</small>
            </article>
            <article class="metric-reading">
              <span>{{ $t('notifications.metricAlert.disk') }}</span>
              <strong>{{ metricPercent(metricMeta.current.disk_percent) }}</strong>
              <small>{{ $t('notifications.metricAlert.threshold', { value: metricForm.disk_percent }) }}</small>
            </article>
            <article class="metric-reading">
              <span>{{ $t('notifications.metricAlert.mount') }}</span>
              <strong class="metric-mount">{{ metricMeta.current.disk_mount || $t('notifications.metricAlert.unknownMount') }}</strong>
              <small>
                {{ metricMeta.current.captured_at
                  ? $t('notifications.metricAlert.capturedAt', { time: formatTime(metricMeta.current.captured_at) })
                  : $t('notifications.metricAlert.noReading') }}
              </small>
            </article>
          </div>

          <div class="metric-state-list">
            <div v-for="metric in ['cpu', 'memory', 'disk']" :key="metric" class="metric-state-row">
              <span>{{ $t(`notifications.metricAlert.${metric}`) }}</span>
              <span :class="['state-chip', metricMeta.states[metric].exceeded_since ? 'warning' : 'enabled']">
                {{ metricMeta.states[metric].exceeded_since
                  ? $t('notifications.metricAlert.sustained')
                  : $t('notifications.metricAlert.normal') }}
              </span>
              <small v-if="metricMeta.states[metric].exceeded_since">
                {{ $t('notifications.metricAlert.since', {
                  time: formatTime(metricMeta.states[metric].exceeded_since),
                }) }}
              </small>
              <small v-if="metricMeta.states[metric].last_alerted_at">
                {{ $t('notifications.metricAlert.lastAlerted', {
                  time: formatTime(metricMeta.states[metric].last_alerted_at),
                }) }}
              </small>
            </div>
          </div>

          <div class="field-grid three">
            <label class="field toggle-field">
              <span>{{ $t('notifications.metricAlert.ruleEnabled') }}</span>
              <label class="switch-row">
                <input v-model="metricForm.enabled" type="checkbox" />
                <span>{{ metricForm.enabled ? $t('notifications.on') : $t('notifications.off') }}</span>
              </label>
            </label>
            <label class="field">
              <span>{{ $t('notifications.metricAlert.cpuThreshold') }}</span>
              <input v-model.number="metricForm.cpu_percent" class="form-input" type="number" min="0" max="100" />
              <small>{{ $t('notifications.metricAlert.zeroDisables') }}</small>
            </label>
            <label class="field">
              <span>{{ $t('notifications.metricAlert.memoryThreshold') }}</span>
              <input v-model.number="metricForm.memory_percent" class="form-input" type="number" min="0" max="100" />
              <small>{{ $t('notifications.metricAlert.zeroDisables') }}</small>
            </label>
          </div>
          <div class="field-grid three">
            <label class="field">
              <span>{{ $t('notifications.metricAlert.diskThreshold') }}</span>
              <input v-model.number="metricForm.disk_percent" class="form-input" type="number" min="0" max="100" />
              <small>{{ $t('notifications.metricAlert.zeroDisables') }}</small>
            </label>
            <label class="field">
              <span>{{ $t('notifications.metricAlert.sustainedMinutes') }}</span>
              <input v-model.number="metricForm.sustained_minutes" class="form-input" type="number" min="0" max="10080" />
              <small>{{ $t('notifications.metricAlert.sustainedHint') }}</small>
            </label>
            <label class="field">
              <span>{{ $t('notifications.metricAlert.repeatMinutes') }}</span>
              <input v-model.number="metricForm.repeat_minutes" class="form-input" type="number" min="0" max="10080" />
              <small>{{ $t('notifications.metricAlert.repeatHint') }}</small>
            </label>
          </div>
          <label class="field metric-snooze">
            <span>{{ $t('notifications.metricAlert.snoozeUntil') }}</span>
            <input v-model="metricForm.snoozed_until" class="form-input" type="datetime-local" />
          </label>
          <p v-if="metricError" class="inline-error">{{ metricError }}</p>
          <div class="form-actions">
            <button class="btn btn-primary" :disabled="metricSaving" @click="saveMetricAlert">
              {{ metricSaving ? '...' : $t('notifications.save') }}
            </button>
            <button class="btn btn-secondary" :disabled="metricSaving" @click="removeMetricAlert">
              {{ $t('notifications.metricAlert.reset') }}
            </button>
          </div>
        </div>
      </section>

      <section v-if="selectedChannel" class="surface">
        <div class="surface-head">
          <div>
            <h3>{{ $t('notifications.deliveries.title') }}</h3>
            <p>{{ $t('notifications.deliveries.subtitle', { name: selectedChannel.name || selectedChannel.url }) }}</p>
          </div>
          <div class="head-actions">
            <span class="unacked-count">
              {{ $t('notifications.deliveries.unacked', { count: unackedDeliveries }) }}
            </span>
            <button class="btn btn-secondary btn-sm" :disabled="deliveriesLoading" @click="loadDeliveries">
              {{ $t('notifications.deliveries.reload') }}
            </button>
            <button class="btn btn-secondary btn-sm" :disabled="deliveryBulkBusy" @click="retryChannelFailed">
              {{ $t('notifications.deliveries.retryFailed') }}
            </button>
            <button class="btn btn-secondary btn-sm" @click="closeDeliveries">
              {{ $t('notifications.close') }}
            </button>
          </div>
        </div>
        <div v-if="deliveriesLoading && deliveries.length === 0" class="loading-line">
          {{ $t('notifications.loading') }}
        </div>
        <div v-else-if="deliveries.length === 0" class="empty-state compact">
          <p>{{ $t('notifications.deliveries.empty') }}</p>
        </div>
        <div v-else class="delivery-table-wrap">
          <table class="delivery-table">
            <thead>
              <tr>
                <th>{{ $t('notifications.deliveries.event') }}</th>
                <th>{{ $t('notifications.deliveries.status') }}</th>
                <th>{{ $t('notifications.deliveries.attempts') }}</th>
                <th>{{ $t('notifications.deliveries.queued') }}</th>
                <th>{{ $t('notifications.deliveries.result') }}</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="delivery in deliveries" :key="delivery.id">
                <td>
                  <div>{{ eventLabel(delivery.event_type) }}</div>
                  <small>{{ delivery.device_id || $t('notifications.deliveries.accountEvent') }}</small>
                </td>
                <td>
                  <span :class="['state-chip', deliveryTone(delivery)]">{{ $t(`notifications.deliveryStatus.${delivery.status}`) }}</span>
                  <div v-if="delivery.held_until" class="table-note">
                    {{ $t('notifications.deliveries.heldUntil', { time: formatTime(delivery.held_until) }) }}
                  </div>
                </td>
                <td>{{ delivery.attempts }}</td>
                <td>{{ formatTime(delivery.created_at) }}</td>
                <td>
                  <div v-if="delivery.last_status">HTTP {{ delivery.last_status }}</div>
                  <div v-if="delivery.last_error" class="table-error">{{ delivery.last_error }}</div>
                  <div v-if="delivery.acked_at" class="table-note">
                    {{ $t('notifications.deliveries.acked', { time: formatTime(delivery.acked_at) }) }}
                  </div>
                </td>
                <td class="delivery-actions">
                  <button
                    v-if="deliveryCanRetry(delivery)"
                    class="text-action"
                    :disabled="deliveryBusy === delivery.id"
                    @click="retryDelivery(delivery)"
                  >{{ $t('notifications.deliveries.retry') }}</button>
                  <button
                    v-if="deliveryCanAck(delivery)"
                    class="text-action"
                    :disabled="deliveryBusy === delivery.id"
                    @click="ackDelivery(delivery)"
                  >{{ $t('notifications.deliveries.ack') }}</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </template>

    <div v-if="channelDialog.show" class="dialog-overlay" @click.self="closeChannelDialog">
      <div class="dialog channel-dialog">
        <div class="dialog-head">
          <h3>{{ channelDialog.id ? $t('notifications.channels.editTitle') : $t('notifications.channels.createTitle') }}</h3>
          <button class="icon-close" :title="$t('notifications.close')" @click="closeChannelDialog">✕</button>
        </div>
        <div class="field-grid two">
          <label class="field">
            <span>{{ $t('notifications.channels.name') }}</span>
            <input v-model="channelDialog.name" class="form-input" type="text" :placeholder="$t('notifications.channels.namePlaceholder')" />
          </label>
          <label class="field toggle-field">
            <span>{{ $t('notifications.channels.state') }}</span>
            <label class="switch-row">
              <input v-model="channelDialog.enabled" type="checkbox" />
              <span>{{ channelDialog.enabled ? $t('notifications.channels.enabled') : $t('notifications.channels.paused') }}</span>
            </label>
          </label>
        </div>
        <label class="field">
          <span>{{ $t('notifications.channels.url') }}</span>
          <input v-model="channelDialog.url" class="form-input" type="url" placeholder="https://example.com/quickdesk/webhook" />
        </label>
        <label class="field">
          <span>{{ $t('notifications.channels.secret') }}</span>
          <input
            v-model="channelDialog.secret"
            class="form-input"
            type="password"
            autocomplete="new-password"
            :placeholder="channelDialog.id ? $t('notifications.channels.secretKeep') : $t('notifications.channels.secretOptional')"
          />
          <small>{{ $t('notifications.channels.secretHint') }}</small>
        </label>
        <div class="subsection">
          <div class="subsection-title">{{ $t('notifications.channels.events') }}</div>
          <div class="event-picker">
            <label v-for="event in supportedEvents" :key="event" class="event-choice">
              <input
                type="checkbox"
                :checked="channelDialog.events.includes(event)"
                @change="toggleChannelEvent(event, $event.target.checked)"
              />
              <span>{{ eventLabel(event) }}</span>
              <code>{{ event }}</code>
            </label>
          </div>
        </div>
        <div class="form-actions">
          <button class="btn btn-primary" :disabled="channelDialog.saving" @click="saveChannel">
            {{ channelDialog.saving ? '...' : $t('notifications.save') }}
          </button>
          <button class="btn btn-secondary" :disabled="channelDialog.saving" @click="closeChannelDialog">
            {{ $t('user.cancel') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import WindowEditor from '../components/WindowEditor.vue'
import { userApi } from '../api/userApi'
import {
  buildChannelPayload,
  buildMetricAlertPayload,
  buildOfflineAlertPayload,
  buildQuietHoursPayload,
  normalizeChannelList,
  normalizeMetricAlert,
  normalizeOfflineAlert,
  normalizeQuietHours,
  toDatetimeLocalValue,
} from '../api/notificationModels'

const { t } = useI18n()
const authState = inject('authState')
const showToast = inject('showToast')
const showLogin = inject('showLogin')

const channels = ref([])
const supportedEvents = ref([])
const signatureHeader = ref('')
const signatureScheme = ref('')
const channelsLoading = ref(false)
const loading = ref(false)
const retryingAll = ref(false)
const devices = ref([])
const selectedDeviceId = ref('')
const offlineSaving = ref(false)
const metricDeviceId = ref('')
const metricSaving = ref(false)
const metricError = ref('')
const selectedChannelId = ref(null)
const deliveries = ref([])
const unackedDeliveries = ref(0)
const deliveriesLoading = ref(false)
const deliveryBusy = ref(0)
const deliveryBulkBusy = ref(false)
const quietSaving = ref(false)

const quietForm = reactive({
  snoozed_until: '',
  quiet_hours: [],
  timezone: '',
})
const quietMeta = reactive({
  active: false,
  active_until: null,
})

const offlineForm = reactive({
  enabled: false,
  after_minutes: 10,
  repeat_minutes: 0,
  snoozed_until: '',
  maintenance_windows: [],
  maintenance_timezone: '',
})
const offlineMeta = reactive({
  in_maintenance_window: false,
  offline_since: null,
  last_alerted_at: null,
  updated_at: null,
})

const metricForm = reactive({
  enabled: false,
  cpu_percent: 0,
  memory_percent: 0,
  disk_percent: 0,
  sustained_minutes: 5,
  repeat_minutes: 0,
  snoozed_until: '',
})
const metricMeta = reactive({
  current: {
    captured_at: null,
    cpu_percent: null,
    memory_percent: null,
    disk_percent: null,
    disk_mount: '',
  },
  states: {
    cpu: { exceeded_since: null, last_alerted_at: null },
    memory: { exceeded_since: null, last_alerted_at: null },
    disk: { exceeded_since: null, last_alerted_at: null },
  },
})

const channelDialog = reactive({
  show: false,
  id: null,
  name: '',
  url: '',
  secret: '',
  events: [],
  enabled: true,
  saving: false,
})

const selectedChannel = computed(() => (
  channels.value.find((channel) => channel.id === selectedChannelId.value) || null
))

const eventLabelKeys = {
  'device.access.denied': 'accessDenied',
  'device.access.granted': 'accessGranted',
  'device.access.session': 'accessSession',
  'device.online.changed': 'onlineChanged',
  'device.offline.sustained': 'offlineSustained',
  'device.bound': 'bound',
  'device.unbound': 'unbound',
  'device.access_approval.requested': 'approvalRequested',
  'device.access_approval.decided': 'approvalDecided',
  'scheduled_task.failed': 'scheduledTaskFailed',
  'device.metric.alert': 'metricAlert',
}

function eventLabel(event) {
  const key = eventLabelKeys[event]
  if (!key) return event
  return t(`notifications.eventNames.${key}`)
}

function deviceName(device) {
  return device.remark || device.device_name || device.device_id
}

function formatTime(value) {
  if (!value) return ''
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? '' : parsed.toLocaleString()
}

function metricPercent(value) {
  if (value === null || value === undefined || value === '') {
    return t('notifications.metricAlert.noReading')
  }
  return Number.isFinite(Number(value)) ? `${Number(value).toFixed(1)}%` : t('notifications.metricAlert.noReading')
}

function statusTone(channel) {
  if (!channel.enabled) return 'paused'
  if (channel.last_status >= 200 && channel.last_status < 300) return 'enabled'
  if (channel.last_status) return 'danger'
  return 'neutral'
}

function deliveryTone(delivery) {
  if (delivery.status === 'delivered') return delivery.acked_at ? 'enabled' : 'warning'
  if (delivery.status === 'failed') return 'danger'
  if (delivery.status === 'delivering') return 'warning'
  return 'neutral'
}

function deliveryCanRetry(delivery) {
  return delivery.status === 'failed'
}

function deliveryCanAck(delivery) {
  return !delivery.acked_at && ['delivered', 'failed'].includes(delivery.status)
}

function toastError(result, fallbackKey = 'toast.networkError') {
  showToast(result?.error || t(fallbackKey), 'error')
}

async function refreshAll() {
  if (loading.value) return
  loading.value = true
  try {
    await Promise.all([loadChannels(), loadQuietHours(), loadDevices()])
  } finally {
    loading.value = false
  }
}

async function loadChannels() {
  channelsLoading.value = true
  const result = await userApi.fetchNotificationChannels()
  channelsLoading.value = false
  if (!result.ok) {
    toastError(result, 'notifications.channels.loadFailed')
    return
  }
  const view = normalizeChannelList(result.data)
  channels.value = view.items
  supportedEvents.value = view.supportedEvents
  signatureHeader.value = view.signatureHeader
  signatureScheme.value = view.signatureScheme
  if (selectedChannelId.value && !view.items.some((row) => row.id === selectedChannelId.value)) {
    closeDeliveries()
  }
}

async function loadDevices() {
  const result = await userApi.fetchMyDevices()
  if (!result.ok) return
  devices.value = Array.isArray(result.data?.items) ? result.data.items : []
  if (selectedDeviceId.value && !devices.value.some((row) => row.device_id === selectedDeviceId.value)) {
    selectedDeviceId.value = ''
  }
  if (metricDeviceId.value && !devices.value.some((row) => row.device_id === metricDeviceId.value)) {
    metricDeviceId.value = ''
    applyMetricAlert({})
  }
}

async function loadQuietHours() {
  const result = await userApi.fetchNotificationQuietHours()
  if (!result.ok) {
    toastError(result, 'notifications.quiet.loadFailed')
    return
  }
  applyQuietHours(result.data)
}

function applyQuietHours(data) {
  const view = normalizeQuietHours(data)
  quietForm.snoozed_until = toDatetimeLocalValue(view.snoozed_until)
  quietForm.quiet_hours = view.quiet_hours
  quietForm.timezone = view.timezone
  quietMeta.active = view.active
  quietMeta.active_until = view.active_until
}

async function saveQuietHours() {
  quietSaving.value = true
  const result = await userApi.updateNotificationQuietHours(buildQuietHoursPayload({
    ...quietForm,
  }))
  quietSaving.value = false
  if (!result.ok) {
    toastError(result, 'notifications.quiet.saveFailed')
    return
  }
  applyQuietHours(result.data)
  showToast(t('notifications.quiet.saved'), 'success')
}

async function clearQuietHours() {
  quietSaving.value = true
  const result = await userApi.updateNotificationQuietHours({
    snoozed_until: null,
    quiet_hours: [],
    timezone: '',
  })
  quietSaving.value = false
  if (!result.ok) {
    toastError(result, 'notifications.quiet.saveFailed')
    return
  }
  applyQuietHours(result.data)
  showToast(t('notifications.quiet.cleared'), 'success')
}

function openCreateChannel() {
  channelDialog.show = true
  channelDialog.id = null
  channelDialog.name = ''
  channelDialog.url = ''
  channelDialog.secret = ''
  channelDialog.events = [...supportedEvents.value]
  channelDialog.enabled = true
  channelDialog.saving = false
}

function openEditChannel(channel) {
  channelDialog.show = true
  channelDialog.id = channel.id
  channelDialog.name = channel.name || ''
  channelDialog.url = channel.url || ''
  channelDialog.secret = ''
  channelDialog.events = [...channel.events]
  channelDialog.enabled = !!channel.enabled
  channelDialog.saving = false
}

function closeChannelDialog() {
  if (channelDialog.saving) return
  channelDialog.show = false
}

function toggleChannelEvent(event, checked) {
  const next = new Set(channelDialog.events)
  if (checked) next.add(event)
  else next.delete(event)
  channelDialog.events = [...next]
}

async function saveChannel() {
  const payload = buildChannelPayload(channelDialog, { partial: !!channelDialog.id })
  if (!payload.url || payload.events.length === 0) {
    showToast(t('notifications.channels.required'), 'error')
    return
  }
  channelDialog.saving = true
  const result = channelDialog.id
    ? await userApi.updateNotificationChannel(channelDialog.id, payload)
    : await userApi.createNotificationChannel(payload)
  channelDialog.saving = false
  if (!result.ok) {
    toastError(result, 'notifications.channels.saveFailed')
    return
  }
  channelDialog.show = false
  await loadChannels()
  showToast(t('notifications.channels.saved'), 'success')
}

async function toggleChannel(channel) {
  const result = await userApi.updateNotificationChannel(channel.id, buildChannelPayload({
    ...channel,
    enabled: !channel.enabled,
  }, { partial: true }))
  if (!result.ok) {
    toastError(result, 'notifications.channels.saveFailed')
    return
  }
  await loadChannels()
}

async function testChannel(channel) {
  const result = await userApi.testNotificationChannel(channel.id)
  if (!result.ok) {
    toastError(result, 'notifications.channels.testFailed')
    return
  }
  showToast(t('notifications.channels.testQueued'), 'success')
}

async function deleteChannel(channel) {
  if (!confirm(t('notifications.channels.deleteConfirm', { name: channel.name || channel.url }))) return
  const result = await userApi.deleteNotificationChannel(channel.id)
  if (!result.ok) {
    toastError(result, 'notifications.channels.deleteFailed')
    return
  }
  if (selectedChannelId.value === channel.id) closeDeliveries()
  await loadChannels()
  showToast(t('notifications.channels.deleted'), 'success')
}

async function openDeliveries(channel) {
  selectedChannelId.value = channel.id
  deliveries.value = []
  await loadDeliveries()
}

function closeDeliveries() {
  selectedChannelId.value = null
  deliveries.value = []
  unackedDeliveries.value = 0
}

async function loadDeliveries() {
  if (!selectedChannelId.value) return
  deliveriesLoading.value = true
  const result = await userApi.fetchNotificationDeliveries(selectedChannelId.value, 100)
  deliveriesLoading.value = false
  if (!result.ok) {
    toastError(result, 'notifications.deliveries.loadFailed')
    return
  }
  deliveries.value = Array.isArray(result.data?.items) ? result.data.items : []
  unackedDeliveries.value = Number(result.data?.unacked) || 0
}

async function retryDelivery(delivery) {
  deliveryBusy.value = delivery.id
  const result = await userApi.retryNotificationDelivery(selectedChannelId.value, delivery.id)
  deliveryBusy.value = 0
  if (!result.ok) {
    toastError(result, 'notifications.deliveries.retryFailed')
    return
  }
  await loadDeliveries()
}

async function ackDelivery(delivery) {
  deliveryBusy.value = delivery.id
  const result = await userApi.ackNotificationDelivery(selectedChannelId.value, delivery.id)
  deliveryBusy.value = 0
  if (!result.ok) {
    toastError(result, 'notifications.deliveries.ackFailed')
    return
  }
  await loadDeliveries()
  showToast(t('notifications.deliveries.ackedToast'), 'success')
}

async function retryChannelFailed() {
  if (!selectedChannelId.value) return
  deliveryBulkBusy.value = true
  const result = await userApi.retryFailedNotificationDeliveries(selectedChannelId.value, {})
  deliveryBulkBusy.value = false
  if (!result.ok) {
    toastError(result, 'notifications.deliveries.retryFailed')
    return
  }
  showToast(t('notifications.deliveries.requeued', {
    count: Number(result.data?.requeued) || 0,
  }), 'success')
  await loadDeliveries()
}

async function retryAllFailed() {
  if (!confirm(t('notifications.channels.retryAllConfirm'))) return
  retryingAll.value = true
  const result = await userApi.retryFailedNotificationDeliveriesForUser({})
  retryingAll.value = false
  if (!result.ok) {
    toastError(result, 'notifications.deliveries.retryFailed')
    return
  }
  showToast(t('notifications.channels.retryAllDone', {
    count: Number(result.data?.requeued) || 0,
  }), 'success')
  if (selectedChannelId.value) await loadDeliveries()
}

async function loadOfflineAlert() {
  if (!selectedDeviceId.value) {
    applyOfflineAlert({ device_id: '' })
    return
  }
  const result = await userApi.fetchDeviceOfflineAlert(selectedDeviceId.value)
  if (!result.ok) {
    toastError(result, 'notifications.offline.loadFailed')
    return
  }
  applyOfflineAlert(result.data)
}

function applyOfflineAlert(data) {
  const view = normalizeOfflineAlert(data)
  offlineForm.enabled = view.enabled
  offlineForm.after_minutes = view.after_minutes
  offlineForm.repeat_minutes = view.repeat_minutes
  offlineForm.snoozed_until = toDatetimeLocalValue(view.snoozed_until)
  offlineForm.maintenance_windows = view.maintenance_windows
  offlineForm.maintenance_timezone = view.maintenance_timezone
  offlineMeta.in_maintenance_window = view.in_maintenance_window
  offlineMeta.offline_since = view.offline_since
  offlineMeta.last_alerted_at = view.last_alerted_at
  offlineMeta.updated_at = view.updated_at
}

async function saveOfflineAlert() {
  if (!selectedDeviceId.value) return
  offlineSaving.value = true
  const result = await userApi.updateDeviceOfflineAlert(
    selectedDeviceId.value,
    buildOfflineAlertPayload(offlineForm),
  )
  offlineSaving.value = false
  if (!result.ok) {
    toastError(result, 'notifications.offline.saveFailed')
    return
  }
  applyOfflineAlert(result.data)
  showToast(t('notifications.offline.saved'), 'success')
}

async function removeOfflineAlert() {
  if (!selectedDeviceId.value || !confirm(t('notifications.offline.resetConfirm'))) return
  offlineSaving.value = true
  const result = await userApi.deleteDeviceOfflineAlert(selectedDeviceId.value)
  offlineSaving.value = false
  if (!result.ok) {
    toastError(result, 'notifications.offline.saveFailed')
    return
  }
  applyOfflineAlert({ device_id: selectedDeviceId.value })
  showToast(t('notifications.offline.resetDone'), 'success')
}

async function loadMetricAlert() {
  metricError.value = ''
  if (!metricDeviceId.value) {
    applyMetricAlert({})
    return
  }
  const result = await userApi.fetchDeviceMetricAlert(metricDeviceId.value)
  if (!result.ok) {
    setMetricError(result, 'notifications.metricAlert.loadFailed')
    return
  }
  applyMetricAlert(result.data)
}

function applyMetricAlert(data) {
  const view = normalizeMetricAlert(data)
  metricForm.enabled = view.enabled
  metricForm.cpu_percent = view.cpu_percent
  metricForm.memory_percent = view.memory_percent
  metricForm.disk_percent = view.disk_percent
  metricForm.sustained_minutes = view.sustained_minutes
  metricForm.repeat_minutes = view.repeat_minutes
  metricForm.snoozed_until = toDatetimeLocalValue(view.snoozed_until)
  metricMeta.current = view.current
  metricMeta.states = view.states
}

function validateMetricAlert() {
  const thresholds = [
    Number(metricForm.cpu_percent),
    Number(metricForm.memory_percent),
    Number(metricForm.disk_percent),
  ]
  if (thresholds.some((value) => !Number.isFinite(value) || value < 0 || value > 100)) {
    return t('notifications.metricAlert.invalidThreshold')
  }
  if (metricForm.enabled && thresholds.every((value) => value === 0)) {
    return t('notifications.metricAlert.thresholdRequired')
  }
  const sustainedMinutes = Number(metricForm.sustained_minutes)
  if (!Number.isInteger(sustainedMinutes) || sustainedMinutes < 0 || sustainedMinutes > 10080) {
    return t('notifications.metricAlert.invalidSustained')
  }
  const repeatMinutes = Number(metricForm.repeat_minutes)
  if (!Number.isInteger(repeatMinutes) || repeatMinutes < 0 || repeatMinutes > 10080) {
    return t('notifications.metricAlert.invalidRepeat')
  }
  return ''
}

async function saveMetricAlert() {
  if (!metricDeviceId.value) return
  const validationError = validateMetricAlert()
  if (validationError) {
    metricError.value = validationError
    showToast(validationError, 'error')
    return
  }
  metricSaving.value = true
  metricError.value = ''
  const result = await userApi.updateDeviceMetricAlert(
    metricDeviceId.value,
    buildMetricAlertPayload(metricForm),
  )
  metricSaving.value = false
  if (!result.ok) {
    setMetricError(result, 'notifications.metricAlert.saveFailed')
    return
  }
  applyMetricAlert(result.data)
  showToast(t('notifications.metricAlert.saved'), 'success')
}

async function removeMetricAlert() {
  if (!metricDeviceId.value || !confirm(t('notifications.metricAlert.resetConfirm'))) return
  metricSaving.value = true
  metricError.value = ''
  const result = await userApi.deleteDeviceMetricAlert(metricDeviceId.value)
  metricSaving.value = false
  if (!result.ok) {
    setMetricError(result, 'notifications.metricAlert.saveFailed')
    return
  }
  applyMetricAlert({ device_id: metricDeviceId.value })
  showToast(t('notifications.metricAlert.resetDone'), 'success')
}

function setMetricError(result, fallbackKey) {
  const message = result?.error || t(fallbackKey)
  metricError.value = message
  showToast(message, 'error')
}

onMounted(() => {
  if (authState.isLoggedIn) refreshAll()
})
</script>

<style scoped>
.notifications-page {
  max-width: 1180px;
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
  max-width: 700px;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.login-button {
  margin-top: 12px;
}

.surface {
  margin-bottom: 18px;
  padding: 22px;
  border: 1px solid rgba(255, 255, 255, 0.09);
  border-radius: 8px;
  background: var(--bg-card);
}

.surface-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
  margin-bottom: 18px;
}

.surface-head h3 {
  margin-bottom: 4px;
  font-size: 17px;
  letter-spacing: 0;
}

.surface-head p {
  max-width: 700px;
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.55;
}

.head-actions,
.row-actions,
.form-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.channel-list {
  display: grid;
  gap: 1px;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.08);
}

.channel-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(280px, auto);
  gap: 18px;
  padding: 16px;
  background: rgba(15, 22, 41, 0.72);
}

.channel-title-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.channel-title-line strong {
  font-size: 15px;
}

.channel-url {
  margin-top: 5px;
  overflow-wrap: anywhere;
  color: #b7c6ec;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
}

.event-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  margin-top: 10px;
}

.event-tag {
  padding: 3px 7px;
  border: 1px solid rgba(89, 196, 178, 0.24);
  border-radius: 5px;
  background: rgba(89, 196, 178, 0.08);
  color: #9de1d4;
  font-size: 11px;
}

.channel-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 16px;
  margin-top: 9px;
  color: var(--text-secondary);
  font-size: 11px;
}

.row-actions {
  align-content: flex-start;
  justify-content: flex-end;
}

.state-chip {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 2px 8px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 999px;
  color: var(--text-secondary);
  font-size: 11px;
  white-space: nowrap;
}

.state-chip.enabled {
  border-color: rgba(76, 175, 80, 0.35);
  background: rgba(76, 175, 80, 0.1);
  color: #a9e9ad;
}

.state-chip.warning {
  border-color: rgba(255, 193, 7, 0.35);
  background: rgba(255, 193, 7, 0.09);
  color: #ffdc7a;
}

.state-chip.danger {
  border-color: rgba(244, 67, 54, 0.36);
  background: rgba(244, 67, 54, 0.1);
  color: #ffaaa3;
}

.state-chip.paused,
.state-chip.neutral {
  background: rgba(255, 255, 255, 0.04);
}

.btn-danger {
  border: 1px solid rgba(244, 67, 54, 0.3);
  background: rgba(244, 67, 54, 0.1);
  color: #ffaaa3;
}

.field-grid {
  display: grid;
  gap: 14px;
}

.field-grid.two {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.field-grid.three {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.field {
  display: grid;
  gap: 6px;
  color: var(--text-secondary);
  font-size: 12px;
}

.field small {
  color: var(--text-secondary);
  font-size: 11px;
  line-height: 1.5;
}

.toggle-field {
  align-content: start;
}

.switch-row {
  display: flex;
  min-height: 40px;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid var(--border);
  border-radius: 8px;
  color: var(--text-primary);
}

.switch-row input {
  accent-color: var(--success);
}

.subsection {
  margin-top: 18px;
}

.subsection-title {
  margin-bottom: 9px;
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 600;
}

.form-actions {
  margin-top: 18px;
}

.offline-form {
  margin-top: 18px;
}

.metric-alert-form {
  margin-top: 18px;
}

.metric-current-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
}

.metric-reading {
  display: grid;
  align-content: start;
  gap: 5px;
  min-height: 104px;
  padding: 12px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.025);
}

.metric-reading > span,
.metric-reading small {
  color: var(--text-secondary);
  font-size: 11px;
}

.metric-reading strong {
  overflow-wrap: anywhere;
  font-size: 22px;
  line-height: 1.15;
}

.metric-reading .metric-mount {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 14px;
}

.metric-state-list {
  display: grid;
  gap: 6px;
  margin: 12px 0 18px;
}

.metric-state-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px 10px;
  min-height: 30px;
  padding: 4px 10px;
  border-left: 2px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.02);
  font-size: 12px;
}

.metric-state-row > span:first-child {
  min-width: 54px;
  font-weight: 600;
}

.metric-state-row small {
  color: var(--text-secondary);
  font-size: 10px;
}

.metric-snooze {
  max-width: 360px;
  margin-top: 14px;
}

.inline-error {
  margin-top: 14px;
  color: #ffaaa3;
  font-size: 12px;
  line-height: 1.5;
}

.meta-strip,
.signature-line,
.unacked-count {
  color: var(--text-secondary);
  font-size: 11px;
}

.signature-line {
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.07);
}

.meta-strip {
  margin-top: 14px;
  padding: 10px 12px;
  border-left: 2px solid rgba(255, 193, 7, 0.5);
  background: rgba(255, 193, 7, 0.045);
}

.loading-line {
  padding: 24px 0;
  color: var(--text-secondary);
  font-size: 13px;
}

.empty-state.compact {
  padding: 26px 14px;
}

.delivery-table-wrap {
  overflow-x: auto;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
}

.delivery-table {
  width: 100%;
  min-width: 840px;
  border-collapse: collapse;
  font-size: 12px;
}

.delivery-table th,
.delivery-table td {
  padding: 10px 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.07);
  text-align: left;
  vertical-align: top;
}

.delivery-table th {
  background: rgba(255, 255, 255, 0.035);
  color: var(--text-secondary);
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
}

.delivery-table tr:last-child td {
  border-bottom: 0;
}

.delivery-table small,
.table-note {
  color: var(--text-secondary);
  font-size: 10px;
}

.table-error {
  max-width: 250px;
  margin-top: 4px;
  color: #ffaaa3;
  overflow-wrap: anywhere;
}

.delivery-actions {
  white-space: nowrap;
}

.text-action {
  border: 0;
  background: transparent;
  color: #9db9ff;
  cursor: pointer;
  font: inherit;
}

.text-action + .text-action {
  margin-left: 8px;
}

.text-action:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.dialog-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: grid;
  place-items: center;
  padding: 24px;
  background: rgba(5, 9, 20, 0.76);
}

.channel-dialog {
  width: min(760px, 100%);
  max-height: min(820px, calc(100vh - 48px));
  overflow-y: auto;
  padding: 22px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 8px;
  background: var(--bg-card);
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.4);
}

.dialog-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 18px;
}

.dialog-head h3 {
  font-size: 18px;
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

.channel-dialog > .field {
  margin-top: 14px;
}

.event-picker {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px;
}

.event-choice {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 2px 8px;
  padding: 9px 10px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 7px;
  color: var(--text-primary);
  font-size: 12px;
}

.event-choice input {
  grid-row: 1 / span 2;
  margin-top: 2px;
  accent-color: var(--success);
}

.event-choice code {
  overflow: hidden;
  color: var(--text-secondary);
  font-size: 9px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 900px) {
  .channel-row {
    grid-template-columns: 1fr;
  }

  .row-actions {
    justify-content: flex-start;
  }

  .field-grid.three {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 680px) {
  .page-heading,
  .surface-head {
    flex-direction: column;
  }

  .field-grid.two,
  .event-picker {
    grid-template-columns: 1fr;
  }

  .metric-current-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .surface {
    padding: 16px;
  }
}

@media (max-width: 420px) {
  .metric-current-grid {
    grid-template-columns: 1fr;
  }
}
</style>
