<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ChevronRight, RefreshCw, Search, Edit3 } from 'lucide-vue-next'
import { useRealtimeStore } from '@/stores/realtime'
import { useEdgeStore } from '@/stores/edge'
import WritePointModal from '@/components/edge/WritePointModal.vue'
import StatusBadge from '@/components/edge/StatusBadge.vue'
import type { EdgeXPointInfo, WritePointRequest } from '@/types/edgex'
import { controlApi } from '@/api/index'

const route = useRoute()
const router = useRouter()
const rtStore = useRealtimeStore()
const edgeStore = useEdgeStore()

const nodeId = computed(() => route.params.nodeId as string)
const deviceId = computed(() => route.params.deviceId as string)
const node = computed(() => edgeStore.nodes.find(n => n.node_id === nodeId.value))
const deviceList = computed(() => (edgeStore.devicesByNode[nodeId.value] ?? []))
const device = computed(() => deviceList.value.find(d => d.device_id === deviceId.value))

const search = ref('')
const writeModalVisible = ref(false)
const writingPoint = ref<EdgeXPointInfo | null>(null)

const points = computed(() => rtStore.getPoints(nodeId.value, deviceId.value))
const loading = computed(() => rtStore.isLoading(nodeId.value, deviceId.value))

const filtered = computed(() => {
  if (!search.value) return points.value
  const q = search.value.toLowerCase()
  return points.value.filter(p =>
    p.point_id.toLowerCase().includes(q) || p.point_name.toLowerCase().includes(q)
  )
})

async function load() {
  if (edgeStore.nodes.length === 0) await edgeStore.fetchNodes()
  if (!deviceList.value.length) await edgeStore.fetchDevices(nodeId.value)
  await rtStore.fetchPoints(nodeId.value, deviceId.value)
}

onMounted(load)
watch([nodeId, deviceId], load)

function openWrite(point: EdgeXPointInfo) {
  writingPoint.value = point
  writeModalVisible.value = true
}

async function handleWrite(pointId: string, value: unknown) {
  const req: WritePointRequest = {
    node_id: nodeId.value,
    device_id: deviceId.value,
    point_id: pointId,
    value,
  }
  await controlApi.writePoint(req)
  writeModalVisible.value = false
}

function formatValue(v: unknown) {
  if (v === undefined || v === null) return '—'
  if (typeof v === 'boolean') return v ? 'TRUE' : 'FALSE'
  return String(v)
}

function formatTime(ts: number | undefined) {
  if (!ts) return '—'
  return new Date(ts < 1e12 ? ts * 1000 : ts).toLocaleTimeString('zh-CN', { hour12: false })
}
</script>

<template>
  <div class="space-y-5">
    <!-- Breadcrumb -->
    <div>
      <nav class="flex items-center gap-1.5 text-xs mb-2 flex-wrap nav-breadcrumb">
        <button @click="router.push('/nodes')" class="hover:text-sky-400 transition-colors">节点管理</button>
        <ChevronRight class="w-3 h-3 flex-shrink-0" style="width:12px;height:12px;" />
        <button @click="router.push(`/nodes/${nodeId}/devices`)" class="hover:text-sky-400 transition-colors breadcrumb-link">
          {{ node?.node_name || nodeId }}
        </button>
        <ChevronRight class="w-3 h-3 flex-shrink-0" style="width:12px;height:12px;" />
        <span class="breadcrumb-link">{{ device?.device_name || deviceId }}</span>
        <ChevronRight class="w-3 h-3 flex-shrink-0" style="width:12px;height:12px;" />
        <span style="color: var(--text-muted);">物模型点位</span>
      </nav>
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-xl font-bold page-title">物模型点位</h1>
          <p class="text-sm mt-1 page-subtitle">实时数据流 · 差量更新高亮 1.5s</p>
        </div>
        <button
          @click="load"
          class="btn-ghost flex items-center gap-2 px-4 py-2 rounded-xl text-sm transition-colors"
        >
          <RefreshCw class="w-4 h-4" :class="loading ? 'animate-spin' : ''" style="width:16px;height:16px;" />
          刷新
        </button>
      </div>
    </div>

    <!-- Stats bar -->
    <div class="flex items-center gap-4 text-xs stats-bar">
      <span>共 <span class="stat-value">{{ points.length }}</span> 个点位</span>
      <span>·</span>
      <span>可写 <span class="stat-accent">{{ points.filter(p => p.read_write).length }}</span> 个</span>
      <span>·</span>
      <span>有值 <span class="stat-success">{{ points.filter(p => p.current_value !== undefined && p.current_value !== null).length }}</span> 个</span>
    </div>

    <!-- Search -->
    <div class="relative">
      <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 search-icon" />
      <input
        v-model="search"
        type="text"
        placeholder="搜索点位 ID 或名称..."
        class="w-full rounded-xl pl-9 pr-4 py-2.5 text-sm outline-none search-input"
        @focus="($event.target as HTMLInputElement).style.borderColor='var(--accent-primary)'"
        @blur="($event.target as HTMLInputElement).style.borderColor='var(--border-color)'"
      />
    </div>

    <!-- Loading -->
    <div v-if="loading && points.length === 0" class="flex items-center justify-center py-16">
      <RefreshCw class="w-6 h-6 animate-spin loading-spinner" />
    </div>

    <!-- Table -->
    <div v-else class="rounded-xl overflow-hidden table-container">
      <table class="w-full text-sm">
        <thead>
          <tr class="table-header-row">
            <th class="text-left px-5 py-3 font-medium text-xs table-header-cell">点位 ID</th>
            <th class="text-left px-4 py-3 font-medium text-xs table-header-cell hidden md:table-cell">名称</th>
            <th class="text-left px-4 py-3 font-medium text-xs table-header-cell hidden lg:table-cell">类型</th>
            <th class="text-left px-4 py-3 font-medium text-xs table-header-cell">当前值</th>
            <th class="text-left px-4 py-3 font-medium text-xs table-header-cell hidden lg:table-cell">单位</th>
            <th class="text-left px-4 py-3 font-medium text-xs table-header-cell hidden xl:table-cell">质量</th>
            <th class="text-left px-4 py-3 font-medium text-xs table-header-cell hidden xl:table-cell">更新时间</th>
            <th class="text-right px-5 py-3 font-medium text-xs table-header-cell">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y" style="divide-color: var(--border-color);">
          <tr
            v-for="point in filtered"
            :key="point.point_id"
            class="transition-all duration-300 table-row"
            :class="rtStore.isDeltaKey(nodeId, deviceId, point.point_id) ? 'delta-highlight' : ''"
          >
            <td class="px-5 py-3">
              <div class="flex items-center">
                <span class="font-mono text-xs point-id">{{ point.point_id }}</span>
              </div>
            </td>
            <td class="px-4 py-3 hidden md:table-cell point-name">{{ point.point_name }}</td>
            <td class="px-4 py-3 hidden lg:table-cell">
              <span class="text-xs font-mono px-1.5 py-0.5 rounded type-badge">{{ point.data_type }}</span>
            </td>
            <td class="px-4 py-3">
              <span class="font-mono text-sm font-semibold tabular-nums point-value"
                :class="[
                  point.current_value !== undefined && point.current_value !== null ? 'has-value' : 'no-value',
                  rtStore.isDeltaKey(nodeId, deviceId, point.point_id) ? 'value-updated' : ''
                ]">
                {{ formatValue(point.current_value) }}
              </span>
            </td>
            <td class="px-4 py-3 text-xs hidden lg:table-cell point-unit">{{ point.units || '—' }}</td>
            <td class="px-4 py-3 hidden xl:table-cell">
              <span v-if="point.quality" class="font-mono text-xs">{{ point.quality }}</span>
              <span v-else class="text-muted">—</span>
            </td>
            <td class="px-4 py-3 text-xs tabular-nums hidden xl:table-cell point-time">{{ formatTime(point.last_updated) }}</td>
            <td class="px-5 py-3 text-right">
              <button
                v-if="point.read_write"
                @click.stop="openWrite(point)"
                class="flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-xs transition-colors btn-write ml-auto"
              >
                <Edit3 class="w-3 h-3" style="width:12px;height:12px;" />
                写入
              </button>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-if="filtered.length === 0 && points.length > 0" class="px-5 py-6 text-sm text-center empty-message">
        未找到匹配的点位
      </div>
      <div v-if="points.length === 0" class="px-5 py-10 text-sm text-center empty-message">
        暂无点位数据，等待设备上报物模型
      </div>
    </div>

    <!-- Write modal -->
    <WritePointModal
      :visible="writeModalVisible"
      :point="writingPoint"
      @close="writeModalVisible = false"
      @submit="handleWrite"
    />
  </div>
</template>

<style scoped>
.nav-breadcrumb { color: var(--text-secondary); }
.breadcrumb-link { color: var(--accent-primary); }
.page-title { color: var(--text-primary); }
.page-subtitle { color: var(--text-secondary); }
.stats-bar { color: var(--text-secondary); }
.stat-value { color: var(--text-primary); }
.stat-accent { color: var(--accent-primary); }
.stat-success { color: #10B981; }
.search-icon { color: var(--text-muted); }
.search-input {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
}
.search-input::placeholder { color: var(--text-muted); }
.loading-spinner { color: var(--accent-primary); }
.table-container {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
}
.table-header-row { border-bottom: 1px solid var(--border-color); }
.table-header-cell { color: var(--text-secondary); }
.table-row { background: transparent; }
.table-row:hover { background: var(--bg-hover); }
.delta-highlight { background: rgba(16,185,129,0.08); }
.point-id { color: var(--accent-primary); }
.point-name { color: var(--text-primary); }
.type-badge { background: rgba(99,102,241,0.1); color: #818CF8; }
.point-value { color: var(--text-primary); }
.point-value.no-value { color: var(--text-muted); }
.point-value.value-updated {
  color: #10B981;
  text-shadow: 0 0 8px rgba(16, 185, 129, 0.6);
  animation: glow 1.5s ease-in-out;
}

@keyframes glow {
  0% { text-shadow: 0 0 0 rgba(16, 185, 129, 0); }
  50% { text-shadow: 0 0 12px rgba(16, 185, 129, 0.8); }
  100% { text-shadow: 0 0 8px rgba(16, 185, 129, 0.6); }
}
.point-unit { color: var(--text-secondary); }
.point-time { color: var(--text-muted); }
.text-muted { color: var(--text-muted); }
.btn-write {
  color: var(--accent-primary);
  background: rgba(14,165,233,0.08);
  border: 1px solid rgba(14,165,233,0.15);
}
.btn-write:hover {
  background: rgba(14,165,233,0.15);
}
.empty-message { color: var(--text-muted); }

.btn-ghost {
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}
.btn-ghost:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}
</style>
