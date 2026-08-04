<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
import { BarChart3, BatteryCharging, Cable, Cpu, Receipt, Zap, Clock } from 'lucide-vue-next'
import { useP3Page } from '@/composables/useP3Page'
import { useBusinessStore } from '@/stores/business'
import { useLiveClock } from '@/composables/useLiveClock'
import AnimatedNumber from '@/components/p3/AnimatedNumber.vue'
import LiveMetricStrip from '@/components/p3/LiveMetricStrip.vue'
import LiveSparkline from '@/components/p3/LiveSparkline.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'

const { page } = useP3Page('business-center')
const store = useBusinessStore()
const { dateTime } = useLiveClock()

const modules = [
  { label: '储能管理', path: '/business-center/energy-storage', icon: BatteryCharging, note: '调峰 / SOC / SOH' },
  { label: '电源BMS', path: '/business-center/power-bms', icon: Cpu, note: '温差 / 压差 / 均衡' },
  { label: '充电管理', path: '/business-center/charging', icon: Cable, note: '枪位 / 排队 / 订单' },
  { label: '能耗监测', path: '/business-center/energy-monitoring', icon: BarChart3, note: '能流 / 峰谷 / 异常' },
  { label: '账务台账', path: '/business-center/ledger', icon: Receipt, note: '账单 / 发票 / 对账' },
]

const revenueDelta = computed(() => 8.2)

onMounted(() => store.start())
onUnmounted(() => store.stop())
</script>

<template>
  <div class="space-y-5">
    <section class="business-hero rounded-2xl border p-5">
      <div class="grid gap-5 xl:grid-cols-[1.25fr_0.75fr]">
        <div>
          <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs business-pill">
            <Zap class="h-3.5 w-3.5" />
            业务经营驾驶舱
          </div>
          <div class="mt-3 flex items-center gap-3">
            <h1 class="text-3xl font-semibold business-title">{{ page.title }}</h1>
            <span class="inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1 font-mono-num text-xs" style="color: var(--text-secondary); border-color: var(--border-color); background: var(--bg-tertiary);">
              <Clock class="h-3.5 w-3.5" style="color: #0EA5E9;" />
              {{ dateTime }}
            </span>
          </div>
          <p class="mt-2 max-w-3xl text-sm business-subtitle">{{ page.subtitle }}</p>

          <!-- 总览实时指标 -->
          <div class="mt-5">
            <LiveMetricStrip
              :items="[
                { label: '在线站点', value: 38, unit: '座', color: '#0EA5E9' },
                { label: '采集设备', value: 428, unit: '台', color: '#8B5CF6' },
                { label: '活跃告警', value: 7, unit: '条', color: '#EF4444', pulse: true },
                { label: '平均延迟', value: 24, unit: 'ms', color: '#F59E0B' },
                { label: '累计充放电', value: 924, unit: 'MWh', color: '#10B981' },
                { label: '调度完成率', value: 96, unit: '%', color: '#10B981', pulse: true },
              ]"
            />
          </div>
        </div>

        <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-1">
          <router-link
            v-for="module in modules"
            :key="module.path"
            :to="module.path"
            class="business-module rounded-xl border p-4 transition-all duration-300 hover:-translate-y-0.5"
          >
            <div class="flex items-center justify-between gap-3">
              <div>
                <div class="text-sm font-medium business-title">{{ module.label }}</div>
                <div class="mt-1 text-xs business-subtitle">{{ module.note }}</div>
              </div>
              <component :is="module.icon" class="h-5 w-5 business-icon" />
            </div>
          </router-link>
        </div>
      </div>
    </section>

    <!-- 今日收益累计动效 -->
    <section class="grid gap-4 xl:grid-cols-[0.9fr_1.1fr]">
      <article class="rounded-2xl border p-5 revenue-card">
        <div class="flex items-center justify-between">
          <span class="text-xs uppercase tracking-[0.18em]" style="color: var(--text-muted);">今日收益</span>
          <span class="rounded-lg px-2 py-1 text-[11px] font-medium text-emerald-500" style="background: rgba(16,185,129,0.1);">
            +{{ revenueDelta.toFixed(1) }}%
          </span>
        </div>
        <div class="mt-2 flex items-baseline gap-1">
          <span class="text-4xl font-semibold text-emerald-500">
            <AnimatedNumber :value="store.todayRevenue" decimals="1" />
          </span>
          <span class="text-sm" style="color: var(--text-secondary);">万元</span>
        </div>
        <p class="mt-1 text-xs" style="color: var(--text-secondary);">
          峰谷套利持续上扬 · <span class="font-mono-num text-emerald-500">+0.35万/拍</span> 稳步累计
        </p>
        <div class="mt-4">
          <LiveSparkline :points="store.revenueSeries" color="#10B981" :height="72" />
        </div>
        <div class="mt-4 grid grid-cols-2 gap-3 text-xs">
          <div class="rounded-lg border px-3 py-2.5" style="background: var(--bg-tertiary); border-color: var(--border-color);">
            <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">储能站点负载</div>
            <div class="mt-1 flex items-center gap-2">
              <span class="text-lg font-semibold font-mono-num" style="color: var(--text-primary);"><AnimatedNumber :value="store.avgSoc" decimals="0" suffix="%" /></span>
              <span class="text-[11px] text-emerald-500">SOC</span>
            </div>
          </div>
          <div class="rounded-lg border px-3 py-2.5" style="background: var(--bg-tertiary); border-color: var(--border-color);">
            <div class="text-[11px] uppercase tracking-[0.14em]" style="color: var(--text-muted);">充电枪利用率</div>
            <div class="mt-1 flex items-center gap-2">
              <span class="text-lg font-semibold font-mono-num" style="color: var(--text-primary);"><AnimatedNumber :value="store.busyRate" suffix="%" /></span>
              <span class="text-[11px] text-amber-500">占用</span>
            </div>
          </div>
        </div>
      </article>

      <div class="grid gap-4">
        <div class="grid gap-4 md:grid-cols-2">
          <article class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
            <div class="flex items-center justify-between">
              <div>
                <h3 class="text-sm font-semibold" style="color: var(--text-primary);">站点通信质量</h3>
                <p class="mt-1 text-xs" style="color: var(--text-secondary);">关键站点链路无明显抖动</p>
              </div>
              <span class="inline-flex items-center gap-1.5 text-[11px] text-emerald-500">
                <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
                实时
              </span>
            </div>
            <div class="mt-3">
              <LiveSparkline :points="store.energySeries.map(v => Math.round(90 + (v % 8)))" color="#0EA5E9" :height="64" />
            </div>
          </article>
          <article class="rounded-xl border p-4" style="background: var(--bg-secondary); border-color: var(--border-color);">
            <div class="flex items-center justify-between">
              <div>
                <h3 class="text-sm font-semibold" style="color: var(--text-primary);">经营风险热区</h3>
                <p class="mt-1 text-xs" style="color: var(--text-secondary);">账务异常集中在 2 个区域</p>
              </div>
              <span class="inline-flex items-center gap-1.5 text-[11px] text-amber-500">
                <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-amber-500" />
                实时
              </span>
            </div>
            <div class="mt-3">
              <LiveSparkline :points="store.energySeries.map(v => Math.round(55 + (v % 20)))" color="#F59E0B" :height="64" />
            </div>
          </article>
        </div>
        <P3EventFeed :title="page.sidePanelTitle" :events="store.domainEvents('overview')" />
      </div>
    </section>
  </div>
</template>

<style scoped>
.business-hero {
  background:
    linear-gradient(90deg, rgba(14, 165, 233, 0.08), transparent 40%),
    var(--bg-secondary);
  border-color: var(--border-color);
}

.business-pill {
  background: rgba(14, 165, 233, 0.08);
  color: #0EA5E9;
}

.revenue-card {
  background:
    radial-gradient(circle at top right, rgba(16, 185, 129, 0.12), transparent 40%),
    var(--bg-secondary);
  border-color: var(--border-color);
}

.business-title {
  color: var(--text-primary);
}

.business-subtitle {
  color: var(--text-secondary);
}

.business-module {
  background: var(--bg-secondary);
  border-color: var(--border-color);
}

.business-module:hover {
  border-color: rgba(14, 165, 233, 0.3);
  background: var(--bg-tertiary);
}

.business-icon {
  color: #0EA5E9;
}
</style>
