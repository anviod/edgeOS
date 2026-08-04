<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
import { Activity, BarChart3, ArrowLeft, Clock, TrendingDown, TrendingUp } from 'lucide-vue-next'
import { useP3Page } from '@/composables/useP3Page'
import { useBusinessStore } from '@/stores/business'
import { useLiveClock } from '@/composables/useLiveClock'
import AnimatedNumber from '@/components/p3/AnimatedNumber.vue'
import LiveSparkline from '@/components/p3/LiveSparkline.vue'
import LiveMetricStrip from '@/components/p3/LiveMetricStrip.vue'
import ScenarioFlowBoard from '@/components/p3/ScenarioFlowBoard.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'

const { page } = useP3Page('energy-monitoring')
const store = useBusinessStore()
const { dateTime } = useLiveClock()

// 最近 24 拍，用于横向峰谷柱状图（波动更平稳，贴合真实负荷曲线）
const railPoints = computed(() => store.energySeries.slice(-24))

const avgRail = computed(() => {
  if (!railPoints.value.length) return 0
  return Math.round(railPoints.value.reduce((a, b) => a + b, 0) / railPoints.value.length)
})

const peakPoint = computed(() => Math.round(Math.max(...railPoints.value, 0)))
const valleyPoint = computed(() => Math.round(Math.min(...railPoints.value, 0)))

onMounted(() => store.start())
onUnmounted(() => store.stop())
</script>

<template>
  <div class="space-y-5">
    <!-- 顶部面板：标题 + 指标条 -->
    <section class="monitor-hero rounded-2xl border p-5">
      <router-link
        to="/business-center"
        class="mb-3 inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm transition-colors hover:bg-white/5"
        style="color: var(--text-secondary); border: 1px solid var(--border-color);"
      >
        <ArrowLeft class="h-4 w-4" />
        返回业务中心
      </router-link>
      <div class="flex flex-wrap items-center gap-3">
        <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs monitor-chip">
          <Activity class="h-3.5 w-3.5" />
          能流观察哨
        </div>
        <h1 class="text-3xl font-semibold monitor-title">{{ page.title }}</h1>
        <span class="inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1 font-mono-num text-xs" style="color: var(--text-secondary); border-color: var(--border-color); background: var(--bg-tertiary);">
          <Clock class="h-3.5 w-3.5" style="color: #0EA5E9;" />
          {{ dateTime }}
        </span>
      </div>
      <p class="mt-2 text-sm monitor-subtitle">{{ page.subtitle }}</p>

      <!-- 能耗域实时指标 -->
      <div class="mt-5">
        <LiveMetricStrip
          :items="[
            { label: '监测回路', value: 37, unit: '条', color: '#0EA5E9' },
            { label: '今日总能耗', value: store.totalEnergy, unit: 'MWh', decimals: 1, color: '#10B981', pulse: true },
            { label: '峰时负载', value: 83, unit: '%', color: '#F59E0B', pulse: true },
            { label: '损耗比', value: 4.8, unit: '%', decimals: 1, color: '#F59E0B' },
            { label: 'Latency', value: store.avgLoopLatency, unit: 'ms', color: '#8B5CF6' },
            { label: 'Quality', value: store.avgLoopQuality, color: '#10B981' },
          ]"
        />
      </div>
    </section>

    <!-- 峰谷观察栏：整行横排，位于顶部面板下方 -->
    <section class="rounded-2xl border p-5 monitor-rail">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex items-center gap-2">
          <BarChart3 class="h-4 w-4 text-sky-500" />
          <span class="text-sm font-medium monitor-title">峰谷观察栏</span>
          <span class="inline-flex items-center gap-1.5 text-[11px] text-emerald-500">
            <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
            实时
          </span>
        </div>
        <div class="flex items-center gap-3 text-xs">
          <span class="inline-flex items-center gap-1 font-mono-num" style="color: #EF4444;">
            <TrendingUp class="h-3.5 w-3.5" />
            峰值 <AnimatedNumber :value="peakPoint" />
          </span>
          <span class="inline-flex items-center gap-1 font-mono-num" style="color: #0EA5E9;">
            <TrendingDown class="h-3.5 w-3.5" />
            谷值 <AnimatedNumber :value="valleyPoint" />
          </span>
          <span class="inline-flex items-center gap-1 font-mono-num" style="color: var(--text-secondary);">
            均值 <AnimatedNumber :value="avgRail" />
          </span>
        </div>
      </div>

      <div class="mt-5 flex h-40 items-end gap-1.5">
        <div
          v-for="(point, index) in railPoints"
          :key="`rail-${index}`"
          class="group relative flex h-full flex-1 flex-col justify-end"
        >
          <div
            class="w-full rounded-t transition-all duration-700"
            :style="{
              height: `${point}%`,
              background: point > 85 ? 'linear-gradient(180deg, rgba(239,68,68,0.85), rgba(239,68,68,0.35))' : 'linear-gradient(180deg, rgba(14,165,233,0.8), rgba(14,165,233,0.25))',
            }"
          />
          <div class="absolute bottom-full left-1/2 mb-1 -translate-x-1/2 whitespace-nowrap rounded px-1.5 py-0.5 text-[10px] font-mono-num opacity-0 transition-opacity group-hover:opacity-100" style="background: var(--bg-secondary); border: 1px solid var(--border-color); color: var(--text-primary);">
            {{ point }}
          </div>
        </div>
      </div>

      <div class="mt-2 flex justify-between text-[10px] font-mono-num" style="color: var(--text-muted);">
        <span>24 拍前</span>
        <span>实时</span>
      </div>

      <div class="mt-4 grid gap-3 md:grid-cols-3">
        <div class="rounded-lg border px-3 py-2 text-xs" style="background: var(--bg-tertiary); border-color: var(--border-color);">
          <div class="monitor-subtitle">今日累计能耗</div>
          <div class="mt-1 font-mono-num font-semibold text-emerald-500">
            <AnimatedNumber :value="store.totalEnergy" decimals="1" suffix=" MWh" />
          </div>
        </div>
        <div class="rounded-lg border px-3 py-2 text-xs" style="background: var(--bg-tertiary); border-color: var(--border-color);">
          <div class="monitor-subtitle">削峰空间</div>
          <div class="mt-1 font-mono-num font-semibold" style="color: #0EA5E9;">31%</div>
        </div>
        <div class="rounded-lg border px-3 py-2 text-xs" style="background: var(--bg-tertiary); border-color: var(--border-color);">
          <div class="monitor-subtitle">损耗热点</div>
          <div class="mt-1 font-mono-num font-semibold" style="color: #EF4444;">2 处</div>
        </div>
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[1.05fr_0.95fr]">
      <div class="space-y-4">
        <div class="grid gap-4 xl:grid-cols-2">
          <article class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
            <div class="flex items-center justify-between">
              <div>
                <h3 class="text-sm font-semibold monitor-title">峰谷趋势</h3>
                <p class="mt-1 text-xs monitor-subtitle">高峰削减初见成效</p>
              </div>
              <span class="inline-flex items-center gap-1.5 text-[11px] text-emerald-500">
                <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
                实时
              </span>
            </div>
            <div class="mt-3">
              <LiveSparkline :points="store.energySeries" color="#0EA5E9" :height="72" />
            </div>
          </article>
          <article class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
            <div class="flex items-center justify-between">
              <div>
                <h3 class="text-sm font-semibold monitor-title">回路质量评分</h3>
                <p class="mt-1 text-xs monitor-subtitle">老旧支路质量偏低</p>
              </div>
              <span class="inline-flex items-center gap-1.5 text-[11px] text-sky-500">
                <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-sky-500" />
                实时
              </span>
            </div>
            <div class="mt-3">
              <LiveSparkline :points="store.energySeries.map(v => Math.round(88 + (v % 10)))" color="#8B5CF6" :height="72" />
            </div>
          </article>
        </div>

        <section class="rounded-xl border overflow-hidden" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <div class="border-b px-5 py-4" style="border-color: var(--border-color);">
            <h3 class="text-sm font-semibold monitor-title">分站点 / 分回路监测</h3>
            <p class="mt-1 text-xs monitor-subtitle">{{ page.mainTable.description }}</p>
          </div>
          <div class="overflow-x-auto">
            <table class="min-w-full text-sm">
              <thead style="background: var(--bg-tertiary);">
                <tr>
                  <th class="px-4 py-3 text-left text-xs font-semibold" style="color: var(--text-secondary);">回路 / 站点</th>
                  <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">能耗</th>
                  <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">Quality</th>
                  <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">Latency</th>
                  <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">Loss</th>
                  <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">状态</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="loop in store.loops"
                  :key="loop.id"
                  class="border-b"
                  :style="{ borderColor: 'var(--border-color)' }"
                >
                  <td class="px-4 py-3">
                    <span class="inline-flex items-center gap-2">
                      <span
                        class="inline-block h-2.5 w-2.5 rounded-full"
                        :style="{ background: loop.status === '异常波动' ? '#EF4444' : loop.status === '观察' ? '#F59E0B' : '#10B981' }"
                      />
                      <span style="color: var(--text-primary);">{{ loop.name }}</span>
                    </span>
                  </td>
                  <td class="px-4 py-3 text-center font-mono-num" style="color: var(--text-secondary);">
                    <AnimatedNumber :value="loop.energy" decimals="1" suffix="MWh" />
                  </td>
                  <td class="px-4 py-3 text-center font-mono-num" style="color: var(--text-secondary);">
                    <AnimatedNumber :value="loop.quality" />
                  </td>
                  <td class="px-4 py-3 text-center font-mono-num" style="color: var(--text-secondary);">
                    <AnimatedNumber :value="loop.latency" suffix="ms" />
                  </td>
                  <td class="px-4 py-3 text-center font-mono-num" style="color: var(--text-secondary);">{{ loop.loss }}%</td>
                  <td class="px-4 py-3 text-center">
                    <span
                      class="inline-block rounded-lg px-2.5 py-1 text-xs"
                      :style="{
                        color: loop.status === '异常波动' ? '#EF4444' : loop.status === '观察' ? '#F59E0B' : '#10B981',
                        background: loop.status === '异常波动' ? 'rgba(239,68,68,0.1)' : loop.status === '观察' ? 'rgba(245,158,11,0.1)' : 'rgba(16,185,129,0.1)',
                      }"
                    >
                      {{ loop.status }}
                    </span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </div>

      <div class="space-y-4">
        <ScenarioFlowBoard :title="page.flowTitle" :nodes="page.flowNodes" />
        <P3EventFeed :title="page.sidePanelTitle" :events="store.domainEvents('energy')" />
      </div>
    </section>
  </div>
</template>

<style scoped>
.monitor-hero,
.monitor-rail {
  background: var(--bg-secondary);
  border-color: var(--border-color);
}

.monitor-hero {
  background:
    linear-gradient(180deg, rgba(14, 165, 233, 0.06), transparent 28%),
    var(--bg-secondary);
}

.monitor-chip {
  background: rgba(14, 165, 233, 0.08);
  color: #0EA5E9;
}

.monitor-title { color: var(--text-primary); }
.monitor-subtitle { color: var(--text-secondary); }
</style>
