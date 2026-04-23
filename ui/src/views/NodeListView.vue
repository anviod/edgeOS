<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { Server, RefreshCw, Trash2, ChevronRight, Cpu, CheckCircle, Eye } from 'lucide-vue-next'
import { useEdgeStore } from '@/stores/edge'
import { useMiddlewareStore } from '@/stores/middleware'
import StatusBadge from '@/components/edge/StatusBadge.vue'

const router = useRouter()
const edgeStore = useEdgeStore()
const middlewareStore = useMiddlewareStore()

// Toast for new node registration
const toastMsg = ref('')
const toastTimer = ref<ReturnType<typeof setTimeout> | null>(null)

// Node detail modal
const showDetailModal = ref(false)
const selectedNode = ref<any>(null)

// Tooltip states
const hoveredButton = ref<string | null>(null)

function showToast(msg: string) {
  toastMsg.value = msg
  if (toastTimer.value) clearTimeout(toastTimer.value)
  toastTimer.value = setTimeout(() => { toastMsg.value = '' }, 4000)
}

// Middleware selection dropdown state
const showMiddlewareDropdown = ref(false)
const dropdownPosition = ref({ top: 0, left: 0 })
const selectedNodeForDiscovery = ref<string | null>(null)

// Connected middlewares only
const connectedMiddlewares = computed(() => {
  return middlewareStore.list.filter(m => m.status === 'connected')
})

function openMiddlewareDropdown(nodeId: string, event: MouseEvent) {
  selectedNodeForDiscovery.value = nodeId
  const rect = (event.target as HTMLElement).getBoundingClientRect()
  const dropdownWidth = 288 // w-64 = 16rem = 256px, add padding
  const dropdownHeight = Math.min(connectedMiddlewares.value.length * 48 + 48, 320) // max 320px

  let top = rect.bottom + 8
  let left = rect.left

  // Boundary detection
  const viewportWidth = window.innerWidth
  const viewportHeight = window.innerHeight

  // Flip to left if too close to right edge
  if (left + dropdownWidth > viewportWidth - 16) {
    left = viewportWidth - dropdownWidth - 16
  }

  // Open upward if not enough space below
  if (top + dropdownHeight > viewportHeight - 16) {
    top = rect.top - dropdownHeight - 8
  }

  // Ensure minimum left
  if (left < 16) left = 16

  dropdownPosition.value = { top, left }
  showMiddlewareDropdown.value = true
}

function closeMiddlewareDropdown() {
  showMiddlewareDropdown.value = false
  selectedNodeForDiscovery.value = null
}

async function triggerDiscoverVia(middlewareId: string, middlewareName: string) {
  closeMiddlewareDropdown()
  const nodeId = selectedNodeForDiscovery.value
  try {
    const { nodeApi } = await import('@/api/index')
    await nodeApi.triggerDiscoverVia(middlewareId)
    showToast(`已通过 ${middlewareName} 触发节点 ${nodeId} 设备发现`)
  } catch (error) {
    showToast(`触发失败: ${(error as Error).message}`)
  }
}

// Intercept applyNodeStatus to show toast for new nodes
const prevCount = ref(0)
onMounted(async () => {
  prevCount.value = edgeStore.nodes.length
  await Promise.all([
    edgeStore.fetchNodes(),
    middlewareStore.fetchList()
  ])
  // Add click outside listener
  document.addEventListener('click', handleClickOutside)
})

// Close dropdown when clicking outside
function handleClickOutside(event: MouseEvent) {
  const target = event.target as HTMLElement
  if (!target.closest('.middleware-dropdown')) {
    closeMiddlewareDropdown()
  }
}

onUnmounted(() => {
  if (toastTimer.value) clearTimeout(toastTimer.value)
  document.removeEventListener('click', handleClickOutside)
})

function formatTime(ts: number) {
  if (!ts) return '—'
  return new Date(ts < 1e12 ? ts * 1000 : ts).toLocaleString('zh-CN', {
    month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit',
  })
}

async function handleDelete(nodeId: string) {
  if (confirm(`确认删除节点 ${nodeId}？`)) {
    await edgeStore.removeNode(nodeId)
  }
}

function openDetailModal(node: any) {
  selectedNode.value = node
  showDetailModal.value = true
}

function closeDetailModal() {
  showDetailModal.value = false
  selectedNode.value = null
}

function handleButtonHover(buttonId: string) {
  hoveredButton.value = buttonId
}

function handleButtonLeave() {
  hoveredButton.value = null
}

async function triggerDiscoveryAll() {
  if (connectedMiddlewares.value.length === 0) {
    showToast('暂无可用的中间件通道')
    return
  }
  try {
    const { nodeApi } = await import('@/api/index')
    await nodeApi.discoverAll()
    showToast(`已向所有中间件通道发送发现请求`)
  } catch (error) {
    showToast(`触发失败: ${(error as Error).message}`)
  }
}
</script>

<template>
  <div class="space-y-5">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-bold" style="color: var(--text-primary);">节点管理</h1>
        <p class="text-sm mt-1" style="color: var(--text-secondary);">监控 EdgeX 边缘网关节点注册与心跳状态</p>
      </div>
      <div class="flex items-center gap-2">
        <button
          v-if="connectedMiddlewares.length > 0"
          @click="triggerDiscoveryAll"
          class="flex items-center gap-2 px-4 py-2 rounded-xl text-sm transition-colors hover:bg-sky-500/10"
          style="color: var(--accent-primary); background: rgba(14,165,233,0.08); border: 1px solid rgba(14,165,233,0.2);"
        >
          <Cpu class="w-4 h-4" style="width:16px;height:16px;" />
          发现所有节点
        </button>
        <button
          @click="edgeStore.fetchNodes()"
          class="flex items-center gap-2 px-4 py-2 rounded-xl text-sm transition-colors hover:bg-white/5"
          style="color: var(--text-secondary); border: 1px solid var(--border-color);"
        >
          <RefreshCw class="w-4 h-4" :class="edgeStore.nodesLoading ? 'animate-spin' : ''" style="width:16px;height:16px;" />
          刷新
        </button>
      </div>
    </div>

    <!-- Toast -->
    <Transition enter-active-class="transition-all duration-300" enter-from-class="opacity-0 translate-y-2" leave-active-class="transition-all duration-200" leave-to-class="opacity-0">
      <div
        v-if="toastMsg"
        class="fixed top-16 right-6 z-50 flex items-center gap-2 rounded-xl px-4 py-3 text-sm font-medium shadow-xl"
        style="background: rgba(16,185,129,0.15); border: 1px solid rgba(16,185,129,0.3); color: #10B981; backdrop-filter: blur(8px);"
      >
        <CheckCircle class="w-4 h-4" style="width:16px;height:16px;" />
        {{ toastMsg }}
      </div>
    </Transition>

    <!-- Middleware Selection Dropdown -->
    <Transition enter-active-class="transition-all duration-200 ease-out" enter-from-class="opacity-0 scale-95 translate-y-1" leave-active-class="transition-all duration-150 ease-in" leave-to-class="opacity-0 scale-95 translate-y-1">
      <div
        v-if="showMiddlewareDropdown"
        class="middleware-dropdown fixed z-50 w-72 rounded-xl shadow-2xl overflow-hidden"
        :style="{ top: dropdownPosition.top + 'px', left: dropdownPosition.left + 'px' }"
        style="background: var(--bg-secondary); border: 1px solid var(--border-color); backdrop-filter: blur(12px);"
      >
        <div class="px-4 py-3 text-xs font-semibold flex items-center gap-2" style="color: var(--text-secondary); border-bottom: 1px solid var(--border-color); background: rgba(255,255,255,0.02);">
          <span class="w-1.5 h-1.5 rounded-full" style="background: var(--accent-primary);"></span>
          选择中间件通道
        </div>
        <div v-if="connectedMiddlewares.length === 0" class="px-4 py-8 text-sm text-center" style="color: var(--text-muted);">
          <div class="flex flex-col items-center gap-2">
            <span class="text-2xl opacity-50">📡</span>
            <span>暂无可用的中间件</span>
          </div>
        </div>
        <div v-else class="py-2 max-h-64 overflow-y-auto custom-scrollbar">
          <div
            v-for="(mw, index) in connectedMiddlewares"
            :key="mw.id"
            class="mx-2"
          >
            <button
              @click="triggerDiscoverVia(mw.id, mw.name)"
              class="w-full flex items-center gap-3 px-3 py-2.5 text-sm text-left transition-all duration-150 hover:bg-white/[0.06] active:bg-white/[0.04] rounded-lg"
              style="color: var(--text-primary);"
            >
              <span
                class="w-2 h-2 rounded-full flex-shrink-0 shadow-sm mw-status-dot"
                :class="mw.status === 'connected' ? 'connected' : 'disconnected'"
              ></span>
              <span class="flex-1 truncate font-medium">{{ mw.name }}</span>
              <span class="text-xs px-2 py-0.5 rounded-md flex-shrink-0" style="background: rgba(99,102,241,0.12); color: #a5b4fc; font-weight: 500;">
                {{ mw.type.toUpperCase() }}
              </span>
            </button>
            <div
              v-if="index < connectedMiddlewares.length - 1"
              class="mx-3 my-0.5"
              style="height: 1px; background: var(--border-color); opacity: 0.3;"
            ></div>
          </div>
        </div>
      </div>
    </Transition>

    <!-- Node Detail Modal -->
    <Transition enter-active-class="transition-all duration-300 ease-out" enter-from-class="opacity-0 scale-95 translate-y-4" leave-active-class="transition-all duration-200 ease-in" leave-to-class="opacity-0 scale-95 translate-y-4">
      <div v-if="showDetailModal" class="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div class="absolute inset-0 bg-black/50 backdrop-blur-sm transition-opacity duration-300" @click="closeDetailModal"></div>
        <div class="relative w-full max-w-4xl rounded-xl overflow-hidden shadow-2xl transform transition-all duration-300" style="background: var(--bg-secondary); border: 1px solid var(--border-color); box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);">
          <div class="px-6 py-4 border-b flex items-center justify-between" style="border-color: var(--border-color); background: rgba(255,255,255,0.02);">
            <h3 class="text-lg font-semibold flex items-center gap-2" style="color: var(--text-primary);">
              <Server class="w-5 h-5" style="color: var(--accent-primary);" />
              节点详情 - {{ selectedNode?.node_name || selectedNode?.node_id }}
            </h3>
            <button 
              @click="closeDetailModal" 
              class="p-2 rounded-lg transition-colors hover:bg-white/5" 
              style="color: var(--text-secondary);"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="h-5 w-5"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
            </button>
          </div>
          <div class="p-6">
            <div v-if="selectedNode" class="space-y-6">
              <!-- Basic Info -->
              <div class="space-y-3">
                <h4 class="text-sm font-medium flex items-center gap-2" style="color: var(--text-secondary);">
                  <span class="w-1.5 h-1.5 rounded-full" style="background: var(--accent-primary);"></span>
                  基本信息
                </h4>
                <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                  <div class="space-y-2 p-3 rounded-lg" style="background: rgba(255,255,255,0.02); border: 1px solid var(--border-color);">
                    <div class="text-xs font-medium" style="color: var(--text-muted);">节点 ID</div>
                    <div class="text-sm font-mono truncate" style="color: var(--accent-primary);">{{ selectedNode.node_id }}</div>
                  </div>
                  <div class="space-y-2 p-3 rounded-lg" style="background: rgba(255,255,255,0.02); border: 1px solid var(--border-color);">
                    <div class="text-xs font-medium" style="color: var(--text-muted);">节点名称</div>
                    <div class="text-sm font-medium" style="color: var(--text-primary);">{{ selectedNode.node_name }}</div>
                  </div>
                  <div class="space-y-2 p-3 rounded-lg" style="background: rgba(255,255,255,0.02); border: 1px solid var(--border-color);">
                    <div class="text-xs font-medium" style="color: var(--text-muted);">版本</div>
                    <div class="text-sm font-mono" style="color: var(--text-secondary);">{{ selectedNode.version || '—' }}</div>
                  </div>
                  <div class="space-y-2 p-3 rounded-lg" style="background: rgba(255,255,255,0.02); border: 1px solid var(--border-color);">
                    <div class="text-xs font-medium" style="color: var(--text-muted);">协议</div>
                    <div class="text-sm font-mono" style="color: var(--text-secondary);">{{ selectedNode.protocol }}</div>
                  </div>
                  <div class="space-y-2 p-3 rounded-lg" style="background: rgba(255,255,255,0.02); border: 1px solid var(--border-color);">
                    <div class="text-xs font-medium" style="color: var(--text-muted);">状态</div>
                    <div class="text-sm"><StatusBadge :status="selectedNode.status" size="sm" /></div>
                  </div>
                  <div class="space-y-2 p-3 rounded-lg" style="background: rgba(255,255,255,0.02); border: 1px solid var(--border-color);">
                    <div class="text-xs font-medium" style="color: var(--text-muted);">最后心跳</div>
                    <div class="text-sm" style="color: var(--text-secondary);">{{ formatTime(selectedNode.last_seen) }}</div>
                  </div>
                </div>
              </div>
              
              <!-- Heartbeat Data -->
              <div class="space-y-3">
                <h4 class="text-sm font-medium flex items-center gap-2" style="color: var(--text-secondary);">
                  <span class="w-1.5 h-1.5 rounded-full" style="background: var(--accent-primary);"></span>
                  心跳数据
                </h4>
                <div class="space-y-4">
                  <div class="p-4 rounded-lg" style="background: rgba(255,255,255,0.02); border: 1px solid var(--border-color);">
                    <div class="text-xs font-medium mb-3" style="color: var(--text-secondary);">系统指标</div>
                    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
                      <div class="space-y-1">
                        <div class="text-xs" style="color: var(--text-muted);">CPU 使用率</div>
                        <div class="text-sm font-medium" style="color: var(--text-primary);">{{ selectedNode.system_metrics?.cpu_usage || 0 }}%</div>
                      </div>
                      <div class="space-y-1">
                        <div class="text-xs" style="color: var(--text-muted);">内存使用率</div>
                        <div class="text-sm font-medium" style="color: var(--text-primary);">{{ selectedNode.system_metrics?.memory_usage || 0 }}%</div>
                      </div>
                      <div class="space-y-1">
                        <div class="text-xs" style="color: var(--text-muted);">磁盘使用率</div>
                        <div class="text-sm font-medium" style="color: var(--text-primary);">{{ selectedNode.system_metrics?.disk_usage || 0 }}%</div>
                      </div>
                      <div class="space-y-1">
                        <div class="text-xs" style="color: var(--text-muted);">负载均值</div>
                        <div class="text-sm font-medium" style="color: var(--text-primary);">{{ selectedNode.system_metrics?.load_average || 0 }}</div>
                      </div>
                    </div>
                  </div>
                  
                  <div class="p-4 rounded-lg" style="background: rgba(255,255,255,0.02); border: 1px solid var(--border-color);">
                    <div class="text-xs font-medium mb-3" style="color: var(--text-secondary);">设备统计</div>
                    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
                      <div class="space-y-1">
                        <div class="text-xs" style="color: var(--text-muted);">总设备数</div>
                        <div class="text-sm font-medium" style="color: var(--text-primary);">{{ selectedNode.device_summary?.total_count || 0 }}</div>
                      </div>
                      <div class="space-y-1">
                        <div class="text-xs" style="color: var(--text-muted);">在线设备</div>
                        <div class="text-sm font-medium" style="color: #10B981;">{{ selectedNode.device_summary?.online_count || 0 }}</div>
                      </div>
                      <div class="space-y-1">
                        <div class="text-xs" style="color: var(--text-muted);">离线设备</div>
                        <div class="text-sm font-medium" style="color: var(--text-secondary);">{{ selectedNode.device_summary?.offline_count || 0 }}</div>
                      </div>
                      <div class="space-y-1">
                        <div class="text-xs" style="color: var(--text-muted);">错误设备</div>
                        <div class="text-sm font-medium" style="color: var(--destructive);">{{ selectedNode.device_summary?.error_count || 0 }}</div>
                      </div>
                    </div>
                  </div>
                  
                  <div class="p-4 rounded-lg" style="background: rgba(255,255,255,0.02); border: 1px solid var(--border-color);">
                    <div class="text-xs font-medium mb-3" style="color: var(--text-secondary);">通道统计</div>
                    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
                      <div class="space-y-1">
                        <div class="text-xs" style="color: var(--text-muted);">总通道数</div>
                        <div class="text-sm font-medium" style="color: var(--text-primary);">{{ selectedNode.channel_summary?.total_count || 0 }}</div>
                      </div>
                      <div class="space-y-1">
                        <div class="text-xs" style="color: var(--text-muted);">连接通道</div>
                        <div class="text-sm font-medium" style="color: #10B981;">{{ selectedNode.channel_summary?.connected_count || 0 }}</div>
                      </div>
                      <div class="space-y-1">
                        <div class="text-xs" style="color: var(--text-muted);">错误通道</div>
                        <div class="text-sm font-medium" style="color: var(--destructive);">{{ selectedNode.channel_summary?.error_count || 0 }}</div>
                      </div>
                      <div class="space-y-1">
                        <div class="text-xs" style="color: var(--text-muted);">成功率</div>
                        <div class="text-sm font-medium" style="color: var(--text-primary);">{{ Math.round((selectedNode.channel_summary?.avg_success_rate || 0) * 100) }}%</div>
                      </div>
                    </div>
                  </div>
                  
                  <div class="p-4 rounded-lg" style="background: rgba(255,255,255,0.02); border: 1px solid var(--border-color);">
                    <div class="text-xs font-medium mb-3" style="color: var(--text-secondary);">任务统计</div>
                    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
                      <div class="space-y-1">
                        <div class="text-xs" style="color: var(--text-muted);">总任务数</div>
                        <div class="text-sm font-medium" style="color: var(--text-primary);">{{ selectedNode.task_summary?.total_count || 0 }}</div>
                      </div>
                      <div class="space-y-1">
                        <div class="text-xs" style="color: var(--text-muted);">运行中</div>
                        <div class="text-sm font-medium" style="color: #10B981;">{{ selectedNode.task_summary?.running_count || 0 }}</div>
                      </div>
                      <div class="space-y-1">
                        <div class="text-xs" style="color: var(--text-muted);">已暂停</div>
                        <div class="text-sm font-medium" style="color: var(--text-secondary);">{{ selectedNode.task_summary?.paused_count || 0 }}</div>
                      </div>
                      <div class="space-y-1">
                        <div class="text-xs" style="color: var(--text-muted);">错误任务</div>
                        <div class="text-sm font-medium" style="color: var(--destructive);">{{ selectedNode.task_summary?.error_count || 0 }}</div>
                      </div>
                    </div>
                  </div>
                  
                  <div class="p-4 rounded-lg" style="background: rgba(255,255,255,0.02); border: 1px solid var(--border-color);">
                    <div class="text-xs font-medium mb-3" style="color: var(--text-secondary);">连接统计</div>
                    <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
                      <div class="space-y-1">
                        <div class="text-xs" style="color: var(--text-muted);">重连次数</div>
                        <div class="text-sm font-medium" style="color: var(--text-primary);">{{ selectedNode.connection_stats?.reconnect_count || 0 }}</div>
                      </div>
                      <div class="space-y-1">
                        <div class="text-xs" style="color: var(--text-muted);">发布计数</div>
                        <div class="text-sm font-medium" style="color: var(--text-primary);">{{ selectedNode.connection_stats?.publish_count || 0 }}</div>
                      </div>
                      <div class="space-y-1">
                        <div class="text-xs" style="color: var(--text-muted);">协议版本</div>
                        <div class="text-sm font-medium" style="color: var(--text-primary);">{{ selectedNode.connection_stats?.protocol_version || '—' }}</div>
                      </div>
                      <div class="space-y-1">
                        <div class="text-xs" style="color: var(--text-muted);">连接时间</div>
                        <div class="text-sm font-medium" style="color: var(--text-secondary);">{{ selectedNode.connection_stats?.connected_since ? formatTime(selectedNode.connection_stats.connected_since) : '—' }}</div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div class="px-6 py-4 border-t flex justify-end" style="border-color: var(--border-color); background: rgba(255,255,255,0.02);">
            <button
              @click="closeDetailModal"
              class="px-6 py-2.5 rounded-lg text-sm font-medium transition-all duration-200 hover:bg-white/5 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-bg-secondary focus:ring-accent-primary"
              style="color: var(--text-secondary); border: 1px solid var(--border-color);"
            >
              关闭
            </button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- Loading -->
    <div v-if="edgeStore.nodesLoading && edgeStore.nodes.length === 0" class="flex items-center justify-center py-16">
      <RefreshCw class="w-6 h-6 animate-spin loading-spinner" />
    </div>

    <!-- Empty -->
    <div v-else-if="edgeStore.nodes.length === 0"
      class="flex flex-col items-center justify-center py-20 rounded-xl"
      style="background: var(--bg-secondary); border: 1px dashed var(--border-color);"
    >
      <Server class="w-10 h-10 mb-3" style="color: var(--text-muted);" />
      <p class="text-base font-medium mb-1" style="color: var(--text-primary);">尚无注册节点</p>
      <p class="text-sm" style="color: var(--text-secondary);">等待 EdgeX 网关通过 MQTT 发起注册</p>
    </div>

    <!-- Table -->
    <div v-else class="rounded-xl overflow-hidden" style="background: var(--bg-secondary); border: 1px solid var(--border-color);">
      <table class="w-full text-sm">
        <thead>
          <tr style="border-bottom: 1px solid var(--border-color);">
            <th class="text-left px-5 py-3 font-medium text-xs" style="color: var(--text-secondary);">节点 ID</th>
            <th class="text-left px-4 py-3 font-medium text-xs" style="color: var(--text-secondary);">名称</th>
            <th class="text-left px-4 py-3 font-medium text-xs hidden md:table-cell" style="color: var(--text-secondary);">版本</th>
            <th class="text-left px-4 py-3 font-medium text-xs hidden lg:table-cell" style="color: var(--text-secondary);">协议</th>
            <th class="text-left px-4 py-3 font-medium text-xs" style="color: var(--text-secondary);">状态</th>
            <th class="text-left px-4 py-3 font-medium text-xs hidden xl:table-cell" style="color: var(--text-secondary);">最后心跳</th>
            <th class="text-right px-5 py-3 font-medium text-xs" style="color: var(--text-secondary);">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y" style="divide-color: var(--border-color);">
          <tr
            v-for="node in edgeStore.nodes"
            :key="node.node_id"
            class="group transition-colors hover:bg-white/[0.02]"
          >
            <td class="px-5 py-3.5">
              <span class="font-mono text-xs" style="color: var(--accent-primary);">{{ node.node_id }}</span>
            </td>
            <td class="px-4 py-3.5 font-medium" style="color: var(--text-primary);">{{ node.node_name }}</td>
            <td class="px-4 py-3.5 font-mono text-xs hidden md:table-cell" style="color: var(--text-secondary);">{{ node.version || '—' }}</td>
            <td class="px-4 py-3.5 hidden lg:table-cell">
              <span class="text-xs px-2 py-0.5 rounded font-mono" style="background: rgba(99,102,241,0.1); color: #818CF8;">{{ node.protocol }}</span>
            </td>
            <td class="px-4 py-3.5"><StatusBadge :status="node.status" size="sm" /></td>
            <td class="px-4 py-3.5 text-xs tabular-nums hidden xl:table-cell" style="color: var(--text-secondary);">{{ formatTime(node.last_seen) }}</td>
            <td class="px-5 py-3.5">
              <div class="flex items-center justify-end gap-2">
                <div class="relative">
                  <button
                    @click.stop="openMiddlewareDropdown(node.node_id, $event)"
                    @mouseenter="handleButtonHover('discover-'+node.node_id)"
                    @mouseleave="handleButtonLeave"
                    class="flex items-center justify-center w-8 h-8 rounded-lg transition-colors hover:bg-sky-500/10 border border-rgba(14,165,233,0.2) hover:border-sky-500"
                    style="color: var(--accent-primary);"
                  >
                    <Cpu class="w-4 h-4" style="width:16px;height:16px;" />
                  </button>
                  <div 
                    v-show="hoveredButton === 'discover-'+node.node_id"
                    class="absolute bottom-full left-1/2 -translate-x-1/2 mb-1 px-2 py-1 text-xs rounded whitespace-nowrap transition-opacity z-10"
                    style="background: rgba(0,0,0,0.8); color: white; backdrop-filter: blur(4px);"
                  >
                    发现设备
                  </div>
                </div>
                <div class="relative">
                  <button
                    @click="router.push(`/nodes/${node.node_id}/devices`)"
                    @mouseenter="handleButtonHover('devices-'+node.node_id)"
                    @mouseleave="handleButtonLeave"
                    class="flex items-center justify-center w-8 h-8 rounded-lg transition-colors hover:bg-sky-500/10 border border-rgba(14,165,233,0.2) hover:border-sky-500"
                    style="color: var(--accent-primary);"
                  >
                    <Server class="w-4 h-4" style="width:16px;height:16px;" />
                  </button>
                  <div 
                    v-show="hoveredButton === 'devices-'+node.node_id"
                    class="absolute bottom-full left-1/2 -translate-x-1/2 mb-1 px-2 py-1 text-xs rounded whitespace-nowrap transition-opacity z-10"
                    style="background: rgba(0,0,0,0.8); color: white; backdrop-filter: blur(4px);"
                  >
                    查看设备
                  </div>
                </div>
                <div class="relative">
                  <button
                    @click="openDetailModal(node)"
                    @mouseenter="handleButtonHover('detail-'+node.node_id)"
                    @mouseleave="handleButtonLeave"
                    class="flex items-center justify-center w-8 h-8 rounded-lg transition-colors hover:bg-sky-500/10 border border-rgba(14,165,233,0.2) hover:border-sky-500"
                    style="color: var(--accent-primary);"
                  >
                    <Eye class="w-4 h-4" style="width:16px;height:16px;" />
                  </button>
                  <div 
                    v-show="hoveredButton === 'detail-'+node.node_id"
                    class="absolute bottom-full left-1/2 -translate-x-1/2 mb-1 px-2 py-1 text-xs rounded whitespace-nowrap transition-opacity z-10"
                    style="background: rgba(0,0,0,0.8); color: white; backdrop-filter: blur(4px);"
                  >
                    查看详情
                  </div>
                </div>
                <div class="relative">
                  <button
                    @click="handleDelete(node.node_id)"
                    @mouseenter="handleButtonHover('delete-'+node.node_id)"
                    @mouseleave="handleButtonLeave"
                    class="flex items-center justify-center w-8 h-8 rounded-lg transition-colors hover:bg-sky-500/10 border border-rgba(14,165,233,0.2) hover:border-sky-500"
                    style="color: var(--accent-primary);"
                  >
                    <Trash2 class="w-4 h-4" style="width:16px;height:16px;" />
                  </button>
                  <div 
                    v-show="hoveredButton === 'delete-'+node.node_id"
                    class="absolute bottom-full left-1/2 -translate-x-1/2 mb-1 px-2 py-1 text-xs rounded whitespace-nowrap transition-opacity z-10"
                    style="background: rgba(0,0,0,0.8); color: white; backdrop-filter: blur(4px);"
                  >
                    删除节点
                  </div>
                </div>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.loading-spinner { color: var(--accent-primary); }
.mw-status-dot.connected { background: #10B981; box-shadow: 0 0 6px rgba(16,185,129,0.5); }
.mw-status-dot.disconnected { background: var(--text-muted); }

/* Custom scrollbar for dropdown */
.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(255,255,255,0.1);
  border-radius: 3px;
}
.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: rgba(255,255,255,0.2);
}
</style>
