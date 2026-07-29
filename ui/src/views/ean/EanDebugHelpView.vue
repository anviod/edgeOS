<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import {
  BookOpen, ChevronRight, Copy, Check, Play, RefreshCw,
  ListOrdered, Bug, Terminal, Radio, ExternalLink,
} from 'lucide-vue-next'
import { useEanStore } from '@/stores/ean'
import EanDisabledBanner from '@/components/ean/EanDisabledBanner.vue'
import StatusBadge from '@/components/edge/StatusBadge.vue'
import {
  eanFlowSteps,
  eanGuideExamples,
  eanTroubleshoot,
  eanHttpApiQuickRef,
  buildInvokeFillPath,
  type EanGuideExample,
} from '@/lib/ean-debug-guides'

const router = useRouter()
const eanStore = useEanStore()

const activeTab = ref<'flow' | 'guides' | 'troubleshoot' | 'api'>('flow')
const copiedKey = ref('')
const expandedGuide = ref<string | null>(eanGuideExamples[0]?.id ?? null)

onMounted(() => {
  eanStore.fetchHealth()
})

const showDisabled = computed(
  () => eanStore.health != null && !eanStore.isEanEnabled,
)

async function copyText(key: string, text: string) {
  try {
    await navigator.clipboard.writeText(text)
    copiedKey.value = key
    setTimeout(() => {
      if (copiedKey.value === key) copiedKey.value = ''
    }, 1600)
  } catch {
    copiedKey.value = ''
  }
}

function toggleGuide(id: string) {
  expandedGuide.value = expandedGuide.value === id ? null : id
}

function goFillInvoke(ex: EanGuideExample) {
  if (!ex.fillInvoke) return
  router.push(buildInvokeFillPath(ex.fillInvoke))
}

function goStep(path: string) {
  router.push(path)
}
</script>

<template>
  <div class="space-y-5">
    <!-- 页头 -->
    <div class="flex items-center justify-between gap-3 flex-wrap">
      <div>
        <h1 class="text-xl font-bold" style="color: var(--text-primary);">联合调试帮助</h1>
        <p class="text-sm mt-1" style="color: var(--text-secondary);">
          EAN 端到端流程 · 可复制 Topic/JSON · 常见失败排查 · 深链到各页面
        </p>
      </div>
      <div class="flex items-center gap-2">
        <StatusBadge
          v-if="eanStore.health"
          :status="eanStore.isEanEnabled ? 'online' : 'offline'"
          size="md"
        />
        <button
          type="button"
          class="flex items-center gap-2 px-4 py-2 rounded-xl text-sm transition-colors hover:bg-white/5"
          style="color: var(--text-secondary); border: 1px solid var(--border-color);"
          @click="eanStore.fetchHealth()"
        >
          <RefreshCw class="w-4 h-4" :class="eanStore.healthLoading ? 'animate-spin' : ''" style="width:16px;height:16px;" />
          刷新健康
        </button>
      </div>
    </div>

    <EanDisabledBanner v-if="showDisabled" />

    <!-- 当前传输健康快照 -->
    <div
      v-if="eanStore.health?.transport_details?.length"
      class="rounded-lg border px-5 py-3.5 grid grid-cols-1 sm:grid-cols-2 gap-3"
      style="background: var(--bg-secondary); border-color: var(--border-color);"
    >
      <div
        v-for="td in eanStore.health!.transport_details"
        :key="td.name"
        class="flex items-center justify-between gap-2 min-w-0"
      >
        <div class="min-w-0">
          <div class="flex items-center gap-2">
            <span class="text-xs font-semibold uppercase font-mono" style="color: var(--text-primary);">{{ td.name }}</span>
            <StatusBadge :status="td.connected ? 'online' : 'offline'" size="sm" />
          </div>
          <p class="text-xs font-mono mt-0.5 truncate" style="color: var(--text-muted);">{{ td.endpoint }}</p>
        </div>
      </div>
    </div>

    <!-- 快捷入口 -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
      <button
        v-for="item in [
          { label: '协调中心', path: '/ean', icon: Radio },
          { label: 'Agent', path: '/ean/agents', icon: ListOrdered },
          { label: 'Invoke', path: '/ean/invoke', icon: Play },
          { label: '事件流', path: '/ean/events', icon: Terminal },
        ]"
        :key="item.path"
        type="button"
        class="flex items-center gap-2 px-3 py-2.5 rounded-lg text-sm transition-colors hover:bg-white/5"
        style="background: var(--bg-secondary); border: 1px solid var(--border-color); color: var(--text-primary);"
        @click="goStep(item.path)"
      >
        <component :is="item.icon" class="w-4 h-4" style="width:16px;height:16px;color:var(--accent-primary);" />
        {{ item.label }}
        <ChevronRight class="w-3.5 h-3.5 ml-auto" style="width:14px;height:14px;color:var(--text-muted);" />
      </button>
    </div>

    <!-- Tab -->
    <div class="flex items-center gap-1 p-1 rounded-xl w-fit flex-wrap" style="background: var(--bg-secondary); border: 1px solid var(--border-color);">
      <button
        v-for="tab in [
          { v: 'flow', l: '联调流程', icon: ListOrdered },
          { v: 'guides', l: '指导用例', icon: BookOpen },
          { v: 'troubleshoot', l: '失败排查', icon: Bug },
          { v: 'api', l: 'HTTP 速查', icon: Terminal },
        ]"
        :key="tab.v"
        type="button"
        class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors"
        :style="activeTab === tab.v
          ? 'background: var(--accent-primary); color: white;'
          : 'color: var(--text-secondary);'"
        @click="activeTab = tab.v as typeof activeTab"
      >
        <component :is="tab.icon" class="w-3.5 h-3.5" style="width:14px;height:14px;" />
        {{ tab.l }}
      </button>
    </div>

    <!-- 流程 -->
    <div v-if="activeTab === 'flow'" class="rounded-lg border overflow-hidden" style="background: var(--bg-secondary); border-color: var(--border-color);">
      <div class="px-5 py-3.5 border-b" style="border-color: var(--border-color);">
        <span class="text-sm font-semibold" style="color: var(--text-primary);">端到端步骤</span>
        <p class="text-xs mt-0.5" style="color: var(--text-muted);">启用 EAN → 双传输 → Discovery → Invoke → Reply → Event → 心跳 offline</p>
      </div>
      <ol class="divide-y" style="divide-color: var(--border-color);">
        <li
          v-for="(step, idx) in eanFlowSteps"
          :key="step.id"
          class="px-5 py-4 flex gap-4"
        >
          <div
            class="w-7 h-7 rounded-lg flex items-center justify-center flex-shrink-0 text-xs font-bold font-mono"
            style="background: rgba(14,165,233,0.12); color: var(--accent-primary);"
          >
            {{ idx + 1 }}
          </div>
          <div class="flex-1 min-w-0 space-y-1.5">
            <div class="flex items-center justify-between gap-2 flex-wrap">
              <p class="text-sm font-medium" style="color: var(--text-primary);">{{ step.title }}</p>
              <button
                v-if="step.link"
                type="button"
                class="inline-flex items-center gap-1 text-xs transition-colors hover:opacity-80"
                style="color: var(--accent-primary);"
                @click="goStep(step.link!.path)"
              >
                {{ step.link.label }}
                <ExternalLink class="w-3 h-3" style="width:12px;height:12px;" />
              </button>
            </div>
            <p class="text-xs leading-relaxed" style="color: var(--text-secondary);">{{ step.detail }}</p>
            <p v-if="step.check" class="text-xs font-mono px-2 py-1 rounded-md inline-block" style="background: var(--bg-primary); color: var(--text-muted);">
              期望：{{ step.check }}
            </p>
          </div>
        </li>
      </ol>
    </div>

    <!-- 指导用例 -->
    <div v-else-if="activeTab === 'guides'" class="space-y-3">
      <div
        v-for="ex in eanGuideExamples"
        :key="ex.id"
        class="rounded-lg border overflow-hidden"
        style="background: var(--bg-secondary); border-color: var(--border-color);"
      >
        <button
          type="button"
          class="w-full px-5 py-3.5 flex items-center justify-between gap-3 text-left transition-colors hover:bg-white/[0.02]"
          @click="toggleGuide(ex.id)"
        >
          <div class="min-w-0">
            <p class="text-sm font-medium" style="color: var(--text-primary);">{{ ex.title }}</p>
            <p class="text-xs mt-0.5 truncate" style="color: var(--text-muted);">{{ ex.description }}</p>
          </div>
          <ChevronRight
            class="w-4 h-4 flex-shrink-0 transition-transform"
            :class="expandedGuide === ex.id ? 'rotate-90' : ''"
            style="width:16px;height:16px;color:var(--text-muted);"
          />
        </button>
        <div v-if="expandedGuide === ex.id" class="px-5 pb-5 space-y-3 border-t" style="border-color: var(--border-color);">
          <div class="pt-3 flex items-center gap-2 flex-wrap">
            <span class="text-xs" style="color: var(--text-muted);">Topic / Subject</span>
            <code class="text-xs font-mono px-2 py-1 rounded-md" style="background: var(--bg-primary); color: var(--accent-primary);">{{ ex.topic }}</code>
            <button
              type="button"
              class="inline-flex items-center gap-1 text-xs px-2 py-1 rounded-lg transition-colors hover:bg-white/5"
              style="color: var(--text-secondary); border: 1px solid var(--border-color);"
              @click="copyText(`${ex.id}-topic`, ex.topic)"
            >
              <component :is="copiedKey === `${ex.id}-topic` ? Check : Copy" class="w-3 h-3" style="width:12px;height:12px;" />
              {{ copiedKey === `${ex.id}-topic` ? '已复制' : '复制 Topic' }}
            </button>
            <button
              v-if="ex.fillInvoke"
              type="button"
              class="inline-flex items-center gap-1 text-xs px-2.5 py-1 rounded-lg ml-auto"
              style="background: var(--accent-primary); color: white;"
              @click="goFillInvoke(ex)"
            >
              <Play class="w-3 h-3" style="width:12px;height:12px;" />
              示例填充到 Invoke
            </button>
          </div>
          <div>
            <div class="flex items-center justify-between mb-1.5">
              <span class="text-xs" style="color: var(--text-muted);">示例 JSON</span>
              <button
                type="button"
                class="inline-flex items-center gap-1 text-xs px-2 py-1 rounded-lg transition-colors hover:bg-white/5"
                style="color: var(--text-secondary); border: 1px solid var(--border-color);"
                @click="copyText(`${ex.id}-json`, ex.payload)"
              >
                <component :is="copiedKey === `${ex.id}-json` ? Check : Copy" class="w-3 h-3" style="width:12px;height:12px;" />
                {{ copiedKey === `${ex.id}-json` ? '已复制' : '复制 JSON' }}
              </button>
            </div>
            <pre class="p-3 rounded-lg text-xs font-mono overflow-x-auto max-h-64" style="background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-secondary);">{{ ex.payload }}</pre>
          </div>
          <p class="text-xs" style="color: var(--text-secondary);">
            <span class="font-medium" style="color: var(--text-primary);">期望结果：</span>{{ ex.expect }}
          </p>
        </div>
      </div>
    </div>

    <!-- 排查 -->
    <div v-else-if="activeTab === 'troubleshoot'" class="rounded-lg border overflow-hidden" style="background: var(--bg-secondary); border-color: var(--border-color);">
      <table class="w-full text-sm">
        <thead>
          <tr style="border-bottom: 1px solid var(--border-color);">
            <th class="text-left px-5 py-3 text-xs font-semibold uppercase tracking-wide" style="color: var(--text-muted);">现象</th>
            <th class="text-left px-5 py-3 text-xs font-semibold uppercase tracking-wide" style="color: var(--text-muted);">原因</th>
            <th class="text-left px-5 py-3 text-xs font-semibold uppercase tracking-wide" style="color: var(--text-muted);">处理</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(row, i) in eanTroubleshoot"
            :key="i"
            style="border-bottom: 1px solid var(--border-color);"
          >
            <td class="px-5 py-3 text-xs align-top" style="color: var(--text-primary);">{{ row.symptom }}</td>
            <td class="px-5 py-3 text-xs align-top" style="color: var(--text-secondary);">{{ row.cause }}</td>
            <td class="px-5 py-3 text-xs align-top" style="color: var(--accent-primary);">{{ row.fix }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- API -->
    <div v-else class="rounded-lg border overflow-hidden" style="background: var(--bg-secondary); border-color: var(--border-color);">
      <div class="px-5 py-3.5 border-b text-sm font-semibold" style="border-color: var(--border-color); color: var(--text-primary);">
        /api/ean/*（JWT 保护）
      </div>
      <div class="divide-y" style="divide-color: var(--border-color);">
        <div
          v-for="(api, i) in eanHttpApiQuickRef"
          :key="i"
          class="px-5 py-3 flex items-start gap-3"
        >
          <span
            class="text-xs font-mono px-2 py-0.5 rounded-md flex-shrink-0"
            :style="{
              background: api.method === 'GET' ? 'rgba(16,185,129,0.12)' : 'rgba(14,165,233,0.12)',
              color: api.method === 'GET' ? '#10B981' : '#0EA5E9',
            }"
          >
            {{ api.method }}
          </span>
          <div class="min-w-0">
            <code class="text-xs font-mono" style="color: var(--text-primary);">{{ api.path }}</code>
            <p class="text-xs mt-0.5" style="color: var(--text-muted);">{{ api.note }}</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
