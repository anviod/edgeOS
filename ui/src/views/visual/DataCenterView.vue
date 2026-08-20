<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
import { Server, Gauge, Thermometer, Activity, Network, Flame } from 'lucide-vue-next'
import { useVisualStore } from '@/stores/visual'
import { useLiveClock } from '@/composables/useLiveClock'
import AnimatedNumber from '@/components/p3/AnimatedNumber.vue'
import LiveMetricStrip from '@/components/p3/LiveMetricStrip.vue'
import LiveSparkline from '@/components/p3/LiveSparkline.vue'
import RingGauge from '@/components/p3/RingGauge.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'
import VisualScreen from '@/components/visual/VisualScreen.vue'
import IsoScene from '@/components/visual/IsoScene.vue'
import IsoCube from '@/components/visual/IsoCube.vue'
import IsoRack from '@/components/visual/IsoRack.vue'
import IsoFlowDot from '@/components/visual/IsoFlowDot.vue'

const store = useVisualStore()
const { dateTime } = useLiveClock()

const ROW_DEF = [
  { y: 72, count: 6 },
  { y: 152, count: 6 },
  { y: 232, count: 5 },
]

const racks = computed(() => {
  const list: { x: number; y: number; status: 'normal' | 'hot' | 'fault'; label: string }[] = []
  ROW_DEF.forEach((row, r) => {
    for (let i = 0; i < row.count; i++) {
      const idx = r * 6 + i
      let status: 'normal' | 'hot' | 'fault' = 'normal'
      if (store.gpuLoad > 74 && idx % 4 === 1) status = 'hot'
      if (idx % 9 === 8) status = 'fault'
      list.push({ x: 24 + i * 34, y: row.y, status, label: `R${idx + 1}` })
    }
  })
  return list
})

const hotRacks = computed(() => racks.value.filter(r => r.status !== 'normal').length)

const puePct = computed(() => clampPct(((store.pue - 1) / 0.5) * 100))

function clampPct(v: number) {
  return Math.max(0, Math.min(100, v))
}

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
      <section class="rounded-2xl border p-5 dc-hero">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs dc-pill">
              <Server class="h-3.5 w-3.5" />
              数据中心仿真 · 2.5D 机房视图
            </div>
            <div class="mt-2 flex items-center gap-3">
              <h1 class="text-2xl font-semibold" style="color: var(--text-primary);">数据中心仿真</h1>
              <span class="font-mono-num text-xs" style="color: var(--text-secondary);">{{ dateTime }}</span>
            </div>
          </div>
          <div class="flex items-center gap-2 text-xs">
            <span class="rounded-lg px-2.5 py-1" style="background: var(--bg-tertiary); color: var(--text-secondary);">A 栋 · 3 排机柜</span>
            <span class="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1" style="background: rgba(16,185,129,0.1); color: #10B981;">
              <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
              运行正常
            </span>
          </div>
        </div>

        <div class="mt-4">
          <LiveMetricStrip
            :items="[
              { label: 'GPU 负载', value: store.gpuLoad, unit: '%', color: '#8B5CF6' },
              { label: 'CPU 负载', value: store.cpuLoad, unit: '%', color: '#0EA5E9' },
              { label: '网络吞吐', value: store.networkThroughput, unit: 'Gbps', color: '#10B981', pulse: true },
              { label: '制冷负载', value: store.dcCooling, unit: '%', color: '#F59E0B' },
              { label: '机房温度', value: store.dcTemp, decimals: 1, unit: '℃', color: '#38BDF8' },
            ]"
          />
        </div>
      </section>

      <section class="grid gap-4 xl:grid-cols-[1.25fr_0.75fr]">
        <article class="rounded-2xl border p-5" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-sm font-semibold" style="color: var(--text-primary);">机房等距视图</h3>
              <p class="mt-1 text-xs" style="color: var(--text-secondary);">服务器机柜 · 精密空调 · 网络数据流</p>
            </div>
            <div class="flex items-center gap-3 text-[11px]" style="color: var(--text-secondary);">
              <span class="flex items-center gap-1"><span class="h-2 w-2 rounded-sm" style="background: #22C55E;" />正常</span>
              <span class="flex items-center gap-1"><span class="h-2 w-2 rounded-sm" style="background: #F59E0B;" />高热</span>
              <span class="flex items-center gap-1"><span class="h-2 w-2 rounded-sm" style="background: #EF4444;" />故障</span>
            </div>
          </div>

          <div class="mt-2 dc-stage rounded-xl border p-3">
            <IsoScene :cols="10" :rows="8" :cell="48" :height="400" :scale="0.84">
              <!-- 地板 -->
              <IsoCube :x="8" :y="48" :width="228" :depth="280" :height="6" top-color="rgba(100,116,139,0.16)" front-color="rgba(100,116,139,0.28)" side-color="rgba(71,85,105,0.38)" />

              <!-- 机柜 -->
              <IsoRack
                v-for="r in racks"
                :key="r.label"
                :x="r.x"
                :y="r.y"
                :status="r.status"
                :label="r.label"
                :color="r.status === 'hot' ? '#F59E0B' : r.status === 'fault' ? '#EF4444' : '#0EA5E9'"
                :hud="{
                  rows: [
                    { label: '编号', value: r.label },
                    { label: 'GPU', value: `${store.gpuLoad}%`, color: '#8B5CF6' },
                    { label: 'CPU', value: `${store.cpuLoad}%`, color: '#0EA5E9' },
                    { label: '温度', value: `${store.dcTemp.toFixed(1)}℃` },
                  ],
                  foot: r.status === 'fault' ? '建议巡检排查' : r.status === 'hot' ? '关注散热' : '运行正常',
                }"
              />

              <!-- 数据流 -->
              <IsoFlowDot :y="128" :left="8" :range="220" :dur="4" color="#38BDF8" />
              <IsoFlowDot :y="208" :left="8" :range="220" :dur="5" :delay="1.2" color="#34D399" />
              <IsoFlowDot :y="288" :left="8" :range="220" :dur="4.6" :delay="2.4" color="#8B5CF6" />

              <!-- 精密空调 -->
              <IsoCube :x="256" :y="72" :width="44" :depth="44" :height="62"
                top-color="rgba(56,189,248,0.16)" front-color="rgba(56,189,248,0.32)" side-color="rgba(2,132,199,0.46)"
                :glow="store.dcTemp > 27" label="空调-1"
                :hud="{ foot: `送风温度 ${store.dcTemp.toFixed(1)}℃ · 制冷负载 ${store.dcCooling}%` }" />
              <IsoCube :x="256" :y="232" :width="44" :depth="44" :height="62"
                top-color="rgba(56,189,248,0.16)" front-color="rgba(56,189,248,0.32)" side-color="rgba(2,132,199,0.46)"
                :glow="store.dcTemp > 27" label="空调-2"
                :hud="{ foot: `送风温度 ${store.dcTemp.toFixed(1)}℃ · 制冷负载 ${store.dcCooling}%` }" />

              <!-- 电源柜 PDU -->
              <IsoCube :x="344" :y="120" :width="40" :depth="40" :height="58"
                top-color="rgba(245,158,11,0.16)" front-color="rgba(245,158,11,0.32)" side-color="rgba(180,120,10,0.46)"
                :beacon="store.dcPower > 920" beacon-color="#F59E0B" label="PDU"
                :hud="{ foot: `机房功耗 ${store.dcPower.toFixed(1)} kW` }" />

              <!-- 网络汇聚柜 -->
              <IsoCube :x="392" :y="120" :width="40" :depth="40" :height="58"
                top-color="rgba(139,92,246,0.16)" front-color="rgba(139,92,246,0.32)" side-color="rgba(109,66,215,0.46)"
                :glow="store.networkThroughput > 220" label="核心交换"
                :hud="{ foot: `网络吞吐 ${store.networkThroughput} Gbps` }" />

              <!-- 发电机 -->
              <IsoCube :x="344" :y="240" :width="88" :depth="40" :height="34"
                top-color="rgba(16,185,129,0.14)" front-color="rgba(16,185,129,0.3)" side-color="rgba(5,150,105,0.42)"
                label="柴发"
                :hud="{ foot: '备用应急电源 · 未启动' }" />
            </IsoScene>
          </div>

          <div class="mt-4 grid grid-cols-4 gap-3 text-center">
            <div class="rounded-lg border px-3 py-3" style="background: var(--bg-tertiary); border-color: var(--border-color);">
              <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">机柜</div>
              <div class="mt-1 font-mono-num text-xl font-bold" style="color: #0EA5E9;"><AnimatedNumber :value="store.rackCount" /></div>
            </div>
            <div class="rounded-lg border px-3 py-3" style="background: var(--bg-tertiary); border-color: var(--border-color);">
              <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">GPU %</div>
              <div class="mt-1 font-mono-num text-xl font-bold" style="color: #8B5CF6;"><AnimatedNumber :value="store.gpuLoad" /></div>
            </div>
            <div class="rounded-lg border px-3 py-3" style="background: var(--bg-tertiary); border-color: var(--border-color);">
              <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">机房功耗 kW</div>
              <div class="mt-1 font-mono-num text-xl font-bold" style="color: #F59E0B;"><AnimatedNumber :value="store.dcPower" decimals="1" /></div>
            </div>
            <div class="rounded-lg border px-3 py-3" style="background: var(--bg-tertiary); border-color: var(--border-color);">
              <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">异常机柜</div>
              <div class="mt-1 font-mono-num text-xl font-bold" :style="{ color: hotRacks > 0 ? '#F59E0B' : '#10B981' }">{{ hotRacks }}</div>
            </div>
          </div>
        </article>

        <div class="grid gap-4">
          <article class="rounded-2xl border p-5" style="background: var(--bg-secondary); border-color: var(--border-color);">
            <div class="flex items-center gap-2">
              <Gauge class="h-4 w-4" style="color: #10B981;" />
              <h3 class="text-sm font-semibold" style="color: var(--text-primary);">能效 PUE</h3>
            </div>
            <div class="mt-3 flex items-center justify-around">
              <RingGauge :value="puePct" :size="150" color="#10B981" label="PUE 目标 ≤ 1.3">
                <span class="font-mono-num text-2xl font-bold" style="color: #10B981;">
                  <AnimatedNumber :value="store.pue" decimals="2" />
                </span>
              </RingGauge>
              <div class="space-y-3 text-xs">
                <div class="flex items-center gap-2 rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
                  <Network class="h-4 w-4" style="color: #0EA5E9;" />
                  <span style="color: var(--text-secondary);">网络吞吐</span>
                  <span class="ml-auto font-mono-num font-semibold" style="color: var(--text-primary);"><AnimatedNumber :value="store.networkThroughput" suffix="G" /></span>
                </div>
                <div class="flex items-center gap-2 rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
                  <Thermometer class="h-4 w-4" style="color: #38BDF8;" />
                  <span style="color: var(--text-secondary);">机房温度</span>
                  <span class="ml-auto font-mono-num font-semibold" style="color: var(--text-primary);"><AnimatedNumber :value="store.dcTemp" decimals="1" suffix="℃" /></span>
                </div>
                <div class="flex items-center gap-2 rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
                  <Flame class="h-4 w-4" style="color: #F59E0B;" />
                  <span style="color: var(--text-secondary);">制冷负载</span>
                  <span class="ml-auto font-mono-num font-semibold" style="color: var(--text-primary);"><AnimatedNumber :value="store.dcCooling" suffix="%" /></span>
                </div>
              </div>
            </div>
          </article>

          <article class="rounded-2xl border p-5" style="background: var(--bg-secondary); border-color: var(--border-color);">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <Activity class="h-4 w-4" style="color: #0EA5E9;" />
                <h3 class="text-sm font-semibold" style="color: var(--text-primary);">网络吞吐趋势</h3>
              </div>
              <span class="inline-flex items-center gap-1.5 text-[11px] text-emerald-500">
                <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
                实时
              </span>
            </div>
            <div class="mt-3">
              <LiveSparkline :points="store.netSeries" color="#0EA5E9" :height="70" />
            </div>
            <div class="mt-4 grid grid-cols-2 gap-3">
              <div class="rounded-lg border px-3 py-2.5" style="background: var(--bg-tertiary); border-color: var(--border-color);">
                <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">CPU 负载</div>
                <div class="mt-1 flex items-center gap-2">
                  <span class="font-mono-num text-base font-semibold" style="color: #0EA5E9;"><AnimatedNumber :value="store.cpuLoad" suffix="%" /></span>
                </div>
              </div>
              <div class="rounded-lg border px-3 py-2.5" style="background: var(--bg-tertiary); border-color: var(--border-color);">
                <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">功耗趋势</div>
                <div class="mt-2">
                  <LiveSparkline :points="store.dcPowerSeries" color="#F59E0B" :height="28" :fill="false" />
                </div>
              </div>
            </div>
          </article>
        </div>
      </section>

      <P3EventFeed title="数据中心事件流" :events="sceneEvents" />
    </div>
  </VisualScreen>
</template>

<style scoped>
.dc-hero {
  background:
    linear-gradient(90deg, rgba(139, 92, 246, 0.07), transparent 40%),
    var(--bg-secondary);
  border-color: var(--border-color);
}

.dc-pill {
  background: rgba(139, 92, 246, 0.08);
  color: #8B5CF6;
}

.dc-stage {
  background:
    radial-gradient(circle at 50% 40%, rgba(139, 92, 246, 0.07), transparent 60%),
    var(--bg-tertiary);
  border-color: var(--border-color);
}
</style>
