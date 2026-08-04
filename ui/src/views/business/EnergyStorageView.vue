<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
import { BatteryCharging, Gauge, ShieldCheck, ArrowLeft, Zap, Pause, Play, Clock } from 'lucide-vue-next'
import { useP3Page } from '@/composables/useP3Page'
import { useBusinessStore, type StorageMode } from '@/stores/business'
import { useLiveClock } from '@/composables/useLiveClock'
import AnimatedNumber from '@/components/p3/AnimatedNumber.vue'
import RingGauge from '@/components/p3/RingGauge.vue'
import LiveSparkline from '@/components/p3/LiveSparkline.vue'
import BarMeter from '@/components/p3/BarMeter.vue'
import LiveMetricStrip from '@/components/p3/LiveMetricStrip.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'
import ExecutionTimeline from '@/components/p3/ExecutionTimeline.vue'
import P3ActionCenter from '@/components/p3/P3ActionCenter.vue'

const { page } = useP3Page('energy-storage')
const store = useBusinessStore()
const { dateTime } = useLiveClock()

const modeMeta = computed(() => {
  switch (store.storageMode) {
    case 'charging': return { label: '充电中', color: '#10B981' }
    case 'discharging': return { label: '放电中', color: '#F59E0B' }
    default: return { label: '待机', color: '#64748B' }
  }
})

function setMode(mode: StorageMode) {
  store.setStorageMode(mode)
}

onMounted(() => store.start())
onUnmounted(() => store.stop())
</script>

<template>
  <div class="space-y-5">
    <section class="storage-hero rounded-2xl border p-5">
      <div class="grid gap-5 xl:grid-cols-[300px_1fr_300px]">
        <div class="rounded-2xl border p-5 storage-gauge">
          <div class="flex items-center justify-between">
            <span class="text-xs uppercase tracking-[0.18em] storage-muted">储能总仓</span>
            <BatteryCharging class="h-5 w-5" :style="{ color: modeMeta.color }" />
          </div>
          <div class="mt-4 flex items-center justify-center">
            <RingGauge :value="store.avgSoc" :color="modeMeta.color">
              <div class="text-xs" style="color: var(--text-muted);">Average SOC</div>
              <div class="mt-1 text-4xl font-semibold" style="color: var(--text-primary);">
                <AnimatedNumber :value="store.avgSoc" decimals="0" suffix="%" />
              </div>
              <span
                class="mt-2 inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-medium"
                :style="{ color: modeMeta.color, background: `${modeMeta.color}1a` }"
              >
                <span class="h-1.5 w-1.5 animate-pulse rounded-full" :style="{ background: modeMeta.color }" />
                {{ modeMeta.label }}
              </span>
            </RingGauge>
          </div>
          <div class="mt-4 grid grid-cols-2 gap-3 text-xs">
            <div class="rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
              <div class="storage-muted">SOH</div>
              <div class="mt-1 text-base font-semibold" style="color: var(--text-primary);">
                <AnimatedNumber :value="store.avgSoh" suffix=" 分" />
              </div>
            </div>
            <div class="rounded-lg border px-3 py-2" style="background: var(--bg-tertiary); border-color: var(--border-color);">
              <div class="storage-muted">Power</div>
              <div class="mt-1 text-base font-semibold" style="color: var(--text-primary);">
                <AnimatedNumber :value="store.totalPower" decimals="1" suffix="MW" />
              </div>
            </div>
          </div>
        </div>

        <div class="space-y-4">
          <div>
            <router-link
              to="/business-center"
              class="mb-3 inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm transition-colors hover:bg-white/5"
              style="color: var(--text-secondary); border: 1px solid var(--border-color);"
            >
              <ArrowLeft class="h-4 w-4" />
              返回业务中心
            </router-link>
            <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs storage-chip">
              <Gauge class="h-3.5 w-3.5" />
              站级调峰运行面板
            </div>
            <div class="mt-3 flex items-center gap-3">
              <h1 class="text-3xl font-semibold storage-title">{{ page.title }}</h1>
              <span class="inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1 font-mono-num text-xs" style="color: var(--text-secondary); border-color: var(--border-color); background: var(--bg-tertiary);">
                <Clock class="h-3.5 w-3.5" style="color: #0EA5E9;" />
                {{ dateTime }}
              </span>
            </div>
            <p class="mt-2 text-sm storage-subtitle">{{ page.subtitle }}</p>
          </div>

          <!-- 储能域实时指标 -->
          <LiveMetricStrip
            :items="[
              { label: '站点数', value: 12, unit: '座', color: '#0EA5E9' },
              { label: '平均 SOC', value: store.avgSoc, unit: '%', decimals: 1, color: '#10B981', pulse: true, badge: modeMeta.label },
              { label: '总功率', value: store.totalPower, unit: 'MW', decimals: 1, color: '#F59E0B', pulse: true },
              { label: 'Latency', value: store.storageLatency, unit: 'ms', color: '#8B5CF6' },
              { label: 'Loss', value: store.storageLoss, unit: '%', decimals: 1, color: '#F59E0B' },
              { label: 'Quality', value: store.storageQuality, color: '#10B981' },
            ]"
          />

          <!-- 实时控制区 -->
          <div class="rounded-2xl border p-4" style="background: var(--bg-tertiary); border-color: var(--border-color);">
            <div class="flex items-center justify-between">
              <div class="text-xs uppercase tracking-[0.16em] storage-muted">实时控制区</div>
              <span class="inline-flex items-center gap-1.5 text-[11px] font-medium" :style="{ color: modeMeta.color }">
                <span class="h-1.5 w-1.5 animate-pulse rounded-full" :style="{ background: modeMeta.color }" />
                策略 {{ modeMeta.label }}
              </span>
            </div>
            <div class="mt-3 grid grid-cols-3 gap-2">
              <button
                class="flex flex-col items-center gap-1.5 rounded-lg px-3 py-2.5 text-sm font-medium transition-all"
                :class="store.storageMode === 'charging' ? 'control-active' : 'control-ghost'"
                :style="{ color: '#10B981', background: store.storageMode === 'charging' ? 'rgba(16,185,129,0.14)' : 'transparent' }"
                @click="setMode('charging')"
              >
                <Zap class="h-4 w-4" />
                开始充电
              </button>
              <button
                class="flex flex-col items-center gap-1.5 rounded-lg px-3 py-2.5 text-sm font-medium transition-all"
                :class="store.storageMode === 'discharging' ? 'control-active' : 'control-ghost'"
                :style="{ color: '#F59E0B', background: store.storageMode === 'discharging' ? 'rgba(245,158,11,0.14)' : 'transparent' }"
                @click="setMode('discharging')"
              >
                <Pause class="h-4 w-4" />
                开始放电
              </button>
              <button
                class="flex flex-col items-center gap-1.5 rounded-lg px-3 py-2.5 text-sm font-medium transition-all"
                :class="store.storageMode === 'idle' ? 'control-active' : 'control-ghost'"
                :style="{ color: '#64748B', background: store.storageMode === 'idle' ? 'rgba(100,116,139,0.14)' : 'transparent' }"
                @click="setMode('idle')"
              >
                <Play class="h-4 w-4" />
                停止策略
              </button>
            </div>
            <div class="mt-3 space-y-2 text-xs">
              <div class="flex items-center justify-between">
                <span class="storage-muted">SOC 变化速率</span>
                <span class="font-mono-num" :style="{ color: modeMeta.color }">
                  {{ store.storageMode === 'charging' ? '+0.7%/tick' : store.storageMode === 'discharging' ? '-0.7%/tick' : '≈0%/tick' }}
                </span>
              </div>
              <div class="flex items-center justify-between">
                <span class="storage-muted">功率指令</span>
                <span class="font-mono-num" style="color: var(--text-primary);">
                  <AnimatedNumber :value="store.totalPower" decimals="1" suffix=" MW" />
                </span>
              </div>
            </div>
          </div>
        </div>

        <div class="rounded-2xl border p-5 storage-right">
          <div class="flex items-center justify-between">
            <div>
              <div class="text-xs uppercase tracking-[0.18em] storage-muted">主站点</div>
              <div class="mt-2 text-lg font-semibold storage-title">{{ page.mainTable.rows[0].site }}</div>
            </div>
            <ShieldCheck class="h-5 w-5 text-sky-500" />
          </div>
          <div class="mt-5 space-y-3 text-sm">
            <div class="flex justify-between">
              <span class="storage-muted">SOC</span>
              <span class="font-mono-num storage-title">
                <AnimatedNumber :value="store.sites[0]?.soc ?? 0" decimals="0" suffix="%" />
              </span>
            </div>
            <div class="flex justify-between">
              <span class="storage-muted">SOH</span>
              <span class="font-mono-num storage-title">
                <AnimatedNumber :value="store.sites[0]?.soh ?? 0" />
              </span>
            </div>
            <div class="flex justify-between">
              <span class="storage-muted">功率</span>
              <span class="font-mono-num storage-title">
                <AnimatedNumber :value="store.sites[0]?.power ?? 0" decimals="1" suffix="MW" />
              </span>
            </div>
            <div class="flex justify-between">
              <span class="storage-muted">策略</span>
              <span class="storage-title">{{ page.mainTable.rows[0].strategy }}</span>
            </div>
          </div>
          <div class="mt-5 space-y-4">
            <div>
              <div class="mb-1 flex justify-between text-xs">
                <span class="storage-muted">主站 SOC</span>
                <span class="font-mono-num" style="color: var(--text-primary);">
                  <AnimatedNumber :value="store.sites[0]?.soc ?? 0" decimals="0" suffix="%" />
                </span>
              </div>
              <BarMeter :value="store.sites[0]?.soc ?? 0" :color="modeMeta.color" :height="6" />
            </div>
            <div>
              <div class="mb-1 flex justify-between text-xs">
                <span class="storage-muted">主站功率</span>
                <span class="font-mono-num" style="color: var(--text-primary);">
                  <AnimatedNumber :value="store.sites[0]?.power ?? 0" decimals="1" suffix="MW" />
                </span>
              </div>
              <BarMeter :value="(store.sites[0]?.power ?? 0) * 10" :max="100" color="#0EA5E9" :height="6" />
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- 实时趋势区 -->
    <section class="grid gap-4 xl:grid-cols-2">
      <article class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
        <div class="flex items-center justify-between">
          <div>
            <h3 class="text-sm font-semibold storage-title">SOC 实时趋势</h3>
            <p class="mt-1 text-xs storage-subtitle">{{ page.trends[0].summary }}</p>
          </div>
          <span class="inline-flex items-center gap-1.5 text-[11px] text-emerald-500">
            <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
            实时
          </span>
        </div>
        <div class="mt-3">
          <LiveSparkline :points="store.socSeries" color="#10B981" :height="72" />
        </div>
      </article>
      <article class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
        <div class="flex items-center justify-between">
          <div>
            <h3 class="text-sm font-semibold storage-title">功率调节质量</h3>
            <p class="mt-1 text-xs storage-subtitle">{{ page.trends[2].summary }}</p>
          </div>
          <span class="inline-flex items-center gap-1.5 text-[11px] text-sky-500">
            <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-sky-500" />
            实时
          </span>
        </div>
        <div class="mt-3">
          <LiveSparkline :points="store.powerSeries" color="#0EA5E9" :height="72" />
        </div>
      </article>
    </section>

    <!-- 站点清单 -->
    <section class="rounded-2xl border overflow-hidden" style="background: var(--bg-secondary); border-color: var(--border-color);">
      <div class="border-b px-5 py-4" style="border-color: var(--border-color);">
        <h3 class="text-sm font-semibold storage-title">储能站点清单</h3>
        <p class="mt-1 text-xs storage-subtitle">{{ page.mainTable.description }}</p>
      </div>
      <div class="overflow-x-auto">
        <table class="min-w-full text-sm">
          <thead style="background: var(--bg-tertiary);">
            <tr>
              <th class="px-4 py-3 text-left text-xs font-semibold" style="color: var(--text-secondary);">站点</th>
              <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">SOC</th>
              <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">SOH</th>
              <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">功率</th>
              <th class="px-4 py-3 text-left text-xs font-semibold" style="color: var(--text-secondary);">调峰策略</th>
              <th class="px-4 py-3 text-center text-xs font-semibold" style="color: var(--text-secondary);">状态</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="site in store.sites"
              :key="site.id"
              class="border-b"
              :style="{ borderColor: 'var(--border-color)' }"
            >
              <td class="px-4 py-3">
                <span class="inline-flex items-center gap-2">
                  <span class="inline-block h-2.5 w-2.5 rounded-full" :style="{ background: site.status === 'warn' ? '#EF4444' : site.status === 'watch' ? '#F59E0B' : '#10B981' }" />
                  <span style="color: var(--text-primary);">{{ site.name }}</span>
                </span>
              </td>
              <td class="px-4 py-3 text-center font-mono-num" style="color: var(--text-secondary);">
                <AnimatedNumber :value="site.soc" decimals="0" suffix="%" />
              </td>
              <td class="px-4 py-3 text-center font-mono-num" style="color: var(--text-secondary);">
                <AnimatedNumber :value="site.soh" />
              </td>
              <td class="px-4 py-3 text-center font-mono-num" style="color: var(--text-secondary);">
                <AnimatedNumber :value="site.power" decimals="1" suffix="MW" />
              </td>
              <td class="px-4 py-3 text-left" style="color: var(--text-secondary);">{{ site.strategy }}</td>
              <td class="px-4 py-3 text-center">
                <span
                  class="inline-block rounded-lg px-2.5 py-1 text-xs"
                  :style="{
                    color: site.status === 'warn' ? '#EF4444' : site.status === 'watch' ? '#F59E0B' : '#10B981',
                    background: site.status === 'warn' ? 'rgba(239,68,68,0.1)' : site.status === 'watch' ? 'rgba(245,158,11,0.1)' : 'rgba(16,185,129,0.1)',
                  }"
                >
                  {{ site.status === 'warn' ? '寿命预警' : site.status === 'watch' ? '观察中' : '运行中' }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[1.15fr_0.85fr]">
      <P3EventFeed :title="page.sidePanelTitle" :events="store.domainEvents('storage')" />
      <div class="space-y-4">
        <ExecutionTimeline :title="page.timelineTitle" :items="page.timeline" />
        <P3ActionCenter :title="page.actionTitle" :actions="page.actions" />
      </div>
    </section>
  </div>
</template>

<style scoped>
.storage-hero,
.storage-gauge,
.storage-right {
  background: var(--bg-secondary);
  border-color: var(--border-color);
}

.storage-hero {
  background:
    radial-gradient(circle at top left, rgba(16, 185, 129, 0.1), transparent 28%),
    var(--bg-secondary);
}

.storage-chip {
  background: rgba(16, 185, 129, 0.08);
  color: #10B981;
}

.storage-title { color: var(--text-primary); }
.storage-subtitle,
.storage-muted { color: var(--text-secondary); }

.control-ghost {
  border: 1px solid var(--border-color);
  background: var(--bg-secondary);
}

.control-active {
  border: 1px solid transparent;
  box-shadow: inset 0 0 0 1px currentColor;
}
</style>
