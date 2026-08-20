<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
import { Gauge, AlertTriangle } from 'lucide-vue-next'
import { useVisualStore } from '@/stores/visual'
import { useLiveClock } from '@/composables/useLiveClock'
import AnimatedNumber from '@/components/p3/AnimatedNumber.vue'
import LiveMetricStrip from '@/components/p3/LiveMetricStrip.vue'
import LiveSparkline from '@/components/p3/LiveSparkline.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'
import ArcGauge from '@/components/visual/ArcGauge.vue'
import VisualScreen from '@/components/visual/VisualScreen.vue'

const store = useVisualStore()
const { dateTime } = useLiveClock()

const primaryGauges = computed(() => store.gauges.filter(g => ['pressure', 'temp', 'flow', 'level'].includes(g.key)))
const secondaryGauges = computed(() => store.gauges.filter(g => !['pressure', 'temp', 'flow', 'level'].includes(g.key)))

const sceneEvents = computed(() =>
  store.events.map(e => ({
    title: e.title,
    subtitle: e.subtitle,
    meta: e.meta,
    status: e.status === 'error' ? 'error' : e.status === 'warning' ? 'warning' : 'resolved',
  }))
)

onMounted(() => store.start())
onUnmounted(() => store.stop())
</script>

<template>
  <VisualScreen>
    <div class="space-y-5">
      <section class="rounded-2xl border p-5 gauge-hero">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs gauge-pill">
            <Gauge class="h-3.5 w-3.5" />
            仪表监控 · 实时工艺参数
          </div>
          <div class="mt-2 flex items-center gap-3">
            <h1 class="text-2xl font-semibold" style="color: var(--text-primary);">仪表监控</h1>
            <span class="font-mono-num text-xs" style="color: var(--text-secondary);">{{ dateTime }}</span>
          </div>
        </div>
        <div
          class="flex items-center gap-2 rounded-lg border px-3 py-1.5 text-xs"
          :style="store.busyGauges > 0
            ? { background: 'rgba(245,158,11,0.1)', borderColor: 'rgba(245,158,11,0.4)', color: '#F59E0B' }
            : { background: 'rgba(16,185,129,0.1)', borderColor: 'rgba(16,185,129,0.4)', color: '#10B981' }"
        >
          <AlertTriangle class="h-3.5 w-3.5" />
          {{ store.busyGauges > 0 ? `${store.busyGauges} 项接近上限` : '全部仪表正常' }}
        </div>
      </div>

      <div class="mt-4">
        <LiveMetricStrip
          :items="[
            { label: '管道压力', value: store.gauges.find(g => g.key === 'pressure')?.value ?? 0, decimals: 2, unit: 'MPa', color: '#0EA5E9' },
            { label: '炉膛温度', value: store.gauges.find(g => g.key === 'temp')?.value ?? 0, unit: '℃', color: '#F59E0B' },
            { label: '瞬时流量', value: store.gauges.find(g => g.key === 'flow')?.value ?? 0, decimals: 1, unit: 'm³/h', color: '#10B981' },
            { label: '储罐液位', value: store.gauges.find(g => g.key === 'level')?.value ?? 0, unit: '%', color: '#8B5CF6' },
          ]"
        />
      </div>
    </section>

    <section class="rounded-2xl border p-5" style="background: var(--bg-secondary); border-color: var(--border-color);">
      <div class="flex items-center justify-between">
        <h3 class="text-sm font-semibold" style="color: var(--text-primary);">工艺主仪表</h3>
        <span class="inline-flex items-center gap-1.5 text-[11px] text-emerald-500">
          <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
          实时采样
        </span>
      </div>
      <div class="mt-4 grid grid-cols-2 gap-x-2 gap-y-4 md:grid-cols-4">
        <div v-for="g in primaryGauges" :key="g.key" class="flex flex-col items-center">
          <ArcGauge
            :label="g.label"
            :value="g.value"
            :min="g.min"
            :max="g.max"
            :unit="g.unit"
            :color="g.color"
            :size="190"
            :decimals="g.max > 100 ? 0 : 2"
            zone
          />
          <div class="mt-1 text-[11px]" style="color: var(--text-secondary);">
            量程 <span class="font-mono-num">{{ g.min }} ~ {{ g.max }} {{ g.unit }}</span>
          </div>
        </div>
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
      <article class="rounded-2xl border p-5" style="background: var(--bg-secondary); border-color: var(--border-color);">
        <div class="flex items-center gap-2">
          <Gauge class="h-4 w-4" style="color: #8B5CF6;" />
          <h3 class="text-sm font-semibold" style="color: var(--text-primary);">辅助监测仪表</h3>
        </div>
        <div class="mt-4 grid grid-cols-2 gap-x-2 gap-y-4 md:grid-cols-3">
          <div v-for="g in secondaryGauges" :key="g.key" class="flex flex-col items-center">
            <ArcGauge
              :label="g.label"
              :value="g.value"
              :min="g.min"
              :max="g.max"
              :unit="g.unit"
              :color="g.color"
              :size="150"
              :decimals="g.max < 100 ? 1 : 0"
              zone
            />
          </div>
        </div>
      </article>

      <article class="rounded-2xl border p-5" style="background: var(--bg-secondary); border-color: var(--border-color);">
        <div class="flex items-center justify-between">
          <div>
            <h3 class="text-sm font-semibold" style="color: var(--text-primary);">压力脉动</h3>
            <p class="mt-1 text-xs" style="color: var(--text-secondary);">管道压力近 30 个采样点</p>
          </div>
          <span class="inline-flex items-center gap-1.5 text-[11px] text-emerald-500">
            <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
            实时
          </span>
        </div>
        <div class="mt-3">
          <LiveSparkline
            :points="store.pressureSeries"
            color="#0EA5E9"
            :height="90"
          />
        </div>
        <div class="mt-4 grid grid-cols-3 gap-3 text-center">
          <div class="rounded-lg border px-3 py-2.5" style="background: var(--bg-tertiary); border-color: var(--border-color);">
            <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">上限占比</div>
            <div class="mt-1 font-mono-num text-base font-semibold" style="color: #F59E0B;">
              <AnimatedNumber :value="(store.gauges.find(g => g.key === 'pressure')?.value ?? 0) * 100" decimals="0" suffix="%" />
            </div>
          </div>
          <div class="rounded-lg border px-3 py-2.5" style="background: var(--bg-tertiary); border-color: var(--border-color);">
            <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">均值</div>
            <div class="mt-1 font-mono-num text-base font-semibold" style="color: #0EA5E9;">0.58</div>
          </div>
          <div class="rounded-lg border px-3 py-2.5" style="background: var(--bg-tertiary); border-color: var(--border-color);">
            <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">峰值</div>
            <div class="mt-1 font-mono-num text-base font-semibold" style="color: #EF4444;">0.82</div>
          </div>
        </div>
      </article>
    </section>

    <P3EventFeed title="仪表事件流" :events="sceneEvents" />
    </div>
  </VisualScreen>
</template>

<style scoped>
.gauge-hero {
  background:
    linear-gradient(90deg, rgba(139, 92, 246, 0.07), transparent 40%),
    var(--bg-secondary);
  border-color: var(--border-color);
}

.gauge-pill {
  background: rgba(139, 92, 246, 0.08);
  color: #8B5CF6;
}
</style>
