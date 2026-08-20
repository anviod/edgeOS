<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
import { Factory, Package, Activity } from 'lucide-vue-next'
import { useVisualStore, type MachineStatus } from '@/stores/visual'
import { useLiveClock } from '@/composables/useLiveClock'
import AnimatedNumber from '@/components/p3/AnimatedNumber.vue'
import LiveMetricStrip from '@/components/p3/LiveMetricStrip.vue'
import LiveSparkline from '@/components/p3/LiveSparkline.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'
import IsoScene from '@/components/visual/IsoScene.vue'
import IsoCube from '@/components/visual/IsoCube.vue'
import IsoConveyor from '@/components/visual/IsoConveyor.vue'
import IsoFlowDot from '@/components/visual/IsoFlowDot.vue'
import VisualScreen from '@/components/visual/VisualScreen.vue'

const store = useVisualStore()
const { dateTime } = useLiveClock()

const STATUS_STYLE: Record<MachineStatus, { top: string; front: string; side: string; glow: boolean; color: string }> = {
  running: { top: 'rgba(16,185,129,0.2)', front: 'rgba(16,185,129,0.4)', side: 'rgba(5,150,105,0.55)', glow: false, color: '#10B981' },
  standby: { top: 'rgba(148,163,184,0.2)', front: 'rgba(148,163,184,0.38)', side: 'rgba(100,116,139,0.52)', glow: false, color: '#94A3B8' },
  warn: { top: 'rgba(245,158,11,0.22)', front: 'rgba(245,158,11,0.42)', side: 'rgba(180,120,10,0.55)', glow: true, color: '#F59E0B' },
  fault: { top: 'rgba(239,68,68,0.24)', front: 'rgba(239,68,68,0.44)', side: 'rgba(185,28,28,0.55)', glow: true, color: '#EF4444' },
}

const machineStyle = computed(() => (status: MachineStatus) => STATUS_STYLE[status])

function statusLabel(status: MachineStatus) {
  if (status === 'running') return '运行中'
  if (status === 'standby') return '待机'
  if (status === 'warn') return '关注'
  return '故障'
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
      <section class="rounded-2xl border p-5 pl-hero">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs pl-pill">
            <Factory class="h-3.5 w-3.5" />
            产线展示 · 2.5D 车间视图
          </div>
          <div class="mt-2 flex items-center gap-3">
            <h1 class="text-2xl font-semibold" style="color: var(--text-primary);">产线展示</h1>
            <span class="font-mono-num text-xs" style="color: var(--text-secondary);">{{ dateTime }}</span>
          </div>
        </div>
        <div class="flex items-center gap-2 text-xs">
          <span class="rounded-lg px-2.5 py-1" style="background: var(--bg-tertiary); color: var(--text-secondary);">车间 A · 总装</span>
          <span class="rounded-lg px-2.5 py-1" style="background: var(--bg-tertiary); color: var(--text-secondary);">2 条产线 · 5 台主机</span>
          <span class="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1" style="background: rgba(16,185,129,0.1); color: #10B981;">
            <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
            生产运行
          </span>
        </div>
      </div>

      <div class="mt-4">
        <LiveMetricStrip
          :items="[
            { label: '产线节拍', value: store.lineSpeed, unit: '%', color: '#0EA5E9' },
            { label: '一次良率', value: store.okRate, decimals: 1, unit: '%', color: '#10B981', pulse: true },
            { label: '平均 OEE', value: store.avgOee, decimals: 1, unit: '%', color: '#8B5CF6' },
            { label: '在制品 WIP', value: store.wip, unit: '件', color: '#F59E0B' },
            { label: '累计产出', value: store.produced, unit: '件', color: '#10B981' },
          ]"
        />
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[1.25fr_0.75fr]">
      <article class="rounded-2xl border p-5" style="background: var(--bg-secondary); border-color: var(--border-color);">
        <div class="flex items-center justify-between">
          <div>
            <h3 class="text-sm font-semibold" style="color: var(--text-primary);">车间等距视图</h3>
            <p class="mt-1 text-xs" style="color: var(--text-secondary);">主机 / 输送带 / AGV 转运实时状态</p>
          </div>
          <div class="flex items-center gap-3 text-[11px]" style="color: var(--text-secondary);">
            <span class="flex items-center gap-1"><span class="h-2 w-2 rounded-sm" style="background: #10B981;" />运行</span>
            <span class="flex items-center gap-1"><span class="h-2 w-2 rounded-sm" style="background: #F59E0B;" />关注</span>
            <span class="flex items-center gap-1"><span class="h-2 w-2 rounded-sm" style="background: #EF4444;" />故障</span>
          </div>
        </div>

        <div class="mt-2 pl-stage rounded-xl border p-3">
          <IsoScene :cols="10" :rows="7" :cell="48" :height="390" :scale="0.9">
            <!-- 车间墙体 / 货架 -->
            <IsoCube :x="4" :y="24" :width="440" :depth="10" :height="14" top-color="rgba(100,116,139,0.16)" front-color="rgba(100,116,139,0.26)" side-color="rgba(71,85,105,0.36)" />
            <IsoCube :x="264" :y="48" :width="180" :depth="26" :height="86" top-color="rgba(139,92,246,0.14)" front-color="rgba(139,92,246,0.28)" side-color="rgba(109,66,215,0.4)" label="立体库" />

            <!-- 一号线主机 -->
            <IsoCube
              v-for="m in store.machines.filter(x => x.id.startsWith('M') && x.id !== 'M4' && x.id !== 'M5')"
              :key="m.id"
              :x="m.col * 48 + 8"
              :y="m.row * 48"
              :width="44"
              :depth="44"
              :height="54"
              :top-color="machineStyle(m.status).top"
              :front-color="machineStyle(m.status).front"
              :side-color="machineStyle(m.status).side"
              :glow="machineStyle(m.status).glow"
              :beacon="m.status === 'warn' || m.status === 'fault'"
              :beacon-color="machineStyle(m.status).color"
              :label="m.id"
              :hud="{
                badge: statusLabel(m.status),
                accent: machineStyle(m.status).color,
                rows: [
                  { label: 'OEE', value: `${m.oee.toFixed(1)}%` },
                  { label: '节拍', value: `${m.rate} 件/分` },
                  { label: '温度', value: `${m.temp.toFixed(1)}℃` },
                ],
                foot: m.status === 'warn' || m.status === 'fault' ? '设备状态需关注' : '设备运行正常',
              }"
            />

            <!-- 一号线输送带 -->
            <IsoConveyor :x="0" :y="168" :width="336" :depth="18" :height="9" color="#0EA5E9" :speed="6" :item-count="5" />

            <!-- 二号线主机 -->
            <IsoCube
              v-for="m in store.machines.filter(x => x.id === 'M4' || x.id === 'M5')"
              :key="m.id"
              :x="m.col * 48 + 8"
              :y="m.row * 48"
              :width="44"
              :depth="44"
              :height="54"
              :top-color="machineStyle(m.status).top"
              :front-color="machineStyle(m.status).front"
              :side-color="machineStyle(m.status).side"
              :glow="machineStyle(m.status).glow"
              :beacon="m.status === 'warn' || m.status === 'fault'"
              :beacon-color="machineStyle(m.status).color"
              :label="m.id"
              :hud="{
                badge: statusLabel(m.status),
                accent: machineStyle(m.status).color,
                rows: [
                  { label: 'OEE', value: `${m.oee.toFixed(1)}%` },
                  { label: '节拍', value: `${m.rate} 件/分` },
                  { label: '温度', value: `${m.temp.toFixed(1)}℃` },
                ],
                foot: m.status === 'warn' || m.status === 'fault' ? '设备状态需关注' : '设备运行正常',
              }"
            />

            <!-- 二号线输送带 -->
            <IsoConveyor :x="264" :y="216" :width="120" :depth="18" :height="9" color="#8B5CF6" :speed="5" :item-count="3" />

            <!-- AGV -->
            <IsoCube
              v-for="agv in store.agvs"
              :key="agv.id"
              :x="agv.x - 10"
              :y="agv.y - 10"
              :width="20"
              :depth="20"
              :height="16"
              :top-color="`${agv.color}44`"
              :front-color="`${agv.color}88`"
              :side-color="`${agv.color}aa`"
              glow
              :label="`${agv.label}${agv.load ? ' ●' : ''}`"
              :hud="{
                badge: agv.load ? '载货' : '空载',
                accent: agv.color,
                rows: [
                  { label: '当前位置', value: `(${agv.x.toFixed(0)}, ${agv.y.toFixed(0)})` },
                  { label: '目标', value: `(${agv.targetX.toFixed(0)}, ${agv.targetY.toFixed(0)})` },
                ],
                foot: 'AGV 自动转运中',
              }"
            />

            <!-- 质检工位 -->
            <IsoCube :x="408" :y="240" :width="44" :depth="44" :height="38" top-color="rgba(56,189,248,0.16)" front-color="rgba(56,189,248,0.32)" side-color="rgba(2,132,199,0.44)" label="质检台" />

            <!-- AGV 路径标记 -->
            <IsoFlowDot :y="150" :left="10" :range="440" :dur="7" color="#38BDF8" />
            <IsoFlowDot :y="240" :left="10" :range="440" :dur="8.5" :delay="2" color="#34D399" />
          </IsoScene>
        </div>
      </article>

      <div class="grid gap-4">
        <article class="rounded-2xl border p-5" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <div class="flex items-center gap-2">
            <Activity class="h-4 w-4" style="color: #0EA5E9;" />
            <h3 class="text-sm font-semibold" style="color: var(--text-primary);">主机状态</h3>
          </div>
          <div class="mt-3 space-y-2">
            <div
              v-for="m in store.machines"
              :key="m.id"
              class="flex items-center gap-3 rounded-lg border px-3 py-2.5"
              style="background: var(--bg-tertiary); border-color: var(--border-color);"
            >
              <span class="h-2 w-2 rounded-sm" :style="{ background: machineStyle(m.status).color, boxShadow: machineStyle(m.status).glow ? `0 0 8px ${machineStyle(m.status).color}` : 'none' }" />
              <div class="min-w-0 flex-1">
                <div class="truncate text-xs font-medium" style="color: var(--text-primary);">{{ m.name }}</div>
                <div class="text-[11px]" style="color: var(--text-secondary);">OEE {{ m.oee.toFixed(1) }}% · {{ m.rate }} 件/分 · {{ m.temp.toFixed(1) }}℃</div>
              </div>
              <span
                class="rounded px-1.5 py-0.5 text-[10px] font-semibold"
                :style="{ color: machineStyle(m.status).color, background: `${machineStyle(m.status).color}1a` }"
              >
                {{ statusLabel(m.status) }}
              </span>
            </div>
          </div>
        </article>

        <article class="rounded-2xl border p-5" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <Package class="h-4 w-4" style="color: #10B981;" />
              <h3 class="text-sm font-semibold" style="color: var(--text-primary);">节拍与良率</h3>
            </div>
            <span class="inline-flex items-center gap-1.5 text-[11px] text-emerald-500">
              <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
              实时
            </span>
          </div>
          <div class="mt-3">
            <LiveSparkline :points="store.lineSeries" color="#0EA5E9" :height="66" />
          </div>
          <div class="mt-3">
            <LiveSparkline :points="store.okSeries" color="#10B981" :height="46" />
          </div>
          <div class="mt-3 grid grid-cols-2 gap-3 text-xs">
            <div class="rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
              <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">线平衡率</div>
              <div class="mt-1 font-mono-num text-base font-semibold" style="color: var(--text-primary);"><AnimatedNumber :value="store.lineOee" decimals="1" suffix="%" /></div>
            </div>
            <div class="rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
              <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">异常主机</div>
              <div class="mt-1 font-mono-num text-base font-semibold" :style="{ color: store.warnCount > 0 ? '#F59E0B' : '#10B981' }">{{ store.warnCount }} 台</div>
            </div>
          </div>
        </article>
      </div>
    </section>

    <P3EventFeed title="产线事件流" :events="sceneEvents" />
    </div>
  </VisualScreen>
</template>

<style scoped>
.pl-hero {
  background:
    linear-gradient(90deg, rgba(16, 185, 129, 0.07), transparent 40%),
    var(--bg-secondary);
  border-color: var(--border-color);
}

.pl-pill {
  background: rgba(16, 185, 129, 0.08);
  color: #10B981;
}

.pl-stage {
  background:
    radial-gradient(circle at 50% 40%, rgba(16, 185, 129, 0.07), transparent 60%),
    var(--bg-tertiary);
  border-color: var(--border-color);
}
</style>
