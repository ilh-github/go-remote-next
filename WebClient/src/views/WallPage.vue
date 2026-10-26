<template>
  <div class="wall-page">
    <div class="wall-head">
      <h2 class="page-title">{{ $t('wall.title') }}</h2>
      <div class="wall-tools">
        <span class="wall-count">{{ $t('wall.onWall', { count: tiles.length }) }}</span>
        <button
          v-if="tiles.length > 1"
          class="btn btn-secondary btn-sm"
          :class="{ primary: rotating }"
          :disabled="!!armedId"
          :title="armedId ? $t('wall.rotateArmedHint') : ''"
          @click="toggleRotate"
        >{{ rotating ? $t('wall.rotateOff') : $t('wall.rotateOn') }}</button>
        <label v-if="rotating" class="wall-cols">
          <span>{{ $t('wall.rotateEvery') }}</span>
          <select :value="rotateSeconds" @change="onRotateSecondsChange($event.target.value)">
            <option v-for="n in [5, 15, 30, 60]" :key="n" :value="n">{{ n }}s</option>
          </select>
        </label>
        <label class="wall-cols">
          <span>{{ $t('wall.columns') }}</span>
          <select :value="columns" @change="onColumnsChange($event.target.value)">
            <option v-for="n in [1, 2, 3, 4]" :key="n" :value="n">{{ n }}</option>
          </select>
        </label>
        <button class="btn btn-secondary btn-sm" @click="refresh">
          🔄 {{ $t('wall.refresh') }}
        </button>
        <button class="btn btn-secondary btn-sm" @click="reloadAll">
          ⟳ {{ $t('wall.reloadAll') }}
        </button>
        <button class="btn btn-secondary btn-sm" @click="pickerOpen = !pickerOpen">
          ＋ {{ $t('wall.addDevice') }}
        </button>
        <button v-if="tiles.length" class="btn btn-secondary btn-sm danger" @click="removeAll">
          ✕ {{ $t('wall.removeAll') }}
        </button>
      </div>
    </div>

    <div v-if="!authState.isLoggedIn" class="card">
      <div class="empty-state">
        <div class="empty-icon">🔒</div>
        <p>{{ $t('wall.loginRequired') }}</p>
        <button class="btn btn-primary btn-sm" style="margin-top:12px;" @click="showLogin()">
          {{ $t('user.login') }}
        </button>
      </div>
    </div>

    <template v-else>
      <!-- Device picker -->
      <div v-if="pickerOpen" class="card picker">
        <div class="card-title">{{ $t('wall.pickDevices') }}</div>
        <div v-if="devices.length === 0" class="empty-state">
          <div class="empty-icon">📱</div>
          <p>{{ $t('wall.noDevices') }}</p>
        </div>
        <div v-for="d in devices" :key="d.device_id" class="picker-row">
          <label class="picker-label">
            <input
              type="checkbox"
              :checked="selected.includes(d.device_id)"
              @change="toggleDevice(d.device_id)"
            />
            <span :class="['picker-dot', d.online && d.logged_in ? 'ok' : 'off']"></span>
            <span class="picker-name">{{ d.remark || d.device_name || d.device_id }}</span>
            <span class="picker-id">{{ d.device_id }}</span>
            <span v-if="!d.online" class="badge">{{ $t('wall.state.offline') }}</span>
            <span v-else-if="!d.logged_in" class="badge">{{ $t('wall.state.idle') }}</span>
          </label>
        </div>
      </div>

      <!-- Empty wall -->
      <div v-if="tiles.length === 0" class="card">
        <div class="empty-state">
          <div class="empty-icon">🧱</div>
          <p>{{ $t('wall.empty') }}</p>
          <p class="hint">{{ $t('wall.emptyHint') }}</p>
          <button class="btn btn-primary btn-sm" style="margin-top:12px;" @click="pickerOpen = true">
            ＋ {{ $t('wall.addDevice') }}
          </button>
        </div>
      </div>

      <!-- The wall itself -->
      <div v-else class="wall-grid" :style="{ '--wall-cols': String(columns) }">
        <div
          v-for="tile in tiles"
          :key="tile.deviceId"
          :class="[
            'wall-tile',
            {
              armed: armedId === tile.deviceId,
              focused: focusedId === tile.deviceId,
              audio: audioId === tile.deviceId,
            },
          ]"
        >
          <div class="tile-head">
            <span :class="['tile-dot', tile.state]"></span>
            <span class="tile-name" :title="tile.deviceId">{{ tile.name }}</span>
            <span :class="['tile-badge', tile.state]">{{ stateLabel(tile.state) }}</span>
            <span v-if="armedId === tile.deviceId" class="tile-armed">{{ $t('wall.armed') }}</span>
            <span v-if="tile.detail" class="tile-detail">{{ tile.detail }}</span>
            <button class="tile-btn" :title="$t('wall.reload')" @click="reloadTile(tile)">⟳</button>
            <button
              class="tile-btn"
              :title="focusedId === tile.deviceId ? $t('wall.unfocus') : $t('wall.focus')"
              @click="toggleFocus(tile)"
            >⤢</button>
            <button
              class="tile-btn"
              :title="armedId === tile.deviceId ? $t('wall.disarmInput') : $t('wall.armInput')"
              :disabled="!canArm(tile)"
              @click="toggleInput(tile)"
            >{{ armedId === tile.deviceId ? '🔓' : '🔒' }}</button>
            <button
              class="tile-btn"
              :class="{ 'audio-on': audioId === tile.deviceId }"
              :title="audioId === tile.deviceId ? $t('wall.audioOff') : $t('wall.audioOn')"
              :disabled="!canArm(tile)"
              @click="toggleAudio(tile)"
            >{{ audioId === tile.deviceId ? '🔊' : '🔈' }}</button>
            <button
              class="tile-btn"
              :title="$t('wall.openWindow')"
              :disabled="tile.state === 'offline' || tile.state === 'idle'"
              @click="openTileWindow(tile)"
            >↗</button>
            <button class="tile-btn danger" :title="$t('wall.remove')" @click="removeTile(tile)">✕</button>
          </div>
          <div class="tile-body">
            <iframe
              v-if="tile.url"
              :ref="(el) => setFrameRef(tile.deviceId, el)"
              :key="tile.url"
              :src="tile.url"
              :title="tile.name"
              allow="autoplay"
            ></iframe>
            <div v-else class="tile-placeholder">
              <div class="tile-placeholder-icon">{{ placeholderIcon(tile.state) }}</div>
              <p>{{ tile.reason || stateLabel(tile.state) }}</p>
              <button
                v-if="tile.state !== 'idle'"
                class="btn btn-secondary btn-sm"
                @click="reloadTile(tile)"
              >⟳ {{ $t('wall.reload') }}</button>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { userApi } from '../api/userApi'
import { openRemoteSession, prepareRemoteSession } from '../utils/remoteLauncher'
import { WallAudioSelector } from '../../js/ui/wall-audio.js'
import { resolveTiles } from '../../js/wall/tileModel.js'
import { mergeLiveState } from '../../js/wall/liveState.js'
import { makeStateLabel } from '../../js/wall/stateLabel.js'

const { t, locale } = useI18n()
const showToast = inject('showToast')
const showLogin = inject('showLogin')
const authState = inject('authState')

const WALL_DEVICES_KEY = 'quickdesk_wall_devices'
const WALL_COLUMNS_KEY = 'quickdesk_wall_columns'

const devices = ref([])
const selected = ref(readSelection())
const pickerOpen = ref(false)
const columns = ref(readColumns())
// Only one tile may hold input at a time: arming a second one locks the first.
const armedId = ref('')
const audioSelector = new WallAudioSelector()
const audioId = ref(audioSelector.activeId)
// The focused tile spans the whole row; rotation walks that focus through the
// wall so a control room can follow several machines on one screen.
const focusedId = ref('')
const rotating = ref(false)
const WALL_ROTATE_KEY = 'quickdesk_wall_rotate_seconds'
const rotateSeconds = ref(readRotateSeconds())
let rotateTimer = null
const frameRefs = new Map()
// Per-device live state. The tile iframe is the only thing that knows whether
// the session really came up, so it reports back over postMessage.
const live = reactive({})

// Resolved through js/wall/tileModel.js rather than inline. The inline version
// did `devices.value.find(...)` inside the map over the selection, which is
// quadratic in the number of tiles - and a wall is a grid of hundreds of tiles by
// design, so that was the common case, not an edge. The measured 200-viewer peak
// sat far above what 100 viewers produced, which is where this started.
const tiles = computed(() => resolveTiles({
  selectedIds: selected.value,
  devices: devices.value,
  live,
  translate: t,
}))

function readSelection() {
  try {
    const raw = JSON.parse(localStorage.getItem(WALL_DEVICES_KEY) || '[]')
    return Array.isArray(raw) ? raw.filter((v) => typeof v === 'string') : []
  } catch { return [] }
}

function readColumns() {
  const n = Number(localStorage.getItem(WALL_COLUMNS_KEY) || 2)
  return [1, 2, 3, 4].includes(n) ? n : 2
}

function readRotateSeconds() {
  // localStorage is also the test hook: the harness sets a short interval so a
  // rotation can be observed without waiting 15 seconds per step.
  const n = Number(localStorage.getItem(WALL_ROTATE_KEY) || 15)
  return Number.isFinite(n) && n >= 1 ? n : 15
}

function persistSelection() {
  try { localStorage.setItem(WALL_DEVICES_KEY, JSON.stringify(selected.value)) } catch { /* private mode */ }
}

function onColumnsChange(value) {
  columns.value = Number(value) || 2
  try { localStorage.setItem(WALL_COLUMNS_KEY, String(columns.value)) } catch { /* private mode */ }
}

// Memoised per state, and rebuilt when the locale changes - depending on
// `locale` is what makes a language switch rebuild it rather than keep serving
// the old language out of the cache. Called from a template expression on every
// tile badge of every render, which is what put vue-i18n's message compiler at
// the top of this page's allocation profile.
const stateLabel = computed(() => {
  void locale.value
  return makeStateLabel(t)
})

// The session reports both "connected" (transport up) and "live" (frames
// flowing), and the later of the two is not guaranteed; either means the tile
// has something to attach input handlers to.
function canArm(tile) {
  return tile.state === 'live' || tile.state === 'connected'
}

function placeholderIcon(state) {
  if (state === 'offline') return '📴'
  if (state === 'idle') return '💤'
  if (state === 'error') return '⚠️'
  return '⏳'
}

async function loadDevices() {
  const r = await userApi.fetchMyDevices()
  if (r.ok && r.data) {
    devices.value = Array.isArray(r.data.items) ? r.data.items : []
    if (audioId.value) {
      const owner = devices.value.find((d) => d.device_id === audioId.value)
      if (!owner || !owner.online || !owner.logged_in) disableAudio()
    }
  }
}

// Each tile needs its own one-shot signal token, so the wall performs the same
// verify→token→handoff dance the devices page does, once per online device.
async function mintTile(deviceId) {
  const device = devices.value.find((d) => d.device_id === deviceId)
  if (!device || !device.online || !device.logged_in) return
  if (!device.access_code) {
    live[deviceId] = { ...(live[deviceId] || {}), state: 'error', reason: t('wall.reasonNoAccessCode'), url: '' }
    return
  }
  live[deviceId] = { ...(live[deviceId] || {}), state: 'connecting', detail: '', reason: '' }
  const v = await userApi.verifyAccessCode(deviceId, device.access_code)
  if (!v.ok || !v.data?.signal_token) {
    live[deviceId] = { state: 'error', reason: v.error || t('wall.tileLoadFailed'), url: '' }
    return
  }
  const url = prepareRemoteSession({
    deviceId,
    signalToken: v.data.signal_token,
    accessCode: device.access_code,
    extraParams: { wall: '1' },
  })
  live[deviceId] = { state: 'connecting', detail: '', reason: '', url }
}

async function buildTiles() {
  await Promise.all(selected.value.map((id) => mintTile(id)))
}

async function refresh() {
  await loadDevices()
  await buildTiles()
}

function toggleDevice(deviceId) {
  if (selected.value.includes(deviceId)) {
    removeDevice(deviceId)
    return
  }
  selected.value = selected.value.concat(deviceId)
  persistSelection()
  mintTile(deviceId)
}

function removeDevice(deviceId) {
  if (audioId.value === deviceId) disableAudio(deviceId)
  if (armedId.value === deviceId) armedId.value = ''
  if (focusedId.value === deviceId) focusedId.value = ''
  selected.value = selected.value.filter((id) => id !== deviceId)
  persistSelection()
  delete live[deviceId]
}

function removeTile(tile) {
  removeDevice(tile.deviceId)
}

function removeAll() {
  disableAudio()
  armedId.value = ''
  focusedId.value = ''
  stopRotate()
  selected.value = []
  persistSelection()
  for (const key of Object.keys(live)) delete live[key]
}

async function reloadTile(tile) {
  if (audioId.value === tile.deviceId) disableAudio(tile.deviceId)
  if (armedId.value === tile.deviceId) armedId.value = ''
  await mintTile(tile.deviceId)
}

async function reloadAll() {
  disableAudio()
  armedId.value = ''
  await Promise.all(selected.value.map((id) => mintTile(id)))
}

function setFrameRef(deviceId, el) {
  if (el) frameRefs.set(deviceId, el)
  else frameRefs.delete(deviceId)
}

// Input arming is a message to the tile, not a wall-side setting: the tile owns
// whether it has input handlers at all.
function sendControl(deviceId, armed) {
  const el = frameRefs.get(deviceId)
  if (!el || !el.contentWindow) return
  el.contentWindow.postMessage(
    { type: 'quickdesk-wall-control', device: deviceId, armed },
    window.location.origin,
  )
}

function sendAudio(deviceId, enabled) {
  const el = frameRefs.get(deviceId)
  if (!el || !el.contentWindow) return
  el.contentWindow.postMessage(
    { type: 'quickdesk-wall-audio', device: deviceId, enabled },
    window.location.origin,
  )
}

function disableAudio(deviceId = '') {
  const previous = audioSelector.clear(deviceId)
  audioId.value = audioSelector.activeId
  if (previous) sendAudio(previous, false)
}

function toggleAudio(tile) {
  if (!canArm(tile)) return
  const next = audioId.value === tile.deviceId ? '' : tile.deviceId
  const previous = audioSelector.setActive(next)
  audioId.value = audioSelector.activeId
  if (previous && previous !== next) sendAudio(previous, false)
  if (next) sendAudio(next, true)
}

function toggleInput(tile) {
  const next = armedId.value === tile.deviceId ? '' : tile.deviceId
  const previous = armedId.value
  armedId.value = next
  // A rotating wall must never rotate away from a tile that holds input.
  if (next) stopRotate()
  if (previous && previous !== next) sendControl(previous, false)
  if (next) sendControl(next, true)
}

function toggleFocus(tile) {
  focusedId.value = focusedId.value === tile.deviceId ? '' : tile.deviceId
}

function stopRotate() {
  if (rotateTimer) {
    clearInterval(rotateTimer)
    rotateTimer = null
  }
  rotating.value = false
}

function toggleRotate() {
  if (rotating.value) {
    stopRotate()
    return
  }
  if (armedId.value || selected.value.length < 2) return
  if (!focusedId.value) focusedId.value = selected.value[0]
  rotating.value = true
  rotateTimer = setInterval(() => {
    const ids = selected.value
    if (ids.length < 2) return
    const current = ids.indexOf(focusedId.value)
    focusedId.value = ids[(current + 1) % ids.length]
  }, Math.max(1, rotateSeconds.value) * 1000)
}

function onRotateSecondsChange(value) {
  const n = Number(value)
  rotateSeconds.value = Number.isFinite(n) && n >= 1 ? n : 15
  try { localStorage.setItem(WALL_ROTATE_KEY, String(rotateSeconds.value)) } catch { /* private mode */ }
  if (rotating.value) {
    stopRotate()
    toggleRotate()
  }
}

// A wall tile cannot re-use its url: the signal token was already consumed, so
// opening the device for real means minting a fresh one.
async function openTileWindow(tile) {
  const device = devices.value.find((d) => d.device_id === tile.deviceId)
  if (!device || !device.online || !device.logged_in) return
  if (!device.access_code) { showToast(t('wall.reasonNoAccessCode'), 'error'); return }
  const v = await userApi.verifyAccessCode(tile.deviceId, device.access_code)
  if (!v.ok || !v.data?.signal_token) {
    showToast(v.error || t('wall.tileLoadFailed'), 'error')
    return
  }
  openRemoteSession({
    deviceId: tile.deviceId,
    signalToken: v.data.signal_token,
    accessCode: device.access_code,
  })
}

function onWallMessage(event) {
  if (event.origin !== window.location.origin) return
  const data = event.data
  if (!data || data.type !== 'quickdesk-wall' || !data.device) return
  if (!selected.value.includes(data.device)) return
  // Merged rather than replaced. Every iframe posts its frame counter twice a
  // second, and replacing the entry made each of those a re-render of the tile -
  // which re-runs the eight `$t()` bindings in the tile template. The 40-viewer
  // soak measured that as 29.6% of all sampled allocation in vue-i18n's message
  // compiler, on a 2008ms period that matches the reporting interval rather than
  // the frame loop.
  mergeLiveState(live, data, selected.value)
  // Re-apply the armed state once the tile actually has a session to attach
  // handlers to, and drop it if the session died.
  if (data.state === 'live' && armedId.value === data.device) {
    sendControl(data.device, true)
  }
  if (data.state === 'live' && audioId.value === data.device) {
    sendAudio(data.device, true)
  }
  if (data.state === 'error') {
    if (armedId.value === data.device) armedId.value = ''
    if (audioId.value === data.device) disableAudio(data.device)
  }
}

let refreshTimer = null

onMounted(async () => {
  window.addEventListener('message', onWallMessage)
  if (authState.isLoggedIn) {
    await refresh()
    // Online state changes on the host side; a slow poll keeps the badges honest
    // without re-minting tokens for tiles that are already up.
    refreshTimer = setInterval(loadDevices, 30000)
  }
})

onUnmounted(() => {
  window.removeEventListener('message', onWallMessage)
  if (refreshTimer) clearInterval(refreshTimer)
  disableAudio()
  stopRotate()
})
</script>

<style scoped>
.wall-page { padding-bottom: 24px; }
.wall-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
.wall-tools { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.wall-count { font-size: 12px; color: var(--text-secondary); }
.wall-cols { display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--text-secondary); }
.wall-cols select {
  background: rgba(255,255,255,0.06); color: var(--text-primary);
  border: 1px solid rgba(255,255,255,0.12); border-radius: 6px; padding: 3px 6px;
}
.picker { margin-bottom: 12px; }
.picker-row { padding: 4px 0; }
.picker-label { display: flex; align-items: center; gap: 8px; cursor: pointer; font-size: 13px; }
.picker-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.picker-dot.ok { background: var(--success); }
.picker-dot.off { background: var(--text-disabled); }
.picker-name { min-width: 140px; }
.picker-id { color: var(--text-secondary); font-family: 'Consolas', monospace; font-size: 12px; }
.badge { padding: 1px 6px; border-radius: 4px; font-size: 11px; background: rgba(255,255,255,0.1); color: var(--text-secondary); }

.wall-grid {
  display: grid;
  grid-template-columns: repeat(var(--wall-cols, 2), minmax(0, 1fr));
  gap: 12px;
}
.wall-tile {
  background: rgba(255,255,255,0.04);
  border: 1px solid rgba(255,255,255,0.08);
  border-radius: var(--radius);
  overflow: hidden;
  display: flex; flex-direction: column;
}
.tile-head { display: flex; align-items: center; gap: 8px; padding: 6px 8px; font-size: 12px; }
.tile-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--text-disabled); flex-shrink: 0; }
.tile-dot.live { background: var(--success); }
.tile-dot.connected { background: var(--accent); }
.tile-dot.connecting { background: var(--accent); }
.tile-dot.error { background: var(--error); }
.tile-name { font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.tile-badge { padding: 1px 6px; border-radius: 4px; font-size: 11px; background: rgba(255,255,255,0.08); color: var(--text-secondary); }
.tile-badge.live { background: rgba(76,175,80,0.15); color: var(--success); }
.tile-badge.error { background: rgba(244,67,54,0.15); color: var(--error); }
.tile-detail { color: var(--text-secondary); font-family: 'Consolas', monospace; }
.tile-btn {
  margin-left: auto; border: none; background: transparent; color: var(--text-secondary);
  cursor: pointer; font-size: 13px; padding: 2px 4px; border-radius: 4px;
}
.tile-btn + .tile-btn { margin-left: 0; }
.tile-btn:hover:not(:disabled) { background: rgba(255,255,255,0.1); color: var(--text-primary); }
.tile-btn.audio-on { background: rgba(33,150,243,0.18); color: var(--accent); }
.tile-btn:disabled { opacity: 0.3; cursor: not-allowed; }
.tile-btn.danger:hover { background: rgba(244,67,54,0.2); color: var(--error); }
.wall-tile.armed { border-color: var(--error); box-shadow: 0 0 0 1px rgba(244,67,54,0.35); }
.wall-tile.audio:not(.armed) { border-color: var(--accent); }
.wall-tile.focused { grid-column: 1 / -1; }
.wall-tile.focused .tile-body { aspect-ratio: auto; height: min(70vh, 1000px); }
.tile-armed { padding: 1px 6px; border-radius: 4px; font-size: 11px; background: rgba(244,67,54,0.18); color: var(--error); }
.tile-body { position: relative; aspect-ratio: 16 / 9; background: #000; }
.tile-body iframe { width: 100%; height: 100%; border: 0; display: block; }
.tile-placeholder {
  position: absolute; inset: 0; display: flex; flex-direction: column;
  align-items: center; justify-content: center; gap: 8px;
  color: var(--text-secondary); font-size: 12px; text-align: center; padding: 12px;
}
.tile-placeholder-icon { font-size: 22px; }
</style>
