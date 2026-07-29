<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import {
  Zap, RefreshCw, Search, ArrowRight,
  Radio, Activity,
} from 'lucide-vue-next'
import { useEanStore } from '@/stores/ean'
import type { EANPointChangeEvent } from '@/types/ean'
import EanDisabledBanner from '@/components/ean/EanDisabledBanner.vue'

const eanStore = useEanStore()

const searchQuery = ref('')
const eventTypeFilter = ref<'all' | 'point_change' | 'device_online' | 'device_offline'>('all')
const eventCount = ref(100)

let pollTimer: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  await Promise.all([eanStore.fetchHealth(), eanStore.fetchEvents(eventCount.value)])
  pollTimer = setInterval(() => {
    eanStore.fetchEvents(eventCount.value)
  }, 5000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})

function classifyEvent(evt: EANPointChangeEvent): 'point_change' | 'device_online' | 'device_offline' {
  if (evt.event_type === 'device.online') return 'device_online'
  if (evt.event_type === 'device.offline') return 'device_offline'
  return 'point_change'
}

const filteredEvents = computed(() => {
  let result = eanStore.events
  if (eventTypeFilter.value !== 'all') {
    result = result.filter(e => classifyEvent(e) === eventTypeFilter.value)
  }
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase()
    result = result.filter(e =>
      e.agent_id.toLowerCase().includes(q) ||
      e.device_id.toLowerCase().includes(q) ||
      (e.point_id || '').toLowerCase().includes(q) ||
      e.event_type.toLowerCase().includes(q)
    )
  }
  return result
})

const eventStats = computed(() => {
  const total = eanStore.events.length
  const pointChange = eanStore.events.filter(e => classifyEvent(e) === 'point_change').length
  const deviceOnline = eanStore.events.filter(e => classifyEvent(e) === 'device_online').length
  const deviceOffline = eanStore.events.filter(e => classifyEvent(e) === 'device_offline').length
  return { total, pointChange, deviceOnline, deviceOffline }
})

function formatTime(ts: number) {
  if (!ts) return '—'
  return new Date(ts < 1e12 ? ts * 1000 : ts).toLocaleString('zh-CN', {
    month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit',
  })
}

function formatValue(val: unknown): string {
  if (val === null || val === undefined) return 'null'
  if (typeof val === 'object') return JSON.stringify(val)
  return String(val)
}

function hasChanged(evt: EANPointChangeEvent): boolean {
  return evt.previous_value !== null &&
         evt.previous_value !== undefined &&
         JSON.stringify(evt.previous_value) !== JSON.stringify(evt.value)
}
</script>

<template>
  <div class="space-y-5">
    <!-- 页头 / Page Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-bold" style="color: var(--text-primary);">事件流</h1>
        <p class="text-sm mt-1" style="color: var(--text-secondary);">点位变化事件 + previous_value + 设备上下线</p>
      </div>
      <button
        @click="eanStore.fetchEvents(eventCount)"
        class="flex items-center gap-2 px-4 py-2 rounded-xl text-sm transition-colors hover:bg-white/5"
        style="color: var(--text-secondary); border: 1px solid var(--border-color);"
      >
        <RefreshCw class="w-4 h-4" style="width:16px;height:16px;" />
        刷新
      </button>
    </div>

    <EanDisabledBanner v-if="eanStore.isEanDisabled" compact />

    <!-- 统计卡片 / Stats Cards -->
    <div class="grid grid-cols-2 xl:grid-cols-4 gap-4">
      <div class="rounded-lg border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
        <div class="flex items-center gap-2 mb-2">
          <Activity class="w-4 h-4" style="width:16px;height:16px;color:#0EA5E9;" />
          <span class="text-xs font-medium" style="color: var(--text-muted);">总事件</span>
        </div>
        <div class="text-2xl font-bold tabular-nums" style="color: var(--text-primary);">{{ eventStats.total }}</div>
      </div>
      <div class="rounded-lg border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
        <div class="flex items-center gap-2 mb-2">
          <Zap class="w-4 h-4" style="width:16px;height:16px;color:#6366F1;" />
          <span class="text-xs font-medium" style="color: var(--text-muted);">点位变化</span>
        </div>
        <div class="text-2xl font-bold tabular-nums" style="color: #6366F1;">{{ eventStats.pointChange }}</div>
      </div>
      <div class="rounded-lg border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
        <div class="flex items-center gap-2 mb-2">
          <Radio class="w-4 h-4" style="width:16px;height:16px;color:#10B981;" />
          <span class="text-xs font-medium" style="color: var(--text-muted);">设备上线</span>
        </div>
        <div class="text-2xl font-bold tabular-nums" style="color: #10B981;">{{ eventStats.deviceOnline }}</div>
      </div>
      <div class="rounded-lg border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
        <div class="flex items-center gap-2 mb-2">
          <Radio class="w-4 h-4" style="width:16px;height:16px;color:#EF4444;" />
          <span class="text-xs font-medium" style="color: var(--text-muted);">设备下线</span>
        </div>
        <div class="text-2xl font-bold tabular-nums" style="color: #EF4444;">{{ eventStats.deviceOffline }}</div>
      </div>
    </div>

    <!-- 筛选栏 / Filter Bar -->
    <div class="flex items-center gap-3">
      <div class="flex-1 relative">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4" style="width:16px;height:16px;color:var(--text-muted);" />
        <input
          v-model="searchQuery"
          placeholder="搜索 Agent / Device / Point / Event Type"
          class="w-full pl-10 pr-4 py-2 rounded-xl text-sm"
          style="background: var(--bg-secondary); border: 1px solid var(--border-color); color: var(--text-primary);"
        />
      </div>
      <div class="flex items-center gap-1 p-1 rounded-xl" style="background: var(--bg-secondary); border: 1px solid var(--border-color);">
        <button
          v-for="opt in [
            { v: 'all', l: '全部' },
            { v: 'point_change', l: '点位变化' },
            { v: 'device_online', l: '上线' },
            { v: 'device_offline', l: '下线' },
          ]"
          :key="opt.v"
          @click="eventTypeFilter = opt.v as 'all' | 'point_change' | 'device_online' | 'device_offline'"
          class="px-3 py-1.5 rounded-lg text-xs font-medium transition-colors whitespace-nowrap"
          :style="eventTypeFilter === opt.v
            ? 'background: var(--accent-primary); color: white;'
            : 'color: var(--text-secondary);'"
        >
          {{ opt.l }}
        </button>
      </div>
    </div>

    <!-- 事件列表 / Event List -->
    <div class="rounded-lg border overflow-hidden" style="background: var(--bg-secondary); border-color: var(--border-color);">
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr style="border-bottom: 1px solid var(--border-color);">
              <th class="text-left px-5 py-3 text-xs font-semibold uppercase tracking-wide" style="color: var(--text-muted);">时间</th>
              <th class="text-left px-5 py-3 text-xs font-semibold uppercase tracking-wide" style="color: var(--text-muted);">事件类型</th>
              <th class="text-left px-5 py-3 text-xs font-semibold uppercase tracking-wide" style="color: var(--text-muted);">Agent</th>
              <th class="text-left px-5 py-3 text-xs font-semibold uppercase tracking-wide" style="color: var(--text-muted);">设备 / 点位</th>
              <th class="text-left px-5 py-3 text-xs font-semibold uppercase tracking-wide" style="color: var(--text-muted);">previous_value</th>
              <th class="text-left px-5 py-3 text-xs font-semibold uppercase tracking-wide" style="color: var(--text-muted);">value</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="filteredEvents.length === 0">
              <td colspan="6" class="px-5 py-12 text-center" style="color: var(--text-secondary);">
                <div class="flex flex-col items-center gap-2">
                  <Zap class="w-8 h-8 opacity-30" style="width:32px;height:32px;" />
                  <span>暂无事件数据</span>
                </div>
              </td>
            </tr>
            <tr
              v-for="(evt, idx) in filteredEvents"
              :key="idx"
              class="transition-colors hover:bg-white/[0.02]"
              style="border-bottom: 1px solid var(--border-color);"
            >
              <td class="px-5 py-2.5 text-xs font-mono whitespace-nowrap" style="color: var(--text-muted);">{{ formatTime(evt.timestamp) }}</td>
              <td class="px-5 py-2.5">
                <span
                  class="text-xs font-mono px-2 py-0.5 rounded-lg"
                  :style="{
                    background: evt.event_type === 'device.online' ? 'rgba(16,185,129,0.1)' : evt.event_type === 'device.offline' ? 'rgba(239,68,68,0.1)' : 'rgba(14,165,233,0.1)',
                    color: evt.event_type === 'device.online' ? '#10B981' : evt.event_type === 'device.offline' ? '#EF4444' : '#0EA5E9',
                  }"
                >
                  {{ evt.event_type }}
                </span>
              </td>
              <td class="px-5 py-2.5 text-xs font-mono" style="color: var(--text-secondary);">{{ evt.agent_id }}</td>
              <td class="px-5 py-2.5">
                <div class="flex flex-col gap-0.5">
                  <span class="text-xs font-mono" style="color: var(--text-primary);">{{ evt.device_id }}</span>
                  <span v-if="evt.point_id" class="text-xs font-mono" style="color: var(--accent-primary);">{{ evt.point_id }}</span>
                </div>
              </td>
              <td class="px-5 py-2.5">
                <span
                  v-if="evt.previous_value !== null && evt.previous_value !== undefined"
                  class="text-xs font-mono"
                  :class="hasChanged(evt) ? '' : 'opacity-50'"
                  :style="{ color: hasChanged(evt) ? '#F59E0B' : 'var(--text-muted)' }"
                >
                  {{ formatValue(evt.previous_value) }}
                </span>
                <span v-else class="text-xs" style="color: var(--text-muted);">null</span>
              </td>
              <td class="px-5 py-2.5">
                <div class="flex items-center gap-2">
                  <ArrowRight v-if="hasChanged(evt)" class="w-3 h-3 flex-shrink-0" style="width:12px;height:12px;color:#10B981;" />
                  <span class="text-xs font-mono font-medium" style="color: var(--text-primary);">{{ formatValue(evt.value) }}</span>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
