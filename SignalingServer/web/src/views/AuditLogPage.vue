<template>
  <div class="audit-page" v-loading="loading">
    <div class="page-header">
      <h2>{{ t('auditLog.title') }}</h2>
      <div class="header-actions">
        <el-dropdown @command="exportLogs">
          <el-button :icon="Download" size="small" :loading="exporting">
            {{ t('common.export') }}<el-icon class="el-icon--right"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="csv">{{ t('auditLog.exportCsv') }}</el-dropdown-item>
              <el-dropdown-item command="json">{{ t('auditLog.exportJson') }}</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-button type="primary" :icon="Refresh" size="small" @click="loadLogs">{{ t('common.refresh') }}</el-button>
      </div>
    </div>

    <el-card shadow="never" class="filter-card">
      <div class="filter-bar">
        <el-select v-model="filters.action" :placeholder="t('auditLog.filterAction')" clearable style="width: 160px" @change="handleFilter">
          <el-option label="login" value="login" />
          <el-option label="create_user" value="create_user" />
          <el-option label="update_user" value="update_user" />
          <el-option label="delete_user" value="delete_user" />
          <el-option label="create_admin" value="create_admin" />
          <el-option label="update_admin" value="update_admin" />
          <el-option label="delete_admin" value="delete_admin" />
          <el-option label="update_settings" value="update_settings" />
        </el-select>
        <el-input
          v-model="filters.admin"
          :placeholder="t('auditLog.filterAdmin')"
          clearable
          style="width: 160px"
          @clear="handleFilter"
          @keyup.enter="handleFilter"
        />
        <el-date-picker
          v-model="dateRange"
          type="daterange"
          :start-placeholder="t('auditLog.startDate')"
          :end-placeholder="t('auditLog.endDate')"
          size="default"
          style="width: 280px"
          @change="handleDateChange"
        />
        <el-button :icon="Search" @click="handleFilter">{{ t('common.search') }}</el-button>
      </div>
    </el-card>

    <el-card shadow="never" style="margin-top: 16px">
      <el-table :data="logs" stripe style="width: 100%" size="small">
        <el-table-column prop="created_at" :label="t('auditLog.time')" width="170">
          <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
        </el-table-column>
        <el-table-column prop="admin_username" :label="t('auditLog.admin')" width="120" />
        <el-table-column prop="action" :label="t('auditLog.action')" width="140">
          <template #default="{ row }">
            <el-tag size="small">{{ row.action }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="resource_type" :label="t('auditLog.resourceType')" width="120" />
        <el-table-column prop="resource_id" :label="t('auditLog.resourceId')" width="120" />
        <el-table-column prop="details" :label="t('auditLog.details')" show-overflow-tooltip />
        <el-table-column prop="ip" label="IP" width="140" />
      </el-table>

      <div class="pagination-bar">
        <!-- §3.1 cursor-based pagination. See components/CursorPagination.vue. -->
        <CursorPagination
          :cursor-stack="pagination.cursorStack"
          :next-cursor="pagination.nextCursor"
          :total="pagination.total"
          :limit="pagination.limit"
          :loading="loading"
          @prev="goPrevPage"
          @next="goNextPage"
          @update:limit="onLimitChange"
        />
      </div>
    </el-card>

    <!-- The archived half of the same trail. Retention moves expired rows into a
         per-day file before deleting them, so the live table above only covers the
         retention window; this is the only way "what happened last week" survives
         the purge. -->
    <el-card shadow="never" style="margin-top: 16px">
      <template #header>
        <div class="archive-header">
          <span>{{ t('auditLog.archiveTitle') }}</span>
          <span class="archive-hint">{{ t('auditLog.archiveHint') }}</span>
        </div>
      </template>

      <div class="filter-bar">
        <el-date-picker
          v-model="archiveDate"
          type="date"
          :placeholder="t('auditLog.archiveDate')"
          value-format="YYYY-MM-DD"
          size="default"
          style="width: 180px"
        />
        <el-select
          v-model="archiveFilters.action"
          :placeholder="t('auditLog.filterAction')"
          clearable
          style="width: 160px"
        >
          <el-option label="login" value="login" />
          <el-option label="create_user" value="create_user" />
          <el-option label="delete_user" value="delete_user" />
          <el-option label="update_settings" value="update_settings" />
        </el-select>
        <el-input
          v-model="archiveFilters.q"
          :placeholder="t('auditLog.archiveSearch')"
          clearable
          style="width: 200px"
          @keyup.enter="loadArchive"
        />
        <el-button :icon="Search" :loading="archiveLoading" @click="loadArchive">
          {{ t('auditLog.archiveSearchButton') }}
        </el-button>
      </div>

      <!-- An unconfigured archive directory is a fact the operator can act on, not
           an empty result. Rendering "nothing found" here would read as "retention
           deleted it", which is the opposite of what happened. -->
      <el-alert
        v-if="archiveNotConfigured"
        :title="t('auditLog.archiveNotConfigured')"
        type="warning"
        show-icon
        :closable="false"
        style="margin-bottom: 12px"
      />
      <el-alert
        v-else-if="archiveTruncated"
        :title="t('auditLog.archiveTruncated')"
        type="info"
        show-icon
        :closable="false"
        style="margin-bottom: 12px"
      />
      <!-- That day simply has no file. The archive is on and healthy, so this is
           an empty answer rather than a warning. -->
      <el-alert
        v-else-if="archiveNoFile"
        :title="t('auditLog.archiveNoFile')"
        type="info"
        show-icon
        :closable="false"
        style="margin-bottom: 12px"
      />
        type="info"
        show-icon
        :closable="false"
        style="margin-bottom: 12px"
      />

      <el-table :data="archiveRows" stripe style="width: 100%" size="small">
        <el-table-column prop="created_at" :label="t('auditLog.time')" width="170">
          <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
        </el-table-column>
        <el-table-column prop="admin_username" :label="t('auditLog.admin')" width="120" />
        <el-table-column prop="action" :label="t('auditLog.action')" width="140" />
        <el-table-column prop="resource_type" :label="t('auditLog.resourceType')" width="120" />
        <el-table-column prop="resource_id" :label="t('auditLog.resourceId')" width="120" />
        <el-table-column prop="details" :label="t('auditLog.details')" show-overflow-tooltip />
        <el-table-column prop="ip" label="IP" width="140" />
        <template #empty>{{ t('auditLog.archiveEmpty') }}</template>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { ArrowDown, Download, Refresh, Search } from '@element-plus/icons-vue'
import { exportAuditLogs, getAuditLogs, searchAuditArchive } from '../api/audit.js'
import { archiveErrorKind } from './audit-archive-errors.js'
import CursorPagination from '../components/CursorPagination.vue'

const { t } = useI18n()
const loading = ref(false)
const exporting = ref(false)
const logs = ref([])
const dateRange = ref(null)

const filters = reactive({ action: '', admin: '', dateFrom: '', dateTo: '' })
// §3.1 cursor pagination state.
const pagination = reactive({
  cursorStack: [''],
  nextCursor: '',
  total: 0,
  limit: 20
})

function formatDate(d) {
  if (!d) return '-'
  return new Date(d).toLocaleString('zh-CN')
}

async function loadLogs() {
  loading.value = true
  try {
    const data = await getAuditLogs({
      cursor: pagination.cursorStack.at(-1),
      limit: pagination.limit,
      action: filters.action,
      admin: filters.admin,
      dateFrom: filters.dateFrom,
      dateTo: filters.dateTo
    })
    logs.value = data.items || []
    pagination.nextCursor = data.next_cursor || ''
    if (typeof data.total === 'number') pagination.total = data.total
  } catch (e) {
    ElMessage.error(t('common.loadFailed') + ': ' + e.message)
  } finally {
    loading.value = false
  }
}

function resetCursorAndReload() {
  pagination.cursorStack = ['']
  pagination.nextCursor = ''
  loadLogs()
}

function goPrevPage() {
  if (pagination.cursorStack.length <= 1) return
  pagination.cursorStack.pop()
  loadLogs()
}

function goNextPage() {
  if (!pagination.nextCursor) return
  pagination.cursorStack.push(pagination.nextCursor)
  loadLogs()
}

function onLimitChange(n) {
  pagination.limit = n
  resetCursorAndReload()
}

function handleFilter() {
  resetCursorAndReload()
}

function handleDateChange(val) {
  if (val) {
    filters.dateFrom = val[0].toISOString()
    filters.dateTo = val[1].toISOString()
  } else {
    filters.dateFrom = ''
    filters.dateTo = ''
  }
  handleFilter()
}

async function exportLogs(format) {
  exporting.value = true
  try {
    const { blob, filename } = await exportAuditLogs(format)
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = filename || `audit-logs.${format}`
    link.click()
    URL.revokeObjectURL(url)
  } catch (error) {
    ElMessage.error(`${t('auditLog.exportFailed')}: ${error.message}`)
  } finally {
    exporting.value = false
  }
}

onMounted(loadLogs)

// ---------------------------------------------------------------------------
// Archived rows
//
// Not loaded on mount: the archive is a separate file from the live table, and
// most visits to this page are about today. It is one click, and the answer is
// labelled with the day it came from so a result is never mistaken for the live
// list.
// ---------------------------------------------------------------------------
const archiveDate = ref('')
const archiveFilters = reactive({ action: '', q: '' })
const archiveRows = ref([])
const archiveLoading = ref(false)
const archiveNotConfigured = ref(false)
const archiveNoFile = ref(false)
const archiveTruncated = ref(false)

async function loadArchive() {
  archiveLoading.value = true
  archiveNotConfigured.value = false
  archiveNoFile.value = false
  archiveTruncated.value = false
  try {
    const data = await searchAuditArchive({
      date: archiveDate.value || undefined,
      action: archiveFilters.action || undefined,
      q: archiveFilters.q || undefined,
      limit: 200,
    })
    archiveRows.value = data.items || []
    // The scan reports when it gave up, so an empty result is not proof the event
    // never happened - which is the same distinction the live read has to make
    // between "nothing" and "could not ask".
    archiveTruncated.value = data.truncated === true
    if (data.date) archiveDate.value = data.date
  } catch (error) {
    const kind = archiveErrorKind(error)
    if (kind === 'not_configured') {
      archiveNotConfigured.value = true
      archiveRows.value = []
    } else if (kind === 'no_file') {
      // That day simply has no file. It is an empty answer, not a failure, and
      // saying "not configured" here would send the operator to fix an env var
      // that is already set.
      archiveNoFile.value = true
      archiveRows.value = []
    } else {
      ElMessage.error(t('common.loadFailed') + ': ' + error.message)
    }
  } finally {
    archiveLoading.value = false
  }
}
</script>

<style scoped>
.audit-page { width: 100%; padding: 20px; box-sizing: border-box; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-header h2 { margin: 0; font-size: 24px; font-weight: 600; color: #303133; }
.header-actions { display: flex; gap: 8px; }
.filter-card { border-radius: 8px; }
.archive-header { display: flex; align-items: baseline; gap: 12px; }
/* Secondary to the card title: it explains what the panel is, not what to do. */
.archive-hint { font-size: 12px; font-weight: 400; color: #909399; }
.filter-bar { display: flex; gap: 12px; flex-wrap: wrap; align-items: center; }
.pagination-bar { display: flex; justify-content: flex-end; margin-top: 16px; }
</style>
