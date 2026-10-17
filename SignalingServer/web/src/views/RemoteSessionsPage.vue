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
        <el-button :icon="Refresh" size="small" @click="load(true)">{{ t('common.refresh') }}</el-button>
        <el-button
          v-if="deviceId"
          type="danger"
          size="small"
          :loading="clearing"
          @click="clearDevice"
        >{{ t('remoteSessions.clearDevice') }}</el-button>
      </div>
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
        <el-table-column :label="t('common.operation')" width="110" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.session?.active"
              link
              type="danger"
              size="small"
              :loading="kickingId === row.session?.session_id"
              @click="kick(row)"
            >{{ t('remoteSessions.kick') }}</el-button>
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
import {
  listRemoteSessions,
  revokeRemoteSession,
  revokeDeviceSessions,
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

function formatSize(bytes) {
  const b = Number(bytes) || 0
  if (b <= 0) return '-'
  if (b < 1024) return `${b} B`
  const units = ['KB', 'MB', 'GB']
  let v = b / 1024
  let i = 0
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i += 1 }
  return `${v.toFixed(v >= 10 ? 0 : 1)} ${units[i]}`
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

onMounted(() => {
  load(true)
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
