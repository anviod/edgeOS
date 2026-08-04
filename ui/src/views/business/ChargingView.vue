<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { Cable, CarFront, TimerReset, ArrowLeft, Clock, Zap } from 'lucide-vue-next'
import { useP3Page } from '@/composables/useP3Page'
import { useBusinessStore } from '@/stores/business'
import { useLiveClock } from '@/composables/useLiveClock'
import AnimatedNumber from '@/components/p3/AnimatedNumber.vue'
import LiveSparkline from '@/components/p3/LiveSparkline.vue'
import BarMeter from '@/components/p3/BarMeter.vue'
import LiveMetricStrip from '@/components/p3/LiveMetricStrip.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'
import P3ActionCenter from '@/components/p3/P3ActionCenter.vue'

const { page } = useP3Page('charging')
const store = useBusinessStore()
const { dateTime } = useLiveClock()

const toneColor: Record<string, string> = {
  hot: '#F59E0B',
  calm: '#0EA5E9',
  risk: '#EF4444',
}

function slotClass(slot: boolean) {
  return slot ? 'lane-slot--busy' : 'lane-slot--idle'
}

onMounted(() => store.start())
onUnmounted(() => store.stop())
</script>

<template>
  <div class="space-y-5">
    <section class="charging-top rounded-2xl border p-5">
      <div class="grid gap-5 xl:grid-cols-[1.1fr_0.9fr]">
        <div>
          <router-link
            to="/business-center"
            class="mb-3 inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm transition-colors hover:bg-white/5"
            style="color: var(--text-secondary); border: 1px solid var(--border-color);"
          >
            <ArrowLeft class="h-4 w-4" />
            返回业务中心
          </router-link>
          <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs charging-chip">
            <Cable class="h-3.5 w-3.5" />
            会话调度视图
          </div>
          <div class="mt-3 flex items-center gap-3">
            <h1 class="text-3xl font-semibold charging-title">{{ page.title }}</h1>
            <span class="inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1 font-mono-num text-xs" style="color: var(--text-secondary); border-color: var(--border-color); background: var(--bg-tertiary);">
              <Clock class="h-3.5 w-3.5" style="color: #F59E0B;" />
              {{ dateTime }}
            </span>
          </div>
          <p class="mt-2 text-sm charging-subtitle">{{ page.subtitle }}</p>

          <!-- 充电域实时指标 -->
          <div class="mt-5">
            <LiveMetricStrip
              :items="[
                { label: '在线充电枪', value: 68, unit: '把', color: '#0EA5E9' },
                { label: '当前会话', value: store.activeSessions, unit: '单', color: '#10B981', pulse: true },
                { label: '排队车辆', value: store.queueVehicles, unit: '台', color: '#F59E0B', pulse: true },
                { label: '站点利用率', value: store.busyRate, unit: '%', color: '#8B5CF6', pulse: true },
                { label: '平均排队', value: 12, unit: 'm', color: '#F59E0B' },
                { label: '异常订单', value: 2, unit: '单', color: '#EF4444' },
              ]"
            />
          </div>
        </div>

        <div class="grid gap-3 md:grid-cols-3 xl:grid-cols-3">
          <article class="rounded-xl border p-4 charging-session">
            <Zap class="h-5 w-5 text-emerald-500" />
            <div class="mt-4 text-xs charging-subtitle">正在充电</div>
            <div class="mt-1 text-2xl font-semibold charging-title">
              <AnimatedNumber :value="store.activeSessions" />
            </div>
            <div class="mt-2">
              <BarMeter :value="store.activeSessions" :max="68" color="#10B981" :height="4" />
            </div>
          </article>
          <article class="rounded-xl border p-4 charging-session">
            <CarFront class="h-5 w-5 text-amber-500" />
            <div class="mt-4 text-xs charging-subtitle">排队车辆</div>
            <div class="mt-1 text-2xl font-semibold charging-title">
              <AnimatedNumber :value="store.queueVehicles" />
            </div>
            <div class="mt-2">
              <BarMeter :value="store.queueVehicles" :max="18" color="#F59E0B" :height="4" />
            </div>
          </article>
          <article class="rounded-xl border p-4 charging-session">
            <TimerReset class="h-5 w-5 text-sky-500" />
            <div class="mt-4 text-xs charging-subtitle">平均等待</div>
            <div class="mt-1 text-2xl font-semibold charging-title">12m</div>
            <div class="mt-2">
              <BarMeter :value="54" color="#0EA5E9" :height="4" />
            </div>
          </article>
        </div>
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[0.92fr_1.08fr]">
      <section class="rounded-2xl border p-4 charging-board">
        <div class="flex items-center justify-between">
          <h3 class="text-sm font-semibold charging-title">排队与枪位车道</h3>
          <span class="inline-flex items-center gap-1.5 text-[11px] text-amber-500">
            <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-amber-500" />
            A 区压力最大 · 实时刷新
          </span>
        </div>
        <div class="mt-4 space-y-3">
          <div
            v-for="lane in store.lanes"
            :key="lane.id"
            class="rounded-xl border p-4 transition-all duration-300"
            :style="{ background: `${toneColor[lane.tone]}0d`, borderColor: 'var(--border-color)' }"
          >
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2 text-sm font-medium charging-title">
                <span class="inline-block h-2 w-2 animate-pulse rounded-full" :style="{ background: toneColor[lane.tone] }" />
                {{ lane.name }}
              </div>
              <div class="flex items-center gap-2 text-xs" :style="{ color: toneColor[lane.tone] }">
                <span class="font-mono-num"><AnimatedNumber :value="lane.occupied" />/{{ lane.total }} 占用</span>
              </div>
            </div>
            <div class="mt-3 flex gap-2">
              <span
                v-for="(slot, i) in lane.slots"
                :key="`${lane.id}-${i}`"
                class="lane-slot transition-all duration-500"
                :class="slotClass(slot)"
                :style="{ animationDelay: `${i * 40}ms` }"
              />
            </div>
            <div v-if="lane.note" class="mt-3 text-xs charging-subtitle">{{ lane.note }}</div>
          </div>
        </div>
      </section>

      <div class="grid gap-4 xl:grid-cols-2">
        <article class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-sm font-semibold charging-title">枪位占用走势</h3>
              <p class="mt-1 text-xs charging-subtitle">工作日午峰明显</p>
            </div>
            <span class="inline-flex items-center gap-1.5 text-[11px] text-emerald-500">
              <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
              实时
            </span>
          </div>
          <div class="mt-3">
            <LiveSparkline :points="store.sessionSeries" color="#10B981" :height="72" />
          </div>
        </article>
        <article class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-sm font-semibold charging-title">排队压力曲线</h3>
              <p class="mt-1 text-xs charging-subtitle">A 区压力高于其他站</p>
            </div>
            <span class="inline-flex items-center gap-1.5 text-[11px] text-amber-500">
              <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-amber-500" />
              实时
            </span>
          </div>
          <div class="mt-3">
            <LiveSparkline :points="store.sessionSeries.map(v => Math.min(60, v * 0.8))" color="#F59E0B" :height="72" />
          </div>
        </article>
        <div class="xl:col-span-2">
          <section class="rounded-xl border overflow-hidden" style="background: var(--bg-secondary); border-color: var(--border-color);">
            <div class="border-b px-5 py-4" style="border-color: var(--border-color);">
              <h3 class="text-sm font-semibold charging-title">充电站会话面板</h3>
              <p class="mt-1 text-xs charging-subtitle">{{ page.mainTable.description }}</p>
            </div>
            <div class="overflow-x-auto">
              <table class="min-w-full text-sm">
                <thead style="background: var(--bg-tertiary);">
                  <tr>
                    <th class="px-4 py-3 text-left text-xs font-semibold" style="color: var(--text-secondary);">站点</th>
                    <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">占用枪位</th>
                    <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">排队</th>
                    <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">负载</th>
                    <th class="px-4 py-3 text-left text-xs font-semibold" style="color: var(--text-secondary);">会话状态</th>
                    <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">运行态</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="lane in store.lanes"
                    :key="lane.id"
                    class="border-b"
                    :style="{ borderColor: 'var(--border-color)' }"
                  >
                    <td class="px-4 py-3">
                      <span class="inline-flex items-center gap-2">
                        <span class="inline-block h-2.5 w-2.5 rounded-full" :style="{ background: toneColor[lane.tone] }" />
                        <span style="color: var(--text-primary);">{{ lane.name }}</span>
                      </span>
                    </td>
                    <td class="px-4 py-3 text-center font-mono-num" style="color: var(--text-secondary);">
                      <AnimatedNumber :value="lane.occupied" />/{{ lane.total }}
                    </td>
                    <td class="px-4 py-3 text-center font-mono-num" style="color: var(--text-secondary);">
                      <AnimatedNumber :value="lane.id === 'a' ? store.queueVehicles : Math.max(0, store.queueVehicles - 3)" />
                    </td>
                    <td class="px-4 py-3 text-center font-mono-num" style="color: var(--text-secondary);">
                      {{ Math.round((lane.occupied / lane.total) * 100) }}%
                    </td>
                    <td class="px-4 py-3 text-left" style="color: var(--text-secondary);">
                      {{ lane.tone === 'risk' ? '支付回执延迟' : lane.tone === 'hot' ? '高峰运行' : '平稳' }}
                    </td>
                    <td class="px-4 py-3 text-center">
                      <span
                        class="inline-block rounded-lg px-2.5 py-1 text-xs"
                        :style="{ color: toneColor[lane.tone], background: `${toneColor[lane.tone]}1a` }"
                      >
                        {{ lane.tone === 'risk' ? '异常单' : lane.tone === 'hot' ? '拥堵' : '正常' }}
                      </span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>
        </div>
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[0.95fr_1.05fr]">
      <P3EventFeed :title="page.sidePanelTitle" :events="store.domainEvents('charging')" />
      <P3ActionCenter :title="page.actionTitle" :actions="page.actions" />
    </section>
  </div>
</template>

<style scoped>
.charging-top,
.charging-session,
.charging-board,
.lane {
  background: var(--bg-secondary);
  border-color: var(--border-color);
}

.charging-top {
  background:
    linear-gradient(90deg, rgba(245, 158, 11, 0.08), transparent 45%),
    var(--bg-secondary);
}

.charging-chip {
  background: rgba(245, 158, 11, 0.08);
  color: #F59E0B;
}

.charging-title { color: var(--text-primary); }
.charging-subtitle { color: var(--text-secondary); }

.lane-slot {
  display: inline-block;
  height: 12px;
  flex: 1;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.16);
  transition: background 0.5s ease;
}

.lane-slot--busy { background: rgba(245, 158, 11, 0.8); box-shadow: 0 0 6px rgba(245, 158, 11, 0.4); }
.lane-slot--busy-blue { background: rgba(14, 165, 233, 0.8); box-shadow: 0 0 6px rgba(14, 165, 233, 0.4); }
.lane-slot--idle { background: rgba(148, 163, 184, 0.16); }

.lane-hot { background: rgba(245, 158, 11, 0.05); }
.lane-calm { background: rgba(14, 165, 233, 0.05); }
.lane-risk { background: rgba(239, 68, 68, 0.05); }
</style>
