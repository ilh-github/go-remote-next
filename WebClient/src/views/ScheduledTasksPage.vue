<template>
  <div class="scheduled-page">
    <div class="page-heading">
      <div>
        <h2 class="page-title">{{ $t('scheduled.title') }}</h2>
        <p class="page-copy">{{ $t('scheduled.subtitle') }}</p>
      </div>
      <button class="btn btn-secondary btn-sm" :disabled="loading" @click="loadTasks">
        🔄 {{ $t('scheduled.refresh') }}
      </button>
    </div>

    <div v-if="!authState.isLoggedIn" class="card">
      <div class="empty-state">
        <div class="empty-icon">🔒</div>
        <p>{{ $t('scheduled.loginRequired') }}</p>
        <button class="btn btn-primary btn-sm login-button" @click="showLogin()">
          {{ $t('user.login') }}
        </button>
      </div>
    </div>

    <template v-else>
      <div class="summary-strip">
        <div class="summary-item">
          <span class="summary-value">{{ activeCount }}</span>
          <span class="summary-label">{{ $t('scheduled.summary.active') }}</span>
        </div>
        <div class="summary-item">
          <span class="summary-value">{{ recurringCount }}</span>
          <span class="summary-label">{{ $t('scheduled.summary.recurring') }}</span>
        </div>
        <div class="summary-item">
          <span class="summary-value danger">{{ failedCount }}</span>
          <span class="summary-label">{{ $t('scheduled.summary.failed') }}</span>
        </div>
        <button class="btn btn-primary create-button" @click="openCreate">
          {{ $t('scheduled.create') }}
        </button>
      </div>

      <section class="surface">
        <div class="surface-head">
          <div>
            <h3>{{ $t('scheduled.list.title') }}</h3>
            <p>{{ $t('scheduled.list.subtitle') }}</p>
          </div>
          <span class="clock-note">{{ $t('scheduled.list.workerNote') }}</span>
        </div>

        <div v-if="loading && tasks.length === 0" class="loading-line">
          {{ $t('scheduled.loading') }}
        </div>
        <div v-else-if="tasks.length === 0" class="empty-state compact">
          <div class="empty-icon">🗓️</div>
          <p>{{ $t('scheduled.empty') }}</p>
          <p class="hint">{{ $t('scheduled.emptyHint') }}</p>
          <button class="btn btn-primary btn-sm" @click="openCreate">
            {{ $t('scheduled.create') }}
          </button>
        </div>
        <div v-else class="task-list">
          <article v-for="task in tasks" :key="task.id" class="task-row">
            <div :class="['task-rail', taskTone(task)]"></div>
            <div class="task-main">
              <div class="task-title-line">
                <span class="task-icon">🗓️</span>
                <strong>{{ kindLabel(task.kind) }}</strong>
                <span :class="['state-chip', taskTone(task)]">{{ taskStateLabel(task) }}</span>
                <span class="state-chip neutral">{{ modeLabel(task) }}</span>
              </div>
              <div class="cadence-line">{{ cadenceLabel(task) }}</div>
              <div class="task-meta">
                <span>
                  <b>{{ $t('scheduled.fields.nextRun') }}</b>
                  {{ formatTime(task.next_run_at) }}
                </span>
                <span v-if="task.last_run_at">
                  <b>{{ $t('scheduled.fields.lastRun') }}</b>
                  {{ formatTime(task.last_run_at) }}
                </span>
                <span>
                  <b>{{ $t('scheduled.fields.attempts') }}</b>
                  {{ task.attempts }} / {{ task.max_attempts }}
                </span>
                <span>
                  <b>{{ $t('scheduled.fields.taskId') }}</b>
                  #{{ task.id }}
                </span>
              </div>
              <div v-if="task.last_error" class="task-error">
                <b>{{ $t('scheduled.fields.lastError') }}</b> {{ task.last_error }}
              </div>
            </div>
            <div class="task-actions">
              <button class="btn btn-danger btn-sm" @click="deleteTask(task)">
                {{ $t('scheduled.delete') }}
              </button>
            </div>
          </article>
        </div>
      </section>

      <section class="help-band">
        <div>
          <strong>{{ $t('scheduled.help.title') }}</strong>
          <p>{{ $t('scheduled.help.body') }}</p>
        </div>
        <code>{{ supportedSchedules.join('  ·  ') }}</code>
      </section>
    </template>

    <div v-if="dialog.show" class="dialog-overlay" @click.self="closeDialog">
      <div class="dialog task-dialog">
        <div class="dialog-head">
          <div>
            <h3>{{ $t('scheduled.dialog.title') }}</h3>
            <p>{{ $t('scheduled.dialog.subtitle') }}</p>
          </div>
          <button class="icon-close" :title="$t('scheduled.close')" @click="closeDialog">✕</button>
        </div>

        <div class="mode-switch" role="group" :aria-label="$t('scheduled.fields.mode')">
          <button
            v-for="option in modeOptions"
            :key="option.value"
            :class="['mode-button', { active: dialog.mode === option.value }]"
            @click="dialog.mode = option.value"
          >
            <strong>{{ $t(option.label) }}</strong>
            <span>{{ $t(option.hint) }}</span>
          </button>
        </div>

        <div class="field-grid two">
          <label class="field">
            <span>{{ $t('scheduled.fields.kind') }}</span>
            <select v-model="dialog.kind" class="form-input">
              <option v-for="kind in availableKinds" :key="kind" :value="kind">
                {{ kindLabel(kind) }}
              </option>
            </select>
            <small>{{ $t('scheduled.fields.kindHint') }}</small>
          </label>
          <label class="field">
            <span>{{ $t('scheduled.fields.maxAttempts') }}</span>
            <input v-model.number="dialog.max_attempts" class="form-input" type="number" min="1" max="10" />
            <small>{{ $t('scheduled.fields.maxAttemptsHint') }}</small>
          </label>
        </div>

        <template v-if="dialog.mode !== 'calendar'">
          <div class="field-grid two schedule-fields">
            <label class="field">
              <span>{{ $t('scheduled.fields.runAt') }}</span>
              <input v-model="dialog.run_at" class="form-input" type="datetime-local" />
              <small>{{ $t('scheduled.fields.runAtHint') }}</small>
            </label>
            <label v-if="dialog.mode === 'interval'" class="field">
              <span>{{ $t('scheduled.fields.repeatMinutes') }}</span>
              <input v-model.number="dialog.repeat_minutes" class="form-input" type="number" min="1" max="525600" />
              <small>{{ $t('scheduled.fields.repeatMinutesHint') }}</small>
            </label>
            <div v-else class="once-note">
              <strong>{{ $t('scheduled.dialog.onceTitle') }}</strong>
              <p>{{ $t('scheduled.dialog.onceBody') }}</p>
            </div>
          </div>
        </template>

        <template v-else>
          <div class="field-grid three schedule-fields">
            <label class="field">
              <span>{{ $t('scheduled.fields.cadence') }}</span>
              <select v-model="dialog.calendarType" class="form-input">
                <option value="daily">{{ $t('scheduled.calendar.daily') }}</option>
                <option value="weekdays">{{ $t('scheduled.calendar.weekdays') }}</option>
                <option value="weekly">{{ $t('scheduled.calendar.weekly') }}</option>
                <option value="monthly">{{ $t('scheduled.calendar.monthly') }}</option>
                <option value="every">{{ $t('scheduled.calendar.every') }}</option>
              </select>
            </label>
            <label v-if="dialog.calendarType === 'weekly'" class="field">
              <span>{{ $t('scheduled.fields.weekday') }}</span>
              <select v-model="dialog.weekday" class="form-input">
                <option v-for="day in weekdays" :key="day.value" :value="day.value">{{ day.label }}</option>
              </select>
            </label>
            <label v-if="dialog.calendarType === 'monthly'" class="field">
              <span>{{ $t('scheduled.fields.monthDay') }}</span>
              <input v-model.number="dialog.monthDay" class="form-input" type="number" min="1" max="31" />
              <small>{{ $t('scheduled.fields.monthDayHint') }}</small>
            </label>
            <label v-if="dialog.calendarType === 'every'" class="field">
              <span>{{ $t('scheduled.fields.everyDays') }}</span>
              <input v-model.number="dialog.everyDays" class="form-input" type="number" min="1" max="365" />
              <small>{{ $t('scheduled.fields.everyDaysHint') }}</small>
            </label>
            <label class="field">
              <span>{{ $t('scheduled.fields.time') }}</span>
              <input v-model="dialog.time" class="form-input" type="time" />
            </label>
          </div>
          <div class="field-grid two schedule-fields">
            <label class="field">
              <span>{{ $t('scheduled.fields.timezone') }}</span>
              <input v-model="dialog.timezone" class="form-input" type="text" placeholder="Asia/Shanghai" />
              <small>{{ $t('scheduled.fields.timezoneHint') }}</small>
            </label>
            <label class="holiday-toggle">
              <input v-model="dialog.skip_holidays" type="checkbox" />
              <span>
                <strong>{{ $t('scheduled.fields.skipHolidays') }}</strong>
                <small>{{ $t('scheduled.fields.skipHolidaysHint') }}</small>
              </span>
            </label>
          </div>
        </template>

        <div class="preview-line">
          <span>{{ $t('scheduled.dialog.preview') }}</span>
          <code>{{ payloadPreview }}</code>
        </div>

        <div class="form-actions">
          <button class="btn btn-primary" :disabled="saving" @click="saveTask">
            {{ saving ? '...' : $t('scheduled.dialog.create') }}
          </button>
          <button class="btn btn-secondary" :disabled="saving" @click="closeDialog">
            {{ $t('user.cancel') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import {
  computed,
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
  buildScheduledTaskPayload,
  normalizeScheduledTaskList,
  toScheduledTaskLocalValue,
} from '../api/scheduledTaskModels'
import { fallbackScheduledKinds, scheduledKindLabelKey } from '../../js/scheduled/kindLabels'

const { t } = useI18n()
const authState = inject('authState')
const showToast = inject('showToast')
const showLogin = inject('showLogin')

const tasks = ref([])
const supportedKinds = ref([])
const supportedSchedules = ref([])
const loading = ref(false)
const saving = ref(false)
const refreshPending = ref(false)
let refreshTimer = null
let pollTimer = null

const dialog = reactive({
  show: false,
  mode: 'once',
  kind: 'sign_out_all_sessions',
  run_at: '',
  repeat_minutes: 60,
  max_attempts: 1,
  calendarType: 'daily',
  weekday: 'mon',
  monthDay: 1,
  everyDays: 3,
  time: '22:00',
  timezone: '',
  skip_holidays: false,
})

const modeOptions = [
  { value: 'once', label: 'scheduled.mode.once', hint: 'scheduled.mode.onceHint' },
  { value: 'interval', label: 'scheduled.mode.interval', hint: 'scheduled.mode.intervalHint' },
  { value: 'calendar', label: 'scheduled.mode.calendar', hint: 'scheduled.mode.calendarHint' },
]

const weekdays = computed(() => [
  { value: 'mon', label: t('scheduled.weekdays.mon') },
  { value: 'tue', label: t('scheduled.weekdays.tue') },
  { value: 'wed', label: t('scheduled.weekdays.wed') },
  { value: 'thu', label: t('scheduled.weekdays.thu') },
  { value: 'fri', label: t('scheduled.weekdays.fri') },
  { value: 'sat', label: t('scheduled.weekdays.sat') },
  { value: 'sun', label: t('scheduled.weekdays.sun') },
])

const availableKinds = computed(() => (
  supportedKinds.value.length
    ? supportedKinds.value
    : fallbackScheduledKinds
))
const activeCount = computed(() => tasks.value.filter((task) => task.enabled).length)
const recurringCount = computed(() => tasks.value.filter((task) => (
  task.enabled && (task.schedule || task.repeat_seconds > 0)
)).length)
const failedCount = computed(() => tasks.value.filter((task) => task.last_status === 'failed').length)

const payloadPreview = computed(() => {
  const payload = currentPayload()
  if (!payload) return t('scheduled.dialog.invalidPreview')
  return JSON.stringify(payload, null, 2)
})

// The map lives in a module so a kind without a label is a failing test rather
// than a raw wire name in a table cell.
function kindLabel(kind) {
  return t(`scheduled.kinds.${scheduledKindLabelKey(kind)}`)
}

function weekdayLabel(value) {
  return t(`scheduled.weekdays.${value}`)
}

function formatTime(value) {
  if (!value) return t('scheduled.never')
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? '' : parsed.toLocaleString()
}

function formatInterval(seconds) {
  const minutes = Math.max(1, Math.round(Number(seconds || 0) / 60))
  if (minutes % 1440 === 0) return t('scheduled.duration.days', { count: minutes / 1440 })
  if (minutes % 60 === 0) return t('scheduled.duration.hours', { count: minutes / 60 })
  return t('scheduled.duration.minutes', { count: minutes })
}

function cadenceLabel(task) {
  if (task.schedule) {
    const parts = task.schedule.split('@')
    const time = parts[parts.length - 1]
    if (parts[0] === 'daily') return t('scheduled.cadence.daily', { time })
    if (parts[0] === 'weekdays') return t('scheduled.cadence.weekdays', { time })
    if (parts[0] === 'weekly') {
      return t('scheduled.cadence.weekly', { day: weekdayLabel(parts[1]), time })
    }
    if (parts[0] === 'monthly') return t('scheduled.cadence.monthly', { day: parts[1], time })
    if (parts[0] === 'every') return t('scheduled.cadence.every', { days: parts[1], time })
    return task.schedule
  }
  if (task.repeat_seconds > 0) {
    return t('scheduled.cadence.interval', { interval: formatInterval(task.repeat_seconds) })
  }
  return t('scheduled.cadence.once')
}

function modeLabel(task) {
  if (task.schedule) return t('scheduled.mode.calendar')
  if (task.repeat_seconds > 0) return t('scheduled.mode.interval')
  return t('scheduled.mode.once')
}

function taskTone(task) {
  if (task.last_status === 'failed') return 'danger'
  if (!task.enabled) return 'neutral'
  if (task.attempts > 0) return 'warning'
  return 'enabled'
}

function taskStateLabel(task) {
  if (task.last_status === 'failed') return t('scheduled.status.failed')
  if (!task.enabled) return t('scheduled.status.disabled')
  if (task.attempts > 0) return t('scheduled.status.retrying')
  return t('scheduled.status.scheduled')
}

function defaultRunAt() {
  return toScheduledTaskLocalValue(new Date(Date.now() + 60 * 60 * 1000).toISOString())
}

function resetDialog() {
  dialog.mode = 'once'
  dialog.kind = availableKinds.value[0]
  dialog.run_at = defaultRunAt()
  dialog.repeat_minutes = 60
  dialog.max_attempts = 1
  dialog.calendarType = 'daily'
  dialog.weekday = 'mon'
  dialog.monthDay = 1
  dialog.everyDays = 3
  dialog.time = '22:00'
  dialog.timezone = Intl.DateTimeFormat().resolvedOptions().timeZone || ''
  dialog.skip_holidays = false
}

function openCreate() {
  resetDialog()
  dialog.show = true
}

function closeDialog() {
  if (!saving.value) dialog.show = false
}

function currentPayload() {
  const form = {
    ...dialog,
    repeat_seconds: Number(dialog.repeat_minutes) * 60,
  }
  if (dialog.mode === 'calendar') {
    if (!dialog.time) return null
    if (dialog.calendarType === 'weekly' && !dialog.weekday) return null
    return buildScheduledTaskPayload(form)
  }
  const payload = buildScheduledTaskPayload(form, {
    timeZoneOffsetMinutes: new Date().getTimezoneOffset(),
  })
  if (!payload.run_at || !payload.kind) return null
  if (new Date(payload.run_at).getTime() <= Date.now()) return null
  if (dialog.mode === 'interval' && payload.repeat_seconds < 60) return null
  return payload
}

function validatePayload(payload) {
  if (!payload) return t('scheduled.errors.invalidSchedule')
  if (dialog.mode === 'interval' && (
    payload.repeat_seconds < 60 || payload.repeat_seconds > 365 * 24 * 60 * 60
  )) return t('scheduled.errors.invalidInterval')
  if (payload.max_attempts < 1 || payload.max_attempts > 10) {
    return t('scheduled.errors.invalidAttempts')
  }
  return ''
}

async function loadTasks() {
  if (loading.value) {
    refreshPending.value = true
    return
  }
  loading.value = true
  const result = await userApi.fetchScheduledTasks()
  loading.value = false
  if (refreshPending.value) {
    refreshPending.value = false
    scheduleRefresh()
  }
  if (!result.ok) {
    showToast(result.error || t('scheduled.loadFailed'), 'error')
    return
  }
  const view = normalizeScheduledTaskList(result.data)
  tasks.value = view.items
  supportedKinds.value = view.supportedKinds
  supportedSchedules.value = view.supportedSchedules
}

function scheduleRefresh() {
  clearTimeout(refreshTimer)
  refreshTimer = setTimeout(loadTasks, 150)
}

async function saveTask() {
  const payload = currentPayload()
  const error = validatePayload(payload)
  if (error) {
    showToast(error, 'error')
    return
  }
  saving.value = true
  const result = await userApi.createScheduledTask(payload)
  saving.value = false
  if (!result.ok) {
    showToast(result.error || t('scheduled.createFailed'), 'error')
    return
  }
  dialog.show = false
  await loadTasks()
  showToast(t('scheduled.created'), 'success')
}

async function deleteTask(task) {
  if (!confirm(t('scheduled.deleteConfirm', { kind: kindLabel(task.kind) }))) return
  const result = await userApi.deleteScheduledTask(task.id)
  if (!result.ok) {
    showToast(result.error || t('scheduled.deleteFailed'), 'error')
    return
  }
  tasks.value = tasks.value.filter((row) => row.id !== task.id)
  showToast(t('scheduled.deleted'), 'success')
}

function onTaskChanged(event) {
  if (event.detail?.kind) {
    showToast(t('scheduled.failedEvent', { kind: kindLabel(event.detail.kind) }), 'error')
  }
  scheduleRefresh()
}

// A scheduled connection came due. The server published it; this is the half the
// client owes. Without a listener the prompt stops at the socket, which is the
// same half-wired shape the notification-subscription gap had a round earlier.
function onConnectPrompted() {
  // Refresh rather than build the toast from the frame alone: the row's schedule,
  // last_run_at and status are what the owner needs to act on, and they are
  // already on the server. The frame only carries the task id.
  scheduleRefresh()
  showToast(t('scheduled.promptEvent'), 'info')
}

onMounted(() => {
  if (authState.isLoggedIn) {
    loadTasks()
    pollTimer = setInterval(loadTasks, 30_000)
  }
  userSync.addEventListener('scheduled-task-changed', onTaskChanged)
  userSync.addEventListener('connect-prompted', onConnectPrompted)
})

onBeforeUnmount(() => {
  clearTimeout(refreshTimer)
  clearInterval(pollTimer)
  userSync.removeEventListener('scheduled-task-changed', onTaskChanged)
  userSync.removeEventListener('connect-prompted', onConnectPrompted)
})
</script>

<style scoped>
.scheduled-page {
  max-width: 1120px;
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
  max-width: 720px;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.login-button {
  margin-top: 12px;
}

.summary-strip {
  display: grid;
  grid-template-columns: repeat(3, minmax(110px, 150px)) 1fr;
  gap: 1px;
  margin-bottom: 18px;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.07);
}

.summary-item {
  display: grid;
  gap: 3px;
  padding: 14px 16px;
  background: rgba(15, 22, 41, 0.82);
}

.summary-value {
  font-size: 24px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.summary-value.danger {
  color: #ffaaa3;
}

.summary-label {
  color: var(--text-secondary);
  font-size: 11px;
}

.create-button {
  justify-self: end;
  align-self: center;
  margin-right: 14px;
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
.clock-note {
  color: var(--text-secondary);
  font-size: 11px;
  line-height: 1.55;
}

.task-list {
  display: grid;
  gap: 1px;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.08);
}

.task-row {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 18px;
  padding: 17px 16px 17px 20px;
  background: rgba(15, 22, 41, 0.72);
}

.task-rail {
  position: absolute;
  inset: 0 auto 0 0;
  width: 3px;
  background: rgba(255, 255, 255, 0.14);
}

.task-rail.enabled {
  background: var(--success);
}

.task-rail.warning {
  background: var(--warning);
}

.task-rail.danger {
  background: var(--error);
}

.task-title-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.task-title-line strong {
  font-size: 15px;
}

.task-icon {
  font-size: 16px;
}

.cadence-line {
  margin-top: 7px;
  color: #c5d2ee;
  font-size: 13px;
}

.task-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 5px 18px;
  margin-top: 9px;
  color: var(--text-secondary);
  font-size: 11px;
}

.task-meta b {
  color: #b8c3da;
  font-weight: 500;
}

.task-error {
  margin-top: 10px;
  padding: 8px 10px;
  border-left: 2px solid rgba(244, 67, 54, 0.55);
  background: rgba(244, 67, 54, 0.045);
  color: #ffc2bd;
  font-size: 11px;
  overflow-wrap: anywhere;
}

.task-actions {
  align-self: start;
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

.state-chip.neutral {
  background: rgba(255, 255, 255, 0.035);
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

.empty-state .btn {
  margin-top: 12px;
}

.help-band {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
  margin-top: 14px;
  padding: 14px 16px;
  border-left: 2px solid rgba(89, 196, 178, 0.55);
  background: rgba(89, 196, 178, 0.045);
}

.help-band strong {
  font-size: 12px;
}

.help-band p {
  max-width: 660px;
  margin-top: 3px;
  color: var(--text-secondary);
  font-size: 11px;
  line-height: 1.55;
}

.help-band code {
  color: #9de1d4;
  font-size: 10px;
  text-align: right;
  overflow-wrap: anywhere;
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

.task-dialog {
  width: min(820px, 100%);
  max-height: min(860px, calc(100vh - 48px));
  overflow-y: auto;
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

.mode-switch {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 7px;
  margin-bottom: 18px;
}

.mode-button {
  display: grid;
  gap: 3px;
  padding: 11px 12px;
  border: 1px solid rgba(255, 255, 255, 0.09);
  border-radius: 7px;
  background: rgba(255, 255, 255, 0.025);
  color: var(--text-secondary);
  cursor: pointer;
  text-align: left;
}

.mode-button strong {
  color: var(--text-primary);
  font-size: 12px;
}

.mode-button span {
  font-size: 10px;
  line-height: 1.4;
}

.mode-button.active {
  border-color: rgba(233, 69, 96, 0.55);
  background: rgba(233, 69, 96, 0.08);
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

.schedule-fields {
  margin-top: 14px;
}

.field {
  display: grid;
  gap: 6px;
  color: var(--text-secondary);
  font-size: 12px;
}

.field small,
.holiday-toggle small {
  color: var(--text-secondary);
  font-size: 10px;
  line-height: 1.45;
}

.once-note {
  padding: 11px 12px;
  border: 1px dashed rgba(255, 255, 255, 0.1);
  border-radius: 7px;
}

.once-note strong {
  font-size: 12px;
}

.once-note p {
  margin-top: 4px;
  color: var(--text-secondary);
  font-size: 10px;
  line-height: 1.5;
}

.holiday-toggle {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 11px 12px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 7px;
}

.holiday-toggle input {
  margin-top: 2px;
  accent-color: var(--success);
}

.holiday-toggle span {
  display: grid;
  gap: 3px;
}

.holiday-toggle strong {
  font-size: 12px;
}

.preview-line {
  display: grid;
  gap: 7px;
  margin-top: 18px;
  padding: 10px 12px;
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 7px;
  background: rgba(0, 0, 0, 0.12);
  color: var(--text-secondary);
  font-size: 10px;
}

.preview-line code {
  max-height: 130px;
  overflow: auto;
  color: #bcd4ff;
  font-size: 10px;
  line-height: 1.5;
  white-space: pre-wrap;
}

.form-actions {
  display: flex;
  gap: 8px;
  margin-top: 18px;
}

@media (max-width: 820px) {
  .summary-strip {
    grid-template-columns: repeat(3, 1fr);
  }

  .create-button {
    grid-column: 1 / -1;
    justify-self: stretch;
    margin: 0;
  }

  .field-grid.three {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 680px) {
  .page-heading,
  .surface-head,
  .help-band {
    flex-direction: column;
  }

  .summary-strip {
    grid-template-columns: 1fr;
  }

  .task-row {
    grid-template-columns: 1fr;
  }

  .task-actions {
    justify-self: start;
  }

  .surface,
  .task-dialog {
    padding: 16px;
  }

  .field-grid.two,
  .mode-switch {
    grid-template-columns: 1fr;
  }

  .help-band code {
    text-align: left;
  }
}
</style>
