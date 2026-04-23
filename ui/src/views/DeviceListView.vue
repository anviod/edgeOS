<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Cpu, RefreshCw, ChevronRight, Search } from 'lucide-vue-next'
import { useEdgeStore } from '@/stores/edge'
import StatusBadge from '@/components/edge/StatusBadge.vue'

const route = useRoute()
const router = useRouter()
const edgeStore = useEdgeStore()

const nodeId = computed(() => route.params.nodeId as string)
const node = computed(() => edgeStore.nodes.find(n => n.node_id === nodeId.value))
const devices = computed(() => edgeStore.devicesByNode[nodeId.value] ?? [])

const search = ref('')
const newDeviceIds = ref<Set<string>>(new Set())

const filtered = computed(() => {
  if (!search.value) return devices.value
  const q = search.value.toLowerCase()
  return devices.value.filter(d =>
    d.device_id.toLowerCase().includes(q) || d.device_name.toLowerCase().includes(q)
  )
})

async function load() {
  if (edgeStore.nodes.length === 0) await edgeStore.fetchNodes()
  const prev = new Set((edgeStore.devicesByNode[nodeId.value] ?? []).map(d => d.device_id))
  await edgeStore.fetchDevices(nodeId.value)
  // Highlight new devices
  devices.value.forEach(d => {
    if (!prev.has(d.device_id)) {
      newDeviceIds.value.add(d.device_id)
      setTimeout(() => newDeviceIds.value.delete(d.device_id), 2000)
    }
  })
}

onMounted(load)
watch(nodeId, load)

async function triggerDiscover() {
  const { nodeApi } = await import('@/api/index')
  await nodeApi.triggerDiscover(nodeId.value)
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
    <!-- Breadcrumb + header -->
    <div>
      <nav class="flex items-center gap-1.5 text-xs mb-2 nav-breadcrumb">
        <button @click="router.push('/nodes')" class="hover:text-sky-400 transition-colors">节点管理</button>
        <ChevronRight class="w-3 h-3" style="width:12px;height:12px;" />
        <span style="color: var(--accent-primary);">{{ node?.node_name || nodeId }}</span>
        <ChevronRight class="w-3 h-3" style="width:12px;height:12px;" />
        <span style="color: var(--text-secondary);">子设备</span>
      </nav>
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-xl font-bold page-title">子设备列表</h1>
          <p class="text-sm mt-1 page-subtitle">
            节点 <span class="font-mono" style="color: var(--accent-primary);">{{ nodeId }}</span> 下挂载的设备
          </p>
        </div>
        <div class="flex items-center gap-2">
          <button
            @click="triggerDiscover"
            class="btn-secondary flex items-center gap-2 px-4 py-2 rounded-xl text-sm transition-colors"
          >
            <Cpu class="w-4 h-4" style="width:16px;height:16px;" />
            触发发现
          </button>
          <button
            @click="load"
            class="btn-ghost flex items-center gap-2 px-4 py-2 rounded-xl text-sm transition-colors"
          >
            <RefreshCw class="w-4 h-4" :class="edgeStore.devicesLoading ? 'animate-spin' : ''" style="width:16px;height:16px;" />
            刷新
          </button>
        </div>
      </div>
    </div>

    <!-- Search -->
    <div class="relative">
      <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 search-icon" />
      <input
        v-model="search"
        type="text"
        placeholder="搜索设备 ID 或名称..."
        class="w-full rounded-xl pl-9 pr-4 py-2.5 text-sm outline-none search-input"
        @focus="($event.target as HTMLInputElement).style.borderColor='var(--accent-primary)'"
        @blur="($event.target as HTMLInputElement).style.borderColor='var(--border-color)'"
      />
    </div>

    <!-- Loading -->
    <div v-if="edgeStore.devicesLoading && devices.length === 0" class="flex items-center justify-center py-16">
      <RefreshCw class="w-6 h-6 animate-spin" style="color: var(--accent-primary);" />
    </div>

    <!-- Empty -->
    <div v-else-if="devices.length === 0"
      class="flex flex-col items-center justify-center py-20 rounded-xl empty-state"
    >
      <Cpu class="w-10 h-10 mb-3" style="color: var(--text-muted);" />
      <p class="text-base font-medium mb-1" style="color: var(--text-primary);">尚无设备</p>
      <p class="text-sm mb-4" style="color: var(--text-secondary);">点击「触发发现」或等待设备主动上报</p>
      <button @click="triggerDiscover" class="btn-secondary px-4 py-2 rounded-xl text-sm">
        触发设备发现
      </button>
    </div>

    <!-- Table -->
    <div v-else class="rounded-xl overflow-hidden table-container">
      <table class="w-full text-sm">
        <thead>
          <tr class="table-header-row">
            <th class="text-left px-5 py-3 font-medium text-xs table-header-cell">设备 ID</th>
            <th class="text-left px-4 py-3 font-medium text-xs table-header-cell">名称</th>
            <th class="text-left px-4 py-3 font-medium text-xs table-header-cell hidden md:table-cell">服务</th>
            <th class="text-left px-4 py-3 font-medium text-xs table-header-cell hidden lg:table-cell">Profile</th>
            <th class="text-left px-4 py-3 font-medium text-xs table-header-cell">管理状态</th>
            <th class="text-left px-4 py-3 font-medium text-xs table-header-cell hidden xl:table-cell">最后同步</th>
            <th class="text-right px-5 py-3 font-medium text-xs table-header-cell">物模型</th>
          </tr>
        </thead>
        <tbody class="divide-y" style="divide-color: var(--border-color);">
          <tr
            v-for="device in filtered"
            :key="device.device_id"
            class="group cursor-pointer transition-all duration-300 table-row"
            :class="newDeviceIds.has(device.device_id) ? 'bg-sky-500/10' : 'hover:bg-white/[0.02]'"
            @click="router.push(`/nodes/${nodeId}/devices/${device.device_id}/points`)"
          >
            <td class="px-5 py-3.5">
              <span class="font-mono text-xs" style="color: var(--accent-primary);">{{ device.device_id }}</span>
            </td>
            <td class="px-4 py-3.5 font-medium" style="color: var(--text-primary);">{{ device.device_name }}</td>
            <td class="px-4 py-3.5 text-xs hidden md:table-cell" style="color: var(--text-secondary);">{{ device.service_name || '—' }}</td>
            <td class="px-4 py-3.5 text-xs font-mono hidden lg:table-cell" style="color: var(--text-muted);">{{ device.device_profile || '—' }}</td>
            <td class="px-4 py-3.5">
              <StatusBadge
                :status="device.operating_state"
                size="sm"
              />
            </td>
            <td class="px-4 py-3.5 text-xs tabular-nums hidden xl:table-cell" style="color: var(--text-muted);">{{ formatTime(device.last_sync) }}</td>
            <td class="px-5 py-3.5 text-right">
              <span class="flex items-center justify-end gap-1 text-xs opacity-0 group-hover:opacity-100 transition-opacity action-link" style="color: var(--accent-primary);">
                点位
                <ChevronRight class="w-3 h-3" style="width:12px;height:12px;" />
              </span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.nav-breadcrumb { color: var(--text-secondary); }
.page-title { color: var(--text-primary); }
.page-subtitle { color: var(--text-secondary); }
.search-icon { color: var(--text-muted); }
.search-input {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
}
.search-input::placeholder { color: var(--text-muted); }
.empty-state {
  background: var(--bg-secondary);
  border: 1px dashed var(--border-color);
}
.table-container {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
}
.table-header-row { border-bottom: 1px solid var(--border-color); }
.table-header-cell { color: var(--text-secondary); }
.table-row { background: transparent; }
.table-row:hover { background: var(--bg-hover); }
.action-link:hover { opacity: 0.8; }

.btn-secondary {
  background: rgba(14,165,233,0.08);
  color: var(--accent-primary);
  border: 1px solid rgba(14,165,233,0.2);
}
.btn-secondary:hover {
  background: rgba(14,165,233,0.15);
}
.btn-ghost {
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}
.btn-ghost:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}
</style>
