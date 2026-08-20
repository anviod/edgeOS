<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
import { BatteryCharging, Thermometer, Gauge, Layers, Activity } from 'lucide-vue-next'
import { useVisualStore } from '@/stores/visual'
import { useLiveClock } from '@/composables/useLiveClock'
import AnimatedNumber from '@/components/p3/AnimatedNumber.vue'
import LiveSparkline from '@/components/p3/LiveSparkline.vue'
import RingGauge from '@/components/p3/RingGauge.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'
import IsoScene from '@/components/visual/IsoScene.vue'
import IsoCube from '@/components/visual/IsoCube.vue'
import IsoStorageUnit from '@/components/visual/IsoStorageUnit.vue'
import IsoSolarPanel from '@/components/visual/IsoSolarPanel.vue'
import VisualScreen from '@/components/visual/VisualScreen.vue'

const store = useVisualStore()
const { dateTime } = useLiveClock()

const modes = [
  { key: 'charge', label: '充电', color: '#F59E0B' },
  { key: 'discharge', label: '放电', color: '#10B981' },
  { key: 'idle', label: '待机', color: '#64748B' },
] as const

const modeColor = computed(() => {
  if (store.mode === 'charge') return '#F59E0B'
  if (store.mode === 'discharge') return '#10B981'
  return '#94A3B8'
})

const modeText = computed(() => {
  if (store.mode === 'charge') return '充电'
  if (store.mode === 'discharge') return '放电'
  return '待机'
})

const statusText = computed(() => {
  if (store.mode === 'charge') return 'CHARGING'
  if (store.mode === 'discharge') return 'DISCHARGING'
  return 'IDLE'
})

const powerAbs = computed(() => Math.abs(store.power))

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
      <section class="rounded-2xl border p-5 storage-hero">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs storage-pill">
            <BatteryCharging class="h-3.5 w-3.5" />
            储能电站 · 2.5D 运行视图
          </div>
          <div class="mt-2 flex items-center gap-3">
            <h1 class="text-2xl font-semibold" style="color: var(--text-primary);">储能电站</h1>
            <span class="font-mono-num text-xs" style="color: var(--text-secondary);">{{ dateTime }}</span>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <button
            v-for="mode in modes"
            :key="mode.key"
            class="rounded-lg border px-4 py-1.5 text-sm font-medium transition-colors"
            :style="store.mode === mode.key
              ? { background: `${mode.color}1a`, borderColor: mode.color, color: mode.color }
              : { background: 'var(--bg-tertiary)', borderColor: 'var(--border-color)', color: 'var(--text-secondary)' }"
            @click="store.setMode(mode.key)"
          >
            {{ mode.label }}
          </button>
        </div>
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
      <article class="rounded-2xl border p-5" style="background: var(--bg-secondary); border-color: var(--border-color);">
        <div class="flex items-center justify-between">
          <div>
            <h3 class="text-sm font-semibold" style="color: var(--text-primary);">储能站等距场景</h3>
            <p class="mt-1 text-xs" style="color: var(--text-secondary);">
              储能柜 · PCS 变流 · 变压器 · 冷却塔 · 主控楼
            </p>
          </div>
          <span class="inline-flex items-center gap-1.5 text-[11px]" :style="{ color: modeColor }">
            <span class="h-1.5 w-1.5 animate-pulse rounded-full" :style="{ background: modeColor }" />
            {{ statusText }}
          </span>
        </div>

        <div class="mt-2 visual-stage rounded-xl border p-3">
          <IsoScene :cols="10" :rows="8" :cell="48" :height="400" :scale="0.84">
            <!-- 基础平台 -->
            <IsoCube :x="8" :y="48" :width="208" :depth="276" :height="8" top-color="rgba(100,116,139,0.18)" front-color="rgba(100,116,139,0.3)" side-color="rgba(71,85,105,0.4)" />

            <!-- 储能柜 -->
            <IsoStorageUnit :x="20" :y="64" :status="store.mode" label="储能柜-01" :hud="{ foot: `SOC ${store.soc.toFixed(1)}% · 柜温 ${store.batteryTemp.toFixed(1)}℃` }" />
            <IsoStorageUnit :x="20" :y="160" :status="store.mode" color="#38BDF8" label="储能柜-02" :hud="{ foot: `SOC ${store.soc.toFixed(1)}% · 柜温 ${store.batteryTemp.toFixed(1)}℃` }" />
            <IsoStorageUnit :x="20" :y="256" :status="store.mode" color="#8B5CF6" label="储能柜-03" :hud="{ foot: `SOC ${store.soc.toFixed(1)}% · 柜温 ${store.batteryTemp.toFixed(1)}℃` }" />

            <!-- PCS 变流器 -->
            <IsoCube :x="264" :y="72" :width="40" :depth="40" :height="60"
              :top-color="`${modeColor}22`" :front-color="`${modeColor}55`" :side-color="`${modeColor}77`"
              :glow="store.mode !== 'idle'" label="PCS-1"
              :hud="{ badge: modeText, foot: `输出功率 ${powerAbs.toFixed(1)} MW` }" />
            <IsoCube :x="264" :y="168" :width="40" :depth="40" :height="60"
              :top-color="`${modeColor}22`" :front-color="`${modeColor}55`" :side-color="`${modeColor}77`"
              :glow="store.mode !== 'idle'" label="PCS-2"
              :hud="{ badge: modeText, foot: `输出功率 ${powerAbs.toFixed(1)} MW` }" />

            <!-- 变压器 -->
            <IsoCube :x="360" :y="120" :width="56" :depth="56" :height="68"
              top-color="rgba(100,116,139,0.3)" front-color="rgba(100,116,139,0.45)" side-color="rgba(71,85,105,0.6)"
              label="主变" />

            <!-- 冷却塔 -->
            <IsoCube :x="368" :y="56" :width="48" :depth="48" :height="88"
              top-color="rgba(16,185,129,0.18)" front-color="rgba(16,185,129,0.34)" side-color="rgba(5,150,105,0.48)"
              label="冷却塔" />

                <!-- 主控楼 -->
                <IsoCube :x="344" :y="264" :width="96" :depth="64" :height="40"
                  top-color="rgba(14,165,233,0.16)" front-color="rgba(14,165,233,0.32)" side-color="rgba(7,108,168,0.46)" label="主控楼" />
                <IsoCube :x="352" :y="276" :width="80" :depth="40" :height="26"
                  top-color="rgba(30,41,59,0.55)" front-color="rgba(30,41,59,0.75)" side-color="rgba(30,41,59,0.85)" />

                <!-- 光伏板阵列 -->
                <IsoSolarPanel :x="252" :y="36" :width="104" :depth="44" :height="6" color="#0EA5E9" />
                <IsoSolarPanel :x="380" :y="36" :width="104" :depth="44" :height="6" color="#38BDF8" />

                <!-- 汇流电缆 -->
                <IsoCube :x="196" :y="128" :width="56" :depth="8" :height="4"
                  top-color="rgba(15,23,42,0.5)" front-color="rgba(15,23,42,0.7)" side-color="rgba(15,23,42,0.85)" />
                <IsoCube :x="196" :y="224" :width="56" :depth="8" :height="4"
                  top-color="rgba(15,23,42,0.5)" front-color="rgba(15,23,42,0.7)" side-color="rgba(15,23,42,0.85)" />
          </IsoScene>
        </div>

        <div class="mt-4 grid grid-cols-3 gap-3 text-center">
          <div class="rounded-lg border px-3 py-3" style="background: var(--bg-tertiary); border-color: var(--border-color);">
            <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">功率 MW</div>
            <div class="mt-1 font-mono-num text-2xl font-bold" :style="{ color: modeColor }">
              <AnimatedNumber :value="powerAbs" decimals="1" />
            </div>
            <div class="text-[10px]" :style="{ color: modeColor }">{{ modeText }}</div>
          </div>
          <div class="rounded-lg border px-3 py-3" style="background: var(--bg-tertiary); border-color: var(--border-color);">
            <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">SOC %</div>
            <div class="mt-1 font-mono-num text-2xl font-bold" style="color: #10B981;">
              <AnimatedNumber :value="store.soc" decimals="1" />
            </div>
            <div class="text-[10px]" style="color: var(--text-muted);">荷电状态</div>
          </div>
          <div class="rounded-lg border px-3 py-3" style="background: var(--bg-tertiary); border-color: var(--border-color);">
            <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">SOH %</div>
            <div class="mt-1 font-mono-num text-2xl font-bold" style="color: #0EA5E9;">
              <AnimatedNumber :value="store.soh" decimals="1" />
            </div>
            <div class="text-[10px]" style="color: var(--text-muted);">健康度</div>
          </div>
        </div>
      </article>

      <div class="grid gap-4">
        <article class="rounded-2xl border p-5" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <Gauge class="h-4 w-4" style="color: #10B981;" />
              <h3 class="text-sm font-semibold" style="color: var(--text-primary);">电池能量状态</h3>
            </div>
          </div>
          <div class="mt-3 flex items-center justify-around">
            <RingGauge :value="store.soc" :size="150" color="#10B981" label="SOC">
              <span class="font-mono-num text-2xl font-bold text-emerald-500">
                <AnimatedNumber :value="store.soc" decimals="1" suffix="%" />
              </span>
            </RingGauge>
            <div class="space-y-3 text-xs">
              <div class="flex items-center gap-2 rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
                <Thermometer class="h-4 w-4" style="color: #F59E0B;" />
                <span style="color: var(--text-secondary);">电池温度</span>
                <span class="ml-auto font-mono-num font-semibold" style="color: var(--text-primary);"><AnimatedNumber :value="store.batteryTemp" decimals="1" suffix="℃" /></span>
              </div>
              <div class="flex items-center gap-2 rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
                <Layers class="h-4 w-4" style="color: #8B5CF6;" />
                <span style="color: var(--text-secondary);">可用容量</span>
                <span class="ml-auto font-mono-num font-semibold" style="color: var(--text-primary);"><AnimatedNumber :value="store.capacity" decimals="1" suffix="MWh" /></span>
              </div>
              <div class="flex items-center gap-2 rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
                <BatteryCharging class="h-4 w-4" style="color: #0EA5E9;" />
                <span style="color: var(--text-secondary);">循环次数</span>
                <span class="ml-auto font-mono-num font-semibold" style="color: var(--text-primary);"><AnimatedNumber :value="store.cycles" decimals="0" suffix=" 次" /></span>
              </div>
            </div>
          </div>
        </article>

        <article class="rounded-2xl border p-5" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <Activity class="h-4 w-4" style="color: #0EA5E9;" />
              <h3 class="text-sm font-semibold" style="color: var(--text-primary);">充放电功率趋势</h3>
            </div>
            <span class="inline-flex items-center gap-1.5 text-[11px]" :style="{ color: modeColor }">
              <span class="h-1.5 w-1.5 animate-pulse rounded-full" :style="{ background: modeColor }" />
              实时
            </span>
          </div>
          <div class="mt-3">
            <LiveSparkline :points="store.powerSeries" :color="modeColor" :height="84" />
          </div>
          <div class="mt-4 grid grid-cols-2 gap-3">
            <div class="rounded-lg border px-3 py-2.5" style="background: var(--bg-tertiary); border-color: var(--border-color);">
              <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">放电功率 MW</div>
              <div class="mt-1 font-mono-num text-base font-semibold" style="color: #10B981;">
                <AnimatedNumber :value="Math.max(0, store.power)" decimals="1" />
              </div>
            </div>
            <div class="rounded-lg border px-3 py-2.5" style="background: var(--bg-tertiary); border-color: var(--border-color);">
              <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">充电功率 MW</div>
              <div class="mt-1 font-mono-num text-base font-semibold" style="color: #F59E0B;">
                <AnimatedNumber :value="Math.max(0, -store.power)" decimals="1" />
              </div>
            </div>
          </div>
        </article>
      </div>
    </section>

    <P3EventFeed title="储能电站事件流" :events="sceneEvents" />
    </div>
  </VisualScreen>
</template>

<style scoped>
.storage-hero {
  background:
    linear-gradient(90deg, rgba(16, 185, 129, 0.07), transparent 40%),
    var(--bg-secondary);
  border-color: var(--border-color);
}

.storage-pill {
  background: rgba(16, 185, 129, 0.08);
  color: #10B981;
}

.visual-stage {
  background:
    radial-gradient(circle at 50% 40%, rgba(16, 185, 129, 0.07), transparent 60%),
    var(--bg-tertiary);
  border-color: var(--border-color);
}
</style>
