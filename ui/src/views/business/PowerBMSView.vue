<script setup lang="ts">
import { onMounted, onUnmounted, ref, computed } from 'vue'
import { Cpu, ShieldAlert, ArrowLeft, Thermometer, Activity, BatteryMedium, Clock } from 'lucide-vue-next'
import { useP3Page } from '@/composables/useP3Page'
import { useBusinessStore, type BmsClusterSim } from '@/stores/business'
import { useLiveClock } from '@/composables/useLiveClock'
import AnimatedNumber from '@/components/p3/AnimatedNumber.vue'
import LiveSparkline from '@/components/p3/LiveSparkline.vue'
import BarMeter from '@/components/p3/BarMeter.vue'
import LiveMetricStrip from '@/components/p3/LiveMetricStrip.vue'
import P3EventFeed from '@/components/p3/P3EventFeed.vue'
import ExecutionTimeline from '@/components/p3/ExecutionTimeline.vue'
import P3ActionCenter from '@/components/p3/P3ActionCenter.vue'
import P3AuditTable from '@/components/p3/P3AuditTable.vue'

const { page } = useP3Page('power-bms')
const store = useBusinessStore()
const { dateTime } = useLiveClock()

const selectedId = ref(store.clusters[0]?.id ?? '')
const selected = computed(() => store.clusters.find(c => c.id === selectedId.value) ?? store.clusters[0])

// 行业监控配色规范：绿=正常 / 黄=预警 / 红=危险
const STATUS_META: Record<string, { label: string; color: string; soft: string; bar: string }> = {
  healthy: { label: '健康', color: '#10B981', soft: 'rgba(16,185,129,0.12)', bar: 'linear-gradient(90deg,#10B981,#34D399)' },
  watch: { label: '观察', color: '#F59E0B', soft: 'rgba(245,158,11,0.12)', bar: 'linear-gradient(90deg,#F59E0B,#FBBF24)' },
  risk: { label: '高风险', color: '#EF4444', soft: 'rgba(239,68,68,0.12)', bar: 'linear-gradient(90deg,#EF4444,#F87171)' },
}

function statusMeta(status: string) {
  return STATUS_META[status] ?? STATUS_META.healthy
}

// 温差阈值：≤3℃ 正常（绿）/ 3~5℃ 关注（黄）/ >5℃ 告警（红）
function tempColor(temp: number) {
  if (temp > 5) return '#EF4444'
  if (temp > 3) return '#F59E0B'
  return '#10B981'
}

// 压差阈值：≤0.1V 正常 / 0.1~0.2V 关注 / >0.2V 告警
function voltageColor(voltage: number) {
  if (voltage > 0.2) return '#EF4444'
  if (voltage > 0.1) return '#F59E0B'
  return '#10B981'
}

function cellBackground(cell: BmsClusterSim) {
  const meta = statusMeta(cell.status)
  return `linear-gradient(160deg, ${meta.soft}, var(--bg-secondary) 60%)`
}

function cellShadow(cell: BmsClusterSim, isSelected: boolean) {
  const meta = statusMeta(cell.status)
  return isSelected ? `0 0 0 1px ${meta.color}, 0 0 20px ${meta.color}40` : 'none'
}

onMounted(() => store.start())
onUnmounted(() => store.stop())
</script>

<template>
  <div class="space-y-5">
    <section class="bms-shell rounded-2xl border p-5">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
        <div>
          <router-link
            to="/business-center"
            class="mb-3 inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm transition-colors hover:bg-white/5"
            style="color: var(--text-secondary); border: 1px solid var(--border-color);"
          >
            <ArrowLeft class="h-4 w-4" />
            返回业务中心
          </router-link>
          <div class="inline-flex items-center gap-2 rounded-lg px-3 py-1 text-xs bms-chip">
            <Cpu class="h-3.5 w-3.5" />
            矩阵诊断视图
          </div>
          <div class="mt-3 flex items-center gap-3">
            <h1 class="text-3xl font-semibold bms-title">{{ page.title }}</h1>
            <span class="inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1 font-mono-num text-xs" style="color: var(--text-secondary); border-color: var(--border-color); background: var(--bg-tertiary);">
              <Clock class="h-3.5 w-3.5" style="color: #EF4444;" />
              {{ dateTime }}
            </span>
          </div>
          <p class="mt-2 text-sm bms-subtitle">{{ page.subtitle }}</p>
        </div>
        <div class="rounded-xl border px-4 py-3 bms-warning">
          <div class="flex items-center gap-2 text-sm font-medium text-red-500">
            <ShieldAlert class="h-4 w-4" />
            高风险簇 <AnimatedNumber :value="store.riskCount" /> 个
          </div>
          <div class="mt-1 text-xs bms-subtitle">温差 / 压差 / 寿命指数联判</div>
        </div>
      </div>
      <div class="mt-5">
        <LiveMetricStrip
          :items="[
            { label: '电池簇', value: 144, unit: '簇', color: '#0EA5E9' },
            { label: '平均温差', value: store.avgTemp, unit: '℃', decimals: 1, color: '#F59E0B', pulse: true },
            { label: '均衡中簇', value: store.balancingCount, unit: '簇', color: '#10B981', pulse: true },
            { label: '高风险簇', value: store.riskCount, unit: '簇', color: '#EF4444', pulse: true },
            { label: '均衡效率', value: 92, unit: '%', color: '#8B5CF6' },
            { label: 'Quality', value: 96, color: '#10B981' },
          ]"
        />
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
      <section class="rounded-2xl border p-4 bms-shell">
        <div class="flex items-center justify-between">
          <h3 class="text-sm font-semibold bms-title">电池簇热区矩阵</h3>
          <span class="text-xs bms-subtitle">颜色遵循温差热区规范 · 点击查看详情</span>
        </div>

        <!-- 热区图例 -->
        <div class="mt-4 flex flex-wrap items-center gap-2 rounded-xl border px-3 py-2 text-[11px] bms-panel">
          <span class="bms-subtitle">温差热区</span>
          <span class="inline-flex items-center gap-1.5">
            <span class="h-2.5 w-5 rounded-sm" style="background: #10B981;" />
            正常 ≤3℃
          </span>
          <span class="inline-flex items-center gap-1.5">
            <span class="h-2.5 w-5 rounded-sm" style="background: #F59E0B;" />
            关注 3~5℃
          </span>
          <span class="inline-flex items-center gap-1.5">
            <span class="h-2.5 w-5 rounded-sm" style="background: #EF4444;" />
            告警 &gt;5℃
          </span>
          <span class="mx-1 h-3 w-px" style="background: var(--border-color);" />
          <span class="bms-subtitle">状态</span>
          <span class="inline-flex items-center gap-1">
            <span class="h-2 w-2 rounded-full" style="background: #10B981;" />健康
          </span>
          <span class="inline-flex items-center gap-1">
            <span class="h-2 w-2 rounded-full" style="background: #F59E0B;" />观察
          </span>
          <span class="inline-flex items-center gap-1">
            <span class="h-2 w-2 rounded-full" style="background: #EF4444;" />高风险
          </span>
        </div>

        <div class="mt-4 grid gap-3 md:grid-cols-2">
          <article
            v-for="cell in store.clusters"
            :key="cell.id"
            class="relative cursor-pointer overflow-hidden rounded-xl border p-4 transition-all duration-300"
            :class="selectedId === cell.id ? 'cell-selected' : ''"
            :style="{
              background: cellBackground(cell),
              borderColor: selectedId === cell.id ? statusMeta(cell.status).color : 'var(--border-color)',
              boxShadow: cellShadow(cell, selectedId === cell.id),
              transition: 'background 0.6s ease, border-color 0.3s ease, box-shadow 0.3s ease',
            }"
            @click="selectedId = cell.id"
          >
            <!-- 顶部状态指示条 -->
            <div
              class="absolute inset-x-0 top-0 h-1"
              :style="{ background: statusMeta(cell.status).bar }"
            />

            <div class="flex items-center justify-between gap-3 pt-1">
              <div class="flex items-center gap-2 text-sm font-medium bms-title">
                <span
                  class="inline-block h-2 w-2 animate-pulse rounded-full"
                  :style="{ background: statusMeta(cell.status).color }"
                />
                {{ cell.name }}
              </div>
              <span
                class="rounded-md px-2 py-0.5 text-[11px] font-medium"
                :style="{ color: statusMeta(cell.status).color, background: `${statusMeta(cell.status).color}1a` }"
              >
                {{ statusMeta(cell.status).label }}
              </span>
            </div>

            <div class="mt-4 grid grid-cols-2 gap-3 text-xs">
              <div
                class="rounded-lg border px-3 py-2 bms-panel"
                :style="{ borderLeft: `3px solid ${tempColor(cell.temp)}` }"
              >
                <div class="flex items-center gap-1 bms-subtitle">
                  <Thermometer class="h-3 w-3" />
                  温差
                </div>
                <div
                  class="mt-1 font-mono-num text-base"
                  :style="{ color: tempColor(cell.temp) }"
                >
                  <AnimatedNumber :value="cell.temp" decimals="1" suffix="℃" />
                </div>
              </div>
              <div
                class="rounded-lg border px-3 py-2 bms-panel"
                :style="{ borderLeft: `3px solid ${voltageColor(cell.voltage)}` }"
              >
                <div class="flex items-center gap-1 bms-subtitle">
                  <Activity class="h-3 w-3" />
                  压差
                </div>
                <div
                  class="mt-1 font-mono-num text-base"
                  :style="{ color: voltageColor(cell.voltage) }"
                >
                  <AnimatedNumber :value="cell.voltage" decimals="2" suffix="V" />
                </div>
              </div>
            </div>

            <div class="mt-3 space-y-2">
              <div class="flex items-center justify-between rounded-lg border px-3 py-2 text-xs bms-panel">
                <div class="flex items-center gap-1.5 bms-subtitle">
                  <BatteryMedium class="h-3 w-3" />
                  均衡状态
                </div>
                <span class="font-medium bms-title">{{ cell.balance }}</span>
              </div>
              <div>
                <div class="mb-1 flex justify-between text-[11px]">
                  <span class="bms-subtitle">寿命指数</span>
                  <span class="font-mono-num bms-title">
                    <AnimatedNumber :value="cell.life" />
                  </span>
                </div>
                <BarMeter :value="cell.life" :max="100" :color="statusMeta(cell.status).color" :height="5" />
              </div>
            </div>
          </article>
        </div>
      </section>

      <div class="space-y-4">
        <!-- 选中簇详情 -->
        <section class="rounded-2xl border p-4 bms-shell">
          <div class="flex items-center justify-between">
            <h3 class="text-sm font-semibold bms-title">
              <span class="mr-1.5 inline-block h-2 w-2 rounded-full align-middle" :style="{ background: statusMeta(selected?.status ?? 'healthy').color }" />
              {{ selected?.name }} 实时详情
            </h3>
            <span class="inline-flex items-center gap-1.5 text-[11px] text-emerald-500">
              <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500" />
              实时
            </span>
          </div>
          <div class="mt-4 grid grid-cols-3 gap-3 text-center">
            <div class="rounded-lg border px-2 py-3 bms-panel">
              <div class="text-[11px] bms-subtitle">温差</div>
              <div class="mt-1 font-mono-num text-lg" :style="{ color: tempColor(selected?.temp ?? 0) }">
                <AnimatedNumber :value="selected?.temp ?? 0" decimals="1" suffix="℃" />
              </div>
            </div>
            <div class="rounded-lg border px-2 py-3 bms-panel">
              <div class="text-[11px] bms-subtitle">压差</div>
              <div class="mt-1 font-mono-num text-lg" :style="{ color: voltageColor(selected?.voltage ?? 0) }">
                <AnimatedNumber :value="selected?.voltage ?? 0" decimals="2" suffix="V" />
              </div>
            </div>
            <div class="rounded-lg border px-2 py-3 bms-panel">
              <div class="text-[11px] bms-subtitle">寿命</div>
              <div class="mt-1 font-mono-num text-lg bms-title">
                <AnimatedNumber :value="selected?.life ?? 0" />
              </div>
            </div>
          </div>
          <div class="mt-4">
            <div class="mb-1 flex justify-between text-xs">
              <span class="bms-subtitle">站群平均温差</span>
              <span class="font-mono-num bms-title">
                <AnimatedNumber :value="store.avgTemp" decimals="1" suffix="℃" />
              </span>
            </div>
            <LiveSparkline :points="store.tempSeries" color="#F59E0B" :height="56" />
          </div>
        </section>

        <P3EventFeed :title="page.sidePanelTitle" :events="store.domainEvents('bms')" dense />
        <ExecutionTimeline :title="page.timelineTitle" :items="page.timeline" />
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[0.9fr_1.1fr]">
      <P3ActionCenter :title="page.actionTitle" :actions="page.actions" />
      <P3AuditTable :records="page.auditRecords" title="BMS 操作审计" />
    </section>
  </div>
</template>

<style scoped>
.bms-shell,
.bms-panel,
.bms-warning {
  background: var(--bg-secondary);
  border-color: var(--border-color);
}

.bms-shell {
  background:
    linear-gradient(180deg, rgba(239, 68, 68, 0.06), transparent 24%),
    var(--bg-secondary);
}

.bms-chip {
  background: rgba(239, 68, 68, 0.08);
  color: #EF4444;
}

.bms-title { color: var(--text-primary); }
.bms-subtitle { color: var(--text-secondary); }

.cell-selected {
  transform: translateY(-2px);
}
</style>
