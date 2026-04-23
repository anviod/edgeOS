<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { AlertTriangle, RefreshCw, CheckCircle2, Filter } from 'lucide-vue-next'
import { useAlertStore } from '@/stores/alert'
import StatusBadge from '@/components/edge/StatusBadge.vue'

const alertStore = useAlertStore()
const filterStatus = ref<'all' | 'active' | 'acknowledged' | 'resolved'>('all')

onMounted(() => alertStore.fetchAlerts())

const filtered = computed(() => {
  if (filterStatus.value === 'all') return alertStore.alerts
  return alertStore.alerts.filter(a => a.status === filterStatus.value)
})

const levelConfig: Record<string, { label: string; color: string; bg: string }> = {
  critical: { label: 'CRITICAL', color: '#EF4444', bg: 'rgba(239,68,68,0.15)' },
  major:    { label: 'MAJOR',    color: '#F59E0B', bg: 'rgba(245,158,11,0.15)' },
  minor:    { label: 'MINOR',    color: '#FBBF24', bg: 'rgba(251,191,36,0.12)' },
  info:     { label: 'INFO',     color: '#9CA3AF', bg: 'rgba(107,114,128,0.12)' },
}

function getLevelCfg(level: string) {
  return levelConfig[level] ?? levelConfig.info
}

function formatTime(ts: number) {
  if (!ts) return '—'
  return new Date(ts < 1e12 ? ts * 1000 : ts).toLocaleString('zh-CN', {
    month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit',
  })
}
</script>

<template>
  <div class="space-y-5">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-bold" style="color: var(--text-primary);">告警管理</h1>
        <p class="text-sm mt-1" style="color: var(--text-secondary);">
          共 <span style="color: var(--text-primary);">{{ alertStore.alerts.length }}</span> 条告警，
          <span style="color: var(--destructive);">{{ alertStore.unacknowledgedCount }}</span> 条未确认
        </p>
      </div>
      <button
        @click="alertStore.fetchAlerts()"
        class="flex items-center gap-2 px-4 py-2 rounded-xl text-sm transition-colors hover:bg-white/5"
        style="color: var(--text-secondary); border: 1px solid var(--border-color);"
      >
        <RefreshCw class="w-4 h-4" :class="alertStore.loading ? 'animate-spin' : ''" style="width:16px;height:16px;" />
        刷新
      </button>
    </div>

    <!-- Filter tabs -->
    <div class="flex items-center gap-2">
      <Filter class="w-4 h-4" style="color: var(--text-secondary); width:16px;height:16px;" />
      <div class="flex gap-1 rounded-xl p-1" style="background: var(--bg-secondary); border: 1px solid var(--border-color);">
        <button
          v-for="opt in ['all', 'active', 'acknowledged', 'resolved'] as const"
          :key="opt"
          @click="filterStatus = opt"
          class="px-3 py-1.5 rounded-lg text-xs font-medium transition-all"
          :style="filterStatus === opt
            ? 'background: rgba(14,165,233,0.15); color: var(--accent-primary);'
            : 'color: var(--text-secondary);'
          "
        >
          {{ { all: '全部', active: '活跃', acknowledged: '已确认', resolved: '已解决' }[opt] }}
          <span
            v-if="opt === 'active' && alertStore.unacknowledgedCount > 0"
            class="ml-1 inline-flex items-center justify-center w-4 h-4 rounded-full text-xs"
            style="background: var(--destructive); color: white; font-size:9px;"
          >{{ alertStore.unacknowledgedCount }}</span>
        </button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="alertStore.loading && alertStore.alerts.length === 0" class="flex items-center justify-center py-16">
      <RefreshCw class="w-6 h-6 animate-spin" style="color: var(--accent-primary);" />
    </div>

    <!-- Empty -->
    <div v-else-if="filtered.length === 0"
      class="flex flex-col items-center justify-center py-20 rounded-xl"
      style="background: var(--bg-secondary); border: 1px dashed var(--border-color);"
    >
      <CheckCircle2 class="w-10 h-10 mb-3" style="color: #10B981;" />
      <p class="text-base font-medium mb-1" style="color: var(--text-primary);">没有告警</p>
      <p class="text-sm" style="color: var(--text-secondary);">系统运行正常</p>
    </div>

    <!-- Table -->
    <div v-else class="rounded-xl overflow-hidden" style="background: var(--bg-secondary); border: 1px solid var(--border-color);">
      <table class="w-full text-sm">
        <thead>
          <tr style="border-bottom: 1px solid var(--border-color);">
            <th class="text-left px-5 py-3 font-medium text-xs" style="color: var(--text-secondary);">级别</th>
            <th class="text-left px-4 py-3 font-medium text-xs" style="color: var(--text-secondary);">告警信息</th>
            <th class="text-left px-4 py-3 font-medium text-xs hidden md:table-cell" style="color: var(--text-secondary);">来源</th>
            <th class="text-left px-4 py-3 font-medium text-xs" style="color: var(--text-secondary);">状态</th>
            <th class="text-left px-4 py-3 font-medium text-xs hidden lg:table-cell" style="color: var(--text-secondary);">时间</th>
            <th class="text-right px-5 py-3 font-medium text-xs" style="color: var(--text-secondary);">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y" style="divide-color: var(--border-color);">
          <tr
            v-for="alert in filtered"
            :key="alert.id"
            class="transition-colors hover:bg-white/[0.02]"
            :style="alert.status === 'active' ? `border-left: 2px solid ${getLevelCfg(alert.level).color}` : ''"
          >
            <td class="px-5 py-3.5">
              <span
                class="inline-flex items-center px-2 py-0.5 rounded text-xs font-bold font-mono"
                :style="{ background: getLevelCfg(alert.level).bg, color: getLevelCfg(alert.level).color }"
              >
                {{ getLevelCfg(alert.level).label }}
              </span>
            </td>
            <td class="px-4 py-3.5">
              <p class="font-medium" style="color: var(--text-primary);">{{ alert.message }}</p>
              <p v-if="alert.value !== undefined" class="text-xs mt-0.5" style="color: var(--text-secondary);">
                值: <span class="font-mono" style="color: var(--text-secondary);">{{ String(alert.value) }}</span>
              </p>
            </td>
            <td class="px-4 py-3.5 hidden md:table-cell">
              <div class="text-xs space-y-0.5">
                <div class="font-mono" style="color: var(--accent-primary);">{{ alert.node_id }}</div>
                <div style="color: var(--text-secondary);">{{ alert.device_id || '—' }}</div>
              </div>
            </td>
            <td class="px-4 py-3.5">
              <StatusBadge :status="alert.status" size="sm" />
            </td>
            <td class="px-4 py-3.5 text-xs tabular-nums hidden lg:table-cell" style="color: var(--text-secondary);">
              {{ formatTime(alert.created_at) }}
            </td>
            <td class="px-5 py-3.5 text-right">
              <button
                v-if="alert.status === 'active'"
                @click="alertStore.acknowledge(alert.id)"
                class="flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-xs transition-colors hover:bg-green-500/10 ml-auto"
                style="color: #10B981;"
              >
                <CheckCircle2 class="w-3 h-3" style="width:12px;height:12px;" />
                确认
              </button>
              <span v-else class="text-xs" style="color: var(--text-muted);">—</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
