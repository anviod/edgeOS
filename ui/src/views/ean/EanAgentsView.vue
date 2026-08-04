<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import {
  Server, RefreshCw, Cpu, Search,
  Eye, Layers, Clock, Radio, ArrowLeft, Trash2,
} from 'lucide-vue-next'
import { useEanStore } from '@/stores/ean'
import StatusBadge from '@/components/edge/StatusBadge.vue'
import DangerDialog from '@/components/edge/DangerDialog.vue'
import EanDisabledBanner from '@/components/ean/EanDisabledBanner.vue'
import type { EANAgentDescriptor, EANCapabilityDescriptor } from '@/types/ean'

const eanStore = useEanStore()

const searchQuery = ref('')
const statusFilter = ref<'all' | 'online' | 'offline'>('all')
const showDetailModal = ref(false)
const selectedAgent = ref<EANAgentDescriptor | null>(null)
const selectedCapabilities = ref<EANCapabilityDescriptor[]>([])
const capsLoading = ref(false)
const showDeleteDialog = ref(false)
const deleteTarget = ref<EANAgentDescriptor | null>(null)
const deleteError = ref('')

onMounted(async () => {
  await Promise.all([eanStore.fetchHealth(), eanStore.fetchAgents()])
})

const filteredAgents = computed(() => {
  let result = eanStore.agents
  if (statusFilter.value !== 'all') {
    result = result.filter(a => a.status === statusFilter.value)
  }
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase()
    result = result.filter(a =>
      a.id.toLowerCase().includes(q) ||
      a.kind.toLowerCase().includes(q) ||
      (a.metadata?.hostname || '').toLowerCase().includes(q)
    )
  }
  return result
})

async function openDetail(agent: EANAgentDescriptor) {
  selectedAgent.value = agent
  showDetailModal.value = true
  capsLoading.value = true
  try {
    selectedCapabilities.value = await eanStore.fetchAgentCapabilities(agent.id)
  } finally {
    capsLoading.value = false
  }
}

function closeDetail() {
  showDetailModal.value = false
  selectedAgent.value = null
  selectedCapabilities.value = []
}

async function deleteAgent(agent: EANAgentDescriptor) {
  deleteError.value = ''
  deleteTarget.value = agent
  showDeleteDialog.value = true
}

async function confirmDeleteAgent() {
  const agent = deleteTarget.value
  if (!agent) return
  deleteTarget.value = null
  showDeleteDialog.value = false
  try {
    const { eanApi } = await import('@/api/index')
    await eanApi.deleteAgent(agent.id)
    if (showDetailModal.value && selectedAgent.value?.id === agent.id) {
      closeDetail()
    }
    await eanStore.fetchAgents()
    await eanStore.fetchHealth()
  } catch (error) {
    deleteError.value = (error as Error).message
  }
}

const categoryColors: Record<string, string> = {
  device: '#0EA5E9',
  driver: '#0EA5E9',
  system: '#6366F1',
  ai: '#F59E0B',
  workflow: '#10B981',
}

const permissionColors: Record<string, string> = {
  read: '#10B981',
  write: '#F59E0B',
  admin: '#EF4444',
  ai: '#A855F7',
}
</script>

<template>
  <div class="space-y-5">
    <!-- 页头 / Page Header -->
    <div class="flex items-center justify-between">
      <div>
        <div class="flex items-center gap-2 mb-1">
          <router-link
            to="/ean"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm transition-colors hover:bg-white/5"
            style="color: var(--text-secondary); border: 1px solid var(--border-color);"
          >
            <ArrowLeft class="w-4 h-4" style="width:16px;height:16px;" />
            返回 EAN 协调中心
          </router-link>
        </div>
        <h1 class="text-xl font-bold" style="color: var(--text-primary);">Agent 管理</h1>
        <p class="text-sm mt-1" style="color: var(--text-secondary);">发现中心索引的 Edge Agent 注册表</p>
      </div>
      <button
        @click="eanStore.fetchAgents()"
        class="flex items-center gap-2 px-4 py-2 rounded-xl text-sm transition-colors hover:bg-white/5"
        style="color: var(--text-secondary); border: 1px solid var(--border-color);"
      >
        <RefreshCw class="w-4 h-4" :class="eanStore.loading ? 'animate-spin' : ''" style="width:16px;height:16px;" />
        刷新
      </button>
    </div>

    <EanDisabledBanner v-if="eanStore.isEanDisabled" compact />

    <div
      v-if="eanStore.lastError && !eanStore.isEanDisabled"
      class="rounded-xl border px-4 py-2.5 text-xs"
      style="background: rgba(239,68,68,0.08); border-color: rgba(239,68,68,0.2); color: #EF4444;"
    >
      {{ eanStore.lastError }}
    </div>

    <div
      v-if="deleteError"
      class="rounded-xl border px-4 py-2.5 text-xs flex items-center gap-2"
      style="background: rgba(239,68,68,0.08); border-color: rgba(239,68,68,0.2); color: #EF4444;"
    >
      <span class="flex-1">删除失败：{{ deleteError }}</span>
      <button type="button" class="hover:opacity-80 transition-opacity" @click="deleteError = ''">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
      </button>
    </div>

    <!-- 筛选栏 / Filter Bar -->
    <div class="flex items-center gap-3">
      <div class="flex-1 relative">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4" style="width:16px;height:16px;color:var(--text-muted);" />
        <input
          v-model="searchQuery"
          placeholder="搜索 Agent ID / Kind / Hostname"
          class="w-full pl-10 pr-4 py-2 rounded-xl text-sm"
          style="background: var(--bg-secondary); border: 1px solid var(--border-color); color: var(--text-primary);"
        />
      </div>
      <div class="flex items-center gap-1 p-1 rounded-xl" style="background: var(--bg-secondary); border: 1px solid var(--border-color);">
        <button
          v-for="opt in [{ v: 'all', l: '全部' }, { v: 'online', l: '在线' }, { v: 'offline', l: '离线' }]"
          :key="opt.v"
          @click="statusFilter = opt.v as 'all' | 'online' | 'offline'"
          class="px-3 py-1.5 rounded-lg text-xs font-medium transition-colors"
          :style="statusFilter === opt.v
            ? 'background: var(--accent-primary); color: white;'
            : 'color: var(--text-secondary);'"
        >
          {{ opt.l }}
        </button>
      </div>
    </div>

    <!-- Agent 表格 / Agent Table -->
    <div class="rounded-lg border overflow-hidden" style="background: var(--bg-secondary); border-color: var(--border-color);">
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr style="border-bottom: 1px solid var(--border-color);">
              <th class="text-left px-5 py-3 text-xs font-semibold uppercase tracking-wide" style="color: var(--text-muted);">Agent ID</th>
              <th class="text-left px-5 py-3 text-xs font-semibold uppercase tracking-wide" style="color: var(--text-muted);">类型</th>
              <th class="text-left px-5 py-3 text-xs font-semibold uppercase tracking-wide" style="color: var(--text-muted);">版本</th>
              <th class="text-left px-5 py-3 text-xs font-semibold uppercase tracking-wide" style="color: var(--text-muted);">传输层</th>
              <th class="text-left px-5 py-3 text-xs font-semibold uppercase tracking-wide" style="color: var(--text-muted);">心跳间隔</th>
              <th class="text-left px-5 py-3 text-xs font-semibold uppercase tracking-wide" style="color: var(--text-muted);">状态</th>
              <th class="text-right px-5 py-3 text-xs font-semibold uppercase tracking-wide" style="color: var(--text-muted);">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="filteredAgents.length === 0">
              <td colspan="7" class="px-5 py-12 text-center" style="color: var(--text-secondary);">
                <div class="flex flex-col items-center gap-2">
                  <Server class="w-8 h-8 opacity-30" style="width:32px;height:32px;" />
                  <span>暂无 Agent 数据</span>
                </div>
              </td>
            </tr>
            <tr
              v-for="agent in filteredAgents"
              :key="agent.id"
              class="transition-colors hover:bg-white/[0.02]"
              style="border-bottom: 1px solid var(--border-color);"
            >
              <td class="px-5 py-3">
                <div class="flex items-center gap-3">
                  <div class="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0" style="background: rgba(14,165,233,0.08);">
                    <Cpu class="w-4 h-4" style="width:16px;height:16px;color:var(--accent-primary);" />
                  </div>
                  <div>
                    <p class="text-sm font-medium font-mono" style="color: var(--text-primary);">{{ agent.id }}</p>
                    <p v-if="agent.metadata?.hostname" class="text-xs" style="color: var(--text-muted);">{{ agent.metadata.hostname }}</p>
                  </div>
                </div>
              </td>
              <td class="px-5 py-3">
                <span class="text-xs font-mono px-2 py-1 rounded-lg" style="background: rgba(99,102,241,0.1); color: #a5b4fc;">{{ agent.kind }}</span>
              </td>
              <td class="px-5 py-3 text-xs font-mono" style="color: var(--text-secondary);">v{{ agent.version }}</td>
              <td class="px-5 py-3">
                <div class="flex items-center gap-1">
                  <Radio class="w-3.5 h-3.5" style="width:14px;height:14px;color:var(--text-muted);" />
                  <span class="text-xs font-mono" style="color: var(--text-secondary);">{{ (agent.transport || []).join(' / ') || '—' }}</span>
                </div>
              </td>
              <td class="px-5 py-3 text-xs font-mono" style="color: var(--text-secondary);">{{ agent.heartbeat_interval_sec }}s</td>
              <td class="px-5 py-3"><StatusBadge :status="agent.status" size="sm" /></td>
              <td class="px-5 py-3 text-right">
                <div class="flex items-center justify-end gap-2">
                  <button
                    @click="openDetail(agent)"
                    class="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs transition-colors hover:bg-sky-500/10"
                    style="color: var(--accent-primary); background: rgba(14,165,233,0.08); border: 1px solid rgba(14,165,233,0.2);"
                  >
                    <Eye class="w-3.5 h-3.5" style="width:14px;height:14px;" />
                    详情
                  </button>
                  <button
                    @click="deleteAgent(agent)"
                    class="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs transition-colors hover:bg-red-500/10"
                    style="color: #EF4444; background: rgba(239,68,68,0.06); border: 1px solid rgba(239,68,68,0.2);"
                  >
                    <Trash2 class="w-3.5 h-3.5" style="width:14px;height:14px;" />
                    删除
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Delete confirmation modal -->
    <DangerDialog
      v-model:open="showDeleteDialog"
      title="删除 Agent"
      :description="`确认删除 Agent ${deleteTarget?.id}？将同时移除该 Agent 的能力索引，若对应节点存在也会一并删除。`"
      actionName="删除"
      variant="danger"
      @confirm="confirmDeleteAgent"
    />

    <!-- Agent 详情弹窗 / Agent Detail Modal -->
    <Transition enter-active-class="transition-all duration-300 ease-out" enter-from-class="opacity-0 scale-95 translate-y-4" leave-active-class="transition-all duration-200 ease-in" leave-to-class="opacity-0 scale-95 translate-y-4">
      <div v-if="showDetailModal" class="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div class="absolute inset-0 bg-black/50 backdrop-blur-sm transition-opacity duration-300" @click="closeDetail"></div>
        <div class="relative w-full max-w-4xl rounded-xl overflow-hidden shadow-2xl transform transition-all duration-300" style="background: var(--bg-secondary); border: 1px solid var(--border-color); box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);">
          <!-- Modal Header -->
          <div class="px-6 py-4 border-b flex items-center justify-between" style="border-color: var(--border-color); background: rgba(255,255,255,0.02);">
            <h3 class="text-lg font-semibold flex items-center gap-2" style="color: var(--text-primary);">
              <Cpu class="w-5 h-5" style="color: var(--accent-primary); width:20px;height:20px;" />
              Agent 详情 - {{ selectedAgent?.id }}
            </h3>
            <button @click="closeDetail" class="p-2 rounded-lg transition-colors hover:bg-white/5" style="color: var(--text-secondary);">
              <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="h-5 w-5"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
            </button>
          </div>

          <div class="p-6 space-y-6 max-h-[70vh] overflow-y-auto">
            <!-- 基本信息 / Basic Info -->
            <div class="space-y-3">
              <h4 class="text-sm font-medium flex items-center gap-2" style="color: var(--text-secondary);">
                <span class="w-1.5 h-1.5 rounded-full" style="background: var(--accent-primary);"></span>
                基本信息
              </h4>
              <div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
                <div class="p-3 rounded-lg" style="background: rgba(255,255,255,0.02); border: 1px solid var(--border-color);">
                  <div class="text-xs font-medium" style="color: var(--text-muted);">Agent ID</div>
                  <div class="text-sm font-mono mt-1 truncate" style="color: var(--accent-primary);">{{ selectedAgent?.id }}</div>
                </div>
                <div class="p-3 rounded-lg" style="background: rgba(255,255,255,0.02); border: 1px solid var(--border-color);">
                  <div class="text-xs font-medium" style="color: var(--text-muted);">类型</div>
                  <div class="text-sm font-mono mt-1" style="color: var(--text-primary);">{{ selectedAgent?.kind }}</div>
                </div>
                <div class="p-3 rounded-lg" style="background: rgba(255,255,255,0.02); border: 1px solid var(--border-color);">
                  <div class="text-xs font-medium" style="color: var(--text-muted);">版本</div>
                  <div class="text-sm font-mono mt-1" style="color: var(--text-primary);">v{{ selectedAgent?.version }}</div>
                </div>
                <div class="p-3 rounded-lg" style="background: rgba(255,255,255,0.02); border: 1px solid var(--border-color);">
                  <div class="text-xs font-medium" style="color: var(--text-muted);">传输层</div>
                  <div class="text-sm font-mono mt-1" style="color: var(--text-primary);">{{ (selectedAgent?.transport || []).join(', ') || '—' }}</div>
                </div>
                <div class="p-3 rounded-lg" style="background: rgba(255,255,255,0.02); border: 1px solid var(--border-color);">
                  <div class="text-xs font-medium" style="color: var(--text-muted);">心跳间隔</div>
                  <div class="text-sm font-mono mt-1" style="color: var(--text-primary);">{{ selectedAgent?.heartbeat_interval_sec }}s</div>
                </div>
                <div class="p-3 rounded-lg" style="background: rgba(255,255,255,0.02); border: 1px solid var(--border-color);">
                  <div class="text-xs font-medium" style="color: var(--text-muted);">状态</div>
                  <div class="mt-1"><StatusBadge v-if="selectedAgent" :status="selectedAgent.status" size="sm" /></div>
                </div>
              </div>
            </div>

            <!-- Metadata -->
            <div v-if="selectedAgent?.metadata && Object.keys(selectedAgent.metadata).length > 0" class="space-y-3">
              <h4 class="text-sm font-medium flex items-center gap-2" style="color: var(--text-secondary);">
                <span class="w-1.5 h-1.5 rounded-full" style="background: var(--accent-primary);"></span>
                元数据
              </h4>
              <div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
                <div
                  v-for="(val, key) in selectedAgent.metadata"
                  :key="key"
                  class="p-3 rounded-lg"
                  style="background: rgba(255,255,255,0.02); border: 1px solid var(--border-color);"
                >
                  <div class="text-xs font-medium" style="color: var(--text-muted);">{{ key }}</div>
                  <div class="text-sm font-mono mt-1 break-all" style="color: var(--text-secondary);">{{ val }}</div>
                </div>
              </div>
            </div>

            <!-- Capabilities -->
            <div class="space-y-3">
              <h4 class="text-sm font-medium flex items-center gap-2" style="color: var(--text-secondary);">
                <span class="w-1.5 h-1.5 rounded-full" style="background: var(--accent-primary);"></span>
                能力列表 ({{ selectedCapabilities.length }})
              </h4>
              <div v-if="capsLoading" class="flex items-center justify-center py-8">
                <RefreshCw class="w-5 h-5 animate-spin" style="width:20px;height:20px;color:var(--text-muted);" />
              </div>
              <div v-else-if="selectedCapabilities.length === 0" class="py-6 text-sm text-center" style="color: var(--text-secondary);">
                该 Agent 暂无注册能力
              </div>
              <div v-else class="space-y-2">
                <div
                  v-for="cap in selectedCapabilities"
                  :key="cap.id"
                  class="flex items-center justify-between p-3 rounded-lg transition-colors hover:bg-white/[0.02]"
                  style="background: rgba(255,255,255,0.02); border: 1px solid var(--border-color);"
                >
                  <div class="flex items-center gap-3 min-w-0">
                    <div class="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0" :style="{ background: `rgba(${categoryColors[cap.category] ? '14,165,233' : '99,102,241'},0.08)` }">
                      <Layers class="w-4 h-4" style="width:16px;height:16px;" :style="{ color: categoryColors[cap.category] || '#6366F1' }" />
                    </div>
                    <div class="min-w-0">
                      <p class="text-sm font-medium font-mono truncate" style="color: var(--text-primary);">{{ cap.id }}</p>
                      <p class="text-xs truncate" style="color: var(--text-muted);">{{ cap.description || '无描述' }}</p>
                    </div>
                  </div>
                  <div class="flex items-center gap-2 flex-shrink-0">
                    <span
                      class="text-xs px-2 py-1 rounded-lg font-mono"
                      :style="{ background: `${categoryColors[cap.category] || '#6366F1'}15`, color: categoryColors[cap.category] || '#6366F1' }"
                    >
                      {{ cap.category }}
                    </span>
                    <span
                      class="text-xs px-2 py-1 rounded-lg font-mono"
                      :style="{ background: `${permissionColors[cap.permission] || '#6B7280'}15`, color: permissionColors[cap.permission] || '#6B7280' }"
                    >
                      {{ cap.permission }}
                    </span>
                    <span
                      class="text-xs px-2 py-1 rounded-lg font-mono"
                      :style="cap.source === 'native-ean'
                        ? { background: 'rgba(16,185,129,0.12)', color: '#10B981' }
                        : { background: 'rgba(245,158,11,0.12)', color: '#F59E0B' }"
                    >
                      {{ cap.source || 'unknown' }}
                    </span>
                    <span class="text-xs font-mono" style="color: var(--text-muted);">
                      <Clock class="w-3 h-3 inline" style="width:12px;height:12px;" /> {{ cap.timeout_sec }}s
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>
