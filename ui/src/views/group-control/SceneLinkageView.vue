<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
import { Sparkles, ArrowLeft, Clock, Zap, CheckCircle2 } from 'lucide-vue-next'
import { useP3Page } from '@/composables/useP3Page'
import { useGroupControlStore } from '@/stores/groupControl'
import { useLiveClock } from '@/composables/useLiveClock'
import AnimatedNumber from '@/components/p3/AnimatedNumber.vue'
import LiveMetricStrip from '@/components/p3/LiveMetricStrip.vue'
import LiveSparkline from '@/components/p3/LiveSparkline.vue'
import BarMeter from '@/components/p3/BarMeter.vue'
import P3ActionCenter from '@/components/p3/P3ActionCenter.vue'
import P3AuditTable from '@/components/p3/P3AuditTable.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'
import ScenarioFlowBoard from '@/components/p3/ScenarioFlowBoard.vue'

const { page } = useP3Page('scenario-linkage')
const store = useGroupControlStore()
const { dateTime } = useLiveClock()

function sceneColor(status: string) {
  if (status === 'risk') return '#EF4444'
  if (status === 'watch') return '#F59E0B'
  return '#10B981'
}

function sceneLabel(status: string) {
  if (status === 'risk') return '高风险'
  if (status === 'watch') return '观察'
  return '稳定'
}

const triggerPercent = computed(() => store.triggerCount % 100)

onMounted(() => store.start())
onUnmounted(() => store.stop())
</script>

<template>
  <div class="space-y-5">
    <section class="scene-top rounded-2xl border p-5">
      <div>
        <router-link
          to="/group-control"
          class="mb-3 inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm transition-colors hover:bg-white/5"
          style="color: var(--text-secondary); border: 1px solid var(--border-color);"
        >
          <ArrowLeft class="h-4 w-4" />
          返回群控管理
        </router-link>
        <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs scene-chip">
          <Sparkles class="h-3.5 w-3.5" />
          ECA 场景引擎
        </div>
        <div class="mt-3 flex items-center gap-3">
          <h1 class="text-3xl font-semibold scene-title">{{ page.title }}</h1>
          <span class="inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1 font-mono-num text-xs" style="color: var(--text-secondary); border-color: var(--border-color); background: var(--bg-tertiary);">
            <Clock class="h-3.5 w-3.5" style="color: #10B981;" />
            {{ dateTime }}
          </span>
        </div>
        <p class="mt-2 text-sm scene-subtitle">{{ page.subtitle }}</p>

        <!-- 联动域实时指标 -->
        <div class="mt-5">
          <LiveMetricStrip
            :items="[
              { label: '联动规则', value: 34, unit: '条', color: '#0EA5E9' },
              { label: '今日触发', value: store.triggerCount, unit: '次', color: '#10B981', pulse: true },
              { label: '成功率', value: store.sceneSuccessRate, unit: '%', decimals: 1, color: '#8B5CF6', pulse: true },
              { label: '失败链路', value: store.watchScenes, unit: '条', color: '#EF4444', pulse: true },
              { label: '平均触发时延', value: 19, unit: 'ms', color: '#F59E0B' },
              { label: '动作回执率', value: 96, unit: '%', color: '#10B981' },
            ]"
          />
        </div>
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[1.05fr_0.95fr]">
      <div class="space-y-4">
        <ScenarioFlowBoard :title="page.flowTitle" :nodes="page.flowNodes" />

        <section class="rounded-xl border overflow-hidden" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <div class="border-b px-5 py-4" style="border-color: var(--border-color);">
            <div class="flex items-center justify-between">
              <div>
                <h3 class="text-sm font-semibold scene-title">规则链列表</h3>
                <p class="mt-1 text-xs scene-subtitle">查看事件、条件、动作和执行状态</p>
              </div>
              <span class="inline-flex items-center gap-1.5 text-[11px] text-emerald-500">
                <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
                实时
              </span>
            </div>
          </div>
          <div class="overflow-x-auto">
            <table class="min-w-full text-sm">
              <thead style="background: var(--bg-tertiary);">
                <tr>
                  <th class="px-4 py-3 text-left text-xs font-semibold" style="color: var(--text-secondary);">规则名</th>
                  <th class="px-4 py-3 text-left text-xs font-semibold" style="color: var(--text-secondary);">触发事件</th>
                  <th class="px-4 py-3 text-left text-xs font-semibold" style="color: var(--text-secondary);">条件</th>
                  <th class="px-4 py-3 text-left text-xs font-semibold" style="color: var(--text-secondary);">动作</th>
                  <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">Quality</th>
                  <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">状态</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="scene in store.scenes"
                  :key="scene.id"
                  class="border-b"
                  :style="{ borderColor: 'var(--border-color)' }"
                >
                  <td class="px-4 py-3">
                    <span class="inline-flex items-center gap-2">
                      <span class="inline-block h-2.5 w-2.5 rounded-full" :style="{ background: sceneColor(scene.status) }" />
                      <span style="color: var(--text-primary);">{{ scene.name }}</span>
                    </span>
                  </td>
                  <td class="px-4 py-3 text-left" style="color: var(--text-secondary);">{{ scene.event }}</td>
                  <td class="px-4 py-3 text-left font-mono-num text-xs" style="color: var(--text-secondary);">{{ scene.condition }}</td>
                  <td class="px-4 py-3 text-left" style="color: var(--text-secondary);">{{ scene.action }}</td>
                  <td class="px-4 py-3 text-center font-mono-num" style="color: var(--text-secondary);">
                    <AnimatedNumber :value="scene.quality" />
                  </td>
                  <td class="px-4 py-3 text-center">
                    <span
                      class="inline-block rounded-lg px-2.5 py-1 text-xs"
                      :style="{ color: sceneColor(scene.status), background: `${sceneColor(scene.status)}1a` }"
                    >
                      {{ sceneLabel(scene.status) }}
                    </span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </div>

      <div class="space-y-4">
        <!-- 触发实时面板 -->
        <section class="rounded-2xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <div class="flex items-center justify-between">
            <h3 class="text-sm font-semibold scene-title">触发节奏面板</h3>
            <span class="inline-flex items-center gap-1.5 text-[11px] text-emerald-500">
              <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
              实时
            </span>
          </div>
          <div class="mt-4 grid grid-cols-2 gap-3">
            <div class="rounded-lg border px-3 py-3 text-center" style="background: var(--bg-tertiary); border-color: var(--border-color);">
              <div class="flex items-center justify-center gap-1 text-[11px] scene-subtitle">
                <Zap class="h-3.5 w-3.5 text-emerald-500" />
                今日触发
              </div>
              <div class="mt-1 font-mono-num text-2xl font-semibold" style="color: #10B981;">
                <AnimatedNumber :value="store.triggerCount" />
              </div>
            </div>
            <div class="rounded-lg border px-3 py-3 text-center" style="background: var(--bg-tertiary); border-color: var(--border-color);">
              <div class="flex items-center justify-center gap-1 text-[11px] scene-subtitle">
                <CheckCircle2 class="h-3.5 w-3.5 text-violet-500" />
                成功率
              </div>
              <div class="mt-1 font-mono-num text-2xl font-semibold" style="color: #8B5CF6;">
                <AnimatedNumber :value="store.sceneSuccessRate" decimals="1" suffix="%" />
              </div>
            </div>
          </div>
          <div class="mt-4">
            <LiveSparkline :points="store.triggerSeries" color="#8B5CF6" :height="64" />
          </div>
          <div class="mt-3">
            <div class="mb-1 flex justify-between text-[11px]">
              <span class="scene-subtitle">失败链路</span>
              <span class="font-mono-num" :style="{ color: store.watchScenes > 0 ? '#EF4444' : '#10B981' }">
                <AnimatedNumber :value="store.watchScenes" /> 条
              </span>
            </div>
            <BarMeter :value="triggerPercent" :max="100" color="#8B5CF6" :height="5" />
          </div>
        </section>

        <P3EventFeed :title="page.sidePanelTitle" :events="store.domainEvents('scene')" />
        <P3ActionCenter :title="page.actionTitle" :actions="page.actions" />
      </div>
    </section>

    <P3AuditTable :records="page.auditRecords" title="规则审计日志" />
  </div>
</template>

<style scoped>
.scene-top {
  background:
    linear-gradient(90deg, rgba(16, 185, 129, 0.08), transparent 50%),
    var(--bg-secondary);
  border-color: var(--border-color);
}

.scene-chip {
  background: rgba(16, 185, 129, 0.08);
  color: #10B981;
}

.scene-title { color: var(--text-primary); }
.scene-subtitle { color: var(--text-secondary); }
</style>
