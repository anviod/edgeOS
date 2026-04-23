<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ChevronDown, Send } from 'lucide-vue-next'
import { useEdgeStore } from '@/stores/edge'
import { useRealtimeStore } from '@/stores/realtime'
import { controlApi } from '@/api/index'
import CommandLogPanel from '@/components/edge/CommandLogPanel.vue'
import WritePointModal from '@/components/edge/WritePointModal.vue'
import StatusBadge from '@/components/edge/StatusBadge.vue'
import type { EdgeXPointInfo, CommandRecord, WritePointRequest } from '@/types/edgex'

const edgeStore = useEdgeStore()
const rtStore = useRealtimeStore()

const selectedNodeId = ref('')
const selectedDeviceId = ref('')

// Load nodes on mount
onMounted(async () => {
  await edgeStore.fetchNodes()
  if (edgeStore.nodes.length > 0) {
    selectedNodeId.value = edgeStore.nodes[0].node_id
  }
  // 加载命令记录
  await loadCommands()
})

// 加载命令记录
async function loadCommands() {
  const cmds = await controlApi.listCommands()
  if (cmds && cmds.length > 0) {
    // 清空现有记录
    rtStore.cmdTracks.splice(0)
    // 添加命令记录到追踪列表
    cmds.forEach(cmd => {
      rtStore.addCmdTrack({
        request_id: cmd.id,
        node_id: cmd.node_id,
        device_id: cmd.device_id,
        point_id: cmd.point_id,
        value: cmd.value,
        status: cmd.status as any,
        error: cmd.error,
        ts: cmd.created_at * 1000,
      })
    })
  }
}

// 清空记录
async function handleClear() {
  try {
    await controlApi.clearCommands()
    rtStore.cmdTracks.splice(0)
  } catch (error) {
    console.error('清空记录失败:', error)
  }
}

// When node selected, load devices
watch(selectedNodeId, async (nid) => {
  selectedDeviceId.value = ''
  if (nid) await edgeStore.fetchDevices(nid)
})

// When device selected, load points
watch(selectedDeviceId, async (did) => {
  if (did && selectedNodeId.value) {
    await rtStore.fetchPoints(selectedNodeId.value, did)
  }
})

const devices = computed(() => edgeStore.devicesByNode[selectedNodeId.value] ?? [])
const allPoints = computed(() => {
  if (!selectedNodeId.value || !selectedDeviceId.value) return []
  return rtStore.getPoints(selectedNodeId.value, selectedDeviceId.value)
})
const writablePoints = computed(() => allPoints.value.filter(p => p.read_write))

// Command log from realtime store (cast to CommandRecord shape)
const commands = computed<CommandRecord[]>(() =>
  rtStore.cmdTracks.map(t => ({
    id: t.request_id,
    node_id: t.node_id,
    device_id: t.device_id,
    point_id: t.point_id,
    value: t.value,
    status: t.status,
    error: t.error,
    created_at: t.ts,
    updated_at: t.ts,
  }))
)

// Write modal
const writeModalVisible = ref(false)
const writingPoint = ref<EdgeXPointInfo | null>(null)

function openWrite(point: EdgeXPointInfo) {
  writingPoint.value = point
  writeModalVisible.value = true
}

async function handleWrite(pointId: string, value: unknown) {
  const req: WritePointRequest = {
    node_id: selectedNodeId.value,
    device_id: selectedDeviceId.value,
    point_id: pointId,
    value,
  }
  await controlApi.writePoint(req)
  writeModalVisible.value = false
}

async function handleRetry(cmd: CommandRecord) {
  const req: WritePointRequest = {
    node_id: cmd.node_id,
    device_id: cmd.device_id,
    point_id: cmd.point_id,
    value: cmd.value,
  }
  const newCmd = await controlApi.writePoint(req)
  rtStore.addCmdTrack({
    request_id: newCmd.id,
    node_id: req.node_id,
    device_id: req.device_id,
    point_id: req.point_id,
    value: req.value,
    status: 'pending',
    ts: Date.now(),
  })
}

function formatValue(v: unknown) {
  if (v === undefined || v === null) return '—'
  if (typeof v === 'boolean') return v ? 'TRUE' : 'FALSE'
  return String(v)
}
</script>

<template>
  <div class="space-y-5">
    <!-- Header -->
    <div>
      <h1 class="text-xl font-bold" style="color: var(--text-primary);">设备控制</h1>
      <p class="text-sm mt-1" style="color: var(--text-secondary);">向可写点位下发控制命令，追踪执行状态</p>
    </div>

    <div class="grid grid-cols-1 xl:grid-cols-5 gap-5">
      <!-- Left panel: selectors + writable points -->
      <div class="xl:col-span-3 space-y-4">
        <!-- Selectors -->
        <div class="grid grid-cols-2 gap-3">
          <!-- Node select -->
          <div class="space-y-1.5">
            <label class="text-xs font-medium" style="color: var(--text-secondary);">选择节点</label>
            <div class="relative">
              <select
                v-model="selectedNodeId"
                class="w-full rounded-xl px-3 py-2.5 text-sm outline-none appearance-none cursor-pointer"
                style="background: var(--bg-secondary); border: 1px solid var(--border-color); color: var(--text-primary);"
              >
                <option value="" disabled style="background: var(--bg-secondary);">请选择节点</option>
                <option
                  v-for="node in edgeStore.nodes"
                  :key="node.node_id"
                  :value="node.node_id"
                  style="background: var(--bg-secondary);"
                >{{ node.node_name || node.node_id }}</option>
              </select>
              <ChevronDown class="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 pointer-events-none" style="color: var(--text-secondary); width:16px;height:16px;" />
            </div>
          </div>

          <!-- Device select -->
          <div class="space-y-1.5">
            <label class="text-xs font-medium" style="color: var(--text-secondary);">选择设备</label>
            <div class="relative">
              <select
                v-model="selectedDeviceId"
                :disabled="!selectedNodeId || devices.length === 0"
                class="w-full rounded-xl px-3 py-2.5 text-sm outline-none appearance-none cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
                style="background: var(--bg-secondary); border: 1px solid var(--border-color); color: var(--text-primary);"
              >
                <option value="" disabled style="background: var(--bg-secondary);">请选择设备</option>
                <option
                  v-for="device in devices"
                  :key="device.device_id"
                  :value="device.device_id"
                  style="background: var(--bg-secondary);"
                >{{ device.device_name || device.device_id }}</option>
              </select>
              <ChevronDown class="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 pointer-events-none" style="color: var(--text-secondary); width:16px;height:16px;" />
            </div>
          </div>
        </div>

        <!-- Writable points -->
        <div class="rounded-xl overflow-hidden" style="background: var(--bg-secondary); border: 1px solid var(--border-color);">
          <div class="flex items-center justify-between px-4 py-3" style="border-bottom: 1px solid var(--border-color);">
            <span class="text-sm font-semibold" style="color: var(--text-primary);">可写点位</span>
            <span class="text-xs px-2 py-0.5 rounded-full" style="background: rgba(14,165,233,0.12); color: var(--accent-primary);">{{ writablePoints.length }} 个</span>
          </div>

          <div v-if="!selectedDeviceId" class="px-5 py-8 text-sm text-center" style="color: var(--text-secondary);">
            请先选择节点和设备
          </div>
          <div v-else-if="writablePoints.length === 0" class="px-5 py-8 text-sm text-center" style="color: var(--text-secondary);">
            该设备无可写点位
          </div>
          <div v-else class="divide-y" style="divide-color: var(--border-color);">
            <div
              v-for="point in writablePoints"
              :key="point.point_id"
              class="flex items-center justify-between px-4 py-3 hover:bg-white/[0.02] transition-colors"
            >
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="font-mono text-xs" style="color: var(--accent-primary);">{{ point.point_id }}</span>
                  <span class="text-xs px-1.5 py-0.5 rounded font-mono" style="background: rgba(99,102,241,0.1); color: #818CF8;">{{ point.data_type }}</span>
                </div>
                <div class="text-xs mt-0.5" style="color: var(--text-secondary);">
                  当前值: <span class="font-mono" style="color: var(--text-primary);">{{ formatValue(point.current_value) }}</span>
                  <span v-if="point.units" class="ml-1" style="color: var(--text-secondary);">{{ point.units }}</span>
                </div>
              </div>
              <button
                @click="openWrite(point)"
                class="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-all ml-3 flex-shrink-0"
                style="background: rgba(14,165,233,0.1); color: var(--accent-primary); border: 1px solid rgba(14,165,233,0.2);"
              >
                <Send class="w-3 h-3" style="width:12px;height:12px;" />
                写入
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Right panel: command log -->
      <div class="xl:col-span-2 h-[520px]">
        <CommandLogPanel
          :commands="commands"
          @retry="handleRetry"
          @clear="handleClear"
        />
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
