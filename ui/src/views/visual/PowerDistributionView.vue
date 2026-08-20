<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
import { Zap, Activity, BatteryCharging } from 'lucide-vue-next'
import { useVisualStore, type FeederStatus } from '@/stores/visual'
import { useLiveClock } from '@/composables/useLiveClock'
import AnimatedNumber from '@/components/p3/AnimatedNumber.vue'
import LiveMetricStrip from '@/components/p3/LiveMetricStrip.vue'
import LiveSparkline from '@/components/p3/LiveSparkline.vue'
import BarMeter from '@/components/p3/BarMeter.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'
import VisualScreen from '@/components/visual/VisualScreen.vue'
import IsoScene from '@/components/visual/IsoScene.vue'
import IsoCube from '@/components/visual/IsoCube.vue'
import IsoPylon from '@/components/visual/IsoPylon.vue'
import IsoFlowDot from '@/components/visual/IsoFlowDot.vue'

const store = useVisualStore()
const { dateTime } = useLiveClock()

const FEEDER_X = [300, 480, 660, 840]
const SHORT_NAME: Record<string, string> = { F1: '储能', F2: '数据中心', F3: '产线', F4: '充电站' }

function statusColor(status: FeederStatus) {
  if (status === 'fault') return '#EF4444'
  if (status === 'warn') return '#F59E0B'
  return '#0EA5E9'
}

function statusText(status: FeederStatus) {
  if (status === 'fault') return '过载'
  if (status === 'warn') return '预警'
  return '正常'
}

const avgLoad = computed(() => {
  const sum = store.feeders.reduce((a, f) => a + f.load, 0)
  return Math.round(sum / store.feeders.length)
})

const warnFeeders = computed(() => store.feeders.filter(f => f.status !== 'normal').length)

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
      <section class="rounded-2xl border p-5 pd-hero">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs pd-pill">
              <Zap class="h-3.5 w-3.5" />
              输配电 · 10kV 配电系统
            </div>
            <div class="mt-2 flex items-center gap-3">
              <h1 class="text-2xl font-semibold" style="color: var(--text-primary);">输配电</h1>
              <span class="font-mono-num text-xs" style="color: var(--text-secondary);">{{ dateTime }}</span>
            </div>
          </div>
          <div class="flex items-center gap-2 text-xs">
            <span class="rounded-lg px-2.5 py-1" style="background: var(--bg-tertiary); color: var(--text-secondary);">双回路供电</span>
            <span class="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1" style="background: rgba(16,185,129,0.1); color: #10B981;">
              <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
              并网运行
            </span>
          </div>
        </div>

        <div class="mt-4">
          <LiveMetricStrip
            :items="[
              { label: '总负荷', value: store.gridLoad, unit: 'kW', color: '#F59E0B' },
              { label: '电网频率', value: store.gridFreq, decimals: 2, unit: 'Hz', color: '#10B981', pulse: true },
              { label: '母线电压', value: store.busVoltage, decimals: 1, unit: 'kV', color: '#0EA5E9' },
              { label: '功率因数', value: store.powerFactor, decimals: 2, unit: '', color: '#8B5CF6' },
              { label: '平均负载率', value: avgLoad, unit: '%', color: '#38BDF8' },
            ]"
          />
        </div>
      </section>

      <section class="grid gap-4 xl:grid-cols-[1.3fr_0.7fr]">
        <article class="rounded-2xl border p-5" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-sm font-semibold" style="color: var(--text-primary);">10kV 配电单线图</h3>
              <p class="mt-1 text-xs" style="color: var(--text-secondary);">电网 → 主变 → 母线 → 馈线 → 负荷</p>
            </div>
            <div class="flex items-center gap-3 text-[11px]" style="color: var(--text-secondary);">
              <span class="flex items-center gap-1"><span class="h-2 w-2 rounded-sm" style="background: #0EA5E9;" />正常</span>
              <span class="flex items-center gap-1"><span class="h-2 w-2 rounded-sm" style="background: #F59E0B;" />预警</span>
              <span class="flex items-center gap-1"><span class="h-2 w-2 rounded-sm" style="background: #EF4444;" />过载</span>
            </div>
          </div>

          <div class="mt-3 pd-diagram rounded-xl border p-2">
            <svg viewBox="0 0 940 360" class="w-full">
              <defs>
                <pattern id="pd-dots" width="20" height="20" patternUnits="userSpaceOnUse">
                  <circle cx="2" cy="2" r="1" fill="rgba(14,165,233,0.1)" />
                </pattern>
              </defs>

              <rect x="0" y="0" width="940" height="360" fill="url(#pd-dots)" rx="8" />

              <!-- 电网 -->
              <circle cx="52" cy="60" r="16" fill="rgba(14,165,233,0.14)" stroke="#0EA5E9" stroke-width="2" />
              <text x="52" y="65" text-anchor="middle" class="pd-node-text">电网</text>
              <text x="52" y="92" text-anchor="middle" class="pd-sub-text">110kV</text>
              <line x1="68" y1="60" x2="136" y2="60" class="pd-line" />

              <!-- 主变 -->
              <circle cx="168" cy="60" r="20" fill="rgba(139,92,246,0.12)" stroke="#8B5CF6" stroke-width="2" />
              <circle cx="168" cy="60" r="8" fill="none" stroke="#8B5CF6" stroke-width="1.5" />
              <text x="168" y="92" text-anchor="middle" class="pd-sub-text">主变 110/10kV</text>
              <line x1="188" y1="60" x2="236" y2="60" class="pd-line" />

              <!-- 母线 -->
              <line x1="236" y1="60" x2="920" y2="60" stroke="#0EA5E9" stroke-width="5" />
              <text x="578" y="46" text-anchor="middle" class="pd-node-text" fill="#0EA5E9">10kV 母线</text>

              <!-- 馈线 -->
              <template v-for="(f, i) in store.feeders" :key="f.id">
                <line
                  :x1="FEEDER_X[i]"
                  :y1="62"
                  :x2="FEEDER_X[i]"
                  :y2="292"
                  :stroke="statusColor(f.status)"
                  stroke-width="2.5"
                  class="pd-flow"
                />
                <!-- 断路器 -->
                <rect
                  :x="FEEDER_X[i] - 7"
                  y="124"
                  width="14"
                  height="14"
                  fill="rgba(15,23,42,0.85)"
                  :stroke="statusColor(f.status)"
                  stroke-width="2"
                  :transform="`rotate(45 ${FEEDER_X[i]} 131)`"
                />
                <text :x="FEEDER_X[i]" y="112" text-anchor="middle" class="pd-sub-text" :fill="statusColor(f.status)">
                  {{ f.current }}A
                </text>
                <text :x="FEEDER_X[i]" y="156" text-anchor="middle" class="pd-tag-text" :fill="statusColor(f.status)">
                  {{ SHORT_NAME[f.id] }}
                </text>

                <!-- 负荷节点 -->
                <rect
                  :x="FEEDER_X[i] - 50"
                  y="300"
                  width="100"
                  height="40"
                  rx="5"
                  :fill="`${statusColor(f.status)}14`"
                  :stroke="statusColor(f.status)"
                  stroke-width="1.5"
                />
                <text :x="FEEDER_X[i]" y="318" text-anchor="middle" class="pd-node-text">
                  {{ SHORT_NAME[f.id] }} · {{ Math.round(f.load) }}%
                </text>
                <text :x="FEEDER_X[i]" y="332" text-anchor="middle" class="pd-sub-text">
                  {{ statusText(f.status) }}
                </text>
              </template>
            </svg>
          </div>
        </article>

        <div class="grid gap-4">
          <article class="rounded-2xl border p-5" style="background: var(--bg-secondary); border-color: var(--border-color);">
            <div class="flex items-center gap-2">
              <BatteryCharging class="h-4 w-4" style="color: #F59E0B;" />
              <h3 class="text-sm font-semibold" style="color: var(--text-primary);">变电站等距视图</h3>
            </div>
            <div class="mt-2 pd-stage rounded-xl border p-3">
              <IsoScene :cols="9" :rows="5" :cell="48" :height="280" :scale="0.9">
                <IsoPylon :x="20" :y="40" :height="100" :beacon="true" label="T1" />
                <IsoPylon :x="356" :y="160" :height="100" label="T2" />

                <!-- 主变 -->
                <IsoCube :x="156" :y="32" :width="56" :depth="56" :height="60"
                  top-color="rgba(139,92,246,0.18)" front-color="rgba(139,92,246,0.34)" side-color="rgba(109,66,215,0.5)"
                  :glow="store.gridLoad > 1400" label="主变"
                  :hud="{ foot: `母线电压 ${store.busVoltage.toFixed(1)} kV · 频率 ${store.gridFreq.toFixed(2)} Hz` }" />

                <!-- 母线与断路器 -->
                <IsoCube :x="96" :y="176" :width="196" :depth="12" :height="10"
                  top-color="rgba(14,165,233,0.2)" front-color="rgba(14,165,233,0.36)" side-color="rgba(2,132,199,0.5)" label="10kV 母线"
                  :hud="{ foot: `总负荷 ${store.gridLoad} kW · 功率因数 ${store.powerFactor.toFixed(2)}` }" />
                <IsoCube :x="112" :y="220" :width="20" :depth="20" :height="34"
                  top-color="rgba(15,23,42,0.5)" front-color="rgba(15,23,42,0.8)" side-color="rgba(15,23,42,0.9)" />
                <IsoCube :x="240" :y="220" :width="20" :depth="20" :height="34"
                  top-color="rgba(15,23,42,0.5)" front-color="rgba(15,23,42,0.8)" side-color="rgba(15,23,42,0.9)" />

                <!-- 潮流 -->
                <IsoFlowDot :y="190" :left="104" :range="180" :dur="5" color="#38BDF8" />

                <!-- 站用电 -->
                <IsoCube :x="316" :y="64" :width="40" :depth="40" :height="40"
                  top-color="rgba(16,185,129,0.14)" front-color="rgba(16,185,129,0.3)" side-color="rgba(5,150,105,0.42)"
                  label="站用电" />
              </IsoScene>
            </div>
          </article>

          <article class="rounded-2xl border p-5" style="background: var(--bg-secondary); border-color: var(--border-color);">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <Activity class="h-4 w-4" style="color: #0EA5E9;" />
                <h3 class="text-sm font-semibold" style="color: var(--text-primary);">频率与负荷</h3>
              </div>
              <span class="inline-flex items-center gap-1.5 text-[11px] text-emerald-500">
                <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
                实时
              </span>
            </div>
            <div class="mt-3">
              <LiveSparkline :points="store.gridLoadSeries" color="#F59E0B" :height="64" />
            </div>
            <div class="mt-3">
              <LiveSparkline :points="store.freqSeries" color="#10B981" :height="40" />
            </div>
            <div class="mt-3 grid grid-cols-2 gap-3 text-xs">
              <div class="rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
                <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">母线电压</div>
                <div class="mt-1 font-mono-num text-base font-semibold" style="color: #0EA5E9;"><AnimatedNumber :value="store.busVoltage" decimals="2" suffix="kV" /></div>
              </div>
              <div class="rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
                <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">功率因数</div>
                <div class="mt-1 font-mono-num text-base font-semibold" style="color: #8B5CF6;"><AnimatedNumber :value="store.powerFactor" decimals="2" /></div>
              </div>
            </div>
          </article>
        </div>
      </section>

      <section class="rounded-2xl border p-5" style="background: var(--bg-secondary); border-color: var(--border-color);">
        <div class="flex items-center justify-between">
          <h3 class="text-sm font-semibold" style="color: var(--text-primary);">馈线负载</h3>
          <span class="text-xs" style="color: var(--text-secondary);">{{ warnFeeders > 0 ? `${warnFeeders} 条馈线需关注` : '全部馈线正常' }}</span>
        </div>
        <div class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <div
            v-for="f in store.feeders"
            :key="f.id"
            class="rounded-xl border p-4"
            :style="{ background: 'var(--bg-tertiary)', borderColor: statusColor(f.status) === '#0EA5E9' ? 'var(--border-color)' : `${statusColor(f.status)}66` }"
          >
            <div class="flex items-center justify-between">
              <span class="text-sm font-medium" style="color: var(--text-primary);">{{ f.name }}</span>
              <span
                class="rounded px-1.5 py-0.5 text-[10px] font-semibold"
                :style="{ color: statusColor(f.status), background: `${statusColor(f.status)}1a` }"
              >
                {{ statusText(f.status) }}
              </span>
            </div>
            <div class="mt-3 space-y-1.5 text-xs">
              <div class="flex items-center justify-between" style="color: var(--text-secondary);">
                <span>电压</span>
                <span class="font-mono-num" style="color: var(--text-primary);">{{ f.voltage }}kV</span>
              </div>
              <div class="flex items-center justify-between" style="color: var(--text-secondary);">
                <span>电流</span>
                <span class="font-mono-num" style="color: var(--text-primary);">{{ f.current }}A</span>
              </div>
            </div>
            <div class="mt-3">
              <BarMeter :value="f.load" :max="100" :color="statusColor(f.status)" :height="8" />
            </div>
            <div class="mt-1.5 text-right font-mono-num text-[11px]" :style="{ color: statusColor(f.status) }">
              {{ Math.round(f.load) }}% 负载率
            </div>
          </div>
        </div>
      </section>

      <P3EventFeed title="输配电事件流" :events="sceneEvents" />
    </div>
  </VisualScreen>
</template>

<style scoped>
.pd-hero {
  background:
    linear-gradient(90deg, rgba(245, 158, 11, 0.07), transparent 40%),
    var(--bg-secondary);
  border-color: var(--border-color);
}

.pd-pill {
  background: rgba(245, 158, 11, 0.08);
  color: #F59E0B;
}

.pd-diagram {
  background: var(--bg-tertiary);
  border-color: var(--border-color);
}

.pd-stage {
  background:
    radial-gradient(circle at 50% 40%, rgba(245, 158, 11, 0.07), transparent 60%),
    var(--bg-tertiary);
  border-color: var(--border-color);
}

.pd-line {
  stroke: #0EA5E9;
  stroke-width: 2;
}

.pd-flow {
  stroke-dasharray: 6 12;
  animation: pd-flow-dash 1.4s linear infinite;
}

.pd-node-text {
  font-size: 12px;
  font-weight: 600;
  fill: var(--text-primary);
}

.pd-sub-text {
  font-size: 9px;
  fill: var(--text-muted);
}

.pd-tag-text {
  font-size: 10px;
  font-weight: 600;
}

@keyframes pd-flow-dash {
  to {
    stroke-dashoffset: -18;
  }
}
</style>
