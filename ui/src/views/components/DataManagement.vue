<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  Database,
  FolderLock,
  HardDrive,
  RefreshCw,
  Download,
  Trash2,
  Shrink,
  AlertTriangle,
  Info,
  CheckCircle2,
} from 'lucide-vue-next'
import { dataApi, type DataStats, type BucketStats, type BackupResult, type CompactResult } from '@/api/data'
import DangerDialog from '@/components/edge/DangerDialog.vue'

const loading = ref(false)
const stats = ref<DataStats | null>(null)
const backupResult = ref<BackupResult | null>(null)
const compactResult = ref<CompactResult | null>(null)
const clearResult = ref<string | null>(null)
const showClearConfirm = ref(false)

const configBuckets = computed(() => (stats.value?.buckets ?? []).filter(b => b.database === 'config'))
const runtimeBuckets = computed(() => (stats.value?.buckets ?? []).filter(b => b.database === 'runtime'))
const configBucketCount = computed(() => configBuckets.value.length)
const runtimeBucketCount = computed(() => runtimeBuckets.value.length)
const clearableSize = computed(() =>
  (stats.value?.buckets ?? []).reduce((sum, b) => (b.clearable ? sum + b.total_size : sum), 0)
)

const categoryLabels: Record<string, string> = {
  config: '配置',
  runtime: '运行时',
  history: '历史',
  cache: '缓存',
  unknown: '未知',
}

function formatSize(bytes: number): string {
  const n = Number(bytes)
  if (!n || n <= 0) return '0 MB'
  const mb = n / (1024 * 1024)
  return `${mb < 0.01 ? 0.01 : mb.toFixed(2)} MB`
}

function categoryLabel(b: BucketStats): string {
  return categoryLabels[b.category] || b.category
}

async function fetchStats() {
  loading.value = true
  try {
    stats.value = await dataApi.stats()
  } catch (e) {
    clearResult.value = `获取数据库统计失败 | ${(e as Error)?.message || e}`
  } finally {
    loading.value = false
  }
}

async function handleBackup() {
  backupResult.value = null
  try {
    backupResult.value = await dataApi.backupConfig()
  } catch (e) {
    backupResult.value = { status: 'error', message: `备份失败 | ${(e as Error)?.message || e}`, backup_path: '', backup_time: '', original: '', size_bytes: 0, size_display: '' }
  }
}

async function handleClearAll() {
  showClearConfirm.value = false
  try {
    const res = await dataApi.clearAllRuntime()
    clearResult.value = `已清空 ${res.cleared?.length ?? 0} 个运行时 bucket，配置库不受影响`
    await fetchStats()
  } catch (e) {
    clearResult.value = `清空失败 | ${(e as Error)?.message || e}`
  }
}

async function handleCompact() {
  compactResult.value = null
  try {
    compactResult.value = await dataApi.compactRuntime()
    await fetchStats()
  } catch (e) {
    clearResult.value = `压缩失败 | ${(e as Error)?.message || e}`
  }
}

onMounted(fetchStats)
</script>

<template>
  <div class="space-y-5">
    <!-- 数据库概览 -->
    <div class="data-card" style="background: var(--bg-secondary); border: 1px solid var(--border-color);">
      <div class="card-header" style="border-bottom: 1px solid var(--border-color);">
        <Database class="w-4 h-4" style="width:16px;height:16px;color: var(--accent-primary);" />
        <span class="text-sm font-semibold" style="color: var(--text-primary);">数据库概览</span>
      </div>
      <div class="overview-grid p-5">
        <div class="overview-box" style="border: 1px solid var(--border-color);">
          <div class="flex items-center justify-between mb-2">
            <span class="text-sm font-semibold" style="color: var(--text-primary);">配置库 (config.db)</span>
            <span class="tag tag-protected">受保护 · 不可清理</span>
          </div>
          <div class="overview-field"><span class="field-label">数据库路径</span><span class="field-value mono">{{ stats?.config_db?.path || '-' }}</span></div>
          <div class="overview-field"><span class="field-label">配置 Bucket 数</span><span class="field-value">{{ configBucketCount }} 个</span></div>
        </div>
        <div class="overview-box" style="border: 1px solid var(--border-color);">
          <div class="flex items-center justify-between mb-2">
            <span class="text-sm font-semibold" style="color: var(--text-primary);">运行时库 (edgeos.db)</span>
            <span class="tag tag-clearable">可清理 · 可压缩</span>
          </div>
          <div class="overview-field"><span class="field-label">数据库路径</span><span class="field-value mono">{{ stats?.runtime_db?.path || '-' }}</span></div>
          <div class="overview-field"><span class="field-label">运行时 Bucket 数</span><span class="field-value">{{ runtimeBucketCount }} 个</span></div>
        </div>
        <div class="overview-box overview-stats" style="border: 1px solid var(--border-color);">
          <div class="overview-field"><span class="field-label">总大小</span><span class="field-value">{{ formatSize(stats?.total_size || 0) }}</span></div>
          <div class="overview-field"><span class="field-label">可清理数据</span><span class="field-value text-warning">{{ formatSize(clearableSize) }}</span></div>
        </div>
      </div>
    </div>

    <!-- 配置库管理 -->
    <div class="data-card" style="background: var(--bg-secondary); border: 1px solid var(--border-color);">
      <div class="card-header" style="border-bottom: 1px solid var(--border-color);">
        <FolderLock class="w-4 h-4" style="width:16px;height:16px;color: var(--accent-primary);" />
        <span class="text-sm font-semibold" style="color: var(--text-primary);">配置库管理</span>
      </div>
      <div class="p-5 space-y-3">
        <div class="info-alert">
          <Info class="w-4 h-4 flex-shrink-0" style="width:16px;height:16px;color: var(--accent-primary);" />
          <p class="text-xs" style="color: var(--text-secondary);">
            配置库包含节点、用户、中间件与 EAN 等关键配置。<strong>配置库不可清理</strong>，建议定期备份或导出（系统设置 → 导出配置）。
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <button
            class="action-btn"
            :disabled="loading"
            style="background: rgba(14,165,233,0.1); color: var(--accent-primary); border: 1px solid rgba(14,165,233,0.25);"
            @click="handleBackup"
          >
            <Download class="w-4 h-4" style="width:16px;height:16px;" />
            {{ backupResult ? '备份中...' : '备份配置库' }}
          </button>
          <button
            class="action-btn"
            :disabled="loading"
            style="background: var(--bg-tertiary); color: var(--text-secondary); border: 1px solid var(--border-color);"
            @click="fetchStats"
          >
            <RefreshCw class="w-4 h-4" style="width:16px;height:16px;" :class="{ 'animate-spin': loading }" />
            刷新统计
          </button>
        </div>
        <div v-if="backupResult" class="result-alert" :class="{ error: backupResult.status !== 'success' }">
          <CheckCircle2 v-if="backupResult.status === 'success'" class="w-4 h-4 flex-shrink-0" style="width:16px;height:16px;color:#22c55e;" />
          <AlertTriangle v-else class="w-4 h-4 flex-shrink-0" style="width:16px;height:16px;color:#ef4444;" />
          <div class="text-xs">
            <div style="color: var(--text-primary);">{{ backupResult.message }}</div>
            <div v-if="backupResult.backup_path" class="mono" style="color: var(--text-secondary);">{{ backupResult.backup_path }} · {{ backupResult.size_display }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- 运行时库管理 -->
    <div class="data-card" style="background: var(--bg-secondary); border: 1px solid var(--border-color);">
      <div class="card-header" style="border-bottom: 1px solid var(--border-color);">
        <HardDrive class="w-4 h-4" style="width:16px;height:16px;color: #F59E0B;" />
        <span class="text-sm font-semibold" style="color: var(--text-primary);">运行时库管理</span>
      </div>
      <div class="p-5 space-y-3">
        <div class="info-alert warning">
          <AlertTriangle class="w-4 h-4 flex-shrink-0" style="width:16px;height:16px;color:#F59E0B;" />
          <p class="text-xs" style="color: var(--text-secondary);">
            运行时库包含节点、设备、点位、告警、命令等实时数据。清理运行时数据<strong>不影响采集配置</strong>，但会丢失实时记录。
          </p>
        </div>

        <!-- 运行时 bucket 表格 -->
        <div class="data-table-wrap">
          <table class="data-table" style="border: 1px solid var(--border-color);">
            <thead>
              <tr style="background: var(--bg-tertiary);">
                <th>名称</th>
                <th>分类</th>
                <th>记录数</th>
                <th>大小</th>
                <th>状态</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="b in runtimeBuckets" :key="b.name">
                <td class="mono">{{ b.name }}</td>
                <td>{{ categoryLabel(b) }}</td>
                <td>{{ b.record_count }}</td>
                <td>{{ formatSize(b.total_size) }}</td>
                <td>
                  <span v-if="b.clearable" class="status-clearable">可清理</span>
                  <span v-else class="status-protected">受保护</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="flex flex-wrap items-center gap-3">
          <button
            class="action-btn"
            :disabled="loading || runtimeBuckets.length === 0"
            style="background: rgba(245,158,11,0.1); color: #F59E0B; border: 1px solid rgba(245,158,11,0.25);"
            @click="showClearConfirm = true"
          >
            <Trash2 class="w-4 h-4" style="width:16px;height:16px;" />
            清空全部运行时数据
          </button>
          <button
            class="action-btn"
            :disabled="loading"
            style="background: var(--bg-tertiary); color: var(--text-secondary); border: 1px solid var(--border-color);"
            @click="handleCompact"
          >
            <Shrink class="w-4 h-4" style="width:16px;height:16px;" />
            压缩运行时库
          </button>
        </div>

        <div v-if="compactResult" class="result-alert">
          <CheckCircle2 class="w-4 h-4 flex-shrink-0" style="width:16px;height:16px;color:#22c55e;" />
          <div class="text-xs">
            <div style="color: var(--text-primary);">{{ compactResult.message }}</div>
            <div class="mono" style="color: var(--text-secondary);">
              {{ compactResult.before_size }} → {{ compactResult.after_size }}，节省 {{ compactResult.saved_size }}
            </div>
          </div>
        </div>
        <div v-if="clearResult && !compactResult" class="result-alert">
          <CheckCircle2 class="w-4 h-4 flex-shrink-0" style="width:16px;height:16px;color:#22c55e;" />
          <span class="text-xs" style="color: var(--text-secondary);">{{ clearResult }}</span>
        </div>
      </div>
    </div>

    <!-- 配置库 Bucket 详情（只读） -->
    <div class="data-card" style="background: var(--bg-secondary); border: 1px solid var(--border-color);">
      <div class="card-header" style="border-bottom: 1px solid var(--border-color);">
        <Database class="w-4 h-4" style="width:16px;height:16px;color: var(--accent-primary);" />
        <span class="text-sm font-semibold" style="color: var(--text-primary);">配置库 Bucket 详情（只读）</span>
        <span class="tag tag-protected ml-auto">受保护</span>
      </div>
      <div class="p-5">
        <div class="data-table-wrap">
          <table class="data-table" style="border: 1px solid var(--border-color);">
            <thead>
              <tr style="background: var(--bg-tertiary);">
                <th>名称</th>
                <th>分类</th>
                <th>记录数</th>
                <th>大小</th>
                <th>状态</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="b in configBuckets" :key="b.name">
                <td class="mono">{{ b.name }}</td>
                <td>{{ categoryLabel(b) }}</td>
                <td>{{ b.record_count }}</td>
                <td>{{ formatSize(b.total_size) }}</td>
                <td><span class="status-protected">不可清理</span></td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- 清空运行时确认弹窗 -->
    <DangerDialog
      v-model:open="showClearConfirm"
      title="确认清空全部运行时数据"
      :description="`将清空运行时库（edgeos.db）中全部 ${runtimeBucketCount} 个 bucket 的记录，并自动压缩。采集配置（config.db）不受影响。确认要继续吗？`"
      actionName="清空运行时数据"
      variant="warning"
      @confirm="handleClearAll"
    />
  </div>
</template>

<style scoped>
.data-card { border-radius: 0.75rem; overflow: hidden; }
.card-header { display: flex; align-items: center; gap: 8px; padding: 14px 20px; }

.overview-grid { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 16px; }
.overview-box { border-radius: 0.75rem; padding: 14px 16px; display: flex; flex-direction: column; gap: 8px; }
.overview-stats { justify-content: center; }
.overview-field { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.field-label { font-size: 12px; color: var(--text-secondary); }
.field-value { font-size: 12px; font-weight: 600; color: var(--text-primary); }
.mono { font-family: 'JetBrains Mono', monaco, monospace; font-size: 12px; }

.tag { font-size: 10px; font-weight: 600; padding: 2px 8px; border-radius: 0.375rem; white-space: nowrap; }
.tag-protected { background: rgba(239,68,68,0.08); color: #ef4444; border: 1px solid rgba(239,68,68,0.2); }
.tag-clearable { background: rgba(34,197,94,0.08); color: #22c55e; border: 1px solid rgba(34,197,94,0.2); }

.info-alert { display: flex; align-items: flex-start; gap: 8px; padding: 10px 12px; border-radius: 0.75rem; background: rgba(14,165,233,0.06); border: 1px solid rgba(14,165,233,0.15); }
.info-alert.warning { background: rgba(245,158,11,0.06); border-color: rgba(245,158,11,0.15); }

.action-btn { display: inline-flex; align-items: center; gap: 8px; padding: 9px 14px; border-radius: 0.75rem; font-size: 13px; font-weight: 500; cursor: pointer; transition: all 0.15s ease; }
.action-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.action-btn:hover:not(:disabled) { filter: brightness(1.05); }

.result-alert { display: flex; align-items: center; gap: 8px; padding: 10px 12px; border-radius: 0.75rem; background: rgba(34,197,94,0.06); border: 1px solid rgba(34,197,94,0.15); }
.result-alert.error { background: rgba(239,68,68,0.06); border-color: rgba(239,68,68,0.2); }

.data-table-wrap { overflow-x: auto; border-radius: 0.75rem; }
.data-table { width: 100%; border-collapse: collapse; font-size: 12px; }
.data-table th { text-align: left; padding: 8px 12px; font-weight: 600; color: var(--text-secondary); }
.data-table td { padding: 8px 12px; color: var(--text-primary); border-top: 1px solid var(--border-color); }

.status-clearable { font-size: 11px; color: #22c55e; }
.status-protected { font-size: 11px; color: #ef4444; }
.text-warning { color: #F59E0B; }

@media (max-width: 900px) {
  .overview-grid { grid-template-columns: 1fr; }
}
</style>
