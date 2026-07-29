<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import {
  Network, Server, Activity, Shield, Radio, Zap,
  ChevronRight, RefreshCw, Cpu, CheckCircle,
} from 'lucide-vue-next'
import { useEanStore } from '@/stores/ean'
import StatusBadge from '@/components/edge/StatusBadge.vue'
import EanDisabledBanner from '@/components/ean/EanDisabledBanner.vue'

const router = useRouter()
const eanStore = useEanStore()

let pollTimer: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  await eanStore.fetchAll()
  // 轮询健康状态 - Poll health status every 10s
  pollTimer = setInterval(() => {
    eanStore.fetchHealth()
  }, 10000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})

const healthCards = computed(() => {
  const h = eanStore.health
  const m = h?.invoke_metrics
  const successPct = m ? Math.round((m.success_rate || 0) * 100) : 0
  const details = h?.transport_details || []
  const connected = details.filter(d => d.connected).length
  const registered = h?.registered_transports ?? details.length
  const detailSub = details.length
    ? details.map(d => `${d.name}${d.connected ? '✓' : '○'}`).join(' · ')
    : (h?.transports?.join(', ') || (registered ? '重连中' : '未连接'))
  return [
    {
      label: '传输层',
      value: `${connected}/${registered || connected}`,
      icon: Radio,
      color: '#0EA5E9',
      bg: 'rgba(14,165,233,0.1)',
      sub: detailSub,
      subColor: '#9CA3AF',
    },
    {
      label: '在线 Agent',
      value: h?.online_agents ?? 0,
      icon: Server,
      color: '#10B981',
      bg: 'rgba(16,185,129,0.1)',
      sub: `原生 Cap ${h?.native_ean_caps ?? 0}`,
      subColor: '#9CA3AF',
    },
    {
      label: 'Invoke 成功率',
      value: `${successPct}%`,
      icon: Activity,
      color: '#6366F1',
      bg: 'rgba(99,102,241,0.1)',
      sub: m
        ? `P50 ${m.p50_latency_ms ?? 0}ms · P99 ${m.p99_latency_ms ?? 0}ms`
        : '暂无调用',
      subColor: '#9CA3AF',
    },
    {
      label: '审计记录',
      value: h?.audit_count ?? 0,
      icon: Shield,
      color: '#F59E0B',
      bg: 'rgba(245,158,11,0.1)',
      sub: `${h?.pending_invokes ?? 0} 待响应 · 心跳 ${h?.tracked_agents ?? 0}`,
      subColor: '#9CA3AF',
    },
  ]
})

const transportDetails = computed(() => eanStore.health?.transport_details || [])

const recentEvents = computed(() => eanStore.events.slice(0, 8))

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
</script>

<template>
  <div class="space-y-6">
    <!-- 页头 / Page Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-bold" style="color: var(--text-primary);">EAN 协调中心</h1>
        <p class="text-sm mt-1" style="color: var(--text-secondary);">Edge Agent Network 2.0 发现、编排、订阅与治理</p>
      </div>
      <div class="flex items-center gap-2">
        <button
          @click="router.push('/ean/debug')"
          class="flex items-center gap-2 px-4 py-2 rounded-xl text-sm transition-colors hover:bg-white/5"
          style="color: var(--text-secondary); border: 1px solid var(--border-color);"
        >
          联调帮助
        </button>
        <StatusBadge v-if="eanStore.health" :status="eanStore.isEanEnabled ? 'online' : 'offline'" size="md" />
        <button
          @click="eanStore.fetchAll()"
          class="flex items-center gap-2 px-4 py-2 rounded-xl text-sm transition-colors hover:bg-white/5"
          style="color: var(--text-secondary); border: 1px solid var(--border-color);"
        >
          <RefreshCw class="w-4 h-4" :class="eanStore.loading ? 'animate-spin' : ''" style="width:16px;height:16px;" />
          刷新
        </button>
      </div>
    </div>

    <!-- EAN 未启用提示 / EAN Disabled Notice -->
    <EanDisabledBanner v-if="eanStore.isEanDisabled" />

    <!-- 健康状态卡片 / Health Stat Cards -->
    <div class="grid grid-cols-2 xl:grid-cols-4 gap-4">
      <div
        v-for="card in healthCards"
        :key="card.label"
        class="rounded-lg border transition-all duration-200 hover:scale-[1.01]"
        style="background: var(--bg-secondary); border-color: var(--border-color);"
      >
        <div class="flex items-start justify-between mb-4 p-5">
          <div class="w-10 h-10 rounded-lg flex items-center justify-center" :style="{ background: card.bg }">
            <component :is="card.icon" class="w-5 h-5" :style="{ color: card.color, width:'20px', height:'20px' }" />
          </div>
        </div>
        <div class="p-5 -mt-4">
          <div class="text-3xl font-bold tabular-nums mb-1" :style="{ color: card.color }">{{ card.value }}</div>
          <div class="text-sm" style="color: var(--text-secondary);">{{ card.label }}</div>
          <div class="text-xs mt-1 truncate" :style="{ color: card.subColor }">{{ card.sub }}</div>
        </div>
      </div>
    </div>

    <!-- 双传输明细 / Transport Details -->
    <div
      v-if="transportDetails.length"
      class="rounded-lg border overflow-hidden"
      style="background: var(--bg-secondary); border-color: var(--border-color);"
    >
      <div class="flex items-center justify-between px-5 py-3.5 border-b" style="border-color: var(--border-color);">
        <div class="flex items-center gap-2">
          <Radio class="w-4 h-4" style="color: var(--accent-primary); width:16px;height:16px;" />
          <span class="text-sm font-semibold" style="color: var(--text-primary);">传输状态</span>
          <span class="text-xs font-mono" style="color: var(--text-muted);">{{ eanStore.health?.northbound_runtime || '—' }}</span>
        </div>
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 divide-y sm:divide-y-0 sm:divide-x" style="divide-color: var(--border-color);">
        <div
          v-for="td in transportDetails"
          :key="td.name"
          class="px-5 py-4 flex items-center justify-between gap-3"
        >
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <span class="text-sm font-semibold uppercase font-mono" style="color: var(--text-primary);">{{ td.name }}</span>
              <StatusBadge :status="td.connected ? 'online' : 'offline'" size="sm" />
            </div>
            <p class="text-xs font-mono mt-1 truncate" style="color: var(--text-muted);">{{ td.endpoint || '—' }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 双行布局 / Two Column Layout -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
      <!-- Agent 列表 / Agent List -->
      <div class="rounded-lg border overflow-hidden" style="background: var(--bg-secondary); border-color: var(--border-color);">
        <div class="flex items-center justify-between px-5 py-3.5 border-b" style="border-color: var(--border-color);">
          <div class="flex items-center gap-2">
            <Network class="w-4 h-4" style="color: var(--accent-primary); width:16px;height:16px;" />
            <span class="text-sm font-semibold" style="color: var(--text-primary);">Agent 状态</span>
          </div>
          <button @click="router.push('/ean/agents')" class="flex items-center gap-1 text-xs transition-colors hover:text-sky-400" style="color: var(--text-secondary);">
            全部 <ChevronRight class="w-3 h-3" style="width:12px;height:12px;" />
          </button>
        </div>
        <div class="divide-y" style="divide-color: var(--border-color);">
          <div v-if="eanStore.agents.length === 0" class="px-5 py-6 text-sm text-center" style="color: var(--text-secondary);">
            暂无注册 Agent
          </div>
          <div
            v-for="agent in eanStore.agents.slice(0, 6)"
            :key="agent.id"
            class="flex items-center justify-between px-5 py-3"
          >
            <div class="flex items-center gap-3 min-w-0">
              <div class="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0" style="background: rgba(14,165,233,0.08);">
                <Cpu class="w-4 h-4" style="width:16px;height:16px;color:var(--accent-primary);" />
              </div>
              <div class="min-w-0">
                <p class="text-sm font-medium truncate" style="color: var(--text-primary);">{{ agent.id }}</p>
                <p class="text-xs" style="color: var(--text-muted);">{{ agent.kind }} · v{{ agent.version }}</p>
              </div>
            </div>
            <div class="flex items-center gap-2 flex-shrink-0">
              <span class="text-xs font-mono" style="color: var(--text-muted);">{{ (agent.transport || []).join('/') || '—' }}</span>
              <StatusBadge :status="agent.status" size="sm" />
            </div>
          </div>
        </div>
      </div>

      <!-- 最近事件 / Recent Events -->
      <div class="rounded-lg border overflow-hidden" style="background: var(--bg-secondary); border-color: var(--border-color);">
        <div class="flex items-center justify-between px-5 py-3.5 border-b" style="border-color: var(--border-color);">
          <div class="flex items-center gap-2">
            <Zap class="w-4 h-4" style="color: var(--accent-primary); width:16px;height:16px;" />
            <span class="text-sm font-semibold" style="color: var(--text-primary);">事件流</span>
            <span class="text-xs px-1.5 py-0.5 rounded-lg font-mono" style="background: rgba(99,102,241,0.1); color: #a5b4fc; border: 1px solid rgba(99,102,241,0.2);">{{ eanStore.events.length }}</span>
          </div>
          <button @click="router.push('/ean/events')" class="flex items-center gap-1 text-xs transition-colors hover:text-sky-400" style="color: var(--text-secondary);">
            全部 <ChevronRight class="w-3 h-3" style="width:12px;height:12px;" />
          </button>
        </div>
        <div class="divide-y" style="divide-color: var(--border-color);">
          <div v-if="recentEvents.length === 0" class="px-5 py-6 text-sm text-center" style="color: var(--text-secondary);">
            暂无事件
          </div>
          <div
            v-for="(evt, idx) in recentEvents"
            :key="idx"
            class="px-5 py-2.5"
          >
            <div class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-2 min-w-0">
                <span
                  class="w-1.5 h-1.5 rounded-full flex-shrink-0"
                  :style="{ background: evt.event_type === 'device.online' ? '#10B981' : evt.event_type === 'device.offline' ? '#EF4444' : '#0EA5E9' }"
                />
                <span class="text-xs font-mono truncate" style="color: var(--text-primary);">{{ evt.point_id || evt.device_id }}</span>
              </div>
              <span class="text-xs flex-shrink-0" style="color: var(--text-muted);">{{ formatTime(evt.timestamp) }}</span>
            </div>
            <div class="flex items-center gap-2 mt-1 pl-3.5">
              <span class="text-xs" style="color: var(--text-muted);">{{ evt.event_type }}</span>
              <span class="text-xs font-mono" style="color: var(--text-secondary);">{{ formatValue(evt.value) }}</span>
              <span v-if="evt.previous_value !== null && evt.previous_value !== undefined" class="text-xs" style="color: var(--text-muted);">
                ← {{ formatValue(evt.previous_value) }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 审计记录 / Audit Records -->
    <div class="rounded-lg border overflow-hidden" style="background: var(--bg-secondary); border-color: var(--border-color);">
      <div class="flex items-center justify-between px-5 py-3.5 border-b" style="border-color: var(--border-color);">
        <div class="flex items-center gap-2">
          <Shield class="w-4 h-4" style="color: var(--accent-primary); width:16px;height:16px;" />
          <span class="text-sm font-semibold" style="color: var(--text-primary);">审计日志</span>
          <span class="text-xs px-1.5 py-0.5 rounded-lg font-mono" style="background: rgba(245,158,11,0.1); color: #F59E0B; border: 1px solid rgba(245,158,11,0.2);">{{ eanStore.auditRecords.length }}</span>
        </div>
        <button @click="router.push('/ean/invoke')" class="flex items-center gap-1 text-xs transition-colors hover:text-sky-400" style="color: var(--text-secondary);">
          调用 <ChevronRight class="w-3 h-3" style="width:12px;height:12px;" />
        </button>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr style="border-bottom: 1px solid var(--border-color);">
              <th class="text-left px-5 py-2.5 text-xs font-semibold uppercase tracking-wide" style="color: var(--text-muted);">时间</th>
              <th class="text-left px-5 py-2.5 text-xs font-semibold uppercase tracking-wide" style="color: var(--text-muted);">发起方</th>
              <th class="text-left px-5 py-2.5 text-xs font-semibold uppercase tracking-wide" style="color: var(--text-muted);">目标</th>
              <th class="text-left px-5 py-2.5 text-xs font-semibold uppercase tracking-wide" style="color: var(--text-muted);">能力</th>
              <th class="text-left px-5 py-2.5 text-xs font-semibold uppercase tracking-wide" style="color: var(--text-muted);">状态</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="eanStore.auditRecords.length === 0">
              <td colspan="5" class="px-5 py-6 text-center" style="color: var(--text-secondary);">暂无审计记录</td>
            </tr>
            <tr
              v-for="record in eanStore.auditRecords.slice(0, 8)"
              :key="record.id"
              style="border-bottom: 1px solid var(--border-color);"
            >
              <td class="px-5 py-2.5 text-xs font-mono" style="color: var(--text-muted);">{{ formatTime(record.timestamp) }}</td>
              <td class="px-5 py-2.5 text-xs font-mono" style="color: var(--text-secondary);">{{ record.initiator }}</td>
              <td class="px-5 py-2.5 text-xs font-mono" style="color: var(--text-secondary);">{{ record.target }}</td>
              <td class="px-5 py-2.5 text-xs font-mono" style="color: var(--accent-primary);">{{ record.capability || '—' }}</td>
              <td class="px-5 py-2.5">
                <span class="inline-flex items-center gap-1">
                  <CheckCircle v-if="record.status === 'success'" class="w-3.5 h-3.5" style="width:14px;height:14px;color:#10B981;" />
                  <AlertCircle v-else-if="record.status === 'error'" class="w-3.5 h-3.5" style="width:14px;height:14px;color:#EF4444;" />
                  <RefreshCw v-else class="w-3.5 h-3.5" style="width:14px;height:14px;color:#F59E0B;" />
                  <span class="text-xs" :style="{ color: record.status === 'success' ? '#10B981' : record.status === 'error' ? '#EF4444' : '#F59E0B' }">{{ record.status }}</span>
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
