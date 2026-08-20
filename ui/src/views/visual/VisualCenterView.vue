<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
import { BatteryCharging, Factory, MonitorPlay, Gauge, Server, Zap, Anchor, Clock, Boxes } from 'lucide-vue-next'
import { useVisualStore } from '@/stores/visual'
import { useLiveClock } from '@/composables/useLiveClock'
import AnimatedNumber from '@/components/p3/AnimatedNumber.vue'
import LiveMetricStrip from '@/components/p3/LiveMetricStrip.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'
import LiveSparkline from '@/components/p3/LiveSparkline.vue'
import VisualScreen from '@/components/visual/VisualScreen.vue'
import IsoScene from '@/components/visual/IsoScene.vue'
import IsoCube from '@/components/visual/IsoCube.vue'
import IsoStorageUnit from '@/components/visual/IsoStorageUnit.vue'
import IsoRack from '@/components/visual/IsoRack.vue'
import IsoPylon from '@/components/visual/IsoPylon.vue'

const store = useVisualStore()
const { dateTime } = useLiveClock()

const modules = [
  { label: '储能电站', path: '/visual/storage-station', icon: BatteryCharging, note: '充放电 / SOC / 调峰' },
  { label: '产线展示', path: '/visual/production-line', icon: Factory, note: 'OEE / 节拍 / AGV' },
  { label: '工业大屏', path: '/visual/industrial-screen', icon: MonitorPlay, note: '能耗 / 质量 / 告警' },
  { label: '仪表监控', path: '/visual/instruments', icon: Gauge, note: '压力 / 温度 / 流量' },
  { label: '数据中心仿真', path: '/visual/data-center', icon: Server, note: '机柜 / PUE / 网络' },
  { label: '输配电', path: '/visual/power-distribution', icon: Zap, note: '母线 / 馈线 / 潮流' },
  { label: '港口运输', path: '/visual/port', icon: Anchor, note: '货轮 / 岸桥 / 集卡' },
]

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
      <section class="visual-hero rounded-2xl border p-5">
        <div class="grid gap-5 xl:grid-cols-[1.2fr_0.8fr]">
          <div>
            <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs visual-pill">
              <Boxes class="h-3.5 w-3.5" />
              数据可视化驾驶舱
            </div>
            <div class="mt-3 flex items-center gap-3">
              <h1 class="text-3xl font-semibold visual-title">可视化中心总览</h1>
              <span
                class="inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1 font-mono-num text-xs"
                style="color: var(--text-secondary); border-color: var(--border-color); background: var(--bg-tertiary);"
              >
                <Clock class="h-3.5 w-3.5" style="color: #0EA5E9;" />
                {{ dateTime }}
              </span>
            </div>
            <p class="mt-2 max-w-3xl text-sm visual-subtitle">
              以 2.5D 等距视角呈现储能电站、产线运行、数据中心、输配电、港口运输、工业大屏与仪表监控七大可视化场景，纯前端实时仿真驱动。
            </p>

            <div class="mt-5">
              <LiveMetricStrip
                :items="[
                  { label: '储能 SOC', value: store.soc, decimals: 1, unit: '%', color: '#10B981', pulse: true },
                  { label: '产线节拍', value: store.lineSpeed, unit: '%', color: '#0EA5E9' },
                  { label: '数据中心 PUE', value: store.pue, decimals: 2, unit: '', color: '#8B5CF6' },
                  { label: '总负荷', value: store.powerLoad, decimals: 1, unit: 'kW', color: '#F59E0B' },
                  { label: '告警', value: store.alarmCount, unit: '条', color: '#EF4444', pulse: true },
                ]"
              />
            </div>
          </div>

          <div class="visual-stage rounded-xl border p-3">
            <IsoScene :cols="7" :rows="5" :cell="52" :height="280" :scale="0.9">
              <IsoCube :col="0" :row="1" :width="120" :depth="120" :height="26" top-color="rgba(14,165,233,0.18)" front-color="rgba(14,165,233,0.3)" side-color="rgba(7,108,168,0.4)" />
              <IsoStorageUnit :col="3" :row="1" :cell="52" :width="130" :depth="44" :height="24" :control-h="10" :status="store.mode" label="储能柜" />
              <IsoRack :col="6" :row="1" :cell="52" :status="'normal'" label="机柜" />
              <IsoPylon :col="4" :row="3" :cell="52" :height="90" :beacon="true" />
              <IsoCube :col="1" :row="3" :width="52" :depth="52" :height="78" top-color="rgba(16,185,129,0.2)" front-color="rgba(16,185,129,0.35)" side-color="rgba(5,150,105,0.5)" label="冷却塔" />
            </IsoScene>
          </div>
        </div>
      </section>

    <section class="grid gap-4 xl:grid-cols-[1fr_0.85fr]">
      <article class="rounded-2xl border p-5" style="background: var(--bg-secondary); border-color: var(--border-color);">
        <div class="flex items-center justify-between">
          <div>
            <h3 class="text-sm font-semibold" style="color: var(--text-primary);">可视化模块入口</h3>
            <p class="mt-1 text-xs" style="color: var(--text-secondary);">点击进入对应 2.5D 场景</p>
          </div>
        </div>
        <div class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          <router-link
            v-for="module in modules"
            :key="module.path"
            :to="module.path"
            class="visual-module rounded-xl border p-4 transition-all duration-300 hover:-translate-y-0.5"
          >
            <div class="flex items-center justify-between gap-3">
              <div>
                <div class="text-sm font-medium visual-title">{{ module.label }}</div>
                <div class="mt-1 text-xs visual-subtitle">{{ module.note }}</div>
              </div>
              <component :is="module.icon" class="h-5 w-5 visual-icon" />
            </div>
          </router-link>
        </div>
      </article>

      <article class="rounded-2xl border p-5" style="background: var(--bg-secondary); border-color: var(--border-color);">
        <div class="flex items-center justify-between">
          <h3 class="text-sm font-semibold" style="color: var(--text-primary);">产线综合质量</h3>
          <span class="inline-flex items-center gap-1.5 text-[11px] text-emerald-500">
            <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
            实时
          </span>
        </div>
        <div class="mt-3 grid grid-cols-2 gap-3 text-xs">
          <div class="rounded-lg border px-3 py-2.5" style="background: var(--bg-tertiary); border-color: var(--border-color);">
            <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">一次良率</div>
            <div class="mt-1 flex items-center gap-1.5">
              <span class="text-lg font-semibold font-mono-num" style="color: #10B981;">
                <AnimatedNumber :value="store.okRate" decimals="1" suffix="%" />
              </span>
            </div>
          </div>
          <div class="rounded-lg border px-3 py-2.5" style="background: var(--bg-tertiary); border-color: var(--border-color);">
            <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">在制品 WIP</div>
            <div class="mt-1 flex items-center gap-1.5">
              <span class="text-lg font-semibold font-mono-num" style="color: #0EA5E9;">
                <AnimatedNumber :value="store.wip" />
              </span>
            </div>
          </div>
        </div>
        <div class="mt-3">
          <LiveSparkline :points="store.lineSeries" color="#0EA5E9" :height="72" />
        </div>
      </article>
    </section>

    <P3EventFeed title="可视化实时事件流" :events="sceneEvents" />
    </div>
  </VisualScreen>
</template>

<style scoped>
.visual-hero {
  background:
    linear-gradient(90deg, rgba(14, 165, 233, 0.08), transparent 40%),
    var(--bg-secondary);
  border-color: var(--border-color);
}

.visual-pill {
  background: rgba(14, 165, 233, 0.08);
  color: #0EA5E9;
}

.visual-title {
  color: var(--text-primary);
}

.visual-subtitle {
  color: var(--text-secondary);
}

.visual-stage {
  background:
    radial-gradient(circle at 50% 40%, rgba(14, 165, 233, 0.08), transparent 60%),
    var(--bg-tertiary);
  border-color: var(--border-color);
}

.visual-module {
  background: var(--bg-secondary);
  border-color: var(--border-color);
}

.visual-module:hover {
  border-color: rgba(14, 165, 233, 0.3);
  background: var(--bg-tertiary);
}

.visual-icon {
  color: #0EA5E9;
}
</style>
