<template>
  <div>
    <h2 class="page-title">{{ $t('devices.title') }}</h2>

    <div v-if="!authState.isLoggedIn" class="card">
      <div class="empty-state">
        <div class="empty-icon">🔒</div>
        <p>{{ $t('devices.loginRequired') }}</p>
        <button class="btn btn-primary btn-sm" style="margin-top:12px;" @click="showLogin()">{{ $t('user.login') }}</button>
      </div>
    </div>

    <template v-else>
      <div class="devices-toolbar">
        <div class="group-filter" role="group" :aria-label="$t('devices.groups.filterLabel')">
          <button
            :class="['group-filter-chip', { active: selectedGroupId === null }]"
            @click="selectedGroupId = null"
          >
            {{ $t('devices.groups.all') }}
            <span class="group-filter-count">{{ myDevices.length }}</span>
          </button>
          <button
            v-for="group in deviceGroups"
            :key="group.id"
            :class="['group-filter-chip', { active: selectedGroupId === group.id }]"
            @click="selectedGroupId = group.id"
          >
            <span
              class="group-color-dot"
              :style="{ backgroundColor: group.color || defaultGroupColor }"
            ></span>
            <span class="group-filter-name">{{ group.name }}</span>
            <span class="group-filter-count">{{ boundDeviceCount(group) }}</span>
          </button>
        </div>
        <div class="toolbar-actions">
          <button class="btn btn-secondary btn-sm" @click="openGroupDialog()">
            {{ $t('devices.groups.manage') }}
          </button>
          <button class="btn btn-secondary btn-sm" :disabled="loading" @click="refresh">
            🔄 {{ $t('devices.refresh') }}
          </button>
        </div>
      </div>

      <!-- My Devices -->
      <div class="card">
        <div class="card-heading">
          <div>
            <div class="card-title">
              {{ selectedGroup ? selectedGroup.name : $t('devices.myDevices') }}
            </div>
            <div v-if="selectedGroup?.description" class="card-subtitle group-description">
              {{ selectedGroup.description }}
            </div>
          </div>
          <span class="device-count">
            {{ $t('devices.groups.deviceCount', { count: visibleDevices.length }) }}
          </span>
          <span v-if="quota.limited" class="device-count">
            {{ $t('devices.quota.ofLimit', { owned: quota.owned, quota: quota.quota }) }}
          </span>
          <span v-if="quotaReached" class="device-quota-warning">
            {{ $t('devices.quota.reached') }}
          </span>
        </div>
        <div v-if="myDevices.length === 0" class="empty-state">
          <div class="empty-icon">📱</div>
          <p>{{ $t('devices.noDevices') }}</p>
        </div>
        <div v-else-if="visibleDevices.length === 0" class="empty-state">
          <div class="empty-icon">📂</div>
          <p>{{ $t('devices.groups.noGroupDevices') }}</p>
          <button class="btn btn-secondary btn-sm empty-action" @click="editGroupMembers(selectedGroup)">
            {{ $t('devices.groups.editMembers') }}
          </button>
        </div>
        <div v-for="d in visibleDevices" :key="d.device_id" class="device-item">
          <div :class="['device-status', statusClass(d)]"></div>
          <div class="device-info">
            <div class="device-name">{{ d.remark || d.device_name || d.device_id }}</div>
            <div class="device-id">
              {{ d.device_id }}
              <span v-if="!d.online" class="badge offline">{{ $t('devices.offline') }}</span>
              <span v-else-if="!d.logged_in" class="badge idle">{{ $t('devices.notLoggedIn') }}</span>
            </div>
          </div>
          <div class="device-actions">
            <button
              v-if="d.online && d.logged_in"
              class="btn btn-primary btn-sm"
              :disabled="connecting === d.device_id"
              @click="connectDevice(d.device_id, d.access_code)"
            >
              {{ connecting === d.device_id ? '...' : $t('devices.connect') }}
            </button>
            <button class="icon-btn" :title="$t('devices.setRemark')" @click="setRemark(d)">✏️</button>
            <button class="icon-btn danger" :title="$t('devices.unbind')" @click="unbindDevice(d.device_id)">🗑</button>
            <button
              class="icon-btn"
              :title="$t('devices.control.title')"
              :disabled="controlDialog.loading === d.device_id"
              @click="openControlDialog(d)"
            >🎛️</button>
          </div>
        </div>
      </div>

      <!-- My Favorites -->
      <div class="card">
        <div class="card-title">{{ $t('devices.myFavorites') }}</div>
        <div v-if="myFavorites.length === 0" class="empty-state">
          <div class="empty-icon">⭐</div>
          <p>{{ $t('devices.noFavorites') }}</p>
        </div>
        <div v-for="f in myFavorites" :key="f.device_id" class="device-item">
          <span style="font-size:16px;flex-shrink:0;">⭐</span>
          <div class="device-info">
            <div class="device-name">{{ f.device_name || f.device_id }}</div>
            <div class="device-id">{{ f.device_id }}</div>
          </div>
          <div class="device-actions">
            <button
              class="btn btn-primary btn-sm"
              :disabled="connecting === f.device_id"
              @click="connectDevice(f.device_id, f.access_password)"
            >
              {{ connecting === f.device_id ? '...' : $t('devices.connect') }}
            </button>
            <button class="icon-btn danger" :title="$t('devices.removeFavorite')" @click="removeFav(f.device_id)">✕</button>
          </div>
        </div>
      </div>

      <!-- Connection Logs -->
      <div class="card">
        <div class="card-title">{{ $t('devices.connectionLogs') }}</div>
        <div v-if="logs.length === 0" class="empty-state">
          <div class="empty-icon">📋</div>
          <p>{{ $t('devices.noLogs') }}</p>
        </div>
        <div v-for="log in logs.slice(0, 20)" :key="log.id" class="device-item">
          <div :class="['device-status', log.status === 'success' ? 'online' : 'offline']"></div>
          <div class="device-info">
            <div class="device-name">{{ log.device_id }}</div>
            <div class="device-id">{{ formatTime(log.created_at) }}{{ log.duration ? ` · ${Math.floor(log.duration/60)}m${log.duration%60}s` : '' }}</div>
          </div>
    </div>
      </div>
    </template>

    <!-- Remark Dialog -->
    <div v-if="remarkDialog.show" class="dialog-overlay" @click.self="remarkDialog.show = false">
      <div class="dialog">
        <div class="card-title">{{ $t('devices.setRemark') }}</div>
        <div class="form-group">
          <input v-model="remarkDialog.value" class="form-input" type="text" @keyup.enter="saveRemark" />
        </div>
        <div style="display:flex;gap:8px;">
          <button class="btn btn-primary" style="flex:1;" @click="saveRemark">{{ $t('account.save') }}</button>
          <button class="btn btn-secondary" style="flex:1;" @click="remarkDialog.show = false">{{ $t('user.cancel') }}</button>
        </div>
      </div>
    </div>

    <!-- Control Mode Dialog -->
    <div v-if="controlDialog.show" class="dialog-overlay" @click.self="closeControlDialog">
      <div class="dialog">
        <div class="card-title">{{ $t('devices.control.title') }}</div>
        <p class="hint">{{ $t('devices.control.hint') }}</p>

        <div v-if="controlDialog.loadError" class="form-error">{{ controlDialog.loadError }}</div>

        <div v-if="controlDialog.loading" class="empty-state">
          <div class="empty-icon">⏳</div>
          <p>{{ $t('devices.control.loading') }}</p>
        </div>

        <template v-else>
          <div class="form-group">
            <label class="control-mode-option">
              <input
                v-model="controlDialog.mode"
                type="radio"
                :value="CONTROL_MODE_SHARED"
                name="control-mode"
              />
              <span>
                <strong>{{ $t('devices.control.shared') }}</strong>
                <small>{{ $t('devices.control.sharedHint') }}</small>
              </span>
            </label>
            <label class="control-mode-option">
              <input
                v-model="controlDialog.mode"
                type="radio"
                :value="CONTROL_MODE_EXCLUSIVE"
                name="control-mode"
              />
              <span>
                <strong>{{ $t('devices.control.exclusive') }}</strong>
                <small>{{ $t('devices.control.exclusiveHint') }}</small>
              </span>
            </label>
          </div>

          <p v-if="controlDialog.saveError" class="form-error">{{ controlDialog.saveError }}</p>

          <div style="display:flex;gap:8px;">
            <button
              class="btn btn-primary"
              style="flex:1;"
              :disabled="controlDialog.saving || !controlModeChanged(controlDialog.storedMode, controlDialog.mode)"
              @click="saveControlMode"
            >
              {{ controlDialog.saving ? '...' : $t('account.save') }}
            </button>
            <button class="btn btn-secondary" style="flex:1;" @click="closeControlDialog">
              {{ $t('user.cancel') }}
            </button>
          </div>
        </template>
      </div>
     </div>

    <div v-if="groupDialog.show" class="dialog-overlay" @click.self="closeGroupDialog">
      <div class="dialog group-dialog">
        <div class="dialog-head">
          <div>
            <h3>{{ $t('devices.groups.title') }}</h3>
            <p>{{ $t('devices.groups.subtitle') }}</p>
          </div>
          <button
            class="icon-btn"
            :title="$t('devices.groups.close')"
            @click="closeGroupDialog"
          >✕</button>
        </div>

        <div class="group-dialog-body">
          <aside class="group-list-pane">
            <button class="btn btn-primary btn-sm group-create-button" @click="startNewGroup">
              {{ $t('devices.groups.create') }}
            </button>
            <div v-if="groupsLoading && deviceGroups.length === 0" class="group-pane-note">
              {{ $t('devices.groups.loading') }}
            </div>
            <div v-else-if="deviceGroups.length === 0" class="group-pane-note">
              {{ $t('devices.groups.empty') }}
            </div>
            <button
              v-for="group in deviceGroups"
              :key="group.id"
              :class="['group-list-row', { active: groupDialog.groupId === group.id }]"
              @click="selectGroupForEditing(group)"
            >
              <span
                class="group-color-dot"
                :style="{ backgroundColor: group.color || defaultGroupColor }"
              ></span>
              <span class="group-list-copy">
                <strong>{{ group.name }}</strong>
                <small>{{ $t('devices.groups.deviceCount', { count: boundDeviceCount(group) }) }}</small>
              </span>
            </button>
          </aside>

          <section class="group-editor">
            <div v-if="!groupDialog.groupId && !groupDialog.creating" class="empty-state compact">
              <div class="empty-icon">📂</div>
              <p>{{ $t('devices.groups.pickOrCreate') }}</p>
            </div>

            <template v-else>
              <div class="editor-heading">
                <div>
                  <h4>
                    {{ groupDialog.groupId
                      ? $t('devices.groups.editTitle')
                      : $t('devices.groups.createTitle') }}
                  </h4>
                  <p>{{ $t('devices.groups.detailsHint') }}</p>
                </div>
              </div>

              <div class="field-grid two group-fields">
                <label class="field">
                  <span>{{ $t('devices.groups.name') }}</span>
                  <input
                    v-model="groupDialog.name"
                    class="form-input"
                    type="text"
                    maxlength="100"
                    :placeholder="$t('devices.groups.namePlaceholder')"
                  />
                </label>
                <label class="field color-field">
                  <span>{{ $t('devices.groups.color') }}</span>
                  <div class="color-control">
                    <input
                      v-model="groupDialog.color"
                      class="color-input"
                      type="color"
                      :aria-label="$t('devices.groups.color')"
                    />
                    <input
                      v-model="groupDialog.color"
                      class="form-input color-text"
                      type="text"
                      maxlength="20"
                      placeholder="#e94560"
                    />
                  </div>
                </label>
              </div>

              <div class="color-swatches" :aria-label="$t('devices.groups.colorPresets')">
                <button
                  v-for="color in groupColors"
                  :key="color"
                  :class="['color-swatch', { active: sameColor(groupDialog.color, color) }]"
                  :style="{ backgroundColor: color }"
                  :title="color"
                  @click="groupDialog.color = color"
                ></button>
              </div>

              <label class="field">
                <span>{{ $t('devices.groups.description') }}</span>
                <textarea
                  v-model="groupDialog.description"
                  class="form-input group-description-input"
                  maxlength="500"
                  :placeholder="$t('devices.groups.descriptionPlaceholder')"
                ></textarea>
              </label>

              <div v-if="groupDialog.error" class="form-error">{{ groupDialog.error }}</div>
              <div class="form-actions">
                <button
                  class="btn btn-primary btn-sm"
                  :disabled="groupDialog.savingDetails"
                  @click="saveGroupDetails"
                >
                  {{ groupDialog.savingDetails
                    ? '...'
                    : (groupDialog.groupId
                      ? $t('devices.groups.saveDetails')
                      : $t('devices.groups.create')) }}
                </button>
                <button
                  v-if="groupDialog.groupId"
                  class="btn btn-danger btn-sm"
                  :disabled="groupDialog.deleting"
                  @click="deleteGroup"
                >
                  {{ groupDialog.deleting ? '...' : $t('devices.groups.delete') }}
                </button>
              </div>

              <div v-if="groupDialog.groupId" class="member-section">
                <div class="member-heading">
                  <div>
                    <h4>{{ $t('devices.groups.members') }}</h4>
                    <p>{{ $t('devices.groups.membersHint') }}</p>
                  </div>
                  <div class="member-quick-actions">
                    <button class="text-action" @click="selectAllGroupDevices">
                      {{ $t('devices.groups.selectAll') }}
                    </button>
                    <button class="text-action" @click="clearGroupDevices">
                      {{ $t('devices.groups.clear') }}
                    </button>
                  </div>
                </div>

                <div v-if="myDevices.length === 0" class="group-pane-note">
                  {{ $t('devices.groups.noBoundDevices') }}
                </div>
                <div v-else class="member-list">
                  <label
                    v-for="device in myDevices"
                    :key="device.device_id"
                    class="member-row"
                  >
                    <input
                      v-model="groupDialog.memberIds"
                      type="checkbox"
                      :value="device.device_id"
                    />
                    <span :class="['device-status', statusClass(device)]"></span>
                    <span class="member-copy">
                      <strong>{{ device.remark || device.device_name || device.device_id }}</strong>
                      <small>{{ device.device_id }}</small>
                    </span>
                  </label>
                </div>

                <div v-if="groupDialog.memberError" class="form-error">
                  {{ groupDialog.memberError }}
                </div>
                <div class="form-actions">
                  <button
                    class="btn btn-primary btn-sm"
                    :disabled="groupDialog.savingMembers"
                    @click="saveGroupMembers"
                  >
                    {{ groupDialog.savingMembers
                      ? '...'
                      : $t('devices.groups.saveMembers', {
                        count: normalizeGroupDeviceIds(groupDialog.memberIds).length,
                      }) }}
                  </button>
                </div>
              </div>
            </template>
          </section>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, reactive, inject, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { userApi } from '../api/userApi'
import { userSync } from '../api/userSync'
import { openRemoteSession } from '../utils/remoteLauncher'
import {
  buildDeviceGroupPayload,
  filterDevicesByGroup,
  normalizeDeviceGroupList,
  normalizeGroupDeviceIds,
} from '../api/deviceGroupModels'
import {
  CONTROL_MODE_SHARED,
  CONTROL_MODE_EXCLUSIVE,
  readControlMode,
  controlModeChanged,
  buildAccessPolicyBody,
} from '../api/controlModeModel'
import { deviceQuotaReached, deviceQuotaSummary } from '../../js/devices/quota'

const { t } = useI18n()
const showToast = inject('showToast')
const showLogin = inject('showLogin')
const authState = inject('authState')

const myDevices = ref([])
const myFavorites = ref([])
const logs = ref([])
const remarkDialog = ref({ show: false, device: null, value: '' })
const connecting = ref(null)
const loading = ref(false)
const deviceGroups = ref([])
const groupsLoading = ref(false)
const selectedGroupId = ref(null)
const defaultGroupColor = '#6b7280'
const groupColors = ['#e94560', '#3b82f6', '#10b981', '#f59e0b', '#a855f7', '#64748b']
const groupDialog = reactive({
  show: false,
  groupId: null,
  creating: false,
  name: '',
  description: '',
  color: groupColors[0],
  memberIds: [],
  savingDetails: false,
  savingMembers: false,
  deleting: false,
  error: '',
  memberError: '',
})
// QD-314: which controller may drive the machine. `loading` is keyed by device
// id so opening the dialog for one device does not blank another's row.
const controlDialog = reactive({
  show: false,
  deviceId: '',
  loading: '',
  mode: CONTROL_MODE_SHARED,
  storedMode: CONTROL_MODE_SHARED,
  loadError: '',
  saveError: '',
  saving: false,
})

const selectedGroup = computed(() => (
  deviceGroups.value.find((group) => group.id === selectedGroupId.value) || null
))
const visibleDevices = computed(() => (
  filterDevicesByGroup(myDevices.value, selectedGroup.value)
))

// QD-105: the server now refuses a bind once the account is at its device limit.
// The count is the loaded list, not a separate request: /v1/me already returns
// the quota, and the list is what the owner is looking at while deciding.
const quota = computed(() => deviceQuotaSummary(authState.userInfo, myDevices.value.length))
const quotaReached = computed(() => deviceQuotaReached(authState.userInfo, myDevices.value.length))

function boundDeviceCount(group) {
  return filterDevicesByGroup(myDevices.value, group).length
}

function sameColor(left, right) {
  return String(left || '').trim().toLowerCase() === String(right || '').trim().toLowerCase()
}

function statusClass(d) {
  if (d.online && d.logged_in) return 'online'
  if (d.online) return 'idle'
  return 'offline'
}

async function refresh() {
  if (loading.value) return
  loading.value = true
  try {
    await Promise.all([loadDevices(), loadFavorites(), loadLogs(), loadDeviceGroups()])
  } finally {
    loading.value = false
  }
}

// §2.2 / T2: list envelope is { items, next_cursor }.
async function loadDevices() {
  const r = await userApi.fetchMyDevices()
  if (r.ok && r.data) myDevices.value = Array.isArray(r.data.items) ? r.data.items : []
}

async function loadFavorites() {
  const r = await userApi.fetchFavorites()
  if (r.ok && r.data) myFavorites.value = Array.isArray(r.data.items) ? r.data.items : []
}

async function loadLogs() {
  const r = await userApi.fetchConnectionLogs()
  if (r.ok && r.data) logs.value = Array.isArray(r.data.items) ? r.data.items : []
}

async function loadDeviceGroups() {
  groupsLoading.value = true
  try {
    const r = await userApi.fetchDeviceGroups()
    if (!r.ok) {
      if (r.status !== 401) showToast(r.error || t('devices.groups.loadFailed'), 'error')
      return
    }
    deviceGroups.value = normalizeDeviceGroupList(r.data).items
    if (
      selectedGroupId.value !== null
      && !deviceGroups.value.some((group) => group.id === selectedGroupId.value)
    ) {
      selectedGroupId.value = null
    }
  } finally {
    groupsLoading.value = false
  }
}

function replaceDeviceGroup(saved) {
  const normalized = normalizeDeviceGroupList({ items: [saved] }).items[0]
  if (!normalized || !normalized.id) return null
  const index = deviceGroups.value.findIndex((group) => group.id === normalized.id)
  if (index >= 0) deviceGroups.value.splice(index, 1, normalized)
  else deviceGroups.value.push(normalized)
  deviceGroups.value.sort((left, right) => left.name.localeCompare(right.name))
  return normalized
}

function resetGroupDialog(group = null) {
  groupDialog.groupId = group?.id || null
  groupDialog.creating = !group
  groupDialog.name = group?.name || ''
  groupDialog.description = group?.description || ''
  groupDialog.color = group?.color || groupColors[0]
  groupDialog.memberIds = normalizeGroupDeviceIds(group?.device_ids)
  groupDialog.error = ''
  groupDialog.memberError = ''
  groupDialog.savingDetails = false
  groupDialog.savingMembers = false
  groupDialog.deleting = false
}

function openGroupDialog() {
  groupDialog.show = true
  const preselected = selectedGroup.value || deviceGroups.value[0] || null
  resetGroupDialog(preselected)
}

function closeGroupDialog() {
  groupDialog.show = false
}

function startNewGroup() {
  resetGroupDialog(null)
}

function selectGroupForEditing(group) {
  if (group) resetGroupDialog(group)
}

function editGroupMembers(group) {
  if (!group) return
  groupDialog.show = true
  resetGroupDialog(group)
}

async function saveGroupDetails() {
  const payload = buildDeviceGroupPayload(groupDialog)
  if (!payload.name) {
    groupDialog.error = t('devices.groups.nameRequired')
    return
  }
  const wasCreating = !groupDialog.groupId
  groupDialog.savingDetails = true
  groupDialog.error = ''
  try {
    const r = wasCreating
      ? await userApi.createDeviceGroup(payload)
      : await userApi.updateDeviceGroup(groupDialog.groupId, payload)
    if (!r.ok) {
      groupDialog.error = r.error || t('devices.groups.saveFailed')
      showToast(groupDialog.error, 'error')
      return
    }
    const saved = replaceDeviceGroup(r.data)
    if (!saved) {
      groupDialog.error = t('devices.groups.saveFailed')
      return
    }
    groupDialog.groupId = saved.id
    groupDialog.creating = false
    groupDialog.name = saved.name
    groupDialog.description = saved.description
    groupDialog.color = saved.color || groupColors[0]
    if (wasCreating) groupDialog.memberIds = saved.device_ids.slice()
    selectedGroupId.value = saved.id
    showToast(t(wasCreating ? 'devices.groups.created' : 'devices.groups.updated'), 'success')
  } finally {
    groupDialog.savingDetails = false
  }
}

async function saveGroupMembers() {
  const groupId = groupDialog.groupId
  if (!groupId) return
  const persisted = deviceGroups.value.find((group) => group.id === groupId) || null
  if (!persisted) {
    groupDialog.memberError = t('devices.groups.notFound')
    return
  }
  groupDialog.savingMembers = true
  groupDialog.memberError = ''
  try {
    const memberIds = normalizeGroupDeviceIds(groupDialog.memberIds)
    const r = await userApi.replaceDeviceGroupDevices(groupId, memberIds)
    if (!r.ok) {
      // The server is the membership authority. A failed PUT leaves the
      // persisted row untouched and returns the controls to its saved state.
      groupDialog.memberIds = persisted.device_ids.slice()
      groupDialog.memberError = r.error || t('devices.groups.membersSaveFailed')
      showToast(groupDialog.memberError, 'error')
      return
    }
    const saved = replaceDeviceGroup(r.data)
    if (!saved) {
      groupDialog.memberIds = persisted.device_ids.slice()
      groupDialog.memberError = t('devices.groups.membersSaveFailed')
      return
    }
    groupDialog.memberIds = saved.device_ids.slice()
    showToast(t('devices.groups.membersSaved'), 'success')
  } finally {
    groupDialog.savingMembers = false
  }
}

async function deleteGroup() {
  const group = deviceGroups.value.find((row) => row.id === groupDialog.groupId)
  if (!group) return
  if (!confirm(t('devices.groups.confirmDelete', { name: group.name }))) return
  groupDialog.deleting = true
  groupDialog.error = ''
  try {
    const r = await userApi.deleteDeviceGroup(group.id)
    if (!r.ok) {
      groupDialog.error = r.error || t('devices.groups.deleteFailed')
      showToast(groupDialog.error, 'error')
      return
    }
    deviceGroups.value = deviceGroups.value.filter((row) => row.id !== group.id)
    if (selectedGroupId.value === group.id) selectedGroupId.value = null
    showToast(t('devices.groups.deleted'), 'success')
    const next = deviceGroups.value[0] || null
    resetGroupDialog(next)
  } finally {
    groupDialog.deleting = false
  }
}

function selectAllGroupDevices() {
  groupDialog.memberIds = myDevices.value.map((device) => device.device_id)
}

function clearGroupDevices() {
  groupDialog.memberIds = []
}

// §2.18: verify → signal_token → open remote.html?st=<token>. The access
// code is passed to the remote window via sessionStorage (keyed by the
// short-lived signal_token) so it never enters the URL or browser history.
async function connectDevice(deviceId, accessCode) {
  if (!accessCode) { showToast(t('devices.noAccessCode'), 'error'); return }
  if (connecting.value) return
  connecting.value = deviceId
  try {
    const v = await userApi.verifyAccessCode(deviceId, accessCode)
    if (!v.ok) {
      // §2.10 / §2.15: on TOO_MANY_ATTEMPTS the server sets Retry-After
      // (seconds) so the UI can tell users how long to wait instead of
      // a vague "try again later".
      showToast(errorForCode(v.code, v.error, v.retryAfter), 'error')
      return
    }
    const signalToken = v.data?.signal_token
    if (!signalToken) { showToast(t('toast.networkError'), 'error'); return }
    openRemoteSession({ deviceId, signalToken, accessCode })
  } finally {
    connecting.value = null
  }
}

function errorForCode(code, fallback, retryAfter) {
  // §2.6 response code mapping to i18n keys (errors.* in locale files).
  // If the translation is missing fall back to the server-provided detail.
  if (code === 'TOO_MANY_ATTEMPTS' && retryAfter > 0) {
    return t('errors.TOO_MANY_ATTEMPTS_RETRY', { seconds: retryAfter })
  }
  if (code) {
    const key = `errors.${code}`
    const str = t(key)
    if (str !== key) return str
  }
  return fallback || t('toast.networkError')
}

function setRemark(device) {
  remarkDialog.value = { show: true, device, value: device.remark || device.device_name || '' }
}

// §2.2 / T5: PATCH for partial updates.
async function saveRemark() {
  const { device, value } = remarkDialog.value
  const r = await userApi.setDeviceRemark(device.device_id, value)
  if (!r.ok) {
    showToast(r.error || t('toast.networkError'), 'error')
    return
  }
  remarkDialog.value.show = false
  // Realtime `device.remark.changed` will refresh the list; fetch once as
  // belt-and-suspenders for snappy UI (§T8 refetch exception).
  await loadDevices()
}

async function unbindDevice(deviceId) {
  if (!confirm(t('devices.confirmUnbind'))) return
  const r = await userApi.unbindDevice(deviceId)
  if (!r.ok) { showToast(r.error || t('toast.networkError'), 'error'); return }
  showToast(t('devices.unbindSuccess'), 'success')
  // Realtime will emit device.unbound; patch locally for snappier feedback.
  myDevices.value = myDevices.value.filter(d => d.device_id !== deviceId)
  await loadDeviceGroups()
}

// QD-314: read the stored policy so the dialog opens on what the machine
// actually does, not on an assumption. The GET returns documented defaults when
// no row exists, so a device nobody configured still opens correctly.
async function openControlDialog(device) {
  const deviceId = device.device_id
  controlDialog.show = true
  controlDialog.deviceId = deviceId
  controlDialog.loading = deviceId
  controlDialog.loadError = ''
  controlDialog.saveError = ''
  controlDialog.saving = false

  const r = await userApi.fetchAccessPolicy(deviceId)
  controlDialog.loading = ''
  if (!r.ok) {
    controlDialog.loadError = r.error || t('toast.networkError')
    return
  }
  const mode = readControlMode(r.data)
  controlDialog.storedMode = mode
  controlDialog.mode = mode
}

function closeControlDialog() {
  controlDialog.show = false
  controlDialog.loading = ''
  controlDialog.saveError = ''
  controlDialog.loadError = ''
}

// The endpoint replaces the whole policy, so the body is the stored row with the
// one field the owner changed. Sending only `control_mode` would reset the IP
// lists, the session cap, the windows and the approval settings in the same
// request - which is exactly the failure mode that makes a one-field form
// dangerous on a full-replace endpoint.
async function saveControlMode() {
  const deviceId = controlDialog.deviceId
  if (!deviceId) return
  controlDialog.saving = true
  controlDialog.saveError = ''

  const current = await userApi.fetchAccessPolicy(deviceId)
  if (!current.ok) {
    controlDialog.saving = false
    controlDialog.saveError = current.error || t('toast.networkError')
    return
  }

  const r = await userApi.saveAccessPolicy(
    deviceId,
    buildAccessPolicyBody(current.data, controlDialog.mode),
  )
  controlDialog.saving = false
  if (!r.ok) {
    controlDialog.saveError = r.error || t('toast.networkError')
    return
  }
  controlDialog.storedMode = readControlMode(r.data)
  showToast(t('devices.control.saved'), 'success')
  closeControlDialog()
}

async function removeFav(deviceId) {
  await userApi.removeFavorite(deviceId)
  // Realtime favorite.removed will patch the cached list; mirror locally.
  myFavorites.value = myFavorites.value.filter(f => f.device_id !== deviceId)
}

function formatTime(ts) {
  if (!ts) return ''
  return new Date(ts).toLocaleString()
}

// Realtime handlers — prefer userSync's in-memory cache which the sync
// layer already patched (§T8: do NOT re-fetch on ordinary updates).
function onSyncSnapshot(e) {
  myDevices.value   = e.detail.devices.slice()
  myFavorites.value = e.detail.favorites.slice()
}
function onDevicesChanged(e) {
  myDevices.value = e.detail.devices.slice()
}
function onFavoritesChanged(e) {
  myFavorites.value = e.detail.favorites.slice()
}

onMounted(() => {
  if (authState.isLoggedIn) {
    // Seed from userSync cache (non-empty if sync already ran).
    const cachedD = userSync.getDevices()
    const cachedF = userSync.getFavorites()
    if (cachedD.length) myDevices.value = cachedD
    if (cachedF.length) myFavorites.value = cachedF
    // Logs come from HTTP — sync stream doesn't cover connection history.
    loadLogs()
    // Also fetch once if cache empty; otherwise wait for snapshot.
    if (!cachedD.length) loadDevices()
    if (!cachedF.length) loadFavorites()
    loadDeviceGroups()
  }
  userSync.addEventListener('snapshot', onSyncSnapshot)
  userSync.addEventListener('devices-changed', onDevicesChanged)
  userSync.addEventListener('favorites-changed', onFavoritesChanged)
})

onBeforeUnmount(() => {
  userSync.removeEventListener('snapshot', onSyncSnapshot)
  userSync.removeEventListener('devices-changed', onDevicesChanged)
  userSync.removeEventListener('favorites-changed', onFavoritesChanged)
})
</script>

<style scoped>
.device-item {
  display: flex; align-items: center; gap: 12px;
  padding: 12px; border-radius: var(--radius);
  transition: background 0.15s;
}
.device-item:hover { background: rgba(255,255,255,0.05); }
.device-status { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }
.device-status.online { background: var(--success); }
.device-status.idle   { background: var(--warning); }
.device-status.offline { background: var(--text-disabled); }
.device-info { flex: 1; min-width: 0; }
.device-name { font-size: 14px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.device-id { font-size: 12px; color: var(--text-secondary); font-family: 'Consolas', monospace; margin-top: 2px; }
.badge { margin-left: 6px; padding: 1px 6px; border-radius: 4px; font-size: 11px; font-family: inherit; }
.badge.offline { background: rgba(255,255,255,0.1); color: var(--text-secondary); }
.badge.idle    { background: rgba(255,193,7,0.15); color: var(--warning); }
.device-actions { display: flex; gap: 4px; flex-shrink: 0; }
.icon-btn { width: 28px; height: 28px; border: none; border-radius: 6px; background: transparent; color: var(--text-secondary); cursor: pointer; display: flex; align-items: center; justify-content: center; font-size: 14px; transition: all 0.15s; }
.icon-btn:hover { background: rgba(255,255,255,0.1); color: var(--text-primary); }
.icon-btn.danger:hover { background: rgba(244,67,54,0.2); color: var(--error); }

.dialog-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); z-index: 1000; display: flex; align-items: center; justify-content: center; }
.dialog { background: var(--bg-card); border: 1px solid var(--border); border-radius: 12px; padding: 24px; width: 320px; max-width: 90vw; }

.devices-toolbar {
  display: flex; align-items: flex-start; justify-content: space-between;
  gap: 12px; margin-bottom: 12px; flex-wrap: wrap;
}
.group-filter { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; min-width: 0; }
.group-filter-chip {
  display: inline-flex; align-items: center; gap: 6px;
  max-width: 220px; padding: 6px 10px;
  border: 1px solid var(--border); border-radius: 999px;
  background: rgba(255,255,255,0.04); color: var(--text-secondary);
  font-size: 12px; cursor: pointer; transition: all 0.15s;
}
.group-filter-chip:hover { color: var(--text-primary); border-color: rgba(255,255,255,0.2); }
.group-filter-chip.active {
  color: var(--text-primary); border-color: rgba(233,69,96,0.65);
  background: rgba(233,69,96,0.13);
}
.group-filter-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.group-filter-count {
  min-width: 18px; padding: 1px 5px; border-radius: 999px;
  background: rgba(255,255,255,0.08); text-align: center; font-size: 11px;
}
.group-color-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.toolbar-actions { display: flex; gap: 8px; flex-shrink: 0; }
.card-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
.card-heading .card-title { margin-bottom: 4px; }
.card-heading .card-subtitle { margin-bottom: 12px; }
.group-description {
  max-width: 640px; overflow-wrap: anywhere;
}
.device-count { color: var(--text-secondary); font-size: 12px; white-space: nowrap; }
.device-quota-warning {
  color: var(--warning, #f59e0b);
  font-size: 12px;
  white-space: nowrap;
}
.empty-action { margin-top: 12px; }

.group-dialog {
  width: min(920px, calc(100vw - 32px));
  max-width: none; max-height: calc(100vh - 32px);
  padding: 0; overflow: hidden;
  display: flex; flex-direction: column;
}
.dialog-head {
  display: flex; align-items: flex-start; justify-content: space-between;
  gap: 16px; padding: 20px 22px; border-bottom: 1px solid var(--border);
}
.dialog-head h3 { font-size: 18px; margin-bottom: 4px; }
.dialog-head p { color: var(--text-secondary); font-size: 12px; line-height: 1.5; }
.group-dialog-body {
  display: grid; grid-template-columns: minmax(190px, 240px) minmax(0, 1fr);
  min-height: 0; overflow: auto;
}
.group-list-pane {
  padding: 16px; border-right: 1px solid var(--border);
  background: rgba(0,0,0,0.08); min-width: 0;
}
.group-create-button { width: 100%; margin-bottom: 12px; }
.group-pane-note {
  padding: 14px 4px; color: var(--text-secondary);
  font-size: 12px; line-height: 1.5; overflow-wrap: anywhere;
}
.group-list-row {
  width: 100%; display: flex; align-items: center; gap: 9px;
  padding: 10px; margin-bottom: 4px; border: 1px solid transparent;
  border-radius: 8px; background: transparent; color: var(--text-secondary);
  text-align: left; cursor: pointer;
}
.group-list-row:hover { background: rgba(255,255,255,0.05); color: var(--text-primary); }
.group-list-row.active {
  color: var(--text-primary); border-color: var(--border);
  background: rgba(255,255,255,0.07);
}
.group-list-copy { display: flex; flex-direction: column; min-width: 0; flex: 1; }
.group-list-copy strong { font-size: 13px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.group-list-copy small { color: var(--text-secondary); font-size: 11px; margin-top: 2px; }
.group-editor { padding: 20px 22px; min-width: 0; }
.editor-heading h4, .member-heading h4 { font-size: 15px; margin-bottom: 4px; }
.editor-heading p, .member-heading p {
  color: var(--text-secondary); font-size: 12px; line-height: 1.5;
}
.field-grid { display: grid; gap: 12px; }
.field-grid.two { grid-template-columns: repeat(2, minmax(0, 1fr)); }
.field { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.field > span { color: var(--text-secondary); font-size: 12px; }
.group-fields { margin-top: 16px; }
.color-control { display: flex; gap: 8px; min-width: 0; }
.color-input {
  width: 48px; height: 40px; padding: 3px;
  border: 1px solid var(--border); border-radius: var(--radius);
  background: var(--bg-input); flex-shrink: 0; cursor: pointer;
}
.color-text { min-width: 0; }
.color-swatches { display: flex; gap: 8px; flex-wrap: wrap; margin: 10px 0 16px; }
.color-swatch {
  width: 24px; height: 24px; padding: 0; border-radius: 50%;
  border: 2px solid transparent; cursor: pointer;
  box-shadow: 0 0 0 1px rgba(255,255,255,0.2);
}
.color-swatch.active { border-color: white; box-shadow: 0 0 0 2px var(--accent); }
.group-description-input { min-height: 78px; resize: vertical; font-family: inherit; }
.form-actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; margin-top: 12px; }
.form-error {
  margin-top: 10px; padding: 8px 10px; border-radius: 6px;
  background: rgba(244,67,54,0.12); color: #ff9a93;
  font-size: 12px; line-height: 1.5; overflow-wrap: anywhere;
}
/* QD-314: the two control modes read as one choice, so the radio and its copy
   stay on one line and the whole row is the hit target. */
.control-mode-option {
  display: flex; align-items: flex-start; gap: 10px;
  padding: 10px 12px; margin-bottom: 8px;
  border: 1px solid var(--border); border-radius: 8px;
  cursor: pointer;
}
.control-mode-option:hover { border-color: var(--accent, #e94560); }
.control-mode-option input { margin-top: 3px; flex-shrink: 0; }
.control-mode-option span { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.control-mode-option strong { font-size: 13px; color: var(--text-primary); }
.control-mode-option small { font-size: 11px; color: var(--text-secondary); line-height: 1.5; }
.member-section {
  margin-top: 22px; padding-top: 18px; border-top: 1px solid var(--border);
}
.member-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
.member-quick-actions { display: flex; gap: 10px; flex-shrink: 0; }
.text-action {
  border: none; background: transparent; color: var(--accent);
  font-size: 12px; cursor: pointer; padding: 2px;
}
.text-action:hover { color: var(--accent-hover); }
.member-list {
  display: grid; grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
  gap: 8px; margin-top: 12px;
}
.member-row {
  display: flex; align-items: center; gap: 8px;
  padding: 9px 10px; border: 1px solid var(--border);
  border-radius: 8px; background: rgba(255,255,255,0.025);
  min-width: 0; cursor: pointer;
}
.member-row:hover { background: rgba(255,255,255,0.05); }
.member-row input { accent-color: var(--accent); flex-shrink: 0; }
.member-copy { display: flex; flex-direction: column; min-width: 0; }
.member-copy strong {
  font-size: 12px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.member-copy small {
  color: var(--text-secondary); font-size: 10px; margin-top: 2px;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}

@media (max-width: 700px) {
  .devices-toolbar { align-items: stretch; }
  .group-filter { width: 100%; flex-wrap: nowrap; overflow-x: auto; padding-bottom: 2px; }
  .group-filter-chip { flex-shrink: 0; max-width: 190px; }
  .toolbar-actions { width: 100%; justify-content: flex-end; }
  .group-dialog-body { grid-template-columns: minmax(0, 1fr); }
  .group-list-pane {
    border-right: none; border-bottom: 1px solid var(--border);
    max-height: 210px; overflow-y: auto;
  }
  .group-editor { padding: 18px 16px; }
  .field-grid.two { grid-template-columns: minmax(0, 1fr); }
  .member-heading { flex-direction: column; }
  .member-list { grid-template-columns: minmax(0, 1fr); }
  .device-item { align-items: flex-start; }
  .device-actions { flex-wrap: wrap; justify-content: flex-end; }
}
</style>
