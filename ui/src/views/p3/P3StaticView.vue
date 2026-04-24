<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import BusinessKpiStrip from '@/components/p3/BusinessKpiStrip.vue'
import DispatchQueueTable from '@/components/p3/DispatchQueueTable.vue'
import ExecutionTimeline from '@/components/p3/ExecutionTimeline.vue'
import HealthMetricCard from '@/components/p3/HealthMetricCard.vue'
import LedgerStatusTable from '@/components/p3/LedgerStatusTable.vue'
import QualityTrendCard from '@/components/p3/QualityTrendCard.vue'
import ScenarioFlowBoard from '@/components/p3/ScenarioFlowBoard.vue'
import { getP3PageData } from '@/mock/p3'
import type { P3ActionPreset, P3PageData, P3Tone } from '@/types/p3'

const route = useRoute()
const activeAction = ref<P3ActionPreset | null>(null)

const toneMap: Record<P3Tone, { color: string; bg: string; border: string }> = {
  sky: { color: '#0EA5E9', bg: 'rgba(14,165,233,0.08)', border: 'rgba(14,165,233,0.24)' },
  emerald: { color: '#10B981', bg: 'rgba(16,185,129,0.08)', border: 'rgba(16,185,129,0.24)' },
  amber: { color: '#F59E0B', bg: 'rgba(245,158,11,0.08)', border: 'rgba(245,158,11,0.24)' },
  red: { color: '#EF4444', bg: 'rgba(239,68,68,0.08)', border: 'rgba(239,68,68,0.24)' },
  violet: { color: '#8B5CF6', bg: 'rgba(139,92,246,0.08)', border: 'rgba(139,92,246,0.24)' },
  slate: { color: '#64748B', bg: 'rgba(100,116,139,0.08)', border: 'rgba(100,116,139,0.24)' },
}

const page = computed<P3PageData>(() => {
  const pageKey = String(route.meta.pageKey || '')
  return getP3PageData(pageKey)
})

const isLedger = computed(() => page.value.key === 'ledger')

function openActionPreview(action: P3ActionPreset) {
  activeAction.value = action
}

function actionStyle(tone: P3Tone) {
  return toneMap[tone]
}
</script>

<template>
  <div class="space-y-5">
    <section class="rounded-xl border p-5" style="background: var(--bg-secondary); border-color: var(--border-color);">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div>
          <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs" style="background: var(--bg-tertiary); color: var(--text-secondary);">
            <span>{{ page.section }}</span>
            <span>/</span>
            <span>{{ page.module }}</span>
          </div>
          <h1 class="mt-3 text-2xl font-semibold" style="color: var(--text-primary);">{{ page.title }}</h1>
          <p class="mt-2 max-w-3xl text-sm" style="color: var(--text-secondary);">{{ page.subtitle }}</p>
        </div>
        <div class="grid grid-cols-2 gap-2 text-xs md:grid-cols-3 xl:min-w-[420px]">
          <div class="rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
            <div style="color: var(--text-muted);">系统态</div>
            <div class="mt-1 font-mono-num" style="color: var(--text-primary);">{{ page.statusStrip.system }}</div>
          </div>
          <div class="rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
            <div style="color: var(--text-muted);">Devices</div>
            <div class="mt-1 font-mono-num" style="color: var(--text-primary);">{{ page.statusStrip.devices }}</div>
          </div>
          <div class="rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
            <div style="color: var(--text-muted);">Alarm</div>
            <div class="mt-1 font-mono-num" style="color: var(--text-primary);">{{ page.statusStrip.alarms }}</div>
          </div>
          <div class="rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
            <div style="color: var(--text-muted);">Latency</div>
            <div class="mt-1 font-mono-num" style="color: var(--text-primary);">{{ page.statusStrip.latency }}ms</div>
          </div>
          <div class="rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
            <div style="color: var(--text-muted);">Loss</div>
            <div class="mt-1 font-mono-num" style="color: var(--text-primary);">{{ page.statusStrip.loss }}%</div>
          </div>
          <div class="rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
            <div style="color: var(--text-muted);">Quality</div>
            <div class="mt-1 font-mono-num" style="color: var(--text-primary);">{{ page.statusStrip.quality }}</div>
          </div>
        </div>
      </div>
    </section>

    <BusinessKpiStrip :items="page.kpis" />

    <section class="grid grid-cols-1 gap-4 xl:grid-cols-4">
      <HealthMetricCard v-for="metric in page.metrics" :key="metric.label" :item="metric" />
    </section>

    <section class="grid grid-cols-1 gap-4 xl:grid-cols-3">
      <QualityTrendCard v-for="trend in page.trends" :key="trend.title" :item="trend" />
    </section>

    <section class="grid grid-cols-1 gap-4 xl:grid-cols-12">
      <div class="space-y-4 xl:col-span-8">
        <ScenarioFlowBoard :title="page.flowTitle" :nodes="page.flowNodes" />
        <component :is="isLedger ? LedgerStatusTable : DispatchQueueTable" :table="page.mainTable" />
      </div>

      <div class="space-y-4 xl:col-span-4">
        <section class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <h3 class="text-sm font-semibold" style="color: var(--text-primary);">{{ page.sidePanelTitle }}</h3>
          <div class="mt-4 space-y-3">
            <article
              v-for="event in page.sideEvents"
              :key="`${event.title}-${event.meta}`"
              class="rounded-lg border p-3"
              style="background: var(--bg-tertiary); border-color: var(--border-color);"
            >
              <div class="flex items-center justify-between gap-2">
                <div class="text-sm font-medium" style="color: var(--text-primary);">{{ event.title }}</div>
                <span class="text-[11px]" style="color: var(--accent-primary);">{{ event.status }}</span>
              </div>
              <p class="mt-1 text-xs" style="color: var(--text-secondary);">{{ event.subtitle }}</p>
              <p class="mt-2 text-[11px]" style="color: var(--text-muted);">{{ event.meta }}</p>
            </article>
          </div>
        </section>

        <ExecutionTimeline :title="page.timelineTitle" :items="page.timeline" />

        <section class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <h3 class="text-sm font-semibold" style="color: var(--text-primary);">{{ page.actionTitle }}</h3>
          <div class="mt-4 space-y-3">
            <button
              v-for="action in page.actions"
              :key="action.label"
              class="w-full rounded-xl border px-4 py-3 text-left transition-colors"
              :style="{
                color: actionStyle(action.tone).color,
                background: actionStyle(action.tone).bg,
                borderColor: actionStyle(action.tone).border,
              }"
              @click="openActionPreview(action)"
            >
              <div class="flex items-center justify-between gap-3">
                <span class="text-sm font-medium">{{ action.label }}</span>
                <span class="rounded-md px-2 py-1 text-[11px]" style="background: rgba(255,255,255,0.12); color: inherit;">
                  {{ action.level }}
                </span>
              </div>
              <p class="mt-1 text-xs" style="color: var(--text-secondary);">{{ action.description }}</p>
            </button>
          </div>

          <div v-if="activeAction" class="mt-4 rounded-xl border p-4" style="background: var(--bg-tertiary); border-color: var(--border-color);">
            <div class="flex items-center justify-between gap-2">
              <div>
                <div class="text-sm font-medium" style="color: var(--text-primary);">{{ activeAction.label }}</div>
                <div class="mt-1 text-xs" style="color: var(--text-secondary);">{{ activeAction.description }}</div>
              </div>
              <span
                class="rounded-md px-2 py-1 text-[11px]"
                :style="{ color: actionStyle(activeAction.tone).color, background: actionStyle(activeAction.tone).bg }"
              >
                {{ activeAction.level }}
              </span>
            </div>
            <div class="mt-4 rounded-lg border p-3" style="border-color: var(--border-color);">
              <div class="text-xs" style="color: var(--text-muted);">模拟确认区</div>
              <template v-if="activeAction.level === 'L2'">
                <div class="mt-2 rounded-lg px-3 py-2 text-sm" style="background: rgba(14,165,233,0.08); color: #0EA5E9;">Toast: 配置已记录，等待提交</div>
              </template>
              <template v-else-if="activeAction.level === 'L3'">
                <div class="mt-2 text-sm" style="color: var(--text-primary);">Confirm: 请确认下发目标与风险边界后继续。</div>
              </template>
              <template v-else-if="activeAction.level === 'L4'">
                <div class="mt-2 text-sm" style="color: var(--text-primary);">Confirm + 倒计时: 2s 后允许执行，避免误触。</div>
              </template>
              <template v-else-if="activeAction.level === 'L5'">
                <div class="mt-2 text-sm" style="color: var(--text-primary);">输入确认词后才可执行。</div>
                <input
                  class="mt-3 w-full rounded-lg border px-3 py-2 text-sm"
                  :placeholder="`输入 ${activeAction.confirmText || 'CONFIRM'} 确认`"
                  style="background: var(--bg-secondary); border-color: var(--border-color); color: var(--text-primary);"
                />
              </template>
            </div>
          </div>
        </section>

        <section class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <h3 class="text-sm font-semibold" style="color: var(--text-primary);">操作审计</h3>
          <div class="mt-4 overflow-x-auto">
            <table class="min-w-full text-xs">
              <thead>
                <tr style="color: var(--text-muted);">
                  <th class="px-2 py-2 text-left">User</th>
                  <th class="px-2 py-2 text-left">Action</th>
                  <th class="px-2 py-2 text-left">Target</th>
                  <th class="px-2 py-2 text-left">Timestamp</th>
                  <th class="px-2 py-2 text-left">Result</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="record in page.auditRecords"
                  :key="`${record.user}-${record.timestamp}-${record.action}`"
                  class="border-t"
                  :style="{ borderColor: 'var(--border-color)' }"
                >
                  <td class="px-2 py-2 font-mono-num" style="color: var(--text-primary);">{{ record.user }}</td>
                  <td class="px-2 py-2" style="color: var(--text-secondary);">{{ record.action }}</td>
                  <td class="px-2 py-2" style="color: var(--text-secondary);">{{ record.target }}</td>
                  <td class="px-2 py-2 font-mono-num" style="color: var(--text-secondary);">{{ record.timestamp }}</td>
                  <td class="px-2 py-2" style="color: var(--accent-primary);">{{ record.result }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </div>
    </section>
  </div>
</template>
