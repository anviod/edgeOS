<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ChevronDown, Send, Copy, Check } from 'lucide-vue-next'
import { useEdgeStore } from '@/stores/edge'
import { useRealtimeStore } from '@/stores/realtime'
import { useEanStore } from '@/stores/ean'
import { controlApi } from '@/api/index'
import CommandLogPanel from '@/components/edge/CommandLogPanel.vue'
import WritePointModal from '@/components/edge/WritePointModal.vue'
import type { EdgeXPointInfo, CommandRecord, WritePointRequest } from '@/types/edgex'

const edgeStore = useEdgeStore()
const rtStore = useRealtimeStore()
const eanStore = useEanStore()

const selectedNodeId = ref('')
const selectedDeviceId = ref('')

// Load nodes on mount
onMounted(async () => {
  // 刷新 EAN 健康状态（设备控制页依赖 isEanEnabled 判定 EAN 是否启用）
  // | Refresh EAN health so the device-control page reflects the real EAN state.
  await eanStore.fetchHealth()
  await edgeStore.fetchNodes()
  if (edgeStore.nodes.length > 0) {
    selectedNodeId.value = edgeStore.nodes[0].node_id
  }
  // 加载命令记录
  await loadCommands()
})

// 加载命令记录（兼容历史 V1 命令）
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

// When node selected, load devices + EAN capabilities
watch(selectedNodeId, async (nid) => {
  selectedDeviceId.value = ''
  if (nid) {
    await edgeStore.fetchDevices(nid)
    // 预加载该 Agent 的 EAN Capability（用于查找写操作能力）
    if (eanStore.isEanEnabled) {
      await eanStore.fetchAgentCapabilities(nid)
    }
  }
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

// EAN 写能力查找：在 Agent 的 Capability 列表中找到与**当前设备协议匹配**的写操作能力
// 修复：避免能力与协议不匹配（如 BACnet 设备选到 modbus.write_point 导致调用超时）
// 优先级：① 协议匹配（metadata.protocol == 设备 device_profile）且 permission=write；
//         ② 任意 permission=write 兜底。
const selectedDevice = computed(() =>
  devices.value.find(d => d.device_id === selectedDeviceId.value)
)
const writeCapability = computed(() => {
  const caps = eanStore.capabilitiesByAgent[selectedNodeId.value]
  if (!caps || caps.length === 0) return null
  const devProto = selectedDevice.value?.device_profile
  if (devProto) {
    // 优先选与设备协议一致的 write 能力（如 bacnet-ip → bacnet_ip.write_point）
    const matched = caps.find(c =>
      c.permission === 'write' &&
      c.metadata?.protocol === devProto &&
      c.id.includes('write')
    )
    if (matched) return matched
  }
  // 兜底：任意 write 能力
  return caps.find(c => c.permission === 'write' && c.id.includes('write')) ||
         caps.find(c => c.permission === 'write') ||
         null
})

// EAN 是否可用
const eanAvailable = computed(() => eanStore.isEanEnabled && writeCapability.value !== null)

const copiedCapability = ref(false)

async function copyCapabilityId() {
  if (!writeCapability.value) return
  try {
    await navigator.clipboard.writeText(writeCapability.value.id)
    copiedCapability.value = true
    setTimeout(() => { copiedCapability.value = false }, 1600)
  } catch {
    copiedCapability.value = false
  }
}

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
const writeError = ref('')

function openWrite(point: EdgeXPointInfo) {
  writingPoint.value = point
  writeError.value = ''
  writeModalVisible.value = true
}

// EAN Invoke 写操作（OS-P3-02: 替代 V1 命令 API）
async function handleWrite(pointId: string, value: unknown) {
  const nodeId = selectedNodeId.value
  const deviceId = selectedDeviceId.value
  writeError.value = ''

  // 检查 EAN 是否可用
  if (!eanStore.isEanEnabled) {
    writeError.value = 'EAN 未启用，无法执行写操作'
    return
  }

  // 确保已加载 Capability
  if (!eanStore.capabilitiesByAgent[nodeId]) {
    await eanStore.fetchAgentCapabilities(nodeId)
  }

  if (!writeCapability.value) {
    writeError.value = `节点 ${nodeId} 无可用的写能力（write Capability）`
    return
  }

  try {
    const result = await eanStore.invokeCapability({
      target: nodeId,
      capability: writeCapability.value.id,
      arguments: {
        device_id: deviceId,
        // write_point 能力 schema 用 address（支持 point_id 自动解析），避免参数名不匹配
        address: pointId,
        value,
      },
      timeout_sec: 30,
    })

    // 追踪到命令日志
    const status = result?.response?.status || 'completed'
    const errorMsg = result?.response?.result?.error || ''
    rtStore.addCmdTrack({
      request_id: result?.response?.invoke_id || `ean-${Date.now()}`,
      node_id: nodeId,
      device_id: deviceId,
      point_id: pointId,
      value,
      status: status as any,
      error: errorMsg,
      ts: Date.now(),
    })

    if (status !== 'completed' && status !== 'success') {
      writeError.value = `写入失败: ${errorMsg || status}`
    } else {
      writeModalVisible.value = false
    }
  } catch (error) {
    writeError.value = `EAN Invoke 失败: ${(error as Error).message}`
  }
}

async function handleRetry(cmd: CommandRecord) {
  const nodeId = cmd.node_id
  const deviceId = cmd.device_id
  const pointId = cmd.point_id
  const value = cmd.value

  // 检查 EAN 是否可用
  if (!eanStore.isEanEnabled || !writeCapability.value) {
    // EAN 不可用时降级到 V1 命令（仅用于重试历史命令）
    try {
      const req: WritePointRequest = { node_id: nodeId, device_id: deviceId, point_id: pointId, value }
      const newCmd = await controlApi.writePoint(req)
      rtStore.addCmdTrack({
        request_id: newCmd.id,
        node_id: nodeId,
        device_id: deviceId,
        point_id: pointId,
        value,
        status: 'pending',
        ts: Date.now(),
      })
    } catch (error) {
      console.error('V1 retry failed:', error)
    }
    return
  }

  // EAN 重试
  try {
    const result = await eanStore.invokeCapability({
      target: nodeId,
      capability: writeCapability.value.id,
      arguments: {
        device_id: deviceId,
        // write_point 能力 schema 用 address（支持 point_id 自动解析），避免参数名不匹配
        address: pointId,
        value,
      },
      timeout_sec: 30,
    })

    const status = result?.response?.status || 'completed'
    const errorMsg = result?.response?.result?.error || ''
    rtStore.addCmdTrack({
      request_id: result?.response?.invoke_id || `ean-retry-${Date.now()}`,
      node_id: nodeId,
      device_id: deviceId,
      point_id: pointId,
      value,
      status: status as any,
      error: errorMsg,
      ts: Date.now(),
    })
  } catch (error) {
    console.error('EAN retry failed:', error)
  }
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
    <div class="flex items-start justify-between gap-4">
      <div>
        <h1 class="text-xl font-bold" style="color: var(--text-primary);">设备控制</h1>
        <p class="text-sm mt-1" style="color: var(--text-secondary);">
          通过 EAN Invoke 向可写点位下发控制命令，追踪执行状态
          <span v-if="eanAvailable" class="ml-1 px-1.5 py-0.5 rounded text-xs" style="background: rgba(16,185,129,0.12); color: #10B981;">EAN 就绪</span>
          <span v-else-if="eanStore.isEanEnabled" class="ml-1 px-1.5 py-0.5 rounded text-xs" style="background: rgba(245,158,11,0.12); color: #F59E0B;">无写能力</span>
          <span v-else class="ml-1 px-1.5 py-0.5 rounded text-xs" style="background: rgba(239,68,68,0.12); color: #EF4444;">EAN 未启用</span>
        </p>
      </div>
      <router-link
        to="/ean/invoke"
        class="text-xs px-3 py-1.5 rounded-lg shrink-0 transition-colors hover:bg-white/5"
        style="color: var(--accent-primary); border: 1px solid var(--border-color);"
      >
        EAN 能力控制台 →
      </router-link>
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

        <!-- Write capability info -->
        <div v-if="selectedNodeId && eanStore.isEanEnabled && writeCapability" class="flex items-center gap-2 rounded-lg px-3 py-2" style="background: rgba(16,185,129,0.06); border: 1px solid rgba(16,185,129,0.15);">
          <span class="text-xs" style="color: #10B981;">写能力:</span>
          <code class="text-xs font-mono" style="color: var(--accent-primary);">{{ writeCapability.id }}</code>
          <button
            type="button"
            @click="copyCapabilityId"
            class="inline-flex items-center gap-1 text-xs px-1.5 py-0.5 rounded transition-colors hover:bg-white/5 ml-auto"
            style="color: var(--text-secondary);"
            :title="copiedCapability ? '已复制' : '复制写能力 ID'"
          >
            <component :is="copiedCapability ? Check : Copy" class="w-3 h-3" style="width:12px;height:12px;" />
          </button>
        </div>

        <!-- EAN not enabled warning -->
        <div v-if="selectedNodeId && !eanStore.isEanEnabled" class="flex items-start gap-2 rounded-lg p-3" style="background: rgba(239,68,68,0.08); border: 1px solid rgba(239,68,68,0.2);">
          <span class="text-xs leading-relaxed" style="color: #EF4444;">
            <template v-if="eanStore.healthLoading">正在检查 EAN 状态…</template>
            <template v-else-if="eanStore.health === null">无法获取 EAN 健康状态（后端未响应）。</template>
            <template v-else>EAN Bus 未启用，写操作不可用。请在配置中启用 EAN 功能后重试。</template>
            <button class="ml-2 px-2 py-0.5 rounded text-xs border" style="border-color: rgba(239,68,68,0.3);" @click="eanStore.fetchHealth()">刷新</button>
          </span>
        </div>

        <!-- Write error -->
        <div v-if="writeError" class="flex items-start gap-2 rounded-lg p-3" style="background: rgba(239,68,68,0.08); border: 1px solid rgba(239,68,68,0.2);">
          <span class="text-xs leading-relaxed" style="color: #EF4444;">{{ writeError }}</span>
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
                :disabled="!eanAvailable"
                class="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-all ml-3 flex-shrink-0 disabled:opacity-40 disabled:cursor-not-allowed"
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
