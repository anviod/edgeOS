<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { Server, Cpu, Radio, AlertTriangle, Activity, ChevronRight } from 'lucide-vue-next'
import { useEdgeStore } from '@/stores/edge'
import { useMiddlewareStore } from '@/stores/middleware'
import { useAlertStore } from '@/stores/alert'
import StatusBadge from '@/components/edge/StatusBadge.vue'

const router = useRouter()
const edgeStore = useEdgeStore()
const mwStore = useMiddlewareStore()
const alertStore = useAlertStore()

onMounted(async () => {
  await Promise.all([
    edgeStore.fetchStats(),
    edgeStore.fetchNodes(),
    mwStore.fetchList(),
    alertStore.fetchAlerts('active'),
  ])
})

const statCards = computed(() => [
  {
    label: '节点总数',
    value: edgeStore.stats.total_nodes,
    icon: Server,
    color: '#0EA5E9',
    bg: 'rgba(14,165,233,0.1)',
    sub: `${edgeStore.stats.online_nodes} 在线`,
    subColor: '#10B981',
  },
  {
    label: '在线节点',
    value: edgeStore.stats.online_nodes,
    icon: Activity,
    color: '#10B981',
    bg: 'rgba(16,185,129,0.1)',
    sub: edgeStore.stats.total_nodes > 0
      ? `${Math.round((edgeStore.stats.online_nodes / edgeStore.stats.total_nodes) * 100)}% 在线率`
      : '—',
    subColor: '#10B981',
  },
  {
    label: '设备总数',
    value: edgeStore.stats.total_devices,
    icon: Cpu,
    color: '#6366F1',
    bg: 'rgba(99,102,241,0.1)',
    sub: '已同步',
    subColor: '#9CA3AF',
  },
  {
    label: '今日告警',
    value: edgeStore.stats.today_alerts,
    icon: AlertTriangle,
    color: edgeStore.stats.today_alerts > 0 ? '#EF4444' : '#9CA3AF',
    bg: edgeStore.stats.today_alerts > 0 ? 'rgba(239,68,68,0.1)' : 'rgba(107,114,128,0.1)',
    sub: alertStore.unacknowledgedCount > 0 ? `${alertStore.unacknowledgedCount} 未确认` : '全部处理',
    subColor: alertStore.unacknowledgedCount > 0 ? '#EF4444' : '#10B981',
  },
])

function formatTime(ts: number) {
  if (!ts) return '—'
  return new Date(ts < 1e12 ? ts * 1000 : ts).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}
</script>

<template>
  <div class="space-y-6">
    <!-- Page header -->
    <div>
      <h1 class="text-xl font-bold" style="color: var(--text-primary);">系统总览</h1>
      <p class="text-sm mt-1" style="color: var(--text-secondary);">实时监控边缘平台运行状态</p>
    </div>

    <!-- Stat cards -->
    <div class="grid grid-cols-2 xl:grid-cols-4 gap-4">
      <div
        v-for="card in statCards"
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
          <div class="text-xs mt-1" :style="{ color: card.subColor }">{{ card.sub }}</div>
        </div>
      </div>
    </div>

    <!-- Bottom 2-col -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
      <!-- Middleware status -->
      <div class="rounded-lg border overflow-hidden" style="background: var(--bg-secondary); border-color: var(--border-color);">
        <div class="flex items-center justify-between px-5 py-3.5 border-b" style="border-color: var(--border-color);">
          <div class="flex items-center gap-2">
            <Radio class="w-4 h-4" style="color: var(--accent-primary); width:16px;height:16px;" />
            <span class="text-sm font-semibold" style="color: var(--text-primary);">中间件连接</span>
          </div>
          <button @click="router.push('/middleware')" class="flex items-center gap-1 text-xs transition-colors hover:text-sky-400" style="color: var(--text-secondary);">
            管理 <ChevronRight class="w-3 h-3" style="width:12px;height:12px;" />
          </button>
        </div>
        <div class="divide-y" style="divide-color: var(--border-color);">
          <div v-if="mwStore.list.length === 0" class="px-5 py-6 text-sm text-center" style="color: var(--text-secondary);">
            暂无中间件，<button @click="router.push('/middleware')" style="color: var(--accent-primary);">立即添加</button>
          </div>
          <div
            v-for="mw in mwStore.list"
            :key="mw.id"
            class="flex items-center justify-between px-5 py-3"
          >
            <div class="flex items-center gap-2 min-w-0">
              <span class="text-sm font-medium truncate" style="color: var(--text-primary);">{{ mw.name }}</span>
              <span class="text-xs px-1.5 py-0.5 rounded-lg uppercase font-mono" style="background: rgba(14,165,233,0.1); color: var(--accent-primary); border: 1px solid rgba(14,165,233,0.2);">{{ mw.type }}</span>
            </div>
            <StatusBadge :status="mw.status" size="sm" />
          </div>
        </div>
      </div>

      <!-- Recent alerts -->
      <div class="rounded-lg border overflow-hidden" style="background: var(--bg-secondary); border-color: var(--border-color);">
        <div class="flex items-center justify-between px-5 py-3.5 border-b" style="border-color: var(--border-color);">
          <div class="flex items-center gap-2">
            <AlertTriangle class="w-4 h-4" style="color: var(--destructive); width:16px;height:16px;" />
            <span class="text-sm font-semibold" style="color: var(--text-primary);">最近告警</span>
            <span v-if="alertStore.unacknowledgedCount > 0"
              class="text-xs px-1.5 py-0.5 rounded-lg font-bold" style="background: rgba(239,68,68,0.1); color: var(--destructive); border: 1px solid rgba(239,68,68,0.2);">
              {{ alertStore.unacknowledgedCount }}
            </span>
          </div>
          <button @click="router.push('/alerts')" class="flex items-center gap-1 text-xs transition-colors hover:text-sky-400" style="color: var(--text-secondary);">
            全部 <ChevronRight class="w-3 h-3" style="width:12px;height:12px;" />
          </button>
        </div>
        <div class="divide-y" style="divide-color: var(--border-color);">
          <div v-if="alertStore.alerts.length === 0" class="px-5 py-6 text-sm text-center" style="color: var(--text-secondary);">暂无活跃告警</div>
          <div
            v-for="alert in alertStore.alerts.slice(0, 5)"
            :key="alert.id"
            class="flex items-start gap-3 px-5 py-3"
          >
            <span
              class="w-1.5 h-1.5 rounded-full mt-1.5 flex-shrink-0"
              :class="{
                'bg-red-500': alert.level === 'critical',
                'bg-amber-500': alert.level === 'major',
                'bg-amber-400': alert.level === 'minor',
                'bg-slate-400': !alert.level || alert.level === 'info',
              }"
            />
            <div class="flex-1 min-w-0">
              <p class="text-sm truncate" style="color: var(--text-primary);">{{ alert.message }}</p>
              <p class="text-xs mt-0.5" style="color: var(--text-secondary);">{{ alert.device_id || alert.node_id }} · {{ formatTime(alert.created_at) }}</p>
            </div>
            <StatusBadge :status="alert.level" size="sm" />
          </div>
        </div>
      </div>
    </div>

    <!-- Nodes quick list -->
    <div class="rounded-lg border overflow-hidden" style="background: var(--bg-secondary); border-color: var(--border-color);">
      <div class="flex items-center justify-between px-5 py-3.5 border-b" style="border-color: var(--border-color);">
        <div class="flex items-center gap-2">
          <Server class="w-4 h-4" style="color: var(--accent-primary); width:16px;height:16px;" />
          <span class="text-sm font-semibold" style="color: var(--text-primary);">节点状态</span>
        </div>
        <button @click="router.push('/nodes')" class="flex items-center gap-1 text-xs transition-colors hover:text-sky-400" style="color: var(--text-secondary);">
          全部节点 <ChevronRight class="w-3 h-3" style="width:12px;height:12px;" />
        </button>
      </div>
      <div v-if="edgeStore.nodes.length === 0" class="px-5 py-6 text-sm text-center" style="color: var(--text-secondary);">暂无已注册节点</div>
      <table v-else class="w-full text-sm">
        <thead>
          <tr style="border-bottom: 1px solid var(--border-color); background: var(--bg-primary);">
            <th class="text-left px-5 py-2.5 font-medium text-xs" style="color: var(--text-secondary);">节点 ID</th>
            <th class="text-left px-4 py-2.5 font-medium text-xs" style="color: var(--text-secondary);">名称</th>
            <th class="text-left px-4 py-2.5 font-medium text-xs" style="color: var(--text-secondary);">协议</th>
            <th class="text-left px-4 py-2.5 font-medium text-xs" style="color: var(--text-secondary);">状态</th>
            <th class="text-left px-4 py-2.5 font-medium text-xs" style="color: var(--text-secondary);">最后心跳</th>
          </tr>
        </thead>
        <tbody class="divide-y" style="divide-color: var(--border-color);">
          <tr
            v-for="node in edgeStore.nodes.slice(0, 5)"
            :key="node.node_id"
            class="cursor-pointer transition-colors hover:bg-white/[0.02]"
            @click="router.push(`/nodes/${node.node_id}/devices`)"
          >
            <td class="px-5 py-3 font-mono text-xs" style="color: var(--accent-primary);">{{ node.node_id }}</td>
            <td class="px-4 py-3" style="color: var(--text-primary);">{{ node.node_name }}</td>
            <td class="px-4 py-3 font-mono text-xs" style="color: var(--text-secondary);">{{ node.protocol }}</td>
            <td class="px-4 py-3"><StatusBadge :status="node.status" size="sm" /></td>
            <td class="px-4 py-3 text-xs tabular-nums" style="color: var(--text-secondary);">{{ formatTime(node.last_seen) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
