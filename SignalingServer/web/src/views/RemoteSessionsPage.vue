<template>
  <div class="remote-sessions-page" v-loading="loading">
    <div class="page-header">
      <h2>{{ t('remoteSessions.title') }}</h2>
      <div class="header-actions">
        <el-select v-model="status" style="width: 130px" @change="load(true)">
          <el-option :label="t('remoteSessions.statusActive')" value="active" />
          <el-option :label="t('remoteSessions.statusEnded')" value="ended" />
          <el-option :label="t('remoteSessions.statusAll')" value="all" />
        </el-select>
        <el-input
          v-model="userId"
          :placeholder="t('remoteSessions.userId')"
          style="width: 140px"
          clearable
          @keyup.enter="load(true)"
          @clear="load(true)"
        />
        <el-input
          v-model="deviceId"
          :placeholder="t('remoteSessions.deviceId')"
          style="width: 160px"
          clearable
          @keyup.enter="load(true)"
          @clear="load(true)"
        />
        <el-button :icon="Refresh" size="small" @click="load(true); loadStorage()">{{ t('common.refresh') }}</el-button>
        <el-button
          v-if="deviceId"
          type="danger"
          size="small"
          :loading="clearing"
          @click="clearDevice"
        >{{ t('remoteSessions.clearDevice') }}</el-button>
        <el-button
          size="small"
          :loading="unbanning"
          @click="unbanSource"
        >{{ t('remoteSessions.unbanSource') }}</el-button>
      </div>
    </div>

    <!-- What the recording directory actually costs on disk. The two byte
         totals differ on purpose: the index only knows about rows, the disk walk
         also counts thumbnails and .part staging files, and that gap is exactly
         what a quota is measured against. -->
    <div v-if="recordingStorage" class="storage-summary" :class="{ 'storage-over': storageRows(recordingStorage).over }">
      <span class="storage-title">{{ t('remoteSessions.storageTitle') }}</span>
      <span>{{ t('remoteSessions.storageArtifacts') }}: {{ storageRows(recordingStorage).artifacts }}</span>
      <span>{{ t('remoteSessions.storageFiles') }}: {{ storageRows(recordingStorage).files }}</span>
      <span>{{ t('remoteSessions.storageIndexed') }}: {{ storageRows(recordingStorage).indexed }}</span>
      <span>{{ t('remoteSessions.storageDisk') }}: {{ storageRows(recordingStorage).disk }}</span>
      <span
        v-if="storageGap(recordingStorage).hasGap"
        :title="t('remoteSessions.storageUnindexedHint')"
      >{{ t('remoteSessions.storageUnindexed') }}: {{ formatSize(storageGap(recordingStorage).hidden) }}</span>
      <span>{{ t('remoteSessions.storageQuota') }}: {{ storageRows(recordingStorage).quota }}</span>
      <span v-if="storageRows(recordingStorage).hasQuota">
        {{ t('remoteSessions.storageRemaining') }}: {{ storageRows(recordingStorage).remaining }}
      </span>
      <span>{{ t('remoteSessions.storageRetention') }}: {{ storageRows(recordingStorage).retention }}</span>
      <span>{{ t('remoteSessions.storageStale') }}: {{ storageRows(recordingStorage).stale }}</span>
      <el-tag v-if="storageRows(recordingStorage).over" type="danger" size="small">
        {{ t('remoteSessions.overQuota') }}
      </el-tag>
      <el-button
        size="small"
        :loading="sweeping"
        @click="sweepRecordingsNow"
      >{{ t('remoteSessions.sweep') }}</el-button>
    </div>

    <el-card shadow="never">
      <el-table
        :data="rows"
        stripe
        style="width: 100%"
        size="small"
        :row-key="(row) => row.session?.session_id"
      >
        <el-table-column type="expand">
          <template #default="{ row }">
            <div class="detail-grid">
              <div class="detail-block">
                <div class="detail-title">
                  {{ t('remoteSessions.recordings') }}: {{ row.counts?.recordings ?? 0 }}
                </div>
                <div v-if="(row.recordings ?? []).length === 0" class="detail-empty">
                  {{ t('remoteSessions.none') }}
                </div>
                <div v-for="r in row.recordings" :key="r.id" class="detail-line mono">
                  #{{ r.id }} · {{ r.status }} · {{ formatSize(r.size_bytes) }}
                </div>
              </div>
              <div class="detail-block">
                <div class="detail-title">
                  {{ t('remoteSessions.accessLog') }}: {{ row.counts?.access_log ?? 0 }}
                </div>
                <div v-if="(row.access_log ?? []).length === 0" class="detail-empty">
                  {{ t('remoteSessions.none') }}
                </div>
                <div v-for="l in row.access_log" :key="l.id" class="detail-line">
                  <span>{{ formatTime(l.created_at) }}</span>
                  <span class="mono">{{ l.result }}</span>
                  <span>{{ l.ip }}</span>
                </div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column :label="t('remoteSessions.session')" min-width="210">
          <template #default="{ row }">
            <div class="mono">{{ row.session?.device_id }}</div>
            <div class="sub mono">{{ row.session?.session_id }}</div>
          </template>
        </el-table-column>
        <el-table-column :label="t('remoteSessions.user')" width="90">
          <template #default="{ row }">{{ row.session?.user_id || '-' }}</template>
        </el-table-column>
        <el-table-column :label="t('remoteSessions.state')" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.session?.active" type="success" size="small">
              {{ t('remoteSessions.statusActive') }}
            </el-tag>
            <el-tag v-else type="info" size="small">{{ t('remoteSessions.statusEnded') }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('remoteSessions.client')" min-width="150">
          <template #default="{ row }">
            <div class="mono">{{ row.session?.client_id || '-' }}</div>
            <div class="sub mono">{{ row.session?.client_ip || '-' }}</div>
          </template>
        </el-table-column>
        <el-table-column :label="t('remoteSessions.started')" width="160">
          <template #default="{ row }">{{ formatTime(row.session?.started_at) }}</template>
        </el-table-column>
        <el-table-column :label="t('remoteSessions.duration')" width="90">
          <template #default="{ row }">{{ formatDuration(row.session?.duration_seconds) }}</template>
        </el-table-column>
        <el-table-column :label="t('remoteSessions.endReason')" width="130">
          <template #default="{ row }">
            <span class="mono">{{ row.session?.end_reason || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('common.operation')" width="190" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.session?.active"
              link
              type="danger"
              size="small"
              :loading="kickingId === row.session?.session_id"
              @click="kick(row)"
            >{{ t('remoteSessions.kick') }}</el-button>
            <el-button
              v-if="row.session?.active && row.session?.client_ip"
              link
              type="warning"
              size="small"
              :loading="kickingId === row.session?.session_id"
              @click="banSource(row)"
            >{{ t('remoteSessions.banSource') }}</el-button>
            <span v-else>-</span>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="rows.length === 0 && !loading" class="empty-hint">{{ t('remoteSessions.empty') }}</div>
      <div v-if="nextCursor" class="load-more">
        <el-button size="small" :loading="loading" @click="load(false)">
          {{ t('remoteSessions.loadMore') }}
        </el-button>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { formatSize } from './storage-size.js'
import { storageRows, storageGap } from './recording-storage-model.js'
import {
  listRemoteSessions,
  revokeRemoteSession,
  revokeDeviceSessions,
  banAccessSource,
  unbanAccessSource,
  getRecordingStorage,
  sweepRecordings,
} from '../api/remoteSessions.js'

const { t } = useI18n()

const rows = ref([])
const nextCursor = ref('')
const loading = ref(false)
const status = ref('active')
const userId = ref('')
const deviceId = ref('')
const kickingId = ref('')
const clearing = ref(false)
const unbanning = ref(false)
const recordingStorage = ref(null)
const sweeping = ref(false)
let loadSeq = 0

function formatTime(value) {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

function formatDuration(seconds) {
  const s = Math.max(0, Math.floor(seconds || 0))
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = s % 60
  const pad = (n) => String(n).padStart(2, '0')
  return h > 0 ? `${h}:${pad(m)}:${pad(sec)}` : `${m}:${pad(sec)}`
}


async function loadStorage() {
  try {
    recordingStorage.value = await getRecordingStorage()
  } catch (e) {
    ElMessage.error(t('common.loadFailed') + ': ' + e.message)
  }
}

async function sweepRecordingsNow() {
  try {
    await ElMessageBox.confirm(t('remoteSessions.sweepConfirm'), t('common.confirm'), { type: 'warning' })
  } catch {
    return
  }
  sweeping.value = true
  try {
    const r = await sweepRecordings()
    let msg = t('remoteSessions.sweepDone', {
      purged: r.artifacts_purged ?? 0,
      freed: formatSize(r.bytes_freed ?? 0),
      orphans: r.orphans_removed ?? 0,
      uploads: r.uploads_reaped ?? 0,
    })
    if (r.stale_marked_failed) msg += ' ' + t('remoteSessions.sweepStale', { stale: r.stale_marked_failed })
    ElMessage.success(msg)
    await loadStorage()
  } catch (e) {
    ElMessage.error(t('common.failed') + ': ' + e.message)
  } finally {
    sweeping.value = false
  }
}

async function load(reset = true, silent = false) {
  const seq = ++loadSeq
  loading.value = true
  try {
    const data = await listRemoteSessions({
      status: status.value,
      userId: userId.value ? Number(userId.value) : undefined,
      deviceId: deviceId.value.trim(),
      cursor: reset ? '' : nextCursor.value,
      limit: 50,
    })
    if (seq !== loadSeq) return
    const items = data?.items ?? []
    rows.value = reset ? items : [...rows.value, ...items]
    nextCursor.value = data?.next_cursor || ''
  } catch (err) {
    if (seq === loadSeq && !silent) {
      ElMessage.error(err?.detail || err?.message || String(err))
    }
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

async function kick(row) {
  const session = row.session
  if (!session?.session_id) return
  try {
    await ElMessageBox.confirm(
      t('remoteSessions.kickConfirm', { device: session.device_id }),
      t('remoteSessions.kick'),
      { type: 'warning' },
    )
  } catch {
    return
  }
  kickingId.value = session.session_id
  try {
    await revokeRemoteSession(session.session_id)
    ElMessage.success(t('remoteSessions.kicked'))
    await load(true)
  } catch (err) {
    ElMessage.error(err?.detail || err?.message || String(err))
  } finally {
    kickingId.value = ''
  }
}

async function banSource(row) {
  const session = row.session
  if (!session?.device_id || !session?.client_ip) return
  try {
    await ElMessageBox.confirm(
      t('remoteSessions.banConfirm', { ip: session.client_ip }),
      t('remoteSessions.banSource'),
      { type: 'warning' },
    )
  } catch {
    return
  }
  kickingId.value = session.session_id
  try {
    await banAccessSource(session.device_id, session.client_ip)
    ElMessage.success(t('remoteSessions.banned', { ip: session.client_ip }))
  } catch (err) {
    ElMessage.error(err?.detail || err?.message || String(err))
  } finally {
    kickingId.value = ''
  }
}

async function clearDevice() {
  const id = deviceId.value.trim()
  if (!id) return
  try {
    await ElMessageBox.confirm(
      t('remoteSessions.clearConfirm', { device: id }),
      t('remoteSessions.clearDevice'),
      { type: 'warning' },
    )
  } catch {
    return
  }
  clearing.value = true
  try {
    const data = await revokeDeviceSessions(id)
    const base = t('remoteSessions.clearDone', {
      revoked: data?.revoked ?? 0,
      failed: data?.failed ?? 0,
    })
    ElMessage.success(data?.truncated ? `${base} ${t('remoteSessions.clearTruncated')}` : base)
    await load(true)
  } catch (err) {
    ElMessage.error(err?.detail || err?.message || String(err))
  } finally {
    clearing.value = false
  }
}

let pollTimer = null

function startPolling() {
  pollTimer = setInterval(() => {
    if (document.hidden) return
    if (status.value !== 'active') return
    if (loading.value || clearing.value || kickingId.value) return
    load(true, true)
  }, 10000)
}

async function unbanSource() {
  const id = deviceId.value.trim()
  if (!id) {
    ElMessage.error(t('remoteSessions.deviceRequiredForUnban'))
    return
  }
  let source
  try {
    const result = await ElMessageBox.prompt(
      t('remoteSessions.unbanPrompt'),
      t('remoteSessions.unbanSource'),
      {
        inputPlaceholder: '203.0.113.7 or 10.211.55.0/24',
        inputValidator: (value) => String(value || '').trim().length > 0,
      },
    )
    source = String(result.value || '').trim()
  } catch {
    return
  }
  try {
    await ElMessageBox.confirm(
      t('remoteSessions.unbanConfirm', { ip: source }),
      t('remoteSessions.unbanSource'),
      { type: 'warning' },
    )
  } catch {
    return
  }
  unbanning.value = true
  try {
    await unbanAccessSource(id, source)
    ElMessage.success(t('remoteSessions.unbanned', { ip: source }))
  } catch (err) {
    ElMessage.error(err?.detail || err?.message || String(err))
  } finally {
    unbanning.value = false
  }
}

onMounted(() => {
  load(true)
  loadStorage()
  startPolling()
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<style scoped>
.remote-sessions-page {
  width: 100%;
  padding: 20px;
  box-sizing: border-box;
}
.storage-summary {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 16px;
  padding: 10px 14px;
  margin-bottom: 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  font-size: 13px;
}
.storage-summary.storage-over { border-color: var(--el-color-danger); }
.storage-title { color: var(--el-text-color-primary); font-weight: 600; }
.page-header {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
.page-header h2 {
  margin: 0;
  font-size: 18px;
}
.header-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}
.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
}
.sub {
  color: var(--el-text-color-secondary);
  margin-top: 2px;
}
.detail-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 16px;
  padding: 4px 12px 8px;
}
.detail-title {
  font-weight: 600;
  margin-bottom: 6px;
  font-size: 13px;
}
.detail-empty {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.detail-line {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  font-size: 12px;
  color: var(--el-text-color-regular);
  margin-bottom: 3px;
}
.empty-hint {
  padding: 18px 0;
  text-align: center;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.load-more {
  text-align: center;
  padding-top: 12px;
}
</style>
