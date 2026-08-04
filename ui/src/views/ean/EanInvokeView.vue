<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import {
  Play, RefreshCw, Shield, Terminal, CheckCircle, AlertCircle,
  Loader2, ChevronDown, Clock, ArrowLeft, Copy, Check,
} from 'lucide-vue-next'
import { useEanStore } from '@/stores/ean'
import type { EANInvokeCallResult, EANCapabilityDescriptor } from '@/types/ean'
import EanDisabledBanner from '@/components/ean/EanDisabledBanner.vue'

const eanStore = useEanStore()
const route = useRoute()

const selectedAgentId = ref('')
const selectedCapabilityId = ref('')
const argumentsText = ref('{}')
const timeoutSec = ref(30)
const tenantId = ref('default')
const fillNotice = ref('')

const invoking = ref(false)
const invokeResult = ref<EANInvokeCallResult | null>(null)
const invokeError = ref('')
const capabilities = ref<EANCapabilityDescriptor[]>([])
const capsLoading = ref(false)
const skipNextAgentWatch = ref(false)
const copiedResult = ref(false)

/** 从联合调试页 query 填充示例 */
async function applyFillFromQuery() {
  if (route.query.fill !== '1' && !route.query.target) return
  const target = String(route.query.target || '')
  const capability = String(route.query.capability || '')
  const args = String(route.query.arguments || '{}')
  const timeout = Number(route.query.timeout_sec)
  const tenant = String(route.query.tenant_id || '')

  if (args) {
    try {
      JSON.parse(args)
      argumentsText.value = args
    } catch {
      argumentsText.value = '{}'
    }
  }
  if (Number.isFinite(timeout) && timeout > 0) timeoutSec.value = timeout
  if (tenant) tenantId.value = tenant

  if (target) {
    skipNextAgentWatch.value = true
    selectedAgentId.value = target
    capsLoading.value = true
    try {
      capabilities.value = await eanStore.fetchAgentCapabilities(target)
      if (capability) {
        selectedCapabilityId.value = capability
        if (!capabilities.value.find(c => c.id === capability)) {
          fillNotice.value = `已填充示例：${target} / ${capability}（Capability 尚未发现时可先发 Discovery）`
        } else {
          fillNotice.value = `已从联合调试帮助填充：${target} → ${capability}`
        }
      }
    } finally {
      capsLoading.value = false
    }
  }
}

onMounted(async () => {
  await Promise.all([
    eanStore.fetchHealth(),
    eanStore.fetchAgents(),
    eanStore.fetchAuditRecords(100),
  ])
  await applyFillFromQuery()
})

watch(
  () => route.fullPath,
  () => { applyFillFromQuery() },
)

watch(selectedAgentId, async (newId) => {
  if (skipNextAgentWatch.value) {
    skipNextAgentWatch.value = false
    return
  }
  selectedCapabilityId.value = ''
  capabilities.value = []
  invokeResult.value = null
  invokeError.value = ''
  if (!newId) return
  capsLoading.value = true
  try {
    capabilities.value = await eanStore.fetchAgentCapabilities(newId)
  } finally {
    capsLoading.value = false
  }
})

watch(selectedCapabilityId, (newCapId) => {
  const cap = capabilities.value.find(c => c.id === newCapId)
  if (cap && cap.timeout_sec > 0) {
    timeoutSec.value = cap.timeout_sec
  }
  if (cap?.input_schema && route.query.fill !== '1') {
    argumentsText.value = JSON.stringify(cap.input_schema, null, 2)
  }
})

const selectedCapability = computed(() =>
  capabilities.value.find(c => c.id === selectedCapabilityId.value)
)

const canInvoke = computed(() =>
  Boolean(selectedAgentId.value && selectedCapabilityId.value && !invoking.value)
)

async function handleInvoke() {
  if (!canInvoke.value) return
  invoking.value = true
  invokeResult.value = null
  invokeError.value = ''
  try {
    let args: Record<string, unknown>
    try {
      args = JSON.parse(argumentsText.value || '{}')
    } catch {
      throw new Error('参数 JSON 格式无效')
    }
    const result = await eanStore.invokeCapability({
      target: selectedAgentId.value,
      capability: selectedCapabilityId.value,
      arguments: args,
      timeout_sec: timeoutSec.value,
      tenant_id: tenantId.value,
    })
    invokeResult.value = result
  } catch (error) {
    invokeError.value = (error as Error).message
  } finally {
    invoking.value = false
  }
}

const invokeResultText = computed(() => {
  if (!invokeResult.value) return ''
  if (invokeResult.value.response?.result?.values) {
    return JSON.stringify(invokeResult.value.response.result.values, null, 2)
  }
  return JSON.stringify(invokeResult.value, null, 2)
})

async function copyInvokeResult() {
  if (!invokeResult.value) return
  try {
    await navigator.clipboard.writeText(invokeResultText.value)
    copiedResult.value = true
    setTimeout(() => { copiedResult.value = false }, 1600)
  } catch {
    copiedResult.value = false
  }
}

function formatTime(ts: number) {
  if (!ts) return '—'
  return new Date(ts < 1e12 ? ts * 1000 : ts).toLocaleString('zh-CN', {
    month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit',
  })
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
        <h1 class="text-xl font-bold" style="color: var(--text-primary);">能力调用</h1>
        <p class="text-sm mt-1" style="color: var(--text-secondary);">跨节点 Invoke 编排 + Reply 关联 + 审计追踪</p>
      </div>
    </div>

    <EanDisabledBanner v-if="eanStore.isEanDisabled" compact />

    <div
      v-if="fillNotice"
      class="rounded-xl border px-4 py-2.5 text-xs"
      style="background: rgba(14,165,233,0.08); border-color: rgba(14,165,233,0.2); color: var(--accent-primary);"
    >
      {{ fillNotice }}
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-5">
      <div class="space-y-4">
        <div class="rounded-lg border overflow-hidden" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <div class="px-5 py-3.5 border-b flex items-center gap-2" style="border-color: var(--border-color);">
            <Terminal class="w-4 h-4" style="color: var(--accent-primary); width:16px;height:16px;" />
            <span class="text-sm font-semibold" style="color: var(--text-primary);">Invoke 编排</span>
          </div>
          <div class="p-5 space-y-4">
            <div>
              <label class="text-xs font-medium mb-1.5 block" style="color: var(--text-muted);">目标 Agent</label>
              <div class="relative">
                <select
                  v-model="selectedAgentId"
                  class="w-full px-3 py-2 pr-9 rounded-lg text-sm appearance-none cursor-pointer"
                  style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary);"
                >
                  <option value="">选择 Agent...</option>
                  <option v-for="agent in eanStore.agents" :key="agent.id" :value="agent.id">
                    {{ agent.id }} ({{ agent.status }})
                  </option>
                  <option
                    v-if="selectedAgentId && !eanStore.agents.find(a => a.id === selectedAgentId)"
                    :value="selectedAgentId"
                  >
                    {{ selectedAgentId }} (示例/未发现)
                  </option>
                </select>
                <ChevronDown class="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 pointer-events-none" style="width:16px;height:16px;color:var(--text-muted);" />
              </div>
            </div>

            <div>
              <label class="text-xs font-medium mb-1.5 block" style="color: var(--text-muted);">Capability</label>
              <div v-if="capsLoading" class="flex items-center gap-2 py-2">
                <Loader2 class="w-4 h-4 animate-spin" style="width:16px;height:16px;color:var(--text-muted);" />
                <span class="text-xs" style="color: var(--text-muted);">加载能力列表...</span>
              </div>
              <div v-else-if="!selectedAgentId" class="py-2 text-xs" style="color: var(--text-muted);">请先选择 Agent</div>
              <div v-else class="relative">
                <select
                  v-model="selectedCapabilityId"
                  class="w-full px-3 py-2 pr-9 rounded-lg text-sm appearance-none cursor-pointer"
                  style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary);"
                >
                  <option value="">选择 Capability...</option>
                  <option v-for="cap in capabilities" :key="cap.id" :value="cap.id">
                    {{ cap.id }} [{{ cap.category }}]
                  </option>
                  <option
                    v-if="selectedCapabilityId && !capabilities.find(c => c.id === selectedCapabilityId)"
                    :value="selectedCapabilityId"
                  >
                    {{ selectedCapabilityId }} (示例)
                  </option>
                </select>
                <ChevronDown class="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 pointer-events-none" style="width:16px;height:16px;color:var(--text-muted);" />
              </div>
            </div>

            <div v-if="selectedCapability" class="p-3 rounded-lg space-y-2" style="background: rgba(14,165,233,0.04); border: 1px solid rgba(14,165,233,0.15);">
              <div class="flex items-center gap-2 flex-wrap">
                <span class="text-xs font-mono px-2 py-0.5 rounded-lg" :style="{ background: `${permissionColors[selectedCapability.permission] || '#6B7280'}15`, color: permissionColors[selectedCapability.permission] || '#6B7280' }">
                  {{ selectedCapability.permission }}
                </span>
                <span class="text-xs font-mono flex items-center gap-1" style="color: var(--text-muted);">
                  <Clock class="w-3 h-3" style="width:12px;height:12px;" /> {{ selectedCapability.timeout_sec }}s
                </span>
              </div>
              <p v-if="selectedCapability.description" class="text-xs" style="color: var(--text-secondary);">{{ selectedCapability.description }}</p>
            </div>

            <div>
              <label class="text-xs font-medium mb-1.5 block" style="color: var(--text-muted);">调用参数 (JSON)</label>
              <textarea
                v-model="argumentsText"
                rows="6"
                class="w-full px-3 py-2 rounded-lg text-xs font-mono resize-y"
                style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary);"
                placeholder='{"key": "value"}'
              ></textarea>
            </div>

            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="text-xs font-medium mb-1.5 block" style="color: var(--text-muted);">超时 (秒)</label>
                <input
                  v-model.number="timeoutSec"
                  type="number"
                  min="1"
                  max="300"
                  class="w-full px-3 py-2 rounded-lg text-sm font-mono"
                  style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary);"
                />
              </div>
              <div>
                <label class="text-xs font-medium mb-1.5 block" style="color: var(--text-muted);">租户 ID</label>
                <input
                  v-model="tenantId"
                  type="text"
                  class="w-full px-3 py-2 rounded-lg text-sm font-mono"
                  style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary);"
                />
              </div>
            </div>

            <button
              @click="handleInvoke"
              :disabled="!canInvoke"
              class="w-full flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl text-sm font-medium transition-all"
              :style="canInvoke
                ? 'background: var(--accent-primary); color: white;'
                : 'background: var(--bg-tertiary); color: var(--text-muted); cursor: not-allowed;'"
            >
              <Play v-if="!invoking" class="w-4 h-4" style="width:16px;height:16px;" />
              <Loader2 v-else class="w-4 h-4 animate-spin" style="width:16px;height:16px;" />
              {{ invoking ? '调用中...' : '发起 Invoke' }}
            </button>
          </div>
        </div>

        <div v-if="invokeResult || invokeError" class="rounded-lg border overflow-hidden" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <div class="px-5 py-3.5 border-b flex items-center gap-2" style="border-color: var(--border-color);">
            <component :is="invokeError ? AlertCircle : CheckCircle" class="w-4 h-4" :style="{ width:'16px', height:'16px', color: invokeError ? '#EF4444' : '#10B981' }" />
            <span class="text-sm font-semibold" style="color: var(--text-primary);">调用结果</span>
            <button
              v-if="invokeResult && !invokeError"
              type="button"
              @click="copyInvokeResult"
              class="ml-auto inline-flex items-center gap-1 text-xs px-2 py-1 rounded-lg transition-colors hover:bg-white/5"
              style="color: var(--text-secondary); border: 1px solid var(--border-color);"
            >
              <component :is="copiedResult ? Check : Copy" class="w-3 h-3" style="width:12px;height:12px;" />
              {{ copiedResult ? '已复制' : '复制结果' }}
            </button>
          </div>
          <div class="p-5">
            <div v-if="invokeError" class="text-sm" style="color: #EF4444;">
              {{ invokeError }}
            </div>
            <div v-else-if="invokeResult?.response" class="space-y-3">
              <div class="flex items-center gap-2">
                <span class="text-xs font-medium" style="color: var(--text-muted);">Invoke ID:</span>
                <span class="text-xs font-mono" style="color: var(--accent-primary);">{{ invokeResult.response.invoke_id }}</span>
              </div>
              <div class="flex items-center gap-2">
                <span class="text-xs font-medium" style="color: var(--text-muted);">状态:</span>
                <span class="text-xs font-mono px-2 py-0.5 rounded-lg" :style="{ background: invokeResult.response.result?.success ? 'rgba(16,185,129,0.12)' : 'rgba(239,68,68,0.12)', color: invokeResult.response.result?.success ? '#10B981' : '#EF4444' }">
                  {{ invokeResult.response.status }}
                </span>
              </div>
              <div v-if="invokeResult.response.result?.error" class="p-3 rounded-lg text-xs" style="background: rgba(239,68,68,0.08); color: #EF4444;">
                {{ invokeResult.response.result.error }}
              </div>
              <div v-if="invokeResult.response.result?.values" class="space-y-1">
                <span class="text-xs font-medium" style="color: var(--text-muted);">返回值:</span>
                <pre class="p-3 rounded-lg text-xs font-mono overflow-x-auto" style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-secondary);">{{ JSON.stringify(invokeResult.response.result.values, null, 2) }}</pre>
              </div>
            </div>
            <div v-else class="text-xs font-mono overflow-x-auto" style="color: var(--text-secondary);">
              <pre>{{ JSON.stringify(invokeResult, null, 2) }}</pre>
            </div>
          </div>
        </div>
      </div>

      <div class="rounded-lg border overflow-hidden" style="background: var(--bg-secondary); border-color: var(--border-color);">
        <div class="px-5 py-3.5 border-b flex items-center justify-between" style="border-color: var(--border-color);">
          <div class="flex items-center gap-2">
            <Shield class="w-4 h-4" style="color: var(--accent-primary); width:16px;height:16px;" />
            <span class="text-sm font-semibold" style="color: var(--text-primary);">审计追踪</span>
            <span class="text-xs px-1.5 py-0.5 rounded-lg font-mono" style="background: rgba(245,158,11,0.1); color: #F59E0B;">{{ eanStore.auditRecords.length }}</span>
          </div>
          <button type="button" class="p-1.5 rounded-lg transition-colors hover:bg-white/5" @click="eanStore.fetchAuditRecords(100)">
            <RefreshCw class="w-3.5 h-3.5" style="width:14px;height:14px;color:var(--text-secondary);" />
          </button>
        </div>
        <div class="max-h-[600px] overflow-y-auto">
          <div v-if="eanStore.auditRecords.length === 0" class="px-5 py-12 text-sm text-center" style="color: var(--text-secondary);">
            暂无审计记录
          </div>
          <div
            v-for="record in eanStore.auditRecords"
            :key="record.id"
            class="px-5 py-3 transition-colors hover:bg-white/[0.02]"
            style="border-bottom: 1px solid var(--border-color);"
          >
            <div class="flex items-center justify-between gap-2 mb-1">
              <div class="flex items-center gap-2 min-w-0">
                <span
                  class="w-1.5 h-1.5 rounded-full flex-shrink-0"
                  :style="{ background: record.status === 'success' ? '#10B981' : record.status === 'error' ? '#EF4444' : '#F59E0B' }"
                />
                <span class="text-xs font-mono truncate" style="color: var(--text-primary);">{{ record.capability || 'heartbeat-timeout' }}</span>
              </div>
              <span class="text-xs flex-shrink-0" style="color: var(--text-muted);">{{ formatTime(record.timestamp) }}</span>
            </div>
            <div class="flex items-center gap-3 pl-3.5 text-xs" style="color: var(--text-muted);">
              <span>{{ record.initiator }}</span>
              <span>→</span>
              <span class="font-mono">{{ record.target }}</span>
              <span v-if="record.invoke_id" class="font-mono truncate max-w-[100px]" style="color: var(--accent-primary);" :title="record.invoke_id">{{ record.invoke_id }}</span>
              <span
                class="ml-auto px-1.5 py-0.5 rounded font-mono"
                :style="{ background: record.status === 'success' ? 'rgba(16,185,129,0.1)' : record.status === 'error' ? 'rgba(239,68,68,0.1)' : 'rgba(245,158,11,0.1)', color: record.status === 'success' ? '#10B981' : record.status === 'error' ? '#EF4444' : '#F59E0B' }"
              >
                {{ record.status }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
