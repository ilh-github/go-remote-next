<template>
  <div class="notification-deliveries-page" v-loading="loading">
    <div class="page-header">
      <h2>{{ t('notificationDeliveries.title') }}</h2>
      <span class="page-subtitle">{{ t('notificationDeliveries.subtitle') }}</span>
    </div>

    <!-- A 2xx from an endpoint proves the bytes arrived, not that anyone read
         them. The ack loop is what closes that gap, so the unconfirmed count is
         the headline rather than a footnote. -->
    <el-card v-if="stats" shadow="never" class="stats-card">
      <div class="stats-row">
        <div class="stat">
          <span class="stat-label">{{ t('notificationDeliveries.total') }}</span>
          <span class="stat-value">{{ stats.total }}</span>
        </div>
        <div v-for="row in stats.rows" :key="row[0]" class="stat">
          <span class="stat-label">{{ t(`notificationDeliveries.${row[0]}`) }}</span>
          <span class="stat-value">{{ row[1] }}</span>
        </div>
        <div class="stat">
          <span class="stat-label">{{ t('notificationDeliveries.acked') }}</span>
          <span class="stat-value">{{ stats.acked }}</span>
        </div>
        <div class="stat" :class="{ 'stat-attention': stats.unacked > 0 }">
          <span class="stat-label">{{ t('notificationDeliveries.unacked') }}</span>
          <span class="stat-value">{{ stats.unacked }}</span>
        </div>
      </div>

      <el-alert
        v-if="stats.unacked > 0"
        :title="t('notificationDeliveries.attention')"
        type="warning"
        show-icon
        :closable="false"
      />
      <el-alert
        v-else-if="stats.needsAttention"
        :title="t('notificationDeliveries.stuck')"
        type="warning"
        show-icon
        :closable="false"
      />
      <el-alert
        v-else
        :title="t('notificationDeliveries.healthy')"
        type="success"
        show-icon
        :closable="false"
      />

      <div class="meta-row">
        <span>{{ t('notificationDeliveries.retention') }}: {{ stats.retention }}</span>
        <span>{{ t('notificationDeliveries.archiveRetention') }}: {{ stats.archiveRetention }}</span>
        <span>{{ t('notificationDeliveries.archiveDir') }}: {{ stats.archiveDir }}</span>
      </div>
    </el-card>

    <el-card shadow="never">
      <template #header>
        <span>{{ t('notificationDeliveries.sweepTitle') }}</span>
      </template>
      <el-form :inline="true" @submit.prevent>
        <el-form-item :label="t('notificationDeliveries.olderThanDays')">
          <el-input-number v-model="sweepForm.olderThanDays" :min="0" :controls="false" size="small" style="width: 120px" />
        </el-form-item>
        <el-form-item :label="t('notificationDeliveries.archiveRetentionDays')">
          <el-input-number v-model="sweepForm.archiveRetentionDays" :min="0" :controls="false" size="small" style="width: 120px" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" size="small" :loading="sweeping" @click="runSweep">
            {{ t('notificationDeliveries.sweep') }}
          </el-button>
        </el-form-item>
      </el-form>
      <div class="sweep-hint">{{ t('notificationDeliveries.sweepHint') }}</div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  getNotificationDeliveryStats,
  sweepNotificationDeliveries,
} from '../api/notifications.js'
import { summary } from './notification-stats-model.js'

const { t } = useI18n()

const loading = ref(false)
const sweeping = ref(false)
const raw = ref(null)
const sweepForm = reactive({ olderThanDays: undefined, archiveRetentionDays: undefined })

// Derived rather than stored: the panel shows what the model says the numbers
// mean, so a field the server adds later cannot silently go unrendered.
const stats = ref(null)

async function load() {
  loading.value = true
  try {
    raw.value = await getNotificationDeliveryStats()
    stats.value = summary(raw.value)
  } catch (e) {
    ElMessage.error(t('notificationDeliveries.loadFailed') + ': ' + e.message)
  } finally {
    loading.value = false
  }
}

async function runSweep() {
  try {
    await ElMessageBox.confirm(
      t('notificationDeliveries.sweepConfirm'),
      t('common.confirm'),
      { type: 'warning' },
    )
  } catch {
    return
  }
  sweeping.value = true
  try {
    const r = await sweepNotificationDeliveries({
      olderThanDays: sweepForm.olderThanDays || undefined,
      archiveRetentionDays: sweepForm.archiveRetentionDays || undefined,
    })
    const purged = r.purged ?? 0
    const message = purged > 0
      ? t('notificationDeliveries.sweepDone', {
          purged,
          cutoff: r.cutoff || '-',
          removed: r.archive_removed ?? 0,
          moved: r.archive_moved ?? 0,
          deleted: r.archive_deleted ?? 0,
        })
      : t('notificationDeliveries.sweepNothing')
    ElMessage.success(message)
    await load()
  } catch (e) {
    ElMessage.error(t('common.failed') + ': ' + e.message)
  } finally {
    sweeping.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.notification-deliveries-page {
  width: 100%;
  padding: 20px;
  box-sizing: border-box;
}
.page-header {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 12px;
  margin-bottom: 16px;
}
.page-header h2 { margin: 0; font-size: 18px; }
.page-subtitle { color: var(--el-text-color-secondary); font-size: 13px; }
.stats-card { margin-bottom: 16px; }
.stats-row {
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
  margin-bottom: 12px;
}
.stat { display: flex; flex-direction: column; gap: 2px; }
.stat-label { color: var(--el-text-color-secondary); font-size: 12px; }
.stat-value { font-size: 20px; font-weight: 600; }
.stat-attention .stat-value { color: var(--el-color-danger); }
.meta-row {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin-top: 12px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.sweep-hint { color: var(--el-text-color-secondary); font-size: 12px; }
</style>
