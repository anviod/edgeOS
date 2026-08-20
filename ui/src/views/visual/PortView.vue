<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
import { Ship, Container, Truck, Anchor, Activity, Waves } from 'lucide-vue-next'
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
import IsoWater from '@/components/visual/IsoWater.vue'
import IsoShip from '@/components/visual/IsoShip.vue'
import IsoQuayCrane from '@/components/visual/IsoQuayCrane.vue'
import IsoContainer from '@/components/visual/IsoContainer.vue'
import IsoTruck from '@/components/visual/IsoTruck.vue'

const store = useVisualStore()
const { dateTime } = useLiveClock()

const CONTAINER_COLORS = ['#2563EB', '#DC2626', '#16A34A', '#EA580C', '#7C3AED', '#0284C7']

// 堆场箱区
const YARD_ROWS = [
  { y: 40, count: 3, height: 3 },
  { y: 124, count: 3, height: 2 },
  { y: 208, count: 3, height: 3 },
]

const yardStacks = computed(() => {
  const list: { x: number; y: number; levels: number; colors: string[] }[] = []
  YARD_ROWS.forEach(row => {
    for (let i = 0; i < row.count; i++) {
      const x = 404 + i * 52
      const colors = Array.from({ length: row.height }, (_, k) => CONTAINER_COLORS[(x + row.y + k * 2) % CONTAINER_COLORS.length])
      list.push({ x, y: row.y, levels: row.height, colors })
    }
  })
  return list
})

// 靠泊 2 号位进港船舶运动
const berthShare = computed(() => Math.round(store.teuPerHour / 640 * 100))

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
      <section class="rounded-2xl border p-5 port-hero">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs port-pill">
              <Anchor class="h-3.5 w-3.5" />
              港口运输 · 2.5D 码头调度
            </div>
            <div class="mt-2 flex items-center gap-3">
              <h1 class="text-2xl font-semibold" style="color: var(--text-primary);">港口运输</h1>
              <span class="font-mono-num text-xs" style="color: var(--text-secondary);">{{ dateTime }}</span>
            </div>
          </div>
          <div class="flex items-center gap-2 text-xs">
            <span class="rounded-lg px-2.5 py-1" style="background: var(--bg-tertiary); color: var(--text-secondary);">3 泊位 · 6 岸桥</span>
            <span class="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1" style="background: rgba(16,185,129,0.1); color: #10B981;">
              <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
              作业中
            </span>
          </div>
        </div>

        <div class="mt-4">
          <LiveMetricStrip
            :items="[
              { label: '今日吞吐', value: store.teuToday, unit: 'TEU', color: '#0EA5E9' },
              { label: '吞吐速率', value: store.teuPerHour, unit: 'TEU/h', color: '#8B5CF6', pulse: true },
              { label: '在港船舶', value: store.shipsInPort, unit: '艘', color: '#38BDF8' },
              { label: '泊位利用率', value: store.berthOccupancy, unit: '%', color: '#F59E0B' },
              { label: '岸桥作业率', value: store.craneUtil, unit: '%', color: '#10B981' },
              { label: '闸口排队', value: store.trucksQueued, unit: '辆', color: '#EF4444' },
            ]"
          />
        </div>
      </section>

      <section class="grid gap-4 xl:grid-cols-[1.3fr_0.7fr]">
        <article class="rounded-2xl border p-5" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-sm font-semibold" style="color: var(--text-primary);">码头等距视图</h3>
              <p class="mt-1 text-xs" style="color: var(--text-secondary);">货轮进出港 · 岸桥装卸 · 堆场调运 · 集卡转运</p>
            </div>
            <div class="flex items-center gap-3 text-[11px]" style="color: var(--text-secondary);">
              <span class="flex items-center gap-1"><Waves class="h-3 w-3" style="color: #38BDF8;" />水域</span>
              <span class="flex items-center gap-1"><span class="h-2 w-2 rounded-sm" style="background: #F59E0B;" />岸桥</span>
              <span class="flex items-center gap-1"><span class="h-2 w-2 rounded-sm" style="background: #2563EB;" />集装箱</span>
            </div>
          </div>

          <div class="mt-2 port-stage rounded-xl border p-3">
            <IsoScene :cols="12" :rows="8" :cell="46" :height="420" :scale="0.78">
              <!-- 水域 -->
              <IsoWater :x="0" :y="0" :width="200" :depth="368" :height="5" />

              <!-- 码头岸线 + 堆场陆地 -->
              <IsoCube :x="200" :y="0" :width="352" :depth="368" :height="8"
                top-color="rgba(100,116,139,0.3)" front-color="rgba(100,116,139,0.45)" side-color="rgba(71,85,105,0.6)" />
              <!-- 岸线边缘线 -->
              <IsoCube :x="200" :y="0" :width="10" :depth="368" :height="12"
                top-color="rgba(226,232,240,0.5)" front-color="rgba(226,232,240,0.65)" side-color="rgba(203,213,225,0.8)" />

              <!-- 靠泊船（作业中） -->
              <IsoShip :x="16" :y="48" :width="150" :depth="56" :hull-h="24" color="#475569" label="MSC 圣保罗号"
                :hud="{ foot: `吞吐速率 ${store.teuPerHour} TEU/h · 泊位利用率 ${store.berthOccupancy}%` }" />
              <!-- 进港船 -->
              <IsoShip :x="16" :y="284" :width="120" :depth="46" :hull-h="20" color="#334155" moving :move-from="-190" :move-dur="38" label="COSCO 远航号"
                :hud="{ foot: `吞吐速率 ${store.teuPerHour} TEU/h · 岸桥作业 ${store.cranesWorking} 台` }" />

              <!-- 岸桥 -->
              <IsoQuayCrane :x="214" :y="44" :jib-len="168" :back-len="36" :height="104" :depth="18" label="岸桥 Q7" />

              <!-- 装卸中转箱 -->
              <IsoContainer :x="228" :y="160" :width="44" :depth="22" :height="22" color="#DC2626" />
              <IsoContainer :x="228" :y="200" :width="44" :depth="22" :height="22" color="#2563EB" />

              <!-- 堆场箱区 -->
              <template v-for="(stack, si) in yardStacks" :key="si">
                <IsoContainer
                  v-for="(c, li) in stack.colors"
                  :key="`${si}-${li}`"
                  :x="stack.x"
                  :y="stack.y"
                  :z="li * 22"
                  :color="c"
                />
              </template>

              <!-- 港调中心 -->
              <IsoCube :x="404" :y="280" :width="64" :depth="44" :height="30"
                top-color="rgba(14,165,233,0.18)" front-color="rgba(14,165,233,0.34)" side-color="rgba(7,108,168,0.48)" label="港调中心" />
              <IsoCube :x="412" :y="288" :width="48" :depth="28" :height="20"
                top-color="rgba(30,41,59,0.6)" front-color="rgba(30,41,59,0.85)" side-color="rgba(30,41,59,0.9)" />

              <!-- 闸口 -->
              <IsoCube :x="488" :y="340" :width="6" :depth="6" :height="40" top-color="rgba(15,23,42,0.7)" front-color="rgba(15,23,42,0.9)" side-color="rgba(15,23,42,0.8)" />
              <IsoCube :x="540" :y="340" :width="6" :depth="6" :height="40" top-color="rgba(15,23,42,0.7)" front-color="rgba(15,23,42,0.9)" side-color="rgba(15,23,42,0.8)" />
              <IsoCube :x="486" :y="338" :width="62" :depth="10" :height="5" :z="40"
                top-color="rgba(245,158,11,0.5)" front-color="rgba(245,158,11,0.7)" side-color="rgba(180,120,10,0.85)" label="闸口" />

              <!-- 道路 -->
              <IsoCube :x="200" :y="336" :width="352" :depth="26" :height="4"
                top-color="rgba(15,23,42,0.55)" front-color="rgba(15,23,42,0.75)" side-color="rgba(15,23,42,0.85)" />
              <!-- 道路标线 -->
              <IsoCube :x="200" :y="348" :width="352" :depth="2" :height="5"
                top-color="rgba(250,204,21,0.5)" front-color="rgba(250,204,21,0.6)" side-color="rgba(202,138,4,0.7)" />

              <!-- 集卡 -->
              <IsoTruck :x="214" :y="342" :width="66" :depth="20" container-color="#2563EB" moving :move-from="214" :move-range="520" :move-dur="15" label="A-12" />
              <IsoTruck :x="320" :y="354" :width="66" :depth="20" container-color="#16A34A" moving :move-from="320" :move-range="516" :move-dur="20" :move-delay="2.5" label="B-07" />
              <IsoTruck :x="420" :y="344" :width="66" :depth="20" :has-container="false" color="#64748B" label="空载" />

              <!-- 航标 -->
              <IsoCube :x="40" :y="340" :width="8" :depth="8" :height="26" top-color="rgba(245,158,11,0.5)" front-color="rgba(245,158,11,0.7)" side-color="rgba(180,120,10,0.8)" :beacon="true" beacon-color="#F59E0B" />
            </IsoScene>
          </div>

          <div class="mt-4 grid grid-cols-4 gap-3 text-center">
            <div class="rounded-lg border px-3 py-3" style="background: var(--bg-tertiary); border-color: var(--border-color);">
              <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">累计进港集卡</div>
              <div class="mt-1 font-mono-num text-xl font-bold" style="color: #0EA5E9;"><AnimatedNumber :value="store.trucksIn" /></div>
            </div>
            <div class="rounded-lg border px-3 py-3" style="background: var(--bg-tertiary); border-color: var(--border-color);">
              <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">集卡流量 辆/h</div>
              <div class="mt-1 font-mono-num text-xl font-bold" style="color: #10B981;"><AnimatedNumber :value="store.truckFlow" /></div>
            </div>
            <div class="rounded-lg border px-3 py-3" style="background: var(--bg-tertiary); border-color: var(--border-color);">
              <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">作业岸桥</div>
              <div class="mt-1 font-mono-num text-xl font-bold" style="color: #F59E0B;"><AnimatedNumber :value="store.cranesWorking" /></div>
            </div>
            <div class="rounded-lg border px-3 py-3" style="background: var(--bg-tertiary); border-color: var(--border-color);">
              <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">闸口排队</div>
              <div class="mt-1 font-mono-num text-xl font-bold" :style="{ color: store.trucksQueued > 10 ? '#EF4444' : '#10B981' }"><AnimatedNumber :value="store.trucksQueued" /></div>
            </div>
          </div>
        </article>

        <div class="grid gap-4">
          <article class="rounded-2xl border p-5" style="background: var(--bg-secondary); border-color: var(--border-color);">
            <div class="flex items-center gap-2">
              <Container class="h-4 w-4" style="color: #0EA5E9;" />
              <h3 class="text-sm font-semibold" style="color: var(--text-primary);">泊位利用</h3>
            </div>
            <div class="mt-3 flex items-center justify-around">
              <RingGauge :value="store.berthOccupancy" :size="150" color="#F59E0B" label="泊位占用">
                <span class="font-mono-num text-2xl font-bold" style="color: #F59E0B;">
                  <AnimatedNumber :value="store.berthOccupancy" decimals="1" suffix="%" />
                </span>
              </RingGauge>
              <div class="space-y-3 text-xs">
                <div class="flex items-center gap-2 rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
                  <Ship class="h-4 w-4" style="color: #38BDF8;" />
                  <span style="color: var(--text-secondary);">在港船舶</span>
                  <span class="ml-auto font-mono-num font-semibold" style="color: var(--text-primary);"><AnimatedNumber :value="store.shipsInPort" suffix=" 艘" /></span>
                </div>
                <div class="flex items-center gap-2 rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
                  <Truck class="h-4 w-4" style="color: #10B981;" />
                  <span style="color: var(--text-secondary);">集卡流量</span>
                  <span class="ml-auto font-mono-num font-semibold" style="color: var(--text-primary);"><AnimatedNumber :value="store.truckFlow" suffix=" 辆/h" /></span>
                </div>
                <div class="flex items-center gap-2 rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
                  <Activity class="h-4 w-4" style="color: #F59E0B;" />
                  <span style="color: var(--text-secondary);">岸桥作业率</span>
                  <span class="ml-auto font-mono-num font-semibold" style="color: var(--text-primary);"><AnimatedNumber :value="store.craneUtil" suffix="%" /></span>
                </div>
              </div>
            </div>
          </article>

          <article class="rounded-2xl border p-5" style="background: var(--bg-secondary); border-color: var(--border-color);">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <Activity class="h-4 w-4" style="color: #8B5CF6;" />
                <h3 class="text-sm font-semibold" style="color: var(--text-primary);">吞吐趋势</h3>
              </div>
              <span class="inline-flex items-center gap-1.5 text-[11px] text-emerald-500">
                <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
                实时
              </span>
            </div>
            <div class="mt-3">
              <LiveSparkline :points="store.teuSeries" color="#8B5CF6" :height="70" />
            </div>
            <div class="mt-3">
              <LiveSparkline :points="store.throughputSeries" color="#0EA5E9" :height="42" />
            </div>
            <div class="mt-3 grid grid-cols-2 gap-3 text-xs">
              <div class="rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
                <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">吞吐速率</div>
                <div class="mt-1 font-mono-num text-base font-semibold" style="color: #8B5CF6;"><AnimatedNumber :value="store.teuPerHour" suffix=" TEU/h" /></div>
              </div>
              <div class="rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
                <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">泊位饱和度</div>
                <div class="mt-1 font-mono-num text-base font-semibold" :style="{ color: berthShare > 80 ? '#F59E0B' : '#10B981' }">{{ berthShare }}%</div>
              </div>
            </div>
          </article>
        </div>
      </section>

      <P3EventFeed title="港口调度事件流" :events="sceneEvents" />
    </div>
  </VisualScreen>
</template>

<style scoped>
.port-hero {
  background:
    linear-gradient(90deg, rgba(14, 165, 233, 0.08), transparent 40%),
    var(--bg-secondary);
  border-color: var(--border-color);
}

.port-pill {
  background: rgba(14, 165, 233, 0.08);
  color: #0EA5E9;
}

.port-stage {
  background:
    radial-gradient(circle at 50% 40%, rgba(56, 189, 248, 0.07), transparent 60%),
    var(--bg-tertiary);
  border-color: var(--border-color);
}
</style>
